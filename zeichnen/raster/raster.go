// SPDX-License-Identifier: GPL-3.0-or-later

// Paket raster zeichnet SVG-Dateien als Bild, ohne rsvg-convert. Nur Standardbibliothek. Es kann genau das, was
// Illustrator- und eigene SVGs brauchen: path, rect
// (auch mit rx/ry), circle, ellipse, polygon, polyline, line, g und svg mit transform (matrix, translate, scale,
// rotate, skewX, skewY), Klassen aus <style>, Füllung und Strich (Enden, Ecken, Gehrung, Strichelung, Füllregel),
// opacity, mask (maskUnits userSpaceOnUse, Luminanz) und clipPath (auch mit eigenem transform wie bei Illustrator).
// Alles andere (Text, Verläufe, Filter, Bilder, use, Stil-Attribut, unbekannte Attribute oder Eigenschaften) ist ein
// Fehler, kein stilles Übergehen.
//
// Bildgröße wie rsvg-convert -w: Breite vorgegeben, Höhe aufgerundet, viewBox gleichmäßig skaliert, oben bündig.
package raster

import (
	"fmt"
	"image"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/Andiministrator/schnapp/zeichnen/geo"
	"github.com/Andiministrator/schnapp/zeichnen/svg"
)

// Feinheit: größter Abstand zweier Punkte auf zerlegten Kurven, in Gerätepixeln.
const Feinheit = 0.25

// --- Matrix --------------------------------------------------------------------------------------------------

// matrix a b c d e f: x' = a x + c y + e, y' = b x + d y + f.
type matrix [6]float64

var einheit = matrix{1, 0, 0, 1, 0, 0}

func (m matrix) mal(n matrix) matrix {
	return matrix{
		m[0]*n[0] + m[2]*n[1], m[1]*n[0] + m[3]*n[1],
		m[0]*n[2] + m[2]*n[3], m[1]*n[2] + m[3]*n[3],
		m[0]*n[4] + m[2]*n[5] + m[4], m[1]*n[4] + m[3]*n[5] + m[5],
	}
}

func (m matrix) punkt(p geo.P) geo.P {
	return geo.P{X: m[0]*p.X + m[2]*p.Y + m[4], Y: m[1]*p.X + m[3]*p.Y + m[5]}
}

// massstab: größte Streckung der Matrix (für die Feinheit der Zerlegung).
func (m matrix) massstab() float64 {
	return math.Max(math.Hypot(m[0], m[1]), math.Hypot(m[2], m[3]))
}

var transformTeil = regexp.MustCompile(`^[\s,]*([a-zA-Z]+)\s*\(([^)]*)\)`)
var zahl = regexp.MustCompile(svg.ZahlMuster)

func zahlen(s string) ([]float64, error) {
	var aus []float64
	rest := strings.TrimSpace(zahl.ReplaceAllString(s, ""))
	if strings.Trim(rest, " ,") != "" {
		return nil, fmt.Errorf("Zahlen nicht lesbar: %q", s)
	}
	for _, t := range zahl.FindAllString(s, -1) {
		f, err := strconv.ParseFloat(t, 64)
		if err != nil || math.IsInf(f, 0) {
			return nil, fmt.Errorf("Zahl %q nicht lesbar", t)
		}
		aus = append(aus, f)
	}
	return aus, nil
}

func transform(s string) (matrix, error) {
	m := einheit
	rest := s
	for strings.Trim(rest, " ,\t") != "" {
		t := transformTeil.FindStringSubmatch(rest)
		if t == nil {
			return m, fmt.Errorf("transform nicht lesbar: %q", s)
		}
		rest = rest[len(t[0]):]
		a, err := zahlen(t[2])
		if err != nil {
			return m, err
		}
		var n matrix
		falsch := fmt.Errorf("transform %s mit %d Zahlen: %q", t[1], len(a), s)
		switch t[1] {
		case "matrix":
			if len(a) != 6 {
				return m, falsch
			}
			n = matrix{a[0], a[1], a[2], a[3], a[4], a[5]}
		case "translate":
			switch len(a) {
			case 1:
				n = matrix{1, 0, 0, 1, a[0], 0}
			case 2:
				n = matrix{1, 0, 0, 1, a[0], a[1]}
			default:
				return m, falsch
			}
		case "scale":
			switch len(a) {
			case 1:
				n = matrix{a[0], 0, 0, a[0], 0, 0}
			case 2:
				n = matrix{a[0], 0, 0, a[1], 0, 0}
			default:
				return m, falsch
			}
		case "rotate":
			if len(a) != 1 && len(a) != 3 {
				return m, falsch
			}
			w := a[0] * math.Pi / 180
			c, si := math.Cos(w), math.Sin(w)
			n = matrix{c, si, -si, c, 0, 0}
			if len(a) == 3 {
				n = matrix{1, 0, 0, 1, a[1], a[2]}.mal(n).mal(matrix{1, 0, 0, 1, -a[1], -a[2]})
			}
		case "skewX":
			if len(a) != 1 {
				return m, falsch
			}
			n = matrix{1, 0, math.Tan(a[0] * math.Pi / 180), 1, 0, 0}
		case "skewY":
			if len(a) != 1 {
				return m, falsch
			}
			n = matrix{1, math.Tan(a[0] * math.Pi / 180), 0, 1, 0, 0}
		default:
			return m, fmt.Errorf("transform %s nicht unterstützt", t[1])
		}
		m = m.mal(n)
	}
	return m, nil
}

