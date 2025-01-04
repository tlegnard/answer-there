package main

//TODO : Add contestant, incorrect response, and triple stumper to clue.
import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

type Clue struct {
	Position          string
	Value             string
	OrderNumber       int
	Text              string
	CorrectResponse   string
	CorrectContestant string
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
	ShowNum     int
	AirDate     string
	TapeDate    string
}

type SeasonData struct {
	ID    string // Season ID, e.g., "40"
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

	// Extract show number and air date from title
	title := doc.Find("title").Text()
	showNumRegex := regexp.MustCompile(`Show #(\d+)`)
	airDateRegex := regexp.MustCompile(`aired (\d{4}-\d{2}-\d{2})`)
	if showNumMatch := showNumRegex.FindStringSubmatch(title); len(showNumMatch) > 1 {
		game.ShowNum, _ = strconv.Atoi(showNumMatch[1])
	}
	if airDateMatch := airDateRegex.FindStringSubmatch(title); len(airDateMatch) > 1 {
		game.AirDate = airDateMatch[1]
	}

	// Extract tape date
	tapeDateRegex := regexp.MustCompile(`Game tape date: (\d{4}-\d{2}-\d{2})`)
	doc.Find("h6").Each(func(_ int, h6Html *goquery.Selection) {
		if tapeDateMatch := tapeDateRegex.FindStringSubmatch(h6Html.Text()); len(tapeDateMatch) > 1 {
			game.TapeDate = tapeDateMatch[1]
		}
	})

	// Find contestant table
	doc.Find("#contestants_table").Each(func(contestantIndex int, contestantTable *goquery.Selection) {
		contestantTable.Find("p.contestants").Each(func(i int, contestantHtml *goquery.Selection) {
			var contestant Contestant
			var htmlText string
			htmlText, _ = contestantHtml.Html()

			contestant.Name = contestantHtml.Find("a").Text()
			contestant.Nickname = strings.Fields(contestant.Name)[0]
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

			// Extract correct contestant's name
			clueHtml.Find("td.clue_text table").Each(func(_ int, subTableHtml *goquery.Selection) {
				clue.CorrectContestant = subTableHtml.Find("td.right").Text()
			})

			round.Clues = append(round.Clues, clue)
		})

		game.Rounds = append(game.Rounds, round)
	})

	return game
}
func GetSeasonList(seasonListHTML string) []string {
	var seasons []string

	doc := parseDoc(seasonListHTML)

	doc.Find("a[href*='showseason.php?season=']").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			// Extract the season ID from the URL
			seasonID, err := extractId(href, "season")
			if err == nil && seasonID != "" {
				// Check if the season ID is numeric (regular season)
				if _, err := strconv.Atoi(seasonID); err == nil {
					seasons = append(seasons, seasonID)
				} else {
					// Handle special seasons (like 'superjeopardy', 'trebekpilots', etc.)
					// Only include if they contain archived games
					text := s.Parent().Parent().Find("td.left_padded").Last().Text()
					if strings.Contains(text, "games archived") {
						seasons = append(seasons, seasonID)
					}
				}
			}
		}
	})
	return seasons
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

func writeCluesToCSV(filePath string, season SeasonData) {
	file, err := os.Create(filePath)
	if err != nil {
		log.Fatalf("Failed to create CSV file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{"SeasonID", "GameId", "RoundName", "Category", "Position", "Value", "OrderNumber", "Text", "CorrectResponse"}
	if err := writer.Write(headers); err != nil {
		log.Fatalf("Failed to write headers to CSV file: %v", err)
	}

	for _, game := range season.Games {
		for _, round := range game.Rounds {
			numCategories := len(round.Categories)
			if numCategories == 0 {
				log.Println("No categories found for round:", round.Name)
				continue
			}

			for clueIndex, clue := range round.Clues {
				categoryIndex := clueIndex % numCategories // Assign clue to the correct category
				category := round.Categories[categoryIndex]

				record := []string{
					season.ID,
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
					log.Fatalf("Failed to write record to CSV file: %v", err)
				}
			}
		}
	}

	log.Println("Successfully wrote clues to CSV")
}

func saveHTMLToFile(directory, filename, content string) error {
	if err := os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", directory, err)
	}
	filePath := filepath.Join(directory, filename)
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", filePath, err)
	}
	defer file.Close()
	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("failed to write content to file %s: %v", filePath, err)
	}
	log.Printf("Saved HTML to %s", filePath)
	return nil
}

func loadHTMLFromFile(directory, filename string) (string, error) {
	filePath := filepath.Join(directory, filename)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %v", filePath, err)
	}
	return string(content), nil
}

func RequestGameDataWithCache(gameID int, seasonID string) string {
	seasonDir := fmt.Sprintf("data/season_%s", seasonID)
	filename := fmt.Sprintf("%d_%s_j-archive.html", gameID, seasonID)
	cachedContent, err := loadHTMLFromFile(seasonDir, filename)
	if err == nil {
		log.Printf("Loaded cached game data for game %d from %s", gameID, filename)
		return cachedContent
	}

	log.Printf("Fetching game data for game %d from J-Archive", gameID)
	gameData := RequestGameData(gameID) // Replace with the actual function to fetch game data from the web
	saveErr := saveHTMLToFile(seasonDir, filename, gameData)
	if saveErr != nil {
		log.Printf("Error saving game data: %v", saveErr)
	}
	return gameData
}

func main() {

	seasonListHTML := RequestSeasonList("https://www.j-archive.com/listseasons.php")
	seasonsList := GetSeasonList(seasonListHTML)

	fmt.Printf("Found %d seasons\n", len(seasonsList))
	for _, season := range seasonsList {
		fmt.Printf("Season: %s\n", season)
	}
	seasonID := "40"
	seasonHTML := RequestSeason("https://j-archive.com/showseason.php?season=" + seasonID) // Replace with actual function
	seasonGameList := GetSeasonGameList(seasonHTML)                                        // Extracts list of game IDs for the season

	var seasonData SeasonData
	seasonData.ID = seasonID
	fmt.Println("Processing Data for Season", seasonID)

	for _, gameID := range seasonGameList {
		gameData := RequestGameDataWithCache(gameID, seasonID)
		game := parseGameTableData(gameData)
		game.ID = gameID
		seasonData.Games = append(seasonData.Games, game)
	}

	dbName := "jeopardy.db"
	writeGameList(dbName, seasonData)
	writeClues(dbName, seasonData)
	writeContestants(dbName, seasonData)
	writeCategories(dbName, seasonData)

	fmt.Println("Number of games processed:", len(seasonData.Games))
}

//TODO
// write something to generate the order in which the game was played, and the money earned (I might just be able to write a query for this)
//finalize the tables and data models. add incexes, PKs foreign keys, etc./
//cleanup the code, can probaly write one handler and pas in schema to write the tables
// go-ify the data scraping, run multiple games at once to start, then eventually multiple seasons
// run for all seasons

//plug into superset/visualization
//sentiment analysis on categories
