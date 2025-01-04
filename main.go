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

	"encoding/json"
	"sync"
	"time"

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

// ProcessingState tracks progress for resuming interrupted processing
type ProcessingState struct {
	LastCompletedSeason string
	SeasonProgress      map[string][]int // Maps season ID to completed game IDs
	FailedGames         map[string][]int // Maps season ID to failed game IDs
	LastUpdated         time.Time
}

// saveProcessingState saves current progress to a JSON file
func saveProcessingState(state ProcessingState, filename string) error {
	state.LastUpdated = time.Now()
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

// loadProcessingState loads progress from a JSON file
func loadProcessingState(filename string) (ProcessingState, error) {
	var state ProcessingState
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// Initialize new state if file doesn't exist
			return ProcessingState{
				SeasonProgress: make(map[string][]int),
				FailedGames:    make(map[string][]int),
			}, nil
		}
		return state, err
	}
	err = json.Unmarshal(data, &state)
	return state, err
}

// processGame handles a single game with error reporting
func processGame(gameID int, seasonID string, wg *sync.WaitGroup, results chan<- GameData, errors chan<- error) {
	defer wg.Done()

	// Recover from panics
	defer func() {
		if r := recover(); r != nil {
			errors <- fmt.Errorf("panic processing game %d in season %s: %v", gameID, seasonID, r)
		}
	}()

	gameData := RequestGameDataWithCache(gameID, seasonID)
	game := parseGameTableData(gameData)
	game.ID = gameID

	results <- game
}
func readSeasonsFile(filename string) ([]string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Split content by newlines and filter empty lines
	var seasons []string
	for _, season := range strings.Split(string(content), "\n") {
		if trimmed := strings.TrimSpace(season); trimmed != "" {
			seasons = append(seasons, trimmed)
		}
	}
	return seasons, nil
}

func main() {
	const (
		stateFile     = "processing_state.json"
		seasonsFile   = "seasons.txt"
		maxConcurrent = 5
		dbName        = "jeopardy.db"
	)

	// Load or initialize processing state
	state, err := loadProcessingState(stateFile)
	if err != nil {
		log.Fatalf("Failed to load processing state: %v", err)
	}

	// Try to read seasons from file first
	var seasonsList []string
	if seasons, err := readSeasonsFile(seasonsFile); err == nil {
		fmt.Printf("Using seasons from %s\n", seasonsFile)
		seasonsList = seasons
	} else {
		if !os.IsNotExist(err) {
			log.Printf("Error reading %s: %v. Falling back to web scraping.", seasonsFile, err)
		}
		// Fall back to getting all seasons from web
		seasonListHTML := RequestSeasonList("https://www.j-archive.com/listseasons.php")
		seasonsList = GetSeasonList(seasonListHTML)
	}

	fmt.Printf("Found %d seasons to process\n", len(seasonsList))

	// Process each season
	for _, seasonID := range seasonsList {
		// Skip if season was already completed
		if seasonID == state.LastCompletedSeason {
			continue
		}

		fmt.Printf("\nProcessing Season: %s\n", seasonID)
		seasonHTML := RequestSeason("https://j-archive.com/showseason.php?season=" + seasonID)
		seasonGameList := GetSeasonGameList(seasonHTML)
		fmt.Printf("Found %d games in Season %s\n", len(seasonGameList), seasonID)

		var seasonData SeasonData
		seasonData.ID = seasonID

		// Create channels for results and errors
		results := make(chan GameData, len(seasonGameList))
		errors := make(chan error, len(seasonGameList))

		// Process games concurrently with worker pool
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, maxConcurrent)

		for _, gameID := range seasonGameList {
			// Skip if game was already processed successfully
			found := false
			for _, gameIDInProgress := range state.SeasonProgress[seasonID] {
				if gameIDInProgress == gameID {
					found = true
					break
				}
			}
			if found {
				fmt.Printf("\rSkipping already processed game %d", gameID)
				continue
			}

			wg.Add(1)
			semaphore <- struct{}{} // Acquire semaphore

			go func(gID int) {
				processGame(gID, seasonID, &wg, results, errors)
				<-semaphore // Release semaphore
			}(gameID)
		}

		// Start a goroutine to close channels when all games are processed
		go func() {
			wg.Wait()
			close(results)
			close(errors)
		}()

		// Collect results and errors
		var processedGames []GameData
		var failedGames []int

		// Process results as they come in
		for game := range results {
			processedGames = append(processedGames, game)
			state.SeasonProgress[seasonID] = append(state.SeasonProgress[seasonID], game.ID)

			// Print progress
			fmt.Printf("\rProcessed %d/%d games", len(processedGames), len(seasonGameList))

			// Save state periodically (every 10 games)
			if len(processedGames)%10 == 0 {
				if err := saveProcessingState(state, stateFile); err != nil {
					log.Printf("\nError saving processing state: %v", err)
				}
			}
		}

		//  TODO:Process any errors
		// for err := range errors {
		// 	log.Printf("\nError: %v", err)
		// 	if gameID := extractGameIDFromError(err); gameID != 0 {
		// 		failedGames = append(failedGames, gameID)
		// 	}
		// }

		seasonData.Games = processedGames

		// Write season data to database if we have processed games
		if len(seasonData.Games) > 0 {
			// Wrap database operations in a transaction if possible
			// if err := writeGameList(dbName, seasonData); err != nil {
			// 	log.Printf("\nError writing game list: %v", err)
			// }
			// if err := writeClues(dbName, seasonData); err != nil {
			// 	log.Printf("\nError writing clues: %v", err)
			// }
			// if err := writeContestants(dbName, seasonData); err != nil {
			// 	log.Printf("\nError writing contestants: %v", err)
			// }
			// if err := writeCategories(dbName, seasonData); err != nil {
			// 	log.Printf("\nError writing categories: %v", err)
			// }
			writeGameList(dbName, seasonData)
			writeClues(dbName, seasonData)
			writeContestants(dbName, seasonData)
			writeCategories(dbName, seasonData)
		}

		// Update state
		state.LastCompletedSeason = seasonID
		state.FailedGames[seasonID] = failedGames
		if err := saveProcessingState(state, stateFile); err != nil {
			log.Printf("\nError saving final state for season %s: %v", seasonID, err)
		}

		fmt.Printf("\nSeason %s: Successfully processed %d games, Failed %d games\n",
			seasonID, len(processedGames), len(failedGames))

		// Optional delay between seasons to be nice to the server
		time.Sleep(2 * time.Second)
	}

	fmt.Println("\nFinished processing all seasons")
}

//TODO
// write something to generate the order in which the game was played, and the money earned (I might just be able to write a query for this)
//finalize the tables and data models. add incexes, PKs foreign keys, etc./
//add season list table
//cleanup the code, can probaly write one handler and pas in schema to write the tables

//plug into superset/visualization
//sentiment analysis on categories
//cli to generate categories to quiz myself on
