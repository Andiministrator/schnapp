// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"math"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/Andiministrator/schnapp/internal/fonts"
)

// Schriftmaße (F-181): Ob ein Text oder eine Nummer in die Ansicht reicht, entscheidet die echte Tinte der Glyphen,
// nicht eine Schätzung je Zeichen. Text steht in der Schrift seines Objekts (internal/fonts, Ersatz Go; ADR-0021
// Nachtrag 10), Nummern in Go Bold, ohne Hinting; die Maße hier kommen aus denselben Schriftdateien wie in Renderer
// und Editor. So gilt dieselbe Regel für Editor, Renderer, SVG-Export, Treffer und Zähler (ADR-0023 Punkt 7).

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
	f      *sfnt.Font
	em     float64       // Einheiten je em
	ppem   fixed.Int26_6 // em in Einheiten: Maße kommen in Schrifteinheiten
	versal float64       // Versalhöhe in em
	// refOben und refUnten sind die Tinte der Referenzzeile relativ zur Grundlinie in em (y nach unten, refOben < 0).
	refOben, refUnten float64
	glyphen           map[sfnt.GlyphIndex]glyphMass
}

var (
	schriftMu     sync.Mutex // schützt alles hier, auch den Puffer (sfnt.Buffer ist nicht nebenläufig nutzbar)
	schriftPuffer sfnt.Buffer
	schriftenMass = map[*sfnt.Font]*schriftMass{} // je geladener Schriftdatei (fonts.Laden liefert je Schnitt eine)
	// ohneSchrift stellt die Schrift „nicht ladbar“ (nur Tests): Dann gilt die Schätzung (TestSchaetzungIstObermass).
	ohneSchrift bool
)

// massVon liefert die Maße der Schriftdatei f (aus dem Zwischenspeicher). Nur mit gehaltenem schriftMu.
func massVon(f *fonts.Schrift) *schriftMass {
	if ohneSchrift {
		return nil
	}
	if s, ok := schriftenMass[f.Font]; ok {
		return s
	}
	em := float64(f.Font.UnitsPerEm())
	s := &schriftMass{f: f.Font, em: em, ppem: fixed.Int26_6(f.Font.UnitsPerEm()) << 6, glyphen: map[sfnt.GlyphIndex]glyphMass{}}
	if m, err := f.Font.Metrics(&schriftPuffer, s.ppem, font.HintingNone); err == nil {
		s.versal = float64(m.CapHeight) / 64 / em
	}
	// Höhe der Zeile für das Schild: Oberlänge und Unterlänge der Schrift, unabhängig vom Inhalt (der Rahmen soll
	// beim Tippen nicht springen). Referenz sind Versalien, Oberlängen und Unterlängen.
	if _, t, ok := s.zeile("HdgjpÅ", 1); ok {
		s.refOben, s.refUnten = t.Min.Y, t.Max.Y
	}
	schriftenMass[f.Font] = s
	return s
}

// schriftFuer liefert die Maße von Go Regular bzw. Go Bold (Nummern). Nur mit gehaltenem schriftMu.
func schriftFuer(fett bool) *schriftMass {
	if fett {
		return massVon(fonts.Laden(fonts.Go, fonts.Fett))
	}
	return massVon(fonts.Laden(fonts.Go, fonts.Normal))
}

// Schriftdatei ist die Schrift, in der Messung, Renderer und Editor den Text setzen (ADR-0021 Nachtrag 10): Familie und
// Schnitt mit Ersatz, nie nil.
func (t Text) Schriftdatei() *fonts.Schrift {
	return fonts.Laden(t.Schrift, fonts.SchnittAus(t.Fett, t.Kursiv))
}

// SchriftFehlt meldet, dass die Familie des Texts hier fehlt und Go an ihrer Stelle steht.
func (t Text) SchriftFehlt() bool { return t.Schrift != "" && t.Schriftdatei().Ersatz }

// textSchrift liefert die Maße der Schrift von t. Nur mit gehaltenem schriftMu.
func textSchrift(t Text) *schriftMass { return massVon(t.Schriftdatei()) }

