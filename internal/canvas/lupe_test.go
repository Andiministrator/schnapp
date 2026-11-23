// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"image/color"
	"math"
	"slices"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// lupenSitzung: leeres Bild 400×200, Fenster mit Zoom 1 und Versatz 24 (f), Einrasten aus, damit Lagen genau sind.
func lupenSitzung(t *testing.T) *Sitzung {
	t.Helper()
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 400, 200)))
	if err != nil {
		t.Fatal(err)
	}
	s := NeueSitzung(d)
	s.SetzeFenster(image.Pt(448, 248))
	if s.Abb.Zoom != 1 || !nah(s.Abb.Versatz, document.Pt(24, 24)) {
		t.Fatalf("Abbildung %+v", s.Abb)
	}
	s.SchalteEinrasten()
	return s
}

func rechteck(x0, y0, x1, y1 float64) document.Rechteck {
	return document.R(document.Pt(x0, y0), document.Pt(x1, y1))
}

func naheR(a, b document.Rechteck) bool { return nah(a.Min, b.Min) && nah(a.Max, b.Max) }

// legeLupe zieht mit dem Werkzeug Lupe eine Quelle auf und liefert die Lupe.
func legeLupe(t *testing.T, s *Sitzung, a, b document.Punkt) document.Lupe {
	t.Helper()
	if !s.SetzeWerkzeug(WerkzeugLupe) {
		t.Fatal("Werkzeug")
	}
	zieh(s, f(a.X, a.Y), f(b.X, b.Y))
	l, ok := letztes(t, s).(document.Lupe)
	if !ok {
		t.Fatalf("keine Lupe: %T", letztes(t, s))
	}
	return l
}

// TestLupeAnlegen: Aufziehen wählt die Quelle, das Ziel liegt rechts im Abstand 16 mit Vergrößerung 2; ein Schritt,
// ausgewählt, Strich aus dem Stil. Ein Klick ohne Ziehen legt nichts an.
func TestLupeAnlegen(t *testing.T) {
	s := lupenSitzung(t)
	if !s.SetzeWerkzeug(WerkzeugLupe) {
		t.Fatal("Werkzeug")
	}
	s.Druecken(f(20, 20), TastePrimaer)
	s.Loslassen(f(21, 20))
	if n := len(s.D.Objekte()); n != 0 {
		t.Fatalf("Klick legt %d Objekte an", n)
	}
	l := legeLupe(t, s, document.Pt(20, 20), document.Pt(60, 50))
	if !naheR(l.Quelle, rechteck(20, 20, 60, 50)) || !naheR(l.Ziel, rechteck(76, 20, 156, 80)) {
		t.Errorf("Quelle %v, Ziel %v", l.Quelle, l.Ziel)
	}
	if l.Strich != s.Stil.strich() || l.Strich.Farbe != Beschriftungsfarben[0] {
		t.Errorf("Strich %+v", l.Strich)
	}
	if s.Auswahl != l.ID || schritte(t, s) != 1 {
		t.Errorf("Auswahl %d, Schritte %d", s.Auswahl, schritte(t, s))
	}
	// Zu klein (unter MinObjekt): nichts.
	n := len(s.D.Objekte())
	zieh(s, f(200, 150), f(204, 190))
	if len(s.D.Objekte()) != n {
		t.Error("zu schmale Quelle angelegt")
	}
	// Seitenleiste: Auswahl Lupe mit Strich, Farbe und Vergrößerung, ohne Füllung.
	s.Auswahl = l.ID
	if i := s.Seitenleiste(); !i.Auswahl || i.Werkzeug != WerkzeugLupe || !i.Strich || !i.Farbe || !i.Vergroesserung || i.Text {
		t.Errorf("Seitenleiste %+v", i)
	}
	if s.Fuellbar() {
		t.Error("Lupe hat eine Füllung")
	}
	s.Auswahl = 0
	if i := s.Seitenleiste(); i.Vergroesserung || !i.Strich || !i.Farbe {
		t.Errorf("Seitenleiste ohne Auswahl %+v", i)
	}
}

