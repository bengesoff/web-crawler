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

	"github.com/bengesoff/web-crawler/links_fetcher"
)

type LinkWalker struct {
	logger     *slog.Logger
	rootUrl    *url.URL
	httpClient links_fetcher.HttpClient
	pages      map[string]*PageLinks
}

func NewLinkWalker(logger *slog.Logger, rootUrl *url.URL, httpClient links_fetcher.HttpClient) *LinkWalker {
	return &LinkWalker{
		logger:     logger,
		rootUrl:    rootUrl,
		httpClient: httpClient,
		pages:      map[string]*PageLinks{},
	}
}

func (w *LinkWalker) Walk(ctx context.Context, walkUrl *url.URL) error {
	w.logger.InfoContext(ctx, "Visiting page", slog.String("walk_url", walkUrl.String()))

	// TODO: cache to avoid rewalking?
	links, err := links_fetcher.FetchAndGetLinks(ctx, w.httpClient, walkUrl)
	if err != nil {
		// don't attempt to walk non-HTML pages
		if errors.Is(err, links_fetcher.ErrUnsupportedMediaType) {
			w.logger.DebugContext(ctx, "Unsupported media type", slog.String("walk_url", walkUrl.String()))
			return nil
		}
		w.logger.ErrorContext(ctx, "Failed to fetch links", slog.String("walk_url", walkUrl.String()))
		// TODO: could also implement retries
		return nil
	}

	pageLinks := &PageLinks{
		Url:   walkUrl,
		Links: slices.Collect(links),
	}

	w.pages[stripUrl(walkUrl).String()] = pageLinks

	for _, link := range pageLinks.Links {
		if link.Hostname() == w.rootUrl.Hostname() {
			if _, ok := w.pages[stripUrl(link).String()]; !ok {
				err := w.Walk(ctx, link)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (w *LinkWalker) Pages() iter.Seq[*PageLinks] {
	return maps.Values(w.pages)
}

type PageLinks struct {
	Url   *url.URL
	Links []*url.URL
}

func (s *PageLinks) String() string {
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
