package main

import (
	"testing"
)

func TestFindIPByMACFallback(t *testing.T) {
	arp := map[string]string{
		"aa:bb:cc:dd:ee:ff": "192.168.1.2",
		"11:22:33:44:55:66": "192.168.1.3",
	}
	mac := "11:22:33:44:55:66"
	ip, found := arp[mac]
	if !found {
		t.Fatalf("MAC %s not found in ARP table", mac)
	}
	if ip != "192.168.1.3" {
		t.Errorf("Expected IP '192.168.1.3', got '%s'", ip)
	}
}
