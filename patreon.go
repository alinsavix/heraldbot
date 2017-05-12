package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "github.com/mattn/go-sqlite3"
)

// Stolen from debugging info on the patreon webpage. Will probably change
// out from underneath us at some point.
var patreon_url = `https://api.patreon.com/stream?include=user.null%2Cattachments.null%2Cuser_defined_tags.null%2Ccampaign.earnings_visibility%2Cpoll&fields[post]=change_visibility_at%2Ccomment_count%2Ccontent%2Ccurrent_user_can_delete%2Ccurrent_user_can_view%2Ccurrent_user_has_liked%2Cearly_access_min_cents%2Cembed%2Cimage%2Cis_paid%2Clike_count%2Cmin_cents_pledged_to_view%2Cpost_file%2Cpublished_at%2Cpatron_count%2Cpatreon_url%2Cpost_type%2Cpledge_url%2Cthumbnail_url%2Ctitle%2Cupgrade_url%2Curl&fields[user]=image_url%2Cfull_name%2Curl&fields[campaign]=earnings_visibility&page[cursor]=null&filter[is_by_creator]=false&filter[is_following]=false&filter[contains_exclusive_posts]=false&filter[creator_id]=136449&json-api-version=1.0`

// Set up database and create tables that don't exist
func patreonDbInit(path string) bool {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		log("couldn't open sqlite db: %v\n", err)
	}
	defer db.Close()

	sqlCreate := `
		CREATE TABLE IF NOT EXISTS patreonlog (
			postid VARCHAR(45) PRIMARY KEY,
			ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`
	_, err = db.Exec(sqlCreate)
	if err != nil {
		log("Couldn't init sqlite db: %v\n", err)
		return false
	}

	log("Database initialized\n")

	return true
}

var patreonDbInitialized = false // Only need to initialize db once per run

// Open the database for use
func patreonDbOpen(path string) *sql.DB {
	if patreonDbInitialized == false {
		if patreonDbInit(path) == false {
			log("db was never initialized, can't open")
			return nil
		}
		patreonDbInitialized = true
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		log("couldn't open sqlite db: %v\n", err)
		return nil
	}

	return db
}

// Check to see if we've already seen a patreon post with a given id
func patreonDbCheck(db *sql.DB, id string) (bool, error) {
	var count int

	err := db.QueryRow("SELECT COUNT(postid) AS rowcount FROM patreonlog WHERE postid=?", id).Scan(&count)
	if err != nil {
		log("patreonDbCheck failed: %v\n", err)
		return false, err
	}

	if count > 0 {
		return true, nil
	}

	return false, nil
}

// Mark a given post id as having been seen
func patreonDbSet(db *sql.DB, id string) (bool, error) {
	_, err := db.Exec("INSERT INTO patreonlog (postid) VALUES(?)", id)

	if err != nil {
		log("patreonDbSet failed: %v\n", err)
		return false, err
	}

	return true, nil
}

// Sit in a loop and check patreon every 10 minutes for new community postings
//
// FIXME: The time delay should really be configurable
func patreonWatch() {
	db := patreonDbOpen(opts.Database)
	if db == nil {
		log("No database, disabling patreon watcher\n")
		return
	}
	db.Close()

	for {
		log("Performing scheduled patreon check\n")
		patreonCheck(true)
		time.Sleep(10 * time.Minute)
	}
}

var reHtmlStrip = regexp.MustCompile(`<[^>]*>`)
var reHtmlBr = regexp.MustCompile(`(?i)<br/?>`)
var reHtmlP = regexp.MustCompile(`(?i)<p/?>`)
var reMultiLinefeed = regexp.MustCompile(`\n\n+`)

