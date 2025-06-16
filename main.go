package main

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/bengesoff/web-crawler/links_fetcher"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Missing required argument: URL to crawl")
	}

	rootUrl, err := url.Parse(os.Args[1])
	if err != nil {
		log.Fatalf("Failed to parse url: %v", err)
	}

	client := &http.Client{}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	links, err := links_fetcher.FetchAndGetLinks(ctx, client, rootUrl)
	if err != nil {
		log.Fatalf("Failed to fetch links: %v", err)
	}

	for linkedUrl := range links {
		log.Printf("Found link: %s\n", linkedUrl)
	}
}
