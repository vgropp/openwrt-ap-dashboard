package main

import (
	"testing"
	"time"
)

func TestNewResolverCacheTTL(t *testing.T) {
	res := NewResolver(time.Hour)
	if res.Cache == nil {
		t.Fatal("Cache should be initialized")
	}
	res.Cache.Set("foo", "bar", time.Second)
	val, found := res.Cache.Get("foo")
	if !found || val != "bar" {
		t.Errorf("Expected to find 'foo' with value 'bar', got %v", val)
	}
	// Wait for expiration
	time.Sleep(1100 * time.Millisecond)
	_, found = res.Cache.Get("foo")
	if found {
		t.Error("Expected 'foo' to expire from cache")
	}
}

func TestMergeARPLogic(t *testing.T) {
	remote := map[string]string{"aa:bb:cc:dd:ee:ff": "192.168.1.2"}
	local := map[string]string{"aa:bb:cc:dd:ee:ff": "192.168.1.2", "11:22:33:44:55:66": "192.168.1.3"}
	merged := make(map[string]string)
	for k, v := range remote {
		merged[k] = v
	}
	for k, v := range local {
		if _, exists := merged[k]; !exists {
			merged[k] = v
		}
	}
	if len(merged) != 2 {
		t.Errorf("Expected 2 entries in merged ARP, got %d", len(merged))
	}
	if merged["11:22:33:44:55:66"] != "192.168.1.3" {
		t.Error("Expected local-only MAC to be present in merged ARP")
	}
}
