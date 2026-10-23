package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/klausner17/timeliner/backend/internal/controller"
	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/platform"
	"github.com/klausner17/timeliner/backend/internal/repository/memory"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

type api struct {
	t   *testing.T
	h   http.Handler
	log *bytes.Buffer
}

func newAPI(t *testing.T) *api {
	d := memory.New()
	d.Hasher, d.Clock, d.IDs = cheapHasher{}, platform.SystemClock{}, platform.RandomIDs{}
	return newAPIWith(t, controller.UseCasesFrom(d))
}

func newAPIWith(t *testing.T, uc controller.UseCases) *api {
	buf := &bytes.Buffer{}
	return &api{t, controller.New(uc, slog.New(slog.NewTextHandler(buf, nil)), nil), buf}
}

type cheapHasher struct{}

func (cheapHasher) Hash(p string) (string, error) { return "h:" + p, nil }
func (cheapHasher) Compare(h, p string) bool      { return h == "h:"+p }

type resp struct {
	Code int
	Body map[string]any
	Raw  string
}

func (a *api) do(method, path, token string, body any) resp {
	a.t.Helper()
	var r io.Reader
	if s, ok := body.(string); ok {
		r = strings.NewReader(s)
	} else if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, r)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	a.h.ServeHTTP(rec, req)
	out := resp{Code: rec.Code, Raw: rec.Body.String()}
	_ = json.Unmarshal(rec.Body.Bytes(), &out.Body)
	return out
}

func (a *api) want(r resp, code int) resp {
	a.t.Helper()
	if r.Code != code {
		a.t.Fatalf("status = %d, queria %d: %s", r.Code, code, r.Raw)
	}
	return r
}

func (a *api) register(handle string) string {
	a.t.Helper()
	r := a.want(a.do("POST", "/api/auth/register", "", map[string]any{
		"handle": handle, "name": handle, "email": handle + "@exemplo.com", "password": "senha-forte",
	}), 201)
	return r.Body["token"].(string)
}

func errOf(r resp) map[string]any { return r.Body["error"].(map[string]any) }

func TestHealthAndUnknownRoute(t *testing.T) {
	a := newAPI(t)
	a.want(a.do("GET", "/api/health", "", nil), 200)
	r := a.want(a.do("GET", "/api/nada", "", nil), 404)
	if errOf(r)["code"] != "not_found" {
		t.Fatalf("corpo = %s", r.Raw)
	}
}

func TestAuthFlow(t *testing.T) {
	a := newAPI(t)
	token := a.register("ana")
	me := a.want(a.do("GET", "/api/me", token, nil), 200)
	if me.Body["handle"] != "ana" || me.Body["tint"] != "neutral" || me.Body["password_hash"] != nil {
		t.Fatalf("me = %s", me.Raw)
	}
	a.want(a.do("GET", "/api/me", "", nil), 401)
	r := a.want(a.do("GET", "/api/timelines", "token-velho", nil), 401)
	if errOf(r)["message"] != "Sua sessão expirou. Entre de novo." {
		t.Fatalf("mensagem = %s", r.Raw)
	}
	login := a.want(a.do("POST", "/api/auth/login", "", map[string]any{"email": "ana@exemplo.com", "password": "senha-forte"}), 200)
	if login.Body["token"] == "" {
		t.Fatal("login sem token")
	}
	a.want(a.do("PATCH", "/api/me", token, map[string]any{"bio": "Oi"}), 200)
}

func TestValidationErrorShape(t *testing.T) {
	a := newAPI(t)
	token := a.register("ana")
	r := a.want(a.do("POST", "/api/timelines", token, map[string]any{
		"title": "", "category": "Futebol", "events": []map[string]any{{"year": 1, "title": ""}},
	}), 400)
	e := errOf(r)
	fields := e["fields"].(map[string]any)
	if e["code"] != "invalid" || e["message"] != "Confira os campos destacados." || fields["title"] == nil || fields["category"] == nil || fields["events[0].title"] == nil {
		t.Fatalf("erro = %s", r.Raw)
	}
	r = a.want(a.do("POST", "/api/timelines", token, map[string]any{"title": "x", "category": "História", "events": []map[string]any{{"title": "sem ano"}}}), 400)
	if errOf(r)["fields"].(map[string]any)["events[0].year"] != "Informe o ano." {
		t.Fatalf("ano obrigatório = %s", r.Raw)
	}

	r = a.want(a.do("POST", "/api/timelines", token, `{"title": 5}`), 400)
	if errOf(r)["fields"].(map[string]any)["title"] != "Valor em formato inválido." {
		t.Fatalf("tipo errado = %s", r.Raw)
	}
	r = a.want(a.do("POST", "/api/timelines", token, `{`), 400)
	if errOf(r)["message"] != "Não foi possível ler os dados enviados." {
		t.Fatalf("json quebrado = %s", r.Raw)
	}
	a.want(a.do("GET", "/api/timelines?page=x", "", nil), 400)
}

