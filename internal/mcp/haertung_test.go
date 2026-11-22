// SPDX-License-Identifier: GPL-3.0-or-later

package mcp

// Tests zur Härtung nach der Sitzungs-QA Sprint 12 (ADR-0030 Nachtrag 4, F-467, F-468, F-470, F-471): Abbruch und
// Schlange, Protokollfälle und die Zweige, die bisher kein Test traf (Mutanten p2, p3, p5, p10).

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func aufruf(id string) string {
	return `{"jsonrpc":"2.0","id":` + id + `,"method":"tools/call","params":{"name":"halt",` + metaNeu + `}}` + "\n"
}

// haltServer hat ein Werkzeug „halt“, das bis frei oder bis zum Abbruch wartet. gestartet meldet jeden Start.
func haltServer(gestartet chan<- string, frei <-chan struct{}, zaehler *atomic.Int32) *Server {
	return &Server{Name: "s", Version: "1", Werkzeuge: []Werkzeug{{Name: "halt",
		Aufruf: func(ctx context.Context, _ json.RawMessage, _ Kunde) Ergebnis {
			zaehler.Add(1)
			gestartet <- "start"
			select {
			case <-ctx.Done():
				return FehlerErgebnis("abgebrochen")
			case <-frei:
				return TextErgebnis("fertig")
			}
		}}}}
}

// bedieneImHintergrund startet Bediene auf einer Pipe und liefert Schreibende, Ausgabe und Ende.
func bedieneImHintergrund(ctx context.Context, s *Server) (*io.PipeWriter, *syncPuffer, <-chan error) {
	pr, pw := io.Pipe()
	aus := &syncPuffer{mu: make(chan struct{}, 1)}
	fertig := make(chan error, 1)
	go func() { fertig <- s.Bediene(ctx, pr, aus) }()
	return pw, aus, fertig
}

func schreibeZeilen(t *testing.T, w io.Writer, zeilen ...string) {
	t.Helper()
	for _, z := range zeilen {
		if _, err := io.WriteString(w, z); err != nil {
			t.Fatal(err)
		}
	}
}

func warteEnde(t *testing.T, fertig <-chan error) {
	t.Helper()
	select {
	case err := <-fertig:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Bediene endet nicht")
	}
}

// TestAbbruchStartetSchlangeNicht: Kommt das Signal (ctx) während eines Aufrufs, startet keiner der wartenden Aufrufe
// mehr (F-467).
func TestAbbruchStartetSchlangeNicht(t *testing.T) {
	ctx, abbrechen := context.WithCancel(context.Background())
	gestartet := make(chan string, 8)
	var n atomic.Int32
	pw, _, fertig := bedieneImHintergrund(ctx, haltServer(gestartet, make(chan struct{}), &n))
	schreibeZeilen(t, pw, aufruf("1"))
	<-gestartet
	// Zwei wartende Aufrufe, dazu eine Füllzeile: Wenn ihr Schreiben zurückkehrt, liegen beide Aufrufe in der Schlange.
	schreibeZeilen(t, pw, aufruf("2"), aufruf("3"), `{"jsonrpc":"2.0","method":"x"}`+"\n", `{"jsonrpc":"2.0","method":"y"}`+"\n")
	abbrechen()
	warteEnde(t, fertig)
	pw.Close()
	if got := n.Load(); got != 1 {
		t.Errorf("%d Werkzeugaufrufe, erwartet 1 (nach dem Abbruch startet keiner)", got)
	}
}

