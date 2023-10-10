package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"net/http"
	"github.com/PuerkitoBio/goquery"
 )

func requestGameData(gameId int) string {
	resp, err := http.Get(fmt.Sprintf("https://j-archive.com/showgame.php?game_id=%d", gameId))

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

func parseGameTableData(gameData string) {
	var row []string
	var rows [][]string

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(gameData))
	if err != nil {
		fmt.Println("no url found")
		log.Fatal(err)
	}

	//Find each table: https://gist.github.com/salmoni/27aee5bb0d26536391aabe7f13a72494
	doc.Find("table").Each(func(index int, tablehtml *goquery.Selection) {
		tablehtml.Find("tr").Each(func(indextr int, rowhtml *goquery.Selection) {
			rowhtml.Find("td").Each(func(indextd int, tablecell *goquery.Selection) {
				row = append(row, tablecell.Text())
			})
			rows = append(rows, row)
			row = nil
		})
	})
	fmt.Println(" rows = ", len(rows), rows)
}

func main() {
	var gameId int = 7074
	gameData := requestGameData(gameId)
	parseGameTableData(gameData)
}