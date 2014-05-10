package manfred

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

func LoadManfredGame(key GameKey, c redis.Conn) ManfredGame {
	var game ManfredGame

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
