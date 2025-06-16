package links_fetcher_test

import (
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"

	"github.com/bengesoff/web-crawler/links_fetcher"
)

type mockHttpClient struct {
	handler func(w http.ResponseWriter, r *http.Request)
}

func (c *mockHttpClient) Do(request *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	c.handler(w, request)
	return w.Result(), nil
}

func TestFetchAndGetLinks(t *testing.T) {
	type args struct {
		rootUrl         *url.URL
		responseHeaders map[string]string
		responseBody    []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "Happy path",
			args: args{
				rootUrl: mustParseUrl("https://example.com/"),
				responseHeaders: map[string]string{
					"Content-Type": "text/html; charset=utf-8",
				},
				responseBody: []byte(`
					<!DOCTYPE html>
					<html>
						<body>
							<a href="https://example.com/first" />
							<a href="https://another-domain.com/second" />
							<a href="/relative" />
							<div>
								<a href="/nested" />
							</div>
						</body>
					</html>`),
			},
			want: []string{
				"https://example.com/first",
				"https://another-domain.com/second",
				"https://example.com/relative",
				"https://example.com/nested",
			},
			wantErr: false,
		},
		{
			name: "Incorrect content type",
			args: args{
				rootUrl: mustParseUrl("https://example.com/"),
				responseHeaders: map[string]string{
					"Content-Type": "application/json",
				},
				responseBody: []byte(`
					<!DOCTYPE html>
					<html>
						<body>
							<a href="https://example.com/first" />
						</body>
					</html>`),
			},
			wantErr: true,
		},
		{
			name: "No links",
			args: args{
				rootUrl: mustParseUrl("https://example.com/"),
				responseHeaders: map[string]string{
					"Content-Type": "text/html; charset=utf-8",
				},
				responseBody: []byte(`
					<!DOCTYPE html>
					<html>
						<body>
						</body>
					</html>`),
			},
			want:    []string{},
			wantErr: false,
		},
		{
			name: "Malformed link",
			args: args{
				rootUrl: mustParseUrl("https://example.com/"),
				responseHeaders: map[string]string{
					"Content-Type": "text/html; charset=utf-8",
				},
				responseBody: []byte(`
					<!DOCTYPE html>
					<html>
						<body>
							<a href="https://example.com/first" />
							<a href="wrong@url:huh@notvalid*" />
						</body>
					</html>`),
			},
			want: []string{
				"https://example.com/first",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpClient := &mockHttpClient{func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tt.args.rootUrl, r.URL)
				for k, v := range tt.args.responseHeaders {
					w.Header().Add(k, v)
				}
				_, _ = w.Write(tt.args.responseBody)
			}}

			got, err := links_fetcher.FetchAndGetLinks(t.Context(), httpClient, tt.args.rootUrl)
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchAndGetLinks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(tt.want) == 0 {
				return
			}

			// convert to map first to remove duplicates induced by the HTML parsing package
			gotLinksMap := map[string]struct{}{}
			for link := range got {
				gotLinksMap[link.String()] = struct{}{}
			}
			gotLinks := slices.Collect(maps.Keys(gotLinksMap))

			if diff := cmp.Diff(tt.want, gotLinks, cmpopts.SortSlices(compareStrings)); diff != "" {
				t.Errorf("FetchAndGetLinks() mismatch (-want +got):\n%s", diff)
			}
		})
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
