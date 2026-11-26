// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// F-829: Verschieben, Griffe und Fußzeile rechnen den Pfeil mit dem, was zu sehen ist (Pfeil.Gestalt): samt Köpfen,
// die über die Enden ragen, und ohne den Kontrollpunkt K.

func gestaltSitzung(t *testing.T) *Sitzung {
	t.Helper()
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 200, 100)))
	if err != nil {
		t.Fatal(err)
	}
	s := NeueSitzung(d)
	s.SetzeFenster(image.Pt(248, 148))
	return s
}

func punktPfeil(a, b, k document.Punkt) document.Pfeil {
	return document.Pfeil{A: a, B: b, K: k, Strich: document.Strich{Farbe: color.NRGBA{R: 200, A: 255}, Breite: 6}}.
		MitKoepfen(document.KopfKeine, document.KopfPunkt)
}

// TestPfeilKopfAmRandZaehltAlsSichtbar: Der Pfeil liegt links neben dem Bild, nur sein Punkt-Kopf (Radius 12) ragt
// über MinSichtbar hinein: Er gilt als sichtbar. Mit den Grenzen (bis B plus Strich) wäre er es nicht.
func TestPfeilKopfAmRandZaehltAlsSichtbar(t *testing.T) {
	s := gestaltSitzung(t)
	p := punktPfeil(document.Pt(-60, 50), document.Pt(-2, 50), document.Pt(-31, 50))
	mind := MinSichtbar / s.Abb.Zoom
	if g := p.Grenzen(); g.Max.X >= mind {
		t.Fatalf("Grenzen reichen bis %v, der Test braucht weniger als %v", g.Max.X, mind)
	}
	if g := p.Gestalt(); g.Max.X < mind {
		t.Fatalf("Gestalt reicht nur bis %v", g.Max.X)
	}
	if x, _ := s.sichtbarIn(p); !x {
		t.Error("Pfeil mit sichtbarem Kopf gilt als unsichtbar")
	}
}

// TestPfeilKontrollpunktZaehltNicht: Die Kurve bleibt links neben dem Bild (höchstens x = −10), nur der Kontrollpunkt
// liegt im Bild: Der Pfeil ist nicht zu sehen.
func TestPfeilKontrollpunktZaehltNicht(t *testing.T) {
	s := gestaltSitzung(t)
	p := punktPfeil(document.Pt(-100, 50), document.Pt(-50, 50), document.Pt(50, 50)).
		MitKoepfen(document.KopfKeine, document.KopfKeine)
	if x, _ := s.sichtbarIn(p); x {
		t.Errorf("Pfeil gilt über seinen Kontrollpunkt als sichtbar, Gestalt %v", p.Gestalt())
	}
}

// TestPfeilVerschiebenHaeltKopfImBild: Nach links weit aus dem Bild geschoben, bleibt mindestens MinSichtbar des
// Pfeils samt Kopf im Bild — und nicht mehr, als die Grenzen verlangten.
func TestPfeilVerschiebenHaeltKopfImBild(t *testing.T) {
	s := gestaltSitzung(t)
	p := punktPfeil(document.Pt(40, 50), document.Pt(100, 50), document.Pt(70, 50))
	d := s.begrenzeVerschieben(p, document.Pt(-1000, 0))
	v := Verschiebe(p, d).(document.Pfeil)
	mind := MinSichtbar / s.Abb.Zoom
	if g := v.Gestalt(); math.Abs(g.Max.X-mind) > 1e-3 {
		t.Errorf("rechter Rand der Gestalt nach dem Verschieben %v, Soll %v", g.Max.X, mind)
	}
}

// TestFussFeldPfeilMitKopf: Die Fußzeile nennt die Größe samt Kopf.
func TestFussFeldPfeilMitKopf(t *testing.T) {
	s := gestaltSitzung(t)
	p, err := s.D.Hinzufuegen(punktPfeil(document.Pt(40, 50), document.Pt(140, 50), document.Pt(90, 50)))
	if err != nil {
		t.Fatal(err)
	}
	s.Auswahl = p
	// Schaft ab A mit halbem Strich (3), Punkt um B mit Radius 1,5·6+3 = 12: 3 + 100 + 12 breit, 24 hoch.
	if a := feld(t, s.FussFelder(nil), FeldAuswahl); a.Groesse != image.Pt(115, 24) {
		t.Errorf("Größe %v, Soll 115×24", a.Groesse)
	}
}
