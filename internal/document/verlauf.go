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

// schritt ist ein Befehl im Verlauf samt der Kennung des Stands, den er herstellt (F-109).
type schritt struct {
	b     befehl
	stand uint64
}

type verlauf struct {
	erledigt, zurueckgenommen []schritt
	// basis ist der Stand, wenn nichts erledigt ist (nach dem Öffnen 0, nach VergissVerlauf der Stand davor).
	basis uint64
	// letzte ist die zuletzt vergebene Kennung; sie zählt nur aufwärts, auch über VergissVerlauf hinweg.
	letzte uint64
}

// fuehreAus führt b aus, legt ihn mit einer neuen Standkennung in den Verlauf und verwirft die zurückgenommenen
// Befehle.
func (d *Dokument) fuehreAus(b befehl) {
	b.ausfuehren(d)
	d.verlauf.letzte++
	d.verlauf.erledigt = append(d.verlauf.erledigt, schritt{b, d.verlauf.letzte})
	d.verlauf.zurueckgenommen = nil
	d.begrenzeVerlauf()
}

// verlaufGrenze ist die höchste Gesamtgröße der im Verlauf gehaltenen Pixelkopien in Bytes (F-64). 512 MiB sind
// rund 15 Schwärzungen über ein ganzes 4K-Bild (3840 x 2160 x 4 = 33 MB) oder einige tausend übliche Bereiche
// (400 x 100 Pixel = 160 KB). Paketvariable, damit Tests mit einer kleinen Grenze arbeiten.
var verlaufGrenze int64 = 512 << 20

// VerlaufGrenze liefert die Grenze des Rückgängig-Verlaufs in Bytes: so viel Platz hat ein frisch geöffnetes Dokument
// (VerlaufPlatz ohne gehaltene Kopien). Die Rettungskopie prüft damit beim Schreiben dasselbe Flächenbudget, das beim
// Anwenden gilt (F-431).
func VerlaufGrenze() int64 { return verlaufGrenze }

// groessenBefehl ist ein Befehl, der Pixelkopien hält.
type groessenBefehl interface{ pixelBytes() int64 }

func (b *schwaerzen) pixelBytes() int64 {
	if b.vorher == nil {
		return 0
	}
	return int64(len(b.vorher.Pix))
}

// begrenzeVerlauf wirft die ältesten erledigten Schritte weg, solange die Pixelkopien im Verlauf die Grenze
// überschreiten. Der jüngste Schritt bleibt immer. Verworfene Schritte lassen sich nicht mehr zurücknehmen; der
// Stand darunter (basis) ist die Kennung des zuletzt verworfenen Schritts, damit Stand() stimmig bleibt.
func (d *Dokument) begrenzeVerlauf() {
	v := &d.verlauf
	var summe int64
	for _, s := range v.erledigt {
		if g, ok := s.b.(groessenBefehl); ok {
			summe += g.pixelBytes()
		}
	}
	weg := 0
	for summe > verlaufGrenze && weg < len(v.erledigt)-1 {
		if g, ok := v.erledigt[weg].b.(groessenBefehl); ok {
			summe -= g.pixelBytes()
		}
		if _, ok := v.erledigt[weg].b.(*schwaerzen); ok {
			d.schwaerzVerworfen = true // SchwaerzungenImVerlauf ist unvollständig (ADR-0029)
		}
		v.basis = v.erledigt[weg].stand
		weg++
	}
	if weg > 0 {
		v.erledigt = append([]schritt(nil), v.erledigt[weg:]...) // neues Feld, damit die Kopien freigegeben werden
	}
}

// VerlaufPlatz liefert, wie viele Bytes Pixelkopien der Rückgängig-Verlauf noch aufnimmt, ohne Schritte zu verwerfen
// (verlaufGrenze minus die gehaltenen Kopien, nie negativ). Eine Schwärzung über n Pixel belegt 4·n Bytes. Wer mehrere
// Schwärzungen am Stück anwendet (Rettungskopie, F-412), prüft damit vorab, dass keine aus dem Verlauf fällt und
// Rückgängig auf den Stand davor zurückführt.
func (d *Dokument) VerlaufPlatz() int64 {
	var summe int64
	for _, s := range d.verlauf.erledigt {
		if g, ok := s.b.(groessenBefehl); ok {
			summe += g.pixelBytes()
		}
	}
	return max(verlaufGrenze-summe, 0)
}

