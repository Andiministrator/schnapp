// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"math"
	"testing"

	"github.com/Andiministrator/schnapp/document"
)

// letztes liefert das zuoberst liegende Objekt.
func letztes(t *testing.T, s *Sitzung) document.Objekt {
	t.Helper()
	o := s.D.Objekte()
	if len(o) == 0 {
		t.Fatal("keine Objekte")
	}
	return o[len(o)-1]
}

// schritte zählt die Rückgängig-Schritte und stellt sie danach samt Auswahl wieder her.
func schritte(t *testing.T, s *Sitzung) int {
	t.Helper()
	auswahl := s.Auswahl
	defer func() { s.Auswahl = auswahl }()
	n := 0
	for s.Rueckgaengig() {
		n++
	}
	for i := 0; i < n; i++ {
		if !s.Wiederholen() {
			t.Fatal("Wiederholen")
		}
	}
	return n
}

func zieh(s *Sitzung, von, bis document.Punkt) {
	s.Druecken(von, TastePrimaer)
	s.Ziehen(von.Plus(bis.Minus(von).Mal(0.5)))
	s.Loslassen(bis)
}

func TestPfeilAnlegen(t *testing.T) {
	s, _, _ := sitzung(t)
	if !s.SetzeWerkzeug(WerkzeugPfeil) {
		t.Fatal("Werkzeug")
	}
	s.Druecken(f(60, 70), TastePrimaer)
	s.Ziehen(f(62, 70)) // unter der Ziehschwelle: noch kein Objekt
	if n := len(s.D.Objekte()); n != 2 {
		t.Fatalf("%d Objekte unter der Schwelle", n)
	}
	s.Ziehen(f(90, 80))
	if n := len(s.D.Objekte()); n != 3 || !s.Zieht() {
		t.Fatalf("%d Objekte beim Ziehen, zieht %v", n, s.Zieht())
	}
	s.Loslassen(f(100, 90))
	pf, ok := letztes(t, s).(document.Pfeil)
	if !ok {
		t.Fatalf("kein Pfeil: %T", letztes(t, s))
	}
	if !nah(pf.A, document.Pt(60, 70)) || !nah(pf.B, document.Pt(100, 90)) || !nah(pf.K, document.Pt(80, 80)) {
		t.Errorf("Pfeil %v → %v, K %v", pf.A, pf.B, pf.K)
	}
	if pf.Strich != (document.Strich{Farbe: Beschriftungsfarben[0], Breite: 4}) {
		t.Errorf("Strich %+v", pf.Strich)
	}
	if s.Auswahl != pf.ID || s.Werkzeug != WerkzeugPfeil || s.Zieht() || !s.Geaendert() {
		t.Errorf("Auswahl %d (Pfeil %d), Werkzeug %v, zieht %v, geändert %v", s.Auswahl, pf.ID, s.Werkzeug, s.Zieht(),
			s.Geaendert())
	}
	if n := schritte(t, s); n != 1 {
		t.Errorf("%d Verlaufsschritte, erwartet 1", n)
	}
	s.Rueckgaengig()
	if n := len(s.D.Objekte()); n != 2 || s.Geaendert() || s.Auswahl != 0 {
		t.Errorf("nach Rückgängig %d Objekte, geändert %v, Auswahl %d", n, s.Geaendert(), s.Auswahl)
	}
}

func TestPfeilRastetMitUmschalt(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugPfeil)
	s.Umschalt = true
	zieh(s, f(60, 70), f(100, 75)) // 7° → waagerecht, Länge bleibt
	pf := letztes(t, s).(document.Pfeil)
	if l := math.Hypot(40, 5); !nah(pf.B, document.Pt(60+l, 70)) {
		t.Errorf("waagerecht: B %v", pf.B)
	}
	zieh(s, f(20, 60), f(50, 88)) // 43° → 45°
	pf = letztes(t, s).(document.Pfeil)
	l := math.Hypot(30, 28) / math.Sqrt2
	if !nah(pf.B, document.Pt(20+l, 60+l)) {
		t.Errorf("diagonal: B %v, erwartet %v", pf.B, document.Pt(20+l, 60+l))
	}
	s.Umschalt = false
	zieh(s, f(120, 70), f(150, 98))
	if pf = letztes(t, s).(document.Pfeil); !nah(pf.B, document.Pt(150, 98)) {
		t.Errorf("ohne Umschalt: B %v", pf.B)
	}
}