// --- Farben und Stil -----------------------------------------------------------------------------------------

type farbe struct {
	keine   bool
	r, g, b float32
}

var namen = map[string]string{"black": "#000000", "white": "#ffffff"}

func leseFarbe(v string) (farbe, error) {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "none" {
		return farbe{keine: true}, nil
	}
	if n, ok := namen[v]; ok {
		v = n
	}
	if !strings.HasPrefix(v, "#") || (len(v) != 4 && len(v) != 7) {
		return farbe{}, fmt.Errorf("Farbe %q nicht unterstützt", v)
	}
	h := v[1:]
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	x, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return farbe{}, fmt.Errorf("Farbe %q nicht lesbar", v)
	}
	return farbe{r: float32(x>>16) / 255, g: float32(x>>8&255) / 255, b: float32(x&255) / 255}, nil
}

func laenge(v string) (float64, error) {
	v = strings.TrimSpace(v)
	v = strings.TrimSuffix(v, "px")
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, fmt.Errorf("Länge %q nicht lesbar", v)
	}
	return f, nil
}

// stil: geerbte Eigenschaften.
type stil struct {
	fill, stroke         farbe
	strich               strichArt
	evenodd, clipEvenodd bool
}

var startStil = stil{fill: farbe{}, stroke: farbe{keine: true},
	strich: strichArt{breite: 1, ende: "butt", ecke: "miter", gehrung: 4}}

// eigen: nicht geerbte Eigenschaften eines Elements.
type eigen struct {
	clip, maske string
	deckkraft   float64
}

// Eigenschaften, die als Attribut oder in Klassenregeln vorkommen dürfen.
var eigenschaften = map[string]bool{"fill": true, "stroke": true, "stroke-width": true, "stroke-linecap": true,
	"stroke-linejoin": true, "stroke-miterlimit": true, "stroke-dasharray": true, "fill-rule": true,
	"clip-rule": true, "clip-path": true, "mask": true, "opacity": true}

func urlZiel(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "none" {
		return "", nil
	}
	if !strings.HasPrefix(v, "url(#") || !strings.HasSuffix(v, ")") {
		return "", fmt.Errorf("Verweis %q nicht unterstützt", v)
	}
	return strings.Trim(v[5:len(v)-1], `"' `), nil
}

func (st *stil) setzen(e *eigen, k, v string) error {
	var err error
	switch k {
	case "fill":
		st.fill, err = leseFarbe(v)
	case "stroke":
		st.stroke, err = leseFarbe(v)
	case "stroke-width":
		st.strich.breite, err = laenge(v)
	case "stroke-miterlimit":
		st.strich.gehrung, err = laenge(v)
	case "stroke-linecap":
		if v != "butt" && v != "round" && v != "square" {
			return fmt.Errorf("stroke-linecap %q", v)
		}
		st.strich.ende = v
	case "stroke-linejoin":
		if v != "miter" && v != "round" && v != "bevel" {
			return fmt.Errorf("stroke-linejoin %q", v)
		}
		st.strich.ecke = v
	case "stroke-dasharray":
		if v == "none" {
			st.strich.strichelung = nil
			return nil
		}
		var a []float64
		if a, err = zahlen(v); err == nil {
			for _, x := range a {
				if x < 0 {
					return fmt.Errorf("stroke-dasharray %q", v)
				}
			}
			summe := 0.0
			for _, x := range a {
				summe += x
			}
			st.strich.strichelung = a
			if summe == 0 { // Muster mit Summe 0 gilt als none
				st.strich.strichelung = nil
			}
		}
	case "fill-rule", "clip-rule":
		if v != "nonzero" && v != "evenodd" {
			return fmt.Errorf("%s %q", k, v)
		}
		if k == "fill-rule" {
			st.evenodd = v == "evenodd"
		} else {
			st.clipEvenodd = v == "evenodd"
		}
	case "clip-path":
		e.clip, err = urlZiel(v)
	case "mask":
		e.maske, err = urlZiel(v)
	case "opacity":
		e.deckkraft, err = strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil || math.IsNaN(e.deckkraft) {
			return fmt.Errorf("opacity %q nicht lesbar", v)
		}
		e.deckkraft = math.Max(0, math.Min(1, e.deckkraft))
	default:
		return fmt.Errorf("Eigenschaft %s nicht unterstützt", k)
	}
	return err
}

const svgNS = "http://www.w3.org/2000/svg"

// --- Elemente und Attribute ----------------------------------------------------------------------------------

var formen = map[string]bool{"path": true, "rect": true, "circle": true, "ellipse": true, "polygon": true,
	"polyline": true, "line": true}

