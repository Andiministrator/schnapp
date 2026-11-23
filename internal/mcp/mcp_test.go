// SPDX-License-Identifier: GPL-3.0-or-later

package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

const metaNeu = `"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{},"io.modelcontextprotocol/clientInfo":{"name":"Testclient","version":"1"}}`

func testServer() *Server {
	return &Server{Name: "schnapp", Version: "9.9.9", Werkzeuge: []Werkzeug{
		{Name: "echo", Beschreibung: "gibt zurück", Eingabe: json.RawMessage(`{"type":"object"}`),
			Aufruf: func(_ context.Context, args json.RawMessage, k Kunde) Ergebnis {
				return Ergebnis{Inhalt: []Inhalt{TextInhalt(string(args) + "|" + k.Name)}, Strukturiert: map[string]int{"n": 1}}
			}},
		{Name: "fehler", Aufruf: func(context.Context, json.RawMessage, Kunde) Ergebnis {
			return FehlerErgebnis("nicht erlaubt")
		}},
		{Name: "panik", Aufruf: func(context.Context, json.RawMessage, Kunde) Ergebnis { panic("au") }},
	}}
}

// lauf schickt die Zeilen und liefert die Antworten als dekodierte Objekte.
func lauf(t *testing.T, s *Server, zeilen ...string) []map[string]any {
	t.Helper()
	var aus bytes.Buffer
	if err := s.Bediene(context.Background(), strings.NewReader(strings.Join(zeilen, "\n")+"\n"), &aus); err != nil {
		t.Fatalf("Bediene: %v", err)
	}
	var res []map[string]any
	for _, z := range strings.Split(strings.TrimSuffix(aus.String(), "\n"), "\n") {
		if z == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(z), &m); err != nil {
			t.Fatalf("Antwort kein JSON: %q", z)
		}
		res = append(res, m)
	}
	return res
}

func fehlerCode(t *testing.T, m map[string]any) int {
	t.Helper()
	e, ok := m["error"].(map[string]any)
	if !ok {
		t.Fatalf("kein Fehler: %v", m)
	}
	return int(e["code"].(float64))
}

func TestNeueFormDiscoverListeAufruf(t *testing.T) {
	r := lauf(t, testServer(),
		`{"jsonrpc":"2.0","id":"d","method":"server/discover","params":{`+metaNeu+`}}`,
		`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{`+metaNeu+`}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"echo","arguments":{"a":1},`+metaNeu+`}}`,
	)
	if len(r) != 3 {
		t.Fatalf("%d Antworten, erwartet 3", len(r))
	}
	d := r[0]["result"].(map[string]any)
	if d["resultType"] != "complete" || d["supportedVersions"].([]any)[0] != Version {
		t.Errorf("discover: %v", d)
	}
	if d["_meta"].(map[string]any)[metaServer].(map[string]any)["name"] != "schnapp" {
		t.Errorf("serverInfo fehlt: %v", d)
	}
	l := r[1]["result"].(map[string]any)
	ws := l["tools"].([]any)
	if len(ws) != 3 || ws[0].(map[string]any)["name"] != "echo" || l["ttlMs"] == nil || l["cacheScope"] != "private" {
		t.Errorf("tools/list: %v", l)
	}
	if ws[1].(map[string]any)["inputSchema"] == nil {
		t.Error("inputSchema fehlt, wo Eingabe leer ist")
	}
	c := r[2]["result"].(map[string]any)
	if c["isError"] != false || c["content"].([]any)[0].(map[string]any)["text"] != `{"a":1}|Testclient` {
		t.Errorf("tools/call: %v", c)
	}
	if c["structuredContent"].(map[string]any)["n"] != 1.0 {
		t.Errorf("structuredContent: %v", c)
	}
}

