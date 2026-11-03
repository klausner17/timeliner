package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/klausner17/timeliner/backend/internal/domain"
)

// Resend manda e-mails pela API do Resend (https://resend.com/docs/api-reference/emails/send-email).
type Resend struct {
	APIKey string
	// From é o remetente, num domínio verificado no Resend: "Timeliner <ola@seudominio.com>".
	From string
	// Endpoint troca o endereço da API (testes); vazio usa o do Resend.
	Endpoint string
	Client   *http.Client
}

func (r Resend) Send(ctx context.Context, e domain.Email) error {
	body, err := json.Marshal(map[string]any{
		"from": r.From, "to": []string{e.To}, "subject": e.Subject, "text": e.Text, "html": e.HTML,
	})
	if err != nil {
		return err
	}
	endpoint := r.Endpoint
	if endpoint == "" {
		endpoint = "https://api.resend.com/emails"
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+r.APIKey)
	req.Header.Set("Content-Type", "application/json")
	client := r.Client
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("resend: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		msg, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return fmt.Errorf("resend: status %d: %s", res.StatusCode, msg)
	}
	return nil
}

// LogMailer não manda nada: escreve o e-mail no log. Serve para desenvolvimento,
// quando não há chave do Resend; o link aparece no log do servidor.
type LogMailer struct{ Log *slog.Logger }

func (m LogMailer) Send(ctx context.Context, e domain.Email) error {
	m.Log.InfoContext(ctx, "e-mail não enviado (sem RESEND_API_KEY)", "to", e.To, "subject", e.Subject, "text", e.Text)
	return nil
}
