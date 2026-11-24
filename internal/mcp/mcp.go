// SPDX-License-Identifier: GPL-3.0-or-later

// Package mcp ist die kleine Protokollschicht des MCP-Servers `schnapp mcp` (ADR-0030 Nachtrag 2): JSON-RPC 2.0
// zeilenweise über stdin/stdout, nur Werkzeuge (tools/list, tools/call). Es kennt weder Gio noch Schnapp; die
// Werkzeuge kommen als Liste von außen.
//
// Zwei Epochen der Spezifikation (knowledge/sources/mcp-2026-07-28.md):
//   - 2026-07-28: kein Handshake, jede Anfrage trägt in params._meta die Protokollversion und die Fähigkeiten des
//     Clients; `server/discover` beschreibt den Server.
//   - Altform (2025-11-25 und früher): `initialize` wählt sie, danach gelten tools/* ohne _meta.
//
// Anfragen laufen nacheinander. Während ein Werkzeug läuft, liest der Server weiter: `notifications/cancelled` für
// die laufende Anfrage bricht sie ab (dann ohne Antwort), EOF auf stdin bricht sie ab und beendet den Server, andere
// Nachrichten warten in einer begrenzten Schlange (MaxSchlange, MaxSchlangeBytes). Nach dem Ende von ctx startet
// nichts mehr aus der Schlange (ADR-0030 Nachtrag 4).
package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/Andiministrator/schnapp/internal/errprefix"
)

// Protokollversionen. Die neue Form zuerst; die Altform antwortet mit der angefragten Version, wenn sie bekannt ist,
// sonst mit LetzteAltversion.
const (
	Version          = "2026-07-28"
	LetzteAltversion = "2025-11-25"
)

var (
	neueVersionen = []string{Version}
	altVersionen  = []string{"2025-11-25", "2025-06-18", "2025-03-26", "2024-11-05"}
)

// MaxZeile ist die längste gelesene Zeile; eine längere gilt als unlesbar (-32700), der Rest bis zum Zeilenende wird
// verworfen.
const MaxZeile = 1 << 20

// Grenzen der Warteschlange, solange ein Werkzeug läuft (ADR-0030 Nachtrag 4 Punkt 7, F-468): darüber bekommt eine
// Anfrage -32600 und wird verworfen, eine Benachrichtigung fällt still weg.
const (
	MaxSchlange      = 64
	MaxSchlangeBytes = 8 << 20
)

// Schlüssel in params._meta (2026-07-28).
const (
	metaVersion     = "io.modelcontextprotocol/protocolVersion"
	metaFaehigkeit  = "io.modelcontextprotocol/clientCapabilities"
	metaClient      = "io.modelcontextprotocol/clientInfo"
	metaServer      = "io.modelcontextprotocol/serverInfo"
	listenGueltigMs = 3600000
)

// JSON-RPC- und MCP-Fehlercodes.
const (
	codeParse        = -32700
	codeAnfrage      = -32600
	codeMethode      = -32601
	codeParameter    = -32602
	codeIntern       = -32603
	codeVersionFremd = -32022
)

// Werkzeug ist ein Werkzeug des Servers. Eingabe ist das JSON-Schema der Argumente (ein Objekt, nie null).
type Werkzeug struct {
	Name         string
	Titel        string
	Beschreibung string
	Eingabe      json.RawMessage
	// Aufruf führt das Werkzeug aus. args ist das Objekt `arguments` ({} wenn es fehlt). ctx endet, wenn der Client
	// die Anfrage abbricht oder stdin schließt.
	Aufruf func(ctx context.Context, args json.RawMessage, k Kunde) Ergebnis
}

// Kunde ist, was der Client über sich sagt (clientInfo). Ungeprüft: nie für eine Sicherheitsentscheidung nutzen.
type Kunde struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Inhalt ist ein Eintrag in `content`: Text oder Bild.
type Inhalt struct {
	Typ   string `json:"type"`
	Text  string `json:"text,omitempty"`
	Daten string `json:"data,omitempty"`     // Base64
	MIME  string `json:"mimeType,omitempty"` // bei Bildern
}