func TestInternalErrorHidesDetails(t *testing.T) {
	d := memory.New()
	uc := controller.UseCasesFrom(d)
	uc.ListTags = brokenTags{}
	a := newAPIWith(t, uc)
	r := a.want(a.do("GET", "/api/tags", "", nil), 500)
	if strings.Contains(r.Raw, "senha do banco") || errOf(r)["code"] != "internal" {
		t.Fatalf("vazou detalhe: %s", r.Raw)
	}
	if !strings.Contains(a.log.String(), "senha do banco") {
		t.Fatal("o detalhe devia ir para o log")
	}
}

func TestTimelineSuggestionCompareFlow(t *testing.T) {
	a := newAPI(t)
	ana, beto := a.register("ana"), a.register("beto")

	tl := a.want(a.do("POST", "/api/timelines", ana, map[string]any{
		"title": "História do Brasil", "category": "História", "tags": []string{"Brasil"},
		"eras":   []map[string]any{{"name": "Império", "start": 1822, "end": 1889}},
		"events": []map[string]any{{"year": 1822, "title": "Independência", "era": "Império"}, {"year": 0, "title": "Ano zero"}},
	}), 201)
	if tl.Body["slug"] != "historia-do-brasil" || len(tl.Body["events"].([]any)) != 2 || tl.Body["eras"].([]any)[0].(map[string]any)["end"] != 1889.0 {
		t.Fatalf("timeline = %s", tl.Raw)
	}
	a.want(a.do("POST", "/api/timelines", ana, map[string]any{"title": "Escândalos", "category": "Política", "events": []map[string]any{{"year": 1992, "title": "Collor"}}}), 201)

	list := a.want(a.do("GET", "/api/timelines?cat=Hist%C3%B3ria", "", nil), 200)
	if list.Body["total"] != 1.0 {
		t.Fatalf("lista = %s", list.Raw)
	}
	a.want(a.do("GET", "/api/tags", "", nil), 200)

	evID := tl.Body["events"].([]any)[1].(map[string]any)["id"].(string)
	upd := a.want(a.do("PATCH", "/api/timelines/historia-do-brasil/events/"+evID, ana, map[string]any{"year": 1500, "title": "Chegada", "date": "22 de abril"}), 200)
	if upd.Body["date"] != "22 de abril" {
		t.Fatalf("evento = %s", upd.Raw)
	}
	upd = a.want(a.do("PATCH", "/api/timelines/historia-do-brasil/events/"+evID, ana, map[string]any{"date": nil}), 200)
	if upd.Body["date"] != nil || upd.Body["title"] != "Chegada" {
		t.Fatalf("date null devia limpar só a data: %s", upd.Raw)
	}
	a.want(a.do("PATCH", "/api/timelines/historia-do-brasil/events/"+evID, beto, map[string]any{"title": "x"}), 403)

	sug := a.want(a.do("POST", "/api/timelines/historia-do-brasil/suggestions", beto, map[string]any{
		"kind": "new_event", "event": map[string]any{"year": 1888, "title": "Lei Áurea", "era": "Império"}, "note": "faltou",
	}), 201)
	id := sug.Body["id"].(string)
	if sug.Body["status"] != "pendente" || sug.Body["by"] != "beto" || sug.Body["timeline"].(map[string]any)["slug"] != "historia-do-brasil" {
		t.Fatalf("sugestão = %s", sug.Raw)
	}
	a.want(a.do("POST", "/api/suggestions/"+id+"/review", beto, map[string]any{"decision": "approve"}), 403)
	rev := a.want(a.do("POST", "/api/suggestions/"+id+"/review", ana, map[string]any{"decision": "approve"}), 200)
	if rev.Body["status"] != "aprovada" || rev.Body["applied"] == nil {
		t.Fatalf("revisão = %s", rev.Raw)
	}
	a.want(a.do("POST", "/api/suggestions/"+id+"/review", ana, map[string]any{"decision": "reject"}), 409)
	a.want(a.do("POST", "/api/suggestions/"+id+"/reopen", ana, nil), 200)
	pend := a.want(a.do("GET", "/api/timelines/historia-do-brasil/suggestions?status=pendente", "", nil), 200)
	if len(pend.Body["items"].([]any)) != 1 {
		t.Fatalf("pendentes = %s", pend.Raw)
	}

	cmp := a.want(a.do("GET", "/api/compare?slugs=historia-do-brasil,escandalos&start=1800", "", nil), 200)
	lanes := cmp.Body["lanes"].([]any)
	if cmp.Body["start"] != 1800.0 || cmp.Body["end"] != 1992.0 || lanes[1].(map[string]any)["color"] != "b" || lanes[0].(map[string]any)["id"] != "historia-do-brasil" {
		t.Fatalf("comparação = %s", cmp.Raw)
	}
	a.want(a.do("GET", "/api/compare?slugs=historia-do-brasil", "", nil), 400)

	saved := a.want(a.do("POST", "/api/me/comparisons", beto, map[string]any{"timelines": []string{"historia-do-brasil", "escandalos"}, "start": 1800, "end": 2000}), 201)
	all := a.want(a.do("GET", "/api/me/comparisons", beto, nil), 200)
	if len(all.Body["items"].([]any)) != 1 {
		t.Fatalf("salvas = %s", all.Raw)
	}
	a.want(a.do("DELETE", "/api/me/comparisons/"+saved.Body["id"].(string), beto, nil), 204)

	a.want(a.do("POST", "/api/timelines/historia-do-brasil/follow", beto, nil), 204)
	a.want(a.do("POST", "/api/users/ana/follow", beto, nil), 204)
	prof := a.want(a.do("GET", "/api/users/ana", beto, nil), 200)
	if prof.Body["following"] != true || prof.Body["stats"].(map[string]any)["followers"] != 1.0 {
		t.Fatalf("perfil = %s", prof.Raw)
	}
	fol := a.want(a.do("GET", "/api/users/beto/following", "", nil), 200)
	if len(fol.Body["users"].([]any)) != 1 || len(fol.Body["timelines"].([]any)) != 1 {
		t.Fatalf("seguindo = %s", fol.Raw)
	}
	a.want(a.do("GET", "/api/users/beto/contributions", "", nil), 200)

	a.want(a.do("DELETE", "/api/timelines/escandalos", beto, nil), 403)
	a.want(a.do("DELETE", "/api/timelines/escandalos", ana, nil), 204)
	a.want(a.do("GET", "/api/timelines/escandalos", ana, nil), 404)
}

