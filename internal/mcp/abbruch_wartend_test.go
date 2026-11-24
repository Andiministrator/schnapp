// SPDX-License-Identifier: GPL-3.0-or-later

package mcp

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// abbruchServer hat ein blockierendes Werkzeug a (wartet auf frei) und ein Werkzeug b, das zählt, wie oft es lief.
func abbruchServer(frei <-chan struct{}, bLief *atomic.Int32) *Server {
	return &Server{Name: "s", Version: "1", Werkzeuge: []Werkzeug{
		{Name: "a", Aufruf: func(ctx context.Context, _ json.RawMessage, _ Kunde) Ergebnis {
			<-frei
			return Ergebnis{Inhalt: []Inhalt{TextInhalt("a fertig")}}
		}},
		{Name: "b", Aufruf: func(ctx context.Context, _ json.RawMessage, _ Kunde) Ergebnis {
			bLief.Add(1)
			return Ergebnis{Inhalt: []Inhalt{TextInhalt("b fertig")}}
		}},
	}}
}

// laufeAbbruch schickt die Zeilen, gibt a nach 50 ms frei und liefert die gesamte Ausgabe, sobald die Antwort auf ping 99 da ist.
func laufeAbbruch(t *testing.T, s *Server, frei chan struct{}, zeilen ...string) string {
	t.Helper()
	pr, pw := io.Pipe()
	var aus syncPuffer
	fertig := make(chan error, 1)
	go func() { fertig <- s.Bediene(context.Background(), pr, &aus) }()
	for _, z := range zeilen {
		io.WriteString(pw, z+"\n")
	}
	io.WriteString(pw, `{"jsonrpc":"2.0","id":99,"method":"ping"}`+"\n")
	time.Sleep(50 * time.Millisecond) // alle Zeilen sind gelesen, a läuft noch
	close(frei)
	frist := time.Now().Add(5 * time.Second)
	for !strings.Contains(aus.String(), `"id":99`) && time.Now().Before(frist) {
		time.Sleep(time.Millisecond)
	}
	pw.Close()
	if err := <-fertig; err != nil {
		t.Fatal(err)
	}
	return aus.String()
}

func wartendCall(id int, name string) string {
	return `{"jsonrpc":"2.0","id":` + string(rune('0'+id)) + `,"method":"tools/call","params":{"name":"` + name + `",` + metaNeu + `}}`
}

const abbruch2 = `{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":2}}`

// Ein Abbruch für eine noch wartende Anfrage verhindert ihren Start; sie bekommt keine Antwort (F-701).
func TestAbbruchWartendeAnfrageLaeuftNie(t *testing.T) {
	frei := make(chan struct{})
	var b atomic.Int32
	aus := laufeAbbruch(t, abbruchServer(frei, &b), frei, wartendCall(1, "a"), wartendCall(2, "b"), abbruch2)
	if b.Load() != 0 {
		t.Errorf("abgebrochene wartende Anfrage lief %d-mal", b.Load())
	}
	if strings.Contains(aus, `"id":2`) {
		t.Errorf("Antwort auf die abgebrochene Anfrage:\n%s", aus)
	}
	if !strings.Contains(aus, `"id":1`) || !strings.Contains(aus, `"id":99`) {
		t.Errorf("Antworten auf 1 und 99 fehlen:\n%s", aus)
	}
}

// Nur die genannte Anfrage fällt weg, die danach wartende läuft.
func TestAbbruchWartendeNurDieGenannte(t *testing.T) {
	frei := make(chan struct{})
	var b atomic.Int32
	aus := laufeAbbruch(t, abbruchServer(frei, &b), frei, wartendCall(1, "a"), wartendCall(2, "b"), wartendCall(3, "b"), abbruch2)
	if b.Load() != 1 {
		t.Errorf("b lief %d-mal, erwartet 1 (nur Anfrage 3)", b.Load())
	}
	if strings.Contains(aus, `"id":2`) || !strings.Contains(aus, `"id":3`) {
		t.Errorf("falsche Antworten:\n%s", aus)
	}
}

// Ein Abbruch trifft eine Anfrage nur einmal: gleiche Kennung zweimal wartend, ein Abbruch streicht die ältere.
func TestAbbruchWartendeGleicheKennung(t *testing.T) {
	frei := make(chan struct{})
	var b atomic.Int32
	laufeAbbruch(t, abbruchServer(frei, &b), frei, wartendCall(1, "a"), wartendCall(2, "b"), wartendCall(2, "b"), abbruch2)
	if b.Load() != 1 {
		t.Errorf("b lief %d-mal, erwartet 1", b.Load())
	}
}

// Ein Abbruch ohne passende wartende Anfrage ändert nichts (Anfrage 2 kommt erst danach).
func TestAbbruchOhneZielAendertNichts(t *testing.T) {
	frei := make(chan struct{})
	var b atomic.Int32
	aus := laufeAbbruch(t, abbruchServer(frei, &b), frei, wartendCall(1, "a"), abbruch2, wartendCall(2, "b"))
	if b.Load() != 1 || !strings.Contains(aus, `"id":2`) {
		t.Errorf("Anfrage nach einem ins Leere gehenden Abbruch muss laufen (b=%d):\n%s", b.Load(), aus)
	}
}

// Die Zählung der Schlange stimmt nach dem Streichen.
func TestSchlangeStreicheZaehlt(t *testing.T) {
	var q schlange
	q.stelle(zeile{daten: []byte(`{"jsonrpc":"2.0","id":7,"method":"ping"}`)})
	q.stelle(zeile{daten: []byte(`{"jsonrpc":"2.0","id":"x","method":"ping"}`)})
	if q.streiche(json.RawMessage(`8`)) {
		t.Fatal("falsche Kennung gestrichen")
	}
	if !q.streiche(json.RawMessage(`7.0`)) || q.laenge() != 1 {
		t.Fatalf("7 nicht gestrichen, Länge %d", q.laenge())
	}
	if want := len(`{"jsonrpc":"2.0","id":"x","method":"ping"}`); q.bytes != want {
		t.Errorf("bytes %d, erwartet %d", q.bytes, want)
	}
}