// glyph liefert die Maße der Glyphe für r wie font.Drawer sie zeichnet: Ein Zeichen ohne Glyphe erscheint als
// Ersatzkästchen (Glyphe 0). Der Vorschub gilt immer, wie ihn font.Face.GlyphAdvance liefert, mit dem Editor und
// Renderer die nächste Glyphe setzen (F-577): Lässt sich r nicht nachschlagen, zählt der Vorschub der Glyphe 0 und ok
// ist false (der Renderer zeichnet nichts, Unterschneidung gibt es dort nicht, KernPaar); sind die Umrisse unlesbar,
// bleibt der Vorschub und die Tinte ist leer. Nur mit schriftMu.
func (s *schriftMass) glyph(r rune) (gi sfnt.GlyphIndex, m glyphMass, ok bool) {
	gi, err := s.f.GlyphIndex(&schriftPuffer, r)
	ok = err == nil
	if err != nil {
		gi = 0
	}
	if m, hat := s.glyphen[gi]; hat {
		return gi, m, ok
	}
	e := func(v fixed.Int26_6) float64 { return float64(v) / 64 / s.em }
	b, vor, err := s.f.GlyphBounds(&schriftPuffer, gi, s.ppem, font.HintingNone)
	if err != nil {
		// Umrisse unlesbar (LoadGlyph scheitert): nur der Vorschub aus hmtx, wie font.Face.GlyphAdvance.
		a, err := s.f.GlyphAdvance(&schriftPuffer, gi, s.ppem, font.HintingNone)
		if err != nil {
			a = 0
		}
		m = glyphMass{vorschub: e(a), leer: true}
	} else {
		m = glyphMass{vorschub: e(vor), tinte: Rechteck{Min: Pt(e(b.Min.X), e(b.Min.Y)), Max: Pt(e(b.Max.X), e(b.Max.Y))},
			leer: b.Min.X >= b.Max.X || b.Min.Y >= b.Max.Y}
	}
	s.glyphen[gi] = m
	return gi, m, ok
}

