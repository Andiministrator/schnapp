// SPDX-License-Identifier: GPL-3.0-or-later

// Paket svg liest SVG-Dateien und Pfaddaten. Pfade werden in dichte Punktfolgen zerlegt (Kurven und Bögen so fein,
// dass zwei Punkte
// höchstens Schritt Einheiten auseinanderliegen), damit sie sich Punkt für Punkt biegen lassen.
package svg

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/Andiministrator/schnapp/draw/geo"
)

// Schritt: größter Abstand zweier Punkte einer zerlegten Kurve (in viewBox-Einheiten).
const Schritt = 1.0

// ZahlMuster: vollständige SVG-Zahl.
const ZahlMuster = `[-+]?(?:\d+\.?\d*|\.\d+)(?:[eE][-+]?\d+)?`

var zahlAmAnfang = regexp.MustCompile(`^` + ZahlMuster)

// Zeichen: die Elemente, die gezählt und gezeichnet werden.
var Zeichen = map[string]bool{"path": true, "polygon": true, "polyline": true, "rect": true, "circle": true,
	"ellipse": true, "line": true}

// --- XML -----------------------------------------------------------------------------------------------------

// Element ist ein XML-Element mit Attributen in Dateireihenfolge.
type Element struct {
	Tag    string
	NS     string // Namensraum des Elements ("" ohne xmlns)
	Attr   []xml.Attr
	Kinder []*Element
	Text   string
}

// Get liefert den Wert eines Attributs ohne Namensraum ("" wenn es fehlt).
func (e *Element) Get(k string) string {
	for _, a := range e.Attr {
		if a.Name.Space == "" && a.Name.Local == k {
			return a.Value
		}
	}
	return ""
}

// Has sagt, ob das Attribut da ist.
func (e *Element) Has(k string) bool {
	for _, a := range e.Attr {
		if a.Name.Space == "" && a.Name.Local == k {
			return true
		}
	}
	return false
}

// Alle: das Element und alle Nachfahren in Dateireihenfolge (wie iter() in ElementTree).
func (e *Element) Alle() []*Element {
	aus := []*Element{e}
	for _, k := range e.Kinder {
		aus = append(aus, k.Alle()...)
	}
	return aus
}

// normalisieren: Zeilenumbrüche und Tabulatoren in Attributwerten werden Leerzeichen (wie expat).
func normalisieren(v string) string {
	v = strings.ReplaceAll(v, "\r\n", " ")
	return strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(v)
}

// Parse liest ein SVG; DOCTYPE und ENTITY werden abgelehnt.
func Parse(daten []byte, wo string) (*Element, error) {
	if bytes.Contains(daten, []byte("<!DOCTYPE")) || bytes.Contains(daten, []byte("<!ENTITY")) {
		return nil, fmt.Errorf("%s: DOCTYPE/ENTITY im SVG, abgelehnt", wo)
	}
	d := xml.NewDecoder(bytes.NewReader(daten))
	var wurzel *Element
	var stapel []*Element
	for {
		t, err := d.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("%s: %v", wo, err)
		}
		switch x := t.(type) {
		case xml.StartElement:
			e := &Element{Tag: x.Name.Local, NS: x.Name.Space}
			for _, a := range x.Attr {
				a.Value = normalisieren(a.Value)
				e.Attr = append(e.Attr, a)
			}
			if len(stapel) > 0 {
				p := stapel[len(stapel)-1]
				p.Kinder = append(p.Kinder, e)
			} else if wurzel == nil {
				wurzel = e
			}
			stapel = append(stapel, e)
		case xml.EndElement:
			stapel = stapel[:len(stapel)-1]
		case xml.CharData:
			if len(stapel) > 0 {
				stapel[len(stapel)-1].Text += string(x)
			}
		}
	}
	if wurzel == nil {
		return nil, fmt.Errorf("%s: kein Element", wo)
	}
	return wurzel, nil
}

// Laden liest eine SVG-Datei.
func Laden(pfad string) (*Element, error) {
	daten, err := os.ReadFile(pfad)
	if err != nil {
		return nil, err
	}
	return Parse(daten, pfad)
}

// --- Klassen aus <style> -------------------------------------------------------------------------------------

