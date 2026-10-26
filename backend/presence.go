package main

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type PresenceTracker struct {
	mu          sync.Mutex
	client      mqtt.Client
	stations    map[string]struct{}
	stationSeen map[string]map[string]struct{}
	devices     map[string]PresenceDevice
	states      map[string]string
	initialized bool
	topicPrefix string
}

type mqttDiscoveryDevice struct {
	Identifiers  []string `json:"identifiers"`
	Name         string   `json:"name"`
	Manufacturer string   `json:"manufacturer"`
	Model        string   `json:"model"`
}

type mqttDiscoveryConfig struct {
	Name           string              `json:"name"`
	UniqueID       string              `json:"unique_id"`
	StateTopic     string              `json:"state_topic"`
	PayloadHome    string              `json:"payload_home"`
	PayloadNotHome string              `json:"payload_not_home"`
	SourceType     string              `json:"source_type"`
	Device         mqttDiscoveryDevice `json:"device"`
}

func newMQTTDiscoveryConfig(device PresenceDevice, stateTopic, idPrefix string) mqttDiscoveryConfig {
	return mqttDiscoveryConfig{
		Name:           device.Name,
		UniqueID:       idPrefix + "_" + strings.ReplaceAll(device.MAC, ":", ""),
		StateTopic:     stateTopic,
		PayloadHome:    "home",
		PayloadNotHome: "not_home",
		SourceType:     "router",
		Device: mqttDiscoveryDevice{
			Identifiers:  []string{idPrefix},
			Name:         "OpenWrt AP Dashboard",
			Manufacturer: "OpenWrt AP Dashboard",
			Model:        "Wi-Fi presence tracker",
		},
	}
}

func normalizeMAC(value string) (string, error) {
	hw, err := net.ParseMAC(strings.TrimSpace(value))
	if err != nil || len(hw) != 6 {
		return "", fmt.Errorf("invalid MAC address %q", value)
	}
	return strings.ToLower(hw.String()), nil
}

func NewPresenceTracker(config MQTTConfig, devices []PresenceDevice, stationIDs []string) (*PresenceTracker, error) {
	if len(devices) == 0 {
		return nil, nil
	}
	if strings.TrimSpace(config.Broker) == "" {
		return nil, fmt.Errorf("mqtt.broker is required when presence.devices are configured")
	}
	if len(stationIDs) == 0 {
		return nil, fmt.Errorf("at least one station is required when presence.devices are configured")
	}

	if config.ClientID == "" {
		config.ClientID = "openwrt-ap-dashboard"
	}
	if config.TopicPrefix == "" {
		config.TopicPrefix = "openwrt/presence"
	}
	if config.DiscoveryPrefix == "" {
		config.DiscoveryPrefix = "homeassistant"
	}
	if config.DiscoveryIDPrefix == "" {
		config.DiscoveryIDPrefix = "openwrt_ap_dashboard"
	}
	config.TopicPrefix = strings.Trim(config.TopicPrefix, "/")
	config.DiscoveryPrefix = strings.Trim(config.DiscoveryPrefix, "/")
	config.DiscoveryIDPrefix = strings.TrimSpace(config.DiscoveryIDPrefix)
	if config.DiscoveryIDPrefix == "" {
		return nil, fmt.Errorf("mqtt.discovery_id_prefix must not be empty")
	}

	opts := mqtt.NewClientOptions().
		AddBroker(config.Broker).
		SetClientID(config.ClientID).
		SetUsername(config.Username).
		SetPassword(config.Password).
		SetConnectTimeout(10 * time.Second).
		SetAutoReconnect(true)
	client := mqtt.NewClient(opts)
	token := client.Connect()
	if !token.WaitTimeout(15 * time.Second) {
		return nil, fmt.Errorf("timed out connecting to MQTT broker %q", config.Broker)
	}
	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("connect to MQTT broker %q: %w", config.Broker, err)
	}

	tracker := &PresenceTracker{
		client:      client,
		stations:    make(map[string]struct{}, len(stationIDs)),
		stationSeen: make(map[string]map[string]struct{}, len(stationIDs)),
		devices:     make(map[string]PresenceDevice, len(devices)),
		states:      make(map[string]string, len(devices)),
		topicPrefix: config.TopicPrefix,
	}
	for _, id := range stationIDs {
		tracker.stations[id] = struct{}{}
	}
	for _, device := range devices {
		mac, err := normalizeMAC(device.MAC)
		if err != nil {
			client.Disconnect(250)
			return nil, fmt.Errorf("invalid presence device MAC: %w", err)
		}
		device.MAC = mac
		if device.Name == "" {
			device.Name = mac
		}
		tracker.devices[mac] = device
	}

	for _, device := range tracker.devices {
		stateTopic := tracker.topicPrefix + "/" + strings.ReplaceAll(device.MAC, ":", "") + "/state"
		discovery := newMQTTDiscoveryConfig(device, stateTopic, config.DiscoveryIDPrefix)
		payload, err := json.Marshal(discovery)
		if err != nil {
			client.Disconnect(250)
			return nil, fmt.Errorf("encode MQTT discovery for %s: %w", device.MAC, err)
		}
		if err := publishMQTT(client, config.DiscoveryPrefix+"/device_tracker/"+discovery.UniqueID+"/config", payload); err != nil {
			client.Disconnect(250)
			return nil, fmt.Errorf("publish MQTT discovery for %s: %w", device.MAC, err)
		}
	}
	return tracker, nil
}

func (tracker *PresenceTracker) Close() {
	if tracker != nil {
		tracker.client.Disconnect(250)
	}
}

func (tracker *PresenceTracker) UpdateStation(stationID string, currentMACs []string) error {
	if tracker == nil {
		return nil
	}
	seen := make(map[string]struct{}, len(currentMACs))
	for _, value := range currentMACs {
		mac, err := normalizeMAC(value)
		if err != nil {
			return fmt.Errorf("station %s returned invalid MAC %q: %w", stationID, value, err)
		}
		if _, tracked := tracker.devices[mac]; tracked {
			seen[mac] = struct{}{}
		}
	}

	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	if _, knownStation := tracker.stations[stationID]; !knownStation {
		return fmt.Errorf("unknown station %q", stationID)
	}
	tracker.stationSeen[stationID] = seen
	if !tracker.initialized && len(tracker.stationSeen) == len(tracker.stations) {
		tracker.initialized = true
	}
	if !tracker.initialized {
		return nil
	}

	desired := tracker.desiredStates()
	for mac, state := range desired {
		if tracker.states[mac] == state {
			continue
		}
		topic := tracker.topicPrefix + "/" + strings.ReplaceAll(mac, ":", "") + "/state"
		if err := publishMQTT(tracker.client, topic, []byte(state)); err != nil {
			return fmt.Errorf("publish presence state for %s: %w", mac, err)
		}
		tracker.states[mac] = state
	}
	return nil
}

func (tracker *PresenceTracker) desiredStates() map[string]string {
	connected := make(map[string]struct{})
	for _, stationMACs := range tracker.stationSeen {
		for mac := range stationMACs {
			connected[mac] = struct{}{}
		}
	}

	states := make(map[string]string, len(tracker.devices))
	for mac := range tracker.devices {
		states[mac] = "not_home"
		if _, isConnected := connected[mac]; isConnected {
			states[mac] = "home"
		}
	}
	return states
}

func publishMQTT(client mqtt.Client, topic string, payload []byte) error {
	token := client.Publish(topic, 1, true, payload)
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("timed out publishing to %s", topic)
	}
	if err := token.Error(); err != nil {
		return fmt.Errorf("publish to %s: %w", topic, err)
	}
	return nil
}
