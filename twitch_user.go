package main

import (
	"encoding/json"
	_ "github.com/codegangsta/envy/autoload"
	"github.com/garyburd/redigo/redis"
	"io/ioutil"
	"log"
	"net/http"
)

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
	err := json.Unmarshal(body, &user)
	if err != nil {
		return user, err
	}

	return user, nil
}

type TwitchUser struct {
	DisplayName string `json:"display_name"`
	Logo        string `json:"logo"`
	Id          int64  `json:"_id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Email       string `json:"email"`
}

type ManfredPlayer struct {
	Handles map[string]string `json:"handles"`
}

func SaveManfredPlayer(player ManfredPlayer, playerKey TwitchUserKey, c redis.Conn) {
	j, err := json.Marshal(player)
	if err != nil {
		log.Fatal(err)
	}

	c.Do("SET", playerKey, j)
}

func LoadManfredPlayer(key TwitchUserKey, c redis.Conn) (player *ManfredPlayer) {
	value, err := c.Do("GET", key)
	if err != nil {
		log.Fatal(err)
	}

	if value != nil {
		err = json.Unmarshal(value.([]byte), &player)
		if err != nil {
			log.Fatal(err)
		}
	}

	return player

}

type TwitchUserKey string

func (t TwitchUserKey) String() string {
	return "twitchUser:" + string(t)
}