// TestSchlangeBegrenzt: Während eines Aufrufs nimmt die Schlange höchstens MaxSchlange Nachrichten an; Anfragen
// darüber bekommen -32600 sofort, Benachrichtigungen fallen still weg (F-468).
func TestSchlangeBegrenzt(t *testing.T) {
	gestartet := make(chan string, 1)
	frei := make(chan struct{})
	var n atomic.Int32
	pw, aus, fertig := bedieneImHintergrund(context.Background(), haltServer(gestartet, frei, &n))
	schreibeZeilen(t, pw, aufruf("0"))
	<-gestartet
	for i := 1; i <= MaxSchlange+5; i++ {
		schreibeZeilen(t, pw, fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"ping"}`+"\n", i))
	}
	schreibeZeilen(t, pw, `{"jsonrpc":"2.0","method":"notifications/x"}`+"\n", `{"jsonrpc":"2.0","method":"notifications/y"}`+"\n")
	// Die fünf überzähligen sind schon beantwortet, während das Werkzeug noch läuft.
	vorher := aus.String()
	if c := strings.Count(vorher, "too many pending messages"); c != 5 {
		t.Errorf("%d Ablehnungen vor dem Ende des Aufrufs, erwartet 5: %s", c, vorher)
	}
	if !strings.Contains(vorher, fmt.Sprintf(`"id":%d,`, MaxSchlange+5)) || strings.Contains(vorher, `"id":1,`) {
		t.Errorf("falsche Anfragen abgelehnt: %s", vorher)
	}
	close(frei)
	// Erst schließen, wenn alles beantwortet ist: EOF während des Aufrufs beendet den Server (ohne die Schlange).
	ende := time.Now().Add(5 * time.Second)
	for strings.Count(aus.String(), `"resultType":"complete"`) < MaxSchlange+1 && time.Now().Before(ende) {
		time.Sleep(10 * time.Millisecond)
	}
	pw.Close()
	warteEnde(t, fertig)
	if c := strings.Count(aus.String(), `"resultType":"complete"`); c != MaxSchlange+1 {
		t.Errorf("%d Ergebnisse, erwartet %d (Aufruf und die gehaltenen Anfragen)", c, MaxSchlange+1)
	}
}

// TestSchlangeBytegrenze: Auch die Summe der Bytes ist begrenzt.
func TestSchlangeBytegrenze(t *testing.T) {
	gestartet := make(chan string, 1)
	frei := make(chan struct{})
	var n atomic.Int32
	pw, aus, fertig := bedieneImHintergrund(context.Background(), haltServer(gestartet, frei, &n))
	schreibeZeilen(t, pw, aufruf("0"))
	<-gestartet
	fuell := strings.Repeat("a", 900_000)
	anzahl := MaxSchlangeBytes/900_000 + 2 // die letzten passen nicht mehr
	for i := 1; i <= anzahl; i++ {
		schreibeZeilen(t, pw, fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"ping","params":{"p":"%s"}}`+"\n", i, fuell))
	}
	// Zwei Füllzeilen: Kehrt das Schreiben der zweiten zurück, ist die letzte Anfrage verarbeitet.
	schreibeZeilen(t, pw, `{"jsonrpc":"2.0","method":"x"}`+"\n", `{"jsonrpc":"2.0","method":"y"}`+"\n")
	if c := strings.Count(aus.String(), "too many pending messages"); c < 1 {
		t.Errorf("keine Ablehnung über %d MiB", MaxSchlangeBytes>>20)
	}
	close(frei)
	pw.Close()
	warteEnde(t, fertig)
}

// TestProtokollReste: gültiges JSON, das kein Objekt ist, ergibt -32600; eine Antwort des Clients bekommt keine
// Antwort; ping geht auch ohne _meta (F-470).
func TestProtokollReste(t *testing.T) {
	r := lauf(t, testServer(),
		`"nur eine Zeichenkette"`,
		`42`,
		`{"jsonrpc":"2.0","id":7,"result":{}}`,
		`{"jsonrpc":"2.0","id":8,"error":{"code":1,"message":"x"}}`,
		`{"jsonrpc":"2.0","id":5,"method":17}`,
		`{"jsonrpc":"2.0","id":9,"method":"ping"}`,
	)
	if len(r) != 4 {
		t.Fatalf("%d Antworten, erwartet 4 (keine auf die Antworten des Clients): %v", len(r), r)
	}
	for i := range 3 {
		if c := fehlerCode(t, r[i]); c != codeAnfrage {
			t.Errorf("Antwort %d: Code %d, erwartet -32600", i, c)
		}
	}
	if r[3]["id"] != 9.0 || r[3]["result"] == nil {
		t.Errorf("ping: %v", r[3])
	}
	// Ein Batch nennt den Grund (Mutant M16; seit F-470 wäre ein Array sonst nur „kein Objekt“).
	r = lauf(t, testServer(), `[{"jsonrpc":"2.0","id":1,"method":"ping"}]`)
	if len(r) != 1 || !strings.Contains(r[0]["error"].(map[string]any)["message"].(string), "batches") {
		t.Errorf("Batch: %v", r)
	}
}

