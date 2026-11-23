// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"fmt"
	"image/color"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Andiministrator/schnapp/internal/fonts"
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
	// Ebenenwerte liefert Name, Sichtbarkeit, Sperre und Deckkraft der Ebene (ADR-0032; über die Einbettung von Ebene).
	Ebenenwerte() Ebene
	// mitEbene liefert eine Kopie mit den Angaben e.
	mitEbene(e Ebene) Objekt
}

// Strich beschreibt Linienfarbe und -breite in Pixeln des Originals.
type Strich struct {
	Farbe  color.NRGBA
	Breite float64
}

func (s Strich) pruefe() error {
	if !groesse(s.Breite) {
		return fmt.Errorf("document: Strichbreite ungültig: %v", s.Breite)
	}
	return nil
}

// Pfeil ist eine quadratische Bézier-Kurve von A nach B mit Kontrollpunkt K; die Spitze sitzt bei B. Gerade ist er,
// wenn K auf der Mitte liegt. Der Nutzer zieht am Griff (Kurvenpunkt bei t = 0,5), nicht an K.
type Pfeil struct {
	Ebene
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
		return fmt.Errorf("document: Pfeil mit ungültigen Koordinaten")
	}
	if err := p.Ebene.pruefe(); err != nil {
		return err
	}
	return p.Strich.pruefe()
}

// Kasten ist ein Rechteck mit Rand, wahlweise gefüllt (Fuellung.A > 0) und mit runden Ecken.
type Kasten struct {
	Ebene
	ID        int
	Bereich   Rechteck
	Strich    Strich
	Fuellung  color.NRGBA
	Eckradius float64
}

// WirksamerEckradius ist der Radius, mit dem Renderer, Editor und SVG-Export eine Ecke wirklich runden: höchstens die
// halbe kürzere Seite (in beiden Achsen gleich, nie elliptisch), unter einem halben Pixel gar nicht (F-108).
func WirksamerEckradius(radius, breite, hoehe float64) float64 {
	radius = max(0, min(radius, min(breite, hoehe)/2))
	if radius < 0.5 {
		return 0
	}
	return radius
}

// EckigerRahmen zerlegt den Strich eines Kastens ohne Rundung (WirksamerEckradius = 0) in überlappungsfreie
// Rechtecke mit spitzen Außenecken (Gehrung, F-498): oben und unten über die ganze Breite, links und rechts dazwischen.
// Ist der Innenraum durch den Strich verschwunden (Breite oder Höhe höchstens eine Strichbreite), bleibt ein einziges
// Rechteck. Renderer und Editor zeichnen genau diese Rechtecke; das SVG sagt dasselbe mit stroke-linejoin="miter".
func EckigerRahmen(r Rechteck, strichBreite float64) []Rechteck {
	h := strichBreite / 2
	a, b := r.Min.Minus(Punkt{h, h}), r.Max.Plus(Punkt{h, h})
	ia, ib := r.Min.Plus(Punkt{h, h}), r.Max.Minus(Punkt{h, h})
	if ia.X >= ib.X || ia.Y >= ib.Y {
		return []Rechteck{{a, b}}
	}
	return []Rechteck{
		{a, Punkt{b.X, ia.Y}},
		{Punkt{a.X, ib.Y}, b},
		{Punkt{a.X, ia.Y}, Punkt{ia.X, ib.Y}},
		{Punkt{ib.X, ia.Y}, Punkt{b.X, ib.Y}},
	}
}

func (k Kasten) Kennung() int             { return k.ID }
func (k Kasten) Grenzen() Rechteck        { return aufgeweitet(k.Bereich, k.Strich.Breite) }
func (k Kasten) mitKennung(id int) Objekt { k.ID = id; return k }
func (k Kasten) pruefe() error {
	b := k.Bereich
	if !endlich(b.Min.X, b.Min.Y, b.Max.X, b.Max.Y) || !groesse(k.Eckradius) || b.Max.X < b.Min.X || b.Max.Y < b.Min.Y {
		return fmt.Errorf("document: Kasten mit ungültigen Werten")
	}
	if err := k.Ebene.pruefe(); err != nil {
		return err
	}
	return k.Strich.pruefe()
}

// Text ist ein Textfeld; Ort ist die linke obere Ecke, Hintergrund mit A = 0 heißt „ohne“.
type Text struct {
	Ebene
	ID           int
	Ort          Punkt
	Inhalt       string
	Schriftgrad  float64 // in Pixeln des Originals
	Farbe        color.NRGBA
	Hintergrund  color.NRGBA
	BreiteGrenze float64 // Zeilenumbruch ab dieser Breite; 0 = ohne Umbruch
	// Schrift ist der Familienname (leer = Go); fehlt die Familie, setzen alle in Go, der Name bleibt (ADR-0021
	// Nachtrag 10).
	Schrift         string
	Fett, Kursiv    bool
	Unterstrichen   bool
	Durchgestrichen bool
	Ausrichtung     Ausrichtung
}

// Ausrichtung ist die Lage jeder Zeile in der Breite des Textblocks (der längsten Zeile).
type Ausrichtung uint8

