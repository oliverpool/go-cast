package cli

import (
	"time"

	"github.com/oliverpool/go-chromecast"
	"github.com/oliverpool/go-chromecast/discover"
	"github.com/oliverpool/go-chromecast/mdns"
	"context"
)

func Scan(ctx context.Context) chan *chromecast.Device {
	all := make(chan *chromecast.Device, 5)
	scanner := mdns.Scanner{
		Timeout: 3 * time.Second,
	}
	go scanner.Scan(ctx, all)

	uniq := make(chan *chromecast.Device, 5)
	go discover.Uniq(all, uniq)
	return uniq
}
