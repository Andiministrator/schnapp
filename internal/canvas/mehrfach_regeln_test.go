// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

// Tests zur Kritik an Karte #334 (F-895 bis F-898, ADR-0039).

import (
	"image"
	"image/color"
	"slices"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// waehleDrei wählt die drei Kästen von dreiKaesten, Hauptauswahl 3.
func waehleDrei(s *Sitzung) {
	klick(s, f(15, 15), false)
	klick(s, f(65, 15), true)
	klick(s, f(125, 15), true)
}

func pruefeGewaehlt(t *testing.T, s *Sitzung, wo string, soll []int, haupt int) {
	t.Helper()
	if got := s.Gewaehlt(); !slices.Equal(got, soll) || s.Auswahl != haupt {
		t.Errorf("%s: gewählt %v, Haupt %d; Soll %v, Haupt %d", wo, got, s.Auswahl, soll, haupt)
	}
}

func minY(s *Sitzung, id int) float64 {
	o, _ := s.D.Objekt(id)
	return o.Grenzen().Min.Y
}

// F-895: Eine abgelegte Mehrfachauswahl lebt nicht wieder auf, wenn die frühere Hauptauswahl wieder gewählt wird.
func TestMehrfachLebtNichtWieder(t *testing.T) {
	s := dreiKaesten(t)
	waehleDrei(s)
	s.Esc()
	if offen, objekt := s.KontextBei(f(125, 15)); !offen || !objekt {
		t.Fatal("Kontextmenü")
	}
	pruefeGewaehlt(t, s, "Esc, Rechtsklick auf frühere Hauptauswahl", []int{3}, 3)

	waehleDrei(s)
	s.WaehleEbene(1)
	pruefeGewaehlt(t, s, "Ebenenliste 1", []int{1}, 1)
	s.WaehleEbene(3)
	pruefeGewaehlt(t, s, "Ebenenliste 3", []int{3}, 3)
	if s.Loeschen(); len(s.D.Objekte()) != 2 {
		t.Errorf("Löschen entfernte %d statt 1", 3-len(s.D.Objekte()))
	}

	// Rechtsklick ins Leere, dann auf die frühere Hauptauswahl.
	s = dreiKaesten(t)
	waehleDrei(s)
	s.KontextBei(f(90, 80))
	s.KontextBei(f(125, 15))
	pruefeGewaehlt(t, s, "Rechtsklick leer, dann auf 3", []int{3}, 3)

	// Ein ausgeblendetes weiteres fällt endgültig heraus: Rückgängig blendet es ein, gewählt ist es nicht mehr.
	s = dreiKaesten(t)
	klick(s, f(15, 15), false)
	klick(s, f(125, 15), true)
	if !s.SchalteAusgeblendet(1) {
		t.Fatal("ausblenden")
	}
	s.Rueckgaengig()
	pruefeGewaehlt(t, s, "ausgeblendet und zurück", []int{3}, 3)
}

// F-895: Die Bestätigung einer Texteingabe an der Hauptauswahl behält die weiteren gewählten.
func TestTexteingabeBehaeltMehrfachauswahl(t *testing.T) {
	s := dreiKaesten(t)
	tx, err := s.D.Hinzufuegen(document.Text{Ort: document.Pt(60, 60), Inhalt: "x", Schriftgrad: 12, Farbe: Schwaerzfarbe})
	if err != nil {
		t.Fatal(err)
	}
	klick(s, f(15, 15), false)
	if !s.SchalteGewaehlt(tx) || !s.Enter(false) || !s.TextLaeuft() {
		t.Fatal("Texteingabe an der Hauptauswahl")
	}
	s.Tippe("y")
	s.Enter(false)
	pruefeGewaehlt(t, s, "nach Bestätigen", []int{1, tx}, tx)
}

// F-896: Eine gesperrte oder ausgeblendete Hauptauswahl bleibt nicht in der Mehrfachauswahl; das oberste übrige rückt
// nach, und sie wird weder mitgezogen noch ausgerichtet, gedoppelt oder gelöscht.
func TestGesperrteHauptauswahlFaelltHeraus(t *testing.T) {
	s := dreiKaesten(t)
	klick(s, f(15, 15), false)
	klick(s, f(125, 15), true)
	if !s.SchalteGesperrt(3) {
		t.Fatal("sperren")
	}
	pruefeGewaehlt(t, s, "gesperrt", []int{1}, 1)
	s.Druecken(f(15, 15), TastePrimaer)
	s.Ziehen(f(35, 25))
	s.Loslassen(f(35, 25))
	if minX(t, s, 3) != 120 || minX(t, s, 1) != 30 {
		t.Errorf("gezogen: 1 bei %v, 3 bei %v", minX(t, s, 1), minX(t, s, 3))
	}

	s = dreiKaesten(t)
	klick(s, f(15, 15), false)
	klick(s, f(65, 15), true)
	klick(s, f(125, 15), true)
	if !s.SchalteAusgeblendet(3) {
		t.Fatal("ausblenden")
	}
	pruefeGewaehlt(t, s, "ausgeblendet", []int{1, 2}, 2)
	if !s.Ausrichten(AusrichtenUnten) || minX(t, s, 3) != 120 {
		t.Error("Ausrichten")
	}
	if s.Doppeln(); len(s.D.Objekte()) != 5 {
		t.Errorf("Doppeln: %d Objekte, Soll 5", len(s.D.Objekte()))
	}
	s.Loeschen() // die zwei Kopien
	if _, ok := s.D.Objekt(3); !ok || len(s.D.Objekte()) != 3 {
		t.Errorf("Löschen: %d Objekte", len(s.D.Objekte()))
	}
}

// F-896: Wird die Hauptauswahl am Dokument vorbei gesperrt, steht sie allein (Gewaehlt prüft auch sie).
func TestGewaehltPrueftHauptauswahl(t *testing.T) {
	s := dreiKaesten(t)
	klick(s, f(15, 15), false)
	klick(s, f(125, 15), true)
	o, _ := s.D.Objekt(3)
	e := o.Ebenenwerte()
	e.Gesperrt = true
	if err := s.D.Ersetzen(document.MitEbene(o, e)); err != nil {
		t.Fatal(err)
	}
	if s.KannAusrichten() || s.IstGewaehlt(1) {
		t.Errorf("gesperrte Hauptauswahl mit weiteren: %v", s.Gewaehlt())
	}
}

// F-897: Verteilen lässt das Objekt mit der kleinsten linken und das mit der größten rechten Kante stehen; dazwischen
// gleiche Lücken, auch wenn das breiteste nicht das letzte nach der linken Kante ist.
func TestVerteilenAeussereNachKanten(t *testing.T) {
	for _, waagerecht := range []bool{true, false} {
		d, _ := document.Neu(image.NewNRGBA(image.Rect(0, 0, 200, 200)))
		for _, r := range [][2]float64{{0, 10}, {20, 100}, {30, 40}} {
			b := document.R(document.Pt(r[0], 10), document.Pt(r[1], 30))
			if !waagerecht {
				b = document.R(document.Pt(10, r[0]), document.Pt(30, r[1]))
			}
			if _, err := d.Hinzufuegen(document.Kasten{Bereich: b, Fuellung: color.NRGBA{A: 255}}); err != nil {
				t.Fatal(err)
			}
		}
		_ = d.VergissVerlauf()
		s := NeueSitzung(d)
		s.SetzeFenster(image.Pt(248, 248))
		s.WaehleAlle()
		if !s.Verteilen(waagerecht) {
			t.Fatalf("waagerecht %v: Verteilen", waagerecht)
		}
		kante := func(id int) (float64, float64) {
			o, _ := s.D.Objekt(id)
			g := o.Grenzen()
			if waagerecht {
				return g.Min.X, g.Max.X
			}
			return g.Min.Y, g.Max.Y
		}
		a1, b1 := kante(1)
		a2, b2 := kante(2)
		a3, b3 := kante(3)
		// 1 bleibt 0..10, 2 bleibt 20..100, 3 (Breite 10) kommt mit gleichen Lücken dazwischen: 10..20.
		if a1 != 0 || b1 != 10 || a2 != 20 || b2 != 100 || a3 != 10 || b3 != 20 {
			t.Errorf("waagerecht %v: 1 %v..%v, 2 %v..%v, 3 %v..%v", waagerecht, a1, b1, a2, b2, a3, b3)
		}
	}
}

// Verteilen mit zwei mittleren: gleiche Lücken auch zwischen den mittleren.
func TestVerteilenVier(t *testing.T) {
	d, _ := document.Neu(image.NewNRGBA(image.Rect(0, 0, 200, 100)))
	for _, r := range [][2]float64{{0, 10}, {20, 30}, {25, 35}, {90, 100}} {
		if _, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(r[0], 10), document.Pt(r[1], 30)),
			Fuellung: color.NRGBA{A: 255}}); err != nil {
			t.Fatal(err)
		}
	}
	_ = d.VergissVerlauf()
	s := NeueSitzung(d)
	s.SetzeFenster(image.Pt(248, 148))
	s.WaehleAlle()
	// Raum 10…90 = 80, mittlere Breiten 20: drei Lücken zu 20, also 30…40 und 60…70.
	if !s.Verteilen(true) || minX(t, s, 2) != 30 || minX(t, s, 3) != 60 || minX(t, s, 4) != 90 {
		t.Errorf("verteilt: %v %v %v", minX(t, s, 2), minX(t, s, 3), minX(t, s, 4))
	}
}

