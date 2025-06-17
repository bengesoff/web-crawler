package walker_test

import (
	"log/slog"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/bengesoff/web-crawler/mocks"
	"github.com/bengesoff/web-crawler/walker"
)

func TestLinkWalker_Walk(t *testing.T) {
	startUrl := mustParseUrl("https://example.com/")
	responses := map[string][]byte{
		"https://example.com/": []byte(`
			<html>
				<body>
					<a href="/page1">Page 1</a>
					<a href="/page2">Page 2</a>
					<a href="https://external.com/page">External</a>
				</body>
			</html>
		`),
		"https://example.com/page1": []byte(`
			<html>
				<body>
					<a href="/page2">Page 2</a>
				</body>
			</html>
		`),
		"https://example.com/page2": []byte(`
			<html>
				<body>
					<a href="/">Home</a>
				</body>
			</html>
		`),
	}
	expectedPages := map[string][]string{
		"https://example.com/": {
			"https://example.com/page1",
			"https://example.com/page2",
			"https://external.com/page",
		},
		"https://example.com/page1": {
			"https://example.com/page2",
		},
		"https://example.com/page2": {
			"https://example.com/",
		},
	}

	httpClient := &mocks.MockHttpClient{Handler: func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(responses[r.URL.String()])
	}}

	linkWalker := walker.NewLinkWalker(slog.New(slog.DiscardHandler), startUrl, httpClient)

	err := linkWalker.Walk(t.Context(), startUrl)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	actualPages := map[string][]string{}
	for page := range linkWalker.Pages() {
		links := make([]string, 0, len(page.Links))
		for _, link := range page.Links {
			links = append(links, link.String())
		}
		actualPages[page.Url.String()] = links
	}

	if diff := cmp.Diff(expectedPages, actualPages, cmpopts.SortSlices(compareStrings), cmpopts.SortMaps(compareStrings)); diff != "" {
		t.Errorf("Walk() pages mismatch (-want +got):\n%s", diff)
	}
}

func mustParseUrl(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

func compareStrings(a, b string) bool {
	return a < b
}
