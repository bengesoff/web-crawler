# Web crawler

## Instructions

1. Build with `go build -o crawl .`
2. Run with `./crawl "https://example.com"`
3. Wait!

> You may get rate-limited running against against a large site, and therefore get missing pages.
> Try `./crawl "https://blog.rust-lang.org/2015/05/15/Rust-1.0/"` for a small-ish site that will finish in
> reasonable time.

## Testing

To run the tests:

```
go test -v ./...
```
