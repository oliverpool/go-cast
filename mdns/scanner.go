package mdns

import (
	"fmt"
	"strings"
	"time"

	cast "github.com/oliverpool/go-chromecast"
	"github.com/hashicorp/mdns"

	"context"
)

// Scanner uses mdns to scan for chromecasts
type Scanner struct {
	// The chromecasts have 'Timeout' time to reply to each probe.
	Timeout time.Duration
}

// Scan repeatedly scans the network  and synchronously sends the chromecast found into the results channel.
// It finishes when the context is done.
func (s Scanner) Scan(ctx context.Context, results chan<- *cast.Device) error {
	defer close(results)

	// generate entries
	entries := make(chan *mdns.ServiceEntry, 10)
	go func() {
		defer close(entries)
		for {
			if ctx.Err() != nil {
				return
			}
			mdns.Query(&mdns.QueryParam{
				Service: "_googlecast._tcp",
				Domain:  "local",
				Timeout: s.Timeout,
				Entries: entries,
			})
		}
	}()

	// decode entries
	for e := range entries {
		c, err := s.Decode(e)
		if err != nil {
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

// Decode turns an mdns.ServiceEntry into a cast.Chromecast
func (s Scanner) Decode(entry *mdns.ServiceEntry) (*cast.Device, error) {
	if !strings.Contains(entry.Name, "._googlecast") {
		return nil, fmt.Errorf("fdqn '%s does not contain '._googlecast'", entry.Name)
	}

	info := s.ParseProperties(entry.Info)

	return &cast.Device{
		IP:         entry.AddrV4,
		Port:       entry.Port,
		Properties: info,
	}, nil
}

// ParseProperties into a string map
// Input: key1=value1|key2=value2
func (Scanner) ParseProperties(txt string) map[string]string {
	s := strings.Split(txt, "|")
	m := make(map[string]string, len(s))
	for _, v := range s {
		s := strings.SplitN(v, "=", 2)
		if len(s) == 2 {
			m[s[0]] = s[1]
		}
	}
	return m
}