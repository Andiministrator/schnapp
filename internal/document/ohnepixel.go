// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"image"
	"slices"
)

// OhnePixel legt ein Dokument ohne Pixel an: Größe, Zuschnitt, Streifen und Kantenform wie angegeben, ohne Objekte.
// Es dient der Abbildung zwischen Original und Ansicht und dem Schreiben der Angaben (fileformat.AngabenSVG), etwa
// für Objekte in der Zwischenablage (ADR-0045 Punkt 2). Gerendert werden darf es nicht: Das Original hat keine Pixel.
// Die Angaben werden geprüft wie beim Laden (PruefeInhalt).
func OhnePixel(groesse image.Point, zuschnitt image.Rectangle, streifen []Streifen, k Kante) (*Dokument, error) {
	if err := pruefeGroesse(image.Rectangle{Max: groesse}); err != nil {
		return nil, err
	}
	r := image.Rectangle{Max: groesse}
	d := &Dokument{original: &image.NRGBA{Rect: r}, naechsteID: 1, kante: KanteAbriss}
	d.setzeAnsicht(r, nil)
	if zuschnitt != d.Zuschnitt() {
		if err := d.SetzeZuschnitt(zuschnitt); err != nil {
			return nil, err
		}
	}
	if len(streifen) > 0 {
		if err := d.SetzeStreifen(streifen); err != nil {
			return nil, err
		}
	}
	if k != d.kante {
		if err := d.SetzeKante(k); err != nil {
			return nil, err
		}
	}
	d.verlauf = verlauf{} // die Schritte oben gehören nicht zum Inhalt
	return d, nil
}

// NurObjekte liefert ein Dokument ohne Pixel (OhnePixel) mit Größe, Zuschnitt, Streifen und Kantenform von d und nur
// den Objekten mit den Kennungen ids, in der Stapelreihenfolge von d (ADR-0045 Punkt 2). Unbekannte Kennungen fallen
// weg; die Objekte bekommen im neuen Dokument neue Kennungen. Ein laufender Zug zählt nicht (ObjekteOhneZug).
func (d *Dokument) NurObjekte(ids []int) (*Dokument, error) {
	n, err := OhnePixel(d.OriginalGroesse(), d.Zuschnitt(), d.Streifen(), d.Kante())
	if err != nil {
		return nil, err
	}
	for _, o := range d.ObjekteOhneZug() {
		if !slices.Contains(ids, o.Kennung()) {
			continue
		}
		if _, err := n.Hinzufuegen(o); err != nil {
			return nil, err
		}
	}
	n.verlauf = verlauf{}
	return n, nil
}

// GleichOhneKennung meldet, ob a und b bis auf die Kennung gleich sind (Gleich): Einfügen erkennt so dieselben
// Objekte an derselben Stelle (ADR-0045 Punkt 3).
func GleichOhneKennung(a, b Objekt) bool {
	if a == nil || b == nil {
		return a == b
	}
	return Gleich(a.mitKennung(0), b.mitKennung(0))
}

// OhneSchalter liefert o ohne Ausgeblendet und Gesperrt (Einfügen, ADR-0045 Punkt 3); Name, Deckkraft, Schatten und
// Gruppe bleiben.
func OhneSchalter(o Objekt) Objekt {
	e := o.Ebenenwerte()
	e.Ausgeblendet, e.Gesperrt = false, false
	return o.mitEbene(e)
}
