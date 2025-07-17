package main

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func main() {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // No password set
		DB:       0,  // Use default DB
		Protocol: 2,  // Connection protocol
	})
	ctx := context.Background()
	err := client.Set(ctx, "someKey", "some Value", 0).Err()
	if err != nil {
		panic(err)
	}
	val, err := client.Get(ctx, "someKey").Result()
	if err != nil {
		panic(err)
	}
	fmt.Println(val)
	//crawler.ProcessPage("https://www.wikipedia.org/")
}