// TestZielPlatz: rechts, sonst links, darunter, darüber, sonst rechts über den Rand; quer an der Kante der Quelle,
// in die Ansicht gerückt.
func TestZielPlatz(t *testing.T) {
	g := image.Pt(400, 200)
	fall := []struct {
		name string
		q    document.Rechteck
		gr   document.Punkt
		want document.Rechteck
	}{
		{"rechts", rechteck(10, 10, 50, 40), document.Pt(80, 60), rechteck(66, 10, 146, 70)},
		{"rechts genau bis an den Rand", rechteck(10, 10, 50, 40), document.Pt(334, 60), rechteck(66, 10, 400, 70)},
		{"links", rechteck(300, 10, 340, 40), document.Pt(80, 60), rechteck(204, 10, 284, 70)},
		{"rechts, quer nach oben gerückt", rechteck(10, 150, 50, 190), document.Pt(80, 80), rechteck(66, 120, 146, 200)},
		{"darunter", rechteck(150, 10, 250, 40), document.Pt(200, 60), rechteck(150, 56, 350, 116)},
		{"darunter, quer nach links gerückt", rechteck(300, 10, 390, 40), document.Pt(300, 60), rechteck(100, 56, 400, 116)},
		{"darüber", rechteck(150, 130, 250, 160), document.Pt(200, 60), rechteck(150, 54, 350, 114)},
		{"nichts passt: rechts", rechteck(100, 50, 300, 150), document.Pt(400, 200), rechteck(316, 50, 716, 250)},
		{"zu breit für die Seiten: darunter", rechteck(150, 0, 250, 10), document.Pt(200, 20), rechteck(150, 26, 350, 46)},
	}
	for _, c := range fall {
		if got := ZielPlatz(c.q, c.gr, 16, g); !naheR(got, c.want) {
			t.Errorf("%s: %v, erwartet %v", c.name, got, c.want)
		}
	}
	// Höher als die Ansicht: weder rechts noch links; breiter als die Ansicht: weder darunter noch darüber.
	if got := ZielPlatz(rechteck(10, 10, 20, 20), document.Pt(20, 300), 16, g); !naheR(got, rechteck(36, 10, 56, 310)) {
		t.Errorf("zu hoch: %v", got)
	}
}

// TestLupeAnlegenAbstandMitFaktor: Der Abstand wächst mit dem KantenFaktor (ADR-0033 Punkt 8).
func TestLupeAnlegenAbstandMitFaktor(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 2400, 2200))) // Faktor 2
	if err != nil {
		t.Fatal(err)
	}
	if d.KantenFaktor() != 2 {
		t.Fatalf("Faktor %d", d.KantenFaktor())
	}
	s := NeueSitzung(d)
	l := s.neueLupe(rechteck(100, 100, 200, 150))
	if !naheR(l.Ziel, rechteck(232, 100, 432, 200)) {
		t.Errorf("Ziel %v", l.Ziel)
	}
	// Mit Zuschnitt: Platz in der Ansicht, Ziel ins Original zurückgerechnet.
	if err := d.SetzeZuschnitt(image.Rect(50, 0, 2400, 2200)); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]document.Streifen{{Von: 220, Bis: 260}}); err != nil {
		t.Fatal(err)
	}
	l = s.neueLupe(rechteck(100, 100, 200, 150))
	// Ansicht: Quelle 50..150, Ziel ab 150 + 32 = 182; die Ansicht zeigt bis 170 die Spalten 50..220, danach folgt
	// Spalte 260: ZuOriginal(182) = 272.
	if !naheR(l.Ziel, rechteck(272, 100, 472, 200)) {
		t.Errorf("Ziel mit Zuschnitt und Streifen %v", l.Ziel)
	}
}

