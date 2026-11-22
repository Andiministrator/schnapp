// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"bytes"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
)

// rettungInhalt hält, was jede Sicherung tatsächlich geschrieben hätte: die Objekte aus dem SVG der Rettungskopie.
type rettungInhalt struct {
	kopien    [][]document.Objekt
	verworfen int
}

func (r *rettungInhalt) Wiederherstellen(*document.Dokument) (string, bool) { return "", false }
func (r *rettungInhalt) Verwirf()                                           { r.verworfen++ }
func (r *rettungInhalt) Ende()                                              {}
func (r *rettungInhalt) Lage() (string, bool, uint64)                       { return "", false, 0 }
func (r *rettungInhalt) Sichere(d *document.Dokument) {
	a, err := fileformat.LeseAngaben(bytes.NewReader(fileformat.AngabenSVG(d)))
	if err != nil {
		panic(err)
	}
	r.kopien = append(r.kopien, a.Objekte)
}

func (r *rettungInhalt) letzte(t *testing.T) []document.Objekt {
	t.Helper()
	if len(r.kopien) == 0 {
		t.Fatal("keine Kopie geschrieben")
	}
	return r.kopien[len(r.kopien)-1]
}

func rettungZugStart(t *testing.T) (*editor, *rettungInhalt, *time.Time) {
	t.Helper()
	jetzt := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	alt := rettungUhr
	rettungUhr = func() time.Time { return jetzt }
	t.Cleanup(func() { rettungUhr = alt })
	e := testEditor(t)
	r := &rettungInhalt{}
	e.rettung = r
	e.rettungStart()
	haltAendere(t, e) // eine fertige Änderung: Text „y“
	e.rettungPruefen()
	return e, r, &jetzt
}

// F-409: Eine Kopie, die während eines Zugs fällig wird, trägt das Objekt im Zug nicht. Bricht der Nutzer den Zug ab
// (Esc), steht der verworfene Text in keiner Kopie; die fertige Änderung davor ist gesichert (V7-2 bleibt erfüllt).
func TestRettungKopieNachAbgebrochenemZug(t *testing.T) {
	e, r, jetzt := rettungZugStart(t)
	if _, err := e.s.D.BeginneNeuenZug(document.Text{Ort: document.Pt(3, 20), Inhalt: "verworfener Text", Schriftgrad: 10}); err != nil {
		t.Fatal(err)
	}
	*jetzt = jetzt.Add(3 * time.Second)
	e.rettungPruefen() // fällig während des Zugs
	if len(r.kopien) != 1 {
		t.Fatalf("während des Zugs %d Kopien, erwartet 1 (V7-2)", len(r.kopien))
	}
	if err := e.s.D.BrecheZugAb(); err != nil {
		t.Fatal(err)
	}
	*jetzt = jetzt.Add(5 * time.Minute)
	e.rettungPruefen()
	e.rettungPruefen()
	for i, k := range r.kopien {
		for _, o := range k {
			if tx, ok := o.(document.Text); ok && tx.Inhalt == "verworfener Text" {
				t.Errorf("Kopie %d trägt den verworfenen Text", i)
			}
		}
	}
	if got, want := len(r.letzte(t)), len(e.s.D.Objekte()); got != want {
		t.Errorf("letzte Kopie trägt %d Objekte, das Dokument hat %d", got, want)
	}
}

// F-409: Bestätigt der Nutzer den Zug, ändert sich der Stand; die nächste Kopie trägt das neue Objekt.
func TestRettungKopieNachBestaetigtemZug(t *testing.T) {
	e, r, jetzt := rettungZugStart(t)
	if _, err := e.s.D.BeginneNeuenZug(document.Text{Ort: document.Pt(3, 20), Inhalt: "bleibt", Schriftgrad: 10}); err != nil {
		t.Fatal(err)
	}
	*jetzt = jetzt.Add(3 * time.Second)
	e.rettungPruefen()
	if err := e.s.D.BeendeZug(); err != nil {
		t.Fatal(err)
	}
	e.rettungPruefen()
	*jetzt = jetzt.Add(rettungRuhe)
	e.rettungPruefen()
	if len(r.kopien) != 2 {
		t.Fatalf("%d Kopien, erwartet 2 (eine im Zug, eine nach dem Übernehmen)", len(r.kopien))
	}
	gefunden := false
	for _, o := range r.letzte(t) {
		if tx, ok := o.(document.Text); ok && tx.Inhalt == "bleibt" {
			gefunden = true
		}
	}
	if !gefunden {
		t.Error("übernommener Text fehlt in der Kopie nach dem Zug")
	}
}

// F-409: Ein abgebrochenes Verschieben hinterlässt in der Kopie die alte Lage, nie die verworfene.
func TestRettungKopieWaehrendVerschieben(t *testing.T) {
	e, r, jetzt := rettungZugStart(t)
	objekte := e.s.D.Objekte()
	vorher := objekte[len(objekte)-1].(document.Text)
	if err := e.s.D.BeginneZug(vorher.Kennung()); err != nil {
		t.Fatal(err)
	}
	verschoben := vorher
	verschoben.Ort = document.Pt(30, 25)
	if err := e.s.D.ZugSetze(verschoben); err != nil {
		t.Fatal(err)
	}
	*jetzt = jetzt.Add(3 * time.Second)
	e.rettungPruefen()
	k := r.letzte(t)
	tx, ok := k[len(k)-1].(document.Text)
	if !ok || tx.Ort != vorher.Ort {
		t.Errorf("Kopie während des Verschiebens trägt %#v, erwartet die Lage vor dem Zug %v", k[len(k)-1], vorher.Ort)
	}
}
