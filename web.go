package main

import (
	"encoding/json"
	"fmt"
	_ "github.com/codegangsta/envy/autoload"
	"github.com/codegangsta/envy/lib"
	"github.com/dchest/uniuri"
	"github.com/garyburd/redigo/redis"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/oauth2"
	"github.com/martini-contrib/render"
	"github.com/martini-contrib/sessions"
	"github.com/soveran/redisurl"
	"net/http"
)

var TwitchClient = envy.MustGet("TWITCH_CLIENT")
var TwitchSecret = envy.MustGet("TWITCH_SECRET")
var TwitchRedirect = envy.MustGet("TWITCH_REDIRECT")
var SessionSecret = envy.MustGet("SESSION_SECRET")
var RedisURL = envy.MustGet("REDISCLOUD_URL")

func main() {
	oauth2.PathCallback = "/auth/twitch"
	oauth2.PathLogin = "/login"
	oauth2.PathLogout = "/logout"

	// Middleware
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

	// Services
	c, err := redisurl.ConnectToURL(RedisURL)
	if err != nil {
		panic(err)
	}

	m.Map(c)

	// Routes
	m.Get("/", oauth2.LoginRequired, func(r render.Render) {
		r.HTML(200, "home", "Friend")
	})

	m.Get("/newGame", oauth2.LoginRequired, func(r render.Render) {
		r.HTML(200, "newGame", nil)
	})

	m.Post("/createGame", oauth2.LoginRequired, func(c redis.Conn, r render.Render, req *http.Request) {
		uuid := uniuri.NewLen(12)
		mg := ManfredGame{}
		mg.UUID = uuid
		mg.StreamerName = "ApDrop"
		mg.Title = req.PostFormValue("title")
		mg.Game = req.PostFormValue("game")
		// mg.MustFollow = req.PostFormValue("mustFollow")
		// mg.MustSub = req.PostFormValue("mustSub")

		j, err := json.Marshal(mg)
		if err != nil {
			panic(err)
		}
		c.Do("SET", uuid, j)
		c.Do("EXPIRE", uuid, 86400) // Expire after 1 day
		r.Redirect("/thanks/" + uuid)
	})

	m.Get("/thanks/:gameId", oauth2.LoginRequired, func(r render.Render, p martini.Params, req *http.Request) {
		r.HTML(200, "thanks", fmt.Sprintf("http://%s/play/%s", req.Host, p["gameId"]))
	})

	m.Get("/play/:gameId", oauth2.LoginRequired, func(tokens oauth2.Tokens, r render.Render, p martini.Params) {
		r.HTML(200, "play", nil)
	})

	m.Run()
}

func TwitchOAuth(opts *oauth2.Options) martini.Handler {
	opts.AuthUrl = "https://api.twitch.tv/kraken/oauth2/authorize"
	opts.TokenUrl = "https://api.twitch.tv/kraken/oauth2/token"
	return oauth2.NewOAuth2Provider(opts)
}

type ManfredGame struct {
	UUID         string
	StreamerName string
	Title        string
	Game         string
	MustFollow   bool
	MustSub      bool
}
