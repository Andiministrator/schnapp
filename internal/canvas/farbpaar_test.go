// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image/color"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Farbpaar je Werkzeug, aktives Feld, keine Füllung und Tauschen (ADR-0021 Nachtrag 11).

var (
	blau = color.NRGBA{B: 200, A: 255}
	gelb = color.NRGBA{R: 250, G: 220, A: 255}
)

// TestFarbpaarVoreinstellung: Pfeil, Kasten und Nummer rot, der Kasten ohne Füllung, der Text weiß auf Rot — ein
// neuer Text sieht aus wie vor Nachtrag 11.
func TestFarbpaarVoreinstellung(t *testing.T) {
	s, _, _ := sitzung(t)
	rotB := Beschriftungsfarben[0]
	for _, tc := range []struct {
		w          Werkzeug
		vorn, fuel color.NRGBA
		fuellbar   bool
	}{
		{WerkzeugPfeil, rotB, color.NRGBA{}, false},
		{WerkzeugKasten, rotB, color.NRGBA{}, true},
		{WerkzeugText, Weiss, rotB, true},
		{WerkzeugNummer, rotB, color.NRGBA{}, false},
	} {
		s.SetzeWerkzeug(tc.w)
		if s.Farbe(FeldVordergrund) != tc.vorn || s.Stil.Fuellung != tc.fuel || s.Fuellbar() != tc.fuellbar {
			t.Errorf("Werkzeug %v: vorn %v, Füllung %v, füllbar %v", tc.w, s.Farbe(FeldVordergrund), s.Stil.Fuellung, s.Fuellbar())
		}
	}
	s.SetzeWerkzeug(WerkzeugText)
	s.Druecken(f(120, 80), TastePrimaer)
	s.Loslassen(f(120, 80))
	s.Tippe("x")
	s.Enter(false)
	if tx := letztes(t, s).(document.Text); tx.Farbe != Weiss || tx.Hintergrund != rotB {
		t.Errorf("neuer Text %v auf %v", tx.Farbe, tx.Hintergrund)
	}
}

// TestFarbpaarJeWerkzeug: Eine Farbe gilt nur für das Werkzeug, bei dem sie gewählt wurde; nach dem Wechsel und
// zurück ist sie wieder da. Werkzeuge ohne Farbe (Auswählen, Schwärzen) lassen das Paar stehen.
func TestFarbpaarJeWerkzeug(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugKasten)
	s.SetzeFarbe(blau)
	s.SetzeWerkzeug(WerkzeugPfeil)
	if s.Stil.Farbe != Beschriftungsfarben[0] {
		t.Errorf("Pfeil übernahm die Farbe des Kastens: %v", s.Stil.Farbe)
	}
	s.SetzeWerkzeug(WerkzeugSchwaerzen)
	s.SetzeWerkzeug(WerkzeugAuswahl)
	if s.Stil.Farbe != Beschriftungsfarben[0] {
		t.Errorf("Werkzeug ohne Farbe wechselte das Paar: %v", s.Stil.Farbe)
	}
	s.SetzeWerkzeug(WerkzeugKasten)
	if s.Stil.Farbe != blau {
		t.Errorf("Kasten vergaß seine Farbe: %v", s.Stil.Farbe)
	}
	// Ein umgefärbtes Objekt färbt die nächsten Objekte seiner Art, nicht die des aktiven Werkzeugs.
	s2, _, p := sitzung(t)
	s2.SetzeWerkzeug(WerkzeugNummer)
	s2.Auswahl = p
	s2.SetzeFarbe(gelb)
	if s2.Stil.Farbe != Beschriftungsfarben[0] || s2.paar(WerkzeugPfeil).vorn != gelb {
		t.Errorf("Stil %v, Pfeil %v", s2.Stil.Farbe, s2.paar(WerkzeugPfeil).vorn)
	}
}

