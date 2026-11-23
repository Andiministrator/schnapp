// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"fmt"
	"slices"
	"testing"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// F-499: Die zweite Leerung des Spalts bei sich berührenden Nähten (F-438, ADR-0023 Nachtrag 9) ist im Editor
// geprüft, ohne Fenster: fuehreNahtAblaufAus ist die Reihenfolge, die zeichneNaehte abspielt. Der Test zeichnet auf,
// was aufgerufen wird: „Spalt“ und je Durchgang, wie viele der Schichten er bejaht und ob der Schatten darunter ist.
func TestNahtAblaufZweiteLeerung(t *testing.T) {
	alle := document.KantenSchichten()
	var imSpalt int
	for _, s := range alle {
		if s.ImSpalt() {
			imSpalt++
		}
	}
	if imSpalt == 0 || imSpalt == len(alle) {
		t.Fatalf("Annahme des Tests verletzt: %d von %d Schichten liegen im Spalt", imSpalt, len(alle))
	}
	ablauf := func(naehte []document.Naht, faktor int, saumImBild bool) []string {
		var folge []string
		fuehreNahtAblaufAus(naehte, faktor, saumImBild,
			func() { folge = append(folge, "Spalt") },
			func(auswahl func(document.Schicht) bool) {
				n, schatten := 0, 0
				for _, s := range alle {
					if auswahl(s) {
						n++
						if s.ImSpalt() {
							schatten++
						}
					}
				}
				folge = append(folge, fmt.Sprintf("Schichten %d davon im Spalt %d", n, schatten))
			})
		return folge
	}
	getrennt := []document.Naht{{Zeilen: true, Lage: 100}, {Zeilen: true, Lage: 400}}
	kreuzend := []document.Naht{{Zeilen: true, Lage: 100}, {Zeilen: false, Lage: 200}}
	eng := []document.Naht{{Zeilen: true, Lage: 100}, {Zeilen: true, Lage: 105}}
	for name, c := range map[string]struct {
		naehte []document.Naht
		soll   []string
	}{
		"getrennt": {getrennt, []string{"Spalt", fmt.Sprintf("Schichten %d davon im Spalt %d", len(alle), imSpalt)}},
		"kreuzend": {kreuzend, []string{"Spalt", fmt.Sprintf("Schichten %d davon im Spalt 0", len(alle)-imSpalt), "Spalt",
			fmt.Sprintf("Schichten %d davon im Spalt %d", imSpalt, imSpalt)}},
		"eng": {eng, []string{"Spalt", fmt.Sprintf("Schichten %d davon im Spalt 0", len(alle)-imSpalt), "Spalt",
			fmt.Sprintf("Schichten %d davon im Spalt %d", imSpalt, imSpalt)}},
	} {
		if got := ablauf(c.naehte, 1, false); !slices.Equal(got, c.soll) {
			t.Errorf("%s: Ablauf %q, erwartet %q", name, got, c.soll)
		}
	}
	// Weit genug auseinander (mehr als die Bandhöhe) berühren sich Nähte derselben Achse nicht.
	if got := ablauf([]document.Naht{{Zeilen: true, Lage: 100}, {Zeilen: true, Lage: 140}}, 1, false); len(got) != 2 {
		t.Errorf("Nähte im Abstand 40 bei Faktor 1: Ablauf %q, erwartet ohne zweite Leerung", got)
	}
	// Liegen Fasersaum und Abdunklung im Bild der Leinwand (F-407), legt der Editor nur Schachbrett und Schatten, auch
	// bei kreuzenden Nähten: die zweite Leerung steckt dann im Bild (render.KantenOhneSchatten).
	soll := []string{"Spalt", fmt.Sprintf("Schichten %d davon im Spalt %d", imSpalt, imSpalt)}
	for name, n := range map[string][]document.Naht{"getrennt": getrennt, "kreuzend": kreuzend, "eng": eng} {
		if got := ablauf(n, 1, true); !slices.Equal(got, soll) {
			t.Errorf("%s, Saum im Bild: Ablauf %q, erwartet %q", name, got, soll)
		}
	}
}

// F-407: Das Bild der Leinwand trägt Fasersaum und Abdunklung, wenn es Nähte gibt und das Werkzeug nicht Zuschneiden
// ist; Kantenform, Nähte und Werkzeug gehören zum Stand, der ein Neubauen auslöst.
func TestKantenBildStand(t *testing.T) {
	e := editorMitNaht(t, 40, 30)
	k := kantenBildStand(e.s)
	if !k.im {
		t.Fatalf("mit Naht und Werkzeug %v: Saum nicht im Bild", e.s.Werkzeug)
	}
	e.s.Werkzeug = canvas.WerkzeugZuschneiden
	z := kantenBildStand(e.s)
	if z.im || z.gleich(k) {
		t.Errorf("Zuschneiden: im %v, gleich %v; erwartet Saum nicht im Bild und anderer Stand", z.im, z.gleich(k))
	}
	e.s.Werkzeug = canvas.WerkzeugAuswahl
	if err := e.s.D.SetzeKante(document.KanteZacken); err != nil {
		t.Fatal(err)
	}
	if kantenBildStand(e.s).gleich(k) {
		t.Errorf("Kantenform gewechselt, Stand gleich geblieben")
	}
	if err := e.s.D.SetzeStreifen(nil); err != nil {
		t.Fatal(err)
	}
	if o := kantenBildStand(e.s); o.im {
		t.Errorf("ohne Nähte liegt nichts im Bild")
	}
}
