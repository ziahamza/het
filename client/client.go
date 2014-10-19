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
	db, err := bolt.Open("./index.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		fmt.Printf("creating db ... \n")
		docs := tx.Bucket([]byte("docs"))

		resultFile, err := os.Create("./spider_result.txt")
		if err != nil {
			log.Fatal(err)
			return nil
		}

		firstLine := true

		docs.ForEach(func(k, v []byte) error {
			if firstLine == false {
				resultFile.WriteString("-------------------------------------------------------------------------------------------\n")
			} else {
				firstLine = false
			}
			doc := het.Document{}
			json.Unmarshal(v, &doc)

			fmt.Fprintf(resultFile, "%s\n", doc.Title)
			fmt.Fprintf(resultFile, "%s\n", k)
			if len(doc.LastModified) > 0 {
				fmt.Fprintf(resultFile, "%s, %d", doc.LastModified, doc.Size)
			} else {
				fmt.Fprintf(resultFile, "No Last Modifited Date, %d", doc.Size)
			}

			resultFile.WriteString("\n")

			sort.Sort(doc.Keywords)

			for _, kw := range doc.Keywords {
				resultFile.WriteString(kw.Word + " " + fmt.Sprintf("%d", kw.Frequency) + ";")
			}
			resultFile.WriteString("\n")

			firstChild := true
			for _, child := range doc.ChildLinks {
				if firstChild == false {
					resultFile.WriteString("\n" + child)
				} else {
					resultFile.WriteString(child)
					firstChild = false
				}

			}

			return nil
		})

		resultFile.Close()

		return nil
	})

}
