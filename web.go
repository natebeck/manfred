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
	"io/ioutil"
	"log"
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
	// Redis
	c, err := redisurl.ConnectToURL(RedisURL)
	if err != nil {
		panic(err)
	}
	m.Map(c)

	// Routes
	m.Get("/", oauth2.LoginRequired, func(s sessions.Session, t oauth2.Tokens, r render.Render) {

		userName := s.Get("userName")
		if userName == nil {
			u, err := GetTwitchUser(t.Access())
			if err != nil {
				log.Fatal(err)
			}

			s.Set("userName", u.DisplayName)
			userName = u.DisplayName
		}

		r.HTML(200, "home", userName)
	})

	m.Get("/newGame", oauth2.LoginRequired, func(r render.Render) {
		r.HTML(200, "newGame", nil)
	})

	m.Post("/createGame", oauth2.LoginRequired, func(c redis.Conn, r render.Render, req *http.Request) {
		uuid := uniuri.NewLen(8)
		mg := ManfredGame{}
		mg.UUID = uuid
		mg.StreamerName = "Streamer"
		mg.Description = req.PostFormValue("description")
		mg.Game = req.PostFormValue("game")
		// mg.MustFollow = req.PostFormValue("mustFollow")
		// mg.MustSub = req.PostFormValue("mustSub")

		j, err := json.Marshal(mg)
		if err != nil {
			log.Fatal(err)
		}
		c.Do("SET", uuid, j)
		c.Do("EXPIRE", uuid, 43200) // Expire after 1 day
		r.Redirect("/thanks/" + uuid)
	})

	m.Get("/thanks/:gameId", oauth2.LoginRequired, func(r render.Render, p martini.Params, req *http.Request) {
		r.HTML(200, "thanks", fmt.Sprintf("http://%s/play/%s", req.Host, p["gameId"]))
	})

	m.Get("/play/:gameId", oauth2.LoginRequired, func(t oauth2.Tokens, r render.Render, p martini.Params) {
		r.HTML(200, "play", nil)
	})

	m.Run()
}

func GetTwitchUser(token string) (TwitchUser, error) {

	url := "https://api.twitch.tv/kraken/user"

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Set("Client-ID", TwitchClient)
	req.Header.Set("Authorization", "OAuth "+token)
	req.Header.Set("Accept", "application/vnd.twitchtv.v3+json")

	client := &http.Client{}
	resp, _ := client.Do(req)
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	var user TwitchUser
	json.Unmarshal(body, &user)

	return user, nil
}

func TwitchOAuth(opts *oauth2.Options) martini.Handler {
	opts.AuthUrl = "https://api.twitch.tv/kraken/oauth2/authorize"
	opts.TokenUrl = "https://api.twitch.tv/kraken/oauth2/token"
	return oauth2.NewOAuth2Provider(opts)
}

type TwitchUser struct {
	DisplayName string `json:"display_name"`
	Logo        string `json:"logo"`
	Id          string `json:"_id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Email       string `json:"email"`
}

type ManfredGame struct {
	UUID         string
	StreamerName string
	Description  string
	Game         string
	MustFollow   bool
	MustSub      bool
}
