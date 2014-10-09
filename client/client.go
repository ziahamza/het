package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"

	"../het"

	"github.com/boltdb/bolt"
)

func main() {
	db, err := bolt.Open("../index.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		fmt.Printf("creating db ... \n")
		docs := tx.Bucket([]byte("docs"))

		resultFile, err := os.OpenFile("../spider_result.txt", os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Fatal(err)
			return nil
		}

		docs.ForEach(func(k, v []byte) error {
			doc := het.Document{}
			json.Unmarshal(v, &doc)

			resultFile.WriteString(doc.Title + "\n")
			resultFile.WriteString(string(k) + "\n")
			resultFile.WriteString("Last Modified: " + doc.LastModified + "\n")

			sort.Sort(doc.Keywords)

			for _, kw := range doc.Keywords[0:10] {
				resultFile.WriteString(kw.Word + " " + fmt.Sprintf("%d", kw.Frequency) + ";")
			}
			resultFile.WriteString("\n")

			resultFile.WriteString("-------------------------------------------------------------------------------------------\n")

			return nil
		})

		resultFile.Close()

		return nil
	})

}
