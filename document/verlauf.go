// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"fmt"
	"image"
)

// befehl ist eine Änderung, die sich zurücknehmen lässt.
type befehl interface {
	ausfuehren(d *Dokument)
	zuruecknehmen(d *Dokument)
}

type verlauf struct {
	erledigt, zurueckgenommen []befehl
}

// fuehreAus führt b aus, legt ihn in den Verlauf und verwirft die zurückgenommenen Befehle.
func (d *Dokument) fuehreAus(b befehl) {
	b.ausfuehren(d)
	d.verlauf.erledigt = append(d.verlauf.erledigt, b)
	d.verlauf.zurueckgenommen = nil
}

// KannRueckgaengig meldet, ob Rückgängig etwas tut (während eines Zugs nie).
func (d *Dokument) KannRueckgaengig() bool { return d.zug == nil && len(d.verlauf.erledigt) > 0 }

// KannWiederholen meldet, ob Wiederholen etwas tut (während eines Zugs nie).
func (d *Dokument) KannWiederholen() bool { return d.zug == nil && len(d.verlauf.zurueckgenommen) > 0 }

// Rueckgaengig nimmt die letzte Änderung zurück. Während eines Zugs: ErrZugLaeuft (F-18).
func (d *Dokument) Rueckgaengig() error {
	if d.zug != nil {
		return ErrZugLaeuft
	}
	n := len(d.verlauf.erledigt)
	if n == 0 {
		return ErrNichtsDa
	}
	b := d.verlauf.erledigt[n-1]
	d.verlauf.erledigt = d.verlauf.erledigt[:n-1]
	b.zuruecknehmen(d)
	d.verlauf.zurueckgenommen = append(d.verlauf.zurueckgenommen, b)
	return nil
}

// Wiederholen führt die zuletzt zurückgenommene Änderung erneut aus.
func (d *Dokument) Wiederholen() error {
	if d.zug != nil {
		return ErrZugLaeuft
	}
	n := len(d.verlauf.zurueckgenommen)
	if n == 0 {
		return ErrNichtsDa
	}
	b := d.verlauf.zurueckgenommen[n-1]
	d.verlauf.zurueckgenommen = d.verlauf.zurueckgenommen[:n-1]
	b.ausfuehren(d)
	d.verlauf.erledigt = append(d.verlauf.erledigt, b)
	return nil
}

// VergissVerlauf leert den Verlauf, etwa nach dem Speichern, damit auch im Speicher keine geschwärzten Pixel mehr
// zurückzuholen sind.
func (d *Dokument) VergissVerlauf() error {
	if d.zug != nil {
		return ErrZugLaeuft
	}
	d.verlauf = verlauf{}
	return nil
}

// --- Objekte ---

type einfuegen struct {
	o      Objekt
	stelle int
}

func (b einfuegen) ausfuehren(d *Dokument) {
	d.objekte = append(d.objekte[:b.stelle], append([]Objekt{b.o}, d.objekte[b.stelle:]...)...)
}
func (b einfuegen) zuruecknehmen(d *Dokument) {
	d.objekte = append(d.objekte[:b.stelle], d.objekte[b.stelle+1:]...)
}

type entfernen struct {
	o      Objekt
	stelle int
}

func (b entfernen) ausfuehren(d *Dokument)    { einfuegen(b).zuruecknehmen(d) }
func (b entfernen) zuruecknehmen(d *Dokument) { einfuegen(b).ausfuehren(d) }

type ersetzen struct{ alt, neu Objekt }

func (b ersetzen) ausfuehren(d *Dokument)    { d.objekte[d.index(b.neu.Kennung())] = b.neu }
func (b ersetzen) zuruecknehmen(d *Dokument) { d.objekte[d.index(b.alt.Kennung())] = b.alt }

// Hinzufuegen legt o zuoberst ab und liefert seine Kennung.
func (d *Dokument) Hinzufuegen(o Objekt) (int, error) {
	if o == nil {
		return 0, ErrUnbekannt
	}
	if d.zug != nil {
		return 0, ErrZugLaeuft
	}
	if len(d.objekte) >= MaxObjekte {
		return 0, fmt.Errorf("dokument: mehr als %d Objekte", MaxObjekte)
	}
	if err := o.pruefe(); err != nil {
		return 0, err
	}
	o = o.mitKennung(d.naechsteID)
	d.naechsteID++
	d.fuehreAus(einfuegen{o, len(d.objekte)})
	return o.Kennung(), nil
}

