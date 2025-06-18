# Web crawler

## Instructions

1. Build with `go build -o crawl .`
2. Run with `./crawl "https://example.com"`
3. Wait!

> You may get rate-limited running against a large site, and therefore get missing pages.
> Try `./crawl "https://blog.rust-lang.org/2015/05/15/Rust-1.0/"` for a small-ish site that will finish in
> reasonable time.

## Testing

To run the tests:

```
go test -v ./...
```

## What's missing

- `robots.txt` isn't respected
- Rate limiting is quite crude - it just limits to 1 request every 50ms instead of checking the `RateLimit-*` headers
- Error handling and non-HTML pages:
  - it doesn't attempt to decide if a page is HTML or a file based on the path (or a `mailto:` link)
  - it doesn't distinguish between retryable and non-retryable errors
- Query parameters and fragments are stripped from URLs so might not work well with some pages
- Storing results in between runs to avoid needing to re-fetch
- More test cases
- Configuration and tuning of buffered channel sizes, number of workers, timeouts, rate limits
