package main

import "time"

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