// Entfernen nimmt das Objekt mit der Kennung id heraus.
func (d *Dokument) Entfernen(id int) error {
	if d.zug != nil {
		return ErrZugLaeuft
	}
	i := d.index(id)
	if i < 0 {
		return ErrUnbekannt
	}
	d.fuehreAus(entfernen{d.objekte[i], i})
	return nil
}

// Ersetzen setzt neu an die Stelle des Objekts mit derselben Kennung (Eigenschaften ändern).
func (d *Dokument) Ersetzen(neu Objekt) error {
	if neu == nil {
		return ErrUnbekannt
	}
	if d.zug != nil {
		return ErrZugLaeuft
	}
	i := d.index(neu.Kennung())
	if neu.Kennung() == 0 || i < 0 {
		return ErrUnbekannt
	}
	if err := neu.pruefe(); err != nil {
		return err
	}
	d.fuehreAus(ersetzen{d.objekte[i], neu})
	return nil
}

// --- Zug: Ziehen mit der Maus als eine einzige Änderung ---

// zug ist eine laufende Änderung mit der Maus. Sie wirkt sofort auf das Dokument (für die Anzeige), landet aber erst
// mit BeendeZug als ein Befehl im Verlauf. Solange sie läuft, sind Rückgängig und alle anderen Änderungen gesperrt —
// sonst zeigte der Zug auf ein Objekt, das es nicht mehr gibt (F-18).
type zug struct {
	alt Objekt // Stand vor dem Zug; nil bei einem neuen Objekt
	id  int
}

// BeginneNeuenZug fügt o für die Dauer des Zugs ein (z. B. einen Pfeil, der gerade aufgezogen wird) und liefert
// seine Kennung.
func (d *Dokument) BeginneNeuenZug(o Objekt) (int, error) {
	if o == nil {
		return 0, ErrUnbekannt
	}
	if d.zug != nil {
		return 0, ErrZugLaeuft
	}
	if len(d.objekte) >= MaxObjekte {
		return 0, fmt.Errorf("dokument: mehr als %d Objekte", MaxObjekte)
	}
	if err := o.pruefe(); err != nil {
		return 0, err
	}
	o = o.mitKennung(d.naechsteID)
	d.naechsteID++
	d.objekte = append(d.objekte, o)
	d.zug = &zug{id: o.Kennung()}
	return o.Kennung(), nil
}

// BeginneZug startet einen Zug an einem vorhandenen Objekt (verschieben, Griff ziehen).
func (d *Dokument) BeginneZug(id int) error {
	if d.zug != nil {
		return ErrZugLaeuft
	}
	i := d.index(id)
	if i < 0 {
		return ErrUnbekannt
	}
	d.zug = &zug{alt: d.objekte[i], id: id}
	return nil
}

// ZugSetze setzt den aktuellen Stand des gezogenen Objekts (gleiche Kennung).
func (d *Dokument) ZugSetze(o Objekt) error {
	if o == nil {
		return ErrUnbekannt
	}
	if d.zug == nil {
		return ErrKeinZug
	}
	if o.Kennung() != d.zug.id {
		return fmt.Errorf("dokument: Zug gilt Objekt %d, nicht %d", d.zug.id, o.Kennung())
	}
	if err := o.pruefe(); err != nil {
		return err
	}
	d.objekte[d.index(d.zug.id)] = o
	return nil
}

// BeendeZug übernimmt den Zug als eine Änderung in den Verlauf. Ein unveränderter Zug an einem vorhandenen Objekt
// hinterlässt nichts.
func (d *Dokument) BeendeZug() error {
	if d.zug == nil {
		return ErrKeinZug
	}
	z := d.zug
	d.zug = nil
	i := d.index(z.id)
	neu := d.objekte[i]
	if z.alt == nil {
		// Für den Verlauf herausnehmen und regulär einfügen: Rückgängig entfernt dann genau dieses Objekt.
		d.objekte = append(d.objekte[:i], d.objekte[i+1:]...)
		d.fuehreAus(einfuegen{neu, i})
		return nil
	}
	if neu != z.alt {
		d.objekte[i] = z.alt
		d.fuehreAus(ersetzen{z.alt, neu})
	}
	return nil
}

// BrecheZugAb stellt den Stand vor dem Zug wieder her (Esc).
func (d *Dokument) BrecheZugAb() error {
	if d.zug == nil {
		return ErrKeinZug
	}
	z := d.zug
	d.zug = nil
	i := d.index(z.id)
	if z.alt == nil {
		d.objekte = append(d.objekte[:i], d.objekte[i+1:]...)
	} else {
		d.objekte[i] = z.alt
	}
	return nil
}