// gruppen: Elemente mit Kindern, die gezeichnet werden.
var gruppen = map[string]bool{"svg": true, "g": true}

// still: Elemente, die nicht selbst gezeichnet werden.
var still = map[string]bool{"defs": true, "style": true, "title": true, "desc": true, "mask": true, "clipPath": true,
	"metadata": true}

var geometrie = map[string][]string{
	"path": {"d"}, "polygon": {"points"}, "polyline": {"points"},
	"rect":   {"x", "y", "width", "height", "rx", "ry"},
	"circle": {"cx", "cy", "r"}, "ellipse": {"cx", "cy", "rx", "ry"}, "line": {"x1", "y1", "x2", "y2"},
	"svg": {"viewBox", "width", "height", "x", "y", "version"}, "mask": {"x", "y", "width", "height", "maskUnits"},
	"clipPath": {"clipPathUnits"},
}

func attributErlaubt(tag, k string) bool {
	switch {
	case k == "id" || k == "class" || k == "transform" || k == "role" || k == "xmlns":
		return true
	case strings.HasPrefix(k, "data-") || strings.HasPrefix(k, "aria-"):
		return true
	case eigenschaften[k]:
		return true
	}
	for _, g := range geometrie[tag] {
		if g == k {
			return true
		}
	}
	return false
}

// --- Ebenen --------------------------------------------------------------------------------------------------

// ebene: Bild mit vormultipliziertem RGBA in float32.
type ebene struct {
	b, h int
	px   []float32
}

func neueEbene(b, h int) *ebene { return &ebene{b: b, h: h, px: make([]float32, b*h*4)} }

// malen trägt eine Farbe mit der Deckung der Maske auf (source-over).
func (z *ebene) malen(m *Maske, f farbe) {
	if m == nil {
		return
	}
	for y := 0; y < m.H; y++ {
		for x := 0; x < m.B; x++ {
			c := m.A[y*m.B+x]
			if c == 0 {
				continue
			}
			i := ((m.Y0+y)*z.b + m.X0 + x) * 4
			r := 1 - c
			z.px[i] = f.r*c + z.px[i]*r
			z.px[i+1] = f.g*c + z.px[i+1]*r
			z.px[i+2] = f.b*c + z.px[i+2]*r
			z.px[i+3] = c + z.px[i+3]*r
		}
	}
}

// auftragen legt eine Ebene mit einem Faktor je Pixel (nil = 1) und einer Deckkraft auf diese (source-over).
func (z *ebene) auftragen(q *ebene, faktor []float32, deckkraft float32) {
	for p := 0; p < z.b*z.h; p++ {
		k := deckkraft
		if faktor != nil {
			k *= faktor[p]
		}
		i := p * 4
		a := q.px[i+3] * k
		if a == 0 {
			continue
		}
		r := 1 - a
		z.px[i] = q.px[i]*k + z.px[i]*r
		z.px[i+1] = q.px[i+1]*k + z.px[i+1]*r
		z.px[i+2] = q.px[i+2]*k + z.px[i+2]*r
		z.px[i+3] = a + z.px[i+3]*r
	}
}

// --- Zeichnen ------------------------------------------------------------------------------------------------

type zeichner struct {
	b, h   int
	ids    map[string]*svg.Element
	eltern map[*svg.Element]*svg.Element
	regeln map[string]*svg.Regel
	tiefe  int
	// box: rechteckiger Clip in Gerätekoordinaten (x0, y0, x1, y1), nil ohne
	box *[4]float64
}

func (z *zeichner) fehler(e *svg.Element, format string, a ...any) error {
	wo := "<" + e.Tag
	if id := e.Get("id"); id != "" {
		wo += ` id="` + id + `"`
	}
	return fmt.Errorf("%s>: %s", wo, fmt.Sprintf(format, a...))
}

// ableiten: Stil eines Elements aus dem geerbten, den Attributen und den Klassenregeln (Regeln vor Attributen,
// wie CSS gegenüber Präsentationsattributen).
func (z *zeichner) ableiten(e *svg.Element, erbe stil) (stil, eigen, error) {
	st := erbe
	st.strich.strichelung = erbe.strich.strichelung
	ei := eigen{deckkraft: 1}
	for _, a := range e.Attr {
		k := a.Name.Local
		if a.Name.Space == "xmlns" || (a.Name.Space != "" && a.Name.Space != svgNS) {
			continue
		}
		if k == "style" {
			return st, ei, z.fehler(e, "Attribut style nicht unterstützt")
		}
		if !attributErlaubt(e.Tag, k) {
			return st, ei, z.fehler(e, "Attribut %s nicht unterstützt", k)
		}
		if eigenschaften[k] {
			if err := st.setzen(&ei, k, a.Value); err != nil {
				return st, ei, z.fehler(e, "%v", err)
			}
		}
	}
	gesetzt := map[string]string{}
	for _, kl := range strings.Fields(e.Get("class")) {
		r := z.regeln[kl]
		if r == nil {
			continue
		}
		for _, k := range r.Schl {
			v := r.Werte[k]
			if alt, ok := gesetzt[k]; ok && alt != v {
				return st, ei, z.fehler(e, "Klassen setzen %s verschieden (%s, %s)", k, alt, v)
			}
			gesetzt[k] = v
			if err := st.setzen(&ei, k, v); err != nil {
				return st, ei, z.fehler(e, "Klasse %s: %v", kl, err)
			}
		}
	}
	return st, ei, nil
}