// Speicherbedarf schätzt, wie viel Speicher das Dokument hält: Pixel des Originals plus die Pixelkopien im
// Rückgängig-Verlauf (erledigt und zurückgenommen). Grundlage der Grenze für gemerkte Aufnahmen beim Wechsel im selben
// Fenster (ADR-0036 Zusatz Karte #356); Objekte und Verwaltung zählen nicht, sie sind klein dagegen.
func (d *Dokument) Speicherbedarf() int64 {
	summe := int64(len(d.original.Pix))
	for _, l := range [][]schritt{d.verlauf.erledigt, d.verlauf.zurueckgenommen} {
		for _, s := range l {
			if g, ok := s.b.(groessenBefehl); ok {
				summe += g.pixelBytes()
			}
		}
	}
	return summe
}

// Stand kennzeichnet den Verlaufsstand des Dokuments (F-109): Jede neue Änderung bekommt eine neue Kennung, die nie
// wiederkehrt; Rückgängig und Wiederholen kehren zu früheren Kennungen zurück. Gleiche Kennung heißt gleicher Inhalt
// — so erkennt der Editor, ob der gespeicherte Stand wieder erreicht ist. Eine Ausnahme ist das Verpixeln: Wiederholen
// führt das Mosaik neu aus, sein Rauschen ist dann ein anderes, die Kennung dieselbe (F-711). Das Mosaik trägt nichts
// vom Inhalt (ADR-0022 Nachtrag 1), „gleich“ heißt dort also: gleich verdeckt. Ein laufender Zug zählt nicht (er
// landet erst mit BeendeZug im Verlauf).
func (d *Dokument) Stand() uint64 {
	if n := len(d.verlauf.erledigt); n > 0 {
		return d.verlauf.erledigt[n-1].stand
	}
	return d.verlauf.basis
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
	s := d.verlauf.erledigt[n-1]
	d.verlauf.erledigt = d.verlauf.erledigt[:n-1]
	s.b.zuruecknehmen(d)
	d.verlauf.zurueckgenommen = append(d.verlauf.zurueckgenommen, s)
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
	s := d.verlauf.zurueckgenommen[n-1]
	d.verlauf.zurueckgenommen = d.verlauf.zurueckgenommen[:n-1]
	s.b.ausfuehren(d)
	d.verlauf.erledigt = append(d.verlauf.erledigt, s)
	return nil
}

// VergissVerlauf leert den Verlauf, etwa nach dem Speichern, damit auch im Speicher keine geschwärzten Pixel mehr
// zurückzuholen sind. Der Stand bleibt, die Kennungen zählen weiter.
func (d *Dokument) VergissVerlauf() error {
	if d.zug != nil {
		return ErrZugLaeuft
	}
	d.verlauf = verlauf{basis: d.Stand(), letzte: d.verlauf.letzte}
	d.schwaerzVerworfen = false
	return nil
}

// Abzug liefert eine unabhängige Kopie des aktuellen Inhalts zum Speichern im Hintergrund (F-143): Original mit allen
// Schwärzungen, Objekte, Zuschnitt, Streifen, Kantenform, Schwärzungszähler und Stand. Der Verlauf bleibt weg (Stand bleibt
// gleich, Rückgängig tut am Abzug nichts). Ändert sich das Dokument danach, berührt das den Abzug nicht und
// umgekehrt. Während eines Zugs: ErrZugLaeuft — der Inhalt stünde zwischen zwei Ständen.
func (d *Dokument) Abzug() (*Dokument, error) {
	if d.zug != nil {
		return nil, ErrZugLaeuft
	}
	a := &Dokument{
		original:          d.Original(),
		objekte:           append([]Objekt(nil), d.objekte...), // Objekte sind Werte; einziger geteilter Zeiger ist Text.Spitze, durch den nie geschrieben wird (neu setzen, ADR-0021 Nachtrag 13)
		naechsteID:        d.naechsteID,
		kante:             d.kante,
		geschwaerzt:       d.geschwaerzt,
		schwaerzStand:     d.schwaerzStand,
		schwaerzKennungen: d.schwaerzKennungen,
		verlauf:           verlauf{basis: d.Stand(), letzte: d.verlauf.letzte},
	}
	a.setzeAnsicht(d.zuschnitt, append([]Streifen(nil), d.streifen...))
	return a, nil
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
		return 0, fmt.Errorf("document: mehr als %d Objekte", MaxObjekte)
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
	// mehrere: Zug über mehrere Objekte (gemeinsam verschieben, ADR-0039), je ihr Stand davor; nil: ein Objekt (alt, id).
	mehrere []Objekt
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
		return 0, fmt.Errorf("document: mehr als %d Objekte", MaxObjekte)
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
	if d.zug.mehrere != nil || o.Kennung() != d.zug.id {
		return fmt.Errorf("document: Zug gilt Objekt %d, nicht %d", d.zug.id, o.Kennung())
	}
	if err := o.pruefe(); err != nil {
		return err
	}
	d.objekte[d.index(d.zug.id)] = o
	return nil
}

