// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"math"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// blasenSitzung ist eine Sitzung auf einem 400×300-Bild ohne Objekte (Original = Ansicht, Fenster = Bild + 24).
func blasenSitzung(t *testing.T) *Sitzung {
	t.Helper()
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 400, 300)))
	if err != nil {
		t.Fatal(err)
	}
	s := NeueSitzung(d)
	s.SetzeFenster(image.Pt(448, 348))
	if s.Abb.Zoom != 1 {
		t.Fatalf("Zoom %v", s.Abb.Zoom)
	}
	return s
}

// bestaetigeBlase setzt den Text und bestätigt; liefert die Blase.
func bestaetigeBlase(t *testing.T, s *Sitzung, text string) document.Text {
	t.Helper()
	s.Tippe(text)
	s.Enter(false)
	tx, ok := letztes(t, s).(document.Text)
	if !ok {
		t.Fatalf("kein Text: %T", letztes(t, s))
	}
	return tx
}

// Aufziehen: Der Druckpunkt ist die Spitze, das Loslassen der Ort des Texts (ADR-0021 Nachtrag 13 Punkt 5).
func TestSprechblaseAufziehen(t *testing.T) {
	s := blasenSitzung(t)
	s.SchalteEinrasten() // sonst rastet das Schild an der Bildmitte ein (eigener Test unten)
	s.SetzeWerkzeug(WerkzeugSprechblase)
	s.Druecken(f(100, 200), TastePrimaer)
	s.Ziehen(f(150, 150))
	if !s.Zieht() || len(s.D.Objekte()) != 1 {
		t.Fatalf("während des Ziehens: zieht %v, Objekte %d, Soll eine Vorschau", s.Zieht(), len(s.D.Objekte()))
	}
	s.Loslassen(f(160, 120))
	if !s.TextLaeuft() || s.Zieht() {
		t.Fatalf("nach dem Loslassen: Eingabe %v, zieht %v", s.TextLaeuft(), s.Zieht())
	}
	tx := bestaetigeBlase(t, s, "Hallo")
	if tx.Spitze == nil || !nah(*tx.Spitze, document.Pt(100, 200)) {
		t.Errorf("Spitze %v, Soll (100, 200)", tx.Spitze)
	}
	if !nah(tx.Ort, document.Pt(160, 120)) {
		t.Errorf("Ort %v, Soll (160, 120)", tx.Ort)
	}
	if tx.Farbe != Weiss || tx.Hintergrund != Beschriftungsfarben[0] {
		t.Errorf("Farben %v/%v: Soll das Paar des Texts", tx.Farbe, tx.Hintergrund)
	}
	if s.Auswahl != tx.ID {
		t.Errorf("Auswahl %d, Soll die neue Blase %d", s.Auswahl, tx.ID)
	}
	// Ein Rückgängig-Schritt für das Anlegen samt Text.
	if !s.Rueckgaengig() || len(s.D.Objekte()) != 0 {
		t.Errorf("nach Rückgängig %d Objekte, Soll 0", len(s.D.Objekte()))
	}
}

// Klick ohne Zug: Text an der Klickstelle, Spitze 3 Schriftgrade links unter der linken unteren Ecke.
func TestSprechblaseKlick(t *testing.T) {
	s := blasenSitzung(t)
	s.SetzeWerkzeug(WerkzeugSprechblase)
	s.Druecken(f(200, 100), TastePrimaer)
	s.Loslassen(f(200, 100))
	if !s.TextLaeuft() {
		t.Fatal("keine Texteingabe nach dem Klick")
	}
	tx := bestaetigeBlase(t, s, "Hi")
	if !nah(tx.Ort, document.Pt(200, 100)) || tx.Spitze == nil {
		t.Fatalf("Ort %v, Spitze %v", tx.Ort, tx.Spitze)
	}
	// Die Spitze wurde beim Anlegen am leeren Schild bemessen: 3 Schriftgrade links und unter der linken unteren Ecke.
	leer := tx
	leer.Inhalt = ""
	g := leer.Grenzen()
	soll := document.Pt(g.Min.X-3*tx.Schriftgrad, g.Max.Y+3*tx.Schriftgrad)
	if !nah(*tx.Spitze, soll) {
		t.Errorf("Spitze %v, Soll %v", *tx.Spitze, soll)
	}
}

