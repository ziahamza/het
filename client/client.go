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

		resultFile, err := os.Create("../spider_result.txt")
		if err != nil {
			log.Fatal(err)
			return nil
		}

		docs.ForEach(func(k, v []byte) error {
			doc := het.Document{}
			json.Unmarshal(v, &doc)

			fmt.Fprintf(resultFile, "%s\n", doc.Title)
			fmt.Fprintf(resultFile, "%s\n", k)
			fmt.Fprintf(resultFile, "Size: %d", doc.Size)

			if len(doc.LastModified) > 0 {
				resultFile.WriteString(" - " + doc.LastModified)
			}
			resultFile.WriteString("\n")

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
