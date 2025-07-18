package main

import (
	"fmt"
	"searchEngine/crawler"
)

func main() {
	url := "https://www.wikipedia.org/"
	q := crawler.NewRedisIndexQueue("localhost:6379", "", 0, 2)
	text, links, images, err := crawler.ProcessPage(url)
	if err != nil {
		panic(err)
	}
	err = q.Enque(crawler.IndexEntry{Url: url, Text: text, Links: links, Images: images})
	if err != nil {
		panic(err)
	}
	data, err := q.GetEntries(1)
	if err != nil {
		panic(err)
	}

	fmt.Println(data[0].Text)
}
