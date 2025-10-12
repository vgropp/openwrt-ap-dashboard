package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

type OUILookup struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewOUILookup(path string) (*OUILookup, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	lookup := &OUILookup{data: make(map[string]string)}
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "(hex)") {
			parts := strings.SplitN(line, "(hex)", 2)
			if len(parts) < 2 {
				continue
			}
			prefix := strings.TrimSpace(parts[0])
			vendor := strings.TrimSpace(parts[1])

			prefix = strings.ReplaceAll(prefix, "-", ":")
			lookup.data[prefix] = vendor
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lookup, nil
}

func (o *OUILookup) Vendor(mac string) (string, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	mac = strings.ToUpper(mac)
	parts := strings.Split(mac, ":")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid MAC")
	}
	prefix := strings.Join(parts[:3], ":")
	if vendor, ok := o.data[prefix]; ok {
		return vendor, nil
	}
	return "", fmt.Errorf("unknown vendor")
}