// geerbt: Stil, den ein Element von seinen Vorfahren erbt (für mask und clipPath, deren Inhalt vom Ort der
// Definition erbt, nicht vom verweisenden Element).
func (z *zeichner) geerbt(e *svg.Element) (stil, error) {
	var kette []*svg.Element
	for p := z.eltern[e]; p != nil; p = z.eltern[p] {
		kette = append(kette, p)
	}
	st := startStil
	for i := len(kette) - 1; i >= 0; i-- {
		var err error
		if st, _, err = z.ableiten(kette[i], st); err != nil {
			return st, err
		}
	}
	return st, nil
}

func (z *zeichner) zahl(e *svg.Element, k string, vorgabe float64) (float64, error) {
	if !e.Has(k) {
		return vorgabe, nil
	}
	f, err := laenge(e.Get(k))
	if err != nil {
		return 0, z.fehler(e, "%s: %v", k, err)
	}
	return f, nil
}

// teilpfade: Geometrie einer Form in Benutzereinheiten, zerlegt mit der Schrittweite schritt.
func (z *zeichner) teilpfade(e *svg.Element, schritt float64) ([]svg.Teilpfad, error) {
	r := func(f float64) string { return strconv.FormatFloat(f, 'g', -1, 64) }
	var d string
	switch e.Tag {
	case "path":
		d = e.Get("d")
	case "polygon", "polyline":
		a, err := zahlen(e.Get("points"))
		if err != nil || len(a)%2 == 1 {
			return nil, z.fehler(e, "points nicht lesbar")
		}
		if len(a) < 4 {
			return nil, nil
		}
		var b strings.Builder
		for i := 0; i < len(a); i += 2 {
			if i == 0 {
				b.WriteString("M")
			} else {
				b.WriteString("L")
			}
			b.WriteString(r(a[i]) + "," + r(a[i+1]))
		}
		if e.Tag == "polygon" {
			b.WriteString("Z")
		}
		d = b.String()
	case "rect":
		var v [6]float64
		for i, k := range []string{"x", "y", "width", "height"} {
			f, err := z.zahl(e, k, 0)
			if err != nil {
				return nil, err
			}
			v[i] = f
		}
		x, y, w, h := v[0], v[1], v[2], v[3]
		if w <= 0 || h <= 0 {
			return nil, nil
		}
		rx, err := z.zahl(e, "rx", -1)
		if err != nil {
			return nil, err
		}
		ry, err := z.zahl(e, "ry", -1)
		if err != nil {
			return nil, err
		}
		if rx < 0 {
			rx = ry
		}
		if ry < 0 {
			ry = rx
		}
		rx, ry = math.Max(0, math.Min(rx, w/2)), math.Max(0, math.Min(ry, h/2))
		if rx == 0 || ry == 0 {
			d = fmt.Sprintf("M%s,%sH%sV%sH%sZ", r(x), r(y), r(x+w), r(y+h), r(x))
		} else {
			a := func(px, py float64) string { return fmt.Sprintf("A%s,%s 0 0 1 %s,%s", r(rx), r(ry), r(px), r(py)) }
			d = fmt.Sprintf("M%s,%sH%s%sV%s%sH%s%sV%s%sZ", r(x+rx), r(y), r(x+w-rx), a(x+w, y+ry), r(y+h-ry),
				a(x+w-rx, y+h), r(x+rx), a(x, y+h-ry), r(y+ry), a(x+rx, y))
		}
	case "circle", "ellipse":
		cx, err := z.zahl(e, "cx", 0)
		if err != nil {
			return nil, err
		}
		cy, err := z.zahl(e, "cy", 0)
		if err != nil {
			return nil, err
		}
		var rx, ry float64
		if e.Tag == "circle" {
			if rx, err = z.zahl(e, "r", 0); err != nil {
				return nil, err
			}
			ry = rx
		} else {
			if rx, err = z.zahl(e, "rx", 0); err != nil {
				return nil, err
			}
			if ry, err = z.zahl(e, "ry", 0); err != nil {
				return nil, err
			}
		}
		if rx <= 0 || ry <= 0 {
			return nil, nil
		}
		d = fmt.Sprintf("M%s,%sA%s,%s 0 1 0 %s,%sA%s,%s 0 1 0 %s,%sZ", r(cx-rx), r(cy), r(rx), r(ry),
			r(cx+rx), r(cy), r(rx), r(ry), r(cx-rx), r(cy))
	case "line":
		var v [4]float64
		for i, k := range []string{"x1", "y1", "x2", "y2"} {
			f, err := z.zahl(e, k, 0)
			if err != nil {
				return nil, err
			}
			v[i] = f
		}
		d = fmt.Sprintf("M%s,%sL%s,%s", r(v[0]), r(v[1]), r(v[2]), r(v[3]))
	}
	tps, err := svg.TeilpfadeSchritt(d, schritt)
	if err != nil {
		return nil, z.fehler(e, "%v", err)
	}
	return tps, nil
}

