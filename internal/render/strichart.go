// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"math"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Gemusterte Striche (ADR-0021 Nachtrag 25): Die Stücke rechnet document (StrichAbschnitte, RundRahmenAbschnitte,
// EckigerRahmenMuster); hier werden sie zu Teilen wie der volle Strich — je Strich ein Linienstrich mit geraden Enden, je
// Punkt eine Scheibe. Renderer, Editor und die Vorschauen füllen dieselben Teile.

// musterGewicht sind die Kosten je Stück eines gemusterten Strichs (F-840): Zerlegen kostet rund 0,25 µs je Stück, auch
// für Stücke außerhalb des Bilds, die strichJeStueckMit nicht zählt (gemessen 1,4 Mio. Scheiben in 352 ms).
const musterGewicht = 1000.0

// musterKosten zählt die Stücke eines gemusterten Strichs im Aufwand (Trockenlauf von Pruefe), damit viele große
// Muster das Öffnen nicht beliebig lange aufhalten (F-840).
func (z zeichner) musterKosten(s document.Strich, teile int) {
	if s.Art != document.StrichVoll && !z.ohneZaehlen {
		z.auf.kanten += musterGewicht * float64(teile)
	}
}

// musterLinie fügt die Stücke des Musters m zu einem linienStrich zusammen: Zwischen zwei Stücken liegt eine Lücke
// ohne Teil (strecke −1). So läuft der gemusterte Strich über dieselbe Bereichssuche wie ein voller (strichJeStueckMit).
func musterLinie(m document.Strichmuster, hw float64) linienStrich {
	var ls linienStrich
	dazu := func(sub linienStrich) {
		if len(sub.pts) == 0 {
			return
		}
		if len(ls.pts) > 0 {
			ls.strecke = append(ls.strecke, -1) // Lücke vom letzten Punkt des vorigen Stücks zum ersten dieses
		}
		bt := len(ls.teile)
		ls.teile = append(ls.teile, sub.teile...)
		ls.pts = append(ls.pts, sub.pts...)
		for _, i := range sub.strecke {
			ls.strecke = append(ls.strecke, i+bt)
		}
		for k, g := range sub.gelenk {
			if g >= 0 {
				g += bt
			}
			ls.gelenk = append(ls.gelenk, g)
			ls.gefaltet = append(ls.gefaltet, k < len(sub.gefaltet) && sub.gefaltet[k])
		}
	}
	for _, z := range m.Striche {
		sub := zerlegeLinie(z, false, hw, false)
		if len(sub.pts) >= 2 { // ein Strich ohne Länge ist keiner (keine Scheibe)
			dazu(sub)
		}
	}
	for _, p := range m.Punkte {
		dazu(zerlegeLinie([]document.Punkt{p}, false, hw, true))
	}
	return ls
}

// Scheibe ist der Umriss eines Punkts eines gepunkteten Strichs (Mitte m, Radius hw), wie ihn der Renderer füllt.
func Scheibe(m document.Punkt, hw float64) []document.Punkt {
	return bogen(m, hw, document.Pt(hw, 0), document.Pt(hw, 0), 2*math.Pi)
}

// StrichZerlegung sind die überlappungsfreien Teile eines Strichs um einen Linienzug, voll oder gemustert, und der
// Linienzug für die Suche nach Bereichen (document.StreckenStuecke mit Suche).
type StrichZerlegung struct {
	Teile [][]document.Punkt
	Zug   []document.Punkt
	Zu    bool
	Suche float64
	ls    linienStrich
}

func zerlegungAus(ls linienStrich, hw float64) StrichZerlegung {
	return StrichZerlegung{Teile: ls.teile, Zug: ls.pts, Zu: ls.zu, Suche: hw + 1, ls: ls}
}

// ZerlegeStrich zerlegt den Strich s um den offenen Linienzug pts (Original oder Ansicht): voll mit runden Enden wie
// bisher, sonst nach seiner Strichart (document.StrichAbschnitte).
func ZerlegeStrich(pts []document.Punkt, s document.Strich) StrichZerlegung {
	hw := s.Breite / 2
	if s.Art == document.StrichVoll || !(hw > 0) {
		return zerlegungAus(neuerLinienStrich(pts, false, hw), hw)
	}
	return zerlegungAus(musterLinie(document.StrichAbschnitte(pts, s.Art, s.Breite), hw), hw)
}

// ZerlegeRundenRahmen zerlegt den gemusterten Strich eines runden Kastens (Bereich b, Eckradius radius; Original);
// für den vollen gilt document.NeuerKastenStrich.
func ZerlegeRundenRahmen(b document.Rechteck, radius float64, s document.Strich) StrichZerlegung {
	hw := s.Breite / 2
	return zerlegungAus(musterLinie(document.RundRahmenAbschnitte(b, radius, s.Art, s.Breite), hw), hw)
}

// RahmenTeile sind die Teile des eckigen Rahmens um r mit dem Strich s (Kasten ohne Rundung, Rahmen der Lupe): Rechtecke
// (voll und gestrichelt, document.EckigerRahmenMuster) bzw. Scheiben (gepunktet), alle gleich herum.
func RahmenTeile(r document.Rechteck, s document.Strich) [][]document.Punkt {
	balken, punkte := document.EckigerRahmenMuster(r, s.Breite, s.Art)
	teile := make([][]document.Punkt, 0, len(balken)+len(punkte))
	for _, q := range balken {
		teile = append(teile, []document.Punkt{q.Min, document.Pt(q.Max.X, q.Min.Y), q.Max, document.Pt(q.Min.X, q.Max.Y)})
	}
	for _, p := range punkte {
		teile = append(teile, Scheibe(p, s.Breite/2))
	}
	return teile
}