// Setzt die Oberfläche (oder ein Test) die Auswahl am Paket vorbei, gelten die weiteren nicht mehr (mf.fuer).
func TestDirekteZuweisungLoestAuf(t *testing.T) {
	s := dreiKaesten(t)
	waehleDrei(s)
	s.Auswahl = 1
	pruefeGewaehlt(t, s, "direkt zugewiesen", []int{1}, 1)
}

// F-898: Strg+A wählt nie in Zuschneiden und Stauchen und nicht während der Texteingabe; ein Zeichenwerkzeug wechselt
// zu Auswählen.
func TestWaehleAlleWerkzeuge(t *testing.T) {
	for _, w := range []Werkzeug{WerkzeugZuschneiden, WerkzeugStauchen} {
		s := dreiKaesten(t)
		s.SetzeWerkzeug(w)
		if s.KannAlleWaehlen() || s.WaehleAlle() || s.Auswahl != 0 {
			t.Errorf("Werkzeug %v: Strg+A wählt %v", w, s.Gewaehlt())
		}
	}
	s := dreiKaesten(t)
	s.SetzeWerkzeug(WerkzeugText)
	s.Druecken(f(90, 80), TastePrimaer)
	s.Loslassen(f(90, 80))
	s.Tippe("abc")
	if s.KannAlleWaehlen() || s.WaehleAlle() || !s.TextLaeuft() {
		t.Errorf("Texteingabe: Strg+A wählt %v, Text läuft %v", s.Gewaehlt(), s.TextLaeuft())
	}
	s = dreiKaesten(t)
	s.SetzeWerkzeug(WerkzeugPfeil)
	if !s.KannAlleWaehlen() || !s.WaehleAlle() || s.Werkzeug != WerkzeugAuswahl || s.AnzahlGewaehlt() != 3 {
		t.Errorf("Pfeil: Werkzeug %v, gewählt %v", s.Werkzeug, s.Gewaehlt())
	}
}