// Leerer Text verwirft das Objekt, auch eine aufgezogene Blase.
func TestSprechblaseLeerWirdVerworfen(t *testing.T) {
	s := blasenSitzung(t)
	s.SetzeWerkzeug(WerkzeugSprechblase)
	s.Druecken(f(100, 200), TastePrimaer)
	s.Ziehen(f(150, 150))
	s.Loslassen(f(160, 120))
	s.Tippe("  ")
	s.Enter(false)
	if len(s.D.Objekte()) != 0 || s.Geaendert() {
		t.Errorf("Objekte %d, geändert %v: leere Blase bleibt liegen", len(s.D.Objekte()), s.Geaendert())
	}
}

// Esc während des Aufziehens bricht ab.
func TestSprechblaseEscBrichtAb(t *testing.T) {
	s := blasenSitzung(t)
	s.SetzeWerkzeug(WerkzeugSprechblase)
	s.Druecken(f(100, 200), TastePrimaer)
	s.Ziehen(f(150, 150))
	s.Esc()
	if len(s.D.Objekte()) != 0 || s.Zieht() || s.TextLaeuft() {
		t.Errorf("Objekte %d, zieht %v, Eingabe %v", len(s.D.Objekte()), s.Zieht(), s.TextLaeuft())
	}
}

// Ein Klick auf einen vorhandenen Text bearbeitet ihn, statt eine Blase zu setzen.
func TestSprechblaseKlickAufTextBearbeitet(t *testing.T) {
	s := blasenSitzung(t)
	id, err := s.D.Hinzufuegen(document.Text{Ort: document.Pt(100, 100), Inhalt: "x", Schriftgrad: 20, Farbe: Weiss})
	if err != nil {
		t.Fatal(err)
	}
	s.SetzeWerkzeug(WerkzeugSprechblase)
	s.Druecken(f(102, 104), TastePrimaer)
	s.Loslassen(f(102, 104))
	if !s.TextLaeuft() || s.Auswahl != id || len(s.D.Objekte()) != 1 {
		t.Errorf("Eingabe %v, Auswahl %d, Objekte %d", s.TextLaeuft(), s.Auswahl, len(s.D.Objekte()))
	}
}

// Farbpaar und Stil der Sprechblase sind die des Text-Werkzeugs (Punkt 5).
func TestSprechblasePaarWieText(t *testing.T) {
	s := blasenSitzung(t)
	s.SetzeWerkzeug(WerkzeugText)
	s.SetzeFeld(FeldFuellung)
	s.SetzeFarbe(Beschriftungsfarben[1])
	s.SetzeWerkzeug(WerkzeugSprechblase)
	if s.Stil.Fuellung != Beschriftungsfarben[1] || s.Stil.Farbe != Weiss {
		t.Errorf("Paar der Sprechblase %v/%v, Soll das des Texts (Weiß auf Blau)", s.Stil.Farbe, s.Stil.Fuellung)
	}
	s.SetzeFarbe(Beschriftungsfarben[2])
	s.SetzeWerkzeug(WerkzeugKasten)
	s.SetzeWerkzeug(WerkzeugText)
	if s.Stil.Fuellung != Beschriftungsfarben[2] {
		t.Errorf("Füllung des Texts %v, Soll Grün aus der Sprechblase", s.Stil.Fuellung)
	}
	if !s.Fuellbar() {
		t.Error("Text ohne Füllfeld")
	}
	s.SetzeWerkzeug(WerkzeugSprechblase)
	if !s.Fuellbar() {
		t.Error("Sprechblase ohne Füllfeld")
	}
	if p, _ := StandardPaar(WerkzeugSprechblase); p != Weiss {
		t.Errorf("Standardpaar der Sprechblase %v", p)
	}
}

// blase legt eine Blase (Ort (150,100), Spitze (60,220)) an und wählt sie.
func legeBlase(t *testing.T, s *Sitzung) (int, document.Text) {
	t.Helper()
	sp := document.Pt(60, 220)
	id, err := s.D.Hinzufuegen(document.Text{Ort: document.Pt(150, 100), Inhalt: "Blase", Schriftgrad: 20, Farbe: Weiss,
		Hintergrund: Beschriftungsfarben[0], Spitze: &sp})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.D.VergissVerlauf(); err != nil {
		t.Fatal(err)
	}
	s.Auswahl = id
	o, _ := s.D.Objekt(id)
	return id, o.(document.Text)
}

