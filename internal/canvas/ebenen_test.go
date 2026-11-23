// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ebenenSitzung: drei gefüllte Kästen übereinander (unten a, Mitte b, oben c), Fenster 1:1 mit Versatz 24.
func ebenenSitzung(t *testing.T) (*Sitzung, [3]int) {
	t.Helper()
	s := randSitzung(t)
	var ids [3]int
	for i := range ids {
		id, err := s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(40+float64(i)*10, 40), document.Pt(120, 120)),
			Fuellung: Schwaerzfarbe, Strich: document.Strich{Farbe: Schwaerzfarbe, Breite: 2}})
		if err != nil {
			t.Fatal(err)
		}
		ids[i] = id
	}
	return s, ids
}

func fenster(x, y float64) document.Punkt { return document.Pt(x+24, y+24) }

// ADR-0032 Punkt 6: Die Liste steht von oben (vorderste Ebene) nach unten und zeigt die Wahl.
func TestEbenenListe(t *testing.T) {
	s, ids := ebenenSitzung(t)
	s.Auswahl = ids[1]
	z := s.Ebenen()
	if len(z) != 3 || z[0].ID != ids[2] || z[2].ID != ids[0] || !z[1].Gewaehlt || z[0].Gewaehlt || z[0].Art != WerkzeugKasten {
		t.Fatalf("Liste %+v", z)
	}
}

// Punkte 4 und 5: Ausgeblendete und gesperrte Ebenen trifft der Klick nicht, er greift die darunter; eine gesperrte
// gewählte Ebene hat keine Griffe.
func TestEbenenTrefferUndGriffe(t *testing.T) {
	s, ids := ebenenSitzung(t)
	klick := func() int {
		s.Druecken(fenster(100, 100), TastePrimaer)
		s.Loslassen(fenster(100, 100))
		return s.Auswahl
	}
	if got := klick(); got != ids[2] {
		t.Fatalf("Klick trifft %d, Soll oberste %d", got, ids[2])
	}
	if !s.SchalteAusgeblendet(ids[2]) {
		t.Fatal("Ausblenden")
	}
	if got := klick(); got != ids[1] {
		t.Errorf("ausgeblendete oberste getroffen: %d", got)
	}
	if !s.SchalteGesperrt(ids[1]) {
		t.Fatal("Sperren")
	}
	if got := klick(); got != ids[0] {
		t.Errorf("gesperrte getroffen: %d", got)
	}
	// Griffe: die gesperrte Ebene, über die Liste gewählt, lässt sich am Eckgriff nicht ziehen.
	if !s.WaehleEbene(ids[1]) {
		t.Fatal("Wählen in der Liste")
	}
	vor, _ := s.D.Objekt(ids[1])
	s.Druecken(fenster(120, 120), TastePrimaer) // Ecke unten rechts
	s.Ziehen(fenster(150, 150))
	s.Loslassen(fenster(150, 150))
	if nach, _ := s.D.Objekt(ids[1]); nach != vor {
		t.Error("gesperrte Ebene am Griff verändert")
	}
	// Entsperrt wirkt der Griff wieder.
	s.SchalteGesperrt(ids[1])
	s.WaehleEbene(ids[1])
	s.Druecken(fenster(120, 120), TastePrimaer)
	s.Ziehen(fenster(150, 150))
	s.Loslassen(fenster(150, 150))
	if nach, _ := s.D.Objekt(ids[1]); nach == vor {
		t.Error("entsperrt: Griff wirkt nicht")
	}
}

