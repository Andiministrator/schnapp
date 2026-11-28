// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"fmt"
	"image"
)

// PruefeInhalt prüft Objekte, Zuschnitt und Streifen gegen ein Original der Größe groesse, ohne dass Pixel
// vorliegen, und liefert die Größe der Ansicht. Es nimmt dieselben Wege wie das Laden einer Arbeitsdatei (Neu,
// Hinzufuegen, SetzeZuschnitt, SetzeStreifen), also dieselben Grenzen; wer das Dokument nicht braucht (Index einer
// Bibliothek, fileformat.LeseMetadaten), bekommt so dasselbe Urteil und dieselbe Größe wie beim vollen Lesen
// (F-272). Die Bildgröße unterliegt denselben Grenzen wie Neu.
func PruefeInhalt(groesse image.Point, zuschnitt image.Rectangle, streifen []Streifen, objekte []Objekt) (image.Point, error) {
	if groesse.X < 1 || groesse.Y < 1 || groesse.X > MaxSeite || groesse.Y > MaxSeite || groesse.X*groesse.Y > MaxPixel {
		return image.Point{}, fmt.Errorf("%w: %dx%d", ErrBildGroesse, groesse.X, groesse.Y)
	}
	if n := FreihandPunkteGesamt(objekte); n > MaxFreihandGesamt {
		return image.Point{}, fmt.Errorf("document: %d Freihand-Punkte, mehr als %d", n, MaxFreihandGesamt)
	}
	// Ein Dokument ohne Pixel: die Prüfungen lesen nur die Größe des Originals.
	r := image.Rect(0, 0, groesse.X, groesse.Y)
	d := &Dokument{original: &image.NRGBA{Rect: r}, naechsteID: 1}
	d.setzeAnsicht(r, nil)
	for _, o := range objekte {
		if _, err := d.Hinzufuegen(o); err != nil {
			return image.Point{}, err
		}
	}
	if zuschnitt != d.Zuschnitt() {
		if err := d.SetzeZuschnitt(zuschnitt); err != nil {
			return image.Point{}, err
		}
	}
	if len(streifen) > 0 {
		if err := d.SetzeStreifen(streifen); err != nil {
			return image.Point{}, err
		}
	}
	return d.Groesse(), nil
}
