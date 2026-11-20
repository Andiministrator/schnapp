// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"bytes"
	"image"
	"image/color"
	"reflect"
	"testing"

	"github.com/Andiministrator/schnapp/document"
)

// fv rechnet einen Ansichtspunkt mit der aktuellen Abbildung der Sitzung ins Fenster (nach Zuschneiden und Stauchen
// passt der Editor neu ein, die festen Werte von f gelten dann nicht mehr).
func fv(s *Sitzung, x, y float64) document.Punkt { return s.Abb.ZuFenster(document.Pt(x, y)) }

// ADR-0023 Punkte 1 und 5: Ein Rahmen mit dem Werkzeug Zuschneiden setzt beim Loslassen den Zuschnitt (auf ganze
// Ansichtspixel gerundet), ändert weder Original noch Objekte, ist ein Rückgängig-Schritt und passt die Ansicht neu
// ein; Rückgängig stellt das ganze Bild und die alte Einpassung wieder her.
func TestZuschneidenWerkzeug(t *testing.T) {
	s := verlaufSitzung(t)
	vorher := s.D.Original()
	if !s.SetzeWerkzeug(WerkzeugZuschneiden) || s.Voll() != AblehnungKeine {
		t.Fatal("Werkzeug")
	}
	s.Druecken(f(20.4, 10.4), TastePrimaer)
	s.Ziehen(f(60, 30))
	if a, b, ok := s.Rahmen(); !ok || a != f(20.4, 10.4) || b != f(60, 30) {
		t.Errorf("Rahmen beim Ziehen: %v %v %v", a, b, ok)
	}
	s.Loslassen(f(120.6, 60.6))
	if z := s.D.Zuschnitt(); z != image.Rect(20, 10, 121, 61) {
		t.Fatalf("Zuschnitt %v, Soll (20,10)-(121,61)", z)
	}
	if g := s.D.Groesse(); g != image.Pt(101, 51) {
		t.Errorf("Ansicht %v", g)
	}
	if !bytes.Equal(s.D.Original().Pix, vorher.Pix) || len(s.D.Objekte()) != 0 {
		t.Error("Zuschneiden hat Original oder Objekte geändert")
	}
	if s.BildAenderung() != 1 || !s.Geaendert() {
		t.Errorf("BildAenderung %d, Geaendert %v", s.BildAenderung(), s.Geaendert())
	}
	if s.Abb != Einpassen(image.Pt(101, 51), image.Pt(248, 148), Rand) {
		t.Errorf("nach dem Zuschneiden nicht eingepasst: %+v", s.Abb)
	}
	if n := schritte(t, s); n != 1 {
		t.Errorf("%d Rückgängig-Schritte, Soll 1", n)
	}
	s.Scrollen(f(50, 50), -1) // Nutzer zoomt: Rückgängig passt trotzdem neu ein (die Ansicht hat eine andere Größe)
	vor := s.BildAenderung()
	if !s.Rueckgaengig() || s.D.Zuschnitt() != image.Rect(0, 0, 200, 100) {
		t.Fatal("Rückgängig")
	}
	if s.Abb != Einpassen(image.Pt(200, 100), image.Pt(248, 148), Rand) || s.BildAenderung() != vor+1 {
		t.Errorf("nach Rückgängig: %+v, BildAenderung %d", s.Abb, s.BildAenderung())
	}
}

// Unter der Mindestgröße, mit Esc abgebrochen oder als Rahmen über die ganze Ansicht ändert Zuschneiden nichts.
func TestZuschneidenOhneWirkung(t *testing.T) {
	s := verlaufSitzung(t)
	s.SetzeWerkzeug(WerkzeugZuschneiden)
	zieh(s, f(10, 10), f(14, 80)) // 4 Pixel breit
	s.Druecken(f(10, 10), TastePrimaer)
	s.Ziehen(f(100, 80))
	s.Esc()
	if _, _, ok := s.Rahmen(); ok {
		t.Error("Rahmen nach Esc")
	}
	s.Loslassen(f(100, 80))
	zieh(s, f(-50, -50), f(500, 500)) // beginnt neben dem Bild: nichts (F-150)
	zieh(s, f(0, 0), f(500, 500))     // ganze Ansicht
	if s.Geaendert() || s.BildAenderung() != 0 || s.D.Zuschnitt() != image.Rect(0, 0, 200, 100) {
		t.Errorf("geändert: Zuschnitt %v", s.D.Zuschnitt())
	}
}

