package main

import (
	"encoding/json"
	"fmt"
	"log"

	"search/het"
	"search/indexer"
	"search/stemmer"

	"github.com/boltdb/bolt"
)

const DocLimit = 1000

func main() {
	db, err := bolt.Open("./index.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	stemmer.LoadStopWords("./stopwords.txt")

	err = db.Update(func(tx *bolt.Tx) error {
		fmt.Printf("creating db ... \n")
		docs, err := tx.CreateBucketIfNotExists([]byte("docs"))
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists([]byte("keywords"))
		if err != nil {
			return err
		}

		stats, err := tx.CreateBucketIfNotExists([]byte("stats"))
		if err != nil {
			return err
		}

		sbytes := stats.Get([]byte("count"))
		if sbytes == nil {
			stat := het.CountStats{DocumentCount: 0, KeywordCount: 0}
			sbytes, err = json.Marshal(&stat)
			if err != nil {
				return err
			}

			stats.Put([]byte("count"), sbytes)
		}

		pending, err := tx.CreateBucketIfNotExists([]byte("pending"))
		if err != nil {
			return err
		}

		dbytes, _ := docs.Cursor().First()

		if dbytes == nil {
			pending.Put([]byte("http://www.cse.ust.hk"), []byte(""))
		}

		fmt.Printf("Created db successfully!\n")

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Starting to index pending docs ... \n")

	for true {
		stats, err := indexer.CrawlPage(db)
		if err != nil {
			fmt.Printf("got an error indexing page: %s\n", err.Error())
			break
		}

		if stats.DocumentCount >= DocLimit {
			fmt.Printf("Document Limit %d reached\n", DocLimit)
			break
		}
	}

	fmt.Println("finishing off indexing ... ")
}
