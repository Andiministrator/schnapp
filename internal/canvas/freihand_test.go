// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"math"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/settings"
)

// ziehZug drückt bei pts[0], zieht über alle Punkte und lässt am letzten los.
func ziehZug(s *Sitzung, pts ...document.Punkt) {
	s.Druecken(pts[0], TastePrimaer)
	for _, p := range pts[1:] {
		s.Ziehen(p)
	}
	s.Loslassen(pts[len(pts)-1])
}

// bogen liefert Fensterpunkte eines Halbkreises um (100, 60) mit Radius 30, n Schritte.
func bogen(n int) []document.Punkt {
	var r []document.Punkt
	for i := 0; i <= n; i++ {
		w := math.Pi * float64(i) / float64(n)
		r = append(r, f(100+30*math.Cos(w), 60-30*math.Sin(w)))
	}
	return r
}

// ADR-0049 Punkte 3 und 4: ein freier Zug wird ein Freihand-Strich, beim Loslassen vereinfacht und geglättet;
// Umschalt zieht eine gerade Strecke; ein zu kurzer Zug bleibt nicht liegen.
func TestFreihandZiehen(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugFreihand)
	zug := bogen(200)
	ziehZug(s, zug...)
	fh, ok := letztes(t, s).(document.Freihand)
	if !ok {
		t.Fatalf("kein Freihand-Strich: %+v", letztes(t, s))
	}
	pts := fh.Punkte()
	// Roh rund 48 Punkte (alle 2 Pixel), vereinfacht rund 16, zweimal geglättet 58, wieder vereinfacht rund 17
	// (ADR-0049 Nachtrag 1, F-1036): höchstens so viele wie roh, und geglättet heißt, die inneren Punkte sind keine
	// Rohpunkte mehr (ohne Glätten oder mit dem Rückfall auf den ungeglätteten Zug wären sie es alle).
	roh := map[document.Punkt]bool{}
	for _, p := range zug {
		roh[document.Pt(math.Round(s.original(p).X*100)/100, math.Round(s.original(p).Y*100)/100)] = true
	}
	auf := 0
	for _, p := range pts[1 : len(pts)-1] {
		if roh[p] {
			auf++
		}
	}
	if len(pts) < 10 || len(pts) > 48 || auf > len(pts)/4 {
		t.Errorf("Punkte nach dem Glätten: %d (Soll 10 bis 48), davon %d innere auf Rohpunkten", len(pts), auf)
	}
	if pts[0] != s.original(zug[0]) || pts[len(pts)-1] != s.original(zug[len(zug)-1]) {
		t.Errorf("Anfang/Ende %v %v", pts[0], pts[len(pts)-1])
	}
	for _, p := range pts { // auf dem Bogen, höchstens wenig innen (Chaikin schneidet Ecken)
		if r := p.Minus(document.Pt(100, 60)).Laenge(); r > 30.01 || r < 29 {
			t.Errorf("Punkt %v mit Radius %.2f", p, r)
			break
		}
	}
	if fh.Multipliziert() || fh.Strich.Breite != 6 || s.Auswahl != fh.ID {
		t.Errorf("Stil %+v, Auswahl %d", fh, s.Auswahl)
	}
	n := len(s.D.Objekte())
	s.Umschalt = true
	ziehZug(s, f(20, 20), f(40, 70), f(60, 30))
	if g := letztes(t, s).(document.Freihand); len(g.Punkte()) != 2 || g.Punkte()[1] != document.Pt(60, 30) {
		t.Errorf("Umschalt: %v", g.Punkte())
	}
	s.Umschalt = false
	ziehZug(s, f(150, 80), f(152, 80), f(154, 81)) // über der Ziehschwelle, unter MinObjekt (Länge rund 4)
	if len(s.D.Objekte()) != n+1 {
		t.Error("zu kurzer Zug blieb liegen")
	}
}

