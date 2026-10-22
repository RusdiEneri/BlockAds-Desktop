package filtermanager_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"blockads-desktop/internal/filtermanager"
)

func TestDownloader_ValidateSourceURL(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		allowPrivate bool
		wantErr      error
	}{
		{
			name:         "empty url",
			url:          "",
			allowPrivate: false,
			wantErr:      filtermanager.ErrInvalidURL,
		},
		{
			name:         "insecure http",
			url:          "http://adaway.org/hosts.txt",
			allowPrivate: false,
			wantErr:      filtermanager.ErrInsecureScheme,
		},
		{
			name:         "file scheme",
			url:          "file:///C:/Windows/System32/drivers/etc/hosts",
			allowPrivate: false,
			wantErr:      filtermanager.ErrInsecureScheme,
		},
		{
			name:         "loopback ipv4 blocked",
			url:          "https://127.0.0.1/hosts.txt",
			allowPrivate: false,
			wantErr:      filtermanager.ErrSSRFBlocked,
		},
		{
			name:         "private 192.168.x.x blocked",
			url:          "https://192.168.1.50/hosts.txt",
			allowPrivate: false,
			wantErr:      filtermanager.ErrSSRFBlocked,
		},
		{
			name:         "private 10.x.x.x blocked",
			url:          "https://10.200.1.1/hosts.txt",
			allowPrivate: false,
			wantErr:      filtermanager.ErrSSRFBlocked,
		},
		{
			name:         "cloud metadata ip blocked",
			url:          "https://169.254.169.254/latest/meta-data",
			allowPrivate: false,
			wantErr:      filtermanager.ErrSSRFBlocked,
		},
		{
			name:         "private allowed with explicit flag",
			url:          "https://192.168.1.50/hosts.txt",
			allowPrivate: true,
			wantErr:      nil,
		},
		{
			name:         "valid public https url",
			url:          "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts",
			allowPrivate: false,
			wantErr:      nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := filtermanager.ValidateSourceURL(tc.url, tc.allowPrivate)
			if tc.wantErr != nil {
				if err == nil || !errors.Is(err, tc.wantErr) {
					t.Errorf("expected error %v, got %v", tc.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestDownloader_RedirectAndDowngradeSecurity(t *testing.T) {
	// 1. Redirect downgrade: HTTPS to HTTP must be rejected
	redirectServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "http://insecure.example.com/hosts.txt")
		w.WriteHeader(http.StatusFound)
	}))
	defer redirectServer.Close()

	dl := filtermanager.NewDownloader(filtermanager.DownloaderConfig{
		AllowPrivateNetworks: true, // so httptest server loopback IP is allowed
		Transport:            redirectServer.Client().Transport,
	})

	_, err := dl.Download(context.Background(), filtermanager.DownloadRequest{
		URL: redirectServer.URL,
	})
	if err == nil || !errors.Is(err, filtermanager.ErrRedirectDowngrade) {
		t.Errorf("expected ErrRedirectDowngrade, got %v", err)
	}

	// 2. Loop redirect: exceeding max redirects must be rejected
	loopCount := 0
	loopServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loopCount++
		w.Header().Set("Location", fmt.Sprintf("/redirect-%d", loopCount))
		w.WriteHeader(http.StatusFound)
	}))
	defer loopServer.Close()

	_, err = dl.Download(context.Background(), filtermanager.DownloadRequest{
		URL: loopServer.URL,
	})
	if err == nil || !errors.Is(err, filtermanager.ErrTooManyRedirects) {
		t.Errorf("expected ErrTooManyRedirects, got %v", err)
	}
}

func TestDownloader_ResponseSizeLimit(t *testing.T) {
	// Server returns 200 bytes, but downloader MaxBodyBytes is configured to 50 bytes
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strings.Repeat("A", 200)))
	}))
	defer server.Close()

	dl := filtermanager.NewDownloader(filtermanager.DownloaderConfig{
		AllowPrivateNetworks: true,
		MaxBodyBytes:         50,
	})

	_, err := dl.Download(context.Background(), filtermanager.DownloadRequest{
		URL: server.URL,
	})
	if err == nil || !errors.Is(err, filtermanager.ErrResponseTooLarge) {
		t.Errorf("expected ErrResponseTooLarge, got %v", err)
	}
}

func TestDownloader_ConditionalHeadersAnd304(t *testing.T) {
	receivedETag := ""
	receivedModified := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedETag = r.Header.Get("If-None-Match")
		receivedModified = r.Header.Get("If-Modified-Since")

		if receivedETag == `"etag-123"` {
			w.Header().Set("ETag", `"etag-123"`)
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.Header().Set("ETag", `"etag-123"`)
		w.Header().Set("Last-Modified", "Wed, 21 Oct 2026 07:28:00 GMT")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("127.0.0.1 tracker.example.com\n"))
	}))
	defer server.Close()

	dl := filtermanager.NewDownloader(filtermanager.DownloaderConfig{
		AllowPrivateNetworks: true,
	})

	// 1. Initial unconditional GET
	resp1, err := dl.Download(context.Background(), filtermanager.DownloadRequest{
		URL: server.URL,
	})
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}
	if resp1.StatusCode != http.StatusOK || resp1.NotModified {
		t.Errorf("expected 200 OK, got code=%d notModified=%v", resp1.StatusCode, resp1.NotModified)
	}
	if resp1.ETag != `"etag-123"` {
		t.Errorf("expected ETag \"etag-123\", got %s", resp1.ETag)
	}

	// 2. Conditional GET with ETag & Last-Modified
	resp2, err := dl.Download(context.Background(), filtermanager.DownloadRequest{
		URL:          server.URL,
		ETag:         `"etag-123"`,
		LastModified: "Wed, 21 Oct 2026 07:28:00 GMT",
	})
	if err != nil {
		t.Fatalf("conditional download failed: %v", err)
	}
	if resp2.StatusCode != http.StatusNotModified || !resp2.NotModified {
		t.Errorf("expected 304 Not Modified, got code=%d notModified=%v", resp2.StatusCode, resp2.NotModified)
	}
	if receivedETag != `"etag-123"` || receivedModified != "Wed, 21 Oct 2026 07:28:00 GMT" {
		t.Errorf("server did not receive expected conditional headers: etag=%s, mod=%s", receivedETag, receivedModified)
	}
}
