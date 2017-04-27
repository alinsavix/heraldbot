package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/davecgh/go-spew/spew"
)

// Stolen from debugging info on the patreon webpage. Will probably change
// out from underneath us at some point.
var patreon_url = `https://api.patreon.com/stream?include=user.null%2Cattachments.null%2Cuser_defined_tags.null%2Ccampaign.earnings_visibility%2Cpoll&fields[post]=change_visibility_at%2Ccomment_count%2Ccontent%2Ccurrent_user_can_delete%2Ccurrent_user_can_view%2Ccurrent_user_has_liked%2Cearly_access_min_cents%2Cembed%2Cimage%2Cis_paid%2Clike_count%2Cmin_cents_pledged_to_view%2Cpost_file%2Cpublished_at%2Cpatron_count%2Cpatreon_url%2Cpost_type%2Cpledge_url%2Cthumbnail_url%2Ctitle%2Cupgrade_url%2Curl&fields[user]=image_url%2Cfull_name%2Curl&fields[campaign]=earnings_visibility&page[cursor]=null&filter[is_by_creator]=false&filter[is_following]=false&filter[contains_exclusive_posts]=false&filter[creator_id]=136449&json-api-version=1.0`

// Courtesy https://mholt.github.io/json-to-go/
type PatreonCommunityPosts struct {
	Data []struct {
		Attributes struct {
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
			Title                 interface{} `json:"title"`
			UpgradeURL            string      `json:"upgrade_url"`
			URL                   string      `json:"url"`
		} `json:"attributes"`
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

func patreonDbInit(path string) *sql.DB {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "couldn't open sqlite db: %v\n", err)
	}

	sqlCreate := `
		CREATE TABLE IF NOT EXISTS patreonlog (
			postid VARCHAR(45) PRIMARY KEY,
			ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`
	_, err = db.Exec(sqlCreate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't init sqlite db: %v\n", err)
		return nil
	}

	fmt.Fprintf(os.Stderr, "Database initialized\n")

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

func watchPatreon() {
	posts := getPostIds(patreon_url)
	db := patreonDbInit("./herald.db")
	if db == nil {
		fmt.Fprintf(os.Stderr, "No database, disabling patreon watcher")
		return
	}
	defer db.Close()

	count := 0
	for _, p := range posts {
		check, err := patreonDbCheck(db, p)
		if err == nil && check == false {
			patreonDbSet(db, p)
			fmt.Fprintf(os.Stderr, "Haven't seen post id %s before, announcing\n", p)
			for _, ch := range announceChannels {
				sendFormatted(dg, getChannelByName(ch.GuildName, ch.ChannelName).ChannelId,
					"Hear ye, hear ye, there is a new Patreon community wall post:\n"+
						"https://www.patreon.com/posts/%s", p)
			}
			count++
		}
	}

	if count == 0 {
		for _, ch := range announceChannels {
			sendFormatted(dg, getChannelByName(ch.GuildName, ch.ChannelName).ChannelId,
				"No new Patreon community posts to report!")
		}
	}
}

func getPostIds(url string) []string {
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

	var ret []string

	for _, v := range p.Data {
		ret = append(ret, v.ID)
	}
	// spew.Dump(zot.Data[0])
	fmt.Printf("Returning:\n")
	spew.Dump(ret)

	return ret
}
