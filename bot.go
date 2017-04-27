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
	"github.com/davecgh/go-spew/spew"
)

var version string

func check(e error) {
	if e != nil {
		panic(e)
	}
}

var BotID string

type ChannelInfo struct {
	Guild       *GuildInfo
	ChannelName string
	ChannelId   string
	IsPrivate   bool
}

type GuildInfo struct {
	GuildName string
	GuildId   string
}

type cn struct {
	GuildName   string
	ChannelName string
}

type un struct {
	UserName string
	UserDisc string
}

var channelsById = map[string]*ChannelInfo{}
var channelsByName = map[cn]*ChannelInfo{}
var guildsById = map[string]*GuildInfo{}
var guildsByName = map[string]*GuildInfo{}

var validChannels = map[cn]bool{
// cn{"WWP", "general"}: true,
}

var validAdmins = map[un]bool{
// ch{"Alinsa", "1234"}: true,
}

var announceChannels = []cn{
// cn{"WWP", "general"},
}
var dg *discordgo.Session

func cleanup() {
	fmt.Printf("Termination signal received\n")
	if dg != nil {
		fmt.Printf("Shutting down open Discord connections\n")
		dg.Close()
	}
}

func main() {
	var err error
	initopts()

	if version == "" {
		version = "[unknown version]"
	}

	fmt.Printf("HeraldBot %s starting up...\n", version)
	dg, err = discordgo.New("Bot " + opts.Token)
	check(err)

	err = dg.Open()
	check(err)

	u, err := dg.User("@me")
	check(err)

	//	spew.Dump(dg)

	BotID = u.ID

	// time.Sleep(1000 * time.Millisecond)
	dg.AddHandler(readyEvent)
	dg.AddHandler(messageCreate)
	dg.AddHandler(guildCreate)

	//	a, err := dg.Applications()
	//	spew.Dump(a)

	time.Sleep(2 * time.Second)
	// spew.Dump(channelsByName)
	q := getChannelByName("WWP", "general")
	if q == nil {
		fmt.Printf("Couldn't find channel to emote to\n")
		os.Exit(1)
	}
	fmt.Printf("Trying to write to channel id %s\n", q.ChannelId)
	sendFormatted(dg, q.ChannelId, "HeraldBot **%s** reporting for duty!", version)

	// _, _ = dg.ChannelMessageSend(q.ChannelId, "Hi")
	// _, _ = dg.ChannelMessageSend("306187245986119691", "https://www.youtube.com/watch?v=CEH2HyVnKQM")

	// spew.Dump(channelsById)

	fmt.Printf("Bot running, press CTRL-C to exit\n")
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// 	watchPatreon()

	<-c
	cleanup()
	return
}

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
		fmt.Printf("getGuildById couldn't get guild info for guild id %s\n", gid)
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

func getChannelByName(guild string, channel string) *ChannelInfo {
	if c, ok := channelsByName[cn{guild, channel}]; ok {
		return c
	}

	return nil
}

