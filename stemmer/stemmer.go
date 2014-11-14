package stemmer

import (
	"log"

	"github.com/reiver/go-porterstemmer"

	"io/ioutil"
	"strings"
)

var stopWords map[string]bool

func StemWord(word string) string {
	stem := porterstemmer.StemString(word)
	if stopWords[stem] || len(stem) == 0 {
		return ""
	}

	return stem
}
func RefineQuery(query string) []string {
	keywords := strings.Split(query, " ")
	for i := 0; i < len(keywords); i++ {
		keywords[i] = StemWord(keywords[i])

		if keywords[i] == "" {
			keywords = append(keywords[:i], keywords[i+1:]...)
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

	words := strings.Split(content, "\n")

	for _, word := range words {
		stopWords[word] = true
	}
}
