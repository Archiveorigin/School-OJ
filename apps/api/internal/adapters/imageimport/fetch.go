package imageimport

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"school-oj/apps/api/internal/services"
	"strings"
	"time"
)

type ImportedProblemImage struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	ContentType string `json:"content_type"`
	Data        string `json:"data"`
}

var nonPublicImageNetworks = func() []netip.Prefix {
	var prefixes []netip.Prefix
	for _, p := range []string{
		"0.0.0.0/8", "10.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8", "169.254.0.0/16", "172.16.0.0/12",
		"192.0.0.0/24", "192.0.2.0/24", "192.168.0.0/16", "192.88.99.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "224.0.0.0/3",
		"::/96", "64:ff9b::/96", "64:ff9b:1::/48", "100::/64", "2001::/23", "2001:db8::/32", "2002::/16", "3fff::/20", "fc00::/7", "fe80::/10", "ff00::/8",
	} {
		prefixes = append(prefixes, netip.MustParsePrefix(p))
	}
	return prefixes
}()

func publicImageIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.Zone() != "" {
		return false
	}
	if ip.Is6() && !netip.MustParsePrefix("2000::/3").Contains(ip) {
		return false
	}
	for _, prefix := range nonPublicImageNetworks {
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
}

type imageResolver func(context.Context, string, string) ([]netip.Addr, error)
type imageRequest func(context.Context, *url.URL, netip.Addr) (*http.Response, error)

func checkedImageURL(ctx context.Context, raw string, resolve imageResolver) (*url.URL, netip.Addr, error) {
	u, err := url.Parse(raw)
	if err != nil || len(raw) > 4096 || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || (u.Port() != "" && u.Port() != "443") || u.Fragment != "" {
		return nil, netip.Addr{}, fmt.Errorf("图片地址须为不含账号、片段的公开 HTTPS 地址（端口 443）")
	}
	var addresses []netip.Addr
	if ip, e := netip.ParseAddr(u.Hostname()); e == nil {
		addresses = []netip.Addr{ip}
	} else {
		addresses, err = resolve(ctx, "ip", u.Hostname())
	}
	if err != nil || len(addresses) == 0 {
		return nil, netip.Addr{}, fmt.Errorf("图片域名无法解析")
	}
	for _, ip := range addresses {
		if !publicImageIP(ip) {
			return nil, netip.Addr{}, fmt.Errorf("不允许从内网、保留地址或本机导入图片")
		}
	}
	return u, addresses[0].Unmap(), nil
}

func requestPinnedImage(ctx context.Context, u *url.URL, ip netip.Addr) (*http.Response, error) {
	// No environment proxy, no cookies, no credentials and no implicit redirects.
	// DNS is checked once and this connection uses that exact address; HTTPS still
	// verifies the original hostname. Each redirect goes through validation again.
	transport := &http.Transport{
		Proxy: nil, DisableKeepAlives: true, MaxResponseHeaderBytes: 32 * 1024,
		TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 5 * time.Second,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "tcp", net.JoinHostPort(ip.String(), "443"))
		},
	}
	defer transport.CloseIdleConnections()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "image/png,image/jpeg,image/gif,image/webp")
	return transport.RoundTrip(req)
}

func FetchProblemImage(ctx context.Context, raw string) (ImportedProblemImage, error) {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	return fetchProblemImage(ctx, raw, net.DefaultResolver.LookupNetIP, requestPinnedImage)
}
func fetchProblemImage(ctx context.Context, raw string, resolve imageResolver, request imageRequest) (ImportedProblemImage, error) {
	for redirect := 0; redirect <= 3; redirect++ {
		u, ip, err := checkedImageURL(ctx, raw, resolve)
		if err != nil {
			return ImportedProblemImage{}, err
		}
		response, err := request(ctx, u, ip)
		if err != nil {
			return ImportedProblemImage{}, fmt.Errorf("无法下载图片，请检查地址或上传本地图片")
		}
		if response.StatusCode >= 300 && response.StatusCode < 400 {
			response.Body.Close()
			location, e := url.Parse(response.Header.Get("Location"))
			if e != nil || location.String() == "" {
				return ImportedProblemImage{}, fmt.Errorf("图片重定向地址无效")
			}
			raw = u.ResolveReference(location).String()
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(response.Body, services.MaxProblemAssetSize+1))
		response.Body.Close()
		if response.StatusCode != 200 || readErr != nil {
			return ImportedProblemImage{}, fmt.Errorf("图片下载失败，请上传本地图片")
		}
		if len(body) == 0 || len(body) > services.MaxProblemAssetSize {
			return ImportedProblemImage{}, fmt.Errorf("图片不能为空或超过 5 MiB")
		}
		mime := http.DetectContentType(body)
		ext := map[string]string{"image/png": "png", "image/jpeg": "jpg", "image/gif": "gif", "image/webp": "webp"}[mime]
		if ext == "" || strings.Contains(strings.ToLower(response.Header.Get("Content-Type")), "text/html") {
			return ImportedProblemImage{}, fmt.Errorf("仅支持 PNG、JPEG、GIF、WebP 图片，不支持网页或 SVG")
		}
		path := fmt.Sprintf("assets/%x.%s", sha256.Sum256(body), ext)
		if _, err = services.ValidateProblemAsset(path, body); err != nil {
			return ImportedProblemImage{}, fmt.Errorf("图片内容无效")
		}
		return ImportedProblemImage{Name: "导入图片." + ext, Path: path, ContentType: mime, Data: "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(body)}, nil
	}
	return ImportedProblemImage{}, fmt.Errorf("图片重定向次数过多，请上传本地图片")
}