// Regel: Eigenschaften einer Klasse in Reihenfolge.
type Regel struct {
	Schl  []string
	Werte map[string]string
}

func (r *Regel) setzen(k, v string) {
	if _, ok := r.Werte[k]; !ok {
		r.Schl = append(r.Schl, k)
	}
	r.Werte[k] = v
}

var regelMuster = regexp.MustCompile(`([^{}]+)\{([^}]*)\}`)

// KlassenRegeln: '.cls-2,.cls-5{fill:none;stroke-miterlimit:10;}' -> {"cls-2": {...}, ...}.
func KlassenRegeln(stil string) map[string]*Regel {
	regeln := map[string]*Regel{}
	for _, m := range regelMuster.FindAllStringSubmatch(stil, -1) {
		var schl, werte []string
		for _, p := range strings.Split(m[2], ";") {
			i := strings.Index(p, ":")
			if i < 0 {
				continue
			}
			schl = append(schl, strings.TrimSpace(p[:i]))
			werte = append(werte, strings.TrimSpace(p[i+1:]))
		}
		for _, k := range strings.Split(m[1], ",") {
			k = strings.TrimSpace(k)
			if !strings.HasPrefix(k, ".") {
				continue
			}
			r := regeln[k[1:]]
			if r == nil {
				r = &Regel{Werte: map[string]string{}}
				regeln[k[1:]] = r
			}
			for i := range schl {
				r.setzen(schl[i], werte[i])
			}
		}
	}
	return regeln
}

// Stil: Text aller <style>-Elemente.
func Stil(wurzel *Element) string {
	var b strings.Builder
	for _, e := range wurzel.Alle() {
		if e.Tag == "style" {
			b.WriteString(e.Text)
		}
	}
	return b.String()
}

// ZeichenElemente: alle Zeichenelemente in Dateireihenfolge (0 = Hintergrund).
func ZeichenElemente(wurzel *Element) []*Element {
	var aus []*Element
	for _, e := range wurzel.Alle() {
		if Zeichen[e.Tag] {
			aus = append(aus, e)
		}
	}
	return aus
}

// erlaubt: Attribute, die ElementText übernimmt.
var erlaubt = map[string]bool{"d": true, "points": true, "x": true, "y": true, "width": true, "height": true,
	"cx": true, "cy": true, "r": true, "rx": true, "ry": true, "x1": true, "y1": true, "x2": true, "y2": true,
	"transform": true, "fill": true, "stroke": true, "stroke-width": true, "stroke-miterlimit": true,
	"stroke-linecap": true, "stroke-linejoin": true, "fill-rule": true, "clip-rule": true, "opacity": true}

// ElementText schreibt ein Zeichenelement mit Klassenfarben als Attributen (ohne Farbtausch).
func ElementText(e *Element, regeln map[string]*Regel) (string, error) {
	schl := []string{}
	werte := map[string]string{}
	setzen := func(k, v string) {
		if _, ok := werte[k]; !ok {
			schl = append(schl, k)
		}
		werte[k] = v
	}
	for _, k := range strings.Fields(e.Get("class")) {
		if r := regeln[k]; r != nil {
			for _, s := range r.Schl {
				setzen(s, r.Werte[s])
			}
		}
	}
	for _, a := range e.Attr {
		if a.Name.Space == "" && erlaubt[a.Name.Local] {
			setzen(a.Name.Local, a.Value)
		}
	}
	var teile []string
	for _, k := range schl {
		if !erlaubt[k] {
			continue
		}
		v := werte[k]
		if k == "stroke-width" {
			v = strings.ReplaceAll(v, "px", "")
		}
		if strings.Contains(v, "{{") || strings.Contains(v, "}}") {
			return "", fmt.Errorf("Vorlagenzeichen in Attributwert: %.40q", v)
		}
		teile = append(teile, k+`="`+geo.Escape(v)+`"`)
	}
	return "<" + e.Tag + " " + strings.Join(teile, " ") + "/>", nil
}

// --- Pfade zerlegen ------------------------------------------------------------------------------------------

// Teilpfad: Punktfolge; Geschlossen, wenn er im Original mit Z endet.
type Teilpfad struct {
	Punkte      []geo.P
	Geschlossen bool
}