// Verschieben bewegt Schild und Spitze gemeinsam, und Rückgängig stellt beide her, ohne dass die Zeiger geteilt sind.
func TestSprechblaseVerschiebenMitSpitze(t *testing.T) {
	s := blasenSitzung(t)
	s.SetzeWerkzeug(WerkzeugAuswahl)
	id, vor := legeBlase(t, s)
	start := *vor.Spitze
	s.Druecken(f(vor.Ort.X+5, vor.Ort.Y+5), TastePrimaer)
	s.Ziehen(f(vor.Ort.X+45, vor.Ort.Y+35))
	s.Loslassen(f(vor.Ort.X+45, vor.Ort.Y+35))
	o, _ := s.D.Objekt(id)
	nach := o.(document.Text)
	if !nah(nach.Ort, vor.Ort.Plus(document.Pt(40, 30))) || nach.Spitze == nil || !nah(*nach.Spitze, start.Plus(document.Pt(40, 30))) {
		t.Fatalf("nach dem Verschieben Ort %v, Spitze %v", nach.Ort, nach.Spitze)
	}
	if vor.Spitze == nach.Spitze || !nah(*vor.Spitze, start) {
		t.Error("Spitze des Ausgangsstands geteilt oder verändert")
	}
	if !s.Rueckgaengig() {
		t.Fatal("Rückgängig nicht möglich")
	}
	o, _ = s.D.Objekt(id)
	zurueck := o.(document.Text)
	if !nah(zurueck.Ort, vor.Ort) || !nah(*zurueck.Spitze, start) {
		t.Errorf("nach Rückgängig Ort %v, Spitze %v", zurueck.Ort, *zurueck.Spitze)
	}
	if !s.Wiederholen() {
		t.Fatal("Wiederholen nicht möglich")
	}
	o, _ = s.D.Objekt(id)
	if !nah(*o.(document.Text).Spitze, start.Plus(document.Pt(40, 30))) {
		t.Errorf("nach Wiederholen Spitze %v", *o.(document.Text).Spitze)
	}
}

// Der Griff an der Spitze verschiebt nur sie, als ein Rückgängig-Schritt.
func TestSprechblaseSpitzenGriff(t *testing.T) {
	s := blasenSitzung(t)
	id, vor := legeBlase(t, s)
	if g := Griffe(vor); len(g) != 1 || g[0].Art != GriffSpitze || g[0].Ort != *vor.Spitze {
		t.Fatalf("Griffe %v", g)
	}
	if g := Griffe(document.Text{Ort: document.Pt(1, 1), Inhalt: "x", Schriftgrad: 20}); len(g) != 0 {
		t.Errorf("Text ohne Spitze hat Griffe %v", g)
	}
	s.Druecken(f(60, 220), TastePrimaer)
	s.Ziehen(f(70, 250))
	s.Loslassen(f(80, 260))
	o, _ := s.D.Objekt(id)
	nach := o.(document.Text)
	if !nah(nach.Ort, vor.Ort) || !nah(*nach.Spitze, document.Pt(80, 260)) {
		t.Errorf("Ort %v, Spitze %v: nur die Spitze sollte wandern", nach.Ort, *nach.Spitze)
	}
	if !s.Rueckgaengig() {
		t.Fatal("kein Rückgängig-Schritt")
	}
	o, _ = s.D.Objekt(id)
	if !nah(*o.(document.Text).Spitze, *vor.Spitze) || s.Rueckgaengig() {
		t.Error("ein Schritt sollte die Spitze zurückstellen, mehr nicht")
	}
}

// Die Spitze bleibt im sichtbaren Bild.
func TestSprechblaseSpitzeBleibtImBild(t *testing.T) {
	s := blasenSitzung(t)
	id, _ := legeBlase(t, s)
	s.Druecken(f(60, 220), TastePrimaer)
	s.Ziehen(f(-80, 500))
	s.Loslassen(f(-80, 500))
	o, _ := s.D.Objekt(id)
	sp := *o.(document.Text).Spitze
	if sp.X < 0 || sp.X > 400 || sp.Y < 0 || sp.Y > 300 {
		t.Errorf("Spitze %v außerhalb des Bilds", sp)
	}
}

