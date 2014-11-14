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
		docs := tx.Bucket([]byte("docs"))
		docKeywords := tx.Bucket([]byte("doc-keywords"))
		docLinks := tx.Bucket([]byte("doc-links"))

		resultFile, err := os.Create("../spider_result.txt")
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
			dockeys := het.DocKeywords{}
			doclinks := het.DocLinks{}

			json.Unmarshal(v, &doc)

			kbytes := docKeywords.Get(k)
			lbytes := docLinks.Get(k)

			json.Unmarshal(kbytes, &dockeys)
			json.Unmarshal(lbytes, &doclinks)

			fmt.Fprintf(resultFile, "%s\n", doc.Title)
			fmt.Fprintf(resultFile, "%s\n", k)
			if len(doc.LastModified) > 0 {
				fmt.Fprintf(resultFile, "%s, %d", doc.LastModified, doc.Size)
			} else {
				fmt.Fprintf(resultFile, "No Last Modifited Date, %d", doc.Size)
			}

			resultFile.WriteString("\n")

			sort.Sort(dockeys)

			for _, kw := range dockeys {
				resultFile.WriteString(kw.Word + " " + fmt.Sprintf("%d", kw.Frequency) + ";")
			}
			resultFile.WriteString("\n")

			firstChild := true
			for _, child := range doclinks {
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