// ADR-0049 Punkt 5: Mit „Freehand“ zieht der Textmarker einen freien multiplizierenden Strich, Umschalt waagerecht;
// er gilt als Textmarker. Der Schalter gehört zum Stil des Textmarkers.
func TestTextmarkerFrei(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugTextmarker)
	if s.TextmarkerFrei() || !s.Seitenleiste().Freihand {
		t.Fatal("Voreinstellung oder Kästchen")
	}
	if !s.SchalteTextmarkerFrei() || !s.TextmarkerFrei() {
		t.Fatal("Schalter wirkt nicht")
	}
	ziehZug(s, bogen(60)...)
	fh, ok := letztes(t, s).(document.Freihand)
	if !ok || !fh.Multipliziert() || fh.Strich.Breite != TextmarkerBreite {
		t.Fatalf("freier Textmarker %+v", letztes(t, s))
	}
	if w, _ := objektWerkzeug(fh); w != WerkzeugTextmarker {
		t.Errorf("objektWerkzeug %v", w)
	}
	if !s.Seitenleiste().Freihand { // F-1039: der Strich wählt sich selbst, das Kästchen bleibt
		t.Error("Kästchen weg nach dem Strich")
	}
	if st, _ := objektStandOhneSchatten(fh); st.Art != SchnellTextmarker {
		t.Errorf("Schnellstil-Art %v", st.Art)
	}
	s.Umschalt = true
	ziehZug(s, f(20, 20), f(60, 50))
	if g := letztes(t, s).(document.Freihand); len(g.Punkte()) != 2 || g.Punkte()[0].Y != g.Punkte()[1].Y {
		t.Errorf("Umschalt: %v", g.Punkte())
	}
	s.SetzeWerkzeug(WerkzeugFreihand)
	if s.Seitenleiste().Freihand {
		t.Error("Kästchen beim Werkzeug Freihand")
	}
	// tools.ini trägt den Schalter beim Textmarker.
	var gefunden bool
	for _, e := range s.WerkzeugStile() {
		if e.Art == settings.ArtTextmarker {
			gefunden = e.Freihand
		}
	}
	if !gefunden {
		t.Error("freehand nicht in den Werkzeugstilen")
	}
	s2, _, _ := sitzung(t)
	s2.SetzeWerkzeugStile(s.WerkzeugStile())
	s2.SetzeWerkzeug(WerkzeugTextmarker)
	if !s2.TextmarkerFrei() {
		t.Error("Schalter nicht übernommen")
	}
}

// Verschieben, Treffer, Strich-Setter, Schnellstile und Stil je Werkzeug kennen den Freihand-Strich.
func TestFreihandBearbeiten(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugFreihand)
	ziehZug(s, f(20, 80), f(40, 90), f(60, 80), f(80, 90))
	fh := letztes(t, s).(document.Freihand)
	if !trifft(fh, document.Pt(40, 90), 1) || trifft(fh, document.Pt(40, 60), 1) {
		t.Error("Treffer")
	}
	v := Verschiebe(fh, document.Pt(5, -3)).(document.Freihand)
	if v.Punkte()[0] != fh.Punkte()[0].Plus(document.Pt(5, -3)) || fh.Punkte()[0] != document.Pt(20, 80) {
		t.Errorf("Verschiebe %v / %v", v.Punkte()[0], fh.Punkte()[0])
	}
	s.SetzeWerkzeug(WerkzeugAuswahl)
	s.waehleNur(fh.ID)
	if !s.SetzeBreite(14) || s.Breite() != 14 || !s.SetzeStrichart(document.StrichGepunktet) ||
		s.Strichart() != document.StrichGepunktet {
		t.Error("Strich-Setter")
	}
	if !s.SetzeFarbe(TextmarkerRosa) || letztes(t, s).(document.Freihand).Strich.Farbe != TextmarkerRosa {
		t.Error("Farbe")
	}
	ein := EingebauteSchnellstile(SchnellFreihand)
	if len(ein) != len(EingebauteSchnellstile(SchnellPfeil)) || ein[0].Art != SchnellFreihand {
		t.Fatalf("eingebaute Stile %+v", ein)
	}
	if !s.WendeSchnellstilAn(ein[1]) || letztes(t, s).(document.Freihand).Strich.Farbe != ein[1].Farbe {
		t.Error("Schnellstil")
	}
	if werkzeugAbschnitt(WerkzeugFreihand) != settings.ArtFreihand {
		t.Error("Abschnitt in tools.ini")
	}
	if w, _ := objektWerkzeug(fh); w != WerkzeugFreihand || !strichWerkzeug(w) || fuellWerkzeug(w) {
		t.Errorf("Werkzeug %v", w)
	}
}

