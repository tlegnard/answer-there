package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"time"
)

func RequestGameData(gameId int) string {
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

func RequestSeason(url string) string {
	resp, err := http.Get(url)

	if err != nil {
		log.Fatalln(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	seasonData := string(body)
	return seasonData
}

func RequestSeasonList(url string) string {
	cacheDir := "data/metadata"
	filename := "season_list.html"

	cachedContent, err := loadHTMLFromFile(cacheDir, filename)
	if err == nil {
		log.Printf("Loaded cached season list from %s", filename)
		return cachedContent
	}

	log.Printf("Fetching season list from J-Archive")
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	time.Sleep(time.Duration(1+rand.Intn(2)) * time.Second)

	resp, err := client.Get(url)
	if err != nil {
		log.Fatalf("Failed to fetch season list: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Failed to fetch season list. Status: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}

	content := string(bodyBytes)

	// Save to cache
	saveErr := saveHTMLToFile(cacheDir, filename, content)
	if saveErr != nil {
		log.Printf("Error saving season list: %v", saveErr)
	}

	return content
}
