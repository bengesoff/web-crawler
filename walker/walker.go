package walker

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"maps"
	"net/url"
	"slices"
	"strings"
	"time"

	"golang.org/x/time/rate"

	"github.com/bengesoff/web-crawler/links_fetcher"
)

type LinkWalker struct {
	logger     *slog.Logger
	rootUrl    *url.URL
	httpClient links_fetcher.HttpClient
	pages      map[string]*PageLinks
	limiter    *rate.Limiter
}

func NewLinkWalker(logger *slog.Logger, rootUrl *url.URL, httpClient links_fetcher.HttpClient) *LinkWalker {
	return &LinkWalker{
		logger:     logger,
		rootUrl:    rootUrl,
		httpClient: httpClient,
		pages:      map[string]*PageLinks{},
		limiter:    rate.NewLimiter(rate.Every(50*time.Millisecond), 1),
	}
}

func (w *LinkWalker) Walk(ctx context.Context, walkUrl *url.URL) error {
	linksToFetch := make(chan *url.URL, 100000)
	results := make(chan *PageLinks, 10)
	done := make(chan struct{})

	// Avoids requesting a page multiple times if it is linked from multiple pages.
	// We could request it multiple times if it errors, or if it is linked from multiple pages and none of the fetches
	// have completed yet.
	seen := map[string]struct{}{
		stripUrl(walkUrl).String(): {},
	}

	// A goroutine to process the results and kick off new fetch jobs.
	// It mutates the state (`w.pages`), but it's the only goroutine that does so until the `done` channel closes, so
	// concurrent access is avoided, and we don't need a mutex.
	go func() {
		defer close(done)

		workInProgress := 1
		for {
			select {
			case pageLinks := <-results:
				workInProgress--

				if pageLinks.Error == nil {
					w.pages[stripUrl(pageLinks.Url).String()] = pageLinks

					for _, link := range pageLinks.Links {
						if link.Hostname() == w.rootUrl.Hostname() {
							linkKey := stripUrl(link).String()

							// skip queuing the link if we've already seen it
							if _, ok := seen[linkKey]; !ok {
								seen[linkKey] = struct{}{}
								workInProgress++

								select {
								case linksToFetch <- link:
								default:
									w.logger.ErrorContext(ctx, "Failed to send URL to channel for fetching", slog.String("walk_url", link.String()))
									workInProgress--
								}
							}
						}
					}
				}

				if workInProgress == 0 {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	for range 50 {
		go worker(ctx, w.logger, w.httpClient, w.limiter, linksToFetch, results)
	}

	linksToFetch <- walkUrl

	<-done
	close(linksToFetch)
	return nil
}

// worker iterates over the `linksToFetch` channel and produces the links it finds to the `results` channel
func worker(
	ctx context.Context,
	logger *slog.Logger,
	httpClient links_fetcher.HttpClient,
	limiter *rate.Limiter,
	linksToFetch <-chan *url.URL,
	results chan<- *PageLinks,
) {
	for pageUrl := range linksToFetch {
		logger.InfoContext(ctx, "Visiting page", slog.String("walk_url", pageUrl.String()))

		if err := limiter.Wait(ctx); err != nil {
			results <- &PageLinks{Url: pageUrl, Error: err}
			continue
		}

		links, err := links_fetcher.FetchAndGetLinks(ctx, httpClient, pageUrl)
		if err != nil {
			results <- &PageLinks{Url: pageUrl, Error: err}
			// don't attempt to walk non-HTML pages
			if errors.Is(err, links_fetcher.ErrUnsupportedMediaType) {
				logger.InfoContext(ctx, "Unsupported media type", slog.String("walk_url", pageUrl.String()))
				continue
			}
			logger.ErrorContext(ctx, "Failed to fetch links", slog.String("walk_url", pageUrl.String()))
			// TODO: could also implement retries
			continue
		}

		pageLinks := &PageLinks{
			Url:   pageUrl,
			Links: slices.Collect(links),
		}
		results <- pageLinks
	}
}

func (w *LinkWalker) Pages() iter.Seq[*PageLinks] {
	return maps.Values(w.pages)
}

type PageLinks struct {
	Url   *url.URL
	Links []*url.URL
	Error error
}

func (s *PageLinks) String() string {
	if s.Error != nil {
		return fmt.Sprintf("Error fetching links for %s: %s\n", s.Url, s.Error.Error())
	}
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("Links from %s:\n", s.Url.String()))
	for _, link := range s.Links {
		builder.WriteString("\t" + link.String() + "\n")
	}
	return builder.String()
}

func stripUrl(rawUrl *url.URL) *url.URL {
	strippedUrl := rawUrl.JoinPath("/") // appends trailing slash and copies
	strippedUrl.Fragment = ""
	strippedUrl.RawQuery = ""
	return strippedUrl
}
