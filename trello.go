package main

import (
	"fmt"
	"github.com/VojtechVitek/go-trello"
	"os"
)

const listID = "5d71f9e764e50d46082e17ba"

var (
	api, token string
)

func init() {
	api = os.Getenv("TrelloApiKey")
	token = os.Getenv("TrelloToken")
}

func getFilms() [][]string {
	if api == "" || token == "" {
		return [][]string{}
	}
	client, err := trello.NewAuthClient(api, &token)
	if err != nil {
		return [][]string{}
	}
	list, err := client.List(listID)
	if err != nil {
		return [][]string{}
	}
	cards, err := list.Cards()
	if err != nil {
		return [][]string{}
	}
	result := make([][]string, 0)
	var (
		name, season string
	)
	for _, card := range cards {
		name = card.Name
		season = card.Desc
		if season != "" {
			if len(season) == 1 {
				season = "0" + season
			}
			season = fmt.Sprintf(" [S%[1]s]|[%[1]sx", season)
		}
		result = append(result, parseString(name+season))
	}
	return result
}
