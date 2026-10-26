package main

import (
	"flag"
	"fmt"
	"log"
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
	startHTTP(fmt.Sprintf(":%s", port))
}