func geraet(tps []svg.Teilpfad, m matrix) [][]geo.P {
	aus := make([][]geo.P, 0, len(tps))
	for _, tp := range tps {
		v := make([]geo.P, len(tp.Punkte))
		for i, p := range tp.Punkte {
			v[i] = m.punkt(p)
		}
		aus = append(aus, v)
	}
	return aus
}

// endlich: Gerätekoordinaten ohne NaN/Inf (sonst fiele die ganze Form still weg).
func endlich(polys [][]geo.P) error {
	for _, p := range polys {
		for _, q := range p {
			if math.IsNaN(q.X) || math.IsNaN(q.Y) || math.IsInf(q.X, 0) || math.IsInf(q.Y, 0) {
				return fmt.Errorf("Koordinate nicht endlich")
			}
		}
	}
	return nil
}

func polysGeraet(polys [][]geo.P, m matrix) [][]geo.P {
	aus := make([][]geo.P, len(polys))
	for i, p := range polys {
		v := make([]geo.P, len(p))
		for j, q := range p {
			v[j] = m.punkt(q)
		}
		aus[i] = v
	}
	return aus
}

func (z *zeichner) schritt(m matrix) float64 {
	s := m.massstab()
	if s <= 0 {
		return 1
	}
	return Feinheit / s
}

// form zeichnet Füllung und Strich einer Form auf die Ebene.
func (z *zeichner) form(e *svg.Element, m matrix, st stil, ziel *ebene) error {
	schritt := z.schritt(m)
	tps, err := z.teilpfade(e, schritt)
	if err != nil {
		return err
	}
	if len(tps) == 0 {
		return nil
	}
	if !st.fill.keine && e.Tag != "line" {
		g := geraet(tps, m)
		if err := endlich(g); err != nil {
			return z.fehler(e, "Füllung: %v", err)
		}
		ziel.malen(Fuellen(z.schneiden(g), st.evenodd, z.b, z.h), st.fill)
	}
	if !st.stroke.keine && st.strich.breite > 0 {
		polys, err := strichPolygone(tps, st.strich, schritt)
		if err != nil {
			return z.fehler(e, "%v", err)
		}
		g := polysGeraet(polys, m)
		if err := endlich(g); err != nil {
			return z.fehler(e, "Strich: %v", err)
		}
		ziel.malen(Fuellen(z.schneiden(g), false, z.b, z.h), st.stroke)
	}
	return nil
}

// clipKasten: Ist der clipPath genau ein achsparalleles Rechteck (ohne rx/ry), dessen Ecken in Gerätekoordinaten.
func (z *zeichner) clipKasten(e *svg.Element, id string, m matrix) ([4]float64, bool, error) {
	cp := z.ids[id]
	if cp == nil || cp.Tag != "clipPath" {
		return [4]float64{}, false, z.fehler(e, "clip-path #%s fehlt", id)
	}
	var kinder []*svg.Element
	for _, k := range cp.Kinder {
		if k.Tag != "title" && k.Tag != "desc" {
			kinder = append(kinder, k)
		}
	}
	if len(kinder) != 1 || kinder[0].Tag != "rect" || kinder[0].Has("rx") || kinder[0].Has("ry") ||
		(kinder[0].NS != "" && kinder[0].NS != svgNS) {
		return [4]float64{}, false, nil
	}
	if u := cp.Get("clipPathUnits"); u != "" && u != "userSpaceOnUse" {
		return [4]float64{}, false, z.fehler(cp, "clipPathUnits %s nicht unterstützt", u)
	}
	erbe, err := z.geerbt(cp)
	if err != nil {
		return [4]float64{}, false, err
	}
	cst, cei, err := z.ableiten(cp, erbe)
	if err != nil {
		return [4]float64{}, false, err
	}
	if cei.clip != "" || cei.maske != "" || cei.deckkraft != 1 {
		return [4]float64{}, false, z.fehler(cp, "clip-path, mask oder opacity am clipPath nicht unterstützt")
	}
	k := kinder[0]
	if _, kei, err := z.ableiten(k, cst); err != nil {
		return [4]float64{}, false, err
	} else if kei.clip != "" || kei.maske != "" {
		return [4]float64{}, false, z.fehler(k, "clip-path oder mask in einem clipPath nicht unterstützt")
	}
	km := m
	for _, t := range []string{cp.Get("transform"), k.Get("transform")} {
		if t == "" {
			continue
		}
		n, err := transform(t)
		if err != nil {
			return [4]float64{}, false, z.fehler(cp, "%v", err)
		}
		km = km.mal(n)
	}
	var v [4]float64
	for i, a := range []string{"x", "y", "width", "height"} {
		if v[i], err = z.zahl(k, a, 0); err != nil {
			return [4]float64{}, false, err
		}
	}
	if km[1] != 0 || km[2] != 0 { // gedreht oder geschert: kein Kasten
		return [4]float64{}, false, nil
	}
	if v[2] <= 0 || v[3] <= 0 {
		return [4]float64{0, 0, 0, 0}, true, nil
	}
	p := km.punkt(geo.P{X: v[0], Y: v[1]})
	q := km.punkt(geo.P{X: v[0] + v[2], Y: v[1] + v[3]})
	return [4]float64{fest(math.Min(p.X, q.X)), fest(math.Min(p.Y, q.Y)), fest(math.Max(p.X, q.X)), fest(math.Max(p.Y, q.Y))}, true, nil
}

