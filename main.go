package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/bengesoff/web-crawler/walker"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	if len(os.Args) < 2 {
		logger.Error("Missing required argument: URL to crawl")
		os.Exit(1)
	}

	rootUrl, err := url.Parse(os.Args[1])
	if err != nil {
		logger.Error("Failed to parse url", slog.String("err", err.Error()))
		os.Exit(1)
	}

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	linkWalker := walker.NewLinkWalker(logger, rootUrl, client)
	err = linkWalker.Walk(ctx, rootUrl)
	if err != nil {
		logger.Error("Failed to walk", slog.String("err", err.Error()))
		os.Exit(1)
	}

	for sitemap := range linkWalker.Pages() {
		fmt.Print(sitemap.String())
	}
}
