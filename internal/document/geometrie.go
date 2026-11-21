// SPDX-License-Identifier: GPL-3.0-or-later

// Package document hält eine Aufnahme mit ihren Beschriftungen: das unbeschnittene Original als einzige Pixelquelle,
// Vektorobjekte darüber, Zuschnitt und Stauchen als Angaben und den Rückgängig-Verlauf (ADR-0008, Kernprinzip 1).
//
// Koordinaten: Alle Objekte, der Zuschnitt und die Streifen liegen in Pixeln des Originals. Die Ansicht (was nach
// Zuschnitt und Stauchen sichtbar ist) hat eigene Koordinaten; ZuAnsicht und ZuOriginal rechnen um. So bleiben
// Beschriftungen beim Zuschneiden am Inhalt.
//
// Schwärzen ist die einzige zerstörende Änderung: Sie überschreibt die Pixel des Originals (Kernprinzip 2). Der
// Rückgängig-Verlauf hält die alten Pixel nur im Speicher; er wird nie gespeichert.
package document

import "math"

// Punkt ist ein Punkt in Pixeln (Original oder Ansicht, je nach Zusammenhang).
type Punkt struct{ X, Y float64 }

// Pt baut einen Punkt.
func Pt(x, y float64) Punkt { return Punkt{x, y} }

// Plus addiert.
func (p Punkt) Plus(q Punkt) Punkt { return Punkt{p.X + q.X, p.Y + q.Y} }

// Minus subtrahiert.
func (p Punkt) Minus(q Punkt) Punkt { return Punkt{p.X - q.X, p.Y - q.Y} }

// Mal skaliert.
func (p Punkt) Mal(f float64) Punkt { return Punkt{p.X * f, p.Y * f} }

// Laenge ist der Abstand zum Nullpunkt.
func (p Punkt) Laenge() float64 { return math.Hypot(p.X, p.Y) }

// Rechteck ist ein achsparalleles Rechteck mit Min ≤ Max.
type Rechteck struct{ Min, Max Punkt }

// R baut ein Rechteck aus zwei beliebigen Ecken.
func R(a, b Punkt) Rechteck {
	return Rechteck{Punkt{math.Min(a.X, b.X), math.Min(a.Y, b.Y)}, Punkt{math.Max(a.X, b.X), math.Max(a.Y, b.Y)}}
}

// Breite des Rechtecks.
func (r Rechteck) Breite() float64 { return r.Max.X - r.Min.X }

// Hoehe des Rechtecks.
func (r Rechteck) Hoehe() float64 { return r.Max.Y - r.Min.Y }

// Enthaelt meldet, ob p im Rechteck liegt (Ränder eingeschlossen).
func (r Rechteck) Enthaelt(p Punkt) bool {
	return p.X >= r.Min.X && p.X <= r.Max.X && p.Y >= r.Min.Y && p.Y <= r.Max.Y
}

// Vereint liefert das kleinste Rechteck um r und s.
func (r Rechteck) Vereint(s Rechteck) Rechteck {
	return Rechteck{Punkt{math.Min(r.Min.X, s.Min.X), math.Min(r.Min.Y, s.Min.Y)},
		Punkt{math.Max(r.Max.X, s.Max.X), math.Max(r.Max.Y, s.Max.Y)}}
}

// Wertebereiche für Objekte (F-62): Koordinaten bis ±MaxKoordinate, Größen (Breiten, Radien, Schriftgrad) bis
// MaxGroesse — weit über jedem Bild, aber endlich genug, dass Export und Rendern keine Inf erzeugen oder hängen.
const (
	MaxKoordinate = 1e6
	MaxGroesse    = 1e5
	MaxNummer     = 1_000_000
	// MaxSchrift begrenzt Schriftgrad und Nummer-Radius: Der Renderer rastert jede Glyphe vollständig, der Speicher
	// wächst quadratisch mit der Größe — eine präparierte Datei mit 1e5 bräuchte Dutzende GiB (Kritik A-2, [S]).
	MaxSchrift = 4096
	// MaxTextZeichen begrenzt die Zeichen (Runen) eines Textes, MaxObjekte die Objekte eines Dokuments (F-118): Jede
	// Glyphe kostet beim Rastern Millisekunden, eine präparierte Arbeitsdatei mit Millionen Zeichen fröre den
	// Editor beim Speichern ein. Überschreitung ist ein Fehler, kein stilles Kürzen.
	MaxTextZeichen = 10_000
	MaxObjekte     = 10_000
	// MaxStreifen begrenzt die Stauch-Streifen eines Dokuments (F-136). Echte Dokumente haben eine Handvoll; die
	// Grenze liegt weit darüber und hält Prüfen und Rendern auch bei einer präparierten Datei kurz.
	MaxStreifen = 1024
	// MaxKacheln begrenzt (Zeilenstreifen+1)×(Spaltenstreifen+1): Der SVG-Export schreibt je Kachel ein eigenes
	// PNG (F-141); 512+512 Streifen ergäben 263 169 Kacheln und rund 14 s Export. Gezählt wird die Rohliste,
	// auch vom Zuschnitt verdeckte Streifen (strenger, nie lockerer).
	MaxKacheln = 4096
)

// endlich meldet, ob alle Werte endliche Zahlen im Koordinatenbereich sind.
func endlich(werte ...float64) bool {
	for _, w := range werte {
		if math.IsNaN(w) || math.IsInf(w, 0) || math.Abs(w) > MaxKoordinate {
			return false
		}
	}
	return true
}

// groesse meldet, ob alle Werte gültige Größen sind (0 ≤ w ≤ MaxGroesse).
func groesse(werte ...float64) bool {
	for _, w := range werte {
		if !endlich(w) || w < 0 || w > MaxGroesse {
			return false
		}
	}
	return true
}
