package main

import (
	"encoding/json"
	"log"
	"os"

	notion_blog "github.com/rxrw/notion-wp/pkg"

	"github.com/rxrw/notion-wp/internal"

	"github.com/joho/godotenv"
)

var config notion_blog.BlogConfig

func parseJSONConfig() {
	content, err := os.ReadFile("notionblog.config.json")
	if err != nil {
		log.Fatal("error reading config file: ", err)
	}
	json.Unmarshal(content, &config)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file provided")
	}

	parseJSONConfig()

	err = internal.ParseAndGenerate(config)
	if err != nil {
		log.Fatal(err)
	}

}
