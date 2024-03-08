package main

//update Round to have Round name: J/Jeopardy! Round DJ/Double Jeopardy! Round
//Add contestant and correct answer or triple stumper to clue.
import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Clue struct {
	Position        string
	Value           string
	OrderNumber     int
	Text            string
	CorrectResponse string
}

// Round struct represents a round of the game
type Round struct {
	Categories []string
	Clues      []Clue
}

// GameData struct represents the game data including multiple rounds
type GameData struct {
	Rounds      []Round
	Contestants []Contestant
}

type Contestant struct {
	PlayerID string
	Name     string
	Nickname string
	Bio      string
}

func extractCluePosition(clueHTMLText string) (string, error) {
	// Define a regular expression pattern for the ID
	re := regexp.MustCompile(`(clue_)((J|DJ)_(\d+_\d+))`)

	// Find the first match in the HTML text
	matches := re.FindStringSubmatch(clueHTMLText)
	if len(matches) > 0 {
		return matches[2], nil
	}

	return "", nil
}

func extractPlayerId(contestantHTML string) (string, error) {
	re := regexp.MustCompile(`(player_id=)(\d+)`)
	matches := re.FindStringSubmatch(contestantHTML)
	if len(matches) > 0 {
		return matches[2], nil
	}

	return "", nil

}

func parseGameTableData(gameData string) GameData {
	var game GameData

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(gameData))
	if err != nil {
		fmt.Println("no url found")
		log.Fatal(err)
	}

	// Find contestant table
	doc.Find("#contestants_table").Each(func(contestantIndex int, contestantTable *goquery.Selection) {
		contestantTable.Find("p.contestants").Each(func(i int, contestantHtml *goquery.Selection) {
			var contestant Contestant
			var htmlText string
			htmlText, _ = contestantHtml.Html()

			contestant.Name = contestantHtml.Find("a").Text()
			contestant.PlayerID, _ = extractPlayerId(htmlText)

			// Filter out text matching contestant.Name
			contestantHtml.Contents().Each(func(j int, content *goquery.Selection) {

				text := content.Text()
				if !strings.Contains(text, contestant.Name) {
					// Append non-matching text to player bio
					contestant.Bio += strings.TrimPrefix(text, ", ")
				}
			})
			game.Contestants = append(game.Contestants, contestant)
		})

	})

	// Find each round table
	doc.Find("table").FilterFunction(func(_ int, tableHtml *goquery.Selection) bool {
		return tableHtml.HasClass("round") || tableHtml.HasClass("final_round")
	}).Each(func(roundIndex int, roundHtml *goquery.Selection) {
		var round Round

		// Parse Categories for the round
		roundHtml.Find("td.category").Each(func(index int, categoryHtml *goquery.Selection) {
			categoryName := categoryHtml.Find("td.category_name").Text()
			round.Categories = append(round.Categories, categoryName)
		})

		// Parse Clues for the round (adjust this part based on the actual HTML structure)
		roundHtml.Find("td.clue").Each(func(index int, clueHtml *goquery.Selection) {
			var clue Clue
			clueHTMLText, _ := clueHtml.Html()
			position, err := extractCluePosition(clueHTMLText)
			if err != nil {
				log.Println("Error extracting clue information:", err)
				return
			}

			clue.Position = position
			clue.Value = clueHtml.Find("td.clue_value").Text()
			clue.OrderNumber, _ = strconv.Atoi(clueHtml.Find("td.clue_order_number").Text())

			clue.Text = clueHtml.Find("td.clue_text").First().Text()

			clue.CorrectResponse = clueHtml.Find("td.clue_text em.correct_response").Text()

			round.Clues = append(round.Clues, clue)
		})

		game.Rounds = append(game.Rounds, round)
	})

	return game
}

func main() {
	// seasonData := RequestSeason("https://j-archive.com/showseason.php?season=40")
	var gameId int = 7074
	gameData := RequestGameData(gameId)
	game := parseGameTableData(gameData)
	fmt.Println("Contestants:", game.Contestants)

	// Print the Categories and Clues for each round
	for roundIndex, round := range game.Rounds {
		fmt.Printf("Round %d:\n", roundIndex+1)
		fmt.Println("Categories:", round.Categories)
		fmt.Println("Clues:")
		for _, clue := range round.Clues {
			fmt.Println("---------------------------")
			fmt.Printf("BoardPosition: %s\n Value: %s\n Order Number %d\n Text: %s\n Correct Response: %s\n",
				clue.Position, clue.Value, clue.OrderNumber, clue.Text, clue.CorrectResponse)
			fmt.Println("---------------------------")
		}
		fmt.Println("---------------------------")
	}

	// fmt.Println(seasonData)
}
