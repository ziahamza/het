package main

import (
	"fmt"
	"log"
	"net/http"

	"code.google.com/p/go.net/html"
)

func main() {
	url := "http://www.cs.ust.hk/~dlee/4321/"
	resp, err := http.Get(url)

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

	fmt.Printf("Links in %s \n", url)
	for _, link := range links {
		fmt.Println(link)
	}
}