// TestFeldFuellung: Mit aktiver Füllung färben Farbfeld, Hex und Deckkraft die Fläche; neue Kästen tragen sie. Ohne
// Füllung (Pfeil, Nummer) lässt sich das Feld nicht wählen und Vordergrund bleibt wirksam.
func TestFeldFuellung(t *testing.T) {
	s, _, p := sitzung(t)
	s.SetzeWerkzeug(WerkzeugPfeil)
	if s.SetzeFeld(FeldFuellung) || s.AktivesFeld() != FeldVordergrund {
		t.Error("Füllung beim Pfeil wählbar")
	}
	s.SetzeWerkzeug(WerkzeugKasten)
	if !s.SetzeFeld(FeldFuellung) || s.AktivesFeld() != FeldFuellung {
		t.Fatal("Füllung beim Kasten nicht wählbar")
	}
	s.SetzeFarbe(blau)
	if !s.SetzeHex("#FADC00") || s.Stil.Fuellung != gelb || s.Stil.Farbe != Beschriftungsfarben[0] {
		t.Errorf("Hex: Füllung %v, vorn %v", s.Stil.Fuellung, s.Stil.Farbe)
	}
	if !s.SetzeDeckkraftText("50") || s.Stil.Fuellung.A != 128 {
		t.Errorf("Deckkraft: %v", s.Stil.Fuellung)
	}
	s.Druecken(f(60, 60), TastePrimaer)
	s.Ziehen(f(80, 80))
	s.Loslassen(f(90, 90))
	k := letztes(t, s).(document.Kasten)
	if k.Fuellung != s.Stil.Fuellung || k.Strich.Farbe != Beschriftungsfarben[0] {
		t.Errorf("neuer Kasten: Füllung %v, Strich %v", k.Fuellung, k.Strich.Farbe)
	}
	// Ein gewählter Pfeil hat keine Füllung: das gewählte Feld bleibt gemerkt, wirksam ist der Vordergrund.
	s.SetzeWerkzeug(WerkzeugAuswahl)
	s.Auswahl = p
	if s.AktivesFeld() != FeldVordergrund || s.AktuelleFarbe() != rot.Farbe {
		t.Errorf("Pfeil gewählt: Feld %v, Farbe %v", s.AktivesFeld(), s.AktuelleFarbe())
	}
	s.Auswahl = k.ID
	if s.AktivesFeld() != FeldFuellung || s.AktuelleFarbe() != k.Fuellung {
		t.Errorf("Kasten gewählt: Feld %v, Farbe %v", s.AktivesFeld(), s.AktuelleFarbe())
	}
}

// TestKeineFuellung: „Keine“ setzt die Deckkraft der aktiven Füllung auf 0, am gewählten Objekt als ein Schritt.
func TestKeineFuellung(t *testing.T) {
	s, k, _ := sitzung(t)
	if s.KeineFuellung() {
		t.Error("„Keine“ ohne aktive Füllung wirkte")
	}
	s.Auswahl = k
	s.SetzeFeld(FeldFuellung)
	vor := schritte(t, s)
	if !s.KeineFuellung() {
		t.Fatal("„Keine“ wirkte nicht")
	}
	if o, _ := s.D.Objekt(k); o.(document.Kasten).Fuellung.A != 0 {
		t.Errorf("Kasten %+v", o)
	}
	if n := schritte(t, s); n != vor+1 {
		t.Errorf("%d Schritte, erwartet %d", n, vor+1)
	}
	if s.paar(WerkzeugKasten).fuellung.A != 0 {
		t.Errorf("Paar des Kastens %v", s.paar(WerkzeugKasten).fuellung)
	}
}

