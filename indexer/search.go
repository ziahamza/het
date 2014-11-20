package indexer

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"

	"log"
	"search/het"

	"github.com/boltdb/bolt"
)

type SearchResult struct {
	Title, URL string
	Rank       float64
	Size       int
}

type SearchResults []SearchResult

func (a SearchResults) Len() int           { return len(a) }
func (a SearchResults) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SearchResults) Less(i, j int) bool { return a[i].Rank >= a[j].Rank }

func Search(db *bolt.DB, query string) (SearchResults, error) {
	results := SearchResults{}

	words, length := GetVector(query)

	fmt.Printf("got back query %v with length: %f \n", words, length)

	err := db.View(func(tx *bolt.Tx) error {
		keywords := tx.Bucket([]byte("keywords"))
		docs := tx.Bucket([]byte("docs"))
		stats := tx.Bucket([]byte("stats"))

		countStats := het.CountStats{}
		cbytes := stats.Get([]byte("count"))
		if cbytes == nil {
			log.Fatal(errors.New("Count Statistics not found in the db!"))
		}

		err := json.Unmarshal(cbytes, &countStats)
		if err != nil {
			log.Fatal(err)
		}

		// indexes to sort out data
		docRanks := make(map[string]struct {
			tfIdf map[string]float64
			rank  float64
			doc   het.Document
		})

		keyword := het.Keyword{}
		for word, freq := range words {
			kbytes := keywords.Get([]byte(word))
			if kbytes != nil {

				json.Unmarshal(kbytes, &keyword)

				fmt.Printf("Found index for word: '%s' with docs: %d \n", word, len(keyword.Docs))

				doc := het.Document{}
				for _, ref := range keyword.Docs {
					rank, found := docRanks[ref.URL]

					if !found {
						rank.tfIdf = map[string]float64{}
						dbytes := docs.Get([]byte(ref.URL))
						if dbytes == nil {
							return errors.New("Document not found in the main index, but in keyword index!")
						}

						json.Unmarshal(dbytes, &doc)

						if doc.Length <= 0 {
							fmt.Printf("got back doc with 0 length: %v \n", doc)
							continue
						}

						rank.doc = doc
					}

					/* tf-idf */
					rank.tfIdf[word] = float64(freq*ref.Frequency) * math.Log(float64(countStats.DocumentCount)/float64(len(keyword.Docs)))
					rank.rank += rank.tfIdf[word]

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
				Rank:  rank.rank / (rank.doc.Length * length),
				Size:  rank.doc.Size,
			})
		}

		sort.Sort(results)

		if len(results) > 10 {
			results = results[:10]
		}

		fmt.Printf("returning %d results for query\n", len(results))

		return nil
	})

	return results, err
}
