package main

import (
	"fmt"
	"log"

	"github.com/reiver/go-porterstemmer"

	"io/ioutil"
	"strings"
)

func main1() {
	stopWords := map[string]bool{}
	fileData, err := ioutil.ReadFile("stopwords.txt")
	if err != nil {
		log.Fatal(err)
	}

	content := string(fileData[:])

	words := strings.Split(content, "\n")

	for _, word := range words {
		stopWords[word] = true
	}

	word := ""
	for true {
		fmt.Printf("Please enter a single english word: ")
		fmt.Scanf("%s", &word)

		if len(word) == 0 {
			break
		}

		if stopWords[word] {
			fmt.Printf("It should be stopped\n")
		} else {
			stem := porterstemmer.StemString(word)

			fmt.Printf("The stem of %s is %s \n", word, stem)
		}
	}
}
