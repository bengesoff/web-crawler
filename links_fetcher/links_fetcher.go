package links_fetcher

import (
	"context"
	"fmt"
	"iter"
	"mime"
	"net/http"
	"net/url"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type HttpClient interface {
	Do(request *http.Request) (*http.Response, error)
}

// FetchAndGetLinks calls GetAndParseHtml and gives the result to FindLinks.
func FetchAndGetLinks(ctx context.Context, client HttpClient, rootUrl *url.URL) (iter.Seq[*url.URL], error) {
	document, err := GetAndParseHtml(ctx, client, rootUrl)
	if err != nil {
		return nil, err
	}

	return FindLinks(rootUrl, document), nil
}

// GetAndParseHtml uses the given HTTP client to request an HTML document and parses it into a node tree.
func GetAndParseHtml(ctx context.Context, client HttpClient, rootUrl *url.URL) (*html.Node, error) {
	request, err := http.NewRequestWithContext(ctx, "GET", rootUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Add("Accept", "text/html")

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error fetching URL %s: %w", rootUrl, err)
	}
	defer func() { _ = response.Body.Close() }()

	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil {
		return nil, fmt.Errorf("error parsing content type: %w", err)
	}
	if mediaType != "text/html" {
		return nil, fmt.Errorf("unexpected content type: %s", mediaType)
	}

	document, err := html.Parse(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error parsing HTML: %w", err)
	}
	return document, nil
}

// FindLinks walks an HTML document and returns an iterator over the contained links.
// It only handles anchor (<a>) elements, and parses the href attributes. If the link fails to be parsed to a `url.Url`
// then it is skipped.
// Any relative links are converted to absolute ones by making them relative to the given root URL.
// May return duplicates due to https://github.com/golang/go/issues/23487
func FindLinks(rootUrl *url.URL, document *html.Node) iter.Seq[*url.URL] {
	return func(yield func(*url.URL) bool) {
		for node := range document.Descendants() {
			if node.Type == html.ElementNode && node.DataAtom == atom.A {
				for _, attr := range node.Attr {
					if attr.Key == "href" {
						linkedUrl, err := url.Parse(attr.Val)
						if err != nil {
							continue
						}

						linkedUrl = rootUrl.ResolveReference(linkedUrl)
						if !yield(linkedUrl) {
							return
						}
					}
				}
			}
		}
	}
}