// Der mittlere Griff des neuen, ausgewählten Pfeils biegt ihn — auch mit aktivem Pfeil-Werkzeug, ohne einen zweiten
// Pfeil anzulegen. Anlegen und Biegen sind zwei Schritte.
func TestPfeilBiegenUeberGriff(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugPfeil)
	zieh(s, f(20, 70), f(80, 70))
	id := s.Auswahl
	mitte := document.Pt(50, 70)
	if pf := letztes(t, s).(document.Pfeil); !nah(pf.Griff(), mitte) {
		t.Fatalf("Griff %v", pf.Griff())
	}
	zieh(s, f(50, 70), f(50, 90))
	if n := len(s.D.Objekte()); n != 3 {
		t.Fatalf("%d Objekte — Griff hat einen neuen Pfeil angelegt", n)
	}
	pf := letztes(t, s).(document.Pfeil)
	if pf.ID != id || !nah(pf.Griff(), document.Pt(50, 90)) || !nah(pf.K, document.Pt(50, 110)) {
		t.Errorf("gebogen: ID %d, Griff %v, K %v", pf.ID, pf.Griff(), pf.K)
	}
	if !nah(pf.A, document.Pt(20, 70)) || !nah(pf.B, document.Pt(80, 70)) {
		t.Errorf("Enden verschoben: %v %v", pf.A, pf.B)
	}
	if n := schritte(t, s); n != 2 {
		t.Errorf("%d Schritte, erwartet 2", n)
	}
}

func TestKastenAnlegen(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugKasten)
	zieh(s, f(90, 80), f(60, 60)) // rückwärts gezogen
	k, ok := letztes(t, s).(document.Kasten)
	if !ok {
		t.Fatalf("kein Kasten: %T", letztes(t, s))
	}
	if k.Bereich != document.R(document.Pt(60, 60), document.Pt(90, 80)) {
		t.Errorf("Bereich %v", k.Bereich)
	}
	if k.Eckradius != 4 || k.Fuellung.A != 0 || k.Strich.Breite != 4 || k.Strich.Farbe != Beschriftungsfarben[0] {
		t.Errorf("Stil %+v", k)
	}
	s.Umschalt = true
	zieh(s, f(120, 60), f(140, 70))
	if k = letztes(t, s).(document.Kasten); k.Bereich != document.R(document.Pt(120, 60), document.Pt(140, 80)) {
		t.Errorf("Quadrat %v", k.Bereich)
	}
	// Start neben den Griffen des ausgewählten Quadrats (ein Griff hätte Vorrang).
	zieh(s, f(150, 60), f(140, 30))
	if k = letztes(t, s).(document.Kasten); k.Bereich != document.R(document.Pt(120, 30), document.Pt(150, 60)) {
		t.Errorf("Quadrat nach links oben %v", k.Bereich)
	}
	if n := schritte(t, s); n != 3 {
		t.Errorf("%d Schritte, erwartet 3", n)
	}
}