// ADR-0023 Punkt 2: Doppelklick mit dem Werkzeug Zuschneiden zeigt wieder das ganze Original, als ein Schritt; ohne
// Zuschnitt wirkt er wie ein Druck. Die Streifen bleiben.
func TestZuschnittAufheben(t *testing.T) {
	s := verlaufSitzung(t)
	if err := s.D.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 40, Bis: 50}}); err != nil {
		t.Fatal(err)
	}
	s.SetzeWerkzeug(WerkzeugZuschneiden)
	zieh(s, f(10, 10), f(100, 60))
	if s.D.Zuschnitt() == image.Rect(0, 0, 200, 100) {
		t.Fatal("nicht zugeschnitten")
	}
	vor := schritte(t, s)
	s.Doppelklick(fv(s, 5, 5))
	if s.D.Zuschnitt() != image.Rect(0, 0, 200, 100) || len(s.D.Streifen()) != 1 {
		t.Errorf("nach Doppelklick: Zuschnitt %v, Streifen %v", s.D.Zuschnitt(), s.D.Streifen())
	}
	if n := schritte(t, s); n != vor+1 {
		t.Errorf("%d Schritte, Soll %d", n, vor+1)
	}
	s.Loslassen(fv(s, 5, 5))
	s.Doppelklick(fv(s, 5, 5)) // schon ganz: wie ein Druck, ohne Ziehen nichts
	s.Loslassen(fv(s, 5, 5))
	if n := schritte(t, s); n != vor+1 {
		t.Errorf("zweiter Doppelklick hat etwas geändert: %d Schritte", n)
	}
}

// ADR-0023 Punkt 1: Ein Zuschnitt über eine Naht nimmt den Streifen mit; einer, der an der Naht endet, endet vor dem
// Streifen, einer, der dort beginnt, dahinter.
func TestZuschneidenMitStreifen(t *testing.T) {
	for _, fall := range []struct {
		von, bis float64 // Ansicht, waagrecht
		soll     [2]int  // Original
	}{
		{40, 60, [2]int{40, 110}},
		{10, 50, [2]int{10, 50}},
		{50, 80, [2]int{100, 130}},
	} {
		s := verlaufSitzung(t)
		if err := s.D.SetzeStreifen([]document.Streifen{{Zeilen: false, Von: 50, Bis: 100}}); err != nil {
			t.Fatal(err)
		}
		s.Einpassen()
		s.SetzeWerkzeug(WerkzeugZuschneiden)
		zieh(s, fv(s, fall.von, 10), fv(s, fall.bis, 90))
		if z := s.D.Zuschnitt(); z.Min.X != fall.soll[0] || z.Max.X != fall.soll[1] || z.Min.Y != 10 || z.Max.Y != 90 {
			t.Errorf("Ansicht %v..%v: Zuschnitt %v, Soll x %v", fall.von, fall.bis, z, fall.soll)
		}
	}
}

// ADR-0023 Punkt 3: Senkrecht gezogen nimmt Zeilen heraus, waagrecht Spalten; das Band reicht beim Ziehen über das
// ganze Bild; ein Schritt; Naht an der Stelle.
func TestStauchenRichtung(t *testing.T) {
	s := verlaufSitzung(t)
	if !s.SetzeWerkzeug(WerkzeugStauchen) || s.Voll() != AblehnungKeine {
		t.Fatal("Werkzeug")
	}
	s.Druecken(f(30, 20.3), TastePrimaer)
	s.Ziehen(f(35, 39.6))
	if a, b, ok := s.Rahmen(); !ok || a != f(0, 20) || b != f(200, 40) {
		t.Errorf("Band beim Ziehen: %v %v %v", a, b, ok)
	}
	s.Loslassen(f(35, 39.6))
	if st := s.D.Streifen(); !reflect.DeepEqual(st, []document.Streifen{{Zeilen: true, Von: 20, Bis: 40}}) {
		t.Fatalf("Streifen %v", st)
	}
	if g := s.D.Groesse(); g != image.Pt(200, 80) {
		t.Errorf("Ansicht %v", g)
	}
	if n := s.D.Naehte(); !reflect.DeepEqual(n, []document.Naht{{Zeilen: true, Lage: 20}}) {
		t.Errorf("Naehte %v", n)
	}
	if s.BildAenderung() != 1 {
		t.Errorf("BildAenderung %d, Soll 1", s.BildAenderung())
	}
	if n := schritte(t, s); n != 1 {
		t.Errorf("%d Schritte, Soll 1", n)
	}
	// Waagrecht: Spalten.
	zieh(s, fv(s, 100, 10), fv(s, 150, 30))
	if st := s.D.Streifen(); len(st) != 2 || st[0] != (document.Streifen{Zeilen: false, Von: 100, Bis: 150}) {
		t.Errorf("Spalten: %v", st)
	}
}

