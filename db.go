package main

import (
	"database/sql"
	"log"
	"math/rand"
	"time"
)

func writeGameList(dbName string, season SeasonData) {
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		log.Fatalf("Failed to open SQLite database: %v", err)
	}
	defer db.Close()

	// Create the `gamelist` table if it doesn't exist
	createGameListTableSQL := `
		ATTACH DATABASE ? AS game_data;
		CREATE TABLE IF NOT EXISTS game_data.gamelist (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			season_id TEXT NOT NULL,
			game_id INTEGER NOT NULL UNIQUE,
			show_num INTEGER NOT NULL UNIQUE,
			air_date DATE NOT NULL,
			tape_date DATE NOT NULL
		);
	`
	if _, err := db.Exec(createGameListTableSQL, dbName); err != nil {
		log.Fatalf("Failed to create gamelist table: %v", err)
	}

	// Insert games into the `gamelist` table
	insertGameSQL := `
		INSERT OR IGNORE INTO game_data.gamelist (
			season_id, game_id, show_num, air_date, tape_date
		) VALUES (?, ?, ?, ?, ?);
	`

	for _, game := range season.Games {
		_, err := db.Exec(
			insertGameSQL,
			season.ID,
			game.ID,
			game.ShowNum,
			game.AirDate,
			game.TapeDate,
		)
		if err != nil {
			log.Fatalf("Failed to insert game into gamelist table: %v", err)
		}
	}

	log.Println("Successfully wrote gamelist data to SQLite database")
}

func writeClues(dbName string, season SeasonData) {
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		log.Fatalf("Failed to open SQLite database: %v", err)
	}
	defer db.Close()

	// Create the schema and table if it doesn't exist
	createSchemaSQL := `
		ATTACH DATABASE ? AS game_data;
		CREATE TABLE IF NOT EXISTS game_data.clues (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			season_id TEXT NOT NULL,
			game_id INTEGER NOT NULL,
			round_name TEXT NOT NULL,
			category TEXT NOT NULL,
			position TEXT,
			value TEXT,
			order_number INTEGER,
			text TEXT NOT NULL,
			correct_response TEXT,
			correct_contestant TEXT
		);
	`
	if _, err := db.Exec(createSchemaSQL, dbName); err != nil {
		log.Fatalf("Failed to create schema or table: %v", err)
	}

	// Insert clues into the table
	insertClueSQL := `
		INSERT INTO game_data.clues (
			season_id, game_id, round_name, category, position, value, order_number, text, correct_response, correct_contestant
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`

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

				_, err := db.Exec(
					insertClueSQL,
					season.ID,
					game.ID,
					round.Name,
					category,
					clue.Position,
					clue.Value,
					clue.OrderNumber,
					clue.Text,
					clue.CorrectResponse,
					clue.CorrectContestant,
				)
				if err != nil {
					log.Fatalf("Failed to insert clue into database: %v", err)
				}
			}
		}
	}

	log.Println("Successfully wrote clues to SQLite database")
}

func writeContestants(dbName string, season SeasonData) {
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		log.Fatalf("Failed to open SQLite database: %v", err)
	}
	defer db.Close()

	// Create the `contestants` table if it doesn't exist
	createGameRosterTableSQL := `
		ATTACH DATABASE ? AS game_data;
		CREATE TABLE IF NOT EXISTS game_data.game_roster (
			player_id TEXT NOT NULL,
			season_id TEXT NOT NULL,
			game_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			nickname TEXT,
			bio TEXT
		);
	`

	if _, err := db.Exec(createGameRosterTableSQL, dbName); err != nil {
		log.Fatalf("Failed to create game_roster table: %v", err)
	}

	// Insert contestants into the `contestants` table
	insertGameRosterSQL := `
		INSERT OR IGNORE INTO game_data.game_roster (
			player_id, season_id, game_id, name, nickname, bio
		) VALUES (?, ?, ?, ?, ?, ?);
	`

	for _, game := range season.Games {
		for _, contestant := range game.Contestants {
			_, err := db.Exec(
				insertGameRosterSQL,
				contestant.PlayerID,
				season.ID,
				game.ID,
				contestant.Name,
				contestant.Nickname,
				contestant.Bio,
			)
			if err != nil {
				log.Fatalf("Failed to insert contestant into contestants table: %v", err)
			}
		}
	}

	// Create or replace the `contestants` view
	createContestantViewSQL := `
		DROP VIEW IF EXISTS contestants;
		CREATE VIEW contestants AS
		SELECT DISTINCT player_id, name FROM game_roster;
	`
	if _, err := db.Exec(createContestantViewSQL); err != nil {
		log.Fatalf("Failed to create contestants view: %v", err)
	}

}

func generateRandomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func writeCategories(dbName string, season SeasonData) {
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		log.Fatalf("Failed to open SQLite database: %v", err)
	}
	defer db.Close()

	// Create the `categories` table if it doesn't exist
	createCategoriesTableSQL := `
		ATTACH DATABASE ? AS game_data;
		CREATE TABLE IF NOT EXISTS game_data.categories (
			category_id TEXT PRIMARY KEY,
			season_id TEXT NOT NULL,
			game_id INTEGER NOT NULL,
			round_name TEXT NOT NULL,
			category_name TEXT NOT NULL
		);
	`
	if _, err := db.Exec(createCategoriesTableSQL, dbName); err != nil {
		log.Fatalf("Failed to create categories table: %v", err)
	}

	// Insert categories into the `categories` table
	insertCategorySQL := `
		INSERT OR IGNORE INTO game_data.categories (
			category_id, season_id, game_id, round_name, category_name
		) VALUES (?, ?, ?, ?, ?);
	`

	for _, game := range season.Games {
		for _, round := range game.Rounds {
			for _, categoryName := range round.Categories {
				// Generate a unique random string for the categoryID
				categoryID := generateRandomString(8)

				_, err := db.Exec(
					insertCategorySQL,
					categoryID,
					season.ID,
					game.ID,
					round.Name,
					categoryName,
				)
				if err != nil {
					log.Fatalf("Failed to insert category into categories table: %v", err)
				}
			}
		}
	}

	log.Println("Successfully wrote categories data to SQLite database")
}
