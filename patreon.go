package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "github.com/mattn/go-sqlite3"
)

// Stolen from debugging info on the patreon webpage. Will probably change
// out from underneath us at some point.
var patreon_url = `https://api.patreon.com/stream?include=user.null%2Cattachments.null%2Cuser_defined_tags.null%2Ccampaign.earnings_visibility%2Cpoll&fields[post]=change_visibility_at%2Ccomment_count%2Ccontent%2Ccurrent_user_can_delete%2Ccurrent_user_can_view%2Ccurrent_user_has_liked%2Cearly_access_min_cents%2Cembed%2Cimage%2Cis_paid%2Clike_count%2Cmin_cents_pledged_to_view%2Cpost_file%2Cpublished_at%2Cpatron_count%2Cpatreon_url%2Cpost_type%2Cpledge_url%2Cthumbnail_url%2Ctitle%2Cupgrade_url%2Curl&fields[user]=image_url%2Cfull_name%2Curl&fields[campaign]=earnings_visibility&page[cursor]=null&filter[is_by_creator]=false&filter[is_following]=false&filter[contains_exclusive_posts]=false&filter[creator_id]=136449&json-api-version=1.0`

// Courtesy https://mholt.github.io/json-to-go/
type PatreonUser struct {
	Data struct {
		About     string      `json:"about"`
		Created   time.Time   `json:"created"`
		Facebook  interface{} `json:"facebook"`
		FirstName string      `json:"first_name"`
		FullName  string      `json:"full_name"`
		Gender    int         `json:"gender"`
		ID        string      `json:"id"`
		ImageURL  string      `json:"image_url"`
		LastName  string      `json:"last_name"`
		Links     struct {
			Campaign struct {
				ID interface{} `json:"id"`
			} `json:"campaign"`
		} `json:"links"`
		ThumbURL string      `json:"thumb_url"`
		Twitch   interface{} `json:"twitch"`
		Twitter  interface{} `json:"twitter"`
		Type     string      `json:"type"`
		URL      string      `json:"url"`
		Vanity   interface{} `json:"vanity"`
		Youtube  interface{} `json:"youtube"`
	} `json:"data"`
	Links struct {
		Self string `json:"self"`
	} `json:"links"`
}

type Attributes struct {
	ChangeVisibilityAt    interface{} `json:"change_visibility_at"`
	CommentCount          int         `json:"comment_count"`
	Content               string      `json:"content"`
	CurrentUserCanDelete  bool        `json:"current_user_can_delete"`
	CurrentUserCanView    bool        `json:"current_user_can_view"`
	CurrentUserHasLiked   bool        `json:"current_user_has_liked"`
	EarlyAccessMinCents   interface{} `json:"early_access_min_cents"`
	Embed                 interface{} `json:"embed"`
	Image                 interface{} `json:"image"`
	IsPaid                bool        `json:"is_paid"`
	LikeCount             int         `json:"like_count"`
	MinCentsPledgedToView int         `json:"min_cents_pledged_to_view"`
	PatreonURL            string      `json:"patreon_url"`
	PatronCount           interface{} `json:"patron_count"`
	PledgeURL             string      `json:"pledge_url"`
	PostFile              interface{} `json:"post_file"`
	PostType              string      `json:"post_type"`
	PublishedAt           time.Time   `json:"published_at"`
	Title                 string      `json:"title"`
	UpgradeURL            string      `json:"upgrade_url"`
	URL                   string      `json:"url"`
	UserId                string
}

type PatreonCommunityPosts struct {
	Data []struct {
		Attributes    `json:"attributes"`
		ID            string `json:"id"`
		Relationships struct {
			Attachments struct {
				Data []interface{} `json:"data"`
			} `json:"attachments"`
			Campaign struct {
				Data struct {
					ID   string `json:"id"`
					Type string `json:"type"`
				} `json:"data"`
				Links struct {
					Related string `json:"related"`
				} `json:"links"`
			} `json:"campaign"`
			Poll struct {
				Data interface{} `json:"data"`
			} `json:"poll"`
			User struct {
				Data struct {
					ID   string `json:"id"`
					Type string `json:"type"`
				} `json:"data"`
				Links struct {
					Related string `json:"related"`
				} `json:"links"`
			} `json:"user"`
			UserDefinedTags struct {
				Data []interface{} `json:"data"`
			} `json:"user_defined_tags"`
		} `json:"relationships"`
		Type string `json:"type"`
	} `json:"data"`
	Included []struct {
		Attributes struct {
			EarningsVisibility string `json:"earnings_visibility"`
		} `json:"attributes"`
		ID   string `json:"id"`
		Type string `json:"type"`
	} `json:"included"`
	Links struct {
		First string `json:"first"`
		Next  string `json:"next"`
	} `json:"links"`
	Meta struct {
		PostsCount int `json:"posts_count"`
	} `json:"meta"`
}

