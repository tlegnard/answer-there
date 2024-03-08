package main

import (
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
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