// TestLupeTeileVerschieben: Inneres des Ziels verschiebt nur das Ziel, Inneres der Quelle nur die Quelle, die Linie
// beide; je ein Schritt.
func TestLupeTeileVerschieben(t *testing.T) {
	s := lupenSitzung(t)
	l := legeLupe(t, s, document.Pt(20, 20), document.Pt(60, 50)) // Ziel 76..156 × 20..80
	s.SetzeWerkzeug(WerkzeugAuswahl)
	vor := schritte(t, s)
	zieh(s, f(100, 50), f(110, 60))
	o, _ := s.D.Objekt(l.ID)
	l2 := o.(document.Lupe)
	if !naheR(l2.Quelle, l.Quelle) || !naheR(l2.Ziel, rechteck(86, 30, 166, 90)) {
		t.Errorf("Ziel verschoben: %v %v", l2.Quelle, l2.Ziel)
	}
	zieh(s, f(30, 30), f(25, 40))
	o, _ = s.D.Objekt(l.ID)
	l3 := o.(document.Lupe)
	if !naheR(l3.Quelle, rechteck(15, 30, 55, 60)) || !naheR(l3.Ziel, l2.Ziel) {
		t.Errorf("Quelle verschoben: %v %v", l3.Quelle, l3.Ziel)
	}
	// Linie: Mitten (35,45) und (126,60); bei x = 70 liegt sie bei y = 45 + 35·15/91.
	y := 45 + 35*15/91.0
	zieh(s, f(70, y), f(80, y+5))
	o, _ = s.D.Objekt(l.ID)
	l4 := o.(document.Lupe)
	if !naheR(l4.Quelle, rechteck(25, 35, 65, 65)) || !naheR(l4.Ziel, rechteck(96, 35, 176, 95)) {
		t.Errorf("Linie verschiebt beide: %v %v", l4.Quelle, l4.Ziel)
	}
	if n := schritte(t, s); n != vor+3 {
		t.Errorf("%d Schritte, erwartet %d", n, vor+3)
	}
	// Ein Druck daneben trifft die Lupe nicht.
	if _, ok := Treffer(s.D.Objekte(), document.Pt(300, 150), 1); ok {
		t.Error("Treffer daneben")
	}
}

// TestLupeTeilBleibtSichtbar: Ein Teil lässt sich nicht ganz aus dem Bild schieben.
func TestLupeTeilBleibtSichtbar(t *testing.T) {
	s := lupenSitzung(t)
	l := legeLupe(t, s, document.Pt(20, 20), document.Pt(60, 50))
	s.SetzeWerkzeug(WerkzeugAuswahl)
	s.Druecken(f(100, 50), TastePrimaer)
	s.Ziehen(f(100, 50+1000))
	s.Loslassen(f(100, 50+1000))
	o, _ := s.D.Objekt(l.ID)
	if z := o.(document.Lupe).Ziel; z.Min.Y > 200-MinSichtbar+1e-6 {
		t.Errorf("Ziel aus dem Bild geschoben: %v", z)
	}
}