func getChannelById(cid string) *ChannelInfo {
	if c, ok := channelsById[cid]; ok {
		return c
	}

	c, err := dg.Channel(cid)
	if err != nil {
		fmt.Printf("getChannelById couldn't get channel info for channel id %s\n", cid)
		return nil
	}

	g := getGuildById(c.GuildID)
	if g == nil {
		fmt.Printf("Couldn't get guild info for channel")
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

func doGuild(g *discordgo.Guild) {
	fmt.Printf("Identified guild '%s' (id=%s)\n", g.Name, g.ID)
	for _, c := range g.Channels {
		if c.Name == "" || c.ID == "" {
			fmt.Printf("Didn't get valid Channels structure back from discord\n")
			continue
		}

		fmt.Printf("  Identified channel '%s' (id=%s)\n", c.Name, c.ID)

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

func readyEvent(s *discordgo.Session, r *discordgo.Ready) {
	fmt.Printf("Handling 'Ready' event\n")
	// fmt.Printf("One: \n")
	// spew.Dump(r)
	// fmt.Printf("Two: \n")
	// spew.Dump(r.Guilds)
	for _, g := range r.Guilds {
		// spew.Dump(g)
		if g.Name == "" || g.ID == "" {
			fmt.Printf("Ready message had invalid Guilds, assuming GuildCreate events are incoming\n")
			continue
		}
		doGuild(g)
	}

	//	spew.Dump(channelsById)
}

func guildCreate(s *discordgo.Session, g *discordgo.GuildCreate) {
	fmt.Printf("Handling 'GuildCreate' event\n")
	doGuild(g.Guild)
	//	spew.Dump(channelsById)
	//	spew.Dump(channelsByName)
}

func checkAdmin(user un) bool {
	if _, ok := validAdmins[user]; ok {
		return true
	}

	return false
}

var reChannelMsgToMe = regexp.MustCompile(`(?i)^\s*(!herald)(?:bot)?$`)

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == BotID {
		return
	}

	c := getChannelById(m.ChannelID)

	var priv string
	if c.IsPrivate == true {
		priv = " (private)"
	}
	fmt.Printf("Handling 'MessageCreate' event, channel: %s%s\n", m.ChannelID, priv)

	// fmt.Printf("Handling MessageCreate event\n")
	//	spew.Dump(m)
	// z, _ := dg.Channel(m.ChannelID)
	// spew.Dump(z)
	// spew.Dump(channelsById)

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
			fmt.Printf("Recieved message on unwatched channel: %s:%s\n",
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

	if f, ok := commandTable[cmd]; ok {
		if f.adminOnly == true && !checkAdmin(un{m.Author.Username, m.Author.Discriminator}) {
			fmt.Fprintf(os.Stderr, "Priv'd command %s attempted by unpriv'd user %s#%s\n",
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
		fmt.Printf("sendFormatted: Failed to send message: %s\n", err)
	}
}

type commandHandler func(*discordgo.Session, *discordgo.MessageCreate, string, string)
type commandEntry struct {
	handler   commandHandler
	adminOnly bool
}

var commandTable = map[string]commandEntry{
	"ping":    {cmdPing, false},
	"pong":    {cmdPong, false},
	"help":    {cmdHelp, false},
	"debug":   {cmdDebug, true},
	"patreon": {cmdPatreon, true},
	"die":     {cmdDie, true},
}

func cmdPing(s *discordgo.Session, m *discordgo.MessageCreate, cmd string, remain string) {
	sendFormatted(s, m.ChannelID, "Pong!")
}

func cmdPong(s *discordgo.Session, m *discordgo.MessageCreate, cmd string, remain string) {
	sendFormatted(s, m.ChannelID, "Ping!")
}

func cmdHelp(s *discordgo.Session, m *discordgo.MessageCreate, cmd string, remain string) {
	sendFormatted(s, m.ChannelID, "HeraldBot **%s** reporting for duty!\n\n"+
		"Sorry, but so far, I am helpless.\n", version)
}

func cmdDebug(s *discordgo.Session, m *discordgo.MessageCreate, cmd string, remain string) {
	//    msg := fmt.Sprintf("Debug output for HeraldBot **%s**\n\n", version)
	//	_, _ = s.ChannelMessageSend(m.ChannelID, "Ping!")
	sendFormatted(s, m.ChannelID, "Debug for %s\n\n", version)
	sendFormatted(s, m.ChannelID, "```Go\n%s\n```\n", spew.Sdump(m))
	//    sendFormatted(s, m.ChannelID, "```\n%s\n```\n", sspew.Sprint(m))

	// spew.Dump(m)
}

func cmdDie(s *discordgo.Session, m *discordgo.MessageCreate, cmd string, remain string) {
	sendFormatted(s, m.ChannelID, "Goodbye cruel world!")
	cleanup()
	os.Exit(0)
}

func cmdPatreon(s *discordgo.Session, m *discordgo.MessageCreate, cmd string, remain string) {
	watchPatreon()
}
