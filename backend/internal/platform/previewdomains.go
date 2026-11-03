package platform

import "strings"

// DefaultPreviewDomains são os sites cuja prévia (título, resumo, miniatura) o servidor
// busca. Vale o domínio e qualquer subdomínio: "gov.br" cobre "www.ibge.gov.br". Links
// de outros sites aparecem só com o endereço. Inclui os domínios das imagens
// (wikimedia.org, glbimg.com, ytimg.com...), porque a miniatura também precisa estar aqui.
var DefaultPreviewDomains = []string{
	// Enciclopédias, acervos e pesquisa
	"wikipedia.org", "wikimedia.org", "wikidata.org", "britannica.com", "archive.org",
	"scielo.br", "scielo.org", "fgv.br", "usp.br", "unicamp.br",
	// Governo, Congresso e Justiça
	"gov.br", "leg.br", "jus.br", "mp.br",
	// Imprensa brasileira
	"globo.com", "glbimg.com", "uol.com.br", "estadao.com.br", "abril.com.br", "ebc.com.br",
	"cnnbrasil.com.br", "terra.com.br", "poder360.com.br", "metropoles.com", "nexojornal.com.br",
	"cartacapital.com.br", "correiobraziliense.com.br", "band.uol.com.br",
	// Imprensa internacional
	"bbc.com", "bbc.co.uk", "bbci.co.uk", "reuters.com", "apnews.com", "elpais.com",
	"nytimes.com", "nyt.com", "theguardian.com", "guim.co.uk",
	// Vídeo
	"youtube.com", "youtu.be", "ytimg.com", "vimeo.com", "vimeocdn.com",
}

// PreviewDomains junta a lista padrão com os domínios extras separados por vírgula
// (variável LINK_PREVIEW_EXTRA_DOMAINS).
func PreviewDomains(extra string) []string {
	out := append([]string{}, DefaultPreviewDomains...)
	for _, d := range strings.Split(extra, ",") {
		if d = strings.Trim(strings.ToLower(strings.TrimSpace(d)), "."); d != "" {
			out = append(out, d)
		}
	}
	return out
}

// domainSet diz se um host está na lista, ele mesmo ou como subdomínio. Vazio aceita todos.
type domainSet map[string]bool

func newDomainSet(domains []string) domainSet {
	if domains == nil {
		return nil
	}
	s := domainSet{}
	for _, d := range domains {
		s[strings.ToLower(d)] = true
	}
	return s
}

func (s domainSet) allows(host string) bool {
	if s == nil {
		return true
	}
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	for host != "" {
		if s[host] {
			return true
		}
		i := strings.IndexByte(host, '.')
		if i < 0 {
			return false
		}
		host = host[i+1:]
	}
	return false
}
