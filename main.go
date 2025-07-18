package main

import (
	"fmt"
	"searchEngine/crawler"
)

func main() {

	queue := crawler.NewRedisQueue("localhost:6379", "", 0, 2)
	val, err := queue.Enque("someText")
	if err != nil {
		panic(err)
	}
	if !val {
		fmt.Println(val)
	}
	val, err = queue.EnqueMultiple([]string{"someText", "someText1", "someText2", "someText3"})
	if err != nil {
		panic(err)
	}
	fmt.Println(val)
	res, err := queue.GetUrls(4)
	if err != nil {
		panic(err)
	}
	for _, s := range res {
		fmt.Println(s)
	}
	fmt.Println("done")
}
