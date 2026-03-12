# cache-warmer

A fast, lightweight CLI tool that crawls XML sitemaps and warms your web server's cache by visiting every URL. Supports both regular sitemaps and **sitemap indexes** — just point it at your root `sitemap.xml` and it automatically discovers and crawls all child sitemaps.

Ideal for WordPress sites, static site generators, or any server-side rendered app where cache priming matters.

## Features

- Automatically detects and expands sitemap indexes into child sitemaps
- Configurable delay between requests to avoid overloading your server
- Configurable per-request timeout
- Clean progress output with status codes and response times
- Zero external dependencies — pure Go standard library

## Installation

### Homebrew (recommended)

```sh
brew tap harrisonratcliffe/tap
brew install cache-warmer
```

### Download binary

Download the latest release for your platform from the [releases page](https://github.com/harrisonratcliffe/cache-warmer/releases).

### Build from source

Requires Go 1.22+.

```sh
git clone https://github.com/harrisonratcliffe/cache-warmer.git
cd cache-warmer
go build -o cache-warmer .
```

## Usage

```sh
cache-warmer -sitemaps https://example.com/sitemap.xml
```

Pass your root `sitemap.xml` and the tool handles the rest. If it's a sitemap index, all child sitemaps are discovered and expanded automatically. You can also pass multiple sitemap URLs as a comma-separated list.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-sitemaps` | *(required)* | Comma-separated list of sitemap URLs |
| `-delay` | `1500` | Delay between requests in milliseconds |
| `-timeout` | `30` | Per-request HTTP timeout in seconds |
| `-verbose` | `false` | Show response times in milliseconds instead of seconds |
| `-version` | | Print version and exit |

### Examples

**Single sitemap:**
```sh
cache-warmer -sitemaps https://example.com/sitemap.xml
```

**Multiple sitemaps:**
```sh
cache-warmer -sitemaps https://example.com/sitemap.xml,https://example.com/news-sitemap.xml
```

**Sitemap index (automatically expands):**
```sh
# If sitemap.xml is a sitemap index, all child sitemaps are discovered and warmed automatically
cache-warmer -sitemaps https://example.com/sitemap.xml
```

**Faster warming with lower delay:**
```sh
cache-warmer -sitemaps https://example.com/sitemap.xml -delay 500
```

**Custom timeout:**
```sh
cache-warmer -sitemaps https://example.com/sitemap.xml -timeout 60
```

**Verbose output (millisecond precision):**
```sh
cache-warmer -sitemaps https://example.com/sitemap.xml -verbose
```

### Example output

```
─────────────────────────────────────────────────────────────────
  🔥 Cache Warmer v1.0.0
  Started : 2025-06-01 09:00:00
  Delay   : 1500ms between requests
  Timeout : 30s per request
─────────────────────────────────────────────────────────────────

📄 Fetching sitemap: https://example.com/sitemap.xml
   Sitemap index found — 3 child sitemaps
     📄 https://example.com/post-sitemap.xml
     📄 https://example.com/page-sitemap.xml
     📄 https://example.com/category-sitemap.xml
   Found 142 URLs — warming now...

  ✅ [1/142] 200 (0.43s) — https://example.com/
  ✅ [2/142] 200 (0.51s) — https://example.com/about
  ↪️  [3/142] 301 (0.12s) — https://example.com/old-page
  ...

   ✔ Sitemap complete: 141/142 succeeded

─────────────────────────────────────────────────────────────────
  ✅ Done! 141/142 pages warmed successfully
  Finished: 2025-06-01 09:03:33
─────────────────────────────────────────────────────────────────
```

## License

MIT — see [LICENSE](LICENSE).
