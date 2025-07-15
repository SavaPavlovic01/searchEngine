package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

func processPage(urll string) {
	resp, err := http.Get(urll)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return
	}
	fmt.Print(parseText(doc))
	base, err := url.Parse(urll)
	if err != nil {
		return
	}
	links, images := parseLinks(doc, base)
	for _, link := range links {
		fmt.Println(link)
	}

	for _, image := range images {
		fmt.Println(image)
	}
}

func parseText(doc *html.Node) string {
	if doc.Type == html.ElementNode {
		switch doc.Data {
		case "script", "style", "noscript", "code", "pre":
			return ""
		}
	}
	if doc.Type == html.TextNode {
		data := strings.TrimSpace(doc.Data)
		if data != "" {
			return data + " "
		}
	}
	var result string
	for c := doc.FirstChild; c != nil; c = c.NextSibling {
		result += parseText(c)
	}
	return result
}

// TODO: CLEAN UP
func parseLinks(n *html.Node, base *url.URL) ([]string, []string) {
	var links []string
	var images []string
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, att := range n.Attr {
				if att.Key == "href" {
					href := strings.TrimSpace(att.Val)
					if href == "" || href == "#" {
						continue
					}
					u, err := base.Parse(href)
					if err == nil && (u.Scheme == "http" || u.Scheme == "https") {
						links = append(links, u.String())
					}
				}
			}
		}

		if n.Type == html.ElementNode && n.Data == "img" {
			for _, att := range n.Attr {
				if att.Key == "src" {
					href := strings.TrimSpace(att.Val)
					if href == "" {
						continue
					}
					u, err := base.Parse(href)
					if err == nil && (u.Scheme == "http" || u.Scheme == "https") {
						images = append(images, u.String())
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(n)
	return links, images
}

func main() {
	processPage("https://www.wikipedia.org/")
}
