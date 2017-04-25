package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

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

type bc struct {
	GuildName   string
	GuildId     string
	ChannelName string
	ChannelId   string
}

type cn struct {
	GuildName   string
	ChannelName string
}

var channelsById = map[string]*bc{}
var channelsByName = map[cn]*bc{}

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

	fmt.Printf("Bot running, press CTRL-C to exit\n")
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	cleanup()
	return
}

func doGuild(g *discordgo.Guild) {
	fmt.Printf("Identified guild '%s' (id=%s)\n", g.Name, g.ID)
	for _, c := range g.Channels {
		if c.Name == "" || c.ID == "" {
			fmt.Printf("Didn't get valid Channels structure back from discord\n")
			continue
		}

		fmt.Printf("  Identified channel '%s' (id=%s)\n", c.Name, c.ID)
		ch := &bc{g.ID, g.Name, c.ID, c.Name}
		channelsById[c.ID] = ch
		channelsByName[cn{g.Name, c.Name}] = ch
	}
}

func readyEvent(s *discordgo.Session, r *discordgo.Ready) {
	fmt.Printf("Handling Ready event\n")
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
	fmt.Printf("Handling GuildCreate event\n")
	doGuild(g.Guild)
	//	spew.Dump(channelsById)
	//	spew.Dump(channelsByName)
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == BotID {
		return
	}

	fmt.Printf("Handling MessageCreate event\n")

	if m.Content == "ping" {
		_ = s.ChannelTyping("306187245986119691")
		time.Sleep(1000 * time.Millisecond)
		_, _ = s.ChannelMessageSend(m.ChannelID, "Pong!")
	}

	if m.Content == "pong" {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Ping!")
	}

	// fmt.Printf("%20s %20s %20s > %s\n", m.ChannelID, time.Now().Format(time.Stamp),
	// 	m.Author.Username, m.Content)
	//	spew.Dump(m)
	//	z, _ := s.Channel(m.ChannelID)
	//	spew.Dump(z)

	//	g, _ := s.Guild(z.GuildID)
	//	spew.Dump(g)
}