// Die Spitze rastet an den Kanten anderer Objekte ein (ADR-0021 Nachtrag 12), und die Spitze einer Blase ist selbst ein
// Ziel für andere.
func TestSprechblaseSpitzeRastetEin(t *testing.T) {
	s := blasenSitzung(t)
	kasten := document.Kasten{Bereich: document.R(document.Pt(250, 40), document.Pt(330, 90)),
		Strich: document.Strich{Farbe: Weiss, Breite: 2}}
	if _, err := s.D.Hinzufuegen(kasten); err != nil {
		t.Fatal(err)
	}
	kante := kasten.Grenzen().Min.X // mit halbem Strich
	id, _ := legeBlase(t, s)
	s.Druecken(f(60, 220), TastePrimaer)
	s.Ziehen(f(kante-3, 200))
	s.Loslassen(f(kante-3, 200))
	o, _ := s.D.Objekt(id)
	if sp := *o.(document.Text).Spitze; math.Abs(sp.X-kante) > 1e-6 {
		t.Errorf("Spitze %v rastet nicht auf die Kante x = %v ein", sp, kante)
	}
	// Umgekehrt: Ein Kasten rastet an der Spitze ein.
	xs, ys := s.rastZiele(0)
	hat := func(w []float64, v float64) bool {
		for _, x := range w {
			if math.Abs(x-v) < 1e-6 {
				return true
			}
		}
		return false
	}
	o, _ = s.D.Objekt(id)
	sp := *o.(document.Text).Spitze
	if !hat(xs, sp.X) || !hat(ys, sp.Y) {
		t.Errorf("Spitze %v fehlt unter den Einrast-Zielen", sp)
	}
}

// Der Schalter „Spitze“: aus → an (Standard-Spitze) → aus, je ein Schritt; ohne Text wirkt er nicht.
func TestSprechblaseSchalter(t *testing.T) {
	s := blasenSitzung(t)
	if s.KannSpitze() || s.SchalteSpitze() {
		t.Fatal("Schalter ohne Auswahl wirkt")
	}
	id, err := s.D.Hinzufuegen(document.Text{Ort: document.Pt(200, 100), Inhalt: "x", Schriftgrad: 20, Farbe: Weiss,
		Hintergrund: Beschriftungsfarben[0]})
	if err != nil {
		t.Fatal(err)
	}
	s.Auswahl = id
	if i := s.Seitenleiste(); !i.Spitze || i.Werkzeug != WerkzeugText || !i.Text {
		t.Errorf("Leiste eines Texts: %+v", i)
	}
	if !s.KannSpitze() || s.SpitzeAn() {
		t.Fatalf("KannSpitze %v, SpitzeAn %v", s.KannSpitze(), s.SpitzeAn())
	}
	if !s.SchalteSpitze() || !s.SpitzeAn() {
		t.Fatal("Spitze nicht angeschaltet")
	}
	o, _ := s.D.Objekt(id)
	tx := o.(document.Text)
	if !nah(*tx.Spitze, s.StandardSpitze(document.Text{Ort: tx.Ort, Inhalt: tx.Inhalt, Schriftgrad: 20})) {
		t.Errorf("Spitze %v ist nicht die Standard-Spitze", *tx.Spitze)
	}
	if i := s.Seitenleiste(); i.Werkzeug != WerkzeugSprechblase || !i.Spitze {
		t.Errorf("Leiste einer Blase: %+v", i)
	}
	if !s.SchalteSpitze() || s.SpitzeAn() {
		t.Fatal("Spitze nicht abgeschaltet")
	}
	if !s.Rueckgaengig() || !s.SpitzeAn() {
		t.Error("Rückgängig stellt die Spitze nicht her")
	}
	if !s.Rueckgaengig() || s.SpitzeAn() {
		t.Error("zweites Rückgängig nimmt die Spitze nicht wieder weg")
	}
	// Gesperrte Ebene: keine Wirkung.
	s.SchalteGesperrt(id)
	if s.KannSpitze() || s.SchalteSpitze() {
		t.Error("Schalter wirkt an gesperrter Ebene")
	}
}

// Während der Texteingabe gehört der Schalter zur Eingabe (kein eigener Schritt).
func TestSprechblaseSchalterInDerEingabe(t *testing.T) {
	s := blasenSitzung(t)
	s.SetzeWerkzeug(WerkzeugSprechblase)
	s.Druecken(f(200, 100), TastePrimaer)
	s.Loslassen(f(200, 100))
	if !s.SpitzeAn() {
		t.Fatal("neue Blase ohne Spitze")
	}
	s.SchalteSpitze()
	if s.SpitzeAn() {
		t.Error("Spitze in der Eingabe nicht abgeschaltet")
	}
	tx := bestaetigeBlase(t, s, "ohne")
	if tx.Spitze != nil {
		t.Errorf("Spitze %v, Soll keine", tx.Spitze)
	}
	if !s.Rueckgaengig() || len(s.D.Objekte()) != 0 {
		t.Error("Anlegen samt Schalter sollte ein Schritt sein")
	}
}