// Punkt 3: Jede Aktion ist genau ein Rückgängig-Schritt; Wählen ist keiner.
func TestEbenenAktionenEinSchritt(t *testing.T) {
	s, ids := ebenenSitzung(t)
	schritt := func(name string, f func() bool, pruefe func() bool) {
		t.Helper()
		vor := s.D.Stand()
		if !f() || s.D.Stand() == vor || !pruefe() {
			t.Fatalf("%s: keine Wirkung", name)
		}
		if !s.Rueckgaengig() || s.D.Stand() != vor {
			t.Fatalf("%s: Rückgängig führt nicht zurück", name)
		}
		if !s.Wiederholen() {
			t.Fatalf("%s: Wiederholen", name)
		}
	}
	eb := func(id int) document.Ebene { o, _ := s.D.Objekt(id); return o.Ebenenwerte() }
	vor := s.D.Stand()
	if !s.WaehleEbene(ids[0]) || s.Auswahl != ids[0] || s.D.Stand() != vor || s.Werkzeug != WerkzeugAuswahl {
		t.Fatal("Wählen")
	}
	schritt("Ausblenden", func() bool { return s.SchalteAusgeblendet(ids[0]) }, func() bool { return eb(ids[0]).Ausgeblendet })
	schritt("Sperren", func() bool { return s.SchalteGesperrt(ids[0]) }, func() bool { return eb(ids[0]).Gesperrt })
	schritt("Deckkraft", func() bool { return s.SetzeDeckkraft(ids[0], 30) }, func() bool { return eb(ids[0]).Deckkraft() == 30 })
	schritt("Umbenennen", func() bool { return s.Umbenennen(ids[0], "  Rahmen ") }, func() bool { return eb(ids[0]).Name == "Rahmen" })
	schritt("Nach vorn", func() bool { return s.EbeneSchritt(1) }, func() bool { return s.D.Stelle(ids[0]) == 1 })
	schritt("Nach oben ziehen", func() bool { return s.EbeneNach(ids[0], 2) }, func() bool { return s.D.Stelle(ids[0]) == 2 })
	if s.Umbenennen(ids[0], "a\x07b") || s.SetzeDeckkraft(ids[0], 30) || s.EbeneNach(ids[0], 2) {
		t.Error("ungültiger Name oder Aktion ohne Änderung legt einen Schritt an")
	}
	// Doppeln: Kopie direkt über dem Original, versetzt, gewählt, mit den Angaben der Ebene.
	s.WaehleEbene(ids[1])
	orig, _ := s.D.Objekt(ids[1])
	stelle := s.D.Stelle(ids[1])
	schritt("Doppeln", s.Doppeln, func() bool {
		k, ok := s.D.Objekt(s.Auswahl)
		if !ok || s.Auswahl == ids[1] || s.D.Stelle(s.Auswahl) != stelle+1 {
			return false
		}
		v := DoppelVersatz / s.Abb.Zoom
		return k.(document.Kasten).Bereich.Min == orig.(document.Kasten).Bereich.Min.Plus(document.Pt(v, v))
	})
}

// Deckkraft aus dem Feld, Nachbarwahl per Tastatur, Menü an/aus, Text bearbeiten nicht an einer ausgeblendeten Ebene.
func TestEbenenFeldUndNachbar(t *testing.T) {
	s, ids := ebenenSitzung(t)
	if s.KannEbene() || s.SetzeEbenenDeckkraftText("50") {
		t.Fatal("ohne Auswahl an")
	}
	if !s.WaehleNachbar(1) || s.Auswahl != ids[2] {
		t.Fatalf("ohne Auswahl: darüber wählt die oberste, nicht %d", s.Auswahl)
	}
	if !s.WaehleNachbar(-1) || s.Auswahl != ids[1] || !s.WaehleNachbar(-1) || !s.WaehleNachbar(-1) || s.Auswahl != ids[0] {
		t.Fatalf("darunter: %d", s.Auswahl)
	}
	if s.KannEbeneSchritt(-1) || !s.KannEbeneSchritt(1) {
		t.Error("KannEbeneSchritt an der untersten")
	}
	for text, soll := range map[string]bool{"40": true, " 40 % ": true, "101": false, "x": false, "-1": false} {
		if got := s.SetzeEbenenDeckkraftText(text); got != soll {
			t.Errorf("Feld %q: %v", text, got)
		}
	}
	if o, _ := s.D.Objekt(ids[0]); o.Ebenenwerte().Deckkraft() != 40 {
		t.Error("Deckkraft aus dem Feld nicht gesetzt")
	}
	txt, err := s.D.Hinzufuegen(document.Text{Ort: document.Pt(10, 10), Inhalt: "x", Schriftgrad: 12, Farbe: Schwaerzfarbe,
		Ebene: document.Ebene{Ausgeblendet: true}})
	if err != nil {
		t.Fatal(err)
	}
	s.WaehleEbene(txt)
	if s.KannTextBearbeiten() || s.TextBearbeiten() {
		t.Error("Texteingabe an einer ausgeblendeten Ebene")
	}
}