// TestTauscheFarben: am Werkzeug ohne Schritt, am gewählten Objekt als ein Schritt, in der Texteingabe als Teil der
// Eingabe; ohne Füllung (Pfeil) geht es nicht.
func TestTauscheFarben(t *testing.T) {
	s, k, p := sitzung(t)
	s.SetzeWerkzeug(WerkzeugPfeil)
	if s.KannTauschen() || s.TauscheFarben() {
		t.Error("Tauschen beim Pfeil")
	}
	s.SetzeWerkzeug(WerkzeugText)
	if !s.TauscheFarben() || s.Stil.Farbe != Beschriftungsfarben[0] || s.Stil.Fuellung != Weiss {
		t.Errorf("Text-Paar nach dem Tausch %v auf %v", s.Stil.Farbe, s.Stil.Fuellung)
	}
	s.Druecken(f(120, 80), TastePrimaer)
	s.Loslassen(f(120, 80))
	s.Tippe("x")
	vor := schritte(t, s)
	if !s.TauscheFarben() || !s.TextLaeuft() {
		t.Fatal("Tausch in der Eingabe")
	}
	s.Enter(false)
	if tx := letztes(t, s).(document.Text); tx.Farbe != Weiss || tx.Hintergrund != Beschriftungsfarben[0] {
		t.Errorf("Text nach dem Tausch %v auf %v", tx.Farbe, tx.Hintergrund)
	}
	if n := schritte(t, s); n != vor+1 {
		t.Errorf("%d Schritte, erwartet %d (Eingabe ist ein Schritt)", n, vor+1)
	}
	s.SetzeWerkzeug(WerkzeugAuswahl)
	s.Auswahl = p
	if s.KannTauschen() {
		t.Error("Tauschen am Pfeil")
	}
	s.Auswahl = k
	vor = schritte(t, s)
	alt, _ := s.D.Objekt(k)
	if !s.TauscheFarben() {
		t.Fatal("Tausch am Kasten")
	}
	neu, _ := s.D.Objekt(k)
	if a, n := alt.(document.Kasten), neu.(document.Kasten); n.Fuellung != a.Strich.Farbe || n.Strich.Farbe != a.Fuellung {
		t.Errorf("Kasten %+v", n)
	}
	if n := schritte(t, s); n != vor+1 {
		t.Errorf("%d Schritte, erwartet %d", n, vor+1)
	}
	s.Rueckgaengig()
	if o, _ := s.D.Objekt(k); o != alt {
		t.Errorf("Rückgängig: %+v", o)
	}
}

// TestFarbZugFuellung: Ziehen im Farbwähler färbt bei aktiver Füllung die Fläche des gewählten Kastens, als ein
// Schritt.
func TestFarbZugFuellung(t *testing.T) {
	s, k, _ := sitzung(t)
	s.Auswahl = k
	s.SetzeFeld(FeldFuellung)
	vor := schritte(t, s)
	s.BeginneFarbZug()
	s.FarbZugHSV(HSV{H: 240, S: 1, V: 1}, 200)
	s.BeendeFarbZug()
	want := (HSV{H: 240, S: 1, V: 1}).Farbe(200)
	if o, _ := s.D.Objekt(k); o.(document.Kasten).Fuellung != want || o.(document.Kasten).Strich.Farbe != (color.NRGBA{}) {
		t.Errorf("Kasten %+v", o)
	}
	if n := schritte(t, s); n != vor+1 {
		t.Errorf("%d Schritte, erwartet %d", n, vor+1)
	}
	if s.paar(WerkzeugKasten).fuellung != want {
		t.Errorf("Paar des Kastens %v", s.paar(WerkzeugKasten).fuellung)
	}
}

// TestSeitenleisteFuellung: Die Leiste weiß, ob es eine Füllung gibt.
func TestSeitenleisteFuellung(t *testing.T) {
	s, k, p := sitzung(t)
	for _, tc := range []struct {
		w    Werkzeug
		want bool
	}{{WerkzeugPfeil, false}, {WerkzeugKasten, true}, {WerkzeugText, true}, {WerkzeugNummer, false}, {WerkzeugAuswahl, false}} {
		s.SetzeWerkzeug(tc.w)
		if s.Seitenleiste().Fuellung != tc.want {
			t.Errorf("Werkzeug %v: Füllung %v", tc.w, !tc.want)
		}
	}
	s.Auswahl = k
	if !s.Seitenleiste().Fuellung {
		t.Error("gewählter Kasten ohne Füllung")
	}
	s.Auswahl = p
	if s.Seitenleiste().Fuellung {
		t.Error("gewählter Pfeil mit Füllung")
	}
}
