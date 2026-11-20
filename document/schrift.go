// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// Schriftmaße (F-181): Ob ein Text oder eine Nummer in die Ansicht reicht, entscheidet die echte Tinte der Glyphen,
// nicht eine Schätzung je Zeichen. Renderer und Editor zeichnen Text in Go Regular und Nummern in Go Bold, ohne
// Hinting; die Maße hier kommen aus denselben Schriftdateien (golang.org/x/image/font/gofont). So gilt dieselbe Regel
// für Editor, Renderer, SVG-Export, Treffer und Zähler, gleich welches Paket gebunden ist (ADR-0023 Punkt 7).

// tintenRand weitet die Tinte aus: Kantenglättung (1 Pixel) und Rundung der Glyphenorte auf 1/64 Pixel je Glyphe
// (rundungJeGlyphe) beim Rastern.
const (
	tintenRand      = 1.0
	rundungJeGlyphe = 1.0 / 64
)

// glyphMass sind Vorschub und Tinte einer Glyphe in em, die Tinte relativ zum Ursprung auf der Grundlinie (y nach
// unten); ohne Tinte (Leerzeichen) ist leer gesetzt.
type glyphMass struct {
	vorschub float64
	tinte    Rechteck
	leer     bool
}

type schriftMass struct {
	f       *sfnt.Font
	em      float64       // Einheiten je em
	ppem    fixed.Int26_6 // em in Einheiten: Maße kommen in Schrifteinheiten
	versal  float64       // Versalhöhe in em
	glyphen map[sfnt.GlyphIndex]glyphMass
}

var (
	schriftMu      sync.Mutex // schützt alles hier, auch den Puffer (sfnt.Buffer ist nicht nebenläufig nutzbar)
	schriftPuffer  sfnt.Buffer
	schriftGeladen bool
	schriftenMass  [2]*schriftMass // 0 Go Regular (Text), 1 Go Bold (Nummer)
)

// schriftFuer liefert die Maße der Schrift; nil, wenn sie sich nicht laden lässt (dann schätzt Reichweite). Nur mit
// gehaltenem schriftMu.
func schriftFuer(fett bool) *schriftMass {
	if !schriftGeladen {
		schriftGeladen = true
		for i, roh := range [][]byte{goregular.TTF, gobold.TTF} {
			f, err := sfnt.Parse(roh)
			if err != nil {
				continue
			}
			em := float64(f.UnitsPerEm())
			s := &schriftMass{f: f, em: em, ppem: fixed.Int26_6(f.UnitsPerEm()) << 6, glyphen: map[sfnt.GlyphIndex]glyphMass{}}
			if m, err := f.Metrics(&schriftPuffer, s.ppem, font.HintingNone); err == nil {
				s.versal = float64(m.CapHeight) / 64 / em
			}
			schriftenMass[i] = s
		}
	}
	if fett {
		return schriftenMass[1]
	}
	return schriftenMass[0]
}

// glyph liefert die Maße der Glyphe für r wie font.Drawer sie zeichnet: Ein Zeichen ohne Glyphe erscheint als
// Ersatzkästchen (Glyphe 0); lässt sich r nicht nachschlagen, zeichnet der Renderer nichts. Nur mit schriftMu.
func (s *schriftMass) glyph(r rune) (sfnt.GlyphIndex, glyphMass, bool) {
	gi, err := s.f.GlyphIndex(&schriftPuffer, r)
	if err != nil {
		return 0, glyphMass{}, false
	}
	if m, ok := s.glyphen[gi]; ok {
		return gi, m, true
	}
	b, vor, err := s.f.GlyphBounds(&schriftPuffer, gi, s.ppem, font.HintingNone)
	if err != nil {
		return 0, glyphMass{}, false
	}
	e := func(v fixed.Int26_6) float64 { return float64(v) / 64 / s.em }
	m := glyphMass{vorschub: e(vor), tinte: Rechteck{Min: Pt(e(b.Min.X), e(b.Min.Y)), Max: Pt(e(b.Max.X), e(b.Max.Y))},
		leer: b.Min.X >= b.Max.X || b.Min.Y >= b.Max.Y}
	s.glyphen[gi] = m
	return gi, m, true
}

// zeile misst eine Zeile im Schriftgrad grad (Pixel): Vorschub (wie font.MeasureString) und Tinte relativ zum
// Ursprung auf der Grundlinie, ohne Rand; ok ist false, wenn die Zeile keine Tinte hat. Unterschneidung wie der
// Renderer (opentype.Face.Kern liefert sie unskaliert in Schrifteinheiten). Nur mit schriftMu.
func (s *schriftMass) zeile(text string, grad float64) (vorschub float64, tinte Rechteck, ok bool) {
	x := 0.0
	var vorher sfnt.GlyphIndex
	erste := true
	for _, r := range text {
		gi, m, gut := s.glyph(r)
		if !erste {
			// ppem wie opentype.Face.Kern (Einheiten je em als 26.6-Rohwert), damit dieselbe Zahl herauskommt.
			if k, err := s.f.Kern(&schriftPuffer, vorher, gi, fixed.Int26_6(s.f.UnitsPerEm()), font.HintingNone); err == nil {
				x += float64(k) / 64
			}
		}
		erste, vorher = false, gi
		if !gut {
			continue
		}
		if !m.leer {
			g := Rechteck{Min: Pt(x+m.tinte.Min.X*grad, m.tinte.Min.Y*grad), Max: Pt(x+m.tinte.Max.X*grad, m.tinte.Max.Y*grad)}
			if ok {
				tinte = tinte.Vereint(g)
			} else {
				tinte, ok = g, true
			}
		}
		x += m.vorschub * grad
	}
	return x, tinte, ok
}