const (
	AusrichtungLinks Ausrichtung = iota
	AusrichtungMitte
	AusrichtungRechts
)

func (t Text) Kennung() int { return t.ID }

// Grenzen ist das Schild des Textes: die gemessene Tinte der Glyphen (Schriftdatei, textFlaeche) mit
// TextInnenabstand Schriftgraden Rand. Renderer, Editor und SVG füllen den Hintergrund genau hier, Treffer,
// Auswahlrahmen und Verschieben nutzen dieselbe Fläche (F-108, F-149, F-172). Der Umbruch bei BreiteGrenze ist nicht
// umgesetzt, die Fläche kappt deshalb nicht. Nur wenn die Schrift nicht zu laden ist, gilt die grobe Schätzung: je Zeile
// Schriftgrad × 1,3, je Zeichen Schriftgrad × 0,6, mit demselben Rand.
func (t Text) Grenzen() Rechteck {
	if r, ok := textFlaeche(t); ok {
		return r
	}
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
	d := Pt(TextInnenabstand*t.Schriftgrad, TextInnenabstand*t.Schriftgrad)
	b := float64(laengste) * t.Schriftgrad * 0.6
	return Rechteck{t.Ort.Minus(d), t.Ort.Plus(Punkt{b, float64(zeilen) * t.Schriftgrad * Zeilenabstand}).Plus(d)}
}
func (t Text) mitKennung(id int) Objekt { t.ID = id; return t }
func (t Text) pruefe() error {
	if !endlich(t.Ort.X, t.Ort.Y) || !groesse(t.Schriftgrad, t.BreiteGrenze) || t.Schriftgrad <= 0 || t.Schriftgrad > MaxSchrift {
		return fmt.Errorf("document: Text mit ungültigen Werten")
	}
	if (t.Schrift != "" && !fonts.NameGueltig(t.Schrift)) || t.Ausrichtung > AusrichtungRechts {
		return fmt.Errorf("document: Text mit ungültiger Schrift oder Ausrichtung")
	}
	if err := t.Ebene.pruefe(); err != nil {
		return err
	}
	// Zeichen, die XML nicht tragen kann, überstehen das Speichern nicht unverändert (F-62): ungültiges UTF-8 und
	// Steuerzeichen unter U+0020 außer Tabulator, Zeilenumbruch und Wagenrücklauf sowie U+FFFE/U+FFFF. Alles, was
	// XML 1.0 erlaubt, bleibt erlaubt — sonst würden Texte (etwa mit CRLF) beim Lesen strenger behandelt als beim
	// Schreiben (F-68).
	if !utf8.ValidString(t.Inhalt) {
		return fmt.Errorf("document: Text ist kein gültiges UTF-8")
	}
	if utf8.RuneCountInString(t.Inhalt) > MaxTextZeichen {
		return fmt.Errorf("document: Text hat mehr als %d Zeichen", MaxTextZeichen)
	}
	for _, r := range t.Inhalt {
		if (r < 0x20 && r != '\n' && r != '\t' && r != '\r') || r == 0xfffe || r == 0xffff {
			return fmt.Errorf("document: Text enthält das Steuerzeichen U+%04X", r)
		}
	}
	return nil
}

// Nummer ist eine Schritt-Nummer im Kreis.
type Nummer struct {
	Ebene
	ID     int
	Mitte  Punkt
	Wert   int
	Radius float64
	Farbe  color.NRGBA
}

func (n Nummer) Kennung() int { return n.ID }

// Grenzen umfasst den Kreis und die Ziffern: Bei drei und mehr Stellen ragen sie über den Kreis hinaus (F-172).
func (n Nummer) Grenzen() Rechteck {
	r := Rechteck{n.Mitte.Minus(Punkt{n.Radius, n.Radius}), n.Mitte.Plus(Punkt{n.Radius, n.Radius})}
	if z, ok, _ := nummerTinte(n); ok {
		r = r.Vereint(z)
	}
	return r
}
func (n Nummer) mitKennung(id int) Objekt { n.ID = id; return n }
func (n Nummer) pruefe() error {
	if !endlich(n.Mitte.X, n.Mitte.Y) || !groesse(n.Radius) || n.Radius <= 0 || n.Radius > MaxSchrift ||
		n.Wert < 0 || n.Wert > MaxNummer {
		return fmt.Errorf("document: Nummer mit ungültigen Werten")
	}
	return n.Ebene.pruefe()
}

func aufgeweitet(r Rechteck, um float64) Rechteck {
	h := um / 2
	return Rechteck{r.Min.Minus(Punkt{h, h}), r.Max.Plus(Punkt{h, h})}
}

// Zeichenbar liefert eine Textzeile so, wie Renderer und Editor sie zeichnen: Tabulator als vier Leerzeichen,
// andere Steuerzeichen (Wagenrücklauf, DEL, C1) weg — die Schrift hat für sie keine Glyphe und zeigte Ersatzkästchen
// (Kritik A-3, Verify V-3). Der gespeicherte Text bleibt unverändert.
func Zeichenbar(zeile string) string {
	zeile = strings.ReplaceAll(zeile, "\t", "    ")
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, zeile)
}