// schneiden: Polygone auf den rechteckigen Clip beschneiden (Sutherland-Hodgman je Polygon; die Windungszahl im
// Inneren des Rechtecks bleibt dabei erhalten, nonzero und evenodd stimmen also weiter).
func (z *zeichner) schneiden(polys [][]geo.P) [][]geo.P {
	if z.box == nil {
		return polys
	}
	b := *z.box
	if b[0] >= b[2] || b[1] >= b[3] {
		return nil
	}
	seite := func(p []geo.P, innen func(geo.P) bool, schnitt func(a, c geo.P) geo.P) []geo.P {
		var aus []geo.P
		for i := range p {
			a, c := p[i], p[(i+1)%len(p)]
			ia, ic := innen(a), innen(c)
			if ia {
				aus = append(aus, a)
			}
			if ia != ic {
				aus = append(aus, schnitt(a, c))
			}
		}
		return aus
	}
	beiX := func(x float64) func(a, c geo.P) geo.P {
		return func(a, c geo.P) geo.P { return geo.P{X: x, Y: a.Y + (c.Y-a.Y)*(x-a.X)/(c.X-a.X)} }
	}
	beiY := func(y float64) func(a, c geo.P) geo.P {
		return func(a, c geo.P) geo.P { return geo.P{X: a.X + (c.X-a.X)*(y-a.Y)/(c.Y-a.Y), Y: y} }
	}
	var aus [][]geo.P
	for _, p := range polys {
		p = seite(p, func(q geo.P) bool { return q.X >= b[0] }, beiX(b[0]))
		p = seite(p, func(q geo.P) bool { return q.X <= b[2] }, beiX(b[2]))
		p = seite(p, func(q geo.P) bool { return q.Y >= b[1] }, beiY(b[1]))
		p = seite(p, func(q geo.P) bool { return q.Y <= b[3] }, beiY(b[3]))
		if len(p) >= 3 {
			aus = append(aus, p)
		}
	}
	return aus
}

// abdeckung: Deckung eines clipPath (Vereinigung seiner Formen) als Faktor je Pixel.
func (z *zeichner) abdeckung(e *svg.Element, id string, m matrix) ([]float32, error) {
	cp := z.ids[id]
	if cp == nil || cp.Tag != "clipPath" {
		return nil, z.fehler(e, "clip-path #%s fehlt", id)
	}
	if u := cp.Get("clipPathUnits"); u != "" && u != "userSpaceOnUse" {
		return nil, z.fehler(cp, "clipPathUnits %s nicht unterstützt", u)
	}
	erbe, err := z.geerbt(cp)
	if err != nil {
		return nil, err
	}
	cst, cei, err := z.ableiten(cp, erbe)
	if err != nil {
		return nil, err
	}
	if cei.clip != "" || cei.maske != "" || cei.deckkraft != 1 {
		return nil, z.fehler(cp, "clip-path, mask oder opacity am clipPath nicht unterstützt")
	}
	cm := m
	if t := cp.Get("transform"); t != "" {
		n, err := transform(t)
		if err != nil {
			return nil, z.fehler(cp, "%v", err)
		}
		cm = m.mal(n)
	}
	f := make([]float32, z.b*z.h)
	for _, k := range cp.Kinder {
		if k.Tag == "title" || k.Tag == "desc" {
			continue
		}
		if !formen[k.Tag] || (k.NS != "" && k.NS != svgNS) {
			return nil, z.fehler(k, "im clipPath nur Formen unterstützt")
		}
		kst, kei, err := z.ableiten(k, cst)
		if err != nil {
			return nil, err
		}
		if kei.clip != "" || kei.maske != "" {
			return nil, z.fehler(k, "clip-path oder mask in einem clipPath nicht unterstützt")
		}
		km := cm
		if t := k.Get("transform"); t != "" {
			n, err := transform(t)
			if err != nil {
				return nil, z.fehler(k, "%v", err)
			}
			km = cm.mal(n)
		}
		tps, err := z.teilpfade(k, z.schritt(km))
		if err != nil {
			return nil, err
		}
		mk := Fuellen(geraet(tps, km), kst.clipEvenodd, z.b, z.h)
		if mk == nil {
			continue
		}
		for y := 0; y < mk.H; y++ {
			for x := 0; x < mk.B; x++ {
				c := mk.A[y*mk.B+x]
				i := (mk.Y0+y)*z.b + mk.X0 + x
				f[i] = c + f[i]*(1-c)
			}
		}
	}
	return f, nil
}

