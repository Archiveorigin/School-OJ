package imageimport

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"school-oj/apps/api/internal/services"
	"strings"
	"testing"
)

func TestProblemImageAddressPolicy(t *testing.T) {
	ctx := context.Background()
	resolve := func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
	}
	for _, raw := range []string{"http://example.org/a.png", "https://a:b@example.org/a.png", "https://example.org:8443/a.png", "https://example.org/a.png#x", "https://127.0.0.1/a.png", "https://[::1]/a.png", "https://[::ffff:127.0.0.1]/a.png", "https://169.254.169.254/latest/meta-data", "https://100.64.0.1/a.png", "https://192.168.1.2/a.png", "https://[64:ff9b::7f00:1]/a.png", "https://[2002:7f00:1::]/a.png", "https://198.18.0.1/a.png"} {
		if _, _, err := checkedImageURL(ctx, raw, resolve); err == nil {
			t.Errorf("allowed unsafe URL %s", raw)
		}
	}
	u, ip, err := checkedImageURL(ctx, "https://cdn.example.org/a.png?width=100", resolve)
	if err != nil || u.Hostname() != "cdn.example.org" || ip.String() != "93.184.216.34" {
		t.Fatalf("public image blocked: %v", err)
	}
	mixed := func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{ip, netip.MustParseAddr("10.0.0.1")}, nil
	}
	if _, _, err = checkedImageURL(ctx, "https://example.org/a.png", mixed); err == nil {
		t.Fatal("mixed public/private DNS accepted")
	}
}
func TestProblemImageRedirectLimitsAndCanonicalContent(t *testing.T) {
	ctx := context.Background()
	resolve := func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
	}
	png, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jYksAAAAASUVORK5CYII=")
	response := func(code int, body, location string) *http.Response {
		return &http.Response{StatusCode: code, Header: http.Header{"Location": []string{location}}, Body: io.NopCloser(strings.NewReader(body))}
	}
	calls := 0
	asset, err := fetchProblemImage(ctx, "https://example.org/start", resolve, func(_ context.Context, u *url.URL, ip netip.Addr) (*http.Response, error) {
		calls++
		if ip.String() != "93.184.216.34" {
			t.Fatal("unpinned request")
		}
		if u.Path == "/start" {
			return response(302, "", "/image"), nil
		}
		return response(200, string(png), ""), nil
	})
	if err != nil || calls != 2 || asset.ContentType != "image/png" || !strings.HasPrefix(asset.Path, "assets/") || !strings.HasPrefix(asset.Data, "data:image/png;base64,") {
		t.Fatalf("import: %+v %v calls=%d", asset, err, calls)
	}
	calls = 0
	_, err = fetchProblemImage(ctx, "https://example.org/image", resolve, func(context.Context, *url.URL, netip.Addr) (*http.Response, error) {
		calls++
		return response(302, "", "https://127.0.0.1/secret"), nil
	})
	if err == nil || calls != 1 {
		t.Fatal("followed private redirect")
	}
	for _, body := range []string{"<html>denied</html>", "<svg></svg>", string(png) + strings.Repeat("x", services.MaxProblemAssetSize)} {
		if _, err = fetchProblemImage(ctx, "https://example.org/image", resolve, func(context.Context, *url.URL, netip.Addr) (*http.Response, error) {
			return response(200, body, ""), nil
		}); err == nil {
			t.Fatal("invalid/oversized image accepted")
		}
	}
}
