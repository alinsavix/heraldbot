package main

import (
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

var version string // filled at build tie

func check(e error) {
	if e != nil {
		panic(e)
	}
}

var BotID string // who we are, so we avid loops

// Guild name & ID mappings
type GuildInfo struct {
	GuildName string
	GuildId   string
}

// Channel name & id mappings
type ChannelInfo struct {
	Guild       *GuildInfo
	ChannelName string
	ChannelId   string
	IsPrivate   bool
}

// channel names (server + channel name)
type cn struct {
	GuildName   string
	ChannelName string
}

// usernames (name + discriminator)
type un struct {
	UserName string
	UserDisc string
}

// Caches for converting names & ids
var channelsById = map[string]*ChannelInfo{}
var channelsByName = map[cn]*ChannelInfo{}
var guildsById = map[string]*GuildInfo{}
var guildsByName = map[string]*GuildInfo{}

var validChannels = map[cn]bool{} // channels the bot should use
var validAdmins = map[un]bool{}   // people who can use admin-only cmds
var announceChannels = []cn{}     // channels announcements should be done in

var dg *discordgo.Session

func cleanup() {
	fmt.Printf("Termination signal received\n")
	if dg != nil {
		fmt.Printf("Shutting down open Discord connections\n")
		dg.Close()
	}
}

func log(format string, vals ...interface{}) {
	logmsg := fmt.Sprintf(format, vals...)
	fmt.Fprint(os.Stderr, logmsg)
}

// For whatever reason, sometimes connecting to discord never gives a
// READY frame, so... open connection, wait a second, see if we have
// a ready frame or not, and if we haven't within 5 seconds, fail.
//
// There's probably a more correct way to do this (e.g. we don't have a
// mutex, etc), but this should be sufficient for our simple use case.
var discordReady = false

func tryOpen(d *discordgo.Session) bool {
	fmt.Fprintf(os.Stderr, "Attempting connection to discord...")

	err := d.Open()
	check(err)

	for i := 0; i <= 5; i++ {
		time.Sleep(1 * time.Second)
		if discordReady == true {
			fmt.Fprintf(os.Stderr, "success\n")
			return true
		}
	}

	d.Close()
	fmt.Fprintf(os.Stderr, "failed\n")
	return false
}

func main() {
	var err error
	initopts()

	// Give us a dummy version string if build process didn't give us one
	if version == "" {
		version = "[unknown version]"
	}

	log("HeraldBot %s starting up...\n", version)
	dg, err = discordgo.New("Bot " + opts.Token)
	check(err)

	dg.AddHandler(readyEvent)
	dg.AddHandler(messageCreate)
	dg.AddHandler(guildCreate)

	for i := 0; i <= 10; i++ {
		if tryOpen(dg) {
			break
		} else {
			log("Didn't get a Ready frame from discord in a reasonable time\n")
		}
	}

	if discordReady == false {
		log("Couldn't connect to Discord after many retries, exiting.\n")
		os.Exit(1)
	}

	u, err := dg.User("@me")
	check(err)

	BotID = u.ID

	go patreonWatch() // watch the patreon site for changes
	log("Bot running, press CTRL-C to exit\n")

	// Set up so we can exit on term signal/etc
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c // Wait 'forever' and let everything else do its thing
	cleanup()
	os.Exit(0)
}

// Given guild ID, get the rest of the guild info
func getGuildById(gid string) *GuildInfo {
	if g, ok := guildsById[gid]; ok {
		return g
	}

	if gid == "" {
		gi := &GuildInfo{
			GuildId:   "",
			GuildName: "",
		}

		return gi
	}

	g, err := dg.Guild(gid)
	if err != nil {
		log("getGuildById couldn't get guild info for guild id %s\n", gid)
		return nil
	}

	gi := &GuildInfo{
		GuildId:   gid,
		GuildName: g.Name,
	}

	guildsById[gid] = gi
	guildsByName[g.Name] = gi

	return gi
}

// Given a guild name and channel name, get the full channel info
func getChannelByName(guild string, channel string) *ChannelInfo {
	if c, ok := channelsByName[cn{guild, channel}]; ok {
		return c
	}

	return nil
}

// Given a channel id, get the full channel info
func getChannelById(cid string) *ChannelInfo {
	if c, ok := channelsById[cid]; ok {
		return c
	}

	c, err := dg.Channel(cid)
	if err != nil {
		log("getChannelById couldn't get channel info for channel id %s\n", cid)
		return nil
	}

	g := getGuildById(c.GuildID)
	if g == nil {
		log("Channel couldn't get guild info for guild id %s\n", c.GuildID)
		return nil
	}

	ci := &ChannelInfo{
		Guild:       g,
		ChannelId:   cid,
		ChannelName: c.Name,
		IsPrivate:   c.IsPrivate,
	}

	channelsById[cid] = ci
	channelsByName[cn{g.GuildName, ci.ChannelName}] = ci

	return ci
}

// Used when server info is provided to us via either initial READY or
// follow-on messages
func doGuild(g *discordgo.Guild) {
	log("Identified guild '%s' (id=%s)\n", g.Name, g.ID)
	for _, c := range g.Channels {
		if c.Name == "" || c.ID == "" {
			log("Didn't get valid Channels structure back from discord\n")
			continue
		}
		log("  Identified channel '%s' (id=%s)\n", c.Name, c.ID)

		// This is pretty sloppy, but it'll do for now
		gi := getGuildById(g.ID)
		ci := getChannelById(c.ID)

		ch := &ChannelInfo{
			Guild:       gi,
			ChannelId:   ci.ChannelId,
			ChannelName: ci.ChannelName,
			IsPrivate:   false,
		}

		channelsById[c.ID] = ch
		channelsByName[cn{g.Name, c.Name}] = ch
	}
}

// Process incoming READY event (basically, when we connect)
func readyEvent(s *discordgo.Session, r *discordgo.Ready) {
	log("Handling 'Ready' event\n")

	discordReady = true
	for _, g := range r.Guilds {
		if g.Name == "" || g.ID == "" {
			log("Ready message had invalid Guilds, assuming GuildCreate events are incoming\n")
			continue
		}
		doGuild(g)
	}

	//	spew.Dump(channelsById)
}

// Process incoing GuildCreate event
func guildCreate(s *discordgo.Session, g *discordgo.GuildCreate) {
	log("Handling 'GuildCreate' event\n")
	doGuild(g.Guild)
}

// Is the user specified an admin?
func checkAdmin(user un) bool {
	if _, ok := validAdmins[user]; ok {
		return true
	}

	return false
}

// How I identify messages to me (hardcoded for now, though this should
// really configurable)
var reChannelMsgToMe = regexp.MustCompile(`(?i)^\s*(!herald)(?:bot)?$`)

// Handle 'MessageCreate' events (i.e. new messages to channel or bot)
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == BotID {
		return
	}

	c := getChannelById(m.ChannelID)

	var priv string
	if c.IsPrivate == true {
		priv = " (private)"
	}
	log("Handling 'MessageCreate' event, user: %s, channel: %s%s\n", m.Author.Username, m.ChannelID, priv)
	log("Message: %s\n", m.Content)

	// This little bit is pretty ugly.
	var cmd string
	var remain string

	split := strings.SplitN(m.Content, " ", 3)

	if c.IsPrivate == true {
		if len(split) >= 1 {
			cmd = strings.ToLower(split[0])
		}
		if len(split) >= 2 {
			remain = split[1]
		}
	} else {
		// Not private -- is it on a channel we monitor?
		ch := cn{c.Guild.GuildName, c.ChannelName}
		if _, ok := validChannels[ch]; !ok {
			log("Recieved message on unwatched channel: %s:%s\n",
				c.Guild.GuildName, c.ChannelName)
			return
		}

		if !reChannelMsgToMe.MatchString(split[0]) {
			return
		}

		if len(split) >= 2 {
			cmd = strings.ToLower(split[1])
		}

		if len(split) >= 3 {
			remain = split[2]
		}
	}

	// Dispatch command
	if f, ok := commandTable[cmd]; ok {
		if f.adminOnly == true && !checkAdmin(un{m.Author.Username, m.Author.Discriminator}) {
			log("Priv'd command %s attempted by unpriv'd user %s#%s\n",
				cmd, m.Author.Username, m.Author.Discriminator)
			return
		}
		f.handler(s, m, cmd, remain)
	}

	return
}

func sendFormatted(s *discordgo.Session, cid string, format string, vals ...interface{}) {
	_, err := s.ChannelMessageSend(cid, fmt.Sprintf(format, vals...))
	if err != nil {
		log("sendFormatted: Failed to send message: %s\n", err)
	}
}