// maske: Luminanz einer mask (userSpaceOnUse) mal ihrer Fläche als Faktor je Pixel.
func (z *zeichner) maske(e *svg.Element, id string, m matrix) ([]float32, error) {
	mk := z.ids[id]
	if mk == nil || mk.Tag != "mask" {
		return nil, z.fehler(e, "mask #%s fehlt", id)
	}
	if mk.Get("maskUnits") != "userSpaceOnUse" {
		return nil, z.fehler(mk, "nur maskUnits=userSpaceOnUse unterstützt")
	}
	erbe, err := z.geerbt(mk)
	if err != nil {
		return nil, err
	}
	mst, mei, err := z.ableiten(mk, erbe)
	if err != nil {
		return nil, err
	}
	if mei.clip != "" || mei.maske != "" || mei.deckkraft != 1 {
		return nil, z.fehler(mk, "clip-path, mask oder opacity an der mask nicht unterstützt")
	}
	var v [4]float64
	for i, k := range []string{"x", "y", "width", "height"} {
		if !mk.Has(k) {
			return nil, z.fehler(mk, "%s fehlt", k)
		}
		if v[i], err = z.zahl(mk, k, 0); err != nil {
			return nil, err
		}
	}
	ebene := neueEbene(z.b, z.h)
	for _, k := range mk.Kinder {
		if err := z.zeichnen(k, m, mst, ebene); err != nil {
			return nil, err
		}
	}
	bereich := []geo.P{{X: v[0], Y: v[1]}, {X: v[0] + v[2], Y: v[1]}, {X: v[0] + v[2], Y: v[1] + v[3]}, {X: v[0], Y: v[1] + v[3]}}
	fl := Fuellen(polysGeraet([][]geo.P{bereich}, m), false, z.b, z.h)
	f := make([]float32, z.b*z.h)
	if fl == nil {
		return f, nil
	}
	for y := 0; y < fl.H; y++ {
		for x := 0; x < fl.B; x++ {
			p := (fl.Y0+y)*z.b + fl.X0 + x
			i := p * 4
			lum := 0.2125*ebene.px[i] + 0.7154*ebene.px[i+1] + 0.0721*ebene.px[i+2]
			f[p] = lum * fl.A[y*fl.B+x]
		}
	}
	return f, nil
}

// zeichnen: ein Element mit Kindern auf die Ebene.
func (z *zeichner) zeichnen(e *svg.Element, m matrix, erbe stil, ziel *ebene) error {
	if still[e.Tag] && (e.NS == "" || e.NS == svgNS) {
		return nil
	}
	if e.NS != "" && e.NS != svgNS {
		return z.fehler(e, "Element aus fremdem Namensraum %s nicht unterstützt", e.NS)
	}
	if !gruppen[e.Tag] && !formen[e.Tag] {
		return z.fehler(e, "Element nicht unterstützt")
	}
	if e.Tag == "svg" && z.tiefe > 0 {
		return z.fehler(e, "verschachteltes svg nicht unterstützt")
	}
	z.tiefe++
	defer func() { z.tiefe-- }()
	if z.tiefe > 64 {
		return z.fehler(e, "zu tief verschachtelt")
	}
	st, ei, err := z.ableiten(e, erbe)
	if err != nil {
		return err
	}
	if t := e.Get("transform"); t != "" && e.Tag != "svg" {
		n, err := transform(t)
		if err != nil {
			return z.fehler(e, "%v", err)
		}
		m = m.mal(n)
	} else if t != "" {
		return z.fehler(e, "transform am svg nicht unterstützt")
	}
	inhalt := func(auf *ebene) error {
		if formen[e.Tag] {
			return z.form(e, m, st, auf)
		}
		for _, k := range e.Kinder {
			if err := z.zeichnen(k, m, st, auf); err != nil {
				return err
			}
		}
		return nil
	}
	if ei.clip != "" {
		// Rechteckiger Clip: geometrisch schneiden statt Deckungen zu multiplizieren (wie Cairo); sonst wird eine
		// Kante, auf der Clip und Form zugleich enden, doppelt geglättet (0,25 x 0,25 statt 0,25).
		box, ok, err := z.clipKasten(e, ei.clip, m)
		if err != nil {
			return err
		}
		if ok {
			alt := z.box
			if alt != nil {
				box = [4]float64{math.Max(box[0], alt[0]), math.Max(box[1], alt[1]), math.Min(box[2], alt[2]), math.Min(box[3], alt[3])}
			}
			z.box = &box
			defer func() { z.box = alt }()
			ei.clip = ""
		}
	}
	if ei.clip == "" && ei.maske == "" && ei.deckkraft == 1 {
		return inhalt(ziel)
	}
	ebene := neueEbene(z.b, z.h)
	if err := inhalt(ebene); err != nil {
		return err
	}
	var faktor []float32
	if ei.clip != "" {
		if faktor, err = z.abdeckung(e, ei.clip, m); err != nil {
			return err
		}
	}
	if ei.maske != "" {
		f, err := z.maske(e, ei.maske, m)
		if err != nil {
			return err
		}
		if faktor == nil {
			faktor = f
		} else {
			for i := range faktor {
				faktor[i] *= f[i]
			}
		}
	}
	ziel.auftragen(ebene, faktor, float32(ei.deckkraft))
	return nil
}