// Ein Klick oder ein zu kleines Ziehen legt nichts an und hinterlässt nichts im Verlauf.
func TestMindestgroesse(t *testing.T) {
	s, _, _ := sitzung(t)
	pruefe := func(fall string) {
		t.Helper()
		if n := len(s.D.Objekte()); n != 2 || s.Geaendert() || s.D.KannRueckgaengig() || s.Zieht() {
			t.Errorf("%s: %d Objekte, geändert %v, Verlauf %v, zieht %v", fall, n, s.Geaendert(), s.D.KannRueckgaengig(),
				s.Zieht())
		}
	}
	for _, w := range []Werkzeug{WerkzeugPfeil, WerkzeugKasten} {
		s.SetzeWerkzeug(w)
		s.Druecken(f(60, 70), TastePrimaer)
		s.Loslassen(f(60, 70))
		pruefe("Klick")
		zieh(s, f(60, 70), f(65, 70)) // über der Ziehschwelle, unter MinObjekt
		pruefe("5 px")
	}
	zieh(s, f(60, 70), f(100, 74)) // Kasten 40 × 4: zu flach
	pruefe("flacher Kasten")
	// Bei Zoom 2 zählt die Größe im Fenster: 4 Pixel im Original sind 8 im Fenster und bleiben.
	s.Abb = Abbildung{Zoom: 2}
	s.SetzeWerkzeug(WerkzeugPfeil)
	zieh(s, document.Pt(120, 140), document.Pt(128, 140))
	if pf, ok := letztes(t, s).(document.Pfeil); !ok || !nah(pf.A, document.Pt(60, 70)) || !nah(pf.B, document.Pt(64, 70)) {
		t.Errorf("Zoom 2: %+v", letztes(t, s))
	}
}

// Die Abbildung Fenster → Ansicht → Original beachtet Zoom, Versatz und Zuschnitt.
func TestAnlegenImZuschnitt(t *testing.T) {
	s, _, _ := sitzung(t)
	if err := s.D.SetzeZuschnitt(image.Rect(50, 20, 200, 100)); err != nil {
		t.Fatal(err)
	}
	s.Abb = Abbildung{Zoom: 2, Versatz: document.Pt(10, 10)}
	s.SetzeWerkzeug(WerkzeugPfeil)
	zieh(s, document.Pt(30, 30), document.Pt(90, 70))
	pf := letztes(t, s).(document.Pfeil)
	if !nah(pf.A, document.Pt(60, 30)) || !nah(pf.B, document.Pt(90, 50)) {
		t.Errorf("Pfeil %v → %v, erwartet (60,30) → (90,50)", pf.A, pf.B)
	}
}

func TestNummernZaehlenHoch(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugNummer)
	klick := func(x, y float64) document.Nummer {
		t.Helper()
		s.Druecken(f(x, y), TastePrimaer)
		s.Loslassen(f(x, y))
		n, ok := letztes(t, s).(document.Nummer)
		if !ok {
			t.Fatalf("keine Nummer: %T", letztes(t, s))
		}
		return n
	}
	var werte []int
	for _, x := range []float64{60, 80, 120} {
		werte = append(werte, klick(x, 80).Wert)
	}
	if werte[0] != 1 || werte[1] != 2 || werte[2] != 3 {
		t.Errorf("Werte %v, erwartet 1 2 3", werte)
	}
	n := letztes(t, s).(document.Nummer)
	if !nah(n.Mitte, document.Pt(120, 80)) || n.Radius != 14 || n.Farbe != Beschriftungsfarben[0] || s.Auswahl != n.ID {
		t.Errorf("Nummer %+v, Auswahl %d", n, s.Auswahl)
	}
	if k := schritte(t, s); k != 3 {
		t.Errorf("%d Schritte, erwartet 3", k)
	}
	// Rückgängig der letzten: ihr Wert kommt wieder.
	s.Rueckgaengig()
	if w := klick(150, 80).Wert; w != 3 {
		t.Errorf("nach Rückgängig %d, erwartet 3", w)
	}
	// Größter Wert + 1, auch wenn er nicht der letzte ist.
	if _, err := s.D.Hinzufuegen(document.Nummer{Mitte: document.Pt(5, 90), Wert: 7, Radius: 5}); err != nil {
		t.Fatal(err)
	}
	if w := klick(170, 85).Wert; w != 8 {
		t.Errorf("nach 7: %d", w)
	}
	// Ziehen vor dem Loslassen verschiebt die neue Nummer; ein Schritt.
	vorher := schritte(t, s)
	zieh(s, f(60, 60), f(70, 75))
	if n := letztes(t, s).(document.Nummer); !nah(n.Mitte, document.Pt(70, 75)) || n.Wert != 9 {
		t.Errorf("gezogen: %+v", n)
	}
	if k := schritte(t, s); k != vorher+1 {
		t.Errorf("%d Schritte, erwartet %d", k, vorher+1)
	}
	// Esc beim Ziehen bricht ab: keine Nummer.
	anzahl := len(s.D.Objekte())
	s.Druecken(f(60, 60), TastePrimaer)
	s.Ziehen(f(80, 80))
	s.Esc()
	if len(s.D.Objekte()) != anzahl || s.Zieht() {
		t.Errorf("nach Esc %d Objekte (vorher %d)", len(s.D.Objekte()), anzahl)
	}
}

