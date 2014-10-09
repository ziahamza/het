package main

import (
	"fmt"
	"log"

	"github.com/reiver/go-porterstemmer"

	"io/ioutil"
	"strings"
    "bufio"
    "os"
)

var stopWords map[string]bool

func refineQuery(query string) ([]string) {
    keywords := strings.Split(query, " ");
    for i := range keywords {
        if stopWords[keywords[i]] {
            keywords[i] = ""
        } else {
            stem := porterstemmer.StemString(keywords[i])
            keywords[i] = stem;
        }
    }
    return keywords;
}

func setupStopWords() {
    stopWords = map[string]bool{}
    fileData, err := ioutil.ReadFile("stopwords.txt")
    if err != nil {
        log.Fatal(err)
    }
    
    content := string(fileData[:])
    
    words := strings.Split(content, "\n")
    
    for _, word := range words {
        stopWords[word] = true
    }
}

func testRefineQuery() {
    setupStopWords();

	for true {
		fmt.Printf("Please enter query: ")
        stdio := bufio.NewReader(os.Stdin)
        word, err := stdio.ReadString('\n')
        if err!= nil {
            fmt.Printf("%s", err)
        }

		if len(word) == 0 {
			break
		}

        _refineQuery := refineQuery(word)
        fmt.Printf("Your query: \n")
        for i := range _refineQuery {
            if _refineQuery[i] != "" {
                fmt.Printf("%s\n", _refineQuery[i])
            }
        }

	}
}
