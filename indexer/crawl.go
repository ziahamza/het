package indexer

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

	"search/het"

	"code.google.com/p/go.net/html"
	"github.com/boltdb/bolt"
)

func CrawlPage(db *bolt.DB) (het.CountStats, error) {
	countStats := het.CountStats{}
	err := db.Update(func(tx *bolt.Tx) error {
		fmt.Println("Indexing pages ...")

		pending := tx.Bucket([]byte("pending"))
		docs := tx.Bucket([]byte("docs"))
		docKeywords := tx.Bucket([]byte("doc-keywords"))
		docLinks := tx.Bucket([]byte("doc-links"))

		keywords := tx.Bucket([]byte("keywords"))
		stats := tx.Bucket([]byte("stats"))

		cbytes := stats.Get([]byte("count"))
		if cbytes == nil {
			log.Fatal(errors.New("Count Statistics not found in the db!"))
		}

		err := json.Unmarshal(cbytes, &countStats)
		if err != nil {
			log.Fatal(err)
		}

		ubytes, _ := pending.Cursor().First()
		if ubytes == nil {
			fmt.Printf("no pending doc to index ... \n")

			pending.Put([]byte("http://en.wikipedia.org/wiki/List_of_most_popular_websites"), []byte(""))

			// status one means finished, we saturated the internet
			return errors.New("Somehow saturated the entire internet ?!!")
		}

		uri, _ := url.Parse(string(ubytes[:]))
		uri.Fragment = ""

		// delete the url from pending
		pending.Delete(ubytes)

		// original uri already indexed
		if docs.Get([]byte(uri.String())) != nil {
			// fmt.Printf("uri %s already exists ... ignoring\n", uri.String())
			return nil
		}

		resp, err := http.Get(uri.String())
		if err != nil {
			// not removing page as internet is not working ...
			fmt.Printf("Error getting back a page (%s) ... waiting 2 sec \n", err.Error())

			// add the page back to pending to try again
			pending.Put([]byte(uri.String()), []byte(""))
			return nil
		}

		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 299 {
			fmt.Printf("page %s not found \n", uri.String)
			return nil
		}

		contentType := resp.Header.Get("Content-Type")
		if !(strings.Contains(contentType, "html")) {
			fmt.Printf("non html file (%s) ... ignoring\n", contentType)
			return nil
		}

		parent, _ := url.Parse(resp.Request.URL.String())

		// Optimisation: remove any hash fragments from the url
		parent.Fragment = ""

		parentUri := parent.String()

		// if the new redirected uri already indexed
		if parentUri != uri.String() && docs.Get([]byte(parentUri)) != nil {
			fmt.Printf("uri %s already exists ... ignoring\n", uri.String())
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
								child.Fragment = ""

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

		wordCount, length := GetVector(body)

		dockeys := []het.KeywordRef{}
		doc := het.Document{
			Title:        title,
			Size:         contentSize,
			Length:       length,
			LastModified: strings.Trim(resp.Header.Get("Last-Modified"), " \t\n"),
		}

		for word := range wordCount {
			if wordCount[word] == 0 {
				continue
			}

			dockeys = append(dockeys, het.KeywordRef{
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
				URL:       uri.String(),
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
		kbytes, _ := json.Marshal(&dockeys)
		lbytes, _ := json.Marshal(&links)

		docs.Put([]byte(uri.String()), dbytes)
		docKeywords.Put([]byte(uri.String()), kbytes)
		docLinks.Put([]byte(uri.String()), lbytes)

		if parentUri != uri.String() {
			// it was a redirect .. put the parent uri anyways so we never download it
			docs.Put([]byte(parentUri), dbytes)
			docKeywords.Put([]byte(parentUri), kbytes)
			docLinks.Put([]byte(parentUri), lbytes)
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

		return nil
	})

	return countStats, err
}