type token struct {
	befehl byte
	zahl   float64
}

func zerlegeToken(d string) ([]token, error) {
	var aus []token
	var befehl byte
	zaehler := 0
	for i := 0; i < len(d); {
		c := d[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			befehl = c
			zaehler = 0
			aus = append(aus, token{befehl: c})
			i++
			continue
		}
		if strings.IndexByte(" ,\t\n\r", c) >= 0 {
			i++
			continue
		}
		if (befehl == 'A' || befehl == 'a') && (zaehler%7 == 3 || zaehler%7 == 4) && (c == '0' || c == '1') {
			aus = append(aus, token{zahl: float64(c - '0')})
			zaehler++
			i++
			continue
		}
		m := zahlAmAnfang.FindString(d[i:])
		if m == "" {
			ende := i + 20
			if ende > len(d) {
				ende = len(d)
			}
			return nil, fmt.Errorf("Pfad nicht lesbar bei %q", d[i:ende])
		}
		f, err := strconv.ParseFloat(m, 64)
		if err != nil {
			return nil, err
		}
		aus = append(aus, token{zahl: f})
		zaehler++
		i += len(m)
	}
	return aus, nil
}

func kubisch(p0, p1, p2, p3 geo.P, schritt float64) []geo.P {
	lang := geo.Dist(p0, p1) + geo.Dist(p1, p2) + geo.Dist(p2, p3)
	n := max(1, geo.Ceil(lang/schritt))
	pts := make([]geo.P, 0, n)
	for k := 1; k <= n; k++ {
		t := float64(k) / float64(n)
		u := 1 - t
		pts = append(pts, geo.P{
			X: geo.Pow(u, 3)*p0.X + 3*u*u*t*p1.X + 3*u*t*t*p2.X + geo.Pow(t, 3)*p3.X,
			Y: geo.Pow(u, 3)*p0.Y + 3*u*u*t*p1.Y + 3*u*t*t*p2.Y + geo.Pow(t, 3)*p3.Y,
		})
	}
	return pts
}

func gerade(p0, p1 geo.P, schritt float64) []geo.P {
	n := max(1, geo.Ceil(geo.Dist(p0, p1)/schritt))
	pts := make([]geo.P, 0, n)
	for k := 1; k <= n; k++ {
		pts = append(pts, geo.P{X: p0.X + (p1.X-p0.X)*float64(k)/float64(n), Y: p0.Y + (p1.Y-p0.Y)*float64(k)/float64(n)})
	}
	return pts
}

// bogen: elliptischer Bogen nach SVG-Spezifikation (Anhang F.6) als Punktfolge.
func bogen(p0 geo.P, rx, ry, phi float64, gross, dreh bool, p1 geo.P, schritt float64) []geo.P {
	if geo.Dist(p0, p1) < 1e-9 {
		return nil
	}
	if rx == 0 || ry == 0 {
		return gerade(p0, p1, schritt)
	}
	rx, ry = math.Abs(rx), math.Abs(ry)
	c, s := math.Cos(geo.Radians(phi)), math.Sin(geo.Radians(phi))
	dx, dy := (p0.X-p1.X)/2, (p0.Y-p1.Y)/2
	x1, y1 := c*dx+s*dy, -s*dx+c*dy
	lam := x1*x1/(rx*rx) + y1*y1/(ry*ry)
	if lam > 1 {
		rx, ry = rx*math.Sqrt(lam), ry*math.Sqrt(lam)
	}
	zaehler := rx*rx*ry*ry - rx*rx*y1*y1 - ry*ry*x1*x1
	nenner := rx*rx*y1*y1 + ry*ry*x1*x1
	f := math.Sqrt(math.Max(0.0, zaehler/nenner))
	if gross == dreh {
		f = -f
	}
	cx1, cy1 := f*rx*y1/ry, -f*ry*x1/rx
	cx := c*cx1 - s*cy1 + (p0.X+p1.X)/2
	cy := s*cx1 + c*cy1 + (p0.Y+p1.Y)/2
	winkel := func(ux, uy, vx, vy float64) float64 { return math.Atan2(ux*vy-uy*vx, ux*vx+uy*vy) }
	t1 := winkel(1, 0, (x1-cx1)/rx, (y1-cy1)/ry)
	dt := winkel((x1-cx1)/rx, (y1-cy1)/ry, (-x1-cx1)/rx, (-y1-cy1)/ry)
	if !dreh && dt > 0 {
		dt -= 2 * math.Pi
	} else if dreh && dt < 0 {
		dt += 2 * math.Pi
	}
	n := max(2, geo.Ceil(math.Abs(dt)*math.Max(rx, ry)/schritt))
	pts := make([]geo.P, 0, n)
	for k := 1; k <= n; k++ {
		t := t1 + dt*float64(k)/float64(n)
		ex, ey := rx*math.Cos(t), ry*math.Sin(t)
		pts = append(pts, geo.P{X: c*ex - s*ey + cx, Y: s*ex + c*ey + cy})
	}
	pts[len(pts)-1] = p1
	return pts
}

