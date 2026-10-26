package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type StationConfig struct {
	ID       string   `yaml:"id"`
	Name     string   `yaml:"name"`
	Host     string   `yaml:"host"`
	Port     int      `yaml:"port"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
	Ifaces   []string `yaml:"ifaces"`
}

type Config struct {
	Stations     []StationConfig `yaml:"stations"`
	PollInterval int             `yaml:"poll_interval"`
	Presence     PresenceConfig  `yaml:"presence"`
	MQTT         MQTTConfig      `yaml:"mqtt"`
}

type PresenceConfig struct {
	Devices []PresenceDevice `yaml:"devices"`
}

type PresenceDevice struct {
	Name string `yaml:"name"`
	MAC  string `yaml:"mac"`
}

type MQTTConfig struct {
	Broker            string `yaml:"broker"`
	ClientID          string `yaml:"client_id"`
	Username          string `yaml:"username"`
	Password          string `yaml:"password"`
	TopicPrefix       string `yaml:"topic_prefix"`
	DiscoveryPrefix   string `yaml:"discovery_prefix"`
	DiscoveryIDPrefix string `yaml:"discovery_id_prefix"`
}

func LoadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	if c.PollInterval == 0 {
		c.PollInterval = 15
	}
	if len(c.Presence.Devices) > 0 && strings.TrimSpace(c.MQTT.Broker) == "" {
		return nil, fmt.Errorf("mqtt.broker is required when presence.devices are configured")
	}
	for i := range c.Presence.Devices {
		device := &c.Presence.Devices[i]
		mac, err := normalizeMAC(device.MAC)
		if err != nil {
			return nil, fmt.Errorf("presence.devices[%d].mac: %w", i, err)
		}
		device.MAC = mac
		if strings.TrimSpace(device.Name) == "" {
			device.Name = mac
		}
	}
	return &c, nil
}
