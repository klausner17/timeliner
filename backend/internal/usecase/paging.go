package usecase

import "fmt"

// PageInput é a página pedida pelo cliente (page começa em 1).
type PageInput struct {
	Page     int
	PageSize int
}

// Page é uma página de resultados e o total que bate com o filtro.
type Page[T any] struct {
	Items    []T
	Page     int
	PageSize int
	Total    int
}

// normalize põe os padrões e valida os limites.
func (p *PageInput) normalize(f fieldErrors, def, max int) {
	if p.Page == 0 {
		p.Page = 1
	}
	if p.PageSize == 0 {
		p.PageSize = def
	}
	f.check(p.Page >= 1, "page", "A página começa em 1.")
	f.check(p.PageSize >= 1 && p.PageSize <= max, "pageSize", fmt.Sprintf("Use de 1 a %d.", max))
}

func (p PageInput) paging() Paging {
	return Paging{Limit: p.PageSize, Offset: (p.Page - 1) * p.PageSize}
}

func pageOf[T any](items []T, p PageInput, total int) Page[T] {
	return Page[T]{Items: items, Page: p.Page, PageSize: p.PageSize, Total: total}
}
