package main

import (
	"fmt"
	"os"
	"strings"

	flags "github.com/jessevdk/go-flags"
)

var opts struct {
	Token            string   `long:"token" short:"t" default:"" descrioption:"Bot token to authenticate with"`
	Admins           []string `long:"admin" short:"a" description:"Bot admins"`
	ChannelsListen   []string `long:"channel-listen" description:"Channels to interact with"`
	ChannelsAnnounce []string `long:"channel-announce" description:"Channels to announce things to"`
}

var botAnnounceChannels = []string{}

func initopts() {
	_, err := flags.Parse(&opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't parse command line arguments: %s\n", err)
		os.Exit(1)
	}

	if len(opts.Token) == 0 {
		fmt.Fprintf(os.Stderr, "ERROR: Bot token required\n")
		os.Exit(1)
	}

	for _, admin := range opts.Admins {
		ad := strings.Split(admin, "#")
		if len(ad) != 2 {
			fmt.Fprintf(os.Stderr, "ERROR: Invalid admin username specified: %s\n", admin)
			os.Exit(1)
		}
		validAdmins[un{ad[0], ad[1]}] = true
	}

	for _, channel := range opts.ChannelsListen {
		ch := strings.Split(channel, ":")
		if len(ch) != 2 {
			fmt.Fprintf(os.Stderr, "ERROR: Invalid channel specified: %s\n", channel)
			os.Exit(1)
		}
		validChannels[cn{ch[0], ch[1]}] = true
	}

	for _, channel := range opts.ChannelsAnnounce {
		ch := strings.Split(channel, ":")
		if len(ch) != 2 {
			fmt.Fprintf(os.Stderr, "ERROR: Invalid channel speicifed: %s\n", channel)
			os.Exit(1)
		}
		announceChannels = append(announceChannels, cn{ch[0], ch[1]})
	}
}
