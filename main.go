package main

import (
	"searchEngine/crawler"
)

func main() {
	//url := "https://en.wikipedia.org/wiki/Distributed_web_crawling"
	q := crawler.NewRedisIndexQueue("localhost:6379", "", 0, 2)
	//text, links, images, err := crawler.ProcessPage(url)
	//if err != nil {
	//	panic(err)
	//}
	err := q.Enque(crawler.IndexEntry{Url: "test", Text: "some ranom text", Links: []string{}, Images: []string{}})

	if err != nil {
		panic(err)
	}
	err = q.Enque(crawler.IndexEntry{Url: "test1", Text: "some ranom sava", Links: []string{}, Images: []string{}})
	if err != nil {
		panic(err)
	}
	crawler.IndexMain()
}