func patreonDbInit(path string) bool {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "couldn't open sqlite db: %v\n", err)
	}
	defer db.Close()

	sqlCreate := `
		CREATE TABLE IF NOT EXISTS patreonlog (
			postid VARCHAR(45) PRIMARY KEY,
			ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`
	_, err = db.Exec(sqlCreate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't init sqlite db: %v\n", err)
		return false
	}

	fmt.Fprintf(os.Stderr, "Database initialized\n")

	return true
}

var patreonDbInitialized = false

func patreonDbOpen(path string) *sql.DB {
	if patreonDbInitialized == false {
		if patreonDbInit(path) == false {
			fmt.Fprintf(os.Stderr, "db was never initialized, can't open")
			return nil
		}
		patreonDbInitialized = true
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "couldn't open sqlite db: %v\n", err)
		return nil
	}

	return db
}

func patreonDbCheck(db *sql.DB, id string) (bool, error) {
	var count int

	err := db.QueryRow("SELECT COUNT(postid) AS rowcount FROM patreonlog WHERE postid=?", id).Scan(&count)
	if err != nil {
		fmt.Fprintf(os.Stderr, "patreonDbCheck failed: %v\n", err)
		return false, err
	}

	if count > 0 {
		return true, nil
	}

	return false, nil
}

func patreonDbSet(db *sql.DB, id string) (bool, error) {
	_, err := db.Exec("INSERT INTO patreonlog (postid) VALUES(?)", id)

	if err != nil {
		fmt.Fprintf(os.Stderr, "patreonDbSet failed: %v\n", err)
		return false, err
	}

	return true, nil
}

func patreonWatch() {
	db := patreonDbOpen(opts.Database)
	if db == nil {
		fmt.Fprintf(os.Stderr, "No database, disabling patreon watcher\n")
		return
	}
	db.Close()

	for {
		fmt.Fprintf(os.Stderr, "Performing scheduled patreon check\n")
		patreonCheck(true)
		time.Sleep(10 * time.Minute)
	}
}

var reHtmlStrip = regexp.MustCompile(`<[^>]*>`)
var reHtmlBr = regexp.MustCompile(`(?i)<br/?>`)
var reHtmlP = regexp.MustCompile(`(?i)<p/?>`)
var reMultiLinefeed = regexp.MustCompile(`\n\n+`)

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
	//	 else {
	//		title = post.URL
	//	}

	// spew.Dump(post.Content)

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

	// sendFormatted(dg, getChannelByName(ch.GuildName, ch.ChannelName).ChannelId,
	// 	"Hear ye, hear ye, there is a new Patreon community wall post:\n"+
	// 		"https://www.patreon.com/posts/%s", id)

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

func patreonCheck(quiet bool) {
	patreonCheckMutex.Lock()
	defer patreonCheckMutex.Unlock()

	posts := getPosts(patreon_url)
	db := patreonDbOpen(opts.Database)
	if db == nil {
		fmt.Fprintf(os.Stderr, "Can't open database, not running patreon check\n")
		return
	}
	defer db.Close()

	count := 0
	for id, p := range posts {
		//		fmt.Fprintf(os.Stderr, "%s\n\n", spew.Sdump(p))
		check, err := patreonDbCheck(db, id)
		if err == nil && check == false {
			if count > 0 { // If we're doing a lot, space them out
				time.Sleep(5 * time.Second)
			}
			patreonDbSet(db, id)
			fmt.Fprintf(os.Stderr, "Haven't seen post id %s before, announcing\n", id)

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

func getPosts(url string) map[string]Attributes {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "url fetch error: request: %v\n", err)
		return nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		fmt.Fprintf(os.Stderr, "url fetch error: reading: %v\n", err)
		return nil
	}

	// fmt.Printf("%s", b)

	var p PatreonCommunityPosts
	err = json.Unmarshal(body, &p)

	ret := make(map[string]Attributes)
	for _, v := range p.Data {
		// fmt.Fprintf(os.Stderr, "%s\n\n", spew.Sdump())
		v.Attributes.UserId = v.Relationships.User.Data.ID
		ret[v.ID] = v.Attributes
	}

	return ret
}

func getUserById(id string) *PatreonUser {
	url := fmt.Sprintf("https://api.patreon.com/user/%s", id)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "url fetch error: request: %v\n", err)
		return nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		fmt.Fprintf(os.Stderr, "url fetch error: reading: %v\n", err)
		return nil
	}

	var u PatreonUser
	err = json.Unmarshal(body, &u)

	if err != nil {
		fmt.Fprintf(os.Stderr, "couldn't unmarshal json user response: %v\n", err)
		return nil
	}

	return &u
}
