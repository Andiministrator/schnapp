// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"fmt"
	"image"
	"math"
)

// Naht ist eine Stauch-Naht in der Ansicht (ADR-0023 Punkt 6): bei Zeilen eine waagrechte Linie in Höhe Lage (dort
// fehlen Zeilen), sonst eine senkrechte bei x = Lage. Lage in Pixeln der Ansicht.
type Naht struct {
	Zeilen bool
	Lage   int
}

// Form, Maße und Schichten der Kante an einer Naht stehen in kante.go (ADR-0023 Nachtrag 6).

// KantenReferenz ist die kurze Seite der Ansicht, bis zu der die Kante in den Grundmaßen bleibt (Faktor 1).
const KantenReferenz = 1080

// KantenFaktor ist der ganzzahlige Faktor der Kantenmaße für ein Bild der Größe g: max(1, kurze Seite / 1080),
// ganzzahlig geteilt (F-173). So bleibt die Kante bei HiDPI-Aufnahmen und kleinem Zoom erkennbar.
func KantenFaktor(g image.Point) int { return max(1, min(g.X, g.Y)/KantenReferenz) }

// KantenFaktor ist der Kantenfaktor des Dokuments (ADR-0023 Nachtrag 2 und 4): der Faktor nach der Größe des
// **Originals** (Nachtrag 2: eine gestauchte 4K-Aufnahme behält ihre Pixeldichte), gedeckelt durch die kurze Seite der
// **aktuellen Ansicht** (Nachtrag 4, F-226): Die Bandhöhe (2·KantenBandHalb) bleibt höchstens ein Viertel der kurzen
// Ansichtsseite, mindestens aber bei Faktor 1. Eine 8K-Aufnahme, die auf 300×80 zugeschnitten wird, trägt so kein
// ±64-Pixel-Band auf 80 Pixeln Höhe. Renderer, SVG-Export, Editor und Trefferweite holen den Wert von hier. Mit der
// halben Bandhöhe 16 (Nachtrag 7; vorher 11) ist der Teiler 128.
func (d *Dokument) KantenFaktor() int {
	kurz := min(d.abbX.groesse(), d.abbY.groesse())
	return min(KantenFaktor(d.original.Rect.Size()), max(1, kurz/(8*KantenBandHalb)))
}

// OriginalGroesse liefert die Größe des unbeschnittenen Originals, ohne es zu kopieren (anders als Original).
func (d *Dokument) OriginalGroesse() image.Point { return d.original.Rect.Size() }

// Naehte liefert die Stauch-Nähte der Ansicht, je Achse nach Lage sortiert. Eine Naht am Rand der Ansicht (der
// Streifen schließt an den Zuschnitt an) zählt nicht, mehrere Streifen an derselben Stelle ergeben eine Naht.
func (d *Dokument) Naehte() []Naht {
	var res []Naht
	for _, zeilen := range []bool{false, true} {
		m := d.abbX
		if zeilen {
			m = d.abbY
		}
		g, letzte := m.groesse(), -1
		for i, w := range m.weg {
			lage := w[0] - m.von - m.kum[i]
			if lage <= 0 || lage >= g || lage == letzte {
				continue
			}
			res = append(res, Naht{Zeilen: zeilen, Lage: lage})
			letzte = lage
		}
	}
	return res
}

// ObjektSichtbar meldet, ob o in die Ansicht reicht (ADR-0023 Punkt 7): Sichtbar für seine Reichweite (bei Text und
// Nummer die echte Tinte der Glyphen, F-181). Ein Objekt, für das es nicht gilt, liegt ganz im Verborgenen (außerhalb
// des Zuschnitts, ganz in einem Streifen oder neben dem Bild); Editor, Renderer, SVG-Export, Trefferprüfung und der
// Zähler der verborgenen Beschriftungen lassen es aus.
func (d *Dokument) ObjektSichtbar(o Objekt) bool { return d.Sichtbar(Reichweite(o)) }