// TextInhalt und BildInhalt bauen die beiden Inhaltsarten.
func TextInhalt(s string) Inhalt { return Inhalt{Typ: "text", Text: s} }

// BildInhalt nimmt die Daten schon als Base64.
func BildInhalt(base64, mime string) Inhalt { return Inhalt{Typ: "image", Daten: base64, MIME: mime} }

// Ergebnis eines Werkzeugs. Fehler markiert einen Ausführungsfehler (isError), der Text darin geht an das Modell.
type Ergebnis struct {
	Inhalt       []Inhalt
	Strukturiert any // structuredContent; nil = keiner
	Fehler       bool
}

// FehlerErgebnis ist ein Ausführungsfehler mit einem Satz für das Modell.
func FehlerErgebnis(text string) Ergebnis {
	return Ergebnis{Inhalt: []Inhalt{TextInhalt(text)}, Fehler: true}
}

// Server bedient einen Client über ein Paar aus Leser und Schreiber.
type Server struct {
	Name      string
	Version   string
	Anleitung string // instructions; leer = keine
	Werkzeuge []Werkzeug
	// Log nimmt Meldungen für stderr auf (nie stdout); nil = keine.
	Log io.Writer

	alt   bool  // Altform: initialize gesehen
	kunde Kunde // aus initialize (Altform)
}

type nachricht struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Methode string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	// Result und Error trägt nur eine Antwort des Clients; sie bekommt keine Antwort (F-470).
	Result json.RawMessage `json:"result,omitempty"`
	Error  json.RawMessage `json:"error,omitempty"`
}

type fehlerObjekt struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type antwort struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *fehlerObjekt   `json:"error,omitempty"`
}

var nullID = json.RawMessage("null")

// zeile ist eine gelesene Zeile; zuLang heißt, sie war länger als MaxZeile.
type zeile struct {
	daten  []byte
	zuLang bool
}

// liesZeilen liest Zeilen aus r in den Kanal, bis EOF oder ein Lesefehler; dann schließt es den Kanal. ende bricht
// das Senden ab (der Server ist fertig).
func liesZeilen(r io.Reader, aus chan<- zeile, ende <-chan struct{}) {
	defer close(aus)
	br := bufio.NewReaderSize(r, 64<<10)
	for {
		var buf []byte
		zuLang := false
		for {
			teil, err := br.ReadSlice('\n')
			if !zuLang {
				if len(buf)+len(teil) > MaxZeile+1 { // +1 für das Zeilenende
					zuLang, buf = true, nil
				} else {
					buf = append(buf, teil...)
				}
			}
			if errors.Is(err, bufio.ErrBufferFull) {
				continue
			}
			if err != nil {
				if len(buf) > 0 || zuLang {
					select {
					case aus <- zeile{daten: bytes.TrimSpace(buf), zuLang: zuLang}:
					case <-ende:
					}
				}
				return
			}
			break
		}
		select {
		case aus <- zeile{daten: bytes.TrimSpace(buf), zuLang: zuLang}:
		case <-ende:
			return
		}
	}
}

// Bediene liest Anfragen aus r und schreibt Antworten nach w, bis r endet oder ctx abläuft. Ein Schreibfehler beendet
// den Server mit diesem Fehler; EOF ist das normale Ende (nil).
func (s *Server) Bediene(ctx context.Context, r io.Reader, w io.Writer) (fehler error) {
	defer errprefix.Defer("mcp", &fehler)
	zeilen := make(chan zeile)
	ende := make(chan struct{})
	defer close(ende)
	go liesZeilen(r, zeilen, ende)

	var schlange schlange
	for {
		// Nach einem Abbruch (Signal) startet nichts mehr aus der Schlange (ADR-0030 Nachtrag 4 Punkt 6, F-467).
		if ctx.Err() != nil {
			return nil
		}
		var z zeile
		if schlange.laenge() > 0 {
			z = schlange.nimm()
		} else {
			select {
			case <-ctx.Done():
				return nil
			case neu, ok := <-zeilen:
				if !ok {
					return nil
				}
				z = neu
			}
		}
		if len(z.daten) == 0 && !z.zuLang {
			continue
		}
		n, a := s.lies(z)
		if a != nil {
			if err := schreibe(w, *a); err != nil {
				return err
			}
			continue
		}
		if n.Methode == "" {
			continue // Antwort des Clients: keine Antwort darauf (F-470)
		}
		if n.Methode == "tools/call" && len(n.ID) > 0 {
			res, abgebrochen, eof, err := s.rufeAuf(ctx, n, zeilen, &schlange, w)
			if err == nil && !abgebrochen {
				err = schreibe(w, res)
			}
			if err != nil {
				return err
			}
			if eof {
				return nil
			}
			continue
		}
		if res, ok := s.behandle(n); ok {
			if err := schreibe(w, res); err != nil {
				return err
			}
		}
	}
}

