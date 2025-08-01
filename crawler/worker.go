package crawler

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

func ProcessPage(urll string) (string, []string, []ImageInfo, error) {
	resp, err := http.Get(urll)
	if err != nil {
		return "", nil, nil, err
	}
	defer resp.Body.Close()
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return "", nil, nil, err
	}
	text := parseText(doc)
	base, err := url.Parse(urll)
	if err != nil {
		return "", nil, nil, err
	}
	links, images := parseLinks(doc, base)
	return text, links, images, nil
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

type ImageInfo struct {
	DocumentUrl string `json:"document_url"`
	AltText     string `json:"alt_text"`
	URL         string `json:"image_url"`
	NearbyText  string `json:"nearby_text"`
}

// TODO: CLEAN UP
func parseLinks(doc *html.Node, base *url.URL) ([]string, []ImageInfo) {
	var links []string
	var images []ImageInfo

	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "a":
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
			case "img":
				var src, alt string
				for _, att := range n.Attr {
					switch att.Key {
					case "src":
						src = strings.TrimSpace(att.Val)
					case "alt":
						alt = strings.TrimSpace(att.Val)
					}
				}

				if src != "" {
					u, err := base.Parse(src)
					if err == nil && (u.Scheme == "http" || u.Scheme == "https") {
						var nearbyText string
						if n.Parent != nil {
							nearbyText = parseText(n.Parent)
						}
						images = append(images, ImageInfo{
							DocumentUrl: base.Host,
							URL:         u.String(),
							AltText:     alt,
							NearbyText:  strings.TrimSpace(nearbyText),
						})
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(doc)
	return links, images
}

func CrawlerMain() {
	n := 10
	startCorpus := []string{
		"https://en.wikipedia.org/wiki/Main_Page",
		"https://www.bbc.com",
		"https://www.cnn.com",
		"https://www.nytimes.com",
		"https://www.theguardian.com/international",
		"https://www.reuters.com",
		"https://www.nationalgeographic.com",
		"https://www.scientificamerican.com",
		"https://www.nytimes.com/wirecutter",
		"https://www.aljazeera.com",
	}
	urlQueue := NewRedisQueue("localhost:6379", "", 0, 2)
	indexQueue := NewRedisIndexQueue("localhost:6379", "", 0, 2)
	_, err := urlQueue.EnqueMultiple(startCorpus)
	workerUrlQueues := make([]*RedisQueue, n)
	testRunLimit := 1000
	currentRun := 0
	for i := range n {
		workerUrlQueues[i] = NewRedisQueue("localhost:6379", "", 0, 2)
	}

	if err != nil {
		panic(err)
	}
	for {
		urls, err := urlQueue.GetUrls(n)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		if len(urls) == 0 {
			time.Sleep(time.Duration(2))
			continue
		}
		currentRun += len(urls)
		var wg sync.WaitGroup
		indexQueueBatch := make(chan IndexEntry, len(urls))
		wg.Add(len(urls))
		for i, entry := range urls {
			curUrl := entry
			index := i
			go func() {
				defer wg.Done()
				text, links, images, err := ProcessPage(curUrl)
				if err != nil {
					fmt.Println(err.Error())
				}
				workerUrlQueues[index].EnqueMultiple(links)
				indexQueueBatch <- IndexEntry{Url: curUrl, Text: text, Images: images, Links: links}
			}()
		}

		go func() {
			wg.Wait()
			close(indexQueueBatch)
		}()

		batch := make([]IndexEntry, len(urls))
		j := 0
		for curPage := range indexQueueBatch {
			batch[j] = curPage
			j += 1
		}

		indexQueue.EnqueMultiple(batch)
		if currentRun > testRunLimit {
			fmt.Println("Test run done")
			break
		}
	}
}
