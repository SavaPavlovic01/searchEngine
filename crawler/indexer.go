package crawler

import (
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/reiver/go-porterstemmer"
)

var stopwords = map[string]bool{
	"the": true, "and": true, "is": true, "in": true, "to": true, "a": true,
}

type Posting struct {
	tf int
}

type ReverseIndex map[string]map[string]*Posting

func clean(s string) string {
	var builder strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) {
			builder.WriteRune(unicode.ToLower(r))
		}
	}
	return builder.String()
}

func Index(entry IndexEntry) ReverseIndex {
	var reverseIndex = ReverseIndex{}
	cleanText := clean(entry.Text)
	words := strings.Fields(cleanText)
	for _, word := range words {
		if stopwords[word] {
			continue
		}
		stem := porterstemmer.StemString(word)
		if _, ok := reverseIndex[stem]; !ok {
			reverseIndex[stem] = map[string]*Posting{}
		}
		if _, ok := reverseIndex[stem][entry.Url]; !ok {
			reverseIndex[stem][entry.Url] = &Posting{}
		}
		reverseIndex[stem][entry.Url].tf += 1
	}
	fmt.Println("done with indexing")
	fmt.Println(reverseIndex)
	return reverseIndex
}

func mergeIndex(dest ReverseIndex, src ReverseIndex) {
	for word, docMap := range src {
		if _, ok := dest[word]; !ok {
			dest[word] = map[string]*Posting{}
		}

		for doc, posting := range docMap {
			if _, ok := dest[word][doc]; !ok {
				dest[word][doc] = &Posting{}
			}
			p := dest[word][doc]
			p.tf += posting.tf
			dest[word][doc] = p
		}
	}
}

// TODO: should prob panic if i sleep to much
func IndexMain() {
	n := 5
	q := NewRedisIndexQueue("localhost:6379", "", 0, 2)
	sleepTime := 1
	for {
		data, err := q.GetEntries(n)
		if len(data) == 0 || err != nil {
			fmt.Println("index queue empty or error, sleeping")
			time.Sleep(time.Duration(sleepTime * 2))
			sleepTime++
			continue
		}
		sleepTime = 1
		var wg sync.WaitGroup
		partialReverseIndecies := make(chan map[string]map[string]*Posting, len(data))
		wg.Add(len(data))
		for _, enrty := range data {
			curEntry := enrty
			go func() {
				defer wg.Done()
				cur := Index(curEntry)
				partialReverseIndecies <- cur
			}()
		}
		go func() {
			wg.Wait()
			close(partialReverseIndecies)
		}()
		globalIndex := ReverseIndex{}
		for cur := range partialReverseIndecies {
			mergeIndex(globalIndex, cur)
		}
		fmt.Println(globalIndex)
		fmt.Println("done with everything")
	}
}
