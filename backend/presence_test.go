package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoveryConfigsShareOneDevice(t *testing.T) {
	devices := []PresenceDevice{
		{Name: "Alice phone", MAC: "aa:bb:cc:dd:ee:ff"},
		{Name: "Bob phone", MAC: "11:22:33:44:55:66"},
	}
	var sharedIdentifiers []string
	for _, device := range devices {
		payload, err := json.Marshal(newMQTTDiscoveryConfig(device, "openwrt/presence/"+device.MAC+"/state", "test_installation"))
		if err != nil {
			t.Fatalf("marshal discovery config: %v", err)
		}

		var config mqttDiscoveryConfig
		if err := json.Unmarshal(payload, &config); err != nil {
			t.Fatalf("unmarshal discovery config: %v", err)
		}
		if len(config.Device.Identifiers) != 1 {
			t.Fatalf("device identifiers = %v, want one shared identifier", config.Device.Identifiers)
		}
		if config.Device.Identifiers[0] != "test_installation" {
			t.Fatalf("device identifier = %q, want configured prefix", config.Device.Identifiers[0])
		}
		if !strings.HasPrefix(config.UniqueID, "test_installation_") {
			t.Fatalf("entity unique ID = %q, want configured prefix", config.UniqueID)
		}
		if sharedIdentifiers == nil {
			sharedIdentifiers = config.Device.Identifiers
			continue
		}
		if config.Device.Identifiers[0] != sharedIdentifiers[0] {
			t.Fatalf("device identifiers differ: %v and %v", sharedIdentifiers, config.Device.Identifiers)
		}
	}
}

func TestNormalizeMAC(t *testing.T) {
	for _, input := range []string{"AA:BB:CC:DD:EE:FF", "aa:bb:cc:dd:ee:ff"} {
		mac, err := normalizeMAC(input)
		if err != nil {
			t.Fatalf("normalizeMAC(%q) returned error: %v", input, err)
		}
		if mac != "aa:bb:cc:dd:ee:ff" {
			t.Fatalf("normalizeMAC(%q) = %q, want %q", input, mac, "aa:bb:cc:dd:ee:ff")
		}
	}
}

func TestUpdateStationMatchesUppercaseDashboardMAC(t *testing.T) {
	const configuredMAC = "aa:bb:cc:dd:ee:ff"
	tracker := &PresenceTracker{
		stations: map[string]struct{}{
			"first":  {},
			"second": {},
		},
		stationSeen: make(map[string]map[string]struct{}),
		devices: map[string]PresenceDevice{
			configuredMAC: {Name: "Phone", MAC: configuredMAC},
		},
		states: make(map[string]string),
	}

	if err := tracker.UpdateStation("first", []string{"AA:BB:CC:DD:EE:FF"}); err != nil {
		t.Fatalf("UpdateStation returned error: %v", err)
	}
	if _, found := tracker.stationSeen["first"][configuredMAC]; !found {
		t.Fatal("uppercase dashboard MAC was not matched to the configured device")
	}
}

func TestDesiredStatesAcrossStations(t *testing.T) {
	const phone = "aa:bb:cc:dd:ee:ff"
	tracker := &PresenceTracker{
		devices: map[string]PresenceDevice{
			phone: {Name: "Phone", MAC: phone},
		},
		stationSeen: map[string]map[string]struct{}{
			"first":  {},
			"second": {phone: {}},
		},
	}

	if got := tracker.desiredStates()[phone]; got != "home" {
		t.Fatalf("desired state = %q, want home when present on any station", got)
	}

	tracker.stationSeen["second"] = map[string]struct{}{}
	if got := tracker.desiredStates()[phone]; got != "not_home" {
		t.Fatalf("desired state = %q, want not_home when absent from all stations", got)
	}
}

func TestLoadConfigPresenceAndMQTT(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stations.yaml")
	configYAML := []byte(`
stations:
  - id: living-room
    ifaces: [wlan0]
presence:
  devices:
    - name: Phone
      mac: AA:BB:CC:DD:EE:FF
mqtt:
  broker: tcp://localhost:1883
  discovery_id_prefix: home_network
`)
	if err := os.WriteFile(path, configYAML, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	config, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if config.Presence.Devices[0].MAC != "aa:bb:cc:dd:ee:ff" {
		t.Fatalf("configured MAC = %q, want normalized MAC", config.Presence.Devices[0].MAC)
	}
	if config.MQTT.Broker != "tcp://localhost:1883" {
		t.Fatalf("MQTT broker = %q, want tcp://localhost:1883", config.MQTT.Broker)
	}
	if config.MQTT.DiscoveryIDPrefix != "home_network" {
		t.Fatalf("MQTT discovery ID prefix = %q, want home_network", config.MQTT.DiscoveryIDPrefix)
	}
}

func TestLoadConfigRequiresMQTTBrokerForPresence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stations.yaml")
	configYAML := []byte(`
presence:
  devices:
    - mac: "aa:bb:cc:dd:ee:ff"
`)
	if err := os.WriteFile(path, configYAML, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := LoadConfig(path); err == nil {
		t.Fatal("LoadConfig succeeded without mqtt.broker")
	}
}
