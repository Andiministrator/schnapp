// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"image/color"
	"math"

	"github.com/Andiministrator/schnapp/internal/document"
)

// FeldArt ist die Art eines Felds der Fußzeile (ADR-0031 Punkt 4), in der Reihenfolge von links nach rechts.
type FeldArt int

const (
	FeldAuswahl FeldArt = iota // Größe des gewählten Objekts
	FeldZeiger                 // Zeigerort im Bild
	FeldFarbe                  // Farbe des Werkzeugs mit Deckkraft (anklickbar)
	FeldSchrift                // Schriftgrad (Text-Werkzeug oder gewählter Text)
	FeldBild                   // Größe des sichtbaren Bilds, ob gestaucht
	FeldZoom                   // Zoom (anklickbar)
	FeldStand                  // ungesichert oder gespeichert
)

// FussFeld ist ein Feld der Fußzeile ohne Text: Die Oberfläche setzt den Text in der Sprache (ADR-0014).
type FussFeld struct {
	Art         FeldArt
	Groesse     image.Point // FeldAuswahl, FeldBild
	Ort         image.Point // FeldZeiger (ganze Pixel des Originals)
	Farbe       color.NRGBA // FeldFarbe
	Grad        int         // FeldSchrift (Pixel des Originals)
	Zoom        int         // FeldZoom, Prozent
	Gestaucht   bool        // FeldBild
	Ungesichert bool        // FeldStand
	Klickbar    bool        // Farbe und Zoom
}

// ZeigerOrt rechnet einen Zeigerort in Ansichtskoordinaten in den Ort im Original um (die Koordinaten der Objekte und
// der Auswahlgröße; bei Zuschnitt oder Stauchen also nicht der Ort in der Ansicht). Beim Zuschneiden zeigt die
// Leinwand die Vollansicht, dann gilt deren Fläche. false: Der Zeiger liegt nicht über dem gezeigten Bild (F-513).
func (s *Sitzung) ZeigerOrt(a document.Punkt) (image.Point, bool) {
	f := s.Bildflaeche()
	if !(a.X >= float64(f.Min.X) && a.Y >= float64(f.Min.Y) && a.X < float64(f.Max.X) && a.Y < float64(f.Max.Y)) {
		return image.Point{}, false
	}
	var o document.Punkt
	if s.zuschnittModus() {
		v := s.D.VollVersatz()
		o = s.D.VollZuOriginal(a.Plus(document.Pt(float64(v.X), float64(v.Y))))
	} else {
		o = s.D.ZuOriginal(a)
	}
	return image.Pt(int(math.Floor(o.X)), int(math.Floor(o.Y))), true
}

// FussFelder liefert die Felder der Fußzeile; zeiger ist der Zeigerort in Ansichtskoordinaten (nil: Zeiger nicht über
// dem Bild); das Feld zeigt den Ort im Original (ZeigerOrt). Ein Feld fehlt, wenn es nichts zu sagen hat: Auswahl ohne gewähltes Objekt, Zeiger außerhalb, Schrift
// außer beim Text.
func (s *Sitzung) FussFelder(zeiger *document.Punkt) []FussFeld {
	var f []FussFeld
	var text *document.Text
	if s.Auswahl != 0 {
		if o, ok := s.D.Objekt(s.Auswahl); ok {
			g := o.Grenzen()
			f = append(f, FussFeld{Art: FeldAuswahl, Groesse: image.Pt(int(math.Round(g.Max.X-g.Min.X)),
				int(math.Round(g.Max.Y-g.Min.Y)))})
			if t, ok := o.(document.Text); ok {
				text = &t
			}
		}
	}
	bild := s.D.Groesse()
	if zeiger != nil {
		if ort, ok := s.ZeigerOrt(*zeiger); ok {
			f = append(f, FussFeld{Art: FeldZeiger, Ort: ort})
		}
	}
	f = append(f, FussFeld{Art: FeldFarbe, Farbe: s.Stil.Farbe, Klickbar: true})
	switch {
	case text != nil:
		f = append(f, FussFeld{Art: FeldSchrift, Grad: int(math.Round(text.Schriftgrad))})
	case s.Werkzeug == WerkzeugText:
		f = append(f, FussFeld{Art: FeldSchrift, Grad: int(math.Round(s.Stil.Schriftgrad))})
	}
	f = append(f,
		FussFeld{Art: FeldBild, Groesse: bild, Gestaucht: s.KannZurueckholen()},
		FussFeld{Art: FeldZoom, Zoom: int(math.Round(s.Abb.Zoom * 100)), Klickbar: true},
		FussFeld{Art: FeldStand, Ungesichert: s.Ungesichert()},
	)
	return f
}

// Deckkraft ist der Alphawert einer Farbe in Prozent (gerundet).
func Deckkraft(c color.NRGBA) int { return int(math.Round(float64(c.A) * 100 / 255)) }
