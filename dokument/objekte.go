// SPDX-License-Identifier: GPL-3.0-or-later

package dokument

import (
	"fmt"
	"image/color"
)

// Objekt ist eine Beschriftung über dem Bild. Objekte sind Werte: Ändern heißt, ein geändertes Objekt mit derselben
// Kennung einsetzen (Ersetzen, Zug).
type Objekt interface {
	// Kennung ist eindeutig im Dokument und bleibt über Änderungen gleich; 0 heißt „noch nicht eingefügt“.
	Kennung() int
	// Grenzen umfasst das Objekt (für Treffer und Neuzeichnen; darf großzügig sein).
	Grenzen() Rechteck
	// mitKennung liefert eine Kopie mit gesetzter Kennung.
	mitKennung(int) Objekt
	// pruefe meldet ungültige Werte (NaN, negative Größen).
	pruefe() error
}

// Strich beschreibt Linienfarbe und -breite in Pixeln des Originals.
type Strich struct {
	Farbe  color.NRGBA
	Breite float64
}

func (s Strich) pruefe() error {
	if !endlich(s.Breite) || s.Breite < 0 {
		return fmt.Errorf("Strichbreite ungültig: %v", s.Breite)
	}
	return nil
}

// Pfeil ist eine quadratische Bézier-Kurve von A nach B mit Kontrollpunkt K; die Spitze sitzt bei B. Gerade ist er,
// wenn K auf der Mitte liegt. Der Nutzer zieht am Griff (Kurvenpunkt bei t = 0,5), nicht an K.
type Pfeil struct {
	ID      int
	A, B, K Punkt
	Strich  Strich
}

// GeraderPfeil liefert einen ungebogenen Pfeil von a nach b.
func GeraderPfeil(a, b Punkt, s Strich) Pfeil {
	return Pfeil{A: a, B: b, K: a.Plus(b).Mal(0.5), Strich: s}
}

// Punkt liefert den Kurvenpunkt bei t ∈ [0,1].
func (p Pfeil) Punkt(t float64) Punkt {
	u := 1 - t
	return p.A.Mal(u * u).Plus(p.K.Mal(2 * u * t)).Plus(p.B.Mal(t * t))
}

// Griff ist der Kurvenpunkt bei t = 0,5: (A + 2K + B) / 4.
func (p Pfeil) Griff() Punkt { return p.Punkt(0.5) }

// MitGriff liefert den Pfeil, dessen Kurve durch h läuft: K = 2h − (A+B)/2.
func (p Pfeil) MitGriff(h Punkt) Pfeil {
	p.K = h.Mal(2).Minus(p.A.Plus(p.B).Mal(0.5))
	return p
}

// RichtungAmEnde ist die normierte Tangente bei B (Richtung der Spitze); fallen Punkte zusammen, (1,0).
func (p Pfeil) RichtungAmEnde() Punkt {
	d := p.B.Minus(p.K)
	if d.Laenge() < 1e-9 {
		d = p.B.Minus(p.A)
	}
	if l := d.Laenge(); l >= 1e-9 {
		return d.Mal(1 / l)
	}
	return Punkt{1, 0}
}

func (p Pfeil) Kennung() int { return p.ID }
func (p Pfeil) Grenzen() Rechteck {
	r := R(p.A, p.B).Vereint(R(p.K, p.K))
	return aufgeweitet(r, p.Strich.Breite)
}
func (p Pfeil) mitKennung(id int) Objekt { p.ID = id; return p }
func (p Pfeil) pruefe() error {
	if !endlich(p.A.X, p.A.Y, p.B.X, p.B.Y, p.K.X, p.K.Y) {
		return fmt.Errorf("Pfeil mit ungültigen Koordinaten")
	}
	return p.Strich.pruefe()
}

// Kasten ist ein Rechteck mit Rand, wahlweise gefüllt (Fuellung.A > 0) und mit runden Ecken.
type Kasten struct {
	ID        int
	Bereich   Rechteck
	Strich    Strich
	Fuellung  color.NRGBA
	Eckradius float64
}

func (k Kasten) Kennung() int             { return k.ID }
func (k Kasten) Grenzen() Rechteck        { return aufgeweitet(k.Bereich, k.Strich.Breite) }
func (k Kasten) mitKennung(id int) Objekt { k.ID = id; return k }
func (k Kasten) pruefe() error {
	b := k.Bereich
	if !endlich(b.Min.X, b.Min.Y, b.Max.X, b.Max.Y, k.Eckradius) || b.Max.X < b.Min.X || b.Max.Y < b.Min.Y ||
		k.Eckradius < 0 {
		return fmt.Errorf("Kasten mit ungültigen Werten")
	}
	return k.Strich.pruefe()
}

// Text ist ein Textfeld; Ort ist die linke obere Ecke, Hintergrund mit A = 0 heißt „ohne“.
type Text struct {
	ID           int
	Ort          Punkt
	Inhalt       string
	Schriftgrad  float64 // in Pixeln des Originals
	Farbe        color.NRGBA
	Hintergrund  color.NRGBA
	BreiteGrenze float64 // Zeilenumbruch ab dieser Breite; 0 = ohne Umbruch
}

func (t Text) Kennung() int { return t.ID }

// Grenzen schätzt die Fläche grob (die genaue Größe kennt erst die Schriftrasterung): je Zeile Schriftgrad × 1,3,
// je Zeichen Schriftgrad × 0,6.
func (t Text) Grenzen() Rechteck {
	zeilen, laengste, akt := 1, 0, 0
	for _, r := range t.Inhalt {
		if r == '\n' {
			zeilen++
			akt = 0
			continue
		}
		akt++
		laengste = max(laengste, akt)
	}
	b := float64(laengste) * t.Schriftgrad * 0.6
	if t.BreiteGrenze > 0 {
		b = min(b, t.BreiteGrenze)
	}
	return Rechteck{t.Ort, t.Ort.Plus(Punkt{b, float64(zeilen) * t.Schriftgrad * 1.3})}
}
func (t Text) mitKennung(id int) Objekt { t.ID = id; return t }
func (t Text) pruefe() error {
	if !endlich(t.Ort.X, t.Ort.Y, t.Schriftgrad, t.BreiteGrenze) || t.Schriftgrad <= 0 || t.BreiteGrenze < 0 {
		return fmt.Errorf("Text mit ungültigen Werten")
	}
	return nil
}

// Nummer ist eine Schritt-Nummer im Kreis.
type Nummer struct {
	ID     int
	Mitte  Punkt
	Wert   int
	Radius float64
	Farbe  color.NRGBA
}

func (n Nummer) Kennung() int { return n.ID }
func (n Nummer) Grenzen() Rechteck {
	return Rechteck{n.Mitte.Minus(Punkt{n.Radius, n.Radius}), n.Mitte.Plus(Punkt{n.Radius, n.Radius})}
}
func (n Nummer) mitKennung(id int) Objekt { n.ID = id; return n }
func (n Nummer) pruefe() error {
	if !endlich(n.Mitte.X, n.Mitte.Y, n.Radius) || n.Radius <= 0 {
		return fmt.Errorf("Nummer mit ungültigen Werten")
	}
	return nil
}

func aufgeweitet(r Rechteck, um float64) Rechteck {
	h := um / 2
	return Rechteck{r.Min.Minus(Punkt{h, h}), r.Max.Plus(Punkt{h, h})}
}
