package main

import (
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
)

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
	"die":     {cmdDie, true},
	"say":     {cmdSay, true},
	"patreon": {cmdPatreon, true},
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
	//  _, _ = s.ChannelMessageSend(m.ChannelID, "Ping!")
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

func cmdSay(s *discordgo.Session, m *discordgo.MessageCreate, cmd string, remain string) {
	for _, ch := range announceChannels {
		sendFormatted(dg, getChannelByName(ch.GuildName, ch.ChannelName).ChannelId,
			remain)
	}
}

func cmdPatreon(s *discordgo.Session, m *discordgo.MessageCreate, cmd string, remain string) {
	patreonCheck(false)
}