// Treffer: das Dreieck zählt mit, ein Punkt daneben nicht.
func TestSprechblaseTreffer(t *testing.T) {
	s := blasenSitzung(t)
	id, tx := legeBlase(t, s)
	_, sp, _, ok := tx.Spitzendreieck()
	if !ok {
		t.Fatal("kein Dreieck")
	}
	m := document.Pt((tx.Grenzen().Min.X+tx.Grenzen().Max.X)/2, (tx.Grenzen().Min.Y+tx.Grenzen().Max.Y)/2)
	mitte := m.Plus(sp.Minus(m).Mal(0.8)) // im Dreieck, weit außerhalb des Schilds
	if got, ok := Treffer(s.D.Objekte(), mitte, 0); !ok || got != id {
		t.Errorf("Punkt im Dreieck %v trifft nicht", mitte)
	}
	if _, ok := Treffer(s.D.Objekte(), document.Pt(sp.X+40, sp.Y-5), 0); ok {
		t.Error("Punkt neben der Spitze trifft")
	}
}

// Schnellstile: Die Sprechblase hat ein Werkzeug und die Stile des Texts; ein Stil ändert eine gewählte Blase.
func TestSprechblaseSchnellstile(t *testing.T) {
	s := blasenSitzung(t)
	s.SetzeWerkzeug(WerkzeugSprechblase)
	if a, ok := s.SchnellArt(); !ok || a != SchnellSprechblase {
		t.Fatalf("Art des Werkzeugs: %v %v", a, ok)
	}
	if len(s.Schnellstile(SchnellSprechblase)) != len(EingebauteSchnellstile(SchnellText)) {
		t.Errorf("%d Stile der Sprechblase", len(s.Schnellstile(SchnellSprechblase)))
	}
	gelb := schnell(t, SchnellSprechblase, "black-on-yellow")
	s.WendeSchnellstilAn(gelb) // ohne gewähltes Objekt: nur der Stand des Werkzeugs
	if s.Stil.Fuellung != stilGelb || s.Stil.Farbe != stilDunkl {
		t.Errorf("Stand nach dem Stil: %v/%v", s.Stil.Farbe, s.Stil.Fuellung)
	}
	if i := s.PassenderSchnellstil(SchnellSprechblase); i < 0 || s.Schnellstile(SchnellSprechblase)[i].Kennung != "black-on-yellow" {
		t.Errorf("passender Stil %d", i)
	}
	s.SetzeWerkzeug(WerkzeugText)
	if s.Stil.Fuellung != stilGelb { // gemeinsames Paar
		t.Errorf("Füllung des Texts %v", s.Stil.Fuellung)
	}
	// Gewählte Blase: ein Rückgängig-Schritt am Objekt, Art aus dem Objekt.
	s.SetzeWerkzeug(WerkzeugAuswahl)
	id, _ := legeBlase(t, s)
	if a, ok := s.SchnellArt(); !ok || a != SchnellSprechblase {
		t.Fatalf("Art der gewählten Blase: %v %v", a, ok)
	}
	if !s.WendeSchnellstilAn(schnell(t, SchnellSprechblase, "white-on-blue")) {
		t.Fatal("Stil wirkt nicht auf die gewählte Blase")
	}
	o, _ := s.D.Objekt(id)
	if tx := o.(document.Text); tx.Hintergrund != stilBlau || tx.Spitze == nil {
		t.Errorf("Blase nach dem Stil %+v", tx)
	}
	if !s.Rueckgaengig() {
		t.Error("kein Rückgängig-Schritt")
	}
	// Ein Text-Stil lässt eine gewählte Blase in Ruhe (andere Art).
	if s.WendeSchnellstilAn(schnell(t, SchnellText, "white-on-blue")) {
		t.Error("Text-Stil änderte eine Blase")
	}
}

