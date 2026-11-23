// SPDX-License-Identifier: GPL-3.0-or-later

package icons

import "math"

// Das „s“ des Logos für die Menüzeile (ADR-0031 Punkt 1). Die Maße stammen aus brand/werkzeug (Datei
// brand/logo/svg/schnapp-s-dunkel.svg); TestLogoSWieMarke hält sie gleich. Nie nachsetzen oder ändern — die Marke
// wird nur in brand/werkzeug gezeichnet.
const (
	// LogoSPfad ist die Mittellinie des „s“ als SVG-Pfad (Strich LogoSStrich, runde Enden und Ecken).
	LogoSPfad   = "M76,100 H25 A25,25 0 0 0 25,150 H51 A25,25 0 0 1 51,200 H0"
	LogoSStrich = 22.0
)

// LogoSBox ist die viewBox der Datei (x, y, Breite, Höhe).
var LogoSBox = [4]float64{-16, 84, 108, 132}

// LogoSAuge: Augapfel (weiß, Rand im Akzent 2) und Pupille (Fenstergrund), Mittelpunkte und Radien.
var (
	LogoSAugapfel = [3]float64{30, 125, 11}
	LogoSPupille  = [3]float64{34, 125, 5.5}
)

// LogoSLinie liefert die Mittellinie des „s“ als Punktfolge (Bögen in n Schritten je Halbkreis) in den Koordinaten
// der Datei. Die Oberfläche zeichnet sie als Strich mit LogoSStrich und runden Enden.
func LogoSLinie(n int) []Punkt {
	p := []Punkt{{76, 100}, {25, 100}}
	// erster Bogen: Mitte (25, 125), Radius 25, von oben über links nach unten (SVG-Sweep 0)
	for i := 1; i <= n; i++ {
		w := -math.Pi/2 - math.Pi*float64(i)/float64(n)
		p = append(p, Punkt{25 + 25*math.Cos(w), 125 + 25*math.Sin(w)})
	}
	p = append(p, Punkt{51, 150})
	// zweiter Bogen: Mitte (51, 175), von oben über rechts nach unten (SVG-Sweep 1)
	for i := 1; i <= n; i++ {
		w := -math.Pi/2 + math.Pi*float64(i)/float64(n)
		p = append(p, Punkt{51 + 25*math.Cos(w), 175 + 25*math.Sin(w)})
	}
	return append(p, Punkt{0, 200})
}
