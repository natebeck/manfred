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
func (g ManfredGame) GetUnchosenPlayersSetKey() string {
	return "game:" + g.UUID + ":players:unchosen"
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

	SaveManfredPlayer(player, ConvertToTwitchUserKey(twitchId), c)

	_, err := c.Do("SADD", g.GetPossiblePlayersSetKey(), ConvertToTwitchUserKey(twitchId))
	c.Do("EXPIRE", g.GetPossiblePlayersSetKey(), 43200) // Expire after 1 day

	if err != nil {
		log.Fatal(err)
	}
}

func (g ManfredGame) AddPlayer(twitchId TwitchUserKey, c redis.Conn) {
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

func (g ManfredGame) ChoosePlayers(c redis.Conn) error {
	_, err := c.Do("DEL", g.GetChosenPlayersSetKey())
	if err != nil {
        return err
		log.Fatal(err)
	}

    resp, err := c.Do("SRANDMEMBER", g.GetPossiblePlayersSetKey(), g.PlayerCount)
    if err != nil {
        log.Fatal(err)
    }

    for _, player := range(resp.([]interface{})) {
        id := player.([]byte)
        _, err := c.Do("SADD", g.GetChosenPlayersSetKey(), id)
        if err != nil {
            log.Fatal(err)
        }
    }

    return nil
}

func (g ManfredGame) ReplacePlayer(twitchId TwitchUserKey, c redis.Conn) {
	isMember, err := c.Do("SISMEMBER", g.GetChosenPlayersSetKey(), twitchId)
	if err != nil {
		log.Fatal(err)
	}

    // If the handle isn't a chosen player we're done
    if isMember.(int64) == 0 {
        return
    }

    _, err = c.Do("SDIFFSTORE", g.GetUnchosenPlayersSetKey(), g.GetPossiblePlayersSetKey(), g.GetChosenPlayersSetKey())
	if err != nil {
		log.Fatal(err)
	}

    
    newPlayer, err := c.Do("SRANDMEMBER", g.GetUnchosenPlayersSetKey())
	if err != nil {
		log.Fatal(err)
	}

    _, err = c.Do("SADD", g.GetChosenPlayersSetKey(), newPlayer)
    if err != nil {
        log.Fatal(err)
    }

	_, err = c.Do("SREM", g.GetChosenPlayersSetKey(), twitchId)
	if err != nil {
		log.Fatal(err)
	}
}

func (g ManfredGame) GetChosenPlayers(c redis.Conn) (result []TwitchUserKey) {
	resp, err := c.Do("SMEMBERS", g.GetChosenPlayersSetKey())
	if err != nil {
		log.Fatal(err)
	}

	result = make([]TwitchUserKey, 0, g.PlayerCount)

	for _, h := range resp.([]interface{}) {
		id, ok := h.([]byte)
		if !ok {
			log.Fatal("Bad handle from redis for game: " + g.GetChosenPlayersSetKey())
		}
        result = append(result, TwitchUserKey(id))
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