var anzahl = map[byte]int{'M': 2, 'L': 2, 'H': 1, 'V': 1, 'C': 6, 'S': 4, 'Q': 4, 'T': 2, 'A': 7, 'Z': 0}

// Teilpfade zerlegt Pfaddaten in Teilpfade (Geschlossen bei Z am Ende); Teilpfade mit nur einem Punkt fallen weg.
func Teilpfade(d string) ([]Teilpfad, error) { return TeilpfadeSchritt(d, Schritt) }

// TeilpfadeSchritt wie Teilpfade, mit eigener Schrittweite (der Rasterer zerlegt feiner).
func TeilpfadeSchritt(d string, schritt float64) ([]Teilpfad, error) {
	tok, err := zerlegeToken(d)
	if err != nil {
		return nil, err
	}
	var aus []Teilpfad
	var akt []geo.P
	pos, start := geo.P{}, geo.P{}
	var letzteKontrolle *geo.P
	var letzter byte
	var befehl byte
	for i := 0; i < len(tok); {
		if tok[i].befehl != 0 {
			befehl = tok[i].befehl
			i++
			if befehl == 'Z' || befehl == 'z' {
				if len(akt) > 0 {
					aus = append(aus, Teilpfad{akt, true})
				}
				akt = nil
				pos = start
				letzter = 'Z'
				letzteKontrolle = nil
				continue
			}
		}
		if befehl == 0 {
			return nil, fmt.Errorf("Pfad beginnt ohne Befehl")
		}
		gross := befehl &^ 0x20
		rel := befehl >= 'a'
		n, ok := anzahl[gross]
		if !ok || gross == 'Z' {
			return nil, fmt.Errorf("Befehl %c nicht unterstützt", befehl)
		}
		if i+n > len(tok) {
			return nil, fmt.Errorf("zu wenige Zahlen für %c", befehl)
		}
		a := make([]float64, n)
		for j := 0; j < n; j++ {
			if tok[i+j].befehl != 0 {
				return nil, fmt.Errorf("zu wenige Zahlen für %c", befehl)
			}
			a[j] = tok[i+j].zahl
		}
		i += n
		ox, oy := 0.0, 0.0
		if rel {
			ox, oy = pos.X, pos.Y
		}
		if gross == 'M' {
			if len(akt) > 0 {
				aus = append(aus, Teilpfad{akt, false})
			}
			pos = geo.P{X: ox + a[0], Y: oy + a[1]}
			start = pos
			akt = []geo.P{pos}
			if rel {
				befehl = 'l'
			} else {
				befehl = 'L'
			}
			letzteKontrolle = nil
			letzter = 'M'
			continue
		}
		if akt == nil {
			akt = []geo.P{pos}
		}
		var ziel geo.P
		switch gross {
		case 'L':
			ziel = geo.P{X: ox + a[0], Y: oy + a[1]}
			akt = append(akt, gerade(pos, ziel, schritt)...)
			letzteKontrolle = nil
		case 'H':
			bx := 0.0
			if rel {
				bx = pos.X
			}
			ziel = geo.P{X: bx + a[0], Y: pos.Y}
			akt = append(akt, gerade(pos, ziel, schritt)...)
			letzteKontrolle = nil
		case 'V':
			by := 0.0
			if rel {
				by = pos.Y
			}
			ziel = geo.P{X: pos.X, Y: by + a[0]}
			akt = append(akt, gerade(pos, ziel, schritt)...)
			letzteKontrolle = nil
		case 'C', 'S':
			var k1, k2 geo.P
			if gross == 'C' {
				k1 = geo.P{X: ox + a[0], Y: oy + a[1]}
				k2, ziel = geo.P{X: ox + a[2], Y: oy + a[3]}, geo.P{X: ox + a[4], Y: oy + a[5]}
			} else {
				if (letzter == 'C' || letzter == 'S') && letzteKontrolle != nil {
					k1 = geo.P{X: 2*pos.X - letzteKontrolle.X, Y: 2*pos.Y - letzteKontrolle.Y}
				} else {
					k1 = pos
				}
				k2, ziel = geo.P{X: ox + a[0], Y: oy + a[1]}, geo.P{X: ox + a[2], Y: oy + a[3]}
			}
			akt = append(akt, kubisch(pos, k1, k2, ziel, schritt)...)
			kk := k2
			letzteKontrolle = &kk
		case 'Q', 'T':
			var q geo.P
			if gross == 'Q' {
				q, ziel = geo.P{X: ox + a[0], Y: oy + a[1]}, geo.P{X: ox + a[2], Y: oy + a[3]}
			} else {
				if (letzter == 'Q' || letzter == 'T') && letzteKontrolle != nil {
					q = geo.P{X: 2*pos.X - letzteKontrolle.X, Y: 2*pos.Y - letzteKontrolle.Y}
				} else {
					q = pos
				}
				ziel = geo.P{X: ox + a[0], Y: oy + a[1]}
			}
			k1 := geo.P{X: pos.X + 2.0/3*(q.X-pos.X), Y: pos.Y + 2.0/3*(q.Y-pos.Y)}
			k2 := geo.P{X: ziel.X + 2.0/3*(q.X-ziel.X), Y: ziel.Y + 2.0/3*(q.Y-ziel.Y)}
			akt = append(akt, kubisch(pos, k1, k2, ziel, schritt)...)
			qq := q
			letzteKontrolle = &qq
		case 'A':
			ziel = geo.P{X: ox + a[5], Y: oy + a[6]}
			akt = append(akt, bogen(pos, a[0], a[1], a[2], a[3] != 0, a[4] != 0, ziel, schritt)...)
			letzteKontrolle = nil
		}
		pos = ziel
		letzter = gross
	}
	if len(akt) > 0 {
		aus = append(aus, Teilpfad{akt, false})
	}
	gefiltert := aus[:0]
	for _, tp := range aus {
		if len(tp.Punkte) > 1 {
			gefiltert = append(gefiltert, tp)
		}
	}
	return gefiltert, nil
}

