package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/oliverpool/go-chromecast"

	"github.com/oliverpool/go-chromecast/command/media"
	"github.com/oliverpool/go-chromecast/command/media/defaultreceiver"
	"github.com/oliverpool/go-chromecast/command/media/defaultreceiver/tatort"
	"github.com/oliverpool/go-chromecast/command/media/defaultreceiver/tvnow"
	defaultvimeo "github.com/oliverpool/go-chromecast/command/media/defaultreceiver/vimeo"
	"github.com/oliverpool/go-chromecast/command/media/vimeo"
	"github.com/oliverpool/go-chromecast/command/media/youtube"
	"github.com/oliverpool/go-chromecast/command/urlreceiver"
	"github.com/spf13/cobra"
)

var loadRequestTimeout time.Duration

var useLoader string
var loaders = []namedLoader{
	{"tatort", tatort.URLLoader},
	{"tvnow", tvnow.URLLoader},
	{"vimeo", vimeo.URLLoader},
	{"youtube", youtube.URLLoader},
	{"default.vimeo", defaultvimeo.URLLoader},
	{"default", defaultreceiver.URLLoader},
	{"urlreceiver", urlreceiver.URLLoader},
}

type namedLoader struct {
	name   string
	loader media.URLLoader
}

func (nl namedLoader) load(client chromecast.Client, status chromecast.Status, rawurl string) (<-chan []byte, error) {
	loader, err := nl.loader(rawurl)
	if err != nil {
		return nil, err
	}
	return loader(client, status)
}

func init() {
	loadCmd.Flags().DurationVarP(&loadRequestTimeout, "request-timeout", "r", 10*time.Second, "Duration to wait for a reply to the load request")
	var ll []string
	for _, l := range loaders {
		ll = append(ll, l.name)
	}
	loadCmd.Flags().StringVarP(&useLoader, "loader", "l", "", "Loader to use (supported loaders: "+strings.Join(ll, ", ")+")")
	rootCmd.AddCommand(loadCmd)
}

var loadCmd = &cobra.Command{
	Use:   "load [url]",
	Short: "Load a URL",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rawurl := args[0]

		logger, ctx, cancel := flags()
		defer cancel()

		client, status, err := GetClientWithStatus(ctx, logger)
		if err != nil {
			return fmt.Errorf("could not get a client: %v", err)
		}
		defer client.Close()

		for _, l := range loaders {
			var c <-chan []byte
			var err error

			if useLoader != "" {
				if l.name != useLoader {
					continue
				}
				c, err = l.load(client, status, rawurl)
				if err != nil {
					return err
				}
			} else {
				c, err = l.load(client, status, rawurl)
				if err != nil {
					logger.Log("loader", l.name, "state", "loading", "err", err)
					continue
				}
				fmt.Printf("Loading with %s\n", l.name)
			}
			select {
			case <-c:
			case <-time.After(loadRequestTimeout):
				logger.Log("loader", l.name, "err", "load request didn't return after 10s")
			}
			return nil
		}
		if useLoader != "" {
			var ll []string
			for _, l := range loaders {
				ll = append(ll, l.name)
			}
			return fmt.Errorf("unknown loader '%s' (supported loaders: %s)", useLoader, strings.Join(ll, ", "))
		}
		return fmt.Errorf("no supported loader found for %s", rawurl)
	},
}
