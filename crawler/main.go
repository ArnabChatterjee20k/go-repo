package main

import (
	"fmt"
	"math/rand"
	"time"
)

// an event dirven crawler with no locks , just channels

// a fake web
type URL struct {
	urls []string
}

var web = map[string]URL{
	"https://google.com": {
		urls: []string{
			"https://google.com/search",
			"https://youtube.com",
			"https://github.com",
		},
	},
	"https://google.com/search": {
		urls: []string{
			"https://wikipedia.org",
			"https://stackoverflow.com",
		},
	},
	"https://youtube.com": {
		urls: []string{
			"https://youtube.com/trending",
			"https://youtube.com/music",
		},
	},
	"https://youtube.com/trending": {
		urls: []string{
			"https://reddit.com",
		},
	},
	"https://youtube.com/music": {
		urls: []string{
			"https://youtube.com/trending",
		},
	},
	"https://github.com": {
		urls: []string{
			"https://github.com/explore",
			"https://stackoverflow.com",
		},
	},
	"https://github.com/explore": {
		urls: []string{
			"https://appwrite.io",
		},
	},
	"https://stackoverflow.com": {
		urls: []string{
			"https://stackoverflow.com/questions",
		},
	},
	"https://stackoverflow.com/questions": {
		urls: []string{
			"https://github.com",
		},
	},
	"https://wikipedia.org": {
		urls: []string{
			"https://wikipedia.org/wiki/Go",
		},
	},
	"https://wikipedia.org/wiki/Go": {
		urls: []string{
			"https://stackoverflow.com",
		},
	},
	"https://reddit.com": {
		urls: []string{
			"https://reddit.com/r/programming",
		},
	},
	"https://reddit.com/r/programming": {
		urls: []string{
			"https://github.com/explore",
		},
	},
	"https://appwrite.io": {
		urls: []string{
			"https://github.com",
		},
	},
}

func fetch(url string) []string {
	return web[url].urls
}

type Crawler struct {
	urls  chan []string
	count int
	seen  map[string]bool
}

func NewCrawler() Crawler {
	return Crawler{urls: make(chan []string), count: 0, seen: make(map[string]bool)}
}

func (crawler *Crawler) spawn(url string) {
	time.Sleep(time.Duration(rand.Float64() * float64(time.Millisecond)))
	// new discovered urls
	crawler.urls <- web[url].urls
}

func (crawler *Crawler) Crawl(url string) {
	go func() {
		crawler.urls <- []string{url}
	}()
	urlToProcess := 1

	for {
		urlsToCrawl := <-crawler.urls
		for _, url := range urlsToCrawl {
			if crawler.seen[url] == false {
				crawler.count++
				crawler.seen[url] = true
				go crawler.spawn(url)
				urlToProcess++
			}
		}
		// urls still to process
		urlToProcess--

		if urlToProcess == 0 {
			break
		}
	}
}

func main() {
	crawler := NewCrawler()
	crawler.Crawl("https://google.com")
	TOTAL_URLS := 14
	fmt.Println(crawler.count == TOTAL_URLS)
}
