// SPDX-License-Identifier: GPL-3.0-or-later

// Package icons ist das Symbolset der Oberfläche (Karte #188, ADR-0031): eigene Zeichen auf einem Raster von 24 × 24
// Einheiten, Strich 2, eckige Enden und Ecken (Karte #186), keine Fremd-Icons (ADR-0011, ADR-0012). Jedes Symbol ist
// eine Liste einfacher, gefüllter Vielecke; Linien werden zu Vierecken mit eckigen Enden. So zeichnet die Oberfläche
// sie scharf in jeder Skalierung (gefüllte Pfade, kein Strich-Rasterer), und dieselben Daten ergeben das SVG für Doku
// und Muster. Das Paket kennt weder Gio noch Farben: Teile tragen nur „Vordergrund“ oder „Akzent“.
package icons

import (
	"fmt"
	"math"
	"strings"
)

// Raster ist die Kantenlänge eines Symbols in Einheiten.
const Raster = 24

// Strich ist die Strichbreite in Einheiten.
const Strich = 2

// Punkt ist ein Punkt im Raster.
type Punkt struct{ X, Y float64 }

// Teil ist ein gefülltes Vieleck; Akzent heißt: in der Akzentfarbe (Orange) statt im Vordergrund.
type Teil struct {
	Punkte []Punkt
	Akzent bool
}

// Symbol ist ein benanntes Zeichen.
type Symbol struct {
	Name  string // englisch, stabil (Dateiname, Schlüssel)
	Teile []Teil
}

// Gruppe ordnet die Symbole für das Musterblatt und die Doku.
type Gruppe struct {
	Name    string
	Symbole []Symbol
}

// --- Bausteine ---

// linie ist ein Strich von a nach b mit eckigen Enden (um Strich/2 über die Endpunkte hinaus verlängert).
func linie(x1, y1, x2, y2 float64) Teil {
	dx, dy := x2-x1, y2-y1
	l := math.Hypot(dx, dy)
	if l == 0 {
		return flaeche(x1-Strich/2, y1-Strich/2, Strich, Strich)
	}
	ux, uy := dx/l*Strich/2, dy/l*Strich/2 // halbe Breite entlang
	nx, ny := -uy, ux                      // halbe Breite quer
	a := Punkt{x1 - ux + nx, y1 - uy + ny}
	b := Punkt{x2 + ux + nx, y2 + uy + ny}
	c := Punkt{x2 + ux - nx, y2 + uy - ny}
	d := Punkt{x1 - ux - nx, y1 - uy - ny}
	return Teil{Punkte: []Punkt{a, b, c, d}}
}

// zug ist ein offener Linienzug aus Strichen.
func zug(p ...float64) []Teil {
	var t []Teil
	for i := 0; i+3 < len(p); i += 2 {
		t = append(t, linie(p[i], p[i+1], p[i+2], p[i+3]))
	}
	return t
}

// ring ist ein geschlossener Linienzug.
func ring(p ...float64) []Teil {
	return zug(append(p, p[0], p[1])...)
}

// rahmen ist ein Rechteck aus Strichen; x, y, b, h ist das äußere Maß.
func rahmen(x, y, b, h float64) []Teil {
	s := float64(Strich)
	return []Teil{
		flaeche(x, y, b, s), flaeche(x, y+h-s, b, s),
		flaeche(x, y+s, s, h-2*s), flaeche(x+b-s, y+s, s, h-2*s),
	}
}

// flaeche ist ein gefülltes Rechteck.
func flaeche(x, y, b, h float64) Teil {
	return Teil{Punkte: []Punkt{{x, y}, {x + b, y}, {x + b, y + h}, {x, y + h}}}
}

// vieleck ist ein gefülltes Vieleck.
func vieleck(p ...float64) Teil {
	var pt []Punkt
	for i := 0; i+1 < len(p); i += 2 {
		pt = append(pt, Punkt{p[i], p[i+1]})
	}
	return Teil{Punkte: pt}
}

