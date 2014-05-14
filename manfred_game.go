package main

import (
	"encoding/json"
	"github.com/dchest/uniuri"
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
	PlayerCount  int
}

func (g ManfredGame) GetGameKey() GameKey {
	return GameKey(g.UUID)
}

func (g ManfredGame) GetChosenPlayersSetKey() string {
	return "game:" + g.UUID + ":players:chosen"
}
func (g ManfredGame) GetPossiblePlayersSetKey() string {
	return "game:" + g.UUID + ":players:possible"
}

func (g ManfredGame) AddTestPlayer(game string, c redis.Conn) {
	twitchId := uniuri.NewLen(12)

	player := ManfredPlayer {}
	player.Handles = make(map[string]string)
	player.Handles["TWITCH"] = twitchId
	player.Handles[game] = twitchId

	SaveManfredPlayer(player, TwitchUserKey(twitchId), c)

	_, err := c.Do("SADD", g.GetPossiblePlayersSetKey(), twitchId)
	c.Do("EXPIRE", g.GetPossiblePlayersSetKey(), 43200) // Expire after 1 day

	if err != nil {
		log.Fatal(err)
	}
}

func (g ManfredGame) AddPlayer(twitchId string, c redis.Conn) {
	_, err := c.Do("SADD", g.GetPossiblePlayersSetKey(), twitchId)
	c.Do("EXPIRE", g.GetPossiblePlayersSetKey(), 43200) // Expire after 1 day

	if err != nil {
		log.Fatal(err)
	}
}

func (g ManfredGame) CountPlayersReady(c redis.Conn) int64 {
	resp, err := c.Do("SCARD", g.GetPossiblePlayersSetKey())

	if err != nil {
		log.Fatal(err)
	}

	return resp.(int64)
}

func (g ManfredGame) ChooseAndGetPlayers(c redis.Conn) (result []TwitchUserKey) {
	resp, err := c.Do("SMEMBERS", g.GetChosenPlayersSetKey())
	if err != nil {
		log.Fatal(err)
	}

	result = make([]TwitchUserKey, g.PlayerCount)

	count := 1
	for _, h := range resp.([]interface{}) {
		id, ok := h.([]byte)
		if !ok {
			log.Fatal("Bad handle from redis for game: " + g.GetChosenPlayersSetKey())
		}

		// We only want to take the first PlayerCount number of players, and the rest we want to remove from the set
		// This handles the case where a users decreses the number of people they want to select for the game
		if count <= g.PlayerCount {
			result[count-1] = TwitchUserKey(id)
		} else {
			_, err := c.Do("SREM", g.GetChosenPlayersSetKey(), id)
			if err != nil {
				log.Fatal(err)
			}
		}
		count++
	}

	if count <= g.PlayerCount {
		diffResp, err := c.Do("SDIFF", g.GetPossiblePlayersSetKey(), g.GetChosenPlayersSetKey())
		if err != nil {
			log.Fatal(err)
		}

		newPlayers := diffResp.([]interface{})

		for i := 0; i < len(newPlayers) && count <= g.PlayerCount; i, count = i+1, count+1 {
			id := newPlayers[i].([]byte)
			result[count-1] = TwitchUserKey(id)
			_, err := c.Do("SADD", g.GetChosenPlayersSetKey(), id)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	return
}

func (g ManfredGame) Save(c redis.Conn) error {
	j, err := json.Marshal(g)
	if err != nil {
		return err
	}

	key := GameKey(g.UUID)
	c.Do("SET", key, j)
	c.Do("EXPIRE", key, 43200) // Expire after 1 day

	return nil
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

	return
}
