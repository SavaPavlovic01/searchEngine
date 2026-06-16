# SearchEngine

A full search engine built from scratch in Go and Python — covering web crawling, inverted index construction, TF-IDF ranking, and a REST search API. Built as a learning project to understand how search engines work at each layer.

## What it does

Given a set of seed URLs, the system crawls the web concurrently, extracts text and links from each page, builds an inverted index with TF-IDF scores, and exposes a search endpoint that ranks results using either a custom fuzzy matcher or Postgres's built-in full-text search.

## Architecture

<img width="591" height="431" alt="Untitled Diagram drawio" src="https://github.com/user-attachments/assets/4b3a5b20-ecfe-4a82-b5e9-8be2cfced8aa" />

## Stack

| Layer | Technology |
|---|---|
| Crawler / Indexer | Go |
| Queue / dedup | Redis + Lua scripts |
| Storage | PostgreSQL |
| Search API | Python (Flask, SQLAlchemy, NLTK) |
| Frontend | HTML / CSS / JS |
| Infrastructure | Docker Compose |

## How each component works

### Crawler (`crawler/worker.go`)

`CrawlerMain()` seeds from a list of major news and reference sites (Wikipedia, BBC, Reuters, etc.) and fans out from there. Each batch of URLs is processed concurrently — one goroutine per URL — using Go's `net/html` parser to extract:

- **Text** — visible content only, skipping `<script>`, `<style>`, `<code>` tags
- **Links** — all `<a href>` resolved to absolute URLs, http/https only
- **Images** — `<img src>` with `alt` text and nearby parent text for context

### URL queue (`crawler/Urlqueue.go`)

Backed by Redis with atomic deduplication. A Lua script (`checkAndPushBatch.lua`) does `SADD` into a `visited` set and `RPUSH` into `crawlQueue` atomically — so no URL is ever processed twice, even with concurrent writers.

```lua
-- checkAndPushBatch.lua
for i, url in ipairs(ARGV) do
    if redis.call("SADD", KEYS[1], url) == 1 then
        redis.call("RPUSH", KEYS[2], url)
    end
end
```

### Indexer (`crawler/indexer.go`)

Pulls batches of 50 pages from the index queue and runs three things in parallel:

1. **Writes** documents, images, and link graph to Postgres (concurrent goroutines via `sync.WaitGroup`)
2. **Builds** a partial inverted index per document (one goroutine each), collecting raw TF counts
3. **Merges** all partial indices into a global `ReverseIndex` and bulk-inserts via `pgx.CopyFrom`

Text is cleaned (letters/numbers only, lowercased), stop words removed, and each term is stemmed with the Porter algorithm before counting.

### Search API (`backend/main.py`)

Two search modes on `/search?q=<query>`:

**Default — custom TF-IDF + fuzzy matching:**
The query is tokenized and stemmed (matching the indexer's pipeline), then passed to `search_with_levenshtein()` — a Postgres function that scores documents by TF-IDF but accepts terms within Levenshtein distance 2. Typos still work; fuzzy matches are penalised by a small factor.

**Postgres tsvector (`?ts=true`):**
Uses `plainto_tsquery` + `ts_rank` — Postgres's built-in full-text search. Faster and simpler but less control over ranking.

Image search is available at `/searchImage?q=<query>` via tsvector on image alt text and nearby text.

## Database schema

```
documents        — url, title, content, content_vector (tsvector)
inverted_index   — term, document_url, tf
tf_idf           — document_url, term, tf, idf, tf_idf
links            — from_doc, to_doc
images           — image_url, document_url, alt_text, nearby_text, image_vector
```
