package main

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/patrickmn/go-cache"
)

type Resolver struct {
	cache *cache.Cache
}

func NewResolver(ttl time.Duration) *Resolver {
	return &Resolver{
		cache: cache.New(ttl, 2*ttl), // Default Expiration, Cleanup Interval
	}
}

func (r *Resolver) LookupAddr(ip string) (string, error) {
	if host, found := r.cache.Get("ip:" + ip); found {
		return host.(string), nil
	}

	names, err := net.LookupAddr(ip)
	if err != nil || len(names) == 0 {
		mdnsIPToHost, err := r.mdnsIPToHost(ip, 2*time.Second)
		if err == nil {
			r.cache.Set("ip:"+ip, mdnsIPToHost, cache.DefaultExpiration)
			return mdnsIPToHost, nil
		}
		r.cache.Set("ip:"+ip, ip, cache.DefaultExpiration)
		return ip, nil
	}

	hostname := strings.TrimSuffix(names[0], ".")
	r.cache.Set("ip:"+ip, hostname, cache.DefaultExpiration)
	return hostname, nil
}

func (r *Resolver) LookupHost(host string) ([]string, error) {
	if ips, found := r.cache.Get("host:" + host); found {
		return ips.([]string), nil
	}

	ips, err := net.LookupHost(host)
	if err != nil {
		return nil, err
	}

	r.cache.Set("host:"+host, ips, cache.DefaultExpiration)
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
