package main

import (
	"flag"
	"log"
	"net"
	"os"
)

var stationMap = make(map[string]*StationConfig)

func main() {
	cfgPath := flag.String("config", "../stations.yaml", "path to stations.yaml")
	flag.Parse()
	cfg, err := LoadConfig(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// populate stationMap
	for i := range cfg.Stations {
		st := &cfg.Stations[i]
		stationMap[st.ID] = st
	}

	stationIDs := make([]string, 0, len(cfg.Stations))
	for _, station := range cfg.Stations {
		stationIDs = append(stationIDs, station.ID)
	}
	presence, err := NewPresenceTracker(cfg.MQTT, cfg.Presence, stationIDs)
	if err != nil {
		log.Fatalf("configure presence: %v", err)
	}
	defer presence.Close()

	startPollers(cfg, presence)

	port := os.Getenv("BACKEND_PORT")
	if port == "" {
		port = "8189"
	}
	// BACKEND_READONLY_HOST adds a second listener with only GET /api/clients,
	// e.g. on Docker's bridge for other containers, while the full API stays
	// on BACKEND_HOST. Its port is BACKEND_READONLY_PORT (default: BACKEND_PORT).
	if roHost := os.Getenv("BACKEND_READONLY_HOST"); roHost != "" {
		roPort := os.Getenv("BACKEND_READONLY_PORT")
		if roPort == "" {
			roPort = port
		}
		go startReadOnlyHTTP(net.JoinHostPort(roHost, roPort))
	}

	// BACKEND_HOST limits the listener to one address, e.g. 127.0.0.1 behind a
	// local reverse proxy. Empty: all addresses.
	startHTTP(net.JoinHostPort(os.Getenv("BACKEND_HOST"), port))
}
