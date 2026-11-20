// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

// Naht ist eine Stauch-Naht in der Ansicht (ADR-0023 Punkt 6): bei Zeilen eine waagrechte Linie in Höhe Lage (dort
// fehlen Zeilen), sonst eine senkrechte bei x = Lage. Lage in Pixeln der Ansicht.
type Naht struct {
	Zeilen bool
	Lage   int
}

// Grundmaße und Farben der Zackenkante (ADR-0023 Punkt 6), in Pixeln der Ansicht bei Faktor 1: Zacken ±ZackenHoehe um
// die Naht, eine Zahnbreite ZackenWeite; außen ein helles Band der Dicke 2·ZackenRandHalb, darin eine dunkle Linie der
// Dicke 2·ZackenLinieHalb. Alle vier Maße werden mit ZackenFaktor der Ansicht multipliziert (ADR-0023 Nachtrag 1,
// F-173). Renderer, SVG-Export und Editor zeichnen dieselben Vielecke (Zackenband).
const (
	ZackenHoehe     = 4.0
	ZackenWeite     = 16.0
	ZackenRandHalb  = 3.0
	ZackenLinieHalb = 1.0
)

var (
	ZackenRand  = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	ZackenLinie = color.NRGBA{R: 0x4A, G: 0x50, B: 0x57, A: 0xFF}
)

// ZackenReferenz ist die kurze Seite der Ansicht, bis zu der die Zackenkante in den Grundmaßen bleibt (Faktor 1).
const ZackenReferenz = 1080

// ZackenFaktor ist der ganzzahlige Faktor der Zackenmaße für eine Ansicht der Größe g: max(1, kurze Seite / 1080),
// ganzzahlig geteilt (F-173). So bleibt die Kante bei HiDPI-Aufnahmen und kleinem Zoom erkennbar. Er hängt nur von der
// Ansichtsgröße ab; Zwischenspeicher, die diese Größe als Schlüssel führen, sind damit schon richtig geschlüsselt.
func ZackenFaktor(g image.Point) int { return max(1, min(g.X, g.Y)/ZackenReferenz) }

// ZackenFaktor ist ZackenFaktor(d.Groesse()).
func (d *Dokument) ZackenFaktor() int { return ZackenFaktor(d.Groesse()) }

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

// Zackenband liefert ein Vieleck (Ansicht) entlang der Naht n über die Länge laenge (Breite bzw. Höhe der Ansicht):
// eine Zickzacklinie mit Zacken ±ZackenHoehe, quer zur Naht um ±halb aufgeweitet. Die Zacken beginnen bei 0 und
// reichen höchstens eine halbe Zahnbreite über laenge hinaus (Renderer und SVG beschneiden auf die Ansicht). Zacken-
// höhe, Zahnbreite und halb (ein Grundmaß, z. B. ZackenRandHalb) werden mit faktor (ZackenFaktor, mindestens 1)
// multipliziert.
func Zackenband(n Naht, laenge, faktor int, halb float64) []Punkt {
	f := float64(max(faktor, 1))
	halb *= f
	hoehe := ZackenHoehe * f
	schritt := ZackenWeite * f / 2
	k := int(math.Ceil(float64(max(laenge, 1)) / schritt))
	oben := make([]Punkt, 0, 2*(k+1))
	unten := make([]Punkt, 0, k+1)
	for i := 0; i <= k; i++ {
		t := float64(i) * schritt
		q := float64(n.Lage) - hoehe
		if i%2 == 1 {
			q = float64(n.Lage) + hoehe
		}
		oben = append(oben, quer(n, t, q-halb))
		unten = append(unten, quer(n, t, q+halb))
	}
	for i := len(unten) - 1; i >= 0; i-- {
		oben = append(oben, unten[i])
	}
	return oben
}

// quer baut einen Punkt aus der Lage t entlang der Naht und q quer dazu.
func quer(n Naht, t, q float64) Punkt {
	if n.Zeilen {
		return Punkt{t, q}
	}
	return Punkt{q, t}
}

// ObjektSichtbar meldet, ob o in die Ansicht reicht (ADR-0023 Punkt 7): Sichtbar für seine Reichweite (bei Text und
// Nummer die echte Tinte der Glyphen, F-181). Ein Objekt, für das es nicht gilt, liegt ganz im Verborgenen (außerhalb
// des Zuschnitts, ganz in einem Streifen oder neben dem Bild); Editor, Renderer, SVG-Export, Trefferprüfung und der
// Zähler der verborgenen Beschriftungen lassen es aus.
func (d *Dokument) ObjektSichtbar(o Objekt) bool { return d.Sichtbar(Reichweite(o)) }

// Reichweite ist die Fläche, die Renderer, Editor und SVG von o zeichnen (Original), knapp nach oben abgeschätzt.
// Text: die echte Tinte der Glyphen in Go Regular (textTinte, F-181) plus 1 Pixel Kantenglättung, mit Hintergrund
// zusätzlich dessen Fläche (Grenzen); ohne Tinte (nur Leerraum) und ohne Hintergrund der Ort. Nummer: der Kreis
// (Grenzen) und die Tinte der Ziffern in Go Bold (nummerTinte) — Ziffern können breiter sein als der Kreis. Pfeil: die
// Spitze (max(8, 4·Breite)) um B. Nur wenn die Schrift nicht zu laden ist, gilt die alte grobe Schätzung (schaetzung).
func Reichweite(o Objekt) Rechteck {
	g := o.Grenzen()
	switch v := o.(type) {
	case Text:
		r, ok, geschaetzt := textTinte(v)
		switch {
		case geschaetzt:
			return schaetzung(v, g)
		case !ok:
			r = Rechteck{Min: v.Ort, Max: v.Ort}
		}
		if v.Hintergrund.A > 0 {
			r = r.Vereint(g)
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
		s := math.Max(8, 4*v.Strich.Breite)
		return g.Vereint(Rechteck{Min: v.B.Minus(Punkt{s, s}), Max: v.B.Plus(Punkt{s, s})})
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
		return g.Vereint(Rechteck{Min: v.Ort.Minus(Punkt{0.25 * s, 0.2 * s}),
			Max: v.Ort.Plus(Punkt{(1.1*float64(laengste) + 0.25) * s, s*(1+1.3*float64(zeilen-1)) + 0.3*s})})
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
