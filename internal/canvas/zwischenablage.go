// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"errors"
	"fmt"
	"image"
	"math"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Objekte kopieren, ausschneiden und einfügen (ADR-0045). Die Oberfläche bringt die Objekte in die Zwischenablage und
// liest sie wieder; hier steht, was kopiert wird und wo Eingefügtes landet.

var (
	// ErrNichtJetzt: Kopieren oder Einfügen geht gerade nicht (Zug, Texteingabe, Speichern, Laden).
	ErrNichtJetzt = errors.New("canvas: gerade nicht möglich")
	// ErrNichtsGewaehlt: Kopieren ohne Auswahl.
	ErrNichtsGewaehlt = errors.New("canvas: nichts gewählt")
	// ErrNichtsEinzufuegen: Die Quelle trägt keine Objekte.
	ErrNichtsEinzufuegen = errors.New("canvas: keine Objekte")
)

// einfuegeVersuche begrenzt, wie oft Einfügen um den Versatz des Doppelns weiterrückt, solange an der Stelle schon
// gleiche Objekte stehen.
const einfuegeVersuche = 64

// vergleichGrenze begrenzt die Vergleiche (eingefügte × vorhandene Objekte) je Versuch: darüber rückt Einfügen nicht
// weiter, statt bei Tausenden Objekten spürbar zu stocken.
const vergleichGrenze = 200_000

// kopierBereit meldet, ob Kopieren von Objekten jetzt geht: kein Zug, keine Texteingabe. Während des Speicherns geht
// es wie das Kopieren des Bilds (F-285): Es ändert nichts am Dokument.
func (s *Sitzung) kopierBereit() bool { return s.zug == keinZug && !s.D.ZugLaeuft() }

// zwischenablageBereit meldet, ob Einfügen jetzt geht: zusätzlich nicht während Speichern oder Laden.
func (s *Sitzung) zwischenablageBereit() bool { return s.kopierBereit() && !s.gesperrt() }

// KannObjekteKopieren meldet, ob Strg+C die Auswahl kopiert (ADR-0045 Punkt 1): mindestens ein Objekt gewählt, kein
// Zug, keine Texteingabe. Sonst kopiert Strg+C das Bild.
func (s *Sitzung) KannObjekteKopieren() bool { return s.kopierBereit() && s.AnzahlGewaehlt() > 0 }

// KannAusschneiden meldet, ob Strg+X wirkt: kopierbar und löschbar.
func (s *Sitzung) KannAusschneiden() bool { return s.KannObjekteKopieren() && s.KannLoeschen() }

// KannEinfuegen meldet, ob Einfügen jetzt ginge (die Zwischenablage selbst prüft die Oberfläche).
func (s *Sitzung) KannEinfuegen() bool { return s.zwischenablageBereit() }

// KopierObjekte liefert die gewählten Objekte als Dokument ohne Pixel mit Größe, Zuschnitt, Streifen und Kantenform
// dieser Aufnahme (document.NurObjekte, ADR-0045 Punkt 2), für die Zwischenablage.
func (s *Sitzung) KopierObjekte() (*document.Dokument, error) {
	if !s.kopierBereit() {
		return nil, ErrNichtJetzt
	}
	ids := s.Gewaehlt()
	if len(ids) == 0 {
		return nil, ErrNichtsGewaehlt
	}
	return s.D.NurObjekte(ids)
}

// Ausschneiden löscht die gewählten Objekte nach einem gelungenen Kopieren (ein Rückgängig-Schritt, wie Entf).
func (s *Sitzung) Ausschneiden() bool { return s.Loeschen() }