func TestTextEingabe(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugText)
	s.Druecken(f(60, 60), TastePrimaer)
	s.Loslassen(f(60, 60))
	if !s.TextLaeuft() || s.Zieht() {
		t.Fatalf("Texteingabe %v, zieht %v", s.TextLaeuft(), s.Zieht())
	}
	s.Tippe("Hal")
	s.Tippe("lo\x07") // Steuerzeichen fallen weg
	s.Enter(true)     // Umschalt+Enter: neue Zeile
	s.Tippe("Welt")
	s.TextZurueck()
	s.Tippe("ä")
	s.TextZurueck() // ein Zeichen, nicht ein Byte
	if s.Rueckgaengig() || s.Geaendert() {
		t.Error("Rückgängig oder geändert während der Eingabe")
	}
	if !s.Enter(false) || s.TextLaeuft() {
		t.Fatal("Enter bestätigt nicht")
	}
	tx, ok := letztes(t, s).(document.Text)
	if !ok {
		t.Fatalf("kein Text: %T", letztes(t, s))
	}
	if tx.Inhalt != "Hallo\nWel" || !nah(tx.Ort, document.Pt(60, 60)) || tx.Schriftgrad != 20 ||
		tx.Farbe != Weiss || tx.Hintergrund != Beschriftungsfarben[0] {
		t.Errorf("Text %+v", tx)
	}
	if s.Auswahl != tx.ID || !s.Geaendert() {
		t.Errorf("Auswahl %d, geändert %v", s.Auswahl, s.Geaendert())
	}
	if n := schritte(t, s); n != 1 {
		t.Errorf("%d Schritte, erwartet 1", n)
	}
}

// Leerer Text und Esc legen nichts an; ein Klick daneben bestätigt und beginnt den nächsten Text.
func TestTextVerwerfenUndKlickDaneben(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugText)
	s.Druecken(f(60, 60), TastePrimaer)
	s.Tippe("  ")
	s.Enter(false)
	s.Druecken(f(60, 60), TastePrimaer)
	s.Tippe("weg")
	s.Esc()
	if n := len(s.D.Objekte()); n != 2 || s.Geaendert() || s.D.KannRueckgaengig() || s.Auswahl != 0 {
		t.Fatalf("%d Objekte, geändert %v, Auswahl %d", n, s.Geaendert(), s.Auswahl)
	}
	s.Druecken(f(60, 60), TastePrimaer)
	s.Tippe("eins")
	s.Druecken(f(120, 80), TastePrimaer) // bestätigt „eins“, beginnt neuen Text
	s.Tippe("zwei")
	s.Esc() // verwirft nur „zwei“
	o := s.D.Objekte()
	if len(o) != 3 || o[2].(document.Text).Inhalt != "eins" || s.TextLaeuft() {
		t.Fatalf("Objekte %+v", o)
	}
	if n := schritte(t, s); n != 1 {
		t.Errorf("%d Schritte, erwartet 1", n)
	}
}

