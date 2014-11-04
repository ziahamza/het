package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"io/ioutil"

	"../het"
	"../stemmer"

	"code.google.com/p/go.net/html"
	"github.com/boltdb/bolt"
)

const DocLimit = 30

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
			log.Fatal(errors.New("Count Statistics not found in the db!"))
		}

		countStats := het.CountStats{}
		err := json.Unmarshal(cbytes, &countStats)
		if err != nil {
			log.Fatal(err)
		}

		if countStats.DocumentCount >= 30 {
			fmt.Printf("Document Limit %d reached\n", DocLimit)

			status = 1
			return nil
		}

		ubytes, _ := pending.Cursor().First()
		if ubytes == nil {
			fmt.Printf("no pending doc to index ... \n")

			// status one means finished
			status = 1
			return nil
		}

		uri := string(ubytes[:])

		// delete the url from pending
		pending.Delete(ubytes)

		// original uri already indexed
		if docs.Get(ubytes) != nil {
			fmt.Printf("uri %s already exists ... ignoring\n", uri)
			return nil
		}

		resp, err := http.Get(uri)
		if err != nil {
			// not removing page as internet is not working ...
			fmt.Printf("Error getting back a page (%s) ... waiting 2 sec \n", err.Error())

			// add the page back to pending to try again
			pending.Put(ubytes, []byte(""))
			return nil
		}

		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 299 {
			fmt.Printf("page %s not found \n", uri)
			return nil
		}

		contentType := resp.Header.Get("Content-Type")
		if !(strings.Contains(contentType, "html")) {
			fmt.Printf("non html file (%s) ... ignoring\n", contentType)
			return nil
		}

		parent, _ := url.Parse(resp.Request.URL.String())
		parentUri := parent.String()

		// if the new redirected uri already indexed
		if parentUri != uri && docs.Get([]byte(parentUri)) != nil {
			fmt.Printf("uri %s already exists ... ignoring\n", uri)
			return nil
		}

		buff, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("cannot read all of html body\n")
			return nil
		}

		contentSize := len(buff)

		htmlRoot, err := html.Parse(bytes.NewReader(buff))
		if err != nil {
			fmt.Printf("got back error parsing html ... ignoring\n")
			return nil
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

		wordCount := make(map[string]int)
		for _, token := range strings.Fields(body) {
			word := stemmer.StemWord(token)
			if len(word) > 0 {
				wordCount[word] = wordCount[word] + 1
			}
		}

		doc := het.Document{
			Title:        title,
			Size:         contentSize,
			LastModified: strings.Trim(resp.Header.Get("Last-Modified"), " \t\n"),
			Keywords:     []het.KeywordRef{},
			ChildLinks:   links,
		}

		for word := range wordCount {
			if wordCount[word] == 0 {
				continue
			}

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
			log.Fatal(err)
		}

		stats.Put([]byte("count"), sbytes)

		fmt.Printf("---------------------------------------------\n")
		fmt.Printf("Title         : %s \n", doc.Title)
		fmt.Printf("Url           : %s \n", parentUri)
		fmt.Printf("Size          : %d \n", doc.Size)
		fmt.Printf("Last Modified : %s \n", doc.LastModified)
		fmt.Printf("Children      : %d \n \n", len(links))

		fmt.Printf("Documents Indexed : %d \n", countStats.DocumentCount)
		fmt.Printf("Documents Left    : %d \n", countStats.PendingCount)
		fmt.Printf("Keywords Indexed  : %d \n", countStats.KeywordCount)

		status = 0

		return nil
	})

	if err != nil {
		fmt.Printf("Got back an error indexing page: %s \n", err.Error())
		return 0
	}

	return status
}

func main() {
	db, err := bolt.Open("../index.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	stemmer.LoadStopWords()

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