// ADR-0023 Punkt 3: Überlappende und anstoßende Streifen verschmelzen, es entsteht keine doppelte Naht.
func TestStauchenVerschmilzt(t *testing.T) {
	s := verlaufSitzung(t)
	if err := s.D.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 20, Bis: 40}}); err != nil {
		t.Fatal(err)
	}
	s.Einpassen()
	s.SetzeWerkzeug(WerkzeugStauchen)
	zieh(s, fv(s, 50, 15), fv(s, 52, 25)) // über die Naht bei 20: Original 15 bis 45
	if st := s.D.Streifen(); !reflect.DeepEqual(st, []document.Streifen{{Zeilen: true, Von: 15, Bis: 45}}) {
		t.Errorf("überlappend: %v", st)
	}
	zieh(s, fv(s, 50, 5), fv(s, 52, 15)) // endet an der Naht: Original 5 bis 15, stößt an
	if st := s.D.Streifen(); !reflect.DeepEqual(st, []document.Streifen{{Zeilen: true, Von: 5, Bis: 45}}) {
		t.Errorf("anstoßend: %v", st)
	}
	if n := s.D.Naehte(); len(n) != 1 {
		t.Errorf("Naehte %v", n)
	}
	got := Verschmolzen([]document.Streifen{{Von: 10, Bis: 20}, {Von: 20, Bis: 30}, {Zeilen: true, Von: 5, Bis: 8},
		{Von: 50, Bis: 60}}, document.Streifen{Von: 25, Bis: 40})
	want := []document.Streifen{{Zeilen: true, Von: 5, Bis: 8}, {Von: 50, Bis: 60}, {Von: 10, Bis: 40}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Verschmolzen %v, Soll %v", got, want)
	}
}

// Ein Band über die ganze Ansicht wird abgelehnt (Meldung, nichts geändert); ebenso eins über MaxKacheln.
func TestStauchenAbgelehnt(t *testing.T) {
	s := verlaufSitzung(t)
	s.SetzeWerkzeug(WerkzeugStauchen)
	zieh(s, f(5, 0), f(6, 300))
	if m := s.NimmAnsichtMeldung(); m != AnsichtLeer || s.Geaendert() {
		t.Errorf("ganze Höhe: Meldung %v, geändert %v", m, s.Geaendert())
	}
	if s.NimmAnsichtMeldung() != AnsichtKeine {
		t.Error("Meldung nicht einmalig")
	}

	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 400, 400)))
	if err != nil {
		t.Fatal(err)
	}
	var st []document.Streifen
	for i := 0; i < 63; i++ { // 64 × 64 = MaxKacheln
		st = append(st, document.Streifen{Von: 2*i + 1, Bis: 2*i + 2}, document.Streifen{Zeilen: true, Von: 2*i + 1, Bis: 2*i + 2})
	}
	if err := d.SetzeStreifen(st); err != nil {
		t.Fatal(err)
	}
	s = NeueSitzung(d)
	s.SetzeFenster(image.Pt(800, 800))
	s.SetzeWerkzeug(WerkzeugStauchen)
	vor := s.D.Stand()
	zieh(s, fv(s, 250, 200), fv(s, 300, 205))
	if m := s.NimmAnsichtMeldung(); m != AnsichtZuviele || s.D.Stand() != vor {
		t.Errorf("über MaxKacheln: Meldung %v, Stand geändert %v", m, s.D.Stand() != vor)
	}
	s.Druecken(fv(s, 250, 200), TastePrimaer) // ein neuer Druck löscht eine nicht abgeholte Meldung
	s.Abbrechen()
}