func akzent(t ...Teil) []Teil {
	for i := range t {
		t[i].Akzent = true
	}
	return t
}

func teile(gruppen ...any) []Teil {
	var aus []Teil
	for _, g := range gruppen {
		switch v := g.(type) {
		case Teil:
			aus = append(aus, v)
		case []Teil:
			aus = append(aus, v...)
		default:
			panic(fmt.Sprintf("icons: unbekannter Baustein %T", g))
		}
	}
	return aus
}

// --- Die Symbole ---

// Werkzeuge in der Reihenfolge der Werkzeugleiste (ADR-0031 Punkt 2).
var (
	Auswahl = Symbol{"select", teile(vieleck(6, 3, 6, 19, 10, 15.5, 12.5, 21, 15.5, 19.6, 13, 14, 18.5, 14))}
	Pfeil   = Symbol{"arrow", teile(linie(5, 19, 15.5, 8.5), vieleck(20, 4, 20, 13, 11, 4))}
	Kasten  = Symbol{"rectangle", rahmen(3, 6, 18, 12)}
	Text    = Symbol{"text", teile(flaeche(5, 4, 14, 3), flaeche(10.5, 4, 3, 16), flaeche(8, 18, 8, 2))}
	Nummer  = Symbol{"step", teile(rahmen(3, 3, 18, 18), flaeche(11, 7, 2.5, 10), flaeche(8.5, 7, 3, 2.5),
		flaeche(8.5, 15, 7.5, 2))}
	Schwaerzen = Symbol{"redact", teile(rahmen(2, 5, 20, 14), flaeche(6, 10, 12, 4))}
	Verpixeln  = Symbol{"pixelate", teile(
		flaeche(4, 4, 4, 4), flaeche(12, 4, 4, 4), flaeche(8, 8, 4, 4), flaeche(16, 8, 4, 4),
		flaeche(4, 12, 4, 4), flaeche(12, 12, 4, 4), flaeche(8, 16, 4, 4), flaeche(16, 16, 4, 4))}
	Zuschneiden = Symbol{"crop", teile(zug(7, 2, 7, 17, 22, 17), zug(2, 7, 17, 7, 17, 22))}
	Stauchen    = Symbol{"cut-out", teile(flaeche(4, 3, 16, 5), flaeche(4, 16, 16, 5),
		akzent(zug(3, 12, 7, 10, 11, 12, 15, 10, 19, 12, 21, 11)...))}
	// LupeWerkzeug: kleine Quelle links unten, Verbindungslinie, großes Ziel rechts oben mit vergrößertem Inhalt
	// (ADR-0033); „Lupe“ ist schon das Zoom-Symbol der Menüzeile.
	LupeWerkzeug = Symbol{"magnifier", teile(rahmen(2, 15, 7, 7), linie(9, 15, 12, 12), rahmen(12, 2, 10, 10),
		akzent(flaeche(15, 5, 4, 4)))}
)