// zeile misst eine Zeile im Schriftgrad grad (Pixel): Vorschub (wie font.MeasureString) und Tinte relativ zum
// Ursprung auf der Grundlinie, ohne Rand; ok ist false, wenn die Zeile keine Tinte hat. Unterschneidung im Grad wie
// Renderer und Editor (KernPaar). Nur mit schriftMu.
func (s *schriftMass) zeile(text string, grad float64) (vorschub float64, tinte Rechteck, ok bool) {
	x := 0.0
	var vorher sfnt.GlyphIndex
	vorherGut := false
	erste := true
	for _, r := range text {
		gi, m, gut := s.glyph(r)
		if !erste && gut && vorherGut {
			// Paarunterschneidung (GPOS, sonst kern) im Schriftgrad, wie Renderer und Editor (KernPaar; Sitzungs-QA B-4:
			// opentype.Face.Kern skaliert nicht mit der Größe). Ein Zeichen ohne Nachschlag hat keine (KernPaar: 0).
			if k, err := s.f.Kern(&schriftPuffer, vorher, gi, fixed.Int26_6(math.Round(grad*64)), font.HintingNone); err == nil {
				x += float64(k) / 64
			}
		}
		erste, vorher, vorherGut = false, gi, gut
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
	i, n     int
	vorschub float64 // Breite der Zeile bis hinter das letzte Zeichen (ohne Tinte am Ende der Zeile)
	tinte    Rechteck
	hatTinte bool // false: nur Leerraum, tinte ist leer, nur der Vorschub zählt (F-365)
}

// tintenSchluessel ist Schrift, Inhalt und Grad eines Textes: Die Tinte relativ zum Ort hängt nur davon ab.
type tintenSchluessel struct {
	f      *sfnt.Font
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
	// tintenEintrag schätzt, was ein Eintrag ohne Inhalt und Zeilen belegt: Schlüssel (Schrift, Zeichenkettenkopf und
	// Grad, 32 Byte), Slice-Kopf (24 Byte) und den Anteil der Map an Steuerbytes und Reserve.
	tintenEintrag = 96
	// tintenZeile ist die Größe einer zeilenTinte (zwei int, fünf float64, ein bool mit Füllbytes; der Test vergleicht mit unsafe.Sizeof).
	tintenZeile = 64
)

// tintenKosten schätzt die Belegung eines Eintrags mit Inhalt inhalt und zeilen gemessenen Zeilen (Kapazität).
func tintenKosten(inhalt string, zeilen int) int {
	return len(inhalt) + tintenEintrag + zeilen*tintenZeile
}

// textZeilen liefert die Zeilen von Inhalt mit Tinte oder mit Vorschub (nur Leerzeichen, hatTinte false, F-365), gemessen
// in Grad (aus dem Zwischenspeicher). Nur mit schriftMu.
func textZeilen(s *schriftMass, inhalt string, grad float64) []zeilenTinte {
	k := tintenSchluessel{s.f, inhalt, grad}
	if z, ok := tintenSpeicher[k]; ok {
		return z
	}
	zeilen := []zeilenTinte{}
	for i, z := range strings.Split(inhalt, "\n") {
		z = Zeichenbar(z)
		// Auch eine Zeile nur aus Leerzeichen zählt, wenn sie einen Vorschub hat (F-365): ohne Tinte, aber mit Breite.
		if vor, tinte, hat := s.zeile(z, grad); hat || vor > 0 {
			zeilen = append(zeilen, zeilenTinte{i: i, n: utf8.RuneCountInString(z), vorschub: vor, tinte: tinte, hatTinte: hat})
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

// Lage der Striche in Schriftgraden (ADR-0021 Nachtrag 10 Punkt 7): Oberkante des Unterstrichs unter der Grundlinie,
// Mitte des Durchstrichs über der Grundlinie, Dicke (mindestens ein Pixel).
const (
	UnterstrichAbstand = 0.1
	DurchstrichHoehe   = 0.3
	StrichDicke        = 0.06
)

// blockBreite ist der größte Vorschub der Zeilen: die Breite, in der die Ausrichtung jede Zeile setzt.
func blockBreite(zeilen []zeilenTinte) float64 {
	b := 0.0
	for _, z := range zeilen {
		b = max(b, z.vorschub)
	}
	return b
}

// versatz ist der Abstand einer Zeile mit Vorschub vorschub vom linken Rand des Blocks der Breite block.
func versatz(a Ausrichtung, block, vorschub float64) float64 {
	switch a {
	case AusrichtungMitte:
		return (block - vorschub) / 2
	case AusrichtungRechts:
		return block - vorschub
	}
	return 0
}

// striche liefert Unter- und Durchstriche der gemessenen Zeilen (Original). Nur mit gehaltenem schriftMu.
func (t Text) striche(zeilen []zeilenTinte) []Rechteck {
	if !t.Unterstrichen && !t.Durchgestrichen {
		return nil
	}
	grad, block := t.Schriftgrad, blockBreite(zeilen)
	d := max(1, StrichDicke*grad)
	var r []Rechteck
	for _, z := range zeilen {
		if z.vorschub <= 0 {
			continue
		}
		x := t.Ort.X + versatz(t.Ausrichtung, block, z.vorschub)
		g := t.Grundlinie(z.i)
		if t.Unterstrichen {
			y := g + UnterstrichAbstand*grad
			r = append(r, Rechteck{Min: Pt(x, y), Max: Pt(x+z.vorschub, y+d)})
		}
		if t.Durchgestrichen {
			y := g - DurchstrichHoehe*grad
			r = append(r, Rechteck{Min: Pt(x, y-d/2), Max: Pt(x+z.vorschub, y+d/2)})
		}
	}
	return r
}

// Striche liefert die Rechtecke von Unter- und Durchstrich (Original); Renderer, SVG-Export und Editor füllen sie in
// der Textfarbe.
func (t Text) Striche() []Rechteck {
	schriftMu.Lock()
	defer schriftMu.Unlock()
	s := textSchrift(t)
	if s == nil {
		return nil
	}
	return t.striche(textZeilen(s, t.Inhalt, t.Schriftgrad))
}

// TextZeile ist die Lage einer Zeile: sichtbarer Text (Zeichenbar), Anfang nach der Ausrichtung, Grundlinie und
// Vorschub, alles im Original.
type TextZeile struct {
	Text                    string
	X, Grundlinie, Vorschub float64
}

// Zeilen liefert jede Zeile des Texts (auch leere) mit ihrer Lage: Renderer, Editor und SVG-Export setzen sie dorthin.
func (t Text) Zeilen() []TextZeile {
	schriftMu.Lock()
	defer schriftMu.Unlock()
	var gemessen []zeilenTinte
	if s := textSchrift(t); s != nil {
		gemessen = textZeilen(s, t.Inhalt, t.Schriftgrad)
	}
	block := blockBreite(gemessen)
	vor := map[int]float64{}
	for _, z := range gemessen {
		vor[z.i] = z.vorschub
	}
	var r []TextZeile
	for i, z := range strings.Split(t.Inhalt, "\n") {
		v := vor[i]
		r = append(r, TextZeile{Text: Zeichenbar(z), X: t.Ort.X + versatz(t.Ausrichtung, block, v), Grundlinie: t.Grundlinie(i),
			Vorschub: v})
	}
	return r
}

// Vorschub ist die Breite von zeile (sichtbar gemacht mit Zeichenbar) in der Schrift und im Grad von t: die Lage der
// Schreibmarke hinter diesem Text.
func (t Text) Vorschub(zeile string) float64 {
	schriftMu.Lock()
	defer schriftMu.Unlock()
	s := textSchrift(t)
	if s == nil {
		return float64(utf8.RuneCountInString(Zeichenbar(zeile))) * t.Schriftgrad * 0.6 // wie die Schätzung in Grenzen
	}
	v, _, _ := s.zeile(Zeichenbar(zeile), t.Schriftgrad)
	return v
}

// textTinte ist die Fläche, die Renderer und Editor von t mit Tinte füllen (Original, mit Rand), ohne Hintergrund:
// Glyphen und Striche. Zeilen wie im Renderer: Grundlinie Ort.Y + Grad·(1 + 1,3·i), Anfang nach der Ausrichtung,
// Zeichenbar. ok ist false, wenn der Text keine Tinte hat (nur Leerraum ohne Striche); geschaetzt bleibt für Aufrufer,
// die Schrift ist immer ladbar (fonts.Laden fällt auf Go zurück).
func textTinte(t Text) (r Rechteck, ok, geschaetzt bool) {
	schriftMu.Lock()
	defer schriftMu.Unlock()
	s := textSchrift(t)
	if s == nil {
		return Rechteck{}, false, true
	}
	zeilen := textZeilen(s, t.Inhalt, t.Schriftgrad)
	block := blockBreite(zeilen)
	dazu := func(g Rechteck) {
		if ok {
			r = r.Vereint(g)
		} else {
			r, ok = g, true
		}
	}
	for _, z := range zeilen {
		if !z.hatTinte {
			continue
		}
		grund := Pt(t.Ort.X+versatz(t.Ausrichtung, block, z.vorschub), t.Grundlinie(z.i))
		dazu(mitRand(Rechteck{Min: grund.Plus(z.tinte.Min), Max: grund.Plus(z.tinte.Max)}, z.n))
	}
	for _, st := range t.striche(zeilen) {
		dazu(mitRand(st, 0))
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

// TextInnenabstand ist der Abstand zwischen Glyphen und Rand des Text-Schilds in Schriftgraden (F-149).
const TextInnenabstand = 0.25

// Zeilenabstand ist der Abstand der Grundlinien in Schriftgraden; die erste Grundlinie liegt einen Schriftgrad unter
// Ort.Y. Renderer, SVG-Export und Editor setzen jede Zeile mit derselben Zahl (F-108).
const Zeilenabstand = 1.3

// Grundlinie ist die senkrechte Lage der Grundlinie von Zeile i (ab 0) im Original.
func (t Text) Grundlinie(i int) float64 { return t.Ort.Y + t.Schriftgrad*(1+Zeilenabstand*float64(i)) }

// Grundlinie ist die senkrechte Lage der Grundlinie der Ziffern im Original: eine halbe Versalhöhe unter der Mitte
// (Renderer, Editor und SVG-Export setzen sie dorthin, F-108). Ohne ladbare Schrift gilt 0,36 Schriftgrad.
func (n Nummer) Grundlinie() float64 {
	grad := 1.2 * n.Radius
	schriftMu.Lock()
	defer schriftMu.Unlock()
	if s := schriftFuer(true); s != nil {
		return n.Mitte.Y + s.versal*grad/2
	}
	return n.Mitte.Y + 0.36*grad
}

// NummerTinte ist die Fläche der Ziffern einer Nummer (Original, mit Rand); ok ist false ohne ladbare Schrift.
func NummerTinte(n Nummer) (Rechteck, bool) {
	r, ok, _ := nummerTinte(n)
	return r, ok
}

// textFlaeche ist das Schild eines Textes: die gemessene Tinte und die Zeilenhöhe der Schrift (Oberlänge bis
// Unterlänge je Zeile, damit der Rahmen beim Tippen nicht springt), je Seite um TextInnenabstand Schriftgrade
// geweitet. Eine Quelle für Hintergrund in Renderer, Editor und SVG, Treffer, Auswahlrahmen und Verschieben (F-108,
// F-149, F-172). ok ist false, wenn die Schrift nicht zu laden war.
func textFlaeche(t Text) (Rechteck, bool) {
	schriftMu.Lock()
	defer schriftMu.Unlock()
	s := textSchrift(t)
	if s == nil {
		return Rechteck{}, false
	}
	grad := t.Schriftgrad
	zeilen := strings.Count(t.Inhalt, "\n") + 1
	r := Rechteck{
		Min: Pt(t.Ort.X, t.Grundlinie(0)+s.refOben*grad),
		Max: Pt(t.Ort.X, t.Grundlinie(zeilen-1)+s.refUnten*grad),
	}
	gemessen := textZeilen(s, t.Inhalt, grad)
	block := blockBreite(gemessen)
	for _, z := range gemessen {
		x := t.Ort.X + versatz(t.Ausrichtung, block, z.vorschub)
		g := Pt(x, t.Grundlinie(z.i))
		if !z.hatTinte {
			// Nur Leerraum: die Zeile reicht um ihren Vorschub und die Zeilenhöhe der Schrift.
			r = r.Vereint(Rechteck{Min: Pt(x, g.Y+s.refOben*grad), Max: Pt(x+z.vorschub, g.Y+s.refUnten*grad)})
			continue
		}
		ink := Rechteck{Min: g.Plus(z.tinte.Min), Max: g.Plus(z.tinte.Max)}
		r = r.Vereint(ink).Vereint(Rechteck{Min: Pt(x, ink.Min.Y), Max: Pt(x+z.vorschub, ink.Max.Y)})
	}
	for _, st := range t.striche(gemessen) {
		r = r.Vereint(st)
	}
	d := TextInnenabstand * grad
	return Rechteck{Min: r.Min.Minus(Pt(d, d)), Max: r.Max.Plus(Pt(d, d))}, true
}

// KernPaar ist die Paarunterschneidung zwischen a und b in der Schrift f im Grad grad (26.6-Pixel): GPOS, sonst die
// kern-Tabelle, skaliert mit dem Grad. Renderer und Editor setzen damit wie die Messung (ADR-0021 Nachtrag 10 Punkt 5);
// b ist ein eigener Puffer des Aufrufers.
func KernPaar(f *sfnt.Font, b *sfnt.Buffer, a, c rune, grad float64) fixed.Int26_6 {
	ga, err := f.GlyphIndex(b, a)
	if err != nil {
		return 0
	}
	gc, err := f.GlyphIndex(b, c)
	if err != nil {
		return 0
	}
	k, err := f.Kern(b, ga, gc, fixed.Int26_6(math.Round(grad*64)), font.HintingNone)
	if err != nil {
		return 0
	}
	return k
}
