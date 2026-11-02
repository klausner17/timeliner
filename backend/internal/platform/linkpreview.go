package platform

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf8"

	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
	"golang.org/x/sync/singleflight"

	"github.com/klausner17/timeliner/backend/internal/domain"
)

const (
	previewTimeout   = 6 * time.Second
	previewMaxBytes  = 1 << 20
	previewRedirects = 5
	previewTTL       = 6 * time.Hour
	previewFailTTL   = 15 * time.Minute
	previewCacheSize = 2000
	previewUserAgent = "Mozilla/5.0 (compatible; TimelinerBot/1.0; +prévia de links)"
)

var errBlockedAddress = errors.New("endereço de rede interno")

// LinkPreviews busca a prévia de páginas: as metatags Open Graph (og:title,
// og:description, og:image, og:site_name), com <title> e meta description de reserva.
// Só lê o começo da página, guarda o resultado em memória por algumas horas e recusa
// endereços da rede interna (localhost, 10.x, 192.168.x...), mesmo depois de redirect.
type LinkPreviews struct {
	client *http.Client
	group  singleflight.Group

	mu    sync.Mutex
	cache map[string]previewEntry
	now   func() time.Time
}

type previewEntry struct {
	p   domain.LinkPreview
	err error
	exp time.Time
}

// NewLinkPreviews monta o buscador. allowPrivate libera a rede interna; é só para testes.
func NewLinkPreviews(allowPrivate bool) *LinkPreviews {
	dialer := &net.Dialer{Timeout: previewTimeout}
	if !allowPrivate {
		dialer.Control = func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip, err := netip.ParseAddr(host)
			if err != nil || !publicAddr(ip) {
				return errBlockedAddress
			}
			return nil
		}
	}
	transport := &http.Transport{
		// Sem proxy: a conferência de endereço é feita na conexão.
		Proxy:                 nil,
		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   previewTimeout,
		ResponseHeaderTimeout: previewTimeout,
		MaxIdleConns:          20,
		IdleConnTimeout:       30 * time.Second,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   previewTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= previewRedirects {
				return errors.New("redirecionamentos demais")
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return errors.New("redirecionamento para fora da web")
			}
			return nil
		},
	}
	return &LinkPreviews{client: client, cache: map[string]previewEntry{}, now: time.Now}
}

// publicAddr diz se o IP é da internet pública.
func publicAddr(ip netip.Addr) bool {
	ip = ip.Unmap()
	cgnat := netip.MustParsePrefix("100.64.0.0/10")
	return ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !cgnat.Contains(ip)
}

func (l *LinkPreviews) Preview(ctx context.Context, rawURL string) (domain.LinkPreview, error) {
	if e, ok := l.cached(rawURL); ok {
		return e.p, e.err
	}
	v, _, _ := l.group.Do(rawURL, func() (any, error) {
		// A busca não para se quem pediu desistir: o resultado ainda vai para o cache.
		fctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), previewTimeout)
		defer cancel()
		p, err := l.fetch(fctx, rawURL)
		ttl := previewTTL
		if err != nil {
			ttl = previewFailTTL
		}
		e := previewEntry{p: p, err: err, exp: l.now().Add(ttl)}
		l.store(rawURL, e)
		return e, nil
	})
	e := v.(previewEntry)
	return e.p, e.err
}

func (l *LinkPreviews) cached(u string) (previewEntry, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.cache[u]
	if !ok || l.now().After(e.exp) {
		return previewEntry{}, false
	}
	return e, true
}

func (l *LinkPreviews) store(u string, e previewEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.cache) >= previewCacheSize {
		now := l.now()
		for k, old := range l.cache {
			if now.After(old.exp) {
				delete(l.cache, k)
			}
		}
		// Ainda cheio: abre espaço tirando qualquer um.
		for k := range l.cache {
			if len(l.cache) < previewCacheSize {
				break
			}
			delete(l.cache, k)
		}
	}
	l.cache[u] = e
}

