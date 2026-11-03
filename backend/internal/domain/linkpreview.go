package domain

// LinkPreview é o cartão de um link, como os apps de conversa mostram: título, resumo,
// miniatura e site, lidos das metatags (Open Graph) da página. Só URL é garantido; a
// miniatura é o endereço da imagem no site de origem, nada é guardado aqui.
type LinkPreview struct {
	URL         string
	Title       string
	Description string
	Image       string
	SiteName    string
}