func TestOhneMetaUndOhneInitializeIstParameterfehler(t *testing.T) {
	r := lauf(t, testServer(), `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`)
	if fehlerCode(t, r[0]) != codeParameter || fehlerCode(t, r[1]) != codeParameter {
		t.Errorf("erwartet -32602 zweimal: %v", r)
	}
}

// TestToolsCallPrueftMeta: Auch tools/call verlangt in der neuen Form _meta mit bekannter Version (ohne initialize).
func TestToolsCallPrueftMeta(t *testing.T) {
	r := lauf(t, testServer(),
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"echo"}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"echo","_meta":{"io.modelcontextprotocol/protocolVersion":"1900-01-01","io.modelcontextprotocol/clientCapabilities":{}}}}`,
	)
	if fehlerCode(t, r[0]) != codeParameter || fehlerCode(t, r[1]) != codeVersionFremd {
		t.Errorf("tools/call ohne gültige _meta: %v", r)
	}
}

func TestFremdeVersion(t *testing.T) {
	r := lauf(t, testServer(), `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"1900-01-01","io.modelcontextprotocol/clientCapabilities":{}}}}`)
	if fehlerCode(t, r[0]) != codeVersionFremd {
		t.Fatalf("erwartet -32022: %v", r[0])
	}
	data := r[0]["error"].(map[string]any)["data"].(map[string]any)
	if data["requested"] != "1900-01-01" || len(data["supported"].([]any)) == 0 {
		t.Errorf("data: %v", data)
	}
}

func TestAltform(t *testing.T) {
	r := lauf(t, testServer(),
		`{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"Alt","version":"2"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":1,"method":"ping"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"echo"}}`,
	)
	if len(r) != 3 {
		t.Fatalf("%d Antworten, erwartet 3 (keine auf die Benachrichtigung)", len(r))
	}
	i := r[0]["result"].(map[string]any)
	if i["protocolVersion"] != "2025-06-18" || i["serverInfo"].(map[string]any)["version"] != "9.9.9" {
		t.Errorf("initialize: %v", i)
	}
	if r[2]["result"].(map[string]any)["content"].([]any)[0].(map[string]any)["text"] != "{}|Alt" {
		t.Errorf("Aufruf in der Altform: %v", r[2])
	}
	r = lauf(t, testServer(), `{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":"2099-01-01"}}`)
	if r[0]["result"].(map[string]any)["protocolVersion"] != LetzteAltversion {
		t.Errorf("unbekannte Altversion: %v", r[0])
	}
}

func TestFehlerwege(t *testing.T) {
	lang := `{"jsonrpc":"2.0","id":9,"method":"ping","params":{"x":"` + strings.Repeat("a", MaxZeile) + `"}}`
	r := lauf(t, testServer(),
		`kein json`,
		`[{"jsonrpc":"2.0","id":1,"method":"ping"}]`,
		`{"jsonrpc":"1.0","id":1,"method":"ping"}`,
		`{"jsonrpc":"2.0","id":null,"method":"ping"}`,
		`{"jsonrpc":"2.0","id":1,"method":"gibtsnicht"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"nein",`+metaNeu+`}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":[1],`+metaNeu+`}}`,
		lang,
		`{"jsonrpc":"2.0","id":4,"method":"ping"}`,
	)
	want := []int{codeParse, codeAnfrage, codeAnfrage, codeAnfrage, codeMethode, codeParameter, codeParameter, codeParse}
	if len(r) != len(want)+1 {
		t.Fatalf("%d Antworten, erwartet %d", len(r), len(want)+1)
	}
	for i, c := range want {
		if got := fehlerCode(t, r[i]); got != c {
			t.Errorf("Antwort %d: Code %d, erwartet %d", i, got, c)
		}
	}
	if r[len(want)]["id"] != 4.0 {
		t.Errorf("nach der zu langen Zeile geht es weiter: %v", r[len(want)])
	}
}

