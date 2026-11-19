// SPDX-License-Identifier: GPL-3.0-or-later

package auswahl

import (
	"image"
	"math"
)

// Modus des Overlays.
type Modus int

const (
	// ModusBereich: Bereich aufziehen (`--region`, Druck-Taste). Ein Klick ohne Ziehen wählt nichts.
	ModusBereich Modus = iota
	// ModusMonitor: Monitor per Klick wählen (`--screen` mit Voreinstellung `pick`, ADR-0017 Nachtrag 3). Ein Klick
	// nimmt den ganzen Monitor, Ziehen wählt doch einen Bereich.
	ModusMonitor
)

// KlickSchwelle: Bewegt sich der Zeiger zwischen Drücken und Loslassen um weniger Fensterpixel, gilt es als Klick.
const KlickSchwelle = 4

// Ergebnis einer Auswahl auf einem Monitor.
type Ergebnis struct {
	Fertig  bool            // true, sobald gewählt ist
	Bereich image.Rectangle // in Bildpixeln, nie leer, wenn Fertig
}

// Zustand ist die Auswahl auf einem Monitor (ein Overlay-Fenster). Der Aufrufer meldet Zeiger-Ereignisse in
// Fensterkoordinaten; die Abbildung setzt er bei jeder Fenstergröße neu (SetzeAbbildung).
type Zustand struct {
	Modus   Modus
	Groesse image.Point // Bildgröße in Pixeln
	Abb     Abbildung

	zeiger    Punkt
	imFenster bool
	gedrueckt bool
	gezogen   bool // Zeiger hat sich über die Klick-Schwelle bewegt
	start     Punkt
}

// Neu legt den Zustand für ein Bild der Größe groesse an; bis zur ersten Fenstergröße gilt die Abbildung 1:1.
func Neu(m Modus, groesse image.Point) *Zustand {
	return &Zustand{Modus: m, Groesse: groesse, Abb: Abbildung{Zoom: 1}}
}

// SetzeAbbildung passt die Abbildung an eine neue Fenstergröße an.
func (z *Zustand) SetzeAbbildung(fenster image.Point) { z.Abb = NeueAbbildung(z.Groesse, fenster) }

// Bewegt meldet die Zeigerposition (auch während des Ziehens).
func (z *Zustand) Bewegt(p Punkt) {
	z.zeiger, z.imFenster = p, true
	if z.gedrueckt && !z.gezogen && abstand(p, z.start) >= KlickSchwelle {
		z.gezogen = true
	}
}

// Verlassen meldet, dass der Zeiger das Fenster verlassen hat (anderer Monitor).
func (z *Zustand) Verlassen() { z.imFenster = false }

// Gedrueckt meldet das Drücken der Haupttaste.
func (z *Zustand) Gedrueckt(p Punkt) {
	z.Bewegt(p)
	z.gedrueckt, z.gezogen, z.start = true, false, p
}

// Losgelassen meldet das Loslassen und liefert das Ergebnis: ein aufgezogener Bereich, im Monitor-Modus bei einem
// Klick der ganze Monitor, sonst nichts (Fertig false, die Auswahl geht weiter).
func (z *Zustand) Losgelassen(p Punkt) Ergebnis {
	z.Bewegt(p)
	if !z.gedrueckt {
		return Ergebnis{}
	}
	z.gedrueckt = false
	if !z.gezogen {
		if z.Modus == ModusMonitor {
			return Ergebnis{Fertig: true, Bereich: image.Rectangle{Max: z.Groesse}}
		}
		return Ergebnis{}
	}
	r := z.rahmen(z.start, p)
	z.gezogen = false
	if r.Empty() {
		return Ergebnis{}
	}
	return Ergebnis{Fertig: true, Bereich: r}
}

// Abgebrochen meldet ein Zeiger-Cancel (z. B. Fokusverlust): ein laufendes Ziehen endet ohne Ergebnis.
func (z *Zustand) Abgebrochen() { z.gedrueckt, z.gezogen = false, false }

// Ganz liefert den ganzen Monitor (Enter-Taste).
func (z *Zustand) Ganz() Ergebnis {
	return Ergebnis{Fertig: true, Bereich: image.Rectangle{Max: z.Groesse}}
}

// Rahmen ist der gerade aufgezogene Bereich in Bildpixeln; ok ist false, solange nicht gezogen wird.
func (z *Zustand) Rahmen() (image.Rectangle, bool) {
	if !z.gedrueckt || !z.gezogen {
		return image.Rectangle{}, false
	}
	r := z.rahmen(z.start, z.zeiger)
	return r, !r.Empty()
}

// Zeiger liefert die letzte Zeigerposition und ob der Zeiger im Fenster ist.
func (z *Zustand) Zeiger() (Punkt, bool) { return z.zeiger, z.imFenster }

// Hervorgehoben sagt, ob das Overlay den ganzen Monitor als Ziel zeigt: im Monitor-Modus, solange der Zeiger im
// Fenster ist und nicht gezogen wird.
func (z *Zustand) Hervorgehoben() bool {
	return z.Modus == ModusMonitor && z.imFenster && !z.gezogen
}

// rahmen: Bildpixel, die der Rahmen zwischen zwei Fensterpunkten ganz oder teilweise überdeckt — der Pixel unter
// dem Startpunkt und der unter dem Endpunkt gehören beide dazu. Auf das Bild beschnitten.
func (z *Zustand) rahmen(a, b Punkt) image.Rectangle {
	p, q := z.Abb.Pixel(a, z.Groesse), z.Abb.Pixel(b, z.Groesse)
	r := image.Rectangle{Min: p, Max: q}.Canon()
	r.Max = r.Max.Add(image.Pt(1, 1))
	return r.Intersect(image.Rectangle{Max: z.Groesse})
}

func abstand(a, b Punkt) float64 { return math.Hypot(float64(a.X-b.X), float64(a.Y-b.Y)) }