// TestLupeGriffe: Ecken des Ziels halten das Seitenverhältnis der Quelle (begrenzt auf 1 bis 16), Ecken der Quelle
// ändern sie bei gleicher Vergrößerung, das Ziel wächst von links oben.
func TestLupeGriffe(t *testing.T) {
	l := document.Lupe{Quelle: rechteck(20, 20, 60, 50), Ziel: rechteck(76, 20, 156, 80)}
	if g := Griffe(l); len(g) != 8 || g[0] != (Griff{GriffQuelleNW, l.Quelle.Min}) || g[6] != (Griff{GriffZielSO, l.Ziel.Max}) {
		t.Fatalf("Griffe %v", g)
	}
	z := ZieheGriff(l, GriffZielSO, document.Pt(196, 85)).(document.Lupe) // dx 120 → 3, dy 65 → 2,17: k = 3
	if !naheR(z.Ziel, rechteck(76, 20, 196, 110)) || !naheR(z.Quelle, l.Quelle) {
		t.Errorf("Ziel SO: %v", z.Ziel)
	}
	z = ZieheGriff(l, GriffZielNW, document.Pt(140, 75)).(document.Lupe) // k = 0,4 → 1, links oben von SO (156,80)
	if !naheR(z.Ziel, rechteck(116, 50, 156, 80)) {
		t.Errorf("Ziel NW, kleinster Faktor: %v", z.Ziel)
	}
	z = ZieheGriff(l, GriffZielSO, document.Pt(5000, 20)).(document.Lupe) // k > 16 → 16
	if !naheR(z.Ziel, rechteck(76, 20, 76+640, 20+480)) {
		t.Errorf("Ziel, größter Faktor: %v", z.Ziel)
	}
	z = ZieheGriff(l, GriffZielNO, document.Pt(60, 0)).(document.Lupe) // über die feste Ecke (76,80) nach links: Spiegeln
	if x, y := z.Vergroesserung(); math.Abs(x-y) > 1e-9 || z.Ziel.Max.X != 76 || z.Ziel.Max.Y != 80 {
		t.Errorf("Ziel NO über die Ecke: %v", z.Ziel)
	}
	q := ZieheGriff(l, GriffQuelleSO, document.Pt(70, 60)).(document.Lupe) // Quelle 50×40, Ziel 100×80 ab (76,20)
	if !naheR(q.Quelle, rechteck(20, 20, 70, 60)) || !naheR(q.Ziel, rechteck(76, 20, 176, 100)) {
		t.Errorf("Quelle SO: %v %v", q.Quelle, q.Ziel)
	}
	q = ZieheGriff(l, GriffQuelleNW, document.Pt(60, 50)).(document.Lupe) // auf die feste Ecke: mindestens 1 Pixel
	if !naheR(q.Quelle, rechteck(59, 49, 60, 50)) || !naheR(q.Ziel, rechteck(76, 20, 78, 22)) {
		t.Errorf("Quelle auf die Ecke: %v %v", q.Quelle, q.Ziel)
	}
	q = ZieheGriff(l, GriffQuelleSW, document.Pt(80, 10)).(document.Lupe) // über NO hinaus: neu geordnet
	if !naheR(q.Quelle, rechteck(60, 10, 80, 20)) {
		t.Errorf("Quelle SW gespiegelt: %v", q.Quelle)
	}
	// Ein Griff eines anderen Objekts wirkt nicht auf die Lupe.
	if ZieheGriff(l, GriffNW, document.Pt(0, 0)) != document.Objekt(l) {
		t.Error("Kasten-Griff ändert die Lupe")
	}
	// Über die Sitzung: ein Schritt, Lupe bleibt gewählt.
	s := lupenSitzung(t)
	ll := legeLupe(t, s, document.Pt(20, 20), document.Pt(60, 50))
	vor := schritte(t, s)
	zieh(s, f(156, 80), f(196, 85))
	o, _ := s.D.Objekt(ll.ID)
	if !naheR(o.(document.Lupe).Ziel, rechteck(76, 20, 196, 110)) || schritte(t, s) != vor+1 {
		t.Errorf("Griff über die Sitzung: %v", o.(document.Lupe).Ziel)
	}
}

// TestLupeVergroesserung: Stufen 2×, 3×, 4× über − und +, Zwischenwerte springen zur nächsten Stufe; jede Änderung ist
// ein Schritt und hält die linke obere Ecke des Ziels.
func TestLupeVergroesserung(t *testing.T) {
	s := lupenSitzung(t)
	l := legeLupe(t, s, document.Pt(20, 20), document.Pt(60, 50))
	if k, ok := s.Vergroesserung(); !ok || k != 2 {
		t.Fatalf("Vergrößerung %v %v", k, ok)
	}
	if _, ok := s.VergroesserungStufe(-1); ok {
		t.Error("unter 2×")
	}
	vor := schritte(t, s)
	for _, want := range []float64{3, 4} {
		k, ok := s.VergroesserungStufe(1)
		if !ok || k != want || !s.SetzeVergroesserung(k) {
			t.Fatalf("Stufe %v: %v %v", want, k, ok)
		}
	}
	o, _ := s.D.Objekt(l.ID)
	if z := o.(document.Lupe).Ziel; !naheR(z, rechteck(76, 20, 76+160, 20+120)) {
		t.Errorf("4×: %v", z)
	}
	if _, ok := s.VergroesserungStufe(1); ok {
		t.Error("über 4×")
	}
	if s.SetzeVergroesserung(4) || schritte(t, s) != vor+2 {
		t.Errorf("gleiche Vergrößerung ist ein Schritt, oder Schritte %d", schritte(t, s))
	}
	// Zwischenwert 2,5: + geht zu 3, − zu 2.
	o, _ = s.D.Objekt(l.ID)
	if err := s.D.Ersetzen(MitVergroesserung(o.(document.Lupe), 2.5)); err != nil {
		t.Fatal(err)
	}
	if k, _ := s.VergroesserungStufe(1); k != 3 {
		t.Errorf("2,5 + → %v", k)
	}
	if k, _ := s.VergroesserungStufe(-1); k != 2 {
		t.Errorf("2,5 − → %v", k)
	}
	// Gesperrt: keine Änderung. Ohne Auswahl: nichts.
	o, _ = s.D.Objekt(l.ID)
	e := o.Ebenenwerte()
	e.Gesperrt = true
	if err := s.D.Ersetzen(document.MitEbene(o, e)); err != nil {
		t.Fatal(err)
	}
	if s.SetzeVergroesserung(3) {
		t.Error("gesperrte Lupe geändert")
	}
	s.Auswahl = 0
	if _, ok := s.Vergroesserung(); ok || s.SetzeVergroesserung(3) {
		t.Error("ohne Auswahl")
	}
}

