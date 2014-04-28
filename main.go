package main

import (
	"encoding/json"
	_ "github.com/codegangsta/envy/autoload"
	"github.com/codegangsta/envy/lib"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/oauth2"
	"github.com/martini-contrib/sessions"
	"io/ioutil"
	"net/http"
)

var TwitchClient = envy.MustGet("TWITCH_CLIENT")
var TwitchSecret = envy.MustGet("TWITCH_SECRET")
var TwitchRedirect = envy.MustGet("TWITCH_REDIRECT")
var SessionSecret = envy.MustGet("SESSION_SECRET")

func main() {
	oauth2.PathCallback = "/auth/twitch"
	oauth2.PathLogin = "/login"
	oauth2.PathLogout = "/logout"

	m := martini.Classic()
	m.Use(sessions.Sessions("manfred", sessions.NewCookieStore([]byte(SessionSecret))))

	m.Use(TwitchOAuth(&oauth2.Options{
		ClientId:     TwitchClient,
		ClientSecret: TwitchSecret,
		RedirectURL:  TwitchRedirect,
		Scopes:       []string{"user_subscriptions", "user_read", "channel_read"},
	}))

	m.Get("/", oauth2.LoginRequired, func(tokens oauth2.Tokens) string {
		if tokens.IsExpired() {
			return ":( You didn't do it!"
		}

		content, _ := getContent("https://api.twitch.tv/kraken/user?oauth_token=" + tokens.Access())

		var record TwitchUser
		json.Unmarshal(content, &record)

		return "Hello " + string(record.DisplayName) + "!"
	})

	m.Run()
}

func TwitchOAuth(opts *oauth2.Options) martini.Handler {
	opts.AuthUrl = "https://api.twitch.tv/kraken/oauth2/authorize"
	opts.TokenUrl = "https://api.twitch.tv/kraken/oauth2/token"
	return oauth2.NewOAuth2Provider(opts)
}

// This function fetch the content of a URL will return it as an
// array of bytes if retrieved successfully.
func getContent(url string) ([]byte, error) {
	// Build the request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	// Send the request via a client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	// Defer the closing of the body
	defer resp.Body.Close()
	// Read the content into a byte array
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// At this point we're done - simply return the bytes
	return body, nil
}

type TwitchUser struct {
	DisplayName string `json:"display_name"`
	Id          string `json:"_id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Email       string `json:"email"`
}
