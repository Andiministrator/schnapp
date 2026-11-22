// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"fmt"
	"image"
)

// Angaben für die Rettungskopie (ADR-0029): Was eine Bearbeitung ohne Pixel ausmacht, lässt sich auslesen (Objekte,
// Zuschnitt, Streifen, Schwärzungen im Verlauf) und in einem Rückgängig-Schritt wieder setzen.

// SchwaerzAngabe ist eine Schwärzung oder Verpixelung als Angabe: Bereich im Original und Art. Auf demselben
// Ausgangsbild in derselben Reihenfolge wiederholt, ergibt sie dieselbe Unkenntlichmachung (beim Mosaik mit neuen
// Zufallswerten, ADR-0029 Folgen).
type SchwaerzAngabe struct {
	Bereich image.Rectangle
	Art     Schwaerzung
}

// SchwaerzungenImVerlauf liefert die wirksamen Schwärzungen, die im Rückgängig-Verlauf stehen, in ihrer Reihenfolge.
// Nach einem Speichern mit Schwärzung leert die Oberfläche den Verlauf (canvas.Sitzung, ADR-0022 Punkt 3); die Liste
// nennt also genau die Schwärzungen seit dem letzten Speichern. vollstaendig ist false, wenn die Grenze des Verlaufs
// (verlaufGrenze) seitdem eine Schwärzung verworfen hat: Dann fehlt sie in der Liste.
func (d *Dokument) SchwaerzungenImVerlauf() (liste []SchwaerzAngabe, vollstaendig bool) {
	for _, s := range d.verlauf.erledigt {
		if b, ok := s.b.(*schwaerzen); ok {
			liste = append(liste, SchwaerzAngabe{Bereich: b.r, Art: b.s})
		}
	}
	return liste, !d.schwaerzVerworfen
}

type angabenSetzen struct {
	altO, neuO   []Objekt
	altZ, neuZ   image.Rectangle
	altSt, neuSt []Streifen
}

func (b angabenSetzen) ausfuehren(d *Dokument) {
	d.objekte = append([]Objekt(nil), b.neuO...)
	d.setzeAnsicht(b.neuZ, b.neuSt)
}

func (b angabenSetzen) zuruecknehmen(d *Dokument) {
	d.objekte = append([]Objekt(nil), b.altO...)
	d.setzeAnsicht(b.altZ, b.altSt)
}

// SetzeAngaben ersetzt Objekte, Zuschnitt und Streifen in einem einzigen Rückgängig-Schritt (Wiederherstellen einer
// Rettungskopie, ADR-0029 Punkt 6). Die Objekte bekommen neue Kennungen; es gelten dieselben Grenzen wie bei
// Hinzufuegen, SetzeZuschnitt und SetzeStreifen.
func (d *Dokument) SetzeAngaben(objekte []Objekt, z image.Rectangle, st []Streifen) error {
	if d.zug != nil {
		return ErrZugLaeuft
	}
	if len(objekte) > MaxObjekte {
		return fmt.Errorf("document: mehr als %d Objekte", MaxObjekte)
	}
	z, st = z.Canon(), sortiert(st)
	if err := d.pruefeAnsicht(z, st); err != nil {
		return err
	}
	neu := make([]Objekt, 0, len(objekte))
	id := d.naechsteID
	for _, o := range objekte {
		if o == nil {
			return ErrUnbekannt
		}
		if err := o.pruefe(); err != nil {
			return err
		}
		neu = append(neu, o.mitKennung(id))
		id++
	}
	d.naechsteID = id
	d.fuehreAus(angabenSetzen{
		altO: append([]Objekt(nil), d.objekte...), neuO: neu,
		altZ: d.zuschnitt, neuZ: z,
		altSt: append([]Streifen(nil), d.streifen...), neuSt: st,
	})
	return nil
}