// F-1043 (Kritik #333): Ungedeckte Zeilen des Freihand-Strichs.

// freiPunkt rechnet einen Ansichtspunkt in Fensterkoordinaten bei beliebigem Zoom.
func freiPunkt(s *Sitzung, x, y float64) document.Punkt { return s.Abb.ZuFenster(document.Pt(x, y)) }

// K-10 und K-12 (Mutanten FH33, FH35): Ein neuer Roh-Punkt kommt erst ab freiAbstand Fensterpixeln zum letzten dazu,
// gemessen in Fensterpixeln, nicht im Original: bei Zoom 1 und bei Zoom 0,5 (Schritte von 1 Fensterpixel = 2 Originalpixel).
func TestFreihandRohpunktAbstandInFensterpixeln(t *testing.T) {
	for _, prozent := range []int{100, 50} {
		s, _, _ := sitzung(t)
		s.SetzeZoom(prozent)
		s.SetzeWerkzeug(WerkzeugFreihand)
		start := freiPunkt(s, 60, 50)
		s.Druecken(start, TastePrimaer)
		for i := 1; i <= 40; i++ { // 40 Schritte zu 1 Fensterpixel
			s.Ziehen(start.Plus(document.Pt(float64(i), 0)))
		}
		// Alle 2 Fensterpixel ein Punkt: 1 + 20; ohne Abstand oder im Original gemessen bei Zoom 0,5 alle 41.
		if n := len(s.freiRoh); n < 19 || n > 22 {
			t.Errorf("Zoom %d %%: %d Roh-Punkte, erwartet rund 21", prozent, n)
		}
		s.Loslassen(start.Plus(document.Pt(40, 0)))
	}
}

// K-4 (Mutant FH31): Die Roh-Punkte eines sehr langen Zugs bleiben unter der Grenze; das Ergebnis ist gültig.
func TestFreihandRohpunkteBegrenzt(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugFreihand)
	s.Druecken(f(10, 10), TastePrimaer)
	hoechst := 0
	for i := 1; i <= 3*document.MaxFreihandPunkte; i++ {
		x := 10 + float64(i%2)*150
		s.Ziehen(f(x, 10+float64(i%80)))
		hoechst = max(hoechst, len(s.freiRoh))
	}
	if hoechst > document.MaxFreihandPunkte {
		t.Errorf("%d Roh-Punkte, Grenze %d", hoechst, document.MaxFreihandPunkte)
	}
	s.Loslassen(f(100, 50))
	if fh, ok := letztes(t, s).(document.Freihand); !ok || len(fh.Punkte()) > document.MaxFreihandPunkte {
		t.Errorf("Ergebnis %T", letztes(t, s))
	}
}