var zahlMuster = regexp.MustCompile(ZahlMuster)
var translateMuster = regexp.MustCompile(`^\s*translate\(\s*(` + ZahlMuster + `)(?:[\s,]+(` + ZahlMuster + `))?\s*\)\s*$`)

func pf(s string) (float64, error) { return strconv.ParseFloat(strings.TrimSpace(s), 64) }

func r(f float64) string { return geo.Repr(f) }

// ElementTeilpfade zerlegt ein Zeichenelement (path, polygon, polyline, rect, circle, ellipse, line) in Teilpfade;
// ein translate am Element wird angewendet, andere Transformationen brechen ab.
func ElementTeilpfade(e *Element) ([]Teilpfad, error) {
	var tps []Teilpfad
	var err error
	switch e.Tag {
	case "path":
		tps, err = Teilpfade(e.Get("d"))
	case "polygon", "polyline":
		var z []string
		for _, s := range zahlMuster.FindAllString(e.Get("points"), -1) {
			f, err := pf(s)
			if err != nil {
				return nil, err
			}
			z = append(z, r(f))
		}
		d := "M" + strings.Join(z, " ")
		if e.Tag == "polygon" {
			d += "Z"
		}
		tps, err = Teilpfade(d)
		if err == nil && len(tps) > 0 {
			tps = tps[:1]
		}
	case "rect":
		var x, y, w, h float64
		if e.Has("x") {
			if x, err = pf(e.Get("x")); err != nil {
				return nil, err
			}
		}
		if e.Has("y") {
			if y, err = pf(e.Get("y")); err != nil {
				return nil, err
			}
		}
		if w, err = pf(e.Get("width")); err != nil {
			return nil, err
		}
		if h, err = pf(e.Get("height")); err != nil {
			return nil, err
		}
		tps, err = Teilpfade(fmt.Sprintf("M%s,%sh%sv%sh%sZ", r(x), r(y), r(w), r(h), r(-w)))
	case "circle", "ellipse":
		var cx, cy, rx, ry float64
		if e.Has("cx") {
			if cx, err = pf(e.Get("cx")); err != nil {
				return nil, err
			}
		}
		if e.Has("cy") {
			if cy, err = pf(e.Get("cy")); err != nil {
				return nil, err
			}
		}
		rs := e.Get("r")
		if rs == "" {
			rs = e.Get("rx")
		}
		if rx, err = pf(rs); err != nil {
			return nil, err
		}
		rs = e.Get("r")
		if rs == "" {
			rs = e.Get("ry")
		}
		if ry, err = pf(rs); err != nil {
			return nil, err
		}
		tps, err = Teilpfade(fmt.Sprintf("M%s,%sA%s,%s 0 1 0 %s,%sA%s,%s 0 1 0 %s,%sZ", r(cx-rx), r(cy), r(rx), r(ry),
			r(cx+rx), r(cy), r(rx), r(ry), r(cx-rx), r(cy)))
	case "line":
		var v [4]float64
		for i, k := range []string{"x1", "y1", "x2", "y2"} {
			if v[i], err = pf(e.Get(k)); err != nil {
				return nil, err
			}
		}
		tps = []Teilpfad{{[]geo.P{{X: v[0], Y: v[1]}, {X: v[2], Y: v[3]}}, false}}
	default:
		return nil, fmt.Errorf("Element %s nicht unterstützt", e.Tag)
	}
	if err != nil {
		return nil, err
	}
	dx, dy := 0.0, 0.0
	if t := e.Get("transform"); t != "" {
		m := translateMuster.FindStringSubmatch(t)
		if m == nil {
			return nil, fmt.Errorf("Transform nicht unterstützt: %s", t)
		}
		dx, _ = pf(m[1])
		if m[2] != "" {
			dy, _ = pf(m[2])
		}
	}
	if dx != 0 || dy != 0 {
		for i := range tps {
			v := make([]geo.P, len(tps[i].Punkte))
			for j, p := range tps[i].Punkte {
				v[j] = geo.P{X: p.X + dx, Y: p.Y + dy}
			}
			tps[i].Punkte = v
		}
	}
	return tps, nil
}

