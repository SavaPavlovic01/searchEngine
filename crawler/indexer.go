package crawler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/reiver/go-porterstemmer"
)

var stopwords = map[string]bool{
	"the": true, "and": true, "is": true, "in": true, "to": true, "a": true,
}

type Posting struct {
	tf int
}

type ReverseIndex map[string]map[string]*Posting

// TODO: maybe make it like this while you are building the reverse index
// so you dont have to run through the date twice
func (ri ReverseIndex) dump() [][]interface{} {
	var rows [][]interface{}
	for term, doc := range ri {
		for docName, posting := range doc {
			rows = append(rows, []interface{}{term, docName, posting.tf})
		}
	}
	return rows
}

// TODO: quick and dirty, maybe furhter apstract db
func getDBConection() (*pgxpool.Pool, context.Context, error) {
	ctx := context.Background()
	dbPool, err := pgxpool.New(ctx, "postgresql://adimn:admin@localhost:5432/searchEngine")
	if err != nil {
		return nil, nil, err
	}
	return dbPool, ctx, err
}

func writeReverseIndexToDB(db *pgxpool.Pool, ctx context.Context, index ReverseIndex) error {
	_, err := db.CopyFrom(ctx,
		pgx.Identifier{"inverted_index"},
		[]string{"term", "document_url", "tf"},
		pgx.CopyFromRows(index.dump()))
	return err
}

func writeDocsToDB(db *pgxpool.Pool, ctx context.Context, docs []IndexEntry) error {
	_, err := db.CopyFrom(ctx,
		pgx.Identifier{"documents"},
		[]string{"url", "title", "content"},
		pgx.CopyFromSlice(len(docs), func(i int) ([]any, error) {
			return []any{docs[i].Url, "", docs[i].Text}, nil
		}),
	)
	return err
}

func writeLinksToDB(db *pgxpool.Pool, ctx context.Context, docs []IndexEntry) error {
	var links []struct{ from, to string }
	for _, entry := range docs {
		for _, l := range entry.Links {
			links = append(links, struct {
				from string
				to   string
			}{from: entry.Url, to: l})
		}
	}
	_, err := db.CopyFrom(ctx,
		pgx.Identifier{"links"},
		[]string{"from", "to"},
		pgx.CopyFromSlice(len(links), func(i int) ([]any, error) {
			return []any{links[i].from, links[i].to}, nil
		}))
	return err
}

func writeImagesToDB(db *pgxpool.Pool, ctx context.Context, docs []IndexEntry) error {
	var images []ImageInfo
	for _, doc := range docs {
		images = append(images, doc.Images...)
	}
	_, err := db.CopyFrom(ctx,
		pgx.Identifier{"images"},
		[]string{"image_url", "document_url", "alt_text", "nearby_text"},
		pgx.CopyFromSlice(len(docs), func(i int) ([]any, error) {
			return []any{images[i].URL, images[i].DocumentUrl, images[i].AltText, images[i].NearbyText}, nil
		}))
	return err
}

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

func connectToNeo() (neo4j.DriverWithContext, neo4j.SessionWithContext, context.Context, error) {
	ctx := context.Background()
	uri := "neo4j://localhost:7687"
	auth := neo4j.BasicAuth("neo4j", "testtest123", "")
	driver, err := neo4j.NewDriverWithContext(uri, auth)
	if err != nil {
		return nil, nil, nil, err
	}
	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	return driver, session, ctx, nil
}

func writeLinksToNeo(session neo4j.SessionWithContext, ctx context.Context, data []IndexEntry) error {

	var rels []map[string]any
	for _, entry := range data {
		for _, link := range entry.Links {
			rels = append(rels, map[string]any{"Source": entry.Url, "Target": link})
		}
	}
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `UNWIND $batch AS row
            MERGE (src:Link {url: row.Source})
            MERGE (dst:Link {url: row.Target})
            MERGE (src)-[:LINKS_TO]->(dst)`
		params := map[string]any{"batch": rels}
		_, err := tx.Run(ctx, query, params)
		return nil, err
	})
	return err
}

// TODO: should prob panic if i sleep to much
// clean this up
func IndexMain() {
	n := 50
	q := NewRedisIndexQueue("localhost:6379", "", 0, 2)
	db, ctx, err := getDBConection()
	if err != nil {
		panic(err)
	}
	neoDrive, neo, neoCtx, err := connectToNeo()
	if err != nil {
		panic(err)
	}
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
		var dbWait sync.WaitGroup
		//dbWait.Add(1)

		go func() {
			err = writeDocsToDB(db, ctx, data)
			if err != nil {
				fmt.Println(err.Error())
			}
			fmt.Println("Done writing batch to db")
			dbWait.Done()
		}()

		dbWait.Add(1)

		go func() {
			imageErr := writeImagesToDB(db, ctx, data)
			if imageErr != nil {
				fmt.Println(err.Error())
			}
			dbWait.Done()
			fmt.Println("Done writing images to db")
		}()

		// NEO4J WRITE IS REALLY SLOW!!!!!!!!!
		dbWait.Add(1)
		go func() {
			neoErr := writeLinksToDB(db, ctx, data)
			if neoErr != nil {
				fmt.Println("NEO FAILED")
				fmt.Print(neoErr.Error())
			}
			dbWait.Done()
			fmt.Println("Done writing batch to neo")
		}()

		var wg sync.WaitGroup
		partialReverseIndecies := make(chan map[string]map[string]*Posting, len(data))
		wg.Add(len(data))
		for _, enrty := range data {
			curEntry := enrty
			go func() {
				defer wg.Done()
				cur := Index(curEntry)
				partialReverseIndecies <- cur
				fmt.Println("DOne indexing batch for this thread")
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

		dbWait.Wait()
		err = writeReverseIndexToDB(db, ctx, globalIndex)
		fmt.Println("Done writing reverse index batch to db")
		dbWait.Done()
	}
	neoDrive.Close(neoCtx)
	neo.Close(neoCtx)
	db.Close()
}
