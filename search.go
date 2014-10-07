package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"code.google.com/p/go.net/html"
	"github.com/boltdb/bolt"
)

// used by the docs bucket to refer to a specific keyword under a document
type KeywordRef struct {
	word      string
	frequency int
}

// used by the keywords bucket to refer to a document containing a specific keyword
type DocumentRef struct {
	URL       string
	frequency int
}

// stored in docs bucket
type Doc struct {
	title    string
	size     int
	keywords []KeywordRef
}

// stored in keywords bucket
type Keyword struct {
	frequency int
	docs      []DocumentRef
}

func indexPages(db *bolt.DB) int {
	status := 0
	err := db.Update(func(tx *bolt.Tx) error {
		fmt.Println("Indexing pages ...")

		pending := tx.Bucket([]byte("pending"))
		docs := tx.Bucket([]byte("docs"))
		keywords := tx.Bucket([]byte("keywords"))

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

		if resp.StatusCode < 200 || resp.StatusCode >= 299 {
			fmt.Printf("page %s not found \n", string(uri[:]))
			status = 0
			return nil
		}

		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "text/html") {
			fmt.Printf("non html file (%s) ... ignoring\n", contentType)
			status = 0
			return nil
		}

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

		words := make(map[string]int)

		parent, _ := url.Parse(resp.Request.URL.String())

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
		fmt.Printf("got back url %s with size: %d \n", parent.String(), len(text))

		body := strings.Join(text, "")

		tokens := strings.Split(body, " ")
		for _, token := range tokens {
			word := strings.Trim(token, " ")

			words[word] = words[word] + 1
		}

		for word := range words {
			documents := keywords.Get([]byte(word))
		}

		for _, link := range links {
			fmt.Printf("putting in children: %s \n", link)
			pending.Put([]byte(link), []byte(""))
		}

		docs.Put([]byte(parent.String()), []byte(body))
		docs.Put([]byte(uri), []byte(body))

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
