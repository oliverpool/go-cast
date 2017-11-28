package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/oliverpool/go-chromecast"
	clicast "github.com/oliverpool/go-chromecast/cli"
	"github.com/oliverpool/go-chromecast/command"
	"github.com/oliverpool/go-chromecast/command/media"
	"github.com/oliverpool/go-chromecast/command/media/defaultreceiver"
	"github.com/oliverpool/go-chromecast/command/volume"
	"gopkg.in/urfave/cli.v1"
)

func checkErr(err error) {
	if err != nil {
		if err == context.DeadlineExceeded {
			fmt.Println("Timeout exceeded")
		} else {
			fmt.Println(err)
		}
		os.Exit(1)
	}
}

var logger = kitlog.NewNopLogger()

func init() {
	log.SetOutput(ioutil.Discard)
}

func main() {
	commonFlags := []cli.Flag{
		cli.BoolFlag{
			Name:  "debug, d",
			Usage: "enable debug logging",
		},
		cli.StringFlag{
			Name:  "host",
			Usage: "chromecast hostname or IP",
		},
		cli.IntFlag{
			Name:  "port",
			Usage: "chromecast port",
			Value: 8009,
		},
		cli.StringFlag{
			Name:  "name",
			Usage: "chromecast name",
		},
		cli.DurationFlag{
			Name:  "timeout",
			Value: 15 * time.Second,
		},
	}
	app := cli.NewApp()
	app.Name = "cast"
	app.Usage = "Command line tool for the Chromecast"
	app.Version = chromecast.Version
	app.Flags = commonFlags
	app.Commands = []cli.Command{
		{
			Name:   "status",
			Usage:  "Get status of the Chromecast",
			Action: statusCommand,
		},
		{
			Name:   "discover",
			Usage:  "Discover Chromecast devices",
			Action: discoverCommand,
		},
	}
	app.Run(os.Args)
}

// clientFromContext will try to get a cast client.
// If host is set, it will be used (along port).
// Otherwise, if name is set, a chromecast will be looked-up by name.
// Otherwise the first chromecast found will be returned.
func clientFromContext(ctx context.Context, c *cli.Context) chromecast.Client {
	chr, err := clicast.GetDevice(
		ctx,
		c.GlobalString("host"),
		c.GlobalInt("port"),
		c.GlobalString("name"),
	)
	checkErr(err)
	fmt.Printf("Found '%s' (%s:%d)...\n", chr.Name(), chr.IP, chr.Port)

	if c.GlobalBool("debug") {
		logger = clicast.NewLogger(os.Stdout)
		log.SetOutput(kitlog.NewStdlibAdapter(logger))
	}

	client, err := clicast.NewClient(ctx, chr.Addr(), logger)
	checkErr(err)
	return client
}

func statusCommand(c *cli.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), c.GlobalDuration("timeout"))
	defer cancel()

	client := clientFromContext(ctx, c)

	// Get status
	fmt.Println("Status:")
	status, err := command.Status.Get(client)
	checkErr(err)
	clicast.FprintStatus(os.Stdout, status)

	// Get App
	app, err := media.FromStatus(client, status)
	if err != nil {
		fmt.Println("Launching new " + defaultreceiver.Name)
		app, err = media.Launch(client, defaultreceiver.ID)
	} else {
		fmt.Println("App retrieved")
	}
	checkErr(err)
	go app.UpdateStatus()
	fmt.Println(app)

	fmt.Println(app.GetStatus())

	if false {
		curSession, err := app.CurrentSession()
		for err != nil {
			fmt.Println("waiting for valid session")
			time.Sleep(time.Second)
			curSession, err = app.CurrentSession()
		}
		curSession.Play()
		return
	}

	session, err := app.Load(media.Item{
		// ContentId:   "https://www.soundhelix.com/examples/mp3/SoundHelix-Song-1.mp3",
		// ContentID:  "https://cdn.rawgit.com/mediaelement/mediaelement-files/4d21a042/echo-hereweare.mp4",
		// ContentID:  "http://mirrors.standaloneinstaller.com/video-sample/star_trails.mp4", // seekable
		ContentID:  "http://192.168.178.57:8100/sample.mp4", // local
		StreamType: "BUFFERED",
		// ContentType: "audio/mpeg",
	}, media.Seek(1*time.Second))
	checkErr(err)

	_, err = volume.Set(client, 0)
	checkErr(err)
	_ = session
	/*

		time.Sleep(4 * time.Second)
		session.Pause()
		_, err = volume.Mute(client, true)
		checkErr(err)

		time.Sleep(4 * time.Second)
		session.Play()
		time.Sleep(4 * time.Second)
		// ch, err := session.Stop()
		// <-ch
	*/

	status, err = command.Status.Get(client)
	checkErr(err)
	clicast.FprintStatus(os.Stdout, status)
}

func discoverCommand(c *cli.Context) {
	if c.GlobalBool("debug") {
		logger = clicast.NewLogger(os.Stdout)
		log.SetOutput(kitlog.NewStdlibAdapter(logger))
	}
	timeout := c.GlobalDuration("timeout")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	fmt.Printf("Running scanner for %s...\n", timeout)
	for client := range clicast.Scan(ctx, logger) {
		fmt.Printf("Found: %s:%d '%s' (%s: %s) %s\n", client.IP, client.Port, client.Name(), client.Type(), client.ID(), client.Status())
	}
	fmt.Println("Done")
}