// TestLeereZeilen: Leerzeilen bekommen keine Antwort und brechen nichts (Mutant p10).
func TestLeereZeilen(t *testing.T) {
	r := lauf(t, testServer(), ``, `   `, "\t", `{"jsonrpc":"2.0","id":1,"method":"ping"}`)
	if len(r) != 1 || r[0]["id"] != 1.0 {
		t.Errorf("Antworten: %v", r)
	}
}

// TestAltversionOhneInitialize: Eine Altversion in _meta gilt nur nach initialize (Mutant p3).
func TestAltversionOhneInitialize(t *testing.T) {
	r := lauf(t, testServer(),
		`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2025-06-18","io.modelcontextprotocol/clientCapabilities":{}}}}`)
	if len(r) != 1 || fehlerCode(t, r[0]) != codeVersionFremd {
		t.Errorf("Altversion ohne initialize: %v", r)
	}
}

// TestAbbruchMitGleitkommaKennung: requestId 7.0 bricht Anfrage 7 ab (normID, Mutant p2).
func TestAbbruchMitGleitkommaKennung(t *testing.T) {
	gestartet := make(chan string, 1)
	frei := make(chan struct{})
	var n atomic.Int32
	pw, aus, fertig := bedieneImHintergrund(context.Background(), haltServer(gestartet, frei, &n))
	schreibeZeilen(t, pw, aufruf("7"))
	<-gestartet
	// Die zwei Füllzeilen sichern, dass der Abbruch verarbeitet ist, bevor frei schließt.
	schreibeZeilen(t, pw, `{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":7.0}}`+"\n",
		`{"jsonrpc":"2.0","method":"x"}`+"\n", `{"jsonrpc":"2.0","method":"y"}`+"\n")
	close(frei)
	pw.Close()
	warteEnde(t, fertig)
	if strings.Contains(aus.String(), `"id":7`) {
		t.Errorf("Antwort trotz Abbruch mit requestId 7.0: %s", aus.String())
	}
}

// TestAbbruchMitKennungIstKeiner: notifications/cancelled als Anfrage (mit id) bricht nichts ab (Mutant p5).
func TestAbbruchMitKennungIstKeiner(t *testing.T) {
	gestartet := make(chan string, 1)
	frei := make(chan struct{})
	var n atomic.Int32
	pw, aus, fertig := bedieneImHintergrund(context.Background(), haltServer(gestartet, frei, &n))
	schreibeZeilen(t, pw, aufruf("7"))
	<-gestartet
	schreibeZeilen(t, pw, `{"jsonrpc":"2.0","id":99,"method":"notifications/cancelled","params":{"requestId":7}}`+"\n",
		`{"jsonrpc":"2.0","method":"x"}`+"\n", `{"jsonrpc":"2.0","method":"y"}`+"\n")
	close(frei)
	// Erst schließen, wenn die Antwort da ist (oder nach einer Weile): EOF bräche den Aufruf sonst selbst ab.
	ende := time.Now().Add(2 * time.Second)
	for !strings.Contains(aus.String(), `"id":7,`) && time.Now().Before(ende) {
		time.Sleep(10 * time.Millisecond)
	}
	pw.Close()
	warteEnde(t, fertig)
	if !strings.Contains(aus.String(), `"id":7,`) || !strings.Contains(aus.String(), "fertig") {
		t.Errorf("Anfrage 7 fälschlich abgebrochen: %s", aus.String())
	}
}
