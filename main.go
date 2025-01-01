package main

//TODO : Add contestant, incorrect response, and triple stumper to clue.
import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
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
	// GameID          int
	// RoundName       int
}

// Round struct represents a round of the game
type Round struct {
	Name       string
	Categories []string
	Clues      []Clue
	// GameID     int
}

type Contestant struct {
	PlayerID string
	Name     string
	Nickname string
	Bio      string
}

// GameData struct represents the game data including multiple rounds
type GameData struct {
	ID          int
	Rounds      []Round
	Contestants []Contestant
}

type SeasonData struct {
	Games []GameData
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

func extractId(textHTML string, match_string string) (string, error) {
	re := regexp.MustCompile("(" + match_string + "=)(\\d+)")
	matches := re.FindStringSubmatch(textHTML)
	if len(matches) > 0 {
		return matches[2], nil
	}

	return "", nil

}

func parseDoc(Data string) *goquery.Document {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(Data))
	if err != nil {
		fmt.Println("no url found")
		log.Fatal(err)
	}
	return doc
}

func parseGameTableData(gameData string) GameData {
	var game GameData

	doc := parseDoc(gameData)

	// Find contestant table
	doc.Find("#contestants_table").Each(func(contestantIndex int, contestantTable *goquery.Selection) {
		contestantTable.Find("p.contestants").Each(func(i int, contestantHtml *goquery.Selection) {
			var contestant Contestant
			var htmlText string
			htmlText, _ = contestantHtml.Html()

			contestant.Name = contestantHtml.Find("a").Text()
			contestant.PlayerID, _ = extractId(htmlText, "player_id")

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

		if roundHtml.HasClass("final_round") {
			round.Name = "Final Jeopardy"
		} else if roundIndex == 0 {
			round.Name = "Jeopardy! Round"
		} else if roundIndex == 1 {
			round.Name = "Double Jeopardy! Round"
		}
		roundHtml.Find("td.category").Each(func(index int, categoryHtml *goquery.Selection) {
			categoryName := categoryHtml.Find("td.category_name").Text()
			round.Categories = append(round.Categories, categoryName)
		})

		// Parse Clues for the round
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

func GetSeasonGameList(seasonData string) []int {
	var seasonList []int

	doc := parseDoc(seasonData)

	doc.Find("a[href*='showgame.php?game_id=']").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			gameIDtext, _ := extractId(href, "game_id")
			gameID, err := strconv.Atoi(gameIDtext)
			if err != nil {
				panic(err)
			}
			seasonList = append(seasonList, gameID)
		}
	})

	return seasonList
}

func writeCluesToCSV(filePath string, seasonID string, game GameData) {
	file, err := os.Create(filePath)
	if err != nil {
		log.Fatalf("Failed to creat CSV file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{"SeasonID", "GameId", "RoundName", "Category", "Position", "Value", "OrderNumber", "Text", "CorrectResponse"}
	if err := writer.Write(headers); err != nil {
		log.Fatalf("Failed to write headers to CSV file: %v", err)
	}

	for _, round := range game.Rounds {
		for _, category := range round.Categories {
			for _, clue := range round.Clues {
				record := []string{
					seasonID,
					strconv.Itoa(game.ID),
					round.Name,
					category,
					clue.Position,
					clue.Value,
					strconv.Itoa(clue.OrderNumber),
					clue.Text,
					clue.CorrectResponse,
				}
				if err := writer.Write(record); err != nil {
					log.Fatalf("FaILED TO WRITE RECORD TO CSV FILE: %v", err)
				}
			}
		}
	}
}

func main() {
	seasonID := "40"
	seasonHTML := RequestSeason("https://j-archive.com/showseason.php?season=" + seasonID)
	seasonGameList := GetSeasonGameList(seasonHTML)
	var seasonData SeasonData

	fmt.Println("Processing Data for the following games in Season "+seasonID+": ", seasonGameList)

	for _, gameID := range seasonGameList {
		gameData := RequestGameData(gameID)
		var game GameData = parseGameTableData(gameData)
		game.ID = gameID
		seasonData.Games = append(seasonData.Games, game)
		filePath := fmt.Sprintf("games/game_%d_clues.csv", gameID)
		// TODO: fix data modleing issue putting duplicate clues for each category in each round
		writeCluesToCSV(filePath, seasonID, game)
	}

	// sampleGame := seasonData.Games[len(seasonData.Games)-1]

	fmt.Println("# of games to parse: ", len(seasonGameList))
	fmt.Println("# of games parsed: ", len(seasonData.Games))
	// fmt.Println("Game ID: ", sampleGame.ID)
	// fmt.Println("Contestants:", sampleGame.Contestants)
	// // Print the Categories and Clues for each round
	// for _, round := range sampleGame.Rounds {
	// 	fmt.Println(round.Name)
	// 	fmt.Println("Categories:", round.Categories)
	// 	fmt.Println("Clues:")
	// 	for _, clue := range round.Clues {
	// 		fmt.Println("---------------------------")
	// 		fmt.Printf("BoardPosition: %s\n Value: %s\n Order Number %d\n Text: %s\n Correct Response: %s\n",
	// 			clue.Position, clue.Value, clue.OrderNumber, clue.Text, clue.CorrectResponse)
	// 		fmt.Println("---------------------------")
	// 	}
	// 	fmt.Println("---------------------------")
	// }

}
