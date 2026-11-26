// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"math"

	"github.com/Andiministrator/schnapp/internal/document"
)

// PfeilZerlegung sind die überlappungsfreien Teile eines Pfeils (Original; ADR-0021 Nachtrag 19): der Linienstrich des
// Schafts (neuerLinienStrich), dessen runde Kappe an jedem Ende mit Kopf durch den Umriss des Kopfs ersetzt ist — aus
// denselben Eckpunkten wie die Kappe, die gemeinsame Kante hebt sich in der Füllung auf. Renderer, Editor und Vorschau
// der Ebenenliste füllen dieselben Teile.
type PfeilZerlegung struct {
	Teile  [][]document.Punkt // Schaft samt Köpfen in einer Füllung
	Schaft []document.Punkt   // Linienzug des Schafts ohne doppelte Punkte (für die Suche nach Bereichen)
	// Suche ist der größte Abstand eines Teils von seiner Strecke plus 1 Pixel (document.StreckenStuecke).
	Suche float64
	// Einzeln sind Köpfe ohne Schaft (der Pfeil ist kürzer als seine Köpfe), je eine eigene Füllung.
	Einzeln [][]document.Punkt
	ls      linienStrich
}

// ZerlegePfeil zerlegt den Pfeil p.
func ZerlegePfeil(p document.Pfeil) PfeilZerlegung {
	f := p.Form()
	hw := p.Strich.Breite / 2
	z := PfeilZerlegung{Suche: hw + 1}
	ls := neuerLinienStrich(f.Schaft, false, hw)
	if len(ls.pts) < 2 && len(f.Koepfe) > 0 {
		ls = linienStrich{} // kein Schaft: nur die Köpfe
	}
	n := len(ls.pts)
	for _, k := range f.Koepfe {
		if n < 2 {
			z.Einzeln = append(z.Einzeln, k.Umriss(hw))
			continue
		}
		g := ls.gelenk[n-1]
		if k.AmAnfang {
			g = ls.gelenk[0]
		}
		kappe := ls.teile[g]
		neu := make([]document.Punkt, 0, len(k.Kontur)+2)
		neu = append(neu, kappe[0])
		neu = append(neu, k.Kontur...)
		ls.teile[g] = append(neu, kappe[len(kappe)-1])
		for _, q := range k.Kontur {
			z.Suche = math.Max(z.Suche, q.Minus(k.Schaftende).Laenge()+1)
		}
	}
	z.ls, z.Teile, z.Schaft = ls, ls.teile, ls.pts
	return z
}

// pfeil zeichnet den Pfeil v je sichtbarem Bereich (strichJeStueckMit), Köpfe ohne Schaft je für sich.
func (z zeichner) pfeil(v document.Pfeil) {
	pz := ZerlegePfeil(v)
	z.strichJeStueckMit(pz.ls, v.Strich, pz.Suche)
	for _, k := range pz.Einzeln {
		z.jeStueck(huelleVon(k, 1), func(zs zeichner, abb func(document.Punkt) document.Punkt) {
			pts := make([]document.Punkt, len(k))
			for i, q := range k {
				pts[i] = abb(q)
			}
			zs.fuelle([][]document.Punkt{pts}, nil, v.Strich.Farbe)
		})
	}
}