// This is ugly and stupid, but basically format things for display. There's
// probably a far better way to do this, really, but I don't know what it is
// off the top of my head.
func patreonDiscordFormat(post Attributes) (string, *discordgo.MessageEmbed) {
	u := getUserById(post.UserId)
	name := ""
	if u != nil && u.Data.FullName != "" {
		name = u.Data.FullName
	}

	title := ""
	if post.Title != "" {
		title = post.Title
	}

	content := reHtmlBr.ReplaceAllLiteralString(post.Content, "\n")
	content = reHtmlP.ReplaceAllLiteralString(content, "\n\n")
	content = reHtmlStrip.ReplaceAllLiteralString(content, "")
	content = html.UnescapeString(content)
	content = reMultiLinefeed.ReplaceAllLiteralString(content, "\n\n")

	if len(content) > 1500 {
		content = content[0:1500] + "[...]"
	}

	if name == "" {
		name = "(unknown user)"
	}

	ann := fmt.Sprintf(
		"Hear ye, hear ye! There is a new Patreon community "+
			"wall post from **%s**!", name)

	em := &discordgo.MessageEmbed{
		Description: content,
		URL:         post.URL,
		Title:       post.URL,
		Type:        "rich",
	}
	if title != "" {
		em.Description = "**" + title + "**\n\n" + em.Description
	}

	return ann, em
}

// Not sure sqlite bits are threadsafe, so just in case, since this routine
// shouldn't be called much anyhow
var patreonCheckMutex = &sync.Mutex{}

// Perform actual patreon check. The "quiet" flag means it won't announce
// zero-new-messages to the public
func patreonCheck(quiet bool) {
	patreonCheckMutex.Lock()
	defer patreonCheckMutex.Unlock()

	posts := getPosts(patreon_url)
	db := patreonDbOpen(opts.Database)
	if db == nil {
		log("Can't open database, not running patreon check\n")
		return
	}
	defer db.Close()

	count := 0
	for id, p := range posts {
		check, err := patreonDbCheck(db, id)
		if err == nil && check == false {
			if count > 0 { // If we're doing a lot, space them out
				time.Sleep(5 * time.Second)
			}
			patreonDbSet(db, id)
			log("Haven't seen post id %s before, announcing\n", id)

			ann, emb := patreonDiscordFormat(p)
			for _, ch := range announceChannels {
				sendFormatted(dg, getChannelByName(ch.GuildName, ch.ChannelName).ChannelId, ann)
				dg.ChannelMessageSendEmbed(getChannelByName(ch.GuildName, ch.ChannelName).ChannelId, emb)
			}
			count++
		}
	}

	if count == 0 && quiet == false {
		for _, ch := range announceChannels {
			sendFormatted(dg, getChannelByName(ch.GuildName, ch.ChannelName).ChannelId,
				"Manual check: No new Patreon community posts to report!")
		}
	}
}

// Get current community posts from patreon
func getPosts(url string) map[string]Attributes {
	resp, err := http.Get(url)
	if err != nil {
		log("url fetch error: request: %v\n", err)
		return nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		log("url fetch error: reading: %v\n", err)
		return nil
	}

	// fmt.Printf("%s", b)

	var p PatreonCommunityPosts
	err = json.Unmarshal(body, &p)

	ret := make(map[string]Attributes)
	for _, v := range p.Data {
		// We mostly just want the Attributes structure, so we cheat and
		// add a field to it for the posting user's ID, rather than
		// including everything else we don't care about
		v.Attributes.UserId = v.Relationships.User.Data.ID
		ret[v.ID] = v.Attributes
	}

	return ret
}

// Get a patreon user's info, based on patreon user id
func getUserById(id string) *PatreonUser {
	url := fmt.Sprintf("https://api.patreon.com/user/%s", id)

	resp, err := http.Get(url)
	if err != nil {
		log("url fetch error: request: %v\n", err)
		return nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		log("url fetch error: reading: %v\n", err)
		return nil
	}

	var u PatreonUser
	err = json.Unmarshal(body, &u)

	if err != nil {
		log("couldn't unmarshal json user response: %v\n", err)
		return nil
	}

	return &u
}
