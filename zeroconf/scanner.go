package zeroconf

import (
	"fmt"
	"net"
	"strings"

	"github.com/grandcat/zeroconf"
	chromecast "github.com/oliverpool/go-chromecast"

	"context"
)

type Scanner struct {
	Logger chromecast.Logger
}

func (s Scanner) log(keyvals ...interface{}) {
	if s.Logger != nil {
		vals := make([]interface{}, 0, len(keyvals)+2)
		vals = append(vals, "package", "zeroconf")
		vals = append(vals, keyvals...)
		s.Logger.Log(vals...)
	}
}

// Scan repeatedly scans the network  and synchronously sends the chromecast found into the results channel.
// It finishes when the context is done.
func (s Scanner) Scan(ctx context.Context, results chan<- *chromecast.Device) error {
	defer close(results)

	// generate entries
	// Discover all services on the network (e.g. _workstation._tcp)
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return fmt.Errorf("failed to initialize resolver: %v", err)
	}

	entries := make(chan *zeroconf.ServiceEntry, 5)
	err = resolver.Browse(ctx, "_googlecast._tcp", "local", entries)
	if err != nil {
		return fmt.Errorf("fail to browse services: %v", err)
	}

	// decode entries
	for e := range entries {
		c, err := s.Decode(e)
		if err != nil {
			s.log("step", "Decode", "err", err)
			continue
		}
		select {
		case results <- c:
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return ctx.Err()
}

// Decode turns an mdns.ServiceEntry into a chromecast.Device
func (s Scanner) Decode(entry *zeroconf.ServiceEntry) (*chromecast.Device, error) {
	if !strings.Contains(entry.Service, "_googlecast.") {
		return nil, fmt.Errorf("fdqn '%s does not contain '_googlecast.'", entry.Service)
	}

	info := s.ParseProperties(entry.Text)

	var ip net.IP
	if len(entry.AddrIPv6) > 0 {
		ip = entry.AddrIPv6[0]
	} else if len(entry.AddrIPv4) > 0 {
		ip = entry.AddrIPv4[0]
	}

	return &chromecast.Device{
		IP:         ip,
		Port:       entry.Port,
		Properties: info,
	}, nil
}

// ParseProperties into a string map
// Input: key1=value1|key2=value2
func (Scanner) ParseProperties(s []string) map[string]string {
	m := make(map[string]string, len(s))
	for _, v := range s {
		s := strings.SplitN(v, "=", 2)
		if len(s) == 2 {
			m[s[0]] = s[1]
		}
	}
	return m
}