// Doppelklick (Auswählen), Enter und Text-Werkzeug bearbeiten einen vorhandenen Text; leer bestätigt verschwindet er
// in einem Schritt.
func TestTextBearbeiten(t *testing.T) {
	s, _, _ := sitzung(t)
	id, err := s.D.Hinzufuegen(document.Text{Ort: document.Pt(60, 60), Inhalt: "ab", Schriftgrad: 20, Farbe: Weiss})
	if err != nil {
		t.Fatal(err)
	}
	s.Doppelklick(f(65, 70))
	if !s.TextLaeuft() || s.Auswahl != id {
		t.Fatalf("Doppelklick: Eingabe %v, Auswahl %d", s.TextLaeuft(), s.Auswahl)
	}
	s.Tippe("c")
	s.Enter(false)
	if tx, _ := s.D.Objekt(id); tx.(document.Text).Inhalt != "abc" {
		t.Errorf("Inhalt %q", tx.(document.Text).Inhalt)
	}
	if n := schritte(t, s); n != 2 {
		t.Errorf("%d Schritte, erwartet 2 (Hinzufügen, Bearbeiten)", n)
	}
	// Enter am ausgewählten Text, ohne Änderung bestätigt: kein Schritt.
	if !s.Enter(false) || !s.TextLaeuft() {
		t.Fatal("Enter beginnt keine Eingabe")
	}
	s.Enter(false)
	if n := schritte(t, s); n != 2 {
		t.Errorf("unverändert bestätigt: %d Schritte", n)
	}
	// Text-Werkzeug auf den Text: bearbeiten, alles löschen, bestätigen → entfernt, ein Schritt.
	s.SetzeWerkzeug(WerkzeugText)
	s.Druecken(f(65, 70), TastePrimaer)
	s.Loslassen(f(65, 70))
	for s.TextZurueck() {
	}
	s.Enter(false)
	if _, ok := s.D.Objekt(id); ok || len(s.D.Objekte()) != 2 || s.Auswahl != 0 {
		t.Fatalf("leerer Text nicht entfernt: %+v", s.D.Objekte())
	}
	if n := schritte(t, s); n != 3 {
		t.Errorf("%d Schritte, erwartet 3", n)
	}
	s.Rueckgaengig()
	if tx, ok := s.D.Objekt(id); !ok || tx.(document.Text).Inhalt != "abc" {
		t.Errorf("Rückgängig stellt nicht her: %+v", tx)
	}
	// Esc beim Bearbeiten lässt den Text wie vorher.
	s.Auswahl = id
	s.SetzeWerkzeug(WerkzeugAuswahl)
	s.Doppelklick(f(65, 70))
	s.Tippe("xyz")
	s.Esc()
	if tx, _ := s.D.Objekt(id); tx.(document.Text).Inhalt != "abc" || s.Auswahl != id {
		t.Errorf("nach Esc %q, Auswahl %d", tx.(document.Text).Inhalt, s.Auswahl)
	}
}

// Doppelklick mit einem Zeichenwerkzeug wirkt wie ein Druck, Doppelklick daneben wählt wie ein Klick.
func TestDoppelklickOhneText(t *testing.T) {
	s, k, _ := sitzung(t)
	s.Doppelklick(f(30, 30))
	s.Loslassen(f(30, 30))
	if s.Auswahl != k || s.TextLaeuft() || s.Geaendert() {
		t.Errorf("Auswahl %d, Eingabe %v", s.Auswahl, s.TextLaeuft())
	}
}

func TestSchliessenBestaetigtText(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugText)
	s.Druecken(f(60, 60), TastePrimaer)
	s.Tippe("bleibt")
	gespeichert := false
	z, err := s.Schliessen(func() (bool, error) { gespeichert = true; return false, nil })
	if z != SchlussZu || err != nil || !gespeichert {
		t.Fatalf("Schluss %v, %v, gespeichert %v", z, err, gespeichert)
	}
	if tx, ok := letztes(t, s).(document.Text); !ok || tx.Inhalt != "bleibt" {
		t.Errorf("Text verloren: %+v", letztes(t, s))
	}
}

