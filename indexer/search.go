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
	Doc  het.Document
	Rank float64
	URL  string
	Link het.Link
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
		links := tx.Bucket([]byte("links"))
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
			tfIdf   map[string]float64
			rank    float64
			partial bool
			doc     het.Document
			link    het.Link
		})

		keyword := het.Keyword{}
		for word, freq := range words {
			kbytes := keywords.Get([]byte(word))
			if kbytes != nil {

				json.Unmarshal(kbytes, &keyword)

				fmt.Printf("Found index for word: '%s' with docs: %d \n", word, len(keyword.Docs))

				doc := het.Document{}
				link := het.Link{}
				for _, ref := range keyword.Docs {
					rank, found := docRanks[ref.URL.String()]

					if !found {
						rank.tfIdf = map[string]float64{}
						rank.partial = false

						dbytes := docs.Get([]byte(ref.URL.String()))
						lbytes := links.Get([]byte(ref.URL.String()))
						if dbytes == nil {
							return errors.New("Document not found in the main index, but in keyword index!")
						}

						if lbytes == nil {
							return errors.New("Document link missing ?!!! ... maybe run the index again.")
						}

						json.Unmarshal(dbytes, &doc)

						json.Unmarshal(lbytes, &link)

						if doc.Length <= 0 {
							fmt.Printf("got back doc with 0 length: %v \n", doc)
							continue
						}

						rank.doc = doc
						rank.link = link

					}

					/* tf-idf */
					rank.tfIdf[word] = float64(freq*ref.Frequency) * math.Log(float64(countStats.DocumentCount)/float64(len(keyword.Docs)))
					if ref.Frequency == 0 {
						rank.partial = true
					}
					rank.rank += rank.tfIdf[word]

					docRanks[ref.URL.String()] = rank
				}
			} else {
				fmt.Printf("Cannot find index for keyword: %s\n", word)
			}
		}

		for _, rank := range docRanks {
			if rank.partial || rank.rank == 0 {
				// even if one keyword no found, return early
				continue
			}

			results = append(results, SearchResult{
				Doc:  rank.doc,
				Link: rank.link,
				URL:  rank.link.URL.String(),
				Rank: rank.rank / (rank.doc.Length * length),
			})
		}

		sort.Sort(results)

		fmt.Printf("returning %d results for query\n", len(results))

		if len(results) > 50 {
			results = results[:50]
		}

		return nil
	})

	return results, err
}