func (l *LinkPreviews) fetch(ctx context.Context, rawURL string) (domain.LinkPreview, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return domain.LinkPreview{}, errors.New("link inválido")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return domain.LinkPreview{}, err
	}
	req.Header.Set("User-Agent", previewUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,image/*;q=0.8,*/*;q=0.5")
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en;q=0.6")
	res, err := l.client.Do(req)
	if err != nil {
		return domain.LinkPreview{}, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return domain.LinkPreview{}, fmt.Errorf("status %d", res.StatusCode)
	}
	final := res.Request.URL
	ct := res.Header.Get("Content-Type")
	mt, _, _ := mime.ParseMediaType(ct)
	switch {
	case strings.HasPrefix(mt, "image/"):
		// O link é a própria imagem: ela é a miniatura.
		return domain.LinkPreview{URL: rawURL, Image: final.String(), SiteName: siteName(final)}, nil
	case mt != "" && mt != "text/html" && mt != "application/xhtml+xml":
		return domain.LinkPreview{}, fmt.Errorf("conteúdo %q não é uma página", mt)
	}
	body, err := charset.NewReader(io.LimitReader(res.Body, previewMaxBytes), ct)
	if err != nil {
		return domain.LinkPreview{}, err
	}
	p := parseMeta(body, final)
	p.URL = rawURL
	return p, nil
}

// parseMeta lê as metatags do <head>. base resolve endereços relativos da imagem.
func parseMeta(r io.Reader, base *url.URL) domain.LinkPreview {
	meta := map[string]string{}
	var title, imageSrc string
	z := html.NewTokenizer(r)
	inTitle := false
loop:
	for {
		switch z.Next() {
		case html.ErrorToken:
			break loop
		case html.StartTagToken, html.SelfClosingTagToken:
			name, hasAttr := z.TagName()
			switch string(name) {
			case "body":
				break loop
			case "title":
				inTitle = title == ""
			case "meta":
				if !hasAttr {
					continue
				}
				attrs := tagAttrs(z)
				key := strings.ToLower(firstNonEmpty(attrs["property"], attrs["name"], attrs["itemprop"]))
				if _, seen := meta[key]; key != "" && !seen {
					meta[key] = attrs["content"]
				}
			case "link":
				if hasAttr {
					attrs := tagAttrs(z)
					if strings.EqualFold(attrs["rel"], "image_src") && imageSrc == "" {
						imageSrc = attrs["href"]
					}
				}
			}
		case html.TextToken:
			if inTitle {
				title += string(z.Text())
			}
		case html.EndTagToken:
			name, _ := z.TagName()
			switch string(name) {
			case "title":
				inTitle = false
			case "head":
				break loop
			}
		}
	}
	p := domain.LinkPreview{
		Title:       clip(firstNonEmpty(meta["og:title"], meta["twitter:title"], title), 300),
		Description: clip(firstNonEmpty(meta["og:description"], meta["twitter:description"], meta["description"]), 500),
		SiteName:    clip(firstNonEmpty(meta["og:site_name"], meta["application-name"], siteName(base)), 100),
	}
	img := firstNonEmpty(meta["og:image:secure_url"], meta["og:image"], meta["og:image:url"], meta["twitter:image"], meta["twitter:image:src"], meta["image"], imageSrc)
	if img != "" {
		if iu, err := base.Parse(strings.TrimSpace(img)); err == nil && (iu.Scheme == "http" || iu.Scheme == "https") {
			p.Image = iu.String()
		}
	}
	return p
}

func tagAttrs(z *html.Tokenizer) map[string]string {
	out := map[string]string{}
	for {
		k, v, more := z.TagAttr()
		out[strings.ToLower(string(k))] = string(v)
		if !more {
			return out
		}
	}
}

func siteName(u *url.URL) string {
	return strings.TrimPrefix(u.Hostname(), "www.")
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}

// clip junta espaços em branco e corta o texto em n caracteres, com reticências.
func clip(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return strings.TrimSpace(string(r[:n-1])) + "…"
}
