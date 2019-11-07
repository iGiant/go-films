package main

import (
	"github.com/VojtechVitek/go-trello"
	"os"
)

const listID = "5d71f9e38bebb648f1de6e30"

var (
	api, token string
)

func init() {
	api = os.Getenv("TrelloApiKey")
	token = os.Getenv("TrelloToken")
}

func getFilmsFromTrello() [][]string {
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
	var name string
	for _, card := range cards {
		name = card.Name
		result = append(result, parseString(name))
	}
	return result
}