// Aktionen der Menüzeile und der Fußzeile.
var (
	Aufnehmen = Symbol{"capture", teile(zug(3, 8, 3, 3, 8, 3), zug(16, 3, 21, 3, 21, 8), zug(21, 16, 21, 21, 16, 21),
		zug(8, 21, 3, 21, 3, 16), akzent(flaeche(10, 10, 4, 4)))}
	Oeffnen     = Symbol{"open", ring(3, 5, 9, 5, 11, 7.5, 21, 7.5, 21, 19, 3, 19)}
	Speichern   = Symbol{"save", teile(rahmen(3, 3, 18, 18), flaeche(7, 3, 9, 6), rahmen(7, 13, 10, 8))}
	Exportieren = Symbol{"export", teile(zug(11, 4, 4, 4, 4, 20, 20, 20, 20, 13), linie(11, 13, 18, 6),
		vieleck(21, 3, 21, 11, 13, 3))}
	Kopieren  = Symbol{"copy", teile(rahmen(8, 8, 13, 13), zug(16, 4, 4, 4, 4, 16))}
	Rueck     = Symbol{"undo", teile(zug(9, 8, 18, 8, 18, 18, 10, 18), vieleck(3, 8, 10, 2.5, 10, 13.5))}
	Wieder    = Symbol{"redo", teile(zug(15, 8, 6, 8, 6, 18, 14, 18), vieleck(21, 8, 14, 2.5, 14, 13.5))}
	Einpassen = Symbol{"fit", teile(zug(3, 8, 3, 3, 8, 3), zug(16, 3, 21, 3, 21, 8), zug(21, 16, 21, 21, 16, 21),
		zug(8, 21, 3, 21, 3, 16), rahmen(8, 8, 8, 8))}
	EinsZuEins = Symbol{"actual-size", teile(flaeche(4, 6, 2.5, 12), flaeche(2.5, 6, 2, 2.5),
		flaeche(17.5, 6, 2.5, 12), flaeche(16, 6, 2, 2.5), flaeche(10.75, 9, 2.5, 2.5), flaeche(10.75, 14, 2.5, 2.5))}
	Lupe       = Symbol{"zoom", teile(rahmen(3, 3, 13, 13), linie(16, 16, 21, 21))}
	GanzesBild = Symbol{"whole-image", teile(rahmen(2, 4, 20, 16), vieleck(5, 17, 10, 10, 13, 14, 15, 12, 19, 17),
		akzent(flaeche(15, 7, 3, 3)))}
	Zurueckholen = Symbol{"restore-cut-out", teile(flaeche(4, 10.5, 16, 3), vieleck(12, 2, 7, 7, 17, 7),
		vieleck(12, 22, 7, 17, 17, 17))}
	Schliessen = Symbol{"close", teile(linie(6, 6, 18, 18), linie(18, 6, 6, 18))}
)

// Eigenschaften der rechten Leiste.
var (
	Farbe          = Symbol{"color", teile(rahmen(3, 3, 18, 18), akzent(flaeche(7, 7, 10, 10)))}
	Strichstaerke  = Symbol{"stroke-width", teile(flaeche(3, 4, 18, 1), flaeche(3, 9, 18, 2), flaeche(3, 15, 18, 4))}
	Schriftgroesse = Symbol{"font-size", teile(flaeche(2, 10, 8, 2), flaeche(5, 10, 2, 9), flaeche(11, 4, 11, 3),
		flaeche(15, 4, 3, 16))}
	KanteAbriss = Symbol{"seam-tear", teile(vieleck(3, 3, 21, 3, 21, 12, 18.5, 14, 16, 11, 13, 15, 10, 11.5, 7.5, 14,
		5, 11, 3, 13))}
	KanteWelle = Symbol{"seam-wave", teile(vieleck(3, 3, 21, 3, 21, 12, 19, 13.5, 17, 14, 15, 13.5, 13, 12, 11, 10.5,
		9, 10, 7, 10.5, 5, 12, 3, 13.5))}
	KanteZacken = Symbol{"seam-zigzag", teile(vieleck(3, 3, 21, 3, 21, 14, 19, 11, 17, 14, 15, 11, 13, 14, 11, 11,
		9, 14, 7, 11, 5, 14, 3, 11))}
)