// Einfuegen fügt die Objekte aus q (Dokument ohne Pixel aus der Zwischenablage, KopierObjekte) zuoberst ein, als ein
// Schritt, und wählt sie (ADR-0045 Punkt 3). Die Lage im sichtbaren Bild bleibt, sonst Mitte des Fensterausschnitts;
// stehen an der Stelle schon gleiche Objekte, rückt es um den Versatz des Doppelns weiter. Gruppen bekommen neue
// Kennungen, Ausgeblendet und Gesperrt fallen weg. pruefe (nil: keine Prüfung) bekommt vorher ein Dokument ohne Pixel
// mit allen Objekten danach — die Oberfläche prüft damit das Budget des Öffnens (render.Pruefe); ein Fehler daraus
// verhindert das Einfügen.
func (s *Sitzung) Einfuegen(q *document.Dokument, pruefe func(*document.Dokument) error) ([]int, error) {
	if !s.zwischenablageBereit() || s.TextLaeuft() {
		return nil, ErrNichtJetzt
	}
	os := q.Objekte()
	if len(os) == 0 {
		return nil, ErrNichtsEinzufuegen
	}
	for i, o := range os {
		os[i] = document.OhneSchalter(o)
	}
	b := gemeinsameGrenzen(os)
	delta := s.einfuegeVersatz(q, b)
	if s.Abb.Zoom > 0 && len(os)*len(s.D.Objekte()) <= vergleichGrenze {
		v := DoppelVersatz / s.Abb.Zoom
		for k := 0; k < einfuegeVersuche && s.stehtSchonDa(os, delta); k++ {
			delta = delta.Plus(document.Pt(v, v))
		}
	}
	for i, o := range os {
		os[i] = Verschiebe(o, delta)
	}
	os = s.neueGruppenFuer(os) // eigene Gruppen der Eingefügten (ADR-0044 Punkt 4)
	if pruefe != nil {
		probe, err := s.D.NurObjekte(nil)
		if err != nil {
			return nil, err
		}
		alle := append(s.D.ObjekteOhneZug(), os...)
		if len(alle) > document.MaxObjekte {
			return nil, fmt.Errorf("canvas: mehr als %d Objekte", document.MaxObjekte)
		}
		if _, err := probe.HinzufuegenMehrere(alle); err != nil {
			return nil, err
		}
		if err := pruefe(probe); err != nil {
			return nil, err
		}
	}
	vor := s.D.Stand()
	ids, err := s.D.HinzufuegenMehrere(os)
	if err != nil {
		return nil, err
	}
	s.merkeAenderung(vor)
	if s.Werkzeug != WerkzeugAuswahl {
		s.SetzeWerkzeug(WerkzeugAuswahl) // sonst höbe der Werkzeugwechsel die Auswahl gleich wieder auf (ADR-0039 Nachtrag 3)
	}
	s.pipette = false
	s.setzeAuswahl(ids[len(ids)-1], ids)
	return ids, nil
}

// gemeinsameGrenzen vereint die Grenzen aller Objekte.
func gemeinsameGrenzen(os []document.Objekt) document.Rechteck {
	b := os[0].Grenzen()
	for _, o := range os[1:] {
		b = b.Vereint(o.Grenzen())
	}
	return b
}

// einfuegeVersatz ist die Verschiebung der Objekte mit den Grenzen b (im Original von q) ins Original dieser Aufnahme:
// dieselbe Lage im sichtbaren Bild, wenn das Ganze dort sichtbar bleibt, sonst in der Mitte des Fensterausschnitts,
// ins sichtbare Bild begrenzt.
func (s *Sitzung) einfuegeVersatz(q *document.Dokument, b document.Rechteck) document.Punkt {
	g := s.D.Groesse()
	ansicht := q.ZuAnsicht(b.Min)
	if innerhalb(ansicht, b, g) {
		return s.D.ZuOriginal(ansicht).Minus(b.Min)
	}
	mitte := document.Pt(float64(g.X)/2, float64(g.Y)/2)
	if s.fenster.X > 0 && s.fenster.Y > 0 && s.Abb.Zoom > 0 {
		mitte = s.Abb.ZuAnsicht(document.Pt(float64(s.fenster.X)/2, float64(s.fenster.Y)/2))
	}
	ecke := document.Pt(
		inBereich(mitte.X-b.Breite()/2, 0, float64(g.X)-b.Breite()),
		inBereich(mitte.Y-b.Hoehe()/2, 0, float64(g.Y)-b.Hoehe()))
	return s.D.ZuOriginal(ecke).Minus(b.Min)
}

// innerhalb meldet, ob Grenzen der Größe von b mit der linken oberen Ecke bei ecke (Ansicht) ganz in einer Ansicht der
// Größe g liegen. Die Größe gilt wie im Original; ein Streifen innerhalb der Grenzen ändert sie nur in der Quelle.
func innerhalb(ecke document.Punkt, b document.Rechteck, g image.Point) bool {
	return ecke.X >= 0 && ecke.Y >= 0 && ecke.X+b.Breite() <= float64(g.X) && ecke.Y+b.Hoehe() <= float64(g.Y)
}

// inBereich hält v in [lo, hi]; ist der Bereich leer (größer als das Bild), gilt lo.
func inBereich(v, lo, hi float64) float64 {
	if hi < lo {
		return lo
	}
	return math.Min(math.Max(v, lo), hi)
}

// stehtSchonDa meldet, ob jedes Objekt aus os, um delta verschoben, schon gleich in der Aufnahme steht (bis auf
// Kennung und Gruppe): erneutes Einfügen an derselben Stelle.
func (s *Sitzung) stehtSchonDa(os []document.Objekt, delta document.Punkt) bool {
	vorhanden := s.D.Objekte()
	for _, o := range os {
		v := mitGruppe(Verschiebe(o, delta), 0)
		gefunden := false
		for _, w := range vorhanden {
			if document.GleichOhneKennung(v, mitGruppe(document.OhneSchalter(w), 0)) {
				gefunden = true
				break
			}
		}
		if !gefunden {
			return false
		}
	}
	return true
}