// TestLupeFarbeUndBreite: Farbe nur im Vordergrund (Strich), Strichstärke wie beim Kasten.
func TestLupeFarbeUndBreite(t *testing.T) {
	s := lupenSitzung(t)
	l := legeLupe(t, s, document.Pt(20, 20), document.Pt(60, 50))
	blau := color.NRGBA{B: 255, A: 255}
	if !s.SetzeFarbe(blau) || !s.SetzeBreite(8) {
		t.Fatal("Farbe oder Breite nicht gesetzt")
	}
	o, _ := s.D.Objekt(l.ID)
	if st := o.(document.Lupe).Strich; st.Farbe != blau || st.Breite != 8 || s.Breite() != 8 || s.Farbe(FeldVordergrund) != blau {
		t.Errorf("Strich %+v", st)
	}
	if UmgefaerbtIn(o, FeldFuellung, color.NRGBA{R: 1, A: 1}) != o {
		t.Error("Füllung färbt die Lupe")
	}
}

// TestLupeEinrasten: Als Ziel für andere zählen Quelle und Ziel einzeln, nicht ihre Hülle (ADR-0033 Punkt 9); beim
// Verschieben des Ziels rastet es an einem Kasten ein.
func TestLupeEinrasten(t *testing.T) {
	s := lupenSitzung(t)
	l := legeLupe(t, s, document.Pt(20, 20), document.Pt(60, 50)) // Quelle 20..60, Ziel 76..156 (x)
	s.SchalteEinrasten()
	xs, _ := s.rastZiele(0)
	for _, x := range []float64{20, 40, 60, 76, 116, 156} {
		if !slices.Contains(xs, x) {
			t.Errorf("Linie x = %v fehlt: %v", x, xs)
		}
	}
	if mitte := (l.Grenzen().Min.X + l.Grenzen().Max.X) / 2; slices.Contains(xs, mitte) {
		t.Errorf("Mitte der Hülle %v ist Ziel", mitte)
	}
	k, err := s.D.Hinzufuegen(document.Kasten{Bereich: rechteck(200, 120, 260, 180), Strich: rot})
	if err != nil {
		t.Fatal(err)
	}
	_ = k
	s.SetzeWerkzeug(WerkzeugAuswahl)
	// Ziel (Breite 80) nach rechts: linke Kante bei 76 + 120 = 196, rastet an der Außenkante des Kastens ein
	// (200 − halber Strich 2 = 198).
	s.Druecken(f(100, 50), TastePrimaer)
	s.Ziehen(f(220, 50))
	if h := s.Hilfslinien(); len(h) != 1 || h[0] != (Hilfslinie{Lage: 198}) {
		t.Errorf("Hilfslinien %v", h)
	}
	s.Loslassen(f(220, 50))
	o, _ := s.D.Objekt(l.ID)
	if z := o.(document.Lupe).Ziel; z.Min.X != 198 {
		t.Errorf("Ziel nicht eingerastet: %v", z)
	}
}