func TestEventsPagingAndSummary(t *testing.T) {
	a := newAPI(t)
	ana := a.register("ana")
	events := make([]map[string]any, 200)
	for i := range events {
		events[i] = map[string]any{"year": 1800 + i, "title": fmt.Sprintf("Evento %d", i)}
	}
	tl := a.want(a.do("POST", "/api/timelines", ana, map[string]any{"title": "Grande", "category": "História", "events": events}), 201)
	if len(tl.Body["events"].([]any)) != 100 || tl.Body["eventsNextCursor"] == nil {
		t.Fatalf("detalhe = %d eventos, cursor %v", len(tl.Body["events"].([]any)), tl.Body["eventsNextCursor"])
	}

	page := a.want(a.do("GET", "/api/timelines/grande/events?limit=150&cursor="+tl.Body["eventsNextCursor"].(string), "", nil), 200)
	if items := page.Body["items"].([]any); len(items) != 100 || items[0].(map[string]any)["year"] != 1900.0 || page.Body["nextCursor"] != nil {
		t.Fatalf("página = %d itens, nextCursor %v", len(items), page.Body["nextCursor"])
	}
	bad := a.want(a.do("GET", "/api/timelines/grande/events?cursor=xyz", "", nil), 400)
	if bad.Body["error"].(map[string]any)["fields"].(map[string]any)["cursor"] == nil {
		t.Fatalf("cursor inválido = %s", bad.Raw)
	}

	sum := a.want(a.do("GET", "/api/timelines/grande/summary?buckets=4", "", nil), 200)
	buckets := sum.Body["buckets"].([]any)
	if sum.Body["mode"] != "faixas" || sum.Body["total"] != 200.0 || len(buckets) != 4 || len(sum.Body["events"].([]any)) != 0 {
		t.Fatalf("resumo = %s", sum.Raw)
	}
	if b := buckets[0].(map[string]any); b["from"] != 1800.0 || b["to"] != 1849.0 || b["count"] != 50.0 || len(b["sample"].([]any)) != 3 {
		t.Fatalf("faixa = %v", b)
	}
	zoom := a.want(a.do("GET", "/api/timelines/grande/summary?from=1810&to=1819", "", nil), 200)
	if zoom.Body["mode"] != "eventos" || len(zoom.Body["events"].([]any)) != 10 {
		t.Fatalf("zoom = %s", zoom.Raw)
	}
	a.want(a.do("GET", "/api/timelines/grande/summary?buckets=abc", "", nil), 400)

	a.want(a.do("POST", "/api/timelines", ana, map[string]any{"title": "Outra", "category": "História", "events": []map[string]any{{"year": 1810, "title": "x"}}}), 201)
	cmp := a.want(a.do("GET", "/api/compare?slugs=grande,outra&maxEvents=50&buckets=10", "", nil), 200)
	lanes := cmp.Body["lanes"].([]any)
	if lanes[0].(map[string]any)["mode"] != "faixas" || lanes[1].(map[string]any)["mode"] != "eventos" || cmp.Body["overlapsTotal"] != 1.0 {
		t.Fatalf("comparação = %s", cmp.Raw)
	}
}

type brokenTags struct{}

func (brokenTags) Execute(context.Context, usecase.Empty) ([]domain.TagCount, error) {
	return nil, fmt.Errorf("conectar: %w", errors.New("senha do banco errada"))
}
