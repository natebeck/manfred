package main

import (
	"encoding/json"
	"github.com/garyburd/redigo/redis"
	"log"
)

type GameKey string

func (g GameKey) String() string {
	return "game:" + string(g)
}

type ManfredGame struct {
	UUID         string
	StreamerName string
	Description  string
	Game         string
	MustFollow   bool
	MustSub      bool
}

func (g ManfredGame) GetGameKey() (key GameKey) {
	return GameKey(g.UUID)
}

func (g ManfredGame) GetPlayersSetKey() (key string) {
	return "game:" + g.UUID + ":players"
}

func (g ManfredGame) AddPlayer(handle string, c redis.Conn) {
	_, err := c.Do("SADD", g.GetPlayersSetKey(), handle)
	c.Do("EXPIRE", g.GetPlayersSetKey(), 43200) // Expire after 1 day

	if err != nil {
		log.Fatal(err)
	}
}

func (g ManfredGame) CountPlayersReady(c redis.Conn) int64 {
	resp, err := c.Do("SCARD", g.GetPlayersSetKey())

	if err != nil {
		log.Fatal(err)
	}

	return resp.(int64)
}

func (g ManfredGame) GetPlayers(c redis.Conn) (result []TwitchUserKey) {
	resp, err := c.Do("SMEMBERS", g.GetPlayersSetKey())

	if err != nil {
		log.Fatal(err)
	}

	for _, h := range resp.([]interface{}) {
		id, ok := h.([]byte)
		if !ok {
			log.Fatal("Bad handle from redis for game: " + g.GetPlayersSetKey())
		}
		result = append(result, TwitchUserKey(id))
	}

	return
}

func LoadManfredGame(key GameKey, c redis.Conn) (game *ManfredGame) {
	value, err := c.Do("GET", key)
	if err != nil {
		log.Fatal(err)
	}

	if value != nil {
		err = json.Unmarshal(value.([]byte), &game)
		if err != nil {
			log.Fatal(err)
		}
	}

	return game
}