// Ebenen.
var (
	Sichtbar  = Symbol{"visible", teile(ring(2, 12, 8, 6, 16, 6, 22, 12, 16, 18, 8, 18), akzent(flaeche(10, 10, 4, 4)))}
	Verborgen = Symbol{"hidden", teile(ring(2, 12, 8, 6, 16, 6, 22, 12, 16, 18, 8, 18), linie(4, 21, 20, 3))}
	Hoch      = Symbol{"move-up", teile(linie(12, 21, 12, 9), vieleck(12, 3, 5, 10, 19, 10))}
	Runter    = Symbol{"move-down", teile(linie(12, 3, 12, 15), vieleck(12, 21, 5, 14, 19, 14))}
	Loeschen  = Symbol{"delete", teile(linie(3, 6, 21, 6), flaeche(9, 2, 6, 2), rahmen(5, 8, 14, 14), flaeche(9, 11, 2, 8),
		flaeche(13, 11, 2, 8))}
	Sperren = Symbol{"lock", teile(flaeche(4, 11, 16, 10), zug(8, 11, 8, 4, 16, 4, 16, 11))}
	Doppeln = Symbol{"duplicate", teile(rahmen(8, 8, 13, 13), zug(16, 4, 4, 4, 4, 16), flaeche(13.5, 11, 2, 7),
		flaeche(11, 13.5, 7, 2))}
)

// Status der Fußzeile.
var (
	Gespeichert = Symbol{"saved", zug(4, 12, 10, 18, 20, 6)}
	Ungesichert = Symbol{"unsaved", akzent(flaeche(8, 8, 8, 8))}
	Warnung     = Symbol{"warning", teile(ring(12, 3, 22, 21, 2, 21), akzent(flaeche(11, 9, 2, 6), flaeche(11, 16.5, 2, 2)))}
	Fehler      = Symbol{"error", teile(rahmen(3, 3, 18, 18), linie(8, 8, 16, 16), linie(16, 8, 8, 16))}
)

// Weitere.
var (
	Bibliothek    = Symbol{"library", teile(rahmen(3, 3, 8, 8), rahmen(13, 3, 8, 8), rahmen(3, 13, 8, 8), rahmen(13, 13, 8, 8))}
	Einstellungen = Symbol{"settings", teile(linie(3, 6, 21, 6), flaeche(13, 3, 4, 6), linie(3, 12, 21, 12),
		flaeche(6, 9, 4, 6), linie(3, 18, 21, 18), flaeche(11, 15, 4, 6))}
)

// Gruppen liefert alle Symbole in Gruppen (Musterblatt, Doku, Test).
func Gruppen() []Gruppe {
	return []Gruppe{
		{"tools", []Symbol{Auswahl, Pfeil, Kasten, Text, Nummer, LupeWerkzeug, Schwaerzen, Verpixeln, Zuschneiden, Stauchen}},
		{"actions", []Symbol{Aufnehmen, Oeffnen, Speichern, Exportieren, Kopieren, Rueck, Wieder, Einpassen, EinsZuEins,
			Lupe, GanzesBild, Zurueckholen, Schliessen}},
		{"properties", []Symbol{Farbe, Strichstaerke, Schriftgroesse, KanteAbriss, KanteWelle, KanteZacken}},
		{"layers", []Symbol{Sichtbar, Verborgen, Hoch, Runter, Doppeln, Loeschen, Sperren}},
		{"status", []Symbol{Gespeichert, Ungesichert, Warnung, Fehler}},
		{"other", []Symbol{Bibliothek, Einstellungen}},
	}
}

// SVG liefert das Symbol als eigenständiges SVG (viewBox 0 0 24 24) in den Farben vorn und akzent (CSS-Farben).
func (s Symbol) SVG(vorn, akz string) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d">`, Raster, Raster)
	b.WriteString(s.SVGPfade(vorn, akz))
	b.WriteString("</svg>")
	return b.String()
}

// SVGPfade liefert nur die Pfade (zum Einbetten mit eigener Verschiebung).
func (s Symbol) SVGPfade(vorn, akz string) string {
	var b strings.Builder
	for _, t := range s.Teile {
		f := vorn
		if t.Akzent {
			f = akz
		}
		b.WriteString(`<path fill="` + f + `" d="M`)
		for i, p := range t.Punkte {
			if i > 0 {
				b.WriteString("L")
			}
			b.WriteString(zahl(p.X) + " " + zahl(p.Y))
		}
		b.WriteString(`Z"/>`)
	}
	return b.String()
}

func zahl(f float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.3f", f), "0"), ".")
}