// Groesse: Bildgröße für eine Breite (wie rsvg-convert -w: Höhe aufgerundet).
func Groesse(wurzel *svg.Element, breite int) (int, int, matrix, error) {
	var vb []float64
	if v := wurzel.Get("viewBox"); v != "" {
		a, err := zahlen(v)
		if err != nil || len(a) != 4 || a[2] <= 0 || a[3] <= 0 {
			return 0, 0, einheit, fmt.Errorf("viewBox %q nicht lesbar", v)
		}
		vb = a
	} else {
		w, err1 := laenge(wurzel.Get("width"))
		h, err2 := laenge(wurzel.Get("height"))
		if err1 != nil || err2 != nil || w <= 0 || h <= 0 {
			return 0, 0, einheit, fmt.Errorf("weder viewBox noch width/height")
		}
		vb = []float64{0, 0, w, h}
	}
	if wurzel.Has("width") && wurzel.Has("height") && wurzel.Get("viewBox") != "" {
		w, err1 := laenge(wurzel.Get("width"))
		h, err2 := laenge(wurzel.Get("height"))
		if err1 != nil || err2 != nil {
			return 0, 0, einheit, fmt.Errorf("width/height nicht lesbar")
		}
		if math.Abs(w/h-vb[2]/vb[3]) > 1e-9 {
			return 0, 0, einheit, fmt.Errorf("width/height und viewBox mit verschiedenem Seitenverhältnis nicht unterstützt")
		}
	}
	s := float64(breite) / vb[2]
	// Höhe aufgerundet; der Inhalt beginnt oben, die angebrochene letzte Zeile bleibt teilweise leer (wie rsvg-convert)
	hoehe := int(math.Ceil(vb[3]*s - 1e-9))
	m := matrix{s, 0, 0, s, 0, 0}.mal(matrix{1, 0, 0, 1, -vb[0], -vb[1]})
	return breite, hoehe, m, nil
}

// Zeichnen rastert ein SVG auf die Breite breite.
func Zeichnen(wurzel *svg.Element, breite int) (*image.NRGBA, error) {
	if wurzel.Tag != "svg" {
		return nil, fmt.Errorf("Wurzel ist <%s>, nicht <svg>", wurzel.Tag)
	}
	if breite <= 0 || breite > 20000 {
		return nil, fmt.Errorf("Breite %d außerhalb 1..20000", breite)
	}
	b, h, m, err := Groesse(wurzel, breite)
	if err != nil {
		return nil, err
	}
	stilText := svg.Stil(wurzel)
	for _, sel := range regexp.MustCompile(`([^{}]+)\{`).FindAllStringSubmatch(stilText, -1) {
		for _, k := range strings.Split(sel[1], ",") {
			k = strings.TrimSpace(k)
			if !regexp.MustCompile(`^\.[\w-]+$`).MatchString(k) {
				return nil, fmt.Errorf("<style>: nur einfache Klassenregeln unterstützt, nicht %q", k)
			}
		}
	}
	z := &zeichner{b: b, h: h, ids: map[string]*svg.Element{}, eltern: map[*svg.Element]*svg.Element{},
		regeln: svg.KlassenRegeln(stilText)}
	for _, e := range wurzel.Alle() {
		for _, k := range e.Kinder {
			z.eltern[k] = e
		}
	}
	for kl, r := range z.regeln {
		for _, k := range r.Schl {
			if !eigenschaften[k] {
				return nil, fmt.Errorf("<style> .%s: Eigenschaft %s nicht unterstützt", kl, k)
			}
		}
	}
	for _, e := range wurzel.Alle() {
		if id := e.Get("id"); id != "" {
			if z.ids[id] != nil {
				return nil, fmt.Errorf("id %q doppelt", id)
			}
			z.ids[id] = e
		}
	}
	ebene := neueEbene(b, h)
	if err := z.zeichnen(wurzel, m, startStil, ebene); err != nil {
		return nil, err
	}
	bild := image.NewNRGBA(image.Rect(0, 0, b, h))
	for p := 0; p < b*h; p++ {
		i := p * 4
		a := ebene.px[i+3]
		if a <= 0 {
			continue
		}
		bild.Pix[i+3] = kanal(a)
		bild.Pix[i] = kanal(ebene.px[i] / a)
		bild.Pix[i+1] = kanal(ebene.px[i+1] / a)
		bild.Pix[i+2] = kanal(ebene.px[i+2] / a)
	}
	return bild, nil
}

func kanal(v float32) uint8 {
	x := math.Round(float64(v) * 255)
	if x < 0 {
		return 0
	}
	if x > 255 {
		return 255
	}
	return uint8(x)
}
