package main

import (
	_ "github.com/codegangsta/envy/autoload"
	"github.com/codegangsta/envy/lib"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/oauth2"
	"github.com/martini-contrib/render"
	"github.com/martini-contrib/sessions"
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
	m.Use(render.Renderer(render.Options{
		Layout: "layout",
	}))

	m.Use(TwitchOAuth(&oauth2.Options{
		ClientId:     TwitchClient,
		ClientSecret: TwitchSecret,
		RedirectURL:  TwitchRedirect,
		Scopes:       []string{"user_read", "user_subscriptions"},
	}))

	m.Get("/", oauth2.LoginRequired, func(tokens oauth2.Tokens, r render.Render) {
		r.HTML(200, "home", "Brad")
	})

	m.Get("/create", oauth2.LoginRequired, func() string {
		return "Which Game?"
	})

	m.Run()
}

func TwitchOAuth(opts *oauth2.Options) martini.Handler {
	opts.AuthUrl = "https://api.twitch.tv/kraken/oauth2/authorize"
	opts.TokenUrl = "https://api.twitch.tv/kraken/oauth2/token"
	return oauth2.NewOAuth2Provider(opts)
}
