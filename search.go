package main

import (
	"fmt"
	"log"
	"net/http"

	"os"

	"code.google.com/p/go.net/html"
	"github.com/boltdb/bolt"
)

var db bolt.DB

func indexPages() {
	db.Update(func(tx *bolt.Tx) error {
		pending := tx.Bucket([]byte("pending"))
		docs := tx.Bucket([]byte("docs"))

		url, _ := pending.Cursor().First()
		if url == nil {
			return nil
		}

		pending.Delete(url)

		// doc already indexed ... returning
		if docs.Get(url) != nil {
			fmt.Printf("url %s already exists ... ignoring\n", string(url[:]))
			return nil
		}

		resp, err := http.Get(string(url[:]))

		if err != nil {
			log.Fatal(err)
		}

		defer resp.Body.Close()

		doc, err := html.Parse(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(doc)

		links := []string{}
		text := []string{}

		var f func(*html.Node)
		f = func(n *html.Node) {
			if n.Type == html.ElementNode {
				if n.Data == "a" {
					for _, a := range n.Attr {
						if a.Key == "href" {
							pending.Put([]byte(a.Val), []byte(""))
							links = append(links, a.Val)
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
		fmt.Println("got back text strings: %v", text)

		for _, link := range links {
			fmt.Println(link)
		}

		return nil
	})
}

func main() {
	db, err := bolt.Open("./index.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	db.Update(func(tx *bolt.Tx) error {
		docs, err := tx.CreateBucketIfNotExists([]byte("docs"))
		if err != nil {
			log.Fatal(err)
		}

		keywords, err := tx.CreateBucketIfNotExists([]byte("keywords"))
		if err != nil {
			log.Fatal(err)
		}

		pending, err := tx.CreateBucketIfNotExists([]byte("pending"))
		if err != nil {
			log.Fatal(err)
		}

		doc, _ := docs.Cursor().First()

		if doc == nil {
			pending.Put([]byte("http://www.cse.ust.hk"), []byte(""))
		}

		url, _ := pending.Cursor().First()

		if url == nil {
			fmt.Printf("Entire web space searched!! hurray :D")
			os.Exit(0)
		}

		return nil
	})

	rootUrl := ""
	resp, err := http.Get(rootUrl)

	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(doc)

	links := []string{}
	text := []string{}

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "a" {
				for _, a := range n.Attr {
					if a.Key == "href" {
						links = append(links, a.Val)
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

	fmt.Println("got back text strings: %v", text)

	fmt.Printf("Links in %s \n", rootUrl)
	for _, link := range links {
		fmt.Println(link)
	}
}
