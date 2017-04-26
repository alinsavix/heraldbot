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

var version string

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// import (
// 	"encoding/json"
// 	"fmt"
// 	"io/ioutil"
// 	"net/http"
// 	"os"

// 	"github.com/davecgh/go-spew/spew"
// )

// var url = `https://api.patreon.com/stream?include=user.null%2Cattachments.null%2Cuser_defined_tags.null%2Ccampaign.earnings_visibility%2Cpoll&fields[post]=change_visibility_at%2Ccomment_count%2Ccontent%2Ccurrent_user_can_delete%2Ccurrent_user_can_view%2Ccurrent_user_has_liked%2Cearly_access_min_cents%2Cembed%2Cimage%2Cis_paid%2Clike_count%2Cmin_cents_pledged_to_view%2Cpost_file%2Cpublished_at%2Cpatron_count%2Cpatreon_url%2Cpost_type%2Cpledge_url%2Cthumbnail_url%2Ctitle%2Cupgrade_url%2Curl&fields[user]=image_url%2Cfull_name%2Curl&fields[campaign]=earnings_visibility&page[cursor]=null&filter[is_by_creator]=false&filter[is_following]=false&filter[contains_exclusive_posts]=false&filter[creator_id]=136449&json-api-version=1.0`

// func main() {
// 	resp, err := http.Get(url)
// 	if err != nil {
// 		fmt.Fprintf(os.Stderr, "fetch: %v\n", err)
// 		os.Exit(1)
// 	}

// 	b, err := ioutil.ReadAll(resp.Body)
// 	resp.Body.Close()

// 	if err != nil {
// 		fmt.Fprintf(os.Stderr, "fetch: reading: %v\n", err)
// 		os.Exit(1)
// 	}

// 	// fmt.Printf("%s", b)

// 	var zot interface{}
// 	err = json.Unmarshal(b, &zot)

// 	// fmt.Printf("%#v\n", zot)

// 	zotzot := zot.(map[string]interface{})
// 	spew.Dump(zotzot["data"])
// }

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

var channelsById = map[string]*ChannelInfo{}
var channelsByName = map[cn]*ChannelInfo{}
var guildsById = map[string]*GuildInfo{}
var guildsByName = map[string]*GuildInfo{}

var validChannels = map[cn]bool{
	cn{"WWP", "general"}: true,
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

	fmt.Printf("HeraldBot %s starting up...\n", version)

	dg, err = discordgo.New("Bot " + "MzA2MTg1ODk0NjkwMjkxNzE0.C-AZyA.JzWpgczE7vumYuwE_6f_5Is4MDo")
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

	// _, _ = dg.ChannelMessageSend("306187245986119691", "HeraldBot reporting for duty!")
	time.Sleep(2 * time.Second)
	// spew.Dump(channelsByName)
	q := getChannelByName("WWP", "general")
	if q == nil {
		fmt.Printf("Couldn't find channel to emote to\n")
		os.Exit(1)
	}
	fmt.Printf("Trying to write to channel id %s\n", q.ChannelId)
	_, _ = dg.ChannelMessageSend(q.ChannelId, "Hi")
	_, _ = dg.ChannelMessageSend("306187245986119691", "https://www.youtube.com/watch?v=CEH2HyVnKQM")

	// spew.Dump(channelsById)

	fmt.Printf("Bot running, press CTRL-C to exit\n")
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

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
		//		fmt.Printf("Message on private channel")

		if len(split) >= 1 {
			cmd = strings.ToLower(split[0])
		}
		if len(split) >= 2 {
			remain = split[1]
		}
	} else {
		if reChannelMsgToMe.MatchString(split[0]) {
			// fmt.Printf("Message to %s, cmd: %s, remain: %s\n", split[0], split[1], split[2])
			if len(split) >= 2 {
				cmd = strings.ToLower(split[1])
			}

			if len(split) >= 3 {
				remain = split[2]
			}
		} else {
			return
		}
	}

	if f, ok := commandTable[cmd]; ok {
		f(s, m, cmd, remain)
	}

	// _ = remain

	// if cmd == "ping" {
	// 	// _ = s.ChannelTyping("306187245986119691")
	// 	// time.Sleep(1000 * time.Millisecond)
	// 	_, _ = s.ChannelMessageSend(m.ChannelID, "Pong!")
	// }

	// if cmd == "pong" {
	// 	_, _ = s.ChannelMessageSend(m.ChannelID, "Ping!")
	// }

	// fmt.Printf("%20s %20s %20s > %s\n", m.ChannelID, time.Now().Format(time.Stamp),
	// 	m.Author.Username, m.Content)
	//	spew.Dump(m)
	//	z, _ := s.Channel(m.ChannelID)
	//	spew.Dump(z)

	//	g, _ := s.Guild(z.GuildID)
	//	spew.Dump(g)
}

type commandHandler func(*discordgo.Session, *discordgo.MessageCreate, string, string)

var commandTable = map[string]commandHandler{
	"ping": cmdPing,
	"pong": cmdPong,
	"help": cmdHelp,
}

func cmdPing(s *discordgo.Session, m *discordgo.MessageCreate, cmd string, remain string) {
	_, _ = s.ChannelMessageSend(m.ChannelID, "Pong!")
}

func cmdPong(s *discordgo.Session, m *discordgo.MessageCreate, cmd string, remain string) {
	_, _ = s.ChannelMessageSend(m.ChannelID, "Ping!")
}

func cmdHelp(s *discordgo.Session, m *discordgo.MessageCreate, cmd string, remain string) {
	msg := fmt.Sprintf("Heraldbot **%s** reporting for duty!\n\n"+
		"Sorry, but so far, I am helpless.\n", version)
	_, _ = s.ChannelMessageSend(m.ChannelID, msg)
}