// Seitenleiste: Werkzeug Sprechblase zeigt Farbe und Schrift, aber ohne Auswahl keinen Schalter.
func TestSprechblaseSeitenleisteWerkzeug(t *testing.T) {
	s := blasenSitzung(t)
	s.SetzeWerkzeug(WerkzeugSprechblase)
	i := s.Seitenleiste()
	if !i.Farbe || !i.Text || i.Spitze || i.Strich || i.Werkzeug != WerkzeugSprechblase {
		t.Errorf("Leiste %+v", i)
	}
	fz := s.FussFelder(nil)
	var schrift bool
	for _, x := range fz {
		schrift = schrift || x.Art == FeldSchrift
	}
	if !schrift {
		t.Error("Fußzeile ohne Schriftgrad bei der Sprechblase")
	}
}

// Die Standard-Spitze bleibt im sichtbaren Bild, auch wenn der Klick nahe am Rand liegt.
func TestSprechblaseStandardSpitzeImBild(t *testing.T) {
	s := blasenSitzung(t)
	s.SetzeWerkzeug(WerkzeugSprechblase)
	s.Druecken(f(30, 280), TastePrimaer)
	s.Loslassen(f(30, 280))
	tx := bestaetigeBlase(t, s, "unten")
	if sp := *tx.Spitze; sp.X < 0 || sp.Y > 300 || sp.X > 400 || sp.Y < 0 {
		t.Errorf("Spitze %v außerhalb des Bilds 400 × 300", sp)
	}
}

// Beim Aufziehen rasten Spitze (Druckpunkt) und Schild (Loslassen) an anderen Objekten ein (ADR-0021 Nachtrag 12).
func TestSprechblaseAufziehenRastetEin(t *testing.T) {
	s := blasenSitzung(t)
	kasten := document.Kasten{Bereich: document.R(document.Pt(250, 40), document.Pt(330, 90)),
		Strich: document.Strich{Farbe: Weiss, Breite: 2}}
	if _, err := s.D.Hinzufuegen(kasten); err != nil {
		t.Fatal(err)
	}
	kante := kasten.Grenzen().Min.X
	s.SetzeWerkzeug(WerkzeugSprechblase)
	s.Druecken(f(kante-3, 230), TastePrimaer) // Spitze knapp neben der Kastenkante
	// Ort so, dass die linke Schildkante 3 Pixel neben der rechten Kastenkante liegt.
	rechts := kasten.Grenzen().Max.X
	leer := document.Text{Ort: document.Pt(0, 0), Schriftgrad: s.Stil.Schriftgrad}
	ort := rechts + 3 - leer.Grenzen().Min.X
	s.Ziehen(f(ort, 170))
	s.Loslassen(f(ort, 170))
	tx := bestaetigeBlase(t, s, "x")
	if math.Abs(tx.Spitze.X-kante) > 1e-6 {
		t.Errorf("Spitze %v rastet nicht auf x = %v ein", *tx.Spitze, kante)
	}
	if g := tx.Grenzen(); math.Abs(g.Min.X-rechts) > 1e-6 {
		t.Errorf("Schild links bei %v, Soll auf der Kastenkante %v", g.Min.X, rechts)
	}
}

// Ein eigener Stil der Sprechblase wirkt auf den Stand (Schriftgrad, Füllung), und nach dem Anwenden auf eine gewählte
// Blase passt die Kachel (Art des Objekts ist die Sprechblase).
func TestSprechblaseEigenerStilWirkt(t *testing.T) {
	s := blasenSitzung(t)
	s.SetzeWerkzeug(WerkzeugSprechblase)
	eigen := Schnellstil{Art: SchnellSprechblase, Farbe: Weiss, Fuellung: stilBlau, Schriftgrad: 30}
	s.WendeSchnellstilAn(eigen)
	if s.Stil.Schriftgrad != 30 || s.Stil.Fuellung != stilBlau {
		t.Fatalf("Stand: Schriftgrad %v, Füllung %v", s.Stil.Schriftgrad, s.Stil.Fuellung)
	}
	s.SetzeWerkzeug(WerkzeugAuswahl)
	legeBlase(t, s)
	if !s.WendeSchnellstilAn(schnell(t, SchnellSprechblase, "white-on-blue")) {
		t.Fatal("kein Stil auf die Blase")
	}
	if i := s.PassenderSchnellstil(SchnellSprechblase); i < 0 || s.Schnellstile(SchnellSprechblase)[i].Kennung != "white-on-blue" {
		t.Errorf("passender Stil nach dem Anwenden: %d", i)
	}
}