// Quelle: ein Zeichenelement eines Originals mit seinen Farben (wie pfad.lade).
type Quelle struct {
	Element     *Element
	Fill        string
	Stroke      string
	StrokeWidth float64
}

// LadeOriginal liest ein Illustrator-SVG: viewBox und alle Zeichenelemente mit Farben aus Klassen und Attributen.
func LadeOriginal(pfad string) (string, []Quelle, error) {
	wurzel, err := Laden(pfad)
	if err != nil {
		return "", nil, err
	}
	regeln := KlassenRegeln(Stil(wurzel))
	var aus []Quelle
	for _, e := range ZeichenElemente(wurzel) {
		attr := map[string]string{}
		for _, k := range strings.Fields(e.Get("class")) {
			if r := regeln[k]; r != nil {
				for _, s := range r.Schl {
					attr[s] = r.Werte[s]
				}
			}
		}
		for _, k := range []string{"fill", "stroke", "stroke-width"} {
			if v := e.Get(k); v != "" {
				attr[k] = v
			}
		}
		q := Quelle{Element: e, Fill: "#000", Stroke: "none", StrokeWidth: 1}
		if v, ok := attr["fill"]; ok {
			q.Fill = v
		}
		if v, ok := attr["stroke"]; ok {
			q.Stroke = v
		}
		if v, ok := attr["stroke-width"]; ok {
			f, err := pf(strings.ReplaceAll(v, "px", ""))
			if err != nil {
				return "", nil, fmt.Errorf("%s: stroke-width %q", pfad, v)
			}
			q.StrokeWidth = f
		}
		aus = append(aus, q)
	}
	return wurzel.Get("viewBox"), aus, nil
}