// ADR-0023 Punkt 4: Doppelklick auf eine Naht mit dem Werkzeug Stauchen holt alle Streifen dieser Naht zurück (ein
// Schritt); daneben wirkt er wie ein Druck.
func TestNahtEntfernen(t *testing.T) {
	s := verlaufSitzung(t)
	if err := s.D.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 20, Bis: 40}, {Von: 50, Bis: 60},
		{Von: 60, Bis: 70}}); err != nil {
		t.Fatal(err)
	}
	s.Einpassen()
	s.SetzeWerkzeug(WerkzeugStauchen)
	vor := s.D.Stand()
	s.Doppelklick(fv(s, 100, 70)) // weit weg: kein Streifen weg
	s.Loslassen(fv(s, 100, 70))
	if len(s.D.Streifen()) != 3 || s.D.Stand() != vor {
		t.Fatalf("Doppelklick neben der Naht: %v", s.D.Streifen())
	}
	s.Doppelklick(fv(s, 52, 30)) // 2 Pixel neben der Spaltennaht bei 50 (zwei anstoßende Streifen)
	if st := s.D.Streifen(); !reflect.DeepEqual(st, []document.Streifen{{Zeilen: true, Von: 20, Bis: 40}}) {
		t.Errorf("nach Doppelklick: %v", st)
	}
	if n := schritte(t, s); n != 2 { // SetzeStreifen oben und das Entfernen
		t.Errorf("%d Schritte, Soll 2", n)
	}
	s.SetzeWerkzeug(WerkzeugZuschneiden)
	s.Doppelklick(fv(s, 100, 20)) // anderes Werkzeug: Naht bleibt
	s.Loslassen(fv(s, 100, 20))
	if len(s.D.Streifen()) != 1 {
		t.Error("Doppelklick mit Zuschneiden hat eine Naht entfernt")
	}
}

// ADR-0023 Punkt 7: Ein Objekt ganz im Verborgenen ist nicht zu greifen, eine verborgene Auswahl endet, die Meldung
// nennt es; Rückgängig bringt es zurück. Das Objekt selbst bleibt unverändert.
func TestVerborgeneObjekte(t *testing.T) {
	s, k, p := sitzung(t) // Kasten (10,10)-(50,50) gefüllt, Pfeil (100,50)-(180,50)
	kasten, _ := s.D.Objekt(k)
	s.Auswahl = k
	s.SetzeWerkzeug(WerkzeugStauchen)
	// Zeilen 5 bis 55: der Kasten liegt ganz darin. Begonnen wird weit weg von seinen Griffen (die gingen vor).
	zieh(s, f(75, 5), f(76, 55))
	if st := s.D.Streifen(); len(st) != 1 {
		t.Fatalf("Streifen %v", st)
	}
	if m := s.NimmAnsichtMeldung(); m != AnsichtVerborgen || s.VerborgeneObjekte() != 1 {
		t.Errorf("Meldung %v, verborgen %d", m, s.VerborgeneObjekte())
	}
	if s.Auswahl != 0 {
		t.Error("verborgene Auswahl bleibt")
	}
	if o, _ := s.D.Objekt(k); o != kasten {
		t.Error("Stauchen hat den Kasten geändert")
	}
	s.SetzeWerkzeug(WerkzeugAuswahl)
	s.Druecken(fv(s, 30, 5), TastePrimaer) // auf der Naht über dem Kasten
	s.Loslassen(fv(s, 30, 5))
	if s.Auswahl != 0 {
		t.Errorf("verborgener Kasten getroffen (Auswahl %d)", s.Auswahl)
	}
	s.Druecken(fv(s, 140, 5), TastePrimaer) // der Pfeil (y 50 → Ansicht 5) bleibt greifbar
	s.Loslassen(fv(s, 140, 5))
	if s.Auswahl != p {
		t.Errorf("Pfeil nicht getroffen (Auswahl %d)", s.Auswahl)
	}
	if !s.Rueckgaengig() || s.VerborgeneObjekte() != 0 {
		t.Error("Rückgängig bringt den Kasten nicht zurück")
	}
	// Zuschneiden ohne verborgene Objekte meldet nichts.
	s.SetzeWerkzeug(WerkzeugZuschneiden)
	zieh(s, fv(s, 5, 5), fv(s, 195, 95))
	if m := s.NimmAnsichtMeldung(); m != AnsichtKeine {
		t.Errorf("Meldung %v ohne verborgene Objekte", m)
	}
}

// F-182: Liegt ein Streifen am Rand der Ansicht, ist ein Rahmen über die ganze Ansicht trotzdem „nichts zu tun“ — kein
// unsichtbarer Rückgängig-Schritt, kein Zuschnitt, den der Doppelklick danach aufheben könnte.
func TestZuschneidenGanzeAnsichtMitRandstreifen(t *testing.T) {
	for _, st := range []document.Streifen{{Von: 0, Bis: 10}, {Zeilen: true, Von: 90, Bis: 100}} {
		s := verlaufSitzung(t)
		if err := s.D.SetzeStreifen([]document.Streifen{st}); err != nil {
			t.Fatal(err)
		}
		if err := s.D.VergissVerlauf(); err != nil {
			t.Fatal(err)
		}
		s.Einpassen()
		vor, z := s.D.Stand(), s.D.Zuschnitt()
		g := s.D.Groesse()
		s.SetzeWerkzeug(WerkzeugZuschneiden)
		zieh(s, fv(s, 0, 0), fv(s, float64(g.X), float64(g.Y)))
		if s.D.Stand() != vor || s.D.Zuschnitt() != z || s.BildAenderung() != 0 {
			t.Errorf("Streifen %v: Stand geändert %v, Zuschnitt %v (Soll %v)", st, s.D.Stand() != vor, s.D.Zuschnitt(), z)
		}
		if s.HebeZuschnittAuf() {
			t.Errorf("Streifen %v: Zuschnitt zum Aufheben entstanden", st)
		}
	}
}

