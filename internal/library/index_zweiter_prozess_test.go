// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"os"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/fileformat"
)

// F-755: Prozess 2 hat die Datei einer neuen Aufnahme schon in der Ablage, seine Indexzeile aber noch nicht
// angehängt. Gleicht Prozess 1 in diesem Fenster ab, ergänzt er die Zeile; danach darf haengeAn sie nicht doppelt
// anhängen.
func TestHaengeAnHaengtErgaenzteZeileNichtDoppeltAn(t *testing.T) {
	p1, _ := neueBibliothek(t)
	p2 := Neu(p1.Orte)
	p2.Zone, p2.Jetzt = p1.Zone, p1.Jetzt

	ablegen(t, p1, fileformat.Metadaten{Erstellt: zeit("2026-10-01T10:00:00Z")})
	vorher, err := os.ReadFile(p1.Orte.Index)
	if err != nil {
		t.Fatal(err)
	}
	neu := ablegen(t, p2, fileformat.Metadaten{Erstellt: zeit("2026-11-24T10:00:00Z")})
	// Zustand „Datei da, Zeile noch nicht“ nachstellen.
	if err := os.WriteFile(p1.Orte.Index, vorher, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := p1.Abgleich(); err != nil {
		t.Fatal(err)
	}
	p2.mu.Lock()
	err = p2.haengeAn(neu)
	p2.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	g, err := p1.Laden()
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, e := range g.Eintraege {
		if e.Pfad == neu.Pfad {
			n++
		}
	}
	if n != 1 || len(g.Eintraege) != 2 {
		t.Errorf("Zeilen für %s: %d, Zeilen im Index: %d (Soll 1 und 2)", neu.Pfad, n, len(g.Eintraege))
	}
}

// F-756: Ablegen an einen fehlenden Index baut ihn neu auf, heilt aber keine Vorschaubilder der alten Aufnahmen im
// Vordergrund der Aufnahme. Gezählt wird der Heilweg des Neuaufbaus, dazu eine großzügige Zeitgrenze.
func TestAblegenOhneIndexHeiltKeineAltenVorschauen(t *testing.T) {
	b, _ := neueBibliothek(t)
	const anzahl = 12
	for i := range anzahl {
		ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-01-01T10:00:00Z").Add(time.Duration(i) * time.Hour)})
	}
	if err := os.Remove(b.Orte.Index); err != nil {
		t.Fatal(err)
	}
	es, err := os.ReadDir(b.Orte.Vorschau)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range es {
		if err := os.Remove(b.Orte.Vorschau + "/" + e.Name()); err != nil {
			t.Fatal(err)
		}
	}
	alt := heileNachNeuaufbau
	heilungen := 0
	heileNachNeuaufbau = func(*Bibliothek, string, string, Stand) { heilungen++ }
	t.Cleanup(func() { heileNachNeuaufbau = alt })

	t0 := time.Now()
	neu := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-06-02T10:00:00Z")})
	if d := time.Since(t0); d > 10*time.Second {
		t.Errorf("Ablegen ohne Index dauerte %v", d)
	}
	if heilungen != 0 {
		t.Errorf("%d Vorschauen im Vordergrund geheilt, soll 0", heilungen)
	}
	p := indexPfade(t, b)
	if len(p) != anzahl+1 || !p[neu.Pfad] {
		t.Errorf("Index nach Ablegen: %d Zeilen, soll %d", len(p), anzahl+1)
	}
	if n := vorschauZahl(t, b); n != 1 {
		t.Errorf("%d Vorschaubilder, soll nur das der neuen Aufnahme (1)", n)
	}
}

// F-755, Gegenrichtung: Steht für den Pfad schon eine Zeile (etwa einer verschwundenen Datei, deren Namen die neue
// Aufnahme wieder trägt), ersetzt haengeAn sie mit den Angaben des neuen Eintrags, statt sie stehen zu lassen.
func TestHaengeAnErsetztVorhandeneZeile(t *testing.T) {
	b, _ := neueBibliothek(t)
	e := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-10-01T10:00:00Z")})
	neu := e
	neu.Bearbeitet = true
	neu.Breite = e.Breite + 7
	b.mu.Lock()
	err := b.haengeAn(neu)
	b.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	g, err := b.Laden()
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Eintraege) != 1 || !g.Eintraege[0].Bearbeitet || g.Eintraege[0].Breite != e.Breite+7 {
		t.Errorf("Index %+v, Soll eine Zeile mit den neuen Angaben (Bearbeitet, Breite %d)", g.Eintraege, e.Breite+7)
	}
}