// F-1037 (Kritik #333 K-3): Ein sehr langer Zug (weit über MaxFreihandPunkte Rohpunkte) behält auch am Anfang seine
// Form. Gleichmäßiges Ausdünnen halbierte den Anfang bei jeder Runde wieder: dort lagen die gespeicherten Punkte rund
// 56 px auseinander und die Linie wich um Pixel von der gezogenen Spirale ab, am Ende 11 px.
func TestFreihandLangerZugBehaeltDenAnfang(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 1600, 1000)))
	if err != nil {
		t.Fatal(err)
	}
	s := NeueSitzung(d)
	s.SetzeFenster(image.Pt(1648, 1048))
	s.SetzeWerkzeug(WerkzeugFreihand)
	spirale := func(i int) document.Punkt {
		a := float64(i) * 0.01
		return s.Abb.ZuFenster(document.Pt(800+(20+float64(i)*0.02)*math.Cos(a), 500+(20+float64(i)*0.015)*math.Sin(a)))
	}
	const n = 30000 // je Schritt rund 3 Fensterpixel: rund 15000 Rohpunkte
	s.Druecken(spirale(0), TastePrimaer)
	for i := 1; i <= n; i++ {
		s.Ziehen(spirale(i))
	}
	s.Loslassen(spirale(n))
	fh, ok := letztes(t, s).(document.Freihand)
	if !ok {
		t.Fatalf("kein Freihand-Strich: %T", letztes(t, s))
	}
	pts := fh.Punkte()
	abstand := func(von, bis int) float64 {
		var l float64
		for i := von + 1; i < bis; i++ {
			l += pts[i].Minus(pts[i-1]).Laenge()
		}
		return l / float64(bis-von-1)
	}
	anfang, ende := abstand(0, len(pts)/10), abstand(len(pts)*9/10, len(pts))
	if anfang > 2*ende || ende > 2*anfang {
		t.Errorf("mittlerer Abstand erstes Zehntel %.1f, letztes Zehntel %.1f: nicht gleicher Größenordnung", anfang, ende)
	}
	// Die gezogene Spirale im ersten Zehntel liegt dicht an der gespeicherten Linie. Gemessen 28.11.2026: 2,0 px (so
	// viel Weg passt erst bei rund 1 px Toleranz in die Rohpunkte, dazu Glätten), gleichmäßig ausgedünnt 42 px.
	weit := 0.0
	for i := 0; i <= n/10; i += 7 {
		q, best := s.original(spirale(i)), math.Inf(1)
		for j := 1; j < len(pts); j++ {
			best = math.Min(best, strecke(q, pts[j-1], pts[j]))
		}
		weit = math.Max(weit, best)
	}
	if weit > 3 {
		t.Errorf("Anfang weicht bis %.2f px von der gezogenen Spirale ab", weit)
	}
}

// strecke ist der Abstand von p zur Strecke a–b.
func strecke(p, a, b document.Punkt) float64 {
	d := b.Minus(a)
	l2 := d.X*d.X + d.Y*d.Y
	if l2 == 0 {
		return p.Minus(a).Laenge()
	}
	u := math.Max(0, math.Min(1, ((p.X-a.X)*d.X+(p.Y-a.Y)*d.Y)/l2))
	return p.Minus(a.Plus(d.Mal(u))).Laenge()
}

// K-11 (Mutant FH34): Die Vereinfachungs-Toleranz gilt in Fensterpixeln. Ein Zickzack mit 1,2 Fensterpixeln Ausschlag
// bleibt bei Zoom 4 erhalten (im Original 0,3 Pixel); ohne Teilen durch den Zoom ginge er verloren.
func TestFreihandToleranzFolgtDemZoom(t *testing.T) {
	zahl := func(prozent int) int {
		s, _, _ := sitzung(t)
		s.SetzeZoom(prozent)
		s.SetzeWerkzeug(WerkzeugFreihand)
		s.Druecken(freiPunkt(s, 60, 50), TastePrimaer)
		var letzter document.Punkt
		for i := 1; i <= 40; i++ {
			letzter = freiPunkt(s, 60, 50).Plus(document.Pt(float64(i)*3, float64(i%2)*1.2))
			s.Ziehen(letzter)
		}
		s.Loslassen(letzter)
		return len(letztes(t, s).(document.Freihand).Punkte())
	}
	eins, vier := zahl(100), zahl(400)
	if eins < 20 || vier < eins*3/4 {
		t.Errorf("Punkte bei Zoom 1: %d, bei Zoom 4: %d (gleichen Zug erwartet, nicht weniger als 3/4)", eins, vier)
	}
}