// Reichweite ist die Fläche, die Renderer, Editor und SVG von o zeichnen (Original), knapp nach oben abgeschätzt.
// Text: die echte Tinte der Glyphen in Go Regular (textTinte, F-181) plus 1 Pixel Kantenglättung, mit Hintergrund
// zusätzlich dessen Fläche (Grenzen); ohne Tinte (nur Leerraum) und ohne Hintergrund der Ort. Nummer: der Kreis
// (Grenzen) und die Tinte der Ziffern in Go Bold (nummerTinte) — Ziffern können breiter sein als der Kreis. Pfeil: das
// Dreieck der Spitze (Ecken wie beim Zeichnen) plus halbe Strichbreite und 1 Pixel. Nur wenn die Schrift nicht zu laden ist, gilt die alte grobe Schätzung (schaetzung).
func Reichweite(o Objekt) Rechteck {
	g := o.Grenzen()
	switch v := o.(type) {
	case Text:
		r, ok, geschaetzt := textTinte(v)
		switch {
		case geschaetzt:
			r = schaetzung(v, g)
		case !ok:
			r = Rechteck{Min: v.Ort, Max: v.Ort}
		}
		if !geschaetzt && v.Hintergrund.A > 0 {
			r = r.Vereint(g)
		}
		if v.Spitze != nil {
			// Dreieck der Sprechblase (Füllung oder Schenkel samt Linienbreite) und 1 Pixel Reserve; die Spitze liegt immer
			// in der Hülle, das Dreieck innerhalb von ihr (ADR-0021 Nachtrag 13 Punkt 4).
			h := v.Huelle()
			rand := v.SchenkelBreite()/2 + 1
			r = r.Vereint(Rechteck{Min: h.Min.Minus(Pt(rand, rand)), Max: h.Max.Plus(Pt(rand, rand))})
		}
		return r
	case Nummer:
		r, ok, geschaetzt := nummerTinte(v)
		switch {
		case geschaetzt:
			return schaetzung(v, g)
		case ok:
			return g.Vereint(r)
		}
		return g
	case Pfeil:
		// Die Ecken der Spitze genau wie Renderer und SVG: B, Fuß ± Normale·g/2 mit Fuß = B − r·g (g = max(8, 4·Breite)).
		// Dazu die halbe Strichbreite (Rand der gefüllten Fläche, Kantenglättung) und 1 Pixel Reserve.
		gr := math.Max(8, 4*v.Strich.Breite)
		r := v.RichtungAmEnde()
		n := Pt(-r.Y, r.X)
		fuss := v.B.Minus(r.Mal(gr))
		rand := v.Strich.Breite/2 + 1
		sp := Rechteck{Min: v.B, Max: v.B}
		for _, e := range []Punkt{fuss.Plus(n.Mal(gr / 2)), fuss.Minus(n.Mal(gr / 2))} {
			sp.Min = Punkt{math.Min(sp.Min.X, e.X), math.Min(sp.Min.Y, e.Y)}
			sp.Max = Punkt{math.Max(sp.Max.X, e.X), math.Max(sp.Max.Y, e.Y)}
		}
		sp = Rechteck{Min: sp.Min.Minus(Pt(rand, rand)), Max: sp.Max.Plus(Pt(rand, rand))}
		return g.Vereint(sp)
	}
	return g
}

// schaetzung ist die grobe obere Schranke für Text und Nummer ohne Schrift (sollte nie gebraucht werden): Text je
// Zeichen höchstens 1,1 Schriftgrad (Tabulator vier Zeichen), Glyphen bis 0,25 Schriftgrad links und rechts über, von
// 0,2 Schriftgrad über der Zeile bis 0,3 unter der letzten Grundlinie; Nummer die Ziffernbox des Renderers (halbe
// Breite Grad·(0,4·Stellen + 1), Grad 1,2·Radius).
func schaetzung(o Objekt, g Rechteck) Rechteck {
	switch v := o.(type) {
	case Text:
		zeilen, laengste, akt := 1, 0, 0
		for _, r := range v.Inhalt {
			switch r {
			case '\n':
				zeilen++
				akt = 0
				continue
			case '\t':
				akt += 4
			default:
				akt++
			}
			laengste = max(laengste, akt)
		}
		s := v.Schriftgrad
		// 1 Pixel Reserve rundum: ein Obermaß auch für Zeichen wie U+2044 und U+21A8, deren Tinte weiter reicht.
		return g.Vereint(Rechteck{Min: v.Ort.Minus(Punkt{0.25*s + 1, 0.2*s + 1}),
			Max: v.Ort.Plus(Punkt{(1.1*float64(laengste)+0.25)*s + 1, s*(1+1.3*float64(zeilen-1)) + 0.3*s + 1})})
	case Nummer:
		grad := 1.2 * v.Radius
		halb := grad * (0.4*float64(len(fmt.Sprint(v.Wert))) + 1)
		return g.Vereint(Rechteck{Min: v.Mitte.Minus(Punkt{halb, grad}), Max: v.Mitte.Plus(Punkt{halb, grad})})
	}
	return g
}

// Sichtbar meldet, ob das Rechteck r (Original) mit positiver Überdeckung in einen sichtbaren Teil der Ansicht reicht
// (Regel aus F-66): je Achse schneidet es einen Bereich zwischen Zuschnittkante und Streifen.
func (d *Dokument) Sichtbar(r Rechteck) bool {
	return d.abbX.sichtbar(r.Min.X, r.Max.X) && d.abbY.sichtbar(r.Min.Y, r.Max.Y)
}

// sichtbar meldet, ob [lo, hi] einen sichtbaren Bereich [a, b) der Achse schneidet (lo < b und hi > a).
func (m abbildung) sichtbar(lo, hi float64) bool {
	trifft := func(a, b int) bool { return a < b && lo < float64(b) && hi > float64(a) }
	start := m.von
	for _, w := range m.weg {
		if trifft(start, w[0]) {
			return true
		}
		start = w[1]
	}
	return trifft(start, m.bis)
}