// ZugLaeuft meldet, ob gerade gezogen wird.
func (d *Dokument) ZugLaeuft() bool { return d.zug != nil }

// --- Zuschnitt, Stauchen, Schwärzen ---

type ansichtAendern struct {
	altZ, neuZ   image.Rectangle
	altSt, neuSt []Streifen
}

func (b ansichtAendern) ausfuehren(d *Dokument)    { d.setzeAnsicht(b.neuZ, b.neuSt) }
func (b ansichtAendern) zuruecknehmen(d *Dokument) { d.setzeAnsicht(b.altZ, b.altSt) }

// SetzeZuschnitt setzt den sichtbaren Ausschnitt (im Original); das ganze Bild hebt den Zuschnitt auf.
func (d *Dokument) SetzeZuschnitt(z image.Rectangle) error {
	if d.zug != nil {
		return ErrZugLaeuft
	}
	z = z.Canon()
	if err := d.pruefeAnsicht(z, d.streifen); err != nil {
		return err
	}
	d.fuehreAus(ansichtAendern{d.zuschnitt, z, d.streifen, d.streifen})
	return nil
}

// SetzeStreifen ersetzt alle Stauch-Streifen; sie dürfen sich je Achse nicht überlappen.
func (d *Dokument) SetzeStreifen(st []Streifen) error {
	if d.zug != nil {
		return ErrZugLaeuft
	}
	st = sortiert(st)
	if err := d.pruefeAnsicht(d.zuschnitt, st); err != nil {
		return err
	}
	d.fuehreAus(ansichtAendern{d.zuschnitt, d.zuschnitt, d.streifen, st})
	return nil
}

type schwaerzen struct {
	r      image.Rectangle
	s      Schwaerzung
	vorher *image.NRGBA // alte Pixel, nur im Speicher
}

func (b *schwaerzen) ausfuehren(d *Dokument) {
	if b.vorher == nil {
		b.vorher = image.NewNRGBA(b.r)
	}
	for y := b.r.Min.Y; y < b.r.Max.Y; y++ {
		q := d.original.PixOffset(b.r.Min.X, y)
		z := b.vorher.PixOffset(b.r.Min.X, y)
		copy(b.vorher.Pix[z:z+4*b.r.Dx()], d.original.Pix[q:q+4*b.r.Dx()])
	}
	d.schwaerze(b.r, b.s)
}

func (b *schwaerzen) zuruecknehmen(d *Dokument) {
	for y := b.r.Min.Y; y < b.r.Max.Y; y++ {
		q := b.vorher.PixOffset(b.r.Min.X, y)
		z := d.original.PixOffset(b.r.Min.X, y)
		copy(d.original.Pix[z:z+4*b.r.Dx()], b.vorher.Pix[q:q+4*b.r.Dx()])
	}
}

// Schwaerzen macht den Bereich r (im Original, wird aufs Bild begrenzt) unkenntlich, indem es die Pixel des
// Originals überschreibt (Kernprinzip 2). Verpixeln verlangt Blöcke ab MinBlock Pixeln.
func (d *Dokument) Schwaerzen(r image.Rectangle, s Schwaerzung) error {
	if d.zug != nil {
		return ErrZugLaeuft
	}
	if s.Block > 0 && s.Block < MinBlock {
		return fmt.Errorf("dokument: Blockgröße %d kleiner als %d", s.Block, MinBlock)
	}
	if s.Block <= 0 && s.Farbe.A != 255 {
		// Eine durchsichtige Füllung (etwa der Nullwert) hinterließe ein Loch statt einer Schwärzung.
		return fmt.Errorf("dokument: Schwärzungsfarbe muss deckend sein")
	}
	r = r.Canon().Intersect(d.original.Rect)
	if r.Empty() {
		return fmt.Errorf("dokument: Bereich zum Schwärzen liegt außerhalb des Bilds")
	}
	if s.Block > 0 && (r.Dx() < MinBlock || r.Dy() < MinBlock) {
		// Ein schmaler Bereich lässt sich nicht wirksam verpixeln (ein Block über 1 Pixel ändert nichts).
		return fmt.Errorf("dokument: Bereich zum Verpixeln kleiner als %d Pixel — schwärzen statt verpixeln", MinBlock)
	}
	d.fuehreAus(&schwaerzen{r: r, s: s})
	return nil
}
