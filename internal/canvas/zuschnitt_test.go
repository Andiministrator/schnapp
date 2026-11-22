// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"bytes"
	"image"
	"image/color"
	"reflect"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
)

// fv rechnet einen Ansichtspunkt mit der aktuellen Abbildung der Sitzung ins Fenster (nach Zuschneiden und Stauchen
// passt der Editor neu ein, die festen Werte von f gelten dann nicht mehr).
func fv(s *Sitzung, x, y float64) document.Punkt { return s.Abb.ZuFenster(document.Pt(x, y)) }

// ADR-0023 Punkte 1 und 5, Nachtrag 5: Ein Rahmen mit dem Werkzeug Zuschneiden setzt beim Loslassen den Zuschnitt (auf
// ganze Ansichtspixel gerundet), ändert weder Original noch Objekte, ist ein Rückgängig-Schritt und passt neu ein —
// beim Zuschneiden die Vollansicht, die deshalb stehen bleibt; Rückgängig stellt das ganze Bild wieder her.
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
	if s.BildAenderung() != 2 || !s.Geaendert() { // 1: Vollansicht beim Werkzeugwechsel, 2: Zuschnitt
		t.Errorf("BildAenderung %d, Geaendert %v", s.BildAenderung(), s.Geaendert())
	}
	voll := Einpassen(image.Pt(200, 100), image.Pt(248, 148), Rand)
	if s.Bildflaeche() != image.Rect(-20, -10, 180, 90) || s.Abb.ZuFenster(document.Pt(-20, -10)) != voll.Versatz ||
		s.Abb.Zoom != voll.Zoom {
		t.Errorf("nach dem Zuschneiden steht die Vollansicht nicht still: Fläche %v, %+v", s.Bildflaeche(), s.Abb)
	}
	if n := schritte(t, s); n != 1 {
		t.Errorf("%d Rückgängig-Schritte, Soll 1", n)
	}
	s.Scrollen(f(50, 50), -1) // Nutzer zoomt: Rückgängig passt trotzdem neu ein (die Ansicht hat eine andere Größe)
	vor := s.BildAenderung()
	if !s.Rueckgaengig() || s.D.Zuschnitt() != image.Rect(0, 0, 200, 100) {
		t.Fatal("Rückgängig")
	}
	if s.Abb != voll || s.BildAenderung() != vor+1 {
		t.Errorf("nach Rückgängig: %+v, BildAenderung %d", s.Abb, s.BildAenderung())
	}
	// Mit einem anderen Werkzeug gilt wieder die Ansicht nach dem Zuschnitt (eingepasst, weil Rückgängig einpasst).
	if !s.Wiederholen() || !s.SetzeWerkzeug(WerkzeugAuswahl) || s.Bildflaeche() != image.Rect(0, 0, 101, 51) ||
		s.Abb != Einpassen(image.Pt(101, 51), image.Pt(248, 148), Rand) {
		t.Errorf("nach dem Werkzeugwechsel: Fläche %v, %+v", s.Bildflaeche(), s.Abb)
	}
}

