package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"net/http"
	"net/url"
	"os"

	"search/het"
	"search/indexer"
	"search/stemmer"

	"github.com/boltdb/bolt"
)

func main() {

	dbpath := flag.String("db", "./index.db", "Path to the local db used for indexing and searching.")
	stopwordspath := flag.String("stopwords", "./stopwords.txt", "Path to the stop words used to filter out common words")
	indexLimit := flag.Int("index", 300, "Pages to Index first before starting the local server")
	listen := flag.String("port", ":8080", "Port and host for api server to listen on")
	drop := flag.Bool("drop", false, "Set to reset DB and then fill the DB with the pages specified by index")

	flag.Parse()

	if *drop == true {
		err := os.Remove(*dbpath)
		if err != nil {
			fmt.Printf("Failed to delete the old db ... creating a new one anyways\n")
		}
	}

	db, err := bolt.Open(*dbpath, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	stemmer.LoadStopWords(*stopwordspath)

	err = db.Update(func(tx *bolt.Tx) error {
		fmt.Printf("creating db ... \n")
		docs, err := tx.CreateBucketIfNotExists([]byte("docs"))
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists([]byte("doc-keywords"))
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists([]byte("links"))
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

		pending, err := tx.CreateBucketIfNotExists([]byte("pending"))
		if err != nil {
			return err
		}

		dbytes, _ := docs.Cursor().First()

		stat := het.CountStats{DocumentCount: 0, LinkCount: 0, PendingCount: 0, KeywordCount: 0}
		if dbytes == nil {
			pending.Put([]byte("http://www.cse.ust.hk/~ericzhao/COMP4321/TestPages/testpage.htm"), []byte(""))
			stat.PendingCount++
		}

		sbytes := stats.Get([]byte("count"))
		if sbytes == nil {
			sbytes, err = json.Marshal(&stat)
			if err != nil {
				return err
			}

			stats.Put([]byte("count"), sbytes)
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

		if stats.DocumentCount >= *indexLimit {
			fmt.Printf("Finished indexing enough pages %d \n", stats.DocumentCount)
			break
		}
	}

	fmt.Println("finishing off indexing ... ")

	fmt.Printf("Server listening on http://localhost%s \n", *listen)

	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("./static/"))))

	errorHandler := func(w http.ResponseWriter, message string) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(500)

		bd, err := json.Marshal(map[string]interface{}{
			"success": false,
			"message": message,
		})
		if err != nil {
			if err.Error() == "Unmatched column names/values" {
				// internal db error, exit for debugging ...
				panic(err.Error())
			}
			bd = []byte(`{ "success": false, "message": "` + err.Error() + `" }`)
		}
		w.Write(bd)
		fmt.Println("error:", message)
	}

	dataHandler := func(wr http.ResponseWriter, data map[string]interface{}) {
		wr.Header().Set("Content-Type", "application/json")
		wr.Header().Set("Access-Control-Allow-Origin", "*")
		bd, err := json.Marshal(data)

		if err != nil {
			errorHandler(wr, err.Error())
			return
		}

		wr.WriteHeader(200)
		wr.Write([]byte(bd))
	}

	http.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		values, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			errorHandler(w, err.Error())
			return
		}

		query := values.Get("query")

		if query == "" {
			errorHandler(w, "query not given")
			return
		}

		results, err := indexer.Search(db, query)
		if err != nil {
			errorHandler(w, err.Error())
			return
		}

		dataHandler(w, map[string]interface{}{
			"success": true,
			"results": results,
		})
	})

	err = http.ListenAndServe(*listen, nil)
	if err != nil {
		log.Fatal(err)
	}
}
