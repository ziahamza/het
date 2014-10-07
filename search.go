package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"

	"code.google.com/p/go.net/html"
	"github.com/boltdb/bolt"
)

func indexPages(db *bolt.DB) int {
	status := 0
	err := db.Update(func(tx *bolt.Tx) error {
		fmt.Println("Indexing pages ...")

		pending := tx.Bucket([]byte("pending"))
		docs := tx.Bucket([]byte("docs"))

		uri, _ := pending.Cursor().First()
		if uri == nil {
			fmt.Printf("no pending doc to index ... \n")
			status = 1
			return nil
		}

		pending.Delete(uri)

		// doc already indexed ... returning
		if docs.Get(uri) != nil {
			fmt.Printf("uri %s already exists ... ignoring\n", string(uri[:]))

			status = 0
			return nil
		}

		resp, err := http.Get(string(uri[:]))

		if err != nil {
			log.Fatal(err)
		}

		defer resp.Body.Close()

		doc, err := html.Parse(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		links := []string{}
		text := []string{}

		parent, _ := url.Parse(string(uri[:]))

		var f func(*html.Node)
		f = func(n *html.Node) {
			if n.Type == html.ElementNode {
				if n.Data == "a" {
					for _, a := range n.Attr {
						if a.Key == "href" {

							uri, err := parent.Parse(a.Val)
							if err == nil && (uri.Scheme == "http" || uri.Scheme == "https") {
								links = append(links, uri.String())
							} else {
								fmt.Printf("got back error parsing %s\n", a.Val)
							}

							break
						}
					}
				}

				// ignore scripts, styles
				if n.Data == "script" || n.Data == "style" {
					return
				}
			}
			if n.Type == html.TextNode {
				text = append(text, n.Data)
				return
			}

			for c := n.FirstChild; c != nil; c = c.NextSibling {
				f(c)
			}
		}

		f(doc)

		fmt.Printf("---------------------------------------------\n")
		fmt.Printf("got back url %s with size: %d \n", string(uri[:]), len(text))

		for _, link := range links {
			fmt.Printf("putting in children: %s \n", link)
			pending.Put([]byte(link), []byte(""))
		}

		status = 0
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	return status
}

func main() {
	db, err := bolt.Open("./index.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

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

		pending, err := tx.CreateBucketIfNotExists([]byte("pending"))
		if err != nil {
			return err
		}

		doc, _ := docs.Cursor().First()

		if doc == nil {
			pending.Put([]byte("http://www.cse.ust.hk"), []byte(""))
		}

		fmt.Printf("Created db successfully!\n")

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Starting to index pending docs ... \n")

	for indexPages(db) == 0 {
	}

	fmt.Println("finishing off indexing ... ")
}
