package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"../het"

	"code.google.com/p/go.net/html"
	"github.com/boltdb/bolt"
)

func indexPages(db *bolt.DB) int {
	status := 0
	err := db.Update(func(tx *bolt.Tx) error {
		fmt.Println("Indexing pages ...")

		pending := tx.Bucket([]byte("pending"))
		docs := tx.Bucket([]byte("docs"))
		keywords := tx.Bucket([]byte("keywords"))
		stats := tx.Bucket([]byte("stats"))

		cbytes := stats.Get([]byte("count"))
		if cbytes == nil {
			return errors.New("Count Statistics not found in the db!")
		}

		countStats := het.CountStats{}
		err := json.Unmarshal(cbytes, &countStats)
		if err != nil {
			return err
		}

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

		body := strings.Join(text, "")

		tokens := strings.Split(body, "\n")

		wordCount := make(map[string]int)
		for _, token := range tokens {
			wordLine := strings.TrimSpace(token)
			wordLine = strings.Replace(wordLine, "\t", " ", -1)
			words := strings.Split(wordLine, " ")
			for _, word := range words {
				wordCount[word] = wordCount[word] + 1
			}

		}

		doc := het.Document{
			Title:        title,
			Size:         len(text),
			ModifiedDate: resp.Header.Get("Last-Modified"),
			Keywords:     []het.KeywordRef{},
		}

		for word := range wordCount {
			doc.Keywords = append(doc.Keywords, het.KeywordRef{
				Word:      word,
				Frequency: wordCount[word],
			})

			keyword := het.Keyword{
				Frequency: 0,

				Docs: []het.DocumentRef{},
			}

			kbytes := keywords.Get([]byte(word))
			if kbytes != nil {
				json.Unmarshal(kbytes, &keyword)
			} else {
				// a new keyword count, update stats
				countStats.KeywordCount = countStats.KeywordCount + 1
			}

			keyword.Frequency = keyword.Frequency + wordCount[word]

			keyword.Docs = append(keyword.Docs, het.DocumentRef{
				URL:       uri,
				Frequency: wordCount[word],
			})

			kbytes, _ = json.Marshal(&keyword)

			keywords.Put([]byte(word), kbytes)
		}

		for _, link := range links {
			countStats.PendingCount = countStats.PendingCount + 1
			pending.Put([]byte(link), []byte(""))
		}

		dbytes, _ := json.Marshal(&doc)

		docs.Put(ubytes, dbytes)
		if parentUri != uri {
			docs.Put([]byte(parentUri), dbytes)
		}

		countStats.DocumentCount = countStats.DocumentCount + 1

		sbytes, err := json.Marshal(&countStats)
		if err != nil {
			return nil
		}

		stats.Put([]byte("count"), sbytes)

		fmt.Printf("---------------------------------------------\n")
		fmt.Printf("Title    : %s \n", title)
		fmt.Printf("Url      : %s \n", parentUri)
		fmt.Printf("Size     : %d \n", len(text))
		fmt.Printf("Children : %d \n \n", len(links))

		fmt.Printf("Documents Indexed : %d \n", countStats.DocumentCount)
		fmt.Printf("Documents Left    : %d \n", countStats.PendingCount)
		fmt.Printf("Keywords Indexed  : %d \n", countStats.KeywordCount)

		status = 0
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	return status
}

func main() {
	db, err := bolt.Open("../index.db", 0600, nil)
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

	for indexPages(db) == 0 {
	}

	fmt.Println("finishing off indexing ... ")
}