// F-898: Ein Rechtsklick auf ein weiteres gewähltes behält die Mehrfachauswahl und macht es zur Hauptauswahl.
func TestKontextAufWeiteresBehaelt(t *testing.T) {
	s := dreiKaesten(t)
	waehleDrei(s)
	s.KontextBei(f(15, 15))
	pruefeGewaehlt(t, s, "Rechtsklick auf 1", []int{1, 2, 3}, 1)
}

// F-898: Der Rahmen vergleicht die aufs sichtbare Bild begrenzten Grenzen: über den Bildrand oder in den Zuschnitt
// ragende Objekte sind wählbar.
func TestRahmenSichtbareGrenzen(t *testing.T) {
	d, _ := document.Neu(image.NewNRGBA(image.Rect(0, 0, 200, 100)))
	_, _ = d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(180, 40), document.Pt(230, 60)),
		Fuellung: color.NRGBA{A: 255}})
	_ = d.VergissVerlauf()
	s := NeueSitzung(d)
	s.SetzeFenster(image.Pt(248, 148))
	s.Druecken(f(150, 20), TastePrimaer)
	s.Ziehen(f(222, 90))
	s.Loslassen(f(222, 90))
	if s.AnzahlGewaehlt() != 1 {
		t.Error("über den Bildrand ragendes Objekt nicht gewählt")
	}
	// Der Rahmen bis zum Bildrand reicht nicht über den sichtbaren Teil hinaus: links zu knapp, nichts gewählt.
	s.Esc()
	s.Druecken(f(185, 20), TastePrimaer)
	s.Ziehen(f(222, 90))
	s.Loslassen(f(222, 90))
	if s.AnzahlGewaehlt() != 0 {
		t.Error("Rahmen ohne den linken Teil wählt")
	}

	d, _ = document.Neu(image.NewNRGBA(image.Rect(0, 0, 200, 100)))
	_, _ = d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(80, 40), document.Pt(130, 60)),
		Fuellung: color.NRGBA{A: 255}})
	if err := d.SetzeZuschnitt(image.Rect(0, 0, 100, 100)); err != nil {
		t.Fatal(err)
	}
	_ = d.VergissVerlauf()
	s = NeueSitzung(d)
	s.SetzeFenster(image.Pt(248, 148))
	z, v := s.Abb.Zoom, s.Abb.Versatz
	fp := func(x, y float64) document.Punkt { return document.Pt(x*z+v.X, y*z+v.Y) }
	s.Druecken(fp(50, 20), TastePrimaer)
	s.Ziehen(fp(100, 90))
	s.Loslassen(fp(100, 90))
	if s.AnzahlGewaehlt() != 1 {
		t.Error("in den Zuschnitt ragendes Objekt nicht gewählt")
	}
}

