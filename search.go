package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"code.google.com/p/go.net/html"
	"github.com/boltdb/bolt"
)

type Stats struct {
	DocumentCount, KeywordCount int
}

// used by the docs bucket to refer to a specific keyword under a document
type KeywordRef struct {
	Word      string
	Frequency int
}

// used by the keywords bucket to refer to a document containing a specific keyword
type DocumentRef struct {
	URL       string
	Frequency int
}

// stored in docs bucket
type Document struct {
	Title    string
	Size     int
	Keywords []KeywordRef
}

// stored in keywords bucket
type Keyword struct {
	Frequency int
	Docs      []DocumentRef
}

func indexPages(db *bolt.DB) int {
	status := 0
	err := db.Update(func(tx *bolt.Tx) error {
		fmt.Println("Indexing pages ...")

		pending := tx.Bucket([]byte("pending"))
		docs := tx.Bucket([]byte("docs"))
		keywords := tx.Bucket([]byte("keywords"))

		ubytes, _ := pending.Cursor().First()
		if ubytes == nil {
			fmt.Printf("no pending doc to index ... \n")
			status = 1
			return nil
		}

		uri := string(ubytes[:])

		pending.Delete(ubytes)

		// original uri already indexed
		if docs.Get(ubytes) != nil {
			fmt.Printf("uri %s already exists ... ignoring\n", uri)

			status = 0
			return nil
		}

		resp, err := http.Get(uri)
		if err != nil {
			log.Fatal(err)
		}

		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 299 {
			fmt.Printf("page %s not found \n", uri)
			status = 0
			return nil
		}

		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "text/html") {
			fmt.Printf("non html file (%s) ... ignoring\n", contentType)
			status = 0
			return nil
		}

		parent, _ := url.Parse(resp.Request.URL.String())
		parentUri := parent.String()

		// if the new redirected uri already indexed
		if parentUri != uri && docs.Get([]byte(parentUri)) != nil {
			fmt.Printf("uri %s already exists ... ignoring\n", uri)

			status = 0
			return nil
		}

		htmlRoot, err := html.Parse(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		links := []string{}
		text := []string{}

		var title string

		var f func(*html.Node)
		f = func(n *html.Node) {
			if n.Type == html.ElementNode {
				if n.Data == "a" {
					for _, a := range n.Attr {
						if a.Key == "href" {
							child, err := parent.Parse(a.Val)
							if err == nil && (child.Scheme == "http" || child.Scheme == "https") {
								links = append(links, child.String())
							} else {
								fmt.Printf("got back error parsing %s\n", a.Val)
							}

							break
						}
					}
				}

				if n.Data == "title" {
					// get the first text node inside title
					for c := n.FirstChild; c != nil; c = c.NextSibling {
						if c.Type == html.TextNode {
							title = c.Data
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

		f(htmlRoot)

		fmt.Printf("---------------------------------------------\n")
		fmt.Printf("Title    : %s \n", title)
		fmt.Printf("Url      : %s \n", parentUri)
		fmt.Printf("Size     : %d \n", len(text))
		fmt.Printf("Children : %d \n", len(links))

		body := strings.Join(text, "")

		tokens := strings.Split(body, " ")

		wordCount := make(map[string]int)
		for _, token := range tokens {
			word := strings.Trim(token, " ")

			wordCount[word] = wordCount[word] + 1
		}

		doc := Document{
			Title:    title,
			Size:     len(text),
			Keywords: []KeywordRef{},
		}

		for word := range wordCount {
			doc.Keywords = append(doc.Keywords, KeywordRef{
				Word:      word,
				Frequency: wordCount[word],
			})

			keyword := Keyword{
				Frequency: 0,

				Docs: []DocumentRef{},
			}

			kbytes := keywords.Get([]byte(word))
			if kbytes != nil {
				json.Unmarshal(kbytes, &keyword)
			}

			keyword.Frequency = keyword.Frequency + wordCount[word]

			keyword.Docs = append(keyword.Docs, DocumentRef{
				URL:       uri,
				Frequency: wordCount[word],
			})

			kbytes, _ = json.Marshal(&keyword)

			keywords.Put([]byte(word), kbytes)
		}

		for _, link := range links {
			pending.Put([]byte(link), []byte(""))
		}

		dbytes, _ := json.Marshal(&doc)

		docs.Put(ubytes, dbytes)
		if parentUri != uri {
			docs.Put([]byte(parentUri), dbytes)
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

		/*
			stats, err := tx.CreateBucketIfNotExists([]byte("stats"))
			if err != nil {
				return err
			}

			if stats.Cursor().First() == nil {
				stats.Put([]byte(""), []byte(""))
			}
		*/

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