// lies zerlegt eine Zeile. Ist sie keine brauchbare Nachricht, kommt die Fehlerantwort zurück (bei Benachrichtigungen
// ohne gültige Form nil, nil: Darauf wird nie geantwortet).
func (s *Server) lies(z zeile) (nachricht, *antwort) {
	if z.zuLang {
		s.log("Zeile länger als %d Bytes verworfen", MaxZeile)
		return nachricht{}, fehlerAntwort(nullID, codeParse, "Parse error: line too long", nil)
	}
	if !json.Valid(z.daten) {
		return nachricht{}, fehlerAntwort(nullID, codeParse, "Parse error", nil)
	}
	if z.daten[0] == '[' {
		return nachricht{}, fehlerAntwort(nullID, codeAnfrage, "Invalid Request: batches are not supported", nil)
	}
	var n nachricht
	if z.daten[0] != '{' || json.Unmarshal(z.daten, &n) != nil {
		// Gültiges JSON, aber kein Objekt oder mit falschen Typen: keine Anfrage (F-470).
		return nachricht{}, fehlerAntwort(nullID, codeAnfrage, "Invalid Request", nil)
	}
	id := n.ID
	if len(id) == 0 {
		id = nullID
	}
	if n.Methode == "" && len(n.ID) > 0 && (len(n.Result) > 0 || len(n.Error) > 0) {
		s.log("Antwort des Clients ohne Anfrage verworfen")
		return nachricht{}, nil // Antwort des Clients (F-470): Bediene antwortet nicht
	}
	if n.JSONRPC != "2.0" || n.Methode == "" || !gueltigeID(n.ID) {
		if len(n.ID) == 0 && n.JSONRPC == "2.0" && n.Methode != "" {
			return n, nil // unpassende Benachrichtigung: still
		}
		return nachricht{}, fehlerAntwort(id, codeAnfrage, "Invalid Request", nil)
	}
	return n, nil
}

// gueltigeID: fehlend (Benachrichtigung), Zahl oder Zeichenkette; null ist im MCP nicht erlaubt.
func gueltigeID(id json.RawMessage) bool {
	if len(id) == 0 {
		return true
	}
	switch id[0] {
	case '"':
		var s string
		return json.Unmarshal(id, &s) == nil
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		var f float64
		return json.Unmarshal(id, &f) == nil
	}
	return false
}

// behandle beantwortet alles außer tools/call. ok ist false bei Benachrichtigungen (keine Antwort).
func (s *Server) behandle(n nachricht) (antwort, bool) {
	if len(n.ID) == 0 {
		return antwort{}, false // notifications/initialized, notifications/cancelled ohne laufende Anfrage, Unbekanntes
	}
	switch n.Methode {
	case "initialize":
		return s.initialisiere(n), true
	case "ping":
		return ergebnisAntwort(n.ID, map[string]any{}), true
	case "server/discover":
		if f := s.pruefeMeta(n); f != nil {
			return *f, true
		}
		return ergebnisAntwort(n.ID, s.entdecken()), true
	case "tools/list":
		if f := s.pruefeMeta(n); f != nil {
			return *f, true
		}
		return ergebnisAntwort(n.ID, s.liste()), true
	}
	return *fehlerAntwort(n.ID, codeMethode, "Method not found: "+n.Methode, nil), true
}

