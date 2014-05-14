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
	"log"
	"net/http"
	"strconv"
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

		EnsureSessionVariables(s, t)

		r.HTML(200, "home", NewTemplateData(s))
	})

	m.Get("/newGame", oauth2.LoginRequired, func(s sessions.Session, r render.Render) {
		r.HTML(200, "newGame", NewTemplateData(s))
	})

	m.Post("/createGame", oauth2.LoginRequired, func(c redis.Conn, t oauth2.Tokens, r render.Render, req *http.Request) {
		u, err := GetTwitchUser(t.Access())
		if err != nil {
			log.Fatal(err)
		}

		uuid := uniuri.NewLen(8)
		mg := ManfredGame{}
		mg.UUID = uuid
		mg.StreamerName = u.DisplayName
		mg.Description = req.PostFormValue("description")
		mg.Game = req.PostFormValue("game")
		mg.PlayerCount = 7 // At some point this could be dynamic based off of the game
		// mg.MustFollow = req.PostFormValue("mustFollow")
		// mg.MustSub = req.PostFormValue("mustSub")

		mg.Save(c)

		r.Redirect("/game/" + uuid)
	})

	m.Put("/game/:gameId", oauth2.LoginRequired, func(s sessions.Session, r render.Render, p martini.Params, req *http.Request) (int, string) {
		count, err := strconv.ParseInt(req.PostFormValue("playerCount"), 10, 8)
		if err != nil {
			return 404, "{\"success\": false, \"error\": \"" + err.Error() + "\"}"
		}

		key := GameKey(p["gameId"])
		game := LoadManfredGame(key, c)

		if game == nil {
			return 404, "{\"success\": false, \"error\": \"Unable to find a game with that id\"}"
		}

		game.PlayerCount = int(count)
		game.Save(c) // This will extend the expiration time of the game. We might not want that...

		return 200, "{\"success\": true}"
	})

	m.Get("/game/:gameId/info", oauth2.LoginRequired, func(s sessions.Session, r render.Render, p martini.Params, req *http.Request) (int, string) {
		key := GameKey(p["gameId"])
		game := LoadManfredGame(key, c)

		if game == nil {
			return 404, "{\"success\": false, \"error\": \"Unable to find a game with that id\"}"
		}

		playerIds := game.ChooseAndGetPlayers(c)

		players := make([]ManfredPlayer, len(playerIds))

		for i, id := range playerIds {
			p := LoadManfredPlayer(id, c)
			if p != nil {
				players[i] = *p
			}
		}

		info := struct {
			PlayerCount int64
			Players     []ManfredPlayer
		}{game.CountPlayersReady(c), players}

		infoString, err := json.Marshal(info)
		if err != nil {
			log.Fatal(err)
		}

		return 200, string(infoString)
	})

	m.Get("/game/:gameId", oauth2.LoginRequired, func(s sessions.Session, r render.Render, p martini.Params, req *http.Request) {
		key := GameKey(p["gameId"])
		game := LoadManfredGame(key, c)

		playerIds := game.ChooseAndGetPlayers(c)

		players := make([]ManfredPlayer, len(playerIds))

		for i, id := range playerIds {
			p := LoadManfredPlayer(id, c)
			if p != nil {
				players[i] = *p
			}
		}

		templateData := NewTemplateData(s)
		templateData.Data = struct {
			Game        ManfredGame
			GameUrl     string
			PlayerCount int64
			Players     []ManfredPlayer
		}{*game, fmt.Sprintf("http://%s/play/%s", req.Host, p["gameId"]), game.CountPlayersReady(c), players}

		r.HTML(200, "game", templateData)
	})

	m.Get("/play/:gameId", oauth2.LoginRequired, func(s sessions.Session, t oauth2.Tokens, r render.Render, p martini.Params) {
		EnsureSessionVariables(s, t)
		templateData := NewTemplateData(s)

		gameKey := GameKey(p["gameId"])
		game := LoadManfredGame(gameKey, c)

		playerKey := TwitchUserKey(strconv.FormatInt(s.Get("twitchId").(int64), 10)) // There's got to be a better way to do this...
		player := LoadManfredPlayer(playerKey, c)

		setupUrl := "/play/" + p["gameId"] + "/setup"

		if game == nil {
			r.HTML(404, "missing_game", templateData)
			return
		}

		if player == nil {
			r.Redirect(setupUrl)
			return
		}

		handle, ok := player.Handles[game.Game]

		if !ok || handle == "" {
			r.Redirect(setupUrl)
			return
		}

		log.Println("Here be the handle! " + handle)

		playerTwitchId := strconv.FormatInt(s.Get("twitchId").(int64), 10)

		game.AddPlayer(playerTwitchId, c)
		game.AddTestPlayer(game.Game, c) // Add this for testing so that the number of players will always increase

		templateData.Data = struct {
			Game     ManfredGame
			Handle   string
			SetupUrl string
		}{*game, handle, setupUrl}
		r.HTML(200, "play", templateData)
	})

	m.Get("/play/:gameId/setup", oauth2.LoginRequired, func(s sessions.Session, t oauth2.Tokens, r render.Render, p martini.Params) {
		EnsureSessionVariables(s, t)

		templateData := NewTemplateData(s)

		gameKey := GameKey(p["gameId"])
		game := LoadManfredGame(gameKey, c)

		playerKey := TwitchUserKey(strconv.FormatInt(s.Get("twitchId").(int64), 10)) // There's got to be a better way to do this...
		player := LoadManfredPlayer(playerKey, c)

		currentHandle := ""

		if game == nil {
			r.HTML(404, "missing_game", templateData)
			return
		}

		if player != nil {
			currentHandle = player.Handles[game.Game]
		}

		templateData.Data = struct {
			Game          ManfredGame
			CurrentHandle string
		}{*game, currentHandle}
		r.HTML(200, "setup_player", templateData)
	})

	m.Post("/updateHandle", oauth2.LoginRequired, func(c redis.Conn, t oauth2.Tokens, r render.Render, req *http.Request, s sessions.Session) {

		playerKey := TwitchUserKey(strconv.FormatInt(s.Get("twitchId").(int64), 10)) // There's got to be a better way to do this...
		player := LoadManfredPlayer(playerKey, c)

		if player == nil {
			player = new(ManfredPlayer)
			player.Handles = make(map[string]string)
			player.Handles["TWITCH"] = s.Get("userName").(string)
		}

		player.Handles[req.PostFormValue("game")] = req.PostFormValue("handle") // Is there some sort of validation / santization we should be doing here?

		SaveManfredPlayer(*player, playerKey, c)

		gameId := req.PostFormValue("gameId")

		if gameId == "" {
			r.Redirect("/settings")
		} else {
			r.Redirect("/play/" + gameId)
		}
	})

	m.Run()
}

func EnsureSessionVariables(s sessions.Session, t oauth2.Tokens) {
	userName := s.Get("userName")
	twitchId := s.Get("twitchId")
	if userName == nil || twitchId == nil {
		u, err := GetTwitchUser(t.Access())
		if err != nil {
			log.Fatal(err)
		}

		s.Set("userName", u.DisplayName)
		s.Set("twitchId", u.Id)
	}
}

func TwitchOAuth(opts *oauth2.Options) martini.Handler {
	opts.AuthUrl = "https://api.twitch.tv/kraken/oauth2/authorize"
	opts.TokenUrl = "https://api.twitch.tv/kraken/oauth2/token"
	return oauth2.NewOAuth2Provider(opts)
}

func NewTemplateData(s sessions.Session) TemplateData {
	return TemplateData{
		Name: s.Get("userName").(string),
	}
}

type TemplateData struct {
	Name string
	Data interface{}
}