// BeendeZug übernimmt den Zug als eine Änderung in den Verlauf. Ein unveränderter Zug an einem vorhandenen Objekt (Gleich,
// die Spitze einer Sprechblase dem Wert nach, F-694) hinterlässt nichts.
func (d *Dokument) BeendeZug() error {
	if d.zug == nil {
		return ErrKeinZug
	}
	z := d.zug
	d.zug = nil
	if z.mehrere != nil {
		d.beendeMehrere(z)
		return nil
	}
	i := d.index(z.id)
	neu := d.objekte[i]
	if z.alt == nil {
		// Für den Verlauf herausnehmen und regulär einfügen: Rückgängig entfernt dann genau dieses Objekt.
		d.objekte = append(d.objekte[:i], d.objekte[i+1:]...)
		d.fuehreAus(einfuegen{neu, i})
		return nil
	}
	if !Gleich(neu, z.alt) {
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
	if z.mehrere != nil {
		d.brecheMehrereAb(z)
		return nil
	}
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

// SetzeAnsicht setzt Zuschnitt und Streifen zusammen in einem Rückgängig-Schritt („Ganzes Bild“, F-222).
func (d *Dokument) SetzeAnsicht(z image.Rectangle, st []Streifen) error {
	if d.zug != nil {
		return ErrZugLaeuft
	}
	z, st = z.Canon(), sortiert(st)
	if err := d.pruefeAnsicht(z, st); err != nil {
		return err
	}
	d.fuehreAus(ansichtAendern{d.zuschnitt, z, d.streifen, st})
	return nil
}

type schwaerzen struct {
	r      image.Rectangle
	s      Schwaerzung
	vorher *image.NRGBA // alte Pixel, nur im Speicher
	// kennung ist die eigene Schwärzungs-Kennung (beim ersten Ausführen vergeben, beim Wiederholen dieselbe), davor
	// der SchwaerzStand vor dem Ausführen (F-187).
	kennung, davor uint64
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
	d.geschwaerzt++
	if b.kennung == 0 {
		d.schwaerzKennungen++
		b.kennung = d.schwaerzKennungen
	}
	b.davor, d.schwaerzStand = d.schwaerzStand, b.kennung
}

func (b *schwaerzen) zuruecknehmen(d *Dokument) {
	for y := b.r.Min.Y; y < b.r.Max.Y; y++ {
		q := b.vorher.PixOffset(b.r.Min.X, y)
		z := d.original.PixOffset(b.r.Min.X, y)
		copy(d.original.Pix[z:z+4*b.r.Dx()], b.vorher.Pix[q:q+4*b.r.Dx()])
	}
	d.geschwaerzt--
	d.schwaerzStand = b.davor
}

// SchwaerzStand ist die Kennung der zuletzt wirksamen Schwärzung (0: keine seit Neu bzw. Laden). Jede neue Schwärzung
// bekommt eine neue Kennung; Rückgängig stellt die Kennung davor wieder her, Wiederholen dieselbe wie zuvor. Da
// Schwärzungen nur in Verlaufsreihenfolge zurückgenommen werden, bestimmt die Kennung die Menge der wirksamen
// Schwärzungen. Wer sich den Wert beim Schreiben merkt, weiß später, ob seitdem etwas geschwärzt wurde, das die
// geschriebene Datei nicht enthält (F-169) — auch wenn dazwischen geschwärzt und wieder zurückgenommen wurde (F-187).
func (d *Dokument) SchwaerzStand() uint64 { return d.schwaerzStand }

// Geschwaerzt meldet, ob seit dem Anlegen bzw. Laden mindestens eine Schwärzung wirksam ist (auch nach
// VergissVerlauf). Der Aufrufer weiß damit, dass eine ältere Fassung der Datei ungeschwärzte Pixel enthalten kann.
func (d *Dokument) Geschwaerzt() bool { return d.geschwaerzt > 0 }

// VerlaufMitSchwaerzung meldet, ob der Rückgängig-Verlauf (erledigt oder zurückgenommen) eine Schwärzung enthält und
// damit ungeschwärzte Pixel im Speicher hält.
func (d *Dokument) VerlaufMitSchwaerzung() bool {
	for _, l := range [][]schritt{d.verlauf.erledigt, d.verlauf.zurueckgenommen} {
		for _, s := range l {
			if _, ok := s.b.(*schwaerzen); ok {
				return true
			}
		}
	}
	return false
}

// Schwaerzen macht den Bereich r (im Original, wird aufs Bild begrenzt) unkenntlich, indem es die Pixel des
// Originals überschreibt (Kernprinzip 2). Verpixeln verlangt Blöcke ab MinBlock Pixeln.
func (d *Dokument) Schwaerzen(r image.Rectangle, s Schwaerzung) error {
	if d.zug != nil {
		return ErrZugLaeuft
	}
	r, s, err := pruefeSchwaerzung(d.original.Rect, r, s)
	if err != nil {
		return err
	}
	d.fuehreAus(&schwaerzen{r: r, s: s})
	return nil
}

// PruefeSchwaerzung prüft eine Schwärzung wie Schwaerzen, ohne Pixel anzufassen, und liefert den Bereich, den sie
// überschriebe (aufs Bild begrenzt). Liefert sie keinen Fehler, gelingt Schwaerzen mit denselben Werten, solange kein
// Zug läuft. Für das Wiederherstellen einer Rettungskopie: alle Schwärzungen vorab prüfen und ihre Fläche summieren
// (F-412).
func (d *Dokument) PruefeSchwaerzung(r image.Rectangle, s Schwaerzung) (image.Rectangle, error) {
	r, _, err := pruefeSchwaerzung(d.original.Rect, r, s)
	return r, err
}

// pruefeSchwaerzung begrenzt r auf das Bild, prüft die Art und klemmt die Blockgröße.
func pruefeSchwaerzung(bild, r image.Rectangle, s Schwaerzung) (image.Rectangle, Schwaerzung, error) {
	if s.Mosaik < 0 || s.Mosaik > MaxMosaik {
		return r, s, fmt.Errorf("document: Mosaik %d außerhalb 0..%d", s.Mosaik, MaxMosaik)
	}
	if s.Block > 0 && s.Block < MinBlock {
		return r, s, fmt.Errorf("document: Blockgröße %d kleiner als %d", s.Block, MinBlock)
	}
	if s.Block <= 0 && s.Farbe.A != 255 {
		// Eine durchsichtige Füllung (etwa der Nullwert) hinterließe ein Loch statt einer Schwärzung.
		return r, s, fmt.Errorf("document: Schwärzungsfarbe muss deckend sein")
	}
	r = r.Canon().Intersect(bild)
	if r.Empty() {
		return r, s, fmt.Errorf("document: Bereich zum Schwärzen liegt außerhalb des Bilds")
	}
	if s.Block > 0 && (r.Dx() < MinBlock || r.Dy() < MinBlock) {
		// Ein schmaler Bereich lässt sich nicht wirksam verpixeln (ein Block über 1 Pixel ändert nichts).
		return r, s, fmt.Errorf("document: Bereich zum Verpixeln kleiner als %d Pixel — schwärzen statt verpixeln", MinBlock)
	}
	if s.Block > max(r.Dx(), r.Dy()) {
		// Ein Block größer als der Bereich ist ein einziger Block; ohne Klemme liefe bloecke bei riesigen Werten (etwa aus
		// einer präparierten Rettungskopie) über (Verify 7, V7-4).
		s.Block = max(r.Dx(), r.Dy())
	}
	return r, s, nil
}
