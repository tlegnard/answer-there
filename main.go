package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Round struct represents a round of the game
type Round struct {
	Categories []string
	Clues      []string
}

// GameData struct represents the game data including multiple rounds
type GameData struct {
	Rounds []Round
}

func requestGameData(gameId int) string {
	url := fmt.Sprintf("https://j-archive.com/showgame.php?game_id=%d", gameId)
	resp, err := http.Get(url)

	if err != nil {
		log.Fatalln(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	gameData := string(body)

	return gameData
}

func parseGameTableData(gameData string) GameData {
	var game GameData

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(gameData))
	if err != nil {
		fmt.Println("no url found")
		log.Fatal(err)
	}

	// Find each round table
	doc.Find("table.round").Each(func(roundIndex int, roundHtml *goquery.Selection) {
		var round Round

		// Parse Categories for the round
		roundHtml.Find("td.category").Each(func(index int, categoryHtml *goquery.Selection) {
			categoryName := categoryHtml.Find("td.category_name").Text()
			round.Categories = append(round.Categories, categoryName)
		})

		// Parse Clues for the round (adjust this part based on the actual HTML structure)
		roundHtml.Find("td.clue").Each(func(index int, clueHtml *goquery.Selection) {
			clueText := clueHtml.Text()
			round.Clues = append(round.Clues, clueText)
		})

		// Append the current round to the game
		game.Rounds = append(game.Rounds, round)
	})

	return game
}

func main() {
	var gameId int = 7074
	gameData := requestGameData(gameId)
	game := parseGameTableData(gameData)

	// Print the Categories and Clues for each round
	for roundIndex, round := range game.Rounds {
		fmt.Printf("Round %d:\n", roundIndex+1)
		fmt.Println("Categories:", round.Categories)
		fmt.Println("Clues:", round.Clues)
		fmt.Println("---------------------------")
	}
}