func TestAusfuehrungsfehlerUndPanik(t *testing.T) {
	r := lauf(t, testServer(),
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"fehler",`+metaNeu+`}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"panik",`+metaNeu+`}}`,
	)
	for i, m := range r {
		if m["result"].(map[string]any)["isError"] != true {
			t.Errorf("Antwort %d: isError fehlt: %v", i, m)
		}
	}
}

// blockServer hat ein Werkzeug, das bis zum Abbruch wartet und meldet, ob es abgebrochen wurde.
func blockServer(abgebrochen chan<- bool) *Server {
	return &Server{Name: "s", Version: "1", Werkzeuge: []Werkzeug{{Name: "warte",
		Aufruf: func(ctx context.Context, _ json.RawMessage, _ Kunde) Ergebnis {
			select {
			case <-ctx.Done():
				abgebrochen <- true
			case <-time.After(5 * time.Second):
				abgebrochen <- false
			}
			return TextErgebnis("fertig")
		}}}}
}

// TextErgebnis ist nur für Tests.
func TextErgebnis(s string) Ergebnis { return Ergebnis{Inhalt: []Inhalt{TextInhalt(s)}} }

func TestAbbruchWaehrendDesAufrufs(t *testing.T) {
	ab := make(chan bool, 1)
	pr, pw := io.Pipe()
	var aus syncPuffer
	fertig := make(chan error, 1)
	go func() { fertig <- blockServer(ab).Bediene(context.Background(), pr, &aus) }()
	io.WriteString(pw, `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"warte",`+metaNeu+`}}`+"\n")
	io.WriteString(pw, `{"jsonrpc":"2.0","id":8,"method":"tools/list","params":{`+metaNeu+`}}`+"\n")
	io.WriteString(pw, `{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":7}}`+"\n")
	if !<-ab {
		t.Fatal("Werkzeug nicht abgebrochen")
	}
	// Erst nach der Antwort auf 8 schließen: Ein EOF, das eintrifft, bevor der Server das Werkzeug-Ende abholt, beendet
	// ihn vertragsgemäß samt Schlange (Paketkommentar). So lief der Test in der CI unter -race rot (Commit e08a477).
	frist := time.Now().Add(5 * time.Second)
	for !strings.Contains(aus.String(), `"id":8`) && time.Now().Before(frist) {
		time.Sleep(time.Millisecond)
	}
	pw.Close()
	if err := <-fertig; err != nil {
		t.Fatal(err)
	}
	s := aus.String()
	if strings.Contains(s, `"id":7`) {
		t.Errorf("Antwort auf die abgebrochene Anfrage: %s", s)
	}
	if !strings.Contains(s, `"id":8`) {
		t.Errorf("wartende Anfrage nicht beantwortet: %s", s)
	}
}

func TestEOFWaehrendDesAufrufsBrichtAb(t *testing.T) {
	ab := make(chan bool, 1)
	var aus syncPuffer
	err := blockServer(ab).Bediene(context.Background(),
		strings.NewReader(`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"warte",`+metaNeu+`}}`+"\n"), &aus)
	if err != nil {
		t.Fatal(err)
	}
	if !<-ab {
		t.Fatal("EOF bricht das Werkzeug nicht ab")
	}
}

func TestSchreibfehlerBeendet(t *testing.T) {
	err := testServer().Bediene(context.Background(), strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`+"\n"), kaputt{})
	if err == nil || !strings.HasPrefix(err.Error(), "mcp: ") {
		t.Fatalf("Fehler %v, erwartet mit Präfix mcp:", err)
	}
}

type kaputt struct{}

func (kaputt) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

// syncPuffer ist ein nebenläufig lesbarer Puffer. Der Mutex braucht keine Anlage: Ein träge angelegtes Schloss war
// selbst ein Datenrennen, sobald Test und Server gleichzeitig zugriffen.
type syncPuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (p *syncPuffer) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.buf.Write(b)
}

func (p *syncPuffer) String() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.buf.String()
}
