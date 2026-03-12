package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

var version = "dev"

// --- XML structs ---

type URLSet struct {
	XMLName xml.Name `xml:"urlset"`
	URLs    []URL    `xml:"url"`
}

type URL struct {
	Loc string `xml:"loc"`
}

type SitemapIndex struct {
	XMLName  xml.Name       `xml:"sitemapindex"`
	Sitemaps []SitemapEntry `xml:"sitemap"`
}

type SitemapEntry struct {
	Loc string `xml:"loc"`
}

// --- Config ---

type Config struct {
	Sitemaps []string
	Delay    time.Duration
	Timeout  time.Duration
	Verbose  bool
}

// --- Results ---

type Result struct {
	URL     string
	Status  int
	Elapsed time.Duration
	Err     error
}

// --- HTTP client ---

func newClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
}

// --- Sitemap fetching ---

// fetchSitemap fetches a sitemap URL and returns all page URLs.
// If the URL points to a sitemap index, it recursively fetches every child sitemap.
func fetchSitemap(client *http.Client, sitemapURL string, depth int) ([]string, error) {
	if depth > 4 {
		return nil, fmt.Errorf("sitemap nesting too deep")
	}

	req, err := http.NewRequest("GET", sitemapURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; CacheWarmer/"+version+")")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Check for sitemap index first — expands into multiple child sitemaps.
	var index SitemapIndex
	if err := xml.Unmarshal(body, &index); err == nil && len(index.Sitemaps) > 0 {
		fmt.Printf("   Sitemap index found — %d child sitemaps\n", len(index.Sitemaps))
		var allURLs []string
		for _, entry := range index.Sitemaps {
			loc := strings.TrimSpace(entry.Loc)
			if loc == "" {
				continue
			}
			fmt.Printf("     📄 %s\n", loc)
			urls, err := fetchSitemap(client, loc, depth+1)
			if err != nil {
				fmt.Printf("        ❌ Failed: %v\n", err)
				continue
			}
			allURLs = append(allURLs, urls...)
		}
		return allURLs, nil
	}

	// Regular URL set.
	var urlset URLSet
	if err := xml.Unmarshal(body, &urlset); err != nil {
		return nil, fmt.Errorf("XML parse error: %w", err)
	}

	urls := make([]string, 0, len(urlset.URLs))
	for _, u := range urlset.URLs {
		if u.Loc != "" {
			urls = append(urls, strings.TrimSpace(u.Loc))
		}
	}
	return urls, nil
}

// --- Cache warming ---

func warmURL(client *http.Client, rawURL string) Result {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return Result{URL: rawURL, Err: err}
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; CacheWarmer/"+version+")")

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		return Result{URL: rawURL, Elapsed: elapsed, Err: err}
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) // drain body to complete the request

	return Result{URL: rawURL, Status: resp.StatusCode, Elapsed: elapsed}
}

// --- Formatting helpers ---

func printResult(r Result, index, total int, verbose bool) {
	prefix := fmt.Sprintf("[%d/%d]", index, total)

	if r.Err != nil {
		fmt.Printf("  ❌ %s ERROR — %s (%v)\n", prefix, r.URL, r.Err)
		return
	}

	symbol := "✅"
	if r.Status >= 400 {
		symbol = "⚠️ "
	} else if r.Status >= 300 {
		symbol = "↪️ "
	}

	if verbose {
		fmt.Printf("  %s %s %d (%dms) — %s\n", symbol, prefix, r.Status, r.Elapsed.Milliseconds(), r.URL)
	} else {
		fmt.Printf("  %s %s %d (%.2fs) — %s\n", symbol, prefix, r.Status, r.Elapsed.Seconds(), r.URL)
	}
}

func separator() {
	fmt.Println(strings.Repeat("─", 65))
}

// --- Main ---

func main() {
	sitemapsFlag := flag.String("sitemaps", "", "Comma-separated sitemap URLs — supports sitemap indexes (required)")
	delayMs := flag.Int("delay", 1500, "Delay between requests in milliseconds")
	timeoutSec := flag.Int("timeout", 30, "HTTP request timeout in seconds")
	verbose := flag.Bool("verbose", false, "Show response times in milliseconds")
	versionFlag := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("cache-warmer %s\n", version)
		os.Exit(0)
	}

	if *sitemapsFlag == "" {
		fmt.Fprintln(os.Stderr, "Error: -sitemaps flag is required")
		fmt.Fprintln(os.Stderr, "Example: cache-warmer -sitemaps https://example.com/sitemap.xml")
		os.Exit(1)
	}

	var sitemaps []string
	for _, s := range strings.Split(*sitemapsFlag, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			sitemaps = append(sitemaps, s)
		}
	}

	cfg := Config{
		Sitemaps: sitemaps,
		Delay:    time.Duration(*delayMs) * time.Millisecond,
		Timeout:  time.Duration(*timeoutSec) * time.Second,
		Verbose:  *verbose,
	}

	client := newClient(cfg.Timeout)

	separator()
	fmt.Printf("  🔥 Cache Warmer %s\n", version)
	fmt.Printf("  Started : %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("  Delay   : %dms between requests\n", *delayMs)
	fmt.Printf("  Timeout : %ds per request\n", *timeoutSec)
	separator()

	totalVisited := 0
	totalSuccess := 0
	totalFailed := 0

	for _, sitemapURL := range cfg.Sitemaps {
		fmt.Printf("\n📄 Fetching sitemap: %s\n", sitemapURL)

		urls, err := fetchSitemap(client, sitemapURL, 0)
		if err != nil {
			fmt.Printf("   ❌ Failed to fetch sitemap: %v\n", err)
			continue
		}

		fmt.Printf("   Found %d URLs — warming now...\n\n", len(urls))
		sitemapSuccess := 0

		for i, rawURL := range urls {
			result := warmURL(client, rawURL)
			printResult(result, i+1, len(urls), cfg.Verbose)

			totalVisited++
			if result.Err == nil && result.Status < 400 {
				totalSuccess++
				sitemapSuccess++
			} else {
				totalFailed++
			}

			if i < len(urls)-1 {
				time.Sleep(cfg.Delay)
			}
		}

		fmt.Printf("\n   ✔ Sitemap complete: %d/%d succeeded\n", sitemapSuccess, len(urls))
	}

	fmt.Println()
	separator()
	fmt.Printf("  ✅ Done! %d/%d pages warmed successfully\n", totalSuccess, totalVisited)
	if totalFailed > 0 {
		fmt.Printf("  ⚠️  %d pages failed\n", totalFailed)
	}
	fmt.Printf("  Finished: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	separator()
}