func TestEscStufen(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugKasten)
	s.Druecken(f(60, 60), TastePrimaer)
	s.Ziehen(f(90, 90))
	s.Esc()
	if len(s.D.Objekte()) != 2 || s.Zieht() || s.Werkzeug != WerkzeugKasten {
		t.Fatalf("Esc beim Ziehen: %d Objekte, Werkzeug %v", len(s.D.Objekte()), s.Werkzeug)
	}
	zieh(s, f(60, 60), f(90, 90))
	s.Esc()
	if s.Auswahl != 0 || s.Werkzeug != WerkzeugKasten {
		t.Errorf("erste Stufe: Auswahl %d, Werkzeug %v", s.Auswahl, s.Werkzeug)
	}
	s.Esc()
	if s.Werkzeug != WerkzeugAuswahl {
		t.Errorf("zweite Stufe: Werkzeug %v", s.Werkzeug)
	}
	// Werkzeugwechsel während eines Zugs geht nicht.
	s.Druecken(f(190, 90), TastePrimaer)
	if s.SetzeWerkzeug(WerkzeugPfeil) || s.Werkzeug != WerkzeugAuswahl {
		t.Error("Werkzeugwechsel während des Ziehens")
	}
	s.Loslassen(f(190, 90))
}

func TestFarbe(t *testing.T) {
	s, _, p := sitzung(t)
	s.Auswahl = p
	if !s.NaechsteFarbe() {
		t.Fatal("ausgewählter Pfeil nicht umgefärbt")
	}
	if o, _ := s.D.Objekt(p); o.(document.Pfeil).Strich.Farbe != Beschriftungsfarben[1] {
		t.Errorf("Pfeil %+v", o)
	}
	if n := schritte(t, s); n != 1 {
		t.Errorf("%d Schritte", n)
	}
	s.Auswahl = 0
	s.SetzeWerkzeug(WerkzeugNummer)
	s.Druecken(f(60, 70), TastePrimaer)
	s.Loslassen(f(60, 70))
	if n := letztes(t, s).(document.Nummer); n.Farbe != Beschriftungsfarben[1] {
		t.Errorf("neue Nummer %+v", n)
	}
	for i := 0; i < 3; i++ {
		s.NaechsteFarbe()
	}
	if s.Stil.Farbe != Beschriftungsfarben[0] {
		t.Errorf("reihum: %v", s.Stil.Farbe)
	}
	// Gleiche Farbe: kein Schritt.
	vorher := schritte(t, s)
	if s.SetzeFarbe(Beschriftungsfarben[0]) || schritte(t, s) != vorher {
		t.Error("gleiche Farbe ergab einen Schritt")
	}
	// Während der Texteingabe färbt sie das Schild mit, ohne eigenen Schritt.
	s.SetzeWerkzeug(WerkzeugText)
	s.Druecken(f(120, 80), TastePrimaer)
	s.Tippe("x")
	s.SetzeFarbe(Beschriftungsfarben[2])
	s.Enter(false)
	if tx := letztes(t, s).(document.Text); tx.Hintergrund != Beschriftungsfarben[2] {
		t.Errorf("Schild %v", tx.Hintergrund)
	}
	if n := schritte(t, s); n != vorher+1 {
		t.Errorf("%d Schritte, erwartet %d", n, vorher+1)
	}
}

func TestRastenUndQuadrat(t *testing.T) {
	a := document.Pt(10, 10)
	for _, f := range []struct{ b, soll document.Punkt }{
		{document.Pt(20, 10), document.Pt(20, 10)},
		{a, a},
		{document.Pt(10, 0), document.Pt(10, 0)},
		{document.Pt(0, 11), document.Pt(10-math.Hypot(10, 1), 10)},
	} {
		if g := Rasten45(a, f.b); g.Minus(f.soll).Laenge() > 1e-9 {
			t.Errorf("Rasten45(%v) = %v, erwartet %v", f.b, g, f.soll)
		}
	}
	if q := Quadrat(a, document.Pt(0, 15)); q != document.Pt(0, 20) {
		t.Errorf("Quadrat %v", q)
	}
}
