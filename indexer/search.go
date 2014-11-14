package indexer

import (
	"encoding/json"
	"errors"
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
		docs := tx.Bucket([]byte("docs"))

		// indexes to sort out data
		docRanks := make(map[string]struct {
			keywords map[string]int
			doc      het.Document
		})

		keyword := het.Keyword{}
		for _, word := range words {
			kbytes := keywords.Get([]byte(word))
			if kbytes != nil {

				json.Unmarshal(kbytes, &keyword)

				fmt.Printf("Found index for word: '%s' with docs: %d \n", word, len(keyword.Docs))

				for _, ref := range keyword.Docs {
					rank, found := docRanks[ref.URL]

					if !found {
						rank.keywords = map[string]int{}
						dbytes := docs.Get([]byte(ref.URL))
						if dbytes == nil {
							return errors.New("Document not found in the main index, but in keyword index!")
						}

						json.Unmarshal(dbytes, &rank.doc)
					}

					rank.keywords[word] = ref.Frequency

					docRanks[ref.URL] = rank
				}
			} else {
				fmt.Printf("Cannot find index for keyword: %s\n", word)
			}
		}

		for url, rank := range docRanks {
			results = append(results, SearchResult{
				Title: rank.doc.Title,
				URL:   url,
				Size:  rank.doc.Size,
			})
		}

		fmt.Printf("returning %d results for query\n", len(results))

		return nil
	})

	return results, err
}