// Unter der Mindestgröße, mit Esc abgebrochen oder als Rahmen über die ganze Ansicht (Griff über den Rand der
// Vollansicht hinaus) ändert Zuschneiden nichts.
func TestZuschneidenOhneWirkung(t *testing.T) {
	s := verlaufSitzung(t)
	s.SetzeWerkzeug(WerkzeugZuschneiden)
	bild := s.BildAenderung()
	zieh(s, f(10, 10), f(14, 80)) // 4 Pixel breit
	s.Druecken(f(10, 10), TastePrimaer)
	s.Ziehen(f(100, 80))
	s.Esc()
	if _, _, ok := s.Rahmen(); ok {
		t.Error("Rahmen nach Esc")
	}
	s.Loslassen(f(100, 80))
	zieh(s, f(-50, -50), f(500, 500)) // beginnt neben dem Bild: nichts (F-150)
	zieh(s, f(0, 0), f(-30, -30))     // Griff links oben nach außen: bleibt die ganze Ansicht
	zieh(s, f(100, 50), f(100, 50))   // Klick ohne Ziehen
	if s.Geaendert() || s.BildAenderung() != bild || s.D.Zuschnitt() != image.Rect(0, 0, 200, 100) {
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
	zieh(s, fv(s, 50, 8), fv(s, 52, 15)) // endet an der Naht: Original 8 bis 15, stößt an (Rest oben ≥ Bandhalbhöhe 7)
	if st := s.D.Streifen(); !reflect.DeepEqual(st, []document.Streifen{{Zeilen: true, Von: 8, Bis: 45}}) {
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
	// Ein Rest unter MinObjekt (hier 4 von 100 Zeilen bei Zoom 1) zählt wie nichts (Echtlauf #92); 18 Zeilen bleiben
	// (je Seite mindestens die Bandhalbhöhe 7, ADR-0023 Nachtrag 4).
	zieh(s, f(5, 2), f(6, 98))
	if m := s.NimmAnsichtMeldung(); m != AnsichtLeer || s.Geaendert() {
		t.Errorf("Rest 4: Meldung %v, geändert %v", m, s.Geaendert())
	}
	zieh(s, f(5, 8), f(6, 90))
	if m := s.NimmAnsichtMeldung(); m == AnsichtLeer || !s.Geaendert() {
		t.Errorf("Rest 18: Meldung %v, geändert %v", m, s.Geaendert())
	}
	if err := s.D.Rueckgaengig(); err != nil {
		t.Fatal(err)
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

// ADR-0023 Nachtrag 1: Die Trefferweite einer Naht wächst mit dem Zackenfaktor. Bei einem Dokument von 2200 × 2200
// (Faktor 2) trifft ein Doppelklick 15 Pixel neben der Naht (Weite 6 + 2·7 = 20); mit der Weite des Faktors 1
// (6 + 7 = 13) läge er außerhalb. 22 Pixel daneben trifft auch bei Faktor 2 nicht.
func TestNahtEntfernenZackenFaktor2(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 2200, 2200)))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]document.Streifen{{Von: 1000, Bis: 1010}}); err != nil {
		t.Fatal(err)
	}
	if got := d.ZackenFaktor(); got != 2 {
		t.Fatalf("Zackenfaktor %d, Soll 2", got)
	}
	s := NeueSitzung(d)
	s.SetzeFenster(image.Pt(2400, 2400))
	s.EinsZuEins()
	if s.Abb.Zoom != 1 {
		t.Fatalf("Zoom %v, Soll 1", s.Abb.Zoom)
	}
	s.SetzeWerkzeug(WerkzeugStauchen)
	// Die Naht liegt in der Ansicht bei x = 1000.
	s.Doppelklick(fv(s, 1022, 500))
	s.Loslassen(fv(s, 1022, 500))
	if len(s.D.Streifen()) != 1 {
		t.Fatalf("22 Pixel neben der Naht hat sie entfernt: %v", s.D.Streifen())
	}
	s.Doppelklick(fv(s, 1015, 500))
	s.Loslassen(fv(s, 1015, 500))
	if len(s.D.Streifen()) != 0 {
		t.Errorf("15 Pixel neben der Naht bei Faktor 2 hat sie nicht entfernt: %v", s.D.Streifen())
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
	zieh(s, f(75, 8), f(76, 55))
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
	s.Druecken(fv(s, 140, 8), TastePrimaer) // der Pfeil (y 50 → Ansicht 8) bleibt greifbar
	s.Loslassen(fv(s, 140, 8))
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
// unsichtbarer Rückgängig-Schritt, kein Zuschnitt. „Ganzes Bild“ entfernt den Randstreifen (F-222).
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
		bild := s.BildAenderung()
		zieh(s, fv(s, 0, 0), fv(s, -20, -20))                                               // Ecke nach außen
		zieh(s, fv(s, float64(g.X), float64(g.Y)), fv(s, float64(g.X+20), float64(g.Y+20))) // Gegenecke nach außen
		if s.D.Stand() != vor || s.D.Zuschnitt() != z || s.BildAenderung() != bild {
			t.Errorf("Streifen %v: Stand geändert %v, Zuschnitt %v (Soll %v)", st, s.D.Stand() != vor, s.D.Zuschnitt(), z)
		}
		// „Ganzes Bild“ holt den Randstreifen zurück: Er hat keine Naht und wäre sonst verloren (F-222).
		if !s.HebeZuschnittAuf() || len(s.D.Streifen()) != 0 || s.D.Zuschnitt() != z {
			t.Errorf("Streifen %v: Ganzes Bild ließ %v, Zuschnitt %v", st, s.D.Streifen(), s.D.Zuschnitt())
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
	zieh(s, fv(s, 10, 10), fv(s, 120, 60)) // verbirgt den Kasten
	if m := s.NimmAnsichtMeldung(); m != AnsichtVerborgen || s.VerborgeneObjekte() != 1 {
		t.Fatalf("1. Zuschnitt: Meldung %v, verborgen %d", m, s.VerborgeneObjekte())
	}
	zieh(s, fv(s, 10, 10), fv(s, 80, 40)) // verbirgt nichts Neues
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

// F-181 (Sollwert): Ein Text mit 30 Zeichen im Grad 20 bei x = 20 endet weit vor x = 400. Zuschnitt ab x = 400: Er
// fehlt im SVG-Export und zählt als verborgen. Gegenprobe Zuschnitt ab x = 200: Er steht im SVG, nichts verborgen.
func TestUnsichtbarerTextNichtImSVG(t *testing.T) {
	for _, fall := range []struct {
		ab        int
		imSVG     bool
		verborgen int
	}{{400, false, 1}, {200, true, 0}} {
		b := image.NewNRGBA(image.Rect(0, 0, 1000, 200))
		for i := range b.Pix {
			b.Pix[i] = 0xFF
		}
		d, err := document.Neu(b)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := d.Hinzufuegen(document.Text{Ort: document.Pt(20, 50), Inhalt: "Kennwort hunter2 nicht zeigen!",
			Schriftgrad: 20, Farbe: color.NRGBA{A: 255}}); err != nil {
			t.Fatal(err)
		}
		s := NeueSitzung(d)
		s.SetzeFenster(image.Pt(1200, 400))
		s.SetzeWerkzeug(WerkzeugZuschneiden)
		zieh(s, fv(s, 0, 100), fv(s, float64(fall.ab), 100)) // linke Kante nach innen (ADR-0023 Nachtrag 5)
		if z := d.Zuschnitt(); z.Min.X != fall.ab {
			t.Fatalf("Zuschnitt %v, Soll ab x = %d", z, fall.ab)
		}
		var svg bytes.Buffer
		if err := fileformat.ExportiereSVG(&svg, d); err != nil {
			t.Fatal(err)
		}
		if im := bytes.Contains(svg.Bytes(), []byte("hunter2")); im != fall.imSVG || s.VerborgeneObjekte() != fall.verborgen {
			t.Errorf("Zuschnitt ab %d: im SVG %v (Soll %v), verborgen %d (Soll %d)", fall.ab, im, fall.imSVG,
				s.VerborgeneObjekte(), fall.verborgen)
		}
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

// F-174: Mit Zuschneiden oder Stauchen zieht ein Zug an der Ecke des zuvor ausgewählten Kastens einen Rahmen statt den
// Griff. Der Werkzeugwechsel hebt die Auswahl auf; Druecken wertet bei diesen Werkzeugen auch dann keine Griffe aus,
// wenn die Auswahl trotzdem gesetzt ist.
func TestGriffeGehenNichtVorRahmen(t *testing.T) {
	for _, w := range []Werkzeug{WerkzeugZuschneiden, WerkzeugStauchen} {
		for _, spaeter := range []bool{false, true} {
			s, k, _ := sitzung(t)
			s.Druecken(f(30, 30), TastePrimaer) // Kasten (10,10)-(50,50) auswählen
			s.Loslassen(f(30, 30))
			if s.Auswahl != k {
				t.Fatalf("Auswahl %d, Soll %d", s.Auswahl, k)
			}
			vorher, _ := s.D.Objekt(k)
			if !s.SetzeWerkzeug(w) {
				t.Fatal("Werkzeug")
			}
			if s.Auswahl != 0 {
				t.Errorf("Werkzeug %v: Auswahl %d blieb, Soll 0", w, s.Auswahl)
			}
			if spaeter {
				s.Auswahl = k
			}
			zieh(s, f(50, 50), f(100, 80)) // an der Ecke unten rechts
			if o, _ := s.D.Objekt(k); !reflect.DeepEqual(o, vorher) {
				t.Errorf("Werkzeug %v: Kasten verändert: %v, vorher %v", w, o, vorher)
			}
			if w == WerkzeugZuschneiden {
				if z := s.D.Zuschnitt(); z != image.Rect(50, 50, 100, 80) {
					t.Errorf("Zuschnitt %v, Soll (50,50)-(100,80)", z)
				}
			} else if n := len(s.D.Streifen()); n != 1 {
				t.Errorf("%d Streifen, Soll 1", n)
			}
		}
	}
}

// F-226 (ADR-0023 Nachtrag 4): Zwischen Naht und Bildrand bleibt mindestens die halbe Bandhöhe (Faktor 1: 7 Pixel). Ein
// Band, das weniger übrig ließe, reicht bis an den Rand und wird dort zum Zuschnitt; ab genau 7 bleibt es eine Naht.
func TestStauchenMindestrestJeSeite(t *testing.T) {
	neu := func() *Sitzung {
		s := verlaufSitzung(t) // 200×100, Faktor 1
		s.SetzeWerkzeug(WerkzeugStauchen)
		return s
	}
	s := neu()
	zieh(s, f(5, 6), f(6, 60)) // oben bleiben 6 < 7
	if st := s.D.Streifen(); len(st) != 0 || s.D.Zuschnitt().Min.Y != 60 {
		t.Errorf("Rest 6 oben: Streifen %v, Zuschnitt %v, Soll Zuschnitt ab Zeile 60", st, s.D.Zuschnitt())
	}
	s = neu()
	zieh(s, f(5, 40), f(6, 94)) // unten bleiben 6 < 7
	if st := s.D.Streifen(); len(st) != 0 || s.D.Zuschnitt().Max.Y != 40 {
		t.Errorf("Rest 6 unten: Streifen %v, Zuschnitt %v, Soll Zuschnitt bis Zeile 40", st, s.D.Zuschnitt())
	}
	s = neu()
	zieh(s, f(5, 7), f(6, 40)) // oben genau 7: bleibt eine Naht
	if st := s.D.Streifen(); !reflect.DeepEqual(st, []document.Streifen{{Zeilen: true, Von: 7, Bis: 40}}) {
		t.Errorf("Rest 7: Streifen %v", st)
	}
}

// F-227: Ein Schwärzungsrahmen unter der Mindestgröße tut nichts, sagt das aber einmal; ein bloßer Klick nicht.
func TestSchwaerzenRahmenZuKleinMeldung(t *testing.T) {
	s := verlaufSitzung(t)
	s.SetzeWerkzeug(WerkzeugSchwaerzen)
	zieh(s, f(10, 10), f(14, 40))
	if !s.NimmRahmenZuKlein() || s.NimmRahmenZuKlein() {
		t.Error("Meldung „Rahmen zu klein“ fehlt oder kommt mehrfach")
	}
	if s.Geaendert() {
		t.Error("zu kleiner Rahmen hat geschwärzt")
	}
	s.Druecken(f(50, 50), TastePrimaer)
	s.Loslassen(f(50, 50))
	if s.NimmRahmenZuKlein() {
		t.Error("ein bloßer Klick meldet „zu klein“")
	}
	zieh(s, f(10, 10), f(80, 60))
	if s.NimmRahmenZuKlein() || !s.Geaendert() {
		t.Error("ausreichender Rahmen: Meldung oder nicht geschwärzt")
	}
}
