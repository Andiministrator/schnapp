// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"math"
	"slices"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Einrasten und Hilfslinien (ADR-0021 Nachtrag 12, Karte #196): Beim Verschieben eines Objekts, beim Ziehen eines
// Griffs und beim Aufziehen eines neuen Pfeils, Kastens oder einer Nummer rasten Kanten und Mitten an den Kanten und
// Mitten der anderen sichtbaren Objekte und des sichtbaren Bilds ein. Gerechnet wird im Original; die Fangweite
// gilt in Fensterpixeln, also unabhängig vom Zoom.

// FangWeite ist die Fangweite in Fensterpixeln.
const FangWeite = 6.0

// Hilfslinie ist eine Linie, an der gerade eingerastet ist (Original). Waagrecht: y = Lage, sonst x = Lage.
type Hilfslinie struct {
	Waagrecht bool
	Lage      float64
}

// SetzeAlt meldet die Alt-Taste (die Oberfläche setzt sie je Zeigerereignis): Solange sie gedrückt ist, rastet nichts
// ein.
func (s *Sitzung) SetzeAlt(an bool) { s.alt = an }

// AltTaste meldet, dass die Alt-Taste gedrückt oder losgelassen wurde, ohne dass sich der Zeiger bewegt hat (Tasten-
// ereignis der Oberfläche): Ein laufender Zug wird sofort am letzten Zeigerort neu gerechnet, das Einrasten schaltet
// also ohne die nächste Zeigerbewegung ab oder wieder an (ADR-0021 Nachtrag 12 Punkt 4, Kritik zu Karte #196).
func (s *Sitzung) AltTaste(an bool) {
	if s.alt == an {
		return
	}
	s.alt = an
	if s.bewegt && !s.pan && (s.zug == zugObjekt || s.zug == zugGriff || s.zug == zugNeu) {
		s.Ziehen(s.zugOrt)
	}
}

// EinrastenAn meldet, ob das Einrasten eingeschaltet ist (Menü Ansicht; Voreinstellung an, nicht gespeichert).
func (s *Sitzung) EinrastenAn() bool { return !s.einrastenAus }

// SchalteEinrasten schaltet das Einrasten um.
func (s *Sitzung) SchalteEinrasten() { s.einrastenAus = !s.einrastenAus }

// Hilfslinien liefert die Linien, an denen der laufende Zug gerade einrastet (nur während des Ziehens, sonst nil).
func (s *Sitzung) Hilfslinien() []Hilfslinie {
	if !s.bewegt || (s.zug != zugObjekt && s.zug != zugGriff && s.zug != zugNeu && s.zug != zugMehrere) {
		return nil
	}
	return append([]Hilfslinie(nil), s.linien...)
}

// einrastenAktiv meldet, ob der Zug einrasten darf: eingeschaltet, Alt nicht gedrückt, nicht in der Vollansicht des
// Zuschneidens.
func (s *Sitzung) einrastenAktiv() bool { return !s.einrastenAus && !s.alt && !s.zuschnittModus() }

// rastZiele sammelt die Linien, an denen eingerastet wird (Original): Kanten und Mitte der sichtbaren Objekte außer
// dem mit der Kennung ohne, dazu Ränder und Mitte des sichtbaren Bilds. Ausgeblendete Ebenen und Objekte, die ganz im
// Verborgenen liegen, zählen nicht; von den übrigen Objekten zählen nur die Linien, die selbst zu sehen sind
// (linieSichtbar). Die Mitte des Bilds ist die Mitte der Ansicht, ins Original zurückgerechnet: Bei gestauchten
// Bildern liegt die Mitte zwischen den Rändern des Originals woanders, womöglich in einem herausgeschnittenen Streifen.
func (s *Sitzung) rastZiele(ohne ...int) (xs, ys []float64) {
	g := s.D.Groesse()
	a, b := s.D.ZuOriginal(document.Pt(0, 0)), s.D.ZuOriginal(document.Pt(float64(g.X), float64(g.Y)))
	m := s.D.ZuOriginal(document.Pt(float64(g.X)/2, float64(g.Y)/2))
	xs = []float64{a.X, m.X, b.X}
	ys = []float64{a.Y, m.Y, b.Y}
	for _, o := range s.D.Objekte() {
		if slices.Contains(ohne, o.Kennung()) || o.Ebenenwerte().Ausgeblendet || !s.D.ObjektSichtbar(o) {
			continue
		}
		rs := []document.Rechteck{o.Grenzen()}
		if l, ok := o.(document.Lupe); ok {
			rs = []document.Rechteck{l.Quelle, l.Ziel} // beide Rechtecke zählen, nicht ihre Hülle (ADR-0033 Punkt 9)
		}
		if t, ok := o.(document.Text); ok && t.Spitze != nil { // die Spitze einer Sprechblase zählt als Punkt
			if s.linieSichtbar(t.Spitze.X, false) {
				xs = append(xs, t.Spitze.X)
			}
			if s.linieSichtbar(t.Spitze.Y, true) {
				ys = append(ys, t.Spitze.Y)
			}
		}
		for _, r := range rs {
			for _, x := range []float64{r.Min.X, (r.Min.X + r.Max.X) / 2, r.Max.X} {
				if s.linieSichtbar(x, false) {
					xs = append(xs, x)
				}
			}
			for _, y := range []float64{r.Min.Y, (r.Min.Y + r.Max.Y) / 2, r.Max.Y} {
				if s.linieSichtbar(y, true) {
					ys = append(ys, y)
				}
			}
		}
	}
	return xs, ys
}

// linieSichtbar meldet, ob die Linie x = lage (waagrecht: y = lage) des Originals im sichtbaren Bild liegt, also weder
// außerhalb des Zuschnitts noch in einem herausgeschnittenen Streifen (die Kanten eines Streifens zählen als sichtbar).
// An einer verborgenen Linie rastet nichts ein: Die Hilfslinie stünde an der Naht oder neben dem Bild, und das Ende
// eines neuen Objekts läge im Verborgenen (F-150, F-157).
func (s *Sitzung) linieSichtbar(lage float64, waagrecht bool) bool {
	const eps, weit = 1e-6, 1e18
	if waagrecht {
		return s.D.Sichtbar(document.Rechteck{Min: document.Pt(-weit, lage-eps), Max: document.Pt(weit, lage+eps)})
	}
	return s.D.Sichtbar(document.Rechteck{Min: document.Pt(lage-eps, -weit), Max: document.Pt(lage+eps, weit)})
}

// rasteAchse liefert die kleinste Verschiebung, mit der einer der Werte mv auf eines der Ziele fällt, wenn sie höchstens
// weite beträgt, und die Lage des Ziels.
func rasteAchse(mv, ziele []float64, weite float64) (versatz, lage float64, ok bool) {
	best := weite + 1
	for _, m := range mv {
		for _, z := range ziele {
			if a := math.Abs(z - m); a <= weite && a < best {
				best, versatz, lage, ok = a, z-m, z, true
			}
		}
	}
	return versatz, lage, ok
}

// rasteRechteck rastet das um d verschobene Rechteck g (Kanten und Mitten) ein und liefert den neuen Versatz und
// die Linien.
func (s *Sitzung) rasteRechteck(g document.Rechteck, d document.Punkt, ohne ...int) (document.Punkt, []Hilfslinie) {
	if !s.einrastenAktiv() {
		return d, nil
	}
	xs, ys := s.rastZiele(ohne...)
	weite := FangWeite / s.Abb.Zoom
	var linien []Hilfslinie
	if v, l, ok := rasteAchse([]float64{g.Min.X + d.X, (g.Min.X+g.Max.X)/2 + d.X, g.Max.X + d.X}, xs, weite); ok {
		d.X += v
		linien = append(linien, Hilfslinie{Lage: l})
	}
	if v, l, ok := rasteAchse([]float64{g.Min.Y + d.Y, (g.Min.Y+g.Max.Y)/2 + d.Y, g.Max.Y + d.Y}, ys, weite); ok {
		d.Y += v
		linien = append(linien, Hilfslinie{Waagrecht: true, Lage: l})
	}
	return d, linien
}

// rastePunkt rastet den Punkt q (Original) ein, den ein Griff oder das Ende eines neuen Objekts trägt.
func (s *Sitzung) rastePunkt(q document.Punkt, ohne int) (document.Punkt, []Hilfslinie) {
	if !s.einrastenAktiv() {
		return q, nil
	}
	xs, ys := s.rastZiele(ohne)
	weite := FangWeite / s.Abb.Zoom
	var linien []Hilfslinie
	if v, l, ok := rasteAchse([]float64{q.X}, xs, weite); ok {
		q.X += v
		linien = append(linien, Hilfslinie{Lage: l})
	}
	if v, l, ok := rasteAchse([]float64{q.Y}, ys, weite); ok {
		q.Y += v
		linien = append(linien, Hilfslinie{Waagrecht: true, Lage: l})
	}
	return q, linien
}

// nurWennGleich behält von den Linien nur die, deren Achse im Ergebnis (nach dem Begrenzen) noch auf dem
// Wunschwert liegt: Eine gekürzte Verschiebung rastet nicht mehr ein, also zeigt sie auch keine Linie.
func nurWennGleich(linien []Hilfslinie, wunsch, ergebnis document.Punkt) []Hilfslinie {
	var res []Hilfslinie
	for _, l := range linien {
		w, e := wunsch.X, ergebnis.X
		if l.Waagrecht {
			w, e = wunsch.Y, ergebnis.Y
		}
		if math.Abs(w-e) < 1e-6 {
			res = append(res, l)
		}
	}
	return res
}

// linienAmGriff behält die Linien, auf denen der Griff art des Ergebnisses o tatsächlich liegt: Eine Kantengriff
// bewegt nur eine Achse, die andere Linie bliebe sonst ohne Bezug stehen.
//
// Beim Kasten zählen je bewegter Achse (griffAchsen) beide Kanten des Ergebnisses, nicht der Ort des Griffs: Wird eine
// Seite über die andere gezogen, ordnet ZieheGriff den Bereich neu, und der gezogene Punkt gehört danach zu einem
// anderen Griff — die Kante liegt trotzdem auf der Linie.
func linienAmGriff(linien []Hilfslinie, o document.Objekt, art GriffArt) []Hilfslinie {
	var xs, ys []float64
	if k, ok := o.(document.Kasten); ok {
		x, y := griffAchsen(art)
		if x {
			xs = []float64{k.Bereich.Min.X, k.Bereich.Max.X}
		}
		if y {
			ys = []float64{k.Bereich.Min.Y, k.Bereich.Max.Y}
		}
	} else {
		for _, gr := range Griffe(o) {
			if gr.Art == art {
				xs, ys = []float64{gr.Ort.X}, []float64{gr.Ort.Y}
			}
		}
	}
	var res []Hilfslinie
	for _, l := range linien {
		werte := xs
		if l.Waagrecht {
			werte = ys
		}
		for _, w := range werte {
			if math.Abs(w-l.Lage) < 1e-6 {
				res = append(res, l)
				break
			}
		}
	}
	return res
}

// griffAchsen meldet, welche Achsen ein Griff bewegt: die Kantengriffe des Kastens nur eine, alle anderen beide.
func griffAchsen(art GriffArt) (x, y bool) {
	switch art {
	case GriffN, GriffS:
		return false, true
	case GriffO, GriffW:
		return true, false
	}
	return true, true
}

// AltGedrueckt meldet die zuletzt über SetzeAlt gemeldete Alt-Taste (für Tests der Oberfläche).
func (s *Sitzung) AltGedrueckt() bool { return s.alt }