// F-898: Esc während des Rahmens stellt die Auswahl von vorher her.
func TestRahmenEscStelltHer(t *testing.T) {
	s := dreiKaesten(t)
	waehleDrei(s)
	s.Druecken(f(90, 80), TastePrimaer)
	s.Ziehen(f(100, 90))
	if s.AnzahlGewaehlt() != 0 {
		t.Fatalf("Rahmen ohne Umschalt: %v", s.Gewaehlt())
	}
	s.Esc()
	pruefeGewaehlt(t, s, "Esc im Rahmen", []int{1, 2, 3}, 3)
}

// F-898: Alt während des Ziehens mehrerer rastet sofort aus; der Zeiger zeigt Verschieben.
func TestZugMehrereAltUndZeiger(t *testing.T) {
	s := dreiKaesten(t)
	klick(s, f(15, 15), false)
	klick(s, f(65, 15), true)
	s.Druecken(f(15, 15), TastePrimaer)
	s.Ziehen(f(15, 15+42))
	s.Ziehen(f(15, 15+31)) // Mitte 20+31 = 51 rastet an der Bildmitte 50 ein: oben 40
	p := f(15, 15+31)
	if z := s.Zeiger(&p); z != ZeigerVerschieben {
		t.Errorf("Zeiger beim Ziehen mehrerer: %v", z)
	}
	if minY(s, 1) != 40 {
		t.Fatalf("eingerastet: oben %v", minY(s, 1))
	}
	s.AltTaste(true)
	if minY(s, 1) != 41 {
		t.Errorf("Alt: oben %v, Soll 41", minY(s, 1))
	}
	s.Abbrechen()
}