func (s *Server) initialisiere(n nachricht) antwort {
	var p struct {
		Version string `json:"protocolVersion"`
		Kunde   Kunde  `json:"clientInfo"`
	}
	if len(n.Params) > 0 {
		if err := json.Unmarshal(n.Params, &p); err != nil {
			return *fehlerAntwort(n.ID, codeParameter, "Invalid params", nil)
		}
	}
	v := LetzteAltversion
	if slices.Contains(altVersionen, p.Version) {
		v = p.Version
	}
	s.alt, s.kunde = true, p.Kunde
	r := map[string]any{
		"protocolVersion": v,
		"capabilities":    map[string]any{"tools": map[string]any{}},
		"serverInfo":      Kunde{Name: s.Name, Version: s.Version},
	}
	if s.Anleitung != "" {
		r["instructions"] = s.Anleitung
	}
	return ergebnisAntwort(n.ID, r)
}

func (s *Server) entdecken() map[string]any {
	r := map[string]any{
		"supportedVersions": slices.Concat(neueVersionen, altVersionen),
		"capabilities":      map[string]any{"tools": map[string]any{}},
		"_meta":             map[string]any{metaServer: Kunde{Name: s.Name, Version: s.Version}},
		"ttlMs":             listenGueltigMs,
		"cacheScope":        "private",
	}
	if s.Anleitung != "" {
		r["instructions"] = s.Anleitung
	}
	return r
}

type werkzeugJSON struct {
	Name         string          `json:"name"`
	Titel        string          `json:"title,omitempty"`
	Beschreibung string          `json:"description,omitempty"`
	Eingabe      json.RawMessage `json:"inputSchema"`
}

func (s *Server) liste() map[string]any {
	ws := make([]werkzeugJSON, 0, len(s.Werkzeuge))
	for _, w := range s.Werkzeuge {
		e := w.Eingabe
		if len(e) == 0 {
			e = json.RawMessage(`{"type":"object","additionalProperties":false}`)
		}
		ws = append(ws, werkzeugJSON{Name: w.Name, Titel: w.Titel, Beschreibung: w.Beschreibung, Eingabe: e})
	}
	return map[string]any{"tools": ws, "ttlMs": listenGueltigMs, "cacheScope": "private"}
}

