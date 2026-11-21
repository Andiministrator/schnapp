// SPDX-License-Identifier: GPL-3.0-or-later

// Package canvas ist die Logik der Editor-Leinwand ohne Gio (Karte #35): Abbildung zwischen Ansicht und Fenster mit
// Zoom und Verschieben, Treffer auf Objekte, Griffe, Ziehen. Die Oberfläche (Paket ui) übersetzt Eingaben und zeichnet
// nur; Änderungen am Dokument laufen über dessen Befehle und Züge (Paket document).
//
// Koordinaten: Fensterpunkte in Pixeln des Fensterpuffers, Ansichtspunkte in Pixeln der Ansicht (Original nach
// Zuschnitt und Stauchen, document.Dokument.Groesse), Originalpunkte in Pixeln des Originals. Objekte liegen im
// Original; Treffer und Griffe rechnen deshalb im Original, die Toleranz kommt in Fensterpixeln und wird mit dem Zoom
// umgerechnet.
package canvas

import (
	"image"
	"math"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Grenzen des Zooms: 1/32 bis 32-fach.
const (
	MinZoom = 1.0 / 32
	MaxZoom = 32.0
)

// Abbildung bildet Ansichtspunkte auf Fensterpunkte ab: fenster = ansicht·Zoom + Versatz.
type Abbildung struct {
	Zoom    float64
	Versatz document.Punkt
}

// Einpassen zeigt eine Ansicht der Größe bild mittig in einem Fenster der Größe fenster mit rand Pixeln Abstand,
// höchstens 1:1 (kleine Bilder werden nicht vergrößert). Leere Größen ergeben Zoom 1 ohne Versatz.
func Einpassen(bild, fenster image.Point, rand float64) Abbildung {
	if bild.X <= 0 || bild.Y <= 0 || fenster.X <= 0 || fenster.Y <= 0 {
		return Abbildung{Zoom: 1}
	}
	fx, fy := math.Max(1, float64(fenster.X)-2*rand), math.Max(1, float64(fenster.Y)-2*rand)
	z := math.Min(1, math.Min(fx/float64(bild.X), fy/float64(bild.Y)))
	z = begrenzt(z)
	return Abbildung{
		Zoom: z,
		Versatz: document.Pt(
			(float64(fenster.X)-float64(bild.X)*z)/2,
			(float64(fenster.Y)-float64(bild.Y)*z)/2,
		),
	}
}

// ZuFenster rechnet einen Ansichtspunkt ins Fenster um.
func (a Abbildung) ZuFenster(p document.Punkt) document.Punkt { return p.Mal(a.Zoom).Plus(a.Versatz) }

// ZuAnsicht rechnet einen Fensterpunkt in die Ansicht um.
func (a Abbildung) ZuAnsicht(p document.Punkt) document.Punkt {
	return p.Minus(a.Versatz).Mal(1 / a.Zoom)
}

// ZoomUm ändert den Zoom um faktor so, dass der Fensterpunkt um an derselben Stelle der Ansicht bleibt (Zoomen zum
// Mauszeiger). Der Zoom bleibt in [MinZoom, MaxZoom]; ein ungültiger Faktor ändert nichts.
func (a Abbildung) ZoomUm(um document.Punkt, faktor float64) Abbildung {
	if !(faktor > 0) || math.IsInf(faktor, 0) {
		return a
	}
	fest := a.ZuAnsicht(um)
	a.Zoom = begrenzt(a.Zoom * faktor)
	a.Versatz = um.Minus(fest.Mal(a.Zoom))
	return a
}

// Verschoben verschiebt die Ansicht im Fenster um d Fensterpixel.
func (a Abbildung) Verschoben(d document.Punkt) Abbildung {
	a.Versatz = a.Versatz.Plus(d)
	return a
}

// Eins ist 1:1 (ein Ansichtspixel je Fensterpixel), so dass um fest bleibt.
func (a Abbildung) Eins(um document.Punkt) Abbildung { return a.ZoomUm(um, 1/a.Zoom) }

func begrenzt(z float64) float64 { return math.Min(MaxZoom, math.Max(MinZoom, z)) }
