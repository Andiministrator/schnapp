// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"image/color"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// blaseImStreifen legt eine Sprechblase an, deren Schild ganz in einem Stauch-Streifen liegt; sichtbar bleibt nur das
// Dreieck. Liefert die Sitzung, die Kennung und den Fensterpunkt an der Spitze.
func blaseImStreifen(t *testing.T, stauchen bool) (*Sitzung, int, document.Punkt) {
	t.Helper()
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 320, 160)))
	if err != nil {
		t.Fatal(err)
	}
	sp := document.Pt(280, 140)
	tx := document.Text{Ort: document.Pt(100, 40), Inhalt: "GEHEIM", Schriftgrad: 14, Farbe: color.NRGBA{A: 255},
		Hintergrund: color.NRGBA{R: 200, G: 200, B: 255, A: 255}, Spitze: &sp}
	id, err := d.Hinzufuegen(tx)
	if err != nil {
		t.Fatal(err)
	}
	if stauchen {
		g := tx.Grenzen()
		if err := d.SetzeStreifen([]document.Streifen{{Von: int(g.Min.X) - 6, Bis: int(g.Max.X) + 7}}); err != nil {
			t.Fatal(err)
		}
		if !d.NurSpitzeSichtbar(tx) {
			t.Fatal("Aufbau: Schild soll ganz im Streifen liegen")
		}
	}
	if err := d.VergissVerlauf(); err != nil {
		t.Fatal(err)
	}
	s := NeueSitzung(d)
	s.SetzeFenster(d.Groesse().Add(image.Pt(48, 48)))
	if s.Abb.Zoom != 1 || s.Abb.Versatz != document.Pt(24, 24) {
		t.Fatalf("Abbildung %+v", s.Abb)
	}
	// Knapp neben der Spitze ins Dreieck hinein, in Ansichtskoordinaten.
	q := d.ZuAnsicht(sp.Minus(document.Pt(3, 2)))
	return s, id, f(q.X, q.Y)
}

// F-740 (ADR-0023 Nachtrag 13): Ein Klick mit dem Text- oder Sprechblasen-Werkzeug, ein Doppelklick und Enter auf
// eine Sprechblase, deren Schild ganz im Streifen liegt, öffnen keine Eingabe; die Blase wird gewählt und die
// Statusleiste nennt den Grund. Ohne Streifen öffnet derselbe Klick die Eingabe.
func TestKeineEingabeAnVerborgenemText(t *testing.T) {
	for _, w := range []Werkzeug{WerkzeugText, WerkzeugSprechblase} {
		s, id, p := blaseImStreifen(t, false)
		s.SetzeWerkzeug(w)
		s.Druecken(p, TastePrimaer)
		if !s.TextLaeuft() || s.Auswahl != id {
			t.Fatalf("Gegenstück ohne Streifen, Werkzeug %v: Eingabe %v, Auswahl %d", w, s.TextLaeuft(), s.Auswahl)
		}

		s, id, p = blaseImStreifen(t, true)
		s.SetzeWerkzeug(w)
		s.Druecken(p, TastePrimaer)
		s.Loslassen(p)
		if s.TextLaeuft() {
			t.Errorf("Werkzeug %v: Eingabe an verborgenem Text geöffnet", w)
		}
		if s.Auswahl != id || s.Abgelehnt() != AblehnungTextVerborgen {
			t.Errorf("Werkzeug %v: Auswahl %d (erwartet %d), Ablehnung %v", w, s.Auswahl, id, s.Abgelehnt())
		}
		if len(s.D.Objekte()) != 1 {
			t.Errorf("Werkzeug %v: neues Objekt angelegt", w)
		}
		if s.KannTextBearbeiten() || s.Enter(false) && s.TextLaeuft() {
			t.Errorf("Werkzeug %v: Menü oder Enter öffnen die Eingabe", w)
		}
	}
	s, id, p := blaseImStreifen(t, true)
	s.Doppelklick(p)
	if s.TextLaeuft() || s.Auswahl != id || s.Abgelehnt() != AblehnungTextVerborgen {
		t.Errorf("Doppelklick: Eingabe %v, Auswahl %d, Ablehnung %v", s.TextLaeuft(), s.Auswahl, s.Abgelehnt())
	}
}
