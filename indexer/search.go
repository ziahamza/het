package indexer

import (
	"encoding/json"
	"fmt"
	"strings"

	"search/het"
	"search/stemmer"

	"github.com/boltdb/bolt"
)

type SearchResult struct {
	Title, URL string
	Size       int
}

func Search(db *bolt.DB, query string) ([]SearchResult, error) {
	words := []string{}
	results := []SearchResult{}

	for _, token := range strings.Fields(query) {
		word := stemmer.StemWord(token)
		if len(word) > 0 {
			words = append(words, word)
		}
	}

	err := db.View(func(tx *bolt.Tx) error {
		keywords := tx.Bucket([]byte("keywords"))

		docs := make(map[string]struct {
			keywords map[string]int
		})

		keyword := het.Keyword{}
		for _, word := range words {
			kbytes := keywords.Get([]byte(word))
			if kbytes != nil {

				json.Unmarshal(kbytes, &keyword)

				fmt.Printf("Found index for word: '%s' with docs: %d \n", word, len(keyword.Docs))

				for _, ref := range keyword.Docs {
					doc, found := docs[ref.URL]

					if !found {
						doc.keywords = map[string]int{}
					}

					doc.keywords[word] = ref.Frequency

					docs[ref.URL] = doc
				}
			} else {
				fmt.Printf("Cannot find index for keyword: %s\n", word)
			}
		}

		for url, _ := range docs {
			results = append(results, SearchResult{
				Title: "",
				URL:   url,
				Size:  0,
			})
		}

		fmt.Printf("returning %d results for query\n", len(results))

		return nil
	})

	return results, err
}
