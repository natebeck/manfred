package main

import (
	_ "github.com/codegangsta/envy/autoload"
	"github.com/codegangsta/envy/lib"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/oauth2"
	"github.com/martini-contrib/sessions"
)

var TwitchClient = envy.MustGet("TWITCH_CLIENT")
var TwitchSecret = envy.MustGet("TWITCH_SECRET")
var TwitchRedirect = envy.MustGet("TWITCH_REDIRECT")
var SessionSecret = envy.MustGet("SESSION_SECRET")

func main() {
	oauth2.PathCallback = "/auth/twitch"

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
		return ":) You Did It!"
	})

	m.Run()
}

func TwitchOAuth(opts *oauth2.Options) martini.Handler {
	opts.AuthUrl = "https://api.twitch.tv/kraken/oauth2/authorize"
	opts.TokenUrl = "https://api.twitch.tv/kraken/oauth2/token"
	return oauth2.NewOAuth2Provider(opts)
}
