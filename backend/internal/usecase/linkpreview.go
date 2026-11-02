package usecase

import (
	"context"
	"sync"

	"github.com/klausner17/timeliner/backend/internal/domain"
)

// PreviewEventLinks devolve a prévia de cada link do evento: a fonte primeiro, depois
// os links a mais, na ordem em que foram escritos. Só busca endereços que já estão
// num evento visível, para a API não virar um proxy de qualquer URL.
type PreviewEventLinks struct{ d *Deps }

func NewPreviewEventLinks(d *Deps) *PreviewEventLinks { return &PreviewEventLinks{d} }

func (uc *PreviewEventLinks) Execute(ctx context.Context, in EventRefInput) ([]domain.LinkPreview, error) {
	t, err := visibleTimeline(ctx, uc.d, in.Slug, in.ActorID)
	if err != nil {
		return nil, err
	}
	e, err := eventOf(ctx, uc.d, t, in.EventID)
	if err != nil {
		return nil, err
	}
	urls := eventURLs(e.EventData)
	out := make([]domain.LinkPreview, len(urls))
	var wg sync.WaitGroup
	for i, u := range urls {
		out[i] = domain.LinkPreview{URL: u}
		if uc.d.Previews == nil {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if p, err := uc.d.Previews.Preview(ctx, u); err == nil {
				p.URL = u
				out[i] = p
			}
		}()
	}
	wg.Wait()
	return out, nil
}

// eventURLs são a fonte e os links do evento, sem repetir endereço.
func eventURLs(e domain.EventData) []string {
	seen := map[string]bool{}
	out := []string{}
	add := func(u string) {
		if u != "" && !seen[u] {
			seen[u] = true
			out = append(out, u)
		}
	}
	if e.SourceURL != nil {
		add(*e.SourceURL)
	}
	for _, l := range e.Links {
		add(l.URL)
	}
	return out
}
