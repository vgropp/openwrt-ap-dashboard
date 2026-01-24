package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/patrickmn/go-cache"
)

type Resolver struct {
	Cache *cache.Cache
}

func NewResolver(ttl time.Duration) *Resolver {
	return &Resolver{
		Cache: cache.New(ttl, 2*ttl), // Default Expiration, Cleanup Interval
	}
}

func (r *Resolver) LookupAddr(ip string) (string, error) {
	if host, found := r.Cache.Get("ip:" + ip); found {
		return host.(string), nil
	}

	names, err := net.LookupAddr(ip)
	if err != nil || len(names) == 0 {
		mdnsIPToHost, err := r.mdnsIPToHost(ip, 2*time.Second)
		if err == nil {
			r.Cache.Set("ip:"+ip, mdnsIPToHost, cache.DefaultExpiration)
			return mdnsIPToHost, nil
		}
		r.Cache.Set("ip:"+ip, ip, cache.DefaultExpiration)
		return ip, nil
	}

	hostname := strings.TrimSuffix(names[0], ".")
	r.Cache.Set("ip:"+ip, hostname, cache.DefaultExpiration)
	return hostname, nil
}

func (r *Resolver) LookupHost(host string) ([]string, error) {
	if ips, found := r.Cache.Get("host:" + host); found {
		return ips.([]string), nil
	}

	ips, err := net.LookupHost(host)
	if err != nil {
		return nil, err
	}

	r.Cache.Set("host:"+host, ips, cache.DefaultExpiration)
	return ips, nil
}

func (r *Resolver) mdnsIPToHost(ip string, timeout time.Duration) (string, error) {
	services := []string{
		"_esphomelib._tcp",
		"_shelly_._tcp",
		"_matter._tcp",
		"_http._tcp",
		"_ssh._tcp",
		"_workstation._tcp",
		"_printer._tcp",
	}

	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	found := make(chan string, 1)

	var wg sync.WaitGroup
	for _, svc := range services {
		wg.Add(1)
		go func(svc string) {
			defer wg.Done()
			entries := make(chan *zeroconf.ServiceEntry)
			go func() {
				for entry := range entries {
					for _, addr := range entry.AddrIPv4 {
						if addr.String() == ip {
							select {
							case found <- strings.TrimSuffix(entry.HostName, "."):
							default:
							}
							return
						}
					}
					for _, addr := range entry.AddrIPv6 {
						if addr.String() == ip {
							select {
							case found <- strings.TrimSuffix(entry.HostName, "."):
							default:
							}
							return
						}
					}
				}
			}()

			_ = resolver.Browse(ctx, svc, "local.", entries)
		}(svc)
	}

	select {
	case hostname := <-found:
		cancel()
		wg.Wait()
		return hostname, nil
	case <-ctx.Done():
		wg.Wait()
		return "", fmt.Errorf("no mDNS host found for IP %s", ip)
	}
}

func getARPTable(st StationConfig, token string) (map[string]string, error) {
	params := map[string]interface{}{
		"path": "/proc/net/arp",
	}
	rpcResp, err := ubusBaseCall(st, token, "file", "read", params)
	if err != nil {
		return nil, err
	}
	if len(rpcResp.Result) < 1 {
		return nil, fmt.Errorf("no result")
	}
	code := rpcResp.Result[0]
	if codeFloat, ok := code.(float64); ok && int(codeFloat) != 0 {
		return nil, fmt.Errorf("ubus read failed with code %d", int(codeFloat))
	}
	if len(rpcResp.Result) < 2 {
		return nil, fmt.Errorf("no data in result")
	}
	result, ok := rpcResp.Result[1].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("result not map")
	}
	stdout, ok := result["data"].(string) // for read, it's "data" not "stdout"
	if !ok {
		return nil, fmt.Errorf("no data")
	}
	arp := make(map[string]string)
	lines := strings.Split(stdout, "\n")
	for i, line := range lines {
		if i == 0 { // skip header
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			ip := fields[0]
			mac := fields[3]
			if mac != "00:00:00:00:00:00" && mac != "(incomplete)" { // skip invalid
				arp[strings.ToLower(mac)] = ip
			}
		}
	}
	return arp, nil
}

func getLocalARPTable() (map[string]string, error) {
	file, err := os.Open("/proc/net/arp")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	arp := make(map[string]string)
	scanner := bufio.NewScanner(file)
	scanner.Scan() // skip header

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 4 {
			ip := fields[0]
			mac := strings.ToLower(fields[3])
			if mac != "00:00:00:00:00:00" && mac != "(incomplete)" {
				arp[mac] = ip
			}
		}
	}
	return arp, nil
}

func (r *Resolver) GetARP(st StationConfig, token string) (map[string]string, error) {
	key := "arp:" + st.ID
	if arp, found := r.Cache.Get(key); found {
		return arp.(map[string]string), nil
	}

	// remote ARP
	arp, err := getARPTable(st, token)
	if err != nil {
		log.Printf("getARPTable error %s: %v", key, err)
		if !strings.Contains(err.Error(), "code 6") {
			return nil, err
		}
		// Bei Code 6: starte mit leerer Map
		arp = make(map[string]string)
	}

	// Merge with local arp
	localARP, err := getLocalARPTable()
	if err != nil {
		log.Printf("getLocalARPTable error: %v", err)
	} else {
		// add local entries if not exist in remote
		for mac, ip := range localARP {
			if _, exists := arp[mac]; !exists {
				arp[mac] = ip
			}
		}
	}

	log.Printf("ARP cache success for key: %s (entries: %d)", key, len(arp))

	r.Cache.Set(key, arp, cache.DefaultExpiration)
	return arp, nil
}
