// SPDX-License-Identifier: GPL-3.0-or-later

package main

// Schrift als Pfad: liest die TrueType-Umrisse einer Schriftdatei selbst (head, hhea, hmtx, cmap, loca, glyf) und
// schreibt sie als SVG-Pfad, Zahl für Zahl wie bisher fontTools (SVGPathPen hinter TransformPen, Glyph.draw), damit
// das Logo-Paket byte-gleich bleibt. Nur einfache Glyphen (keine zusammengesetzten), nur quadratische Kurven; mehr
// braucht die Unterzeile aus Open Sans nicht. Was nicht passt, bricht mit Fehler ab, statt still anders zu zeichnen.

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

// pyZahl schreibt eine Zahl wie Pythons str(float): kürzeste Darstellung, ganze Zahlen mit „.0“, Exponent ab 1e16
// und unter 1e-4.
func pyZahl(f float64) string {
	if f == 0 {
		if math.Signbit(f) {
			return "-0.0"
		}
		return "0.0"
	}
	a := math.Abs(f)
	if a >= 1e16 || a < 1e-4 {
		s := strconv.FormatFloat(f, 'e', -1, 64) // 1e-05 statt 1e-5 wie Python
		m, e, _ := strings.Cut(s, "e")
		vz := e[:1]
		ziffern := strings.TrimLeft(e[1:], "0")
		if len(ziffern) < 2 {
			ziffern = strings.Repeat("0", 2-len(ziffern)) + ziffern
		}
		return m + "e" + vz + ziffern
	}
	s := strconv.FormatFloat(f, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}

type punkt struct{ x, y float64 }

type glyphe struct {
	vorschub int
	konturen [][]roh
}

type roh struct {
	x, y     int
	aufKurve bool
}

// schrift ist eine geladene TrueType-Datei.
type schrift struct {
	einheiten int
	zeichen   map[rune]int
	glyphen   map[int]*glyphe
	daten     []byte
	tabellen  map[string][]byte
	vorschub  []int
	kurzLoca  bool
	anzahl    int
}

func ladeSchrift(pfad string) (*schrift, error) {
	d, err := os.ReadFile(pfad)
	if err != nil {
		return nil, err
	}
	if len(d) < 12 {
		return nil, fmt.Errorf("%s: zu kurz", pfad)
	}
	s := &schrift{daten: d, tabellen: map[string][]byte{}, glyphen: map[int]*glyphe{}}
	n := int(binary.BigEndian.Uint16(d[4:]))
	for i := 0; i < n; i++ {
		e := d[12+16*i:]
		tag := string(e[:4])
		off, lae := binary.BigEndian.Uint32(e[8:]), binary.BigEndian.Uint32(e[12:])
		if uint64(off)+uint64(lae) > uint64(len(d)) {
			return nil, fmt.Errorf("%s: Tabelle %q außerhalb der Datei", pfad, tag)
		}
		s.tabellen[tag] = d[off : off+lae]
	}
	for _, t := range []string{"head", "hhea", "hmtx", "maxp", "cmap", "loca", "glyf"} {
		if s.tabellen[t] == nil {
			return nil, fmt.Errorf("%s: Tabelle %q fehlt", pfad, t)
		}
	}
	head := s.tabellen["head"]
	s.einheiten = int(binary.BigEndian.Uint16(head[18:]))
	s.kurzLoca = binary.BigEndian.Uint16(head[50:]) == 0
	s.anzahl = int(binary.BigEndian.Uint16(s.tabellen["maxp"][4:]))
	metriken := int(binary.BigEndian.Uint16(s.tabellen["hhea"][34:]))
	hmtx := s.tabellen["hmtx"]
	s.vorschub = make([]int, s.anzahl)
	for i := 0; i < s.anzahl; i++ {
		j := min(i, metriken-1)
		s.vorschub[i] = int(binary.BigEndian.Uint16(hmtx[4*j:]))
	}
	if err := s.leseCmap(); err != nil {
		return nil, fmt.Errorf("%s: %v", pfad, err)
	}
	return s, nil
}

// leseCmap nimmt die Unicode-Tabelle (3,10 Format 12 oder 3,1 Format 4), wie fontTools getBestCmap.
func (s *schrift) leseCmap() error {
	c := s.tabellen["cmap"]
	n := int(binary.BigEndian.Uint16(c[2:]))
	var f4, f12 []byte
	for i := 0; i < n; i++ {
		e := c[4+8*i:]
		pf, enc, off := binary.BigEndian.Uint16(e), binary.BigEndian.Uint16(e[2:]), binary.BigEndian.Uint32(e[4:])
		sub := c[off:]
		format := binary.BigEndian.Uint16(sub)
		switch {
		case pf == 3 && enc == 10 && format == 12:
			f12 = sub
		case pf == 3 && enc == 1 && format == 4:
			f4 = sub
		}
	}
	s.zeichen = map[rune]int{}
	switch {
	case f12 != nil:
		gruppen := int(binary.BigEndian.Uint32(f12[12:]))
		for i := 0; i < gruppen; i++ {
			g := f12[16+12*i:]
			von, bis, start := binary.BigEndian.Uint32(g), binary.BigEndian.Uint32(g[4:]), binary.BigEndian.Uint32(g[8:])
			for r := von; r <= bis; r++ {
				s.zeichen[rune(r)] = int(start + r - von)
			}
		}
	case f4 != nil:
		seg := int(binary.BigEndian.Uint16(f4[6:])) / 2
		ende := f4[14:]
		anfang := ende[2*seg+2:]
		delta := anfang[2*seg:]
		rangeOff := delta[2*seg:]
		for i := 0; i < seg; i++ {
			e, a := binary.BigEndian.Uint16(ende[2*i:]), binary.BigEndian.Uint16(anfang[2*i:])
			d, ro := binary.BigEndian.Uint16(delta[2*i:]), binary.BigEndian.Uint16(rangeOff[2*i:])
			for r := uint32(a); r <= uint32(e) && r != 0xFFFF; r++ {
				var g uint16
				if ro == 0 {
					g = uint16(r) + d
				} else {
					p := 2*i + int(ro) + 2*int(r-uint32(a))
					g = binary.BigEndian.Uint16(rangeOff[p:])
					if g != 0 {
						g += d
					}
				}
				if g != 0 {
					s.zeichen[rune(r)] = int(g)
				}
			}
		}
	default:
		return fmt.Errorf("keine Unicode-cmap (3,10 Format 12 oder 3,1 Format 4)")
	}
	return nil
}

func (s *schrift) glyphe(id int) (*glyphe, error) {
	if g := s.glyphen[id]; g != nil {
		return g, nil
	}
	loca := s.tabellen["loca"]
	var von, bis int
	if s.kurzLoca {
		von, bis = 2*int(binary.BigEndian.Uint16(loca[2*id:])), 2*int(binary.BigEndian.Uint16(loca[2*id+2:]))
	} else {
		von, bis = int(binary.BigEndian.Uint32(loca[4*id:])), int(binary.BigEndian.Uint32(loca[4*id+4:]))
	}
	g := &glyphe{vorschub: s.vorschub[id]}
	s.glyphen[id] = g
	if bis <= von {
		return g, nil // leer (Leerzeichen)
	}
	d := s.tabellen["glyf"][von:bis]
	nk := int(int16(binary.BigEndian.Uint16(d)))
	if nk < 0 {
		return nil, fmt.Errorf("Glyphe %d ist zusammengesetzt (nicht unterstützt)", id)
	}
	endpunkte := make([]int, nk)
	for i := range endpunkte {
		endpunkte[i] = int(binary.BigEndian.Uint16(d[10+2*i:]))
	}
	n := 0
	if nk > 0 {
		n = endpunkte[nk-1] + 1
	}
	p := 10 + 2*nk
	p += 2 + int(binary.BigEndian.Uint16(d[p:])) // Anweisungen überspringen
	flags := make([]byte, 0, n)
	for len(flags) < n {
		f := d[p]
		p++
		flags = append(flags, f)
		if f&8 != 0 {
			w := int(d[p])
			p++
			for ; w > 0; w-- {
				flags = append(flags, f)
			}
		}
	}
	koord := func(kurz, gleich byte) []int {
		werte := make([]int, n)
		v := 0
		for i, f := range flags {
			switch {
			case f&kurz != 0:
				dv := int(d[p])
				p++
				if f&gleich == 0 {
					dv = -dv
				}
				v += dv
			case f&gleich == 0:
				v += int(int16(binary.BigEndian.Uint16(d[p:])))
				p += 2
			}
			werte[i] = v
		}
		return werte
	}
	xs := koord(2, 16)
	ys := koord(4, 32)
	start := 0
	for _, e := range endpunkte {
		k := make([]roh, 0, e+1-start)
		for i := start; i <= e; i++ {
			k = append(k, roh{xs[i], ys[i], flags[i]&1 != 0})
		}
		g.konturen = append(g.konturen, k)
		start = e + 1
	}
	return g, nil
}

// svgStift schreibt Pfadbefehle wie fontTools SVGPathPen (H/V für waagrechte und senkrechte Linien, doppelte Punkte
// fallen weg, Q für quadratische Kurven).
type svgStift struct {
	befehle    []string
	letzter    string
	lx, ly     float64
	hatLetzten bool
}

func (p *svgStift) punktText(q punkt) string { return pyZahl(q.x) + " " + pyZahl(q.y) }

func (p *svgStift) moveTo(q punkt) {
	if p.letzter == "M" {
		p.befehle = p.befehle[:len(p.befehle)-1]
	}
	p.befehle = append(p.befehle, "M"+p.punktText(q))
	p.letzter, p.lx, p.ly, p.hatLetzten = "M", q.x, q.y, true
}

func (p *svgStift) lineTo(q punkt) {
	var t string
	switch {
	case p.hatLetzten && q.x == p.lx && q.y == p.ly:
		return
	case p.hatLetzten && q.x == p.lx:
		t, p.letzter = "V"+pyZahl(q.y), "V"
	case p.hatLetzten && q.y == p.ly:
		t, p.letzter = "H"+pyZahl(q.x), "H"
	case p.letzter == "M":
		t = " " + p.punktText(q)
	default:
		t, p.letzter = "L"+p.punktText(q), "L"
	}
	p.befehle = append(p.befehle, t)
	p.lx, p.ly, p.hatLetzten = q.x, q.y, true
}

func (p *svgStift) qCurveToOne(a, b punkt) {
	p.befehle = append(p.befehle, "Q"+p.punktText(a)+" "+p.punktText(b))
	p.letzter, p.lx, p.ly, p.hatLetzten = "Q", b.x, b.y, true
}

// qCurveTo zerlegt eine TrueType-Folge (letzter Punkt auf der Kurve) wie BasePen.qCurveTo.
func (p *svgStift) qCurveTo(pts []punkt) {
	if len(pts) == 1 {
		p.lineTo(pts[0])
		return
	}
	for i := 0; i < len(pts)-2; i++ {
		a, b := pts[i], pts[i+1]
		p.qCurveToOne(a, punkt{0.5 * (a.x + b.x), 0.5 * (a.y + b.y)})
	}
	p.qCurveToOne(pts[len(pts)-2], pts[len(pts)-1])
}

func (p *svgStift) closePath() {
	p.befehle = append(p.befehle, "Z")
	p.letzter, p.hatLetzten = "Z", false
}

// zeichne zeichnet g mit der Abbildung (s, 0, 0, -s, dx, dy) wie Glyph.draw hinter TransformPen.
func (p *svgStift) zeichne(g *glyphe, s, dx, dy float64) error {
	tr := func(r roh) punkt {
		x, y := float64(r.x), float64(r.y)
		// Wie fontTools Transform.transformPoint: xx*x + yx*y + dx, xy*x + yy*y + dy (explizit gerundet, kein FMA).
		return punkt{float64(float64(s*x)+float64(0*y)) + dx, float64(float64(0*x)+float64(-s*y)) + dy}
	}
	for _, k := range g.konturen {
		erster := -1
		for i, r := range k {
			if r.aufKurve {
				erster = i
				break
			}
		}
		if erster < 0 {
			return fmt.Errorf("Kontur ohne Punkt auf der Kurve (nicht unterstützt)")
		}
		// Umordnen, sodass die Kontur auf einem Punkt auf der Kurve endet (dieser wird das moveTo).
		k = append(append([]roh(nil), k[erster+1:]...), k[:erster+1]...)
		p.moveTo(tr(k[len(k)-1]))
		for len(k) > 0 {
			naechster := 0
			for i, r := range k {
				if r.aufKurve {
					naechster = i + 1
					break
				}
			}
			if naechster == 1 {
				if len(k) > 1 {
					p.lineTo(tr(k[0]))
				}
			} else {
				pts := make([]punkt, naechster)
				for i := range pts {
					pts[i] = tr(k[i])
				}
				p.qCurveTo(pts)
			}
			k = k[naechster:]
		}
		p.closePath()
	}
	return nil
}

// textAlsPfad setzt text als Pfad (Glyphenumrisse) linksbündig ab x, Grundlinie grundlinie, und liefert das
// path-Element und die Laufweite (wie runde6.text_als_pfad).
func (s *schrift) textAlsPfad(text string, x, grundlinie, groesse float64, farbe string, sperrung float64) (string, float64, error) {
	sk := groesse / float64(s.einheiten)
	var pfade []string
	lauf := 0.0
	for _, z := range text {
		id, ok := s.zeichen[z]
		if !ok {
			return "", 0, fmt.Errorf("Zeichen %q fehlt in der Schrift", z)
		}
		g, err := s.glyphe(id)
		if err != nil {
			return "", 0, err
		}
		st := &svgStift{}
		if err := st.zeichne(g, sk, x+lauf, grundlinie); err != nil {
			return "", 0, err
		}
		if len(st.befehle) > 0 {
			pfade = append(pfade, strings.Join(st.befehle, ""))
		}
		lauf += float64(float64(g.vorschub)*sk) + float64(sperrung*groesse)
	}
	return `<path d="` + strings.Join(pfade, " ") + `" fill="` + farbe + `"/>`, lauf, nil
}