// F-184: Die Meldung „verborgen“ kommt nur, wenn mehr Beschriftungen verborgen sind als vorher — nach Zuschneiden und
// Stauchen wie nach Wiederholen; ein Zuschnitt, der nichts Neues verbirgt, meldet nichts.
func TestMeldungVerborgenNurBeiZuwachs(t *testing.T) {
	s := verlaufSitzung(t) // 200×100, ohne Objekte
	if _, err := s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(150, 70), document.Pt(180, 90)),
		Strich: document.Strich{Farbe: color.NRGBA{R: 255, A: 255}, Breite: 2}}); err != nil {
		t.Fatal(err)
	}
	s.SetzeWerkzeug(WerkzeugZuschneiden)
	zieh(s, fv(s, 0, 0), fv(s, 120, 60)) // verbirgt den Kasten
	if m := s.NimmAnsichtMeldung(); m != AnsichtVerborgen || s.VerborgeneObjekte() != 1 {
		t.Fatalf("1. Zuschnitt: Meldung %v, verborgen %d", m, s.VerborgeneObjekte())
	}
	zieh(s, fv(s, 0, 0), fv(s, 80, 40)) // verbirgt nichts Neues
	if m := s.NimmAnsichtMeldung(); m != AnsichtKeine {
		t.Errorf("2. Zuschnitt ohne Zuwachs: Meldung %v", m)
	}
	if !s.Rueckgaengig() || !s.Rueckgaengig() || s.VerborgeneObjekte() != 0 {
		t.Fatalf("2× Rückgängig: verborgen %d", s.VerborgeneObjekte())
	}
	if m := s.NimmAnsichtMeldung(); m != AnsichtKeine {
		t.Errorf("Rückgängig macht sichtbar, meldet aber %v", m)
	}
	if !s.Wiederholen() {
		t.Fatal("Wiederholen")
	}
	if m := s.NimmAnsichtMeldung(); m != AnsichtVerborgen || s.VerborgeneObjekte() != 1 {
		t.Errorf("Wiederholen: Meldung %v, verborgen %d", m, s.VerborgeneObjekte())
	}
	if !s.Wiederholen() {
		t.Fatal("2. Wiederholen")
	}
	if m := s.NimmAnsichtMeldung(); m != AnsichtKeine {
		t.Errorf("2. Wiederholen ohne Zuwachs: Meldung %v", m)
	}
}

// Die Zackenkante und die neue Sichtbarkeitsregel ändern nichts am Original: Zuschneiden und Stauchen bleiben über
// Rückgängig und Wiederholen bitgleich zum Stand davor.
func TestZuschnittUndStreifenNichtZerstoerend(t *testing.T) {
	s := verlaufSitzung(t)
	if _, err := s.D.Hinzufuegen(document.Text{Ort: document.Pt(5, 5), Inhalt: "x", Schriftgrad: 10,
		Farbe: color.NRGBA{A: 255}}); err != nil {
		t.Fatal(err)
	}
	if err := s.D.VergissVerlauf(); err != nil {
		t.Fatal(err)
	}
	vorher, objekte := s.D.Original(), s.D.Objekte()
	s.SetzeWerkzeug(WerkzeugZuschneiden)
	zieh(s, f(50, 50), f(150, 90))
	s.SetzeWerkzeug(WerkzeugStauchen)
	zieh(s, fv(s, 20, 10), fv(s, 40, 12))
	for s.Rueckgaengig() {
	}
	for s.Wiederholen() {
	}
	for s.Rueckgaengig() {
	}
	if !bytes.Equal(s.D.Original().Pix, vorher.Pix) || !reflect.DeepEqual(s.D.Objekte(), objekte) ||
		s.D.Zuschnitt() != image.Rect(0, 0, 200, 100) || len(s.D.Streifen()) != 0 {
		t.Error("Stand nach Rückgängig weicht ab")
	}
}
