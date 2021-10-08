package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

func main() {

	if _, err := strconv.Atoi(os.Getenv("MONTH")); err != nil {
		log.Fatalf("Invalid month (\"%s\"): %q", os.Getenv("MONTH"), err)
	}
	month := os.Getenv("MONTH")

	if _, err := strconv.Atoi(os.Getenv("YEAR")); err != nil {
		log.Fatalf("Invalid year (\"%s\"): %q", os.Getenv("YEAR"), err)
	}
	year := os.Getenv("YEAR")

	outputFolder := os.Getenv("OUTPUT_FOLDER")
	if outputFolder == "" {
		outputFolder = "./output"
	}

	if err := os.Mkdir(outputFolder, os.ModePerm); err != nil && !os.IsExist(err) {
		log.Fatalf("Error creating output folder(%s): %q", outputFolder, err)
	}

	court := strings.ToLower(os.Getenv("COURT"))
	if court == "" {
		log.Fatalf("Environment variable COURT is mandatory")
	}

	downloads, err := crawl(court, year, month, outputFolder)
	if err != nil {
		log.Fatalf("Error crawling (%s, %s, %s, %s): %v", court, year, month, outputFolder, err)
	}

	// O parser do CNJ espera os arquivos separados por \n. Mudanças aqui tem
	// refletir as expectativas lá.
	fmt.Println(strings.Join(downloads, "\n"))
}
