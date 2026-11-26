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
	// Einzeln sind Köpfe ohne Schaft (der Pfeil ist kürzer als seine Köpfe), zusammen eine Füllung.
	Einzeln [][]document.Punkt
	ls      linienStrich
}

// ZerlegePfeil zerlegt den Pfeil p. Gemustert (ADR-0021 Nachtrag 25) ist der Schaft eine Folge von Strichen mit geraden
// Enden bzw. Punkten: Der Kopf schließt bündig an den letzten Strich an (eigener Teil mit denselben Eckpunkten wie dessen
// gerades Ende); gepunktet ersetzt er den letzten Punkt durch die halbe Scheibe hinter sich, in einem Teil mit ihm.
func ZerlegePfeil(p document.Pfeil) PfeilZerlegung {
	f := p.Form()
	hw := p.Strich.Breite / 2
	z := PfeilZerlegung{Suche: hw + 1}
	art := p.Strich.Art
	var ls linienStrich
	if art == document.StrichVoll || !(hw > 0) {
		ls = neuerLinienStrich(f.Schaft, false, hw)
	} else {
		ls = musterLinie(document.StrichAbschnitte(f.Schaft, art, p.Strich.Breite), hw)
	}
	if len(ls.pts) < 2 && len(f.Koepfe) > 0 {
		ls = linienStrich{} // kein Schaft: nur die Köpfe
	}
	n := len(ls.pts)
	for _, k := range f.Koepfe {
		if n < 2 {
			z.Einzeln = append(z.Einzeln, k.Umriss(hw))
			continue
		}
		stelle := n - 1
		if k.AmAnfang {
			stelle = 0
		}
		g := ls.gelenk[stelle]
		switch {
		case art == document.StrichVoll:
			kappe := ls.teile[g]
			neu := make([]document.Punkt, 0, len(k.Kontur)+2)
			neu = append(neu, kappe[0])
			neu = append(neu, k.Kontur...)
			ls.teile[g] = append(neu, kappe[len(kappe)-1])
		case art == document.StrichGepunktet && g >= 0:
			// Halbe Scheibe von links (Schaftnormale) über rückwärts nach rechts, hinter den Umriss gehängt.
			u := k.Umriss(hw)
			hinten := bogen(k.Schaftende, hw, k.SchaftNormale.Mal(hw), k.SchaftNormale.Mal(-hw), math.Pi)
			ls.teile[g] = append(u, hinten[1:len(hinten)-1]...)
		default: // gestrichelt: gerades Ende ohne Teil, der Kopf wird das Gelenk
			ls.gelenk[stelle] = len(ls.teile)
			ls.teile = append(ls.teile, k.Umriss(hw))
		}
		for _, q := range k.Kontur {
			z.Suche = math.Max(z.Suche, q.Minus(k.Schaftende).Laenge()+1)
		}
	}
	z.ls, z.Teile, z.Schaft = ls, ls.teile, ls.pts
	return z
}

// pfeil zeichnet den Pfeil v je sichtbarem Bereich (strichJeStueckMit), Köpfe ohne Schaft zusammen in einer Füllung.
func (z zeichner) pfeil(v document.Pfeil) {
	pz := ZerlegePfeil(v)
	z.strichJeStueckMit(pz.ls, v.Strich, pz.Suche)
	if len(pz.Einzeln) > 0 {
		// Köpfe ohne Schaft in einer Füllung: Überlappen sie (kurzer Pfeil mit zwei Spitzen), deckt eine
		// halbdurchsichtige Farbe trotzdem einmal (Sitzungs-Kritik 26.11. K-6). Alle Umrisse laufen gleich herum.
		var alle []document.Punkt
		for _, k := range pz.Einzeln {
			alle = append(alle, k...)
		}
		z.jeStueck(huelleVon(alle, 1), func(zs zeichner, abb func(document.Punkt) document.Punkt) {
			teile := make([][]document.Punkt, len(pz.Einzeln))
			for i, k := range pz.Einzeln {
				teile[i] = make([]document.Punkt, len(k))
				for j, q := range k {
					teile[i][j] = abb(q)
				}
			}
			zs.fuelle(teile, nil, v.Strich.Farbe)
		})
	}
}
