// Package fetch provides data fetching operations;
// handles retrieving content from various sources like files, URLs, and APIs.
package fetch

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// File size limits to prevent memory overload
// TODO: make this configurable via command-line flags OR eliminate w/ streaming
const (
	MaxFileSizeBytes = 50 * 1024 * 1024  // 50MB limit for files
	MaxHTTPSizeBytes = 100 * 1024 * 1024 // 100MB limit for HTTP content (may not have Content-Length)
)

// HTTP client timeout configuration; currently set to reasonable defaults
// TODO: make this configurable via command-line flags
const HTTPRequestTimeout = 30 * time.Second

// specific timeout thresholds (based on HTTPRequestTimeout)
var (
	HTTPDialTimeout           = HTTPRequestTimeout / 6 // ~17%, max time to wait for network connection
	HTTPTLSTimeout            = HTTPRequestTimeout / 6 // ~17%, max time to wait for TLS handshake
	HTTPResponseHeaderTimeout = HTTPRequestTimeout / 2 // 50%, max time for response headers (usually the longest phase)
)

// limitedReadCloser wraps an io.ReadCloser to enforce size limits
type limitedReadCloser struct {
	io.ReadCloser
	N      int64  // max bytes remaining
	source string // for error messages
}

func (l *limitedReadCloser) Read(p []byte) (n int, err error) {
	if l.N <= 0 {
		return 0, fmt.Errorf("content from %q exceeds size limit", l.source)
	}
	if int64(len(p)) > l.N {
		p = p[0:l.N]
	}
	n, err = l.ReadCloser.Read(p)
	l.N -= int64(n)
	return
}

// httpClient is a shared HTTP client with appropriate timeouts to prevent indefinite hangs.
// this should be safe for concurrent use across multiple goroutines.
var httpClient = &http.Client{
	Timeout: HTTPRequestTimeout,
	Transport: &http.Transport{
		Dial: (&net.Dialer{
			Timeout: HTTPDialTimeout,
		}).Dial,
		TLSHandshakeTimeout:   HTTPTLSTimeout,
		ResponseHeaderTimeout: HTTPResponseHeaderTimeout,
		// disable keep-alives to avoid connection reuse issues
		DisableKeepAlives: true,
	},
}

// GetContent retrieves content from various source types and returns an io.ReadCloser.
// It supports three types of sources:
//   - "-" reads from standard input
//   - URLs starting with "http://" or "https://" are fetched via HTTP
//   - everything else is treated as a local file path
//
// ctx allows for cancellation and timeout control of fetch operations.
func GetContent(ctx context.Context, source string) (io.ReadCloser, error) {
	switch {
	case source == "-":
		// Wrap stdin with size limit to prevent memory overload
		// This is useful for piping content directly into the program
		return &limitedReadCloser{
			ReadCloser: os.Stdin,
			N:          MaxFileSizeBytes,
			source:     "stdin",
		}, nil
	case strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://"):
		return fetchURL(ctx, source)
	default:
		return fetchFile(ctx, source)
	}
}

// fetchURL retrieves content from an HTTP or HTTPS URL using a client with timeout configuration
// ctx allows for cancellation and timeout control of HTTP requests.
func fetchURL(ctx context.Context, url string) (io.ReadCloser, error) {
	// create request with User-Agent and context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for URL %q: %w", url, err)
	}
	req.Header.Set("User-Agent", "sift/0.1")

	// execute request using shared client
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL %q: %w", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP request failed for URL %q: status %d %s", url, resp.StatusCode, resp.Status)
	}

	// check content-length header if present to prevent memory overload (and exhaustion attacks)
	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		if size, err := strconv.ParseInt(contentLength, 10, 64); err == nil {
			if size > MaxHTTPSizeBytes {
				resp.Body.Close()
				return nil, fmt.Errorf("HTTP content too large (%d bytes > %d bytes limit)",
					size, MaxHTTPSizeBytes)
			}
		}
	}

	// For HTTP content without Content-Length, use limitedReadCloser to prevent memory overload
	return &limitedReadCloser{
		ReadCloser: resp.Body,
		N:          MaxHTTPSizeBytes,
		source:     url,
	}, nil
}

// fetchFile opens a local file for reading with better error messages
// ctx is accepted for API consistency but not actually used for local file operations
func fetchFile(ctx context.Context, path string) (io.ReadCloser, error) {
	// check if file exists and get size
	fileInfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("file %q does not exist", path)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to access file %q: %w", path, err)
	}

	// check file size before opening to prevent memory overload
	if fileInfo.Size() > MaxFileSizeBytes {
		return nil, fmt.Errorf("file %q is too large (%d bytes > %d bytes limit)",
			path, fileInfo.Size(), MaxFileSizeBytes)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", path, err)
	}

	return file, nil
}
