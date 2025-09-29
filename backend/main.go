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

	startPollers(cfg)

	port := os.Getenv("BACKEND_PORT")
	if port == "" {
		port = "8189"
	}
	startHTTP(fmt.Sprintf(":%s", port))
}