// K-15 (Mutant FH37): Der Schalter „Freehand“ wirkt nicht während eines Zugs.
func TestSchalteTextmarkerFreiNichtImZug(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugTextmarker)
	s.Druecken(f(20, 20), TastePrimaer)
	if !s.Zieht() {
		t.Fatal("kein Zug")
	}
	if s.SchalteTextmarkerFrei() || s.TextmarkerFrei() {
		t.Error("Schalter wirkt im Zug")
	}
	s.Loslassen(f(80, 20))
}

// K-16 (Mutant FH38): Breite() liest die Strichstärke des gewählten Freihand-Strichs, nicht die des Werkzeugs.
func TestBreiteLiestFreihand(t *testing.T) {
	s, _, _ := sitzung(t)
	id, err := s.D.Hinzufuegen(document.NeuerFreihand([]document.Punkt{document.Pt(20, 80), document.Pt(60, 90)},
		document.Strich{Farbe: rot.Farbe, Breite: 9}))
	if err != nil {
		t.Fatal(err)
	}
	s.SetzeWerkzeug(WerkzeugAuswahl)
	s.waehleNur(id)
	if b := s.Breite(); b != 9 {
		t.Errorf("Breite %v, erwartet 9", b)
	}
}

// K-9 (Mutant FH32): Bei mehreren gewählten Freihand-Strichen wirken Breite und Strichart auf alle, und gemischte
// Breiten werden erkannt.
func TestMehrereFreihandStil(t *testing.T) {
	s, _, _ := sitzung(t)
	var ids [2]int
	for i := range ids {
		id, err := s.D.Hinzufuegen(document.NeuerFreihand([]document.Punkt{document.Pt(20, float64(60+10*i)), document.Pt(80, float64(65+10*i))},
			document.Strich{Farbe: rot.Farbe, Breite: float64(4 + 2*i)}))
		if err != nil {
			t.Fatal(err)
		}
		ids[i] = id
	}
	s.SetzeWerkzeug(WerkzeugAuswahl)
	s.waehleNur(ids[0])
	s.SchalteGewaehlt(ids[1])
	if s.AnzahlGewaehlt() != 2 || !s.Gemischt(EigBreite) {
		t.Fatalf("Anzahl %d, Breite gemischt: %v", s.AnzahlGewaehlt(), s.Gemischt(EigBreite))
	}
	if !s.SetzeBreite(11) || s.Gemischt(EigBreite) {
		t.Fatal("SetzeBreite")
	}
	for _, id := range ids {
		if b := objekt(t, s, id).(document.Freihand).Strich.Breite; b != 11 {
			t.Errorf("Freihand %d: Breite %v", id, b)
		}
	}
}

// K-19 (Mutant FH40): Der Schalter „freehand“ aus tools.ini gilt nur für den Textmarker; bei einem anderen Werkzeug
// (hier der Pfeil) bleibt er ohne Wirkung.
func TestFreihandSchalterNurBeimTextmarker(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugTextmarker) // beide Werkzeuge haben danach einen Stil
	s.SetzeWerkzeug(WerkzeugPfeil)
	var liste []settings.WerkzeugStil
	for _, e := range s.WerkzeugStile() {
		e.Freihand = true // in jedem Abschnitt gesetzt
		liste = append(liste, e)
	}
	s.SetzeWerkzeugStile(liste)
	s.SetzeWerkzeug(WerkzeugPfeil)
	if s.stilVon(WerkzeugPfeil).Freihand {
		t.Error("Pfeil trägt den Schalter Freihand")
	}
	if !s.stilVon(WerkzeugTextmarker).Freihand {
		t.Error("Textmarker trägt ihn nicht")
	}
}
