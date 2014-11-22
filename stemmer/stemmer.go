package stemmer

import (
	"log"

	"github.com/reiver/go-porterstemmer"

	"io/ioutil"
	"strings"
)

var stopWords map[string]bool

func StemWord(word string) string {
	if stopWords[word] {
		return ""
	}

	stem := porterstemmer.StemString(word)
	if stopWords[stem] || len(stem) == 0 {
		return ""
	}

	return stem
}
func RefineQuery(query string) []string {
	tokens := strings.Split(query, " ")
	keywords := []string{}
	for i := 0; i < len(tokens); i++ {
		word := StemWord(strings.Trim(tokens[i], "\r\n"))
		if word != "" {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

func LoadStopWords(path string) {
	stopWords = map[string]bool{}
	fileData, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	content := string(fileData[:])

	words := strings.Split(content, "\r\n")

	for _, word := range words {
		stopWords[word] = true
	}
}