// meta liest params._meta; ok ist false, wenn params kein Objekt ist.
func meta(params json.RawMessage) (m map[string]json.RawMessage, ok bool) {
	if len(params) == 0 {
		return nil, true
	}
	var p struct {
		Meta map[string]json.RawMessage `json:"_meta"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, false
	}
	return p.Meta, true
}

// pruefeMeta prüft die Pflichtfelder der neuen Form. In der Altform (nach initialize) dürfen sie fehlen; sind sie da,
// gelten sie (ein Client darf neu ansetzen).
func (s *Server) pruefeMeta(n nachricht) *antwort {
	m, ok := meta(n.Params)
	if !ok {
		return fehlerAntwort(n.ID, codeParameter, "Invalid params", nil)
	}
	roh, hatVersion := m[metaVersion]
	if !hatVersion {
		if s.alt {
			return nil
		}
		return fehlerAntwort(n.ID, codeParameter, "Invalid params: missing _meta."+metaVersion, nil)
	}
	var v string
	if err := json.Unmarshal(roh, &v); err != nil {
		return fehlerAntwort(n.ID, codeParameter, "Invalid params: "+metaVersion+" must be a string", nil)
	}
	if !slices.Contains(neueVersionen, v) {
		if slices.Contains(altVersionen, v) && s.alt {
			return nil
		}
		return fehlerAntwort(n.ID, codeVersionFremd, "Unsupported protocol version",
			map[string]any{"supported": slices.Concat(neueVersionen, altVersionen), "requested": v})
	}
	if f, ok := m[metaFaehigkeit]; !ok || len(f) == 0 || f[0] != '{' {
		return fehlerAntwort(n.ID, codeParameter, "Invalid params: missing _meta."+metaFaehigkeit, nil)
	}
	return nil
}

// kundeAus liefert clientInfo der neuen Form, sonst das aus initialize.
func (s *Server) kundeAus(params json.RawMessage) Kunde {
	if m, ok := meta(params); ok {
		if roh, ok := m[metaClient]; ok {
			var k Kunde
			if json.Unmarshal(roh, &k) == nil {
				return k
			}
		}
	}
	return s.kunde
}

// schlange hält die Nachrichten, die während eines Werkzeugaufrufs eintreffen, mit Zählung der Bytes (F-468).
type schlange struct {
	zeilen []zeile
	bytes  int
}

func (q *schlange) laenge() int { return len(q.zeilen) }

func (q *schlange) nimm() zeile {
	z := q.zeilen[0]
	q.zeilen[0] = zeile{} // den Speicher freigeben
	q.zeilen = q.zeilen[1:]
	q.bytes -= len(z.daten)
	return z
}

// stelle hängt z an; false heißt, die Grenze ist erreicht und z wurde nicht angenommen.
func (q *schlange) stelle(z zeile) bool {
	if len(q.zeilen) >= MaxSchlange || q.bytes+len(z.daten) > MaxSchlangeBytes {
		return false
	}
	q.zeilen = append(q.zeilen, z)
	q.bytes += len(z.daten)
	return true
}

// streiche entfernt die älteste wartende Anfrage mit der Kennung id (nur Zeilen mit Methode und Kennung, also keine
// Benachrichtigungen) und meldet, ob eine gefunden wurde (F-701).
func (q *schlange) streiche(id json.RawMessage) bool {
	for i, z := range q.zeilen {
		if z.zuLang || len(z.daten) == 0 || z.daten[0] != '{' {
			continue
		}
		var n nachricht
		if json.Unmarshal(z.daten, &n) != nil || n.Methode == "" || len(n.ID) == 0 || !bytes.Equal(normID(n.ID), normID(id)) {
			continue
		}
		q.bytes -= len(z.daten)
		q.zeilen = slices.Delete(q.zeilen, i, i+1)
		return true
	}
	return false
}

// ueberlauf ist die Antwort auf eine Zeile, die nicht mehr in die Schlange passt: eine Anfrage mit gültiger Kennung
// bekommt -32600, alles andere (Benachrichtigungen, Antworten des Clients, Unlesbares) fällt still weg.
func ueberlauf(z zeile) *antwort {
	if z.zuLang || len(z.daten) == 0 || z.daten[0] != '{' {
		return nil
	}
	var n nachricht
	if json.Unmarshal(z.daten, &n) != nil || n.Methode == "" || len(n.ID) == 0 || !gueltigeID(n.ID) {
		return nil
	}
	return fehlerAntwort(n.ID, codeAnfrage, "Invalid Request: too many pending messages", nil)
}

// rufeAuf führt tools/call aus und liest währenddessen weiter. abgebrochen: Der Client hat die Anfrage abgebrochen
// (keine Antwort). eof: stdin ist zu, der Server endet nach dieser Anfrage. Läuft die Schlange über, schreibt rufeAuf
// die Fehlerantwort gleich nach w (das Werkzeug selbst schreibt nie dorthin).
func (s *Server) rufeAuf(ctx context.Context, n nachricht, zeilen <-chan zeile, q *schlange, w io.Writer) (res antwort, abgebrochen, eof bool, err error) {
	if f := s.pruefeMeta(n); f != nil {
		return *f, false, false, nil
	}
	var p struct {
		Name string          `json:"name"`
		Args json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(n.Params, &p); err != nil || p.Name == "" {
		return *fehlerAntwort(n.ID, codeParameter, "Invalid params: missing tool name", nil), false, false, nil
	}
	i := slices.IndexFunc(s.Werkzeuge, func(w Werkzeug) bool { return w.Name == p.Name })
	if i < 0 {
		return *fehlerAntwort(n.ID, codeParameter, "Unknown tool: "+p.Name, nil), false, false, nil
	}
	args := p.Args
	if len(args) == 0 || string(args) == "null" {
		args = json.RawMessage("{}")
	} else if args[0] != '{' {
		return *fehlerAntwort(n.ID, codeParameter, "Invalid params: arguments must be an object", nil), false, false, nil
	}
	k := s.kundeAus(n.Params)

	rctx, abbrechen := context.WithCancel(ctx)
	defer abbrechen()
	fertig := make(chan Ergebnis, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.log("Werkzeug %s: Panik: %v", p.Name, r)
				fertig <- FehlerErgebnis("Internal error in tool " + p.Name + ".")
			}
		}()
		fertig <- s.Werkzeuge[i].Aufruf(rctx, args, k)
	}()
	for {
		select {
		case e := <-fertig:
			if abgebrochen {
				return antwort{}, true, eof, nil
			}
			return ergebnisAntwort(n.ID, werkzeugErgebnis(e)), false, eof, nil
		case z, ok := <-zeilen:
			if !ok {
				eof = true
				abbrechen()
				zeilen = nil // nicht erneut lesen
				continue
			}
			if istAbbruchFuer(z, n.ID) {
				abgebrochen = true
				abbrechen()
				continue
			}
			if id, ok := abbruchZiel(z); ok && q.streiche(id) {
				continue // Abbruch einer wartenden Anfrage: sie läuft nie an und bekommt keine Antwort (F-701)
			}
			if !q.stelle(z) {
				s.log("Warteschlange voll (%d Nachrichten, %d Bytes), Nachricht verworfen", q.laenge(), q.bytes)
				if a := ueberlauf(z); a != nil {
					if err := schreibe(w, *a); err != nil {
						// Erst abbrechen und auf das Werkzeug warten: Seine Aufräumarbeit (etwa einen geschlossenen
						// Hinweis wieder zeigen) liefe sonst nicht mehr, wenn der Server danach endet (Sitzungs-QA N-1).
						abbrechen()
						<-fertig
						return antwort{}, false, eof, err
					}
				}
			}
		}
	}
}

// istAbbruchFuer erkennt notifications/cancelled mit requestId == id.
func istAbbruchFuer(z zeile, id json.RawMessage) bool {
	ziel, ok := abbruchZiel(z)
	return ok && bytes.Equal(normID(ziel), normID(id))
}

// abbruchZiel liefert die requestId einer notifications/cancelled.
func abbruchZiel(z zeile) (json.RawMessage, bool) {
	if z.zuLang || len(z.daten) == 0 {
		return nil, false
	}
	var n nachricht
	if json.Unmarshal(z.daten, &n) != nil || n.Methode != "notifications/cancelled" || len(n.ID) != 0 {
		return nil, false
	}
	var p struct {
		ID json.RawMessage `json:"requestId"`
	}
	if json.Unmarshal(n.Params, &p) != nil || len(p.ID) == 0 {
		return nil, false
	}
	return p.ID, true
}

// normID macht 1 und 1.0 vergleichbar; Zeichenketten bleiben, wie sie sind.
func normID(id json.RawMessage) []byte {
	var f float64
	if json.Unmarshal(id, &f) == nil {
		return []byte(fmt.Sprint(f))
	}
	return bytes.TrimSpace(id)
}

func werkzeugErgebnis(e Ergebnis) map[string]any {
	inhalt := e.Inhalt
	if inhalt == nil {
		inhalt = []Inhalt{}
	}
	r := map[string]any{"content": inhalt, "isError": e.Fehler}
	if e.Strukturiert != nil {
		r["structuredContent"] = e.Strukturiert
	}
	return r
}

func ergebnisAntwort(id json.RawMessage, r map[string]any) antwort {
	r["resultType"] = "complete"
	return antwort{JSONRPC: "2.0", ID: id, Result: r}
}

func fehlerAntwort(id json.RawMessage, code int, text string, data any) *antwort {
	return &antwort{JSONRPC: "2.0", ID: id, Error: &fehlerObjekt{Code: code, Message: text, Data: data}}
}

// schreibe schreibt eine Antwort als eine Zeile. json.Marshal erzeugt nie einen rohen Zeilenumbruch.
func schreibe(w io.Writer, a antwort) error {
	b, err := json.Marshal(a)
	if err != nil {
		b, _ = json.Marshal(fehlerAntwort(a.ID, codeIntern, "Internal error", nil))
	}
	b = append(b, '\n')
	_, err = w.Write(b)
	return err
}

func (s *Server) log(format string, args ...any) {
	if s.Log != nil {
		fmt.Fprintf(s.Log, "schnapp mcp: "+strings.TrimSuffix(format, "\n")+"\n", args...)
	}
}