// mitRand weitet r um die Kantenglättung und die Rundung von n Glyphen aus.
func mitRand(r Rechteck, n int) Rechteck {
	d := tintenRand + rundungJeGlyphe*float64(n)
	return Rechteck{Min: r.Min.Minus(Pt(d, tintenRand)), Max: r.Max.Plus(Pt(d, tintenRand))}
}

// zeilenTinte ist die Tinte einer Zeile relativ zu ihrer Grundlinie: i ist die Zeilennummer, n die Zahl der Runen.
type zeilenTinte struct {
	i, n  int
	tinte Rechteck
}

// tintenSchluessel ist Inhalt und Grad eines Textes: Die Tinte relativ zum Ort hängt nur davon ab.
type tintenSchluessel struct {
	inhalt string
	grad   float64
}

// tintenSpeicher hält die gemessenen Zeilen je Inhalt und Grad (F-224): Editor, Renderer und Trefferprüfung fragen die
// Reichweite jedes Textes bei jedem Bild ab, auch ganz verborgener; ohne Zwischenspeicher würde jede Glyphe jedes Mal
// neu vermessen. Die Schlüssel teilen sich den Speicher mit dem Text; damit alte Fassungen (Tippen, Verlauf) nicht
// unbegrenzt am Leben bleiben, wird er geleert, sobald die geschätzte Belegung tintenSpeicherMax Bytes übersteigt.
// Gezählt werden Inhalt, Eintrag und jede gemessene Zeile (F-241): Zeilenreiche Texte belegen sonst ein Vielfaches
// ihres Inhalts, und ein leerer Inhalt zählte gar nicht. Nur mit schriftMu.
var (
	tintenSpeicher      = map[tintenSchluessel][]zeilenTinte{}
	tintenSpeicherBytes int
	tintenSpeicherMax   = 64 << 20 // Variable nur für den Test des Leerens
)

const (
	// tintenEintrag schätzt, was ein Eintrag ohne Inhalt und Zeilen belegt: Schlüssel (Zeichenkettenkopf und Grad,
	// 24 Byte), Slice-Kopf (24 Byte) und den Anteil der Map an Steuerbytes und Reserve.
	tintenEintrag = 96
	// tintenZeile ist die Größe einer zeilenTinte (zwei int, vier float64; der Test vergleicht mit unsafe.Sizeof).
	tintenZeile = 48
)

// tintenKosten schätzt die Belegung eines Eintrags mit Inhalt inhalt und zeilen gemessenen Zeilen (Kapazität).
func tintenKosten(inhalt string, zeilen int) int {
	return len(inhalt) + tintenEintrag + zeilen*tintenZeile
}

// textZeilen liefert die Zeilen von Inhalt mit Tinte, gemessen in Grad (aus dem Zwischenspeicher). Nur mit schriftMu.
func textZeilen(s *schriftMass, inhalt string, grad float64) []zeilenTinte {
	k := tintenSchluessel{inhalt, grad}
	if z, ok := tintenSpeicher[k]; ok {
		return z
	}
	zeilen := []zeilenTinte{}
	for i, z := range strings.Split(inhalt, "\n") {
		z = Zeichenbar(z)
		if _, tinte, hat := s.zeile(z, grad); hat {
			zeilen = append(zeilen, zeilenTinte{i: i, n: utf8.RuneCountInString(z), tinte: tinte})
		}
	}
	kosten := tintenKosten(inhalt, cap(zeilen))
	if tintenSpeicherBytes+kosten > tintenSpeicherMax {
		clear(tintenSpeicher)
		tintenSpeicherBytes = 0
	}
	tintenSpeicher[k] = zeilen
	tintenSpeicherBytes += kosten
	return zeilen
}

// textTinte ist die Fläche, die Renderer und Editor von t mit Tinte füllen (Original, mit Rand), ohne Hintergrund.
// Zeilen wie im Renderer: Grundlinie Ort.Y + Grad·(1 + 1,3·i), Zeichenbar. ok ist false, wenn der Text keine Tinte hat
// (nur Leerraum); geschaetzt, wenn die Schrift nicht zu laden war.
func textTinte(t Text) (r Rechteck, ok, geschaetzt bool) {
	schriftMu.Lock()
	defer schriftMu.Unlock()
	s := schriftFuer(false)
	if s == nil {
		return Rechteck{}, false, true
	}
	for _, z := range textZeilen(s, t.Inhalt, t.Schriftgrad) {
		grund := Pt(t.Ort.X, t.Ort.Y+t.Schriftgrad*(1+1.3*float64(z.i)))
		g := mitRand(Rechteck{Min: grund.Plus(z.tinte.Min), Max: grund.Plus(z.tinte.Max)}, z.n)
		if ok {
			r = r.Vereint(g)
		} else {
			r, ok = g, true
		}
	}
	return r, ok, false
}

// nummerTinte ist die Tinte der Ziffern einer Schritt-Nummer wie im Renderer: Go Bold im Grad 1,2·Radius, waagrecht
// mittig nach dem Vorschub, Grundlinie eine halbe Versalhöhe unter der Mitte.
func nummerTinte(n Nummer) (r Rechteck, ok, geschaetzt bool) {
	schriftMu.Lock()
	defer schriftMu.Unlock()
	s := schriftFuer(true)
	if s == nil {
		return Rechteck{}, false, true
	}
	z := strconv.Itoa(n.Wert)
	grad := 1.2 * n.Radius
	breite, tinte, hat := s.zeile(z, grad)
	if !hat {
		return Rechteck{}, false, false
	}
	grund := Pt(n.Mitte.X-breite/2, n.Mitte.Y+s.versal*grad/2)
	return mitRand(Rechteck{Min: grund.Plus(tinte.Min), Max: grund.Plus(tinte.Max)}, len(z)), true, false
}
