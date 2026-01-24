package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

type KnownDevice struct {
	IPAddrs  []string `json:"ipaddrs"`
	IP6Addrs []string `json:"ip6addrs"`
	Name     string   `json:"name,omitempty"` // some devices don’t have a name
}

type Client struct {
	Mac           string `json:"mac"`
	Signal        int    `json:"signal"`
	SignalAvg     int    `json:"signal_avg"`
	Noise         int    `json:"noise"`
	Inactive      int    `json:"inactive"`
	ConnectedTime int    `json:"connected_time"`
	Thr           int    `json:"thr"`
	Authorized    bool   `json:"authorized"`
	Authenticated bool   `json:"authenticated"`
	Preamble      string `json:"preamble"`
	Wme           bool   `json:"wme"`
	Mfp           bool   `json:"mfp"`
	Tdls          bool   `json:"tdls"`
	MeshLlid      int    `json:"mesh llid"`
	MeshPlid      int    `json:"mesh plid"`
	MeshPlink     string `json:"mesh plink"`
	MeshLocalPS   string `json:"mesh local PS"`
	MeshPeerPS    string `json:"mesh peer PS"`
	MeshNonPeerPS string `json:"mesh non-peer PS"`
	Rx            struct {
		DropMisc int  `json:"drop_misc"`
		Packets  int  `json:"packets"`
		Bytes    int  `json:"bytes"`
		Ht       bool `json:"ht"`
		Vht      bool `json:"vht"`
		He       bool `json:"he"`
		Eht      bool `json:"eht"`
		Mhz      int  `json:"mhz"`
		Rate     int  `json:"rate"`
	} `json:"rx"`
	Tx struct {
		Failed  int  `json:"failed"`
		Retries int  `json:"retries"`
		Packets int  `json:"packets"`
		Bytes   int  `json:"bytes"`
		Ht      bool `json:"ht"`
		Vht     bool `json:"vht"`
		He      bool `json:"he"`
		Eht     bool `json:"eht"`
		Mhz     int  `json:"mhz"`
		Rate    int  `json:"rate"`
	} `json:"tx"`
	StationID  string      `json:"station_id,omitempty"`
	Iface      string      `json:"iface,omitempty"`
	Name       string      `json:"name,omitempty"`
	LastSeen   time.Time   `json:"last_seen"`
	DeviceInfo KnownDevice `json:"device_info,omitempty"`
}

var (
	store = NewStore()
)

func startPollers(cfg *Config) {
	resolver := NewResolver(1 * time.Hour)

	for _, s := range cfg.Stations {
		st := s
		// populate stationMap
		stationMap[st.ID] = &st
		go func() {
			interval := time.Duration(cfg.PollInterval) * time.Second
			for {
				if err := pollStation(st, resolver); err != nil {
					log.Printf("poll %s error: %v", st.ID, err)
				}
				notifyClientsUpdate()
				time.Sleep(interval)
			}
		}()
	}
}

func DeviceToString(mac string, d KnownDevice) string {
	ip := strings.Join(d.IPAddrs, ",")
	if len(d.IP6Addrs) > 0 {
		if ip != "" {
			ip += " "
		}
		ip += "(" + strings.Join(d.IP6Addrs, ", ") + ")"
	}
	return fmt.Sprintf("%s [%s] %s", ip, mac, d.Name)
}

func pollStation(st StationConfig, resolver *Resolver) error {
	client := &UbusClient{
		Host:     st.Host,
		Username: st.Username,
		Password: st.Password,
	}
	token, err := client.ubusLoginCached()
	if err != nil {
		return err
	}
	// HostHints für MAC→IP Mapping
	devices := make(map[string]KnownDevice)
	arp, err := resolver.GetARP(st, token)
	if err != nil {
		log.Printf("getARP error: %v", err)
	}
	if devicesMap, err := ubusCallHostDevices(st, token, "luci-rpc", "getHostHints", map[string]interface{}{}); err == nil {
		for mac, v := range devicesMap {
			deviceBytes, _ := json.Marshal(v)
			var d KnownDevice
			if err := json.Unmarshal(deviceBytes, &d); err != nil {
				panic(err)
			}
			if d.Name == "" {
				for _, ip := range d.IPAddrs {
					host, err := resolver.LookupAddr(ip)
					if (err == nil) && (host != ip) {
						d.Name = ip
						break
					}
				}
			}
			if len(d.IPAddrs) == 0 && d.Name != "" {
				d.IPAddrs, _ = resolver.LookupHost(d.Name)
			}
			// Fallback to ARP if no IPs in openwrt or no name to resolve
			if len(d.IPAddrs) == 0 {
				if ip, ok := arp[strings.ToLower(mac)]; ok {
					d.IPAddrs = []string{ip}
				}
			}
			devices[mac] = d
		}
	}
	for _, iface := range st.Ifaces {
		parsed, err := ubusCallHostDevices(st, token, "iwinfo", "assoclist", map[string]string{"device": iface})
		if err != nil {
			// retry mit neuem Login
			token, _, err = client.ubusLogin()
			if err != nil {
				continue
			}
			parsed, err = ubusCallHostDevices(st, token, "iwinfo", "assoclist", map[string]string{"device": iface})
			if err != nil {
				continue
			}
		}
		var resultsWrapper struct {
			Results []Client `json:"results"`
		}
		parsedBytes, _ := json.Marshal(parsed)
		err = json.Unmarshal(parsedBytes, &resultsWrapper)
		if err != nil {
			log.Fatal(err)
		}

		for _, c := range resultsWrapper.Results {
			c.LastSeen = time.Now()
			c.DeviceInfo = devices[c.Mac]
			if c.DeviceInfo.IPAddrs == nil {
				c.DeviceInfo.IPAddrs = []string{}
			}
			if c.DeviceInfo.IP6Addrs == nil {
				c.DeviceInfo.IP6Addrs = []string{}
			}
			c.StationID = st.ID
			c.Iface = iface
			c.Name = st.Name
			store.Upsert(c)
		}
	}
	return nil
}

func notifyClientsUpdate() {
	clients := store.List()
	b, _ := json.Marshal(map[string]interface{}{"clients": clients})
	broadcastWebsocket(b)
}

func StartPoller(ctx context.Context, interval time.Duration, st StationConfig, ub *UbusClient) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	backoffBase := time.Second
	backoffMax := 30 * time.Second
	var backoff time.Duration

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			token, err := ub.ubusLoginCachedContext(ctx)
			if err != nil {
				log.Printf("[poller] login failed: %v", err)
				if backoff == 0 {
					backoff = backoffBase
				} else {
					backoff *= 2
					if backoff > backoffMax {
						backoff = backoffMax
					}
				}
				select {
				case <-time.After(backoff):
					continue
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			backoff = 0

			val, err := ubusCallHostContext(ctx, st, token, "wifi", "status", nil)
			if err != nil {
				log.Printf("[poller] ubus call failed: %v", err)
				continue
			}
			_ = val
		}
	}
}
