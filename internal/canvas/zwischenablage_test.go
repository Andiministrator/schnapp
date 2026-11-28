// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"errors"
	"image"
	"slices"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// kopiere wählt ids, kopiert sie und liefert die Kopie.
func kopiere(t *testing.T, s *Sitzung, ids ...int) *document.Dokument {
	t.Helper()
	s.setzeAuswahl(ids[len(ids)-1], ids)
	q, err := s.KopierObjekte()
	if err != nil {
		t.Fatal(err)
	}
	return q
}

// ADR-0045 Punkt 1: ohne Auswahl, im Zug oder in der Texteingabe kein Objekt-Kopieren; die Kopie trägt genau die
// gewählten in Stapelreihenfolge, Größe, Zuschnitt und Kante der Quelle.
func TestKopierObjekte(t *testing.T) {
	s := dreiKaesten(t)
	if s.KannObjekteKopieren() {
		t.Error("ohne Auswahl kopierbar")
	}
	if _, err := s.KopierObjekte(); !errors.Is(err, ErrNichtsGewaehlt) {
		t.Errorf("ohne Auswahl: %v", err)
	}
	if err := s.D.SetzeZuschnitt(image.Rect(5, 0, 190, 100)); err != nil {
		t.Fatal(err)
	}
	q := kopiere(t, s, 3, 1)
	if !s.KannObjekteKopieren() {
		t.Error("mit Auswahl nicht kopierbar")
	}
	os := q.Objekte()
	if len(os) != 2 || os[0].Grenzen().Min.X != 10 || os[1].Grenzen().Min.X != 120 {
		t.Fatalf("Kopie %v", os)
	}
	if q.OriginalGroesse() != s.D.OriginalGroesse() || q.Zuschnitt() != s.D.Zuschnitt() || q.Kante() != s.D.Kante() {
		t.Errorf("Angaben der Kopie %v %v %v", q.OriginalGroesse(), q.Zuschnitt(), q.Kante())
	}
	s.D.BeginneZug(1)
	if s.KannObjekteKopieren() {
		t.Error("im Zug kopierbar")
	}
	if _, err := s.KopierObjekte(); !errors.Is(err, ErrNichtJetzt) {
		t.Errorf("im Zug: %v", err)
	}
}

// ADR-0045 Punkt 3: Einfügen in eine andere Aufnahme hält die Lage im sichtbaren Bild (hier mit anderem Zuschnitt im
// Ziel), legt zuoberst als ein Schritt ab, wählt alle eingefügten und das Werkzeug Auswählen.
func TestEinfuegenHaeltLageInDerAnsicht(t *testing.T) {
	quelle := dreiKaesten(t)
	if err := quelle.D.SetzeZuschnitt(image.Rect(5, 0, 200, 100)); err != nil { // Ansicht beginnt bei x = 5
		t.Fatal(err)
	}
	q := kopiere(t, quelle, 2) // Kasten bei x 60 im Original, 55 in der Ansicht

	ziel := dreiKaesten(t)
	if err := ziel.D.SetzeZuschnitt(image.Rect(20, 0, 200, 100)); err != nil {
		t.Fatal(err)
	}
	_ = ziel.D.VergissVerlauf()
	ziel.SetzeWerkzeug(WerkzeugPfeil)
	ids, err := ziel.Einfuegen(q, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || minX(t, ziel, ids[0]) != 75 { // 55 in der Ansicht + 20 Zuschnitt
		t.Fatalf("eingefügt %v bei x %v, erwartet 75", ids, minX(t, ziel, ids[0]))
	}
	alle := ziel.D.Objekte()
	if alle[len(alle)-1].Kennung() != ids[0] {
		t.Error("nicht zuoberst")
	}
	if !slices.Equal(ziel.Gewaehlt(), ids) || ziel.Werkzeug != WerkzeugAuswahl {
		t.Errorf("gewählt %v, Werkzeug %v", ziel.Gewaehlt(), ziel.Werkzeug)
	}
	if err := ziel.D.Rueckgaengig(); err != nil || len(ziel.D.Objekte()) != 3 || ziel.D.KannRueckgaengig() {
		t.Errorf("ein Rückgängig-Schritt erwartet: %v, %d Objekte", err, len(ziel.D.Objekte()))
	}
}

// ADR-0045 Punkt 3: Erneutes Einfügen in dieselbe Aufnahme rückt um den Versatz des Doppelns weiter, jedes Mal.
func TestEinfuegenRueckt(t *testing.T) {
	s := dreiKaesten(t)
	q := kopiere(t, s, 1)
	v := DoppelVersatz / s.Abb.Zoom
	for i := 1; i <= 3; i++ {
		ids, err := s.Einfuegen(q, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := minX(t, s, ids[0]), 10+float64(i)*v; got != want {
			t.Errorf("Einfügen %d bei x %v, erwartet %v", i, got, want)
		}
	}
}

// ADR-0045 Punkt 3: Was im Ziel nicht ganz sichtbar bliebe, kommt in die Mitte des Fensterausschnitts, ins Bild
// begrenzt.
func TestEinfuegenMitteWennUnsichtbar(t *testing.T) {
	quelle := dreiKaesten(t)
	q := kopiere(t, quelle, 3) // x 120 … 140

	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 100, 60)))
	if err != nil {
		t.Fatal(err)
	}
	ziel := NeueSitzung(d)
	ziel.SetzeFenster(image.Pt(148, 108)) // eingepasst: Mitte des Fensters = Mitte des Bilds (50, 30)
	ids, err := ziel.Einfuegen(q, nil)
	if err != nil {
		t.Fatal(err)
	}
	o, _ := ziel.D.Objekt(ids[0])
	if g := o.Grenzen(); g.Min.X != 40 || g.Min.Y != 20 {
		t.Errorf("Grenzen %v, erwartet ab (40, 20)", g)
	}
}

// ADR-0045 Punkt 3: Gruppen bekommen neue Kennungen, Ausgeblendet und Gesperrt fallen weg.
func TestEinfuegenGruppenUndSchalter(t *testing.T) {
	s := dreiKaesten(t)
	s.setzeAuswahl(2, []int{1, 2})
	if !s.Gruppieren() {
		t.Fatal("Gruppieren")
	}
	q := kopiere(t, s, 1, 2)
	o, _ := s.D.Objekt(1)
	alteGruppe := o.Ebenenwerte().Gruppe
	// in der Kopie ausblenden und sperren
	os := q.Objekte()
	for _, x := range os {
		e := x.Ebenenwerte()
		e.Ausgeblendet, e.Gesperrt = true, true
		if err := q.Ersetzen(document.MitEbene(x, e)); err != nil {
			t.Fatal(err)
		}
	}
	ids, err := s.Einfuegen(q, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(s.Gewaehlt(), ids) || s.Auswahl != ids[1] {
		t.Errorf("gewählt %v (Haupt %d), erwartet %v", s.Gewaehlt(), s.Auswahl, ids)
	}
	var gruppen []int
	for _, id := range ids {
		o, _ := s.D.Objekt(id)
		e := o.Ebenenwerte()
		if e.Ausgeblendet || e.Gesperrt {
			t.Errorf("Objekt %d ausgeblendet %v gesperrt %v", id, e.Ausgeblendet, e.Gesperrt)
		}
		gruppen = append(gruppen, e.Gruppe)
	}
	if gruppen[0] == 0 || gruppen[0] != gruppen[1] || gruppen[0] == alteGruppe {
		t.Errorf("Gruppen der Eingefügten %v, alte %d", gruppen, alteGruppe)
	}
}

// ADR-0045 Punkt 3: leer oder im Zug wird nichts eingefügt.
func TestEinfuegenNicht(t *testing.T) {
	s := dreiKaesten(t)
	leer, err := document.OhnePixel(image.Pt(10, 10), image.Rect(0, 0, 10, 10), nil, document.KanteAbriss)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Einfuegen(leer, nil); !errors.Is(err, ErrNichtsEinzufuegen) {
		t.Errorf("leer: %v", err)
	}
	q := kopiere(t, s, 1)
	s.D.BeginneZug(1)
	if _, err := s.Einfuegen(q, nil); !errors.Is(err, ErrNichtJetzt) {
		t.Errorf("im Zug: %v", err)
	}
	_ = s.D.BrecheZugAb()
	if n := len(s.D.Objekte()); n != 3 {
		t.Errorf("%d Objekte", n)
	}
}

// ADR-0045 Punkt 3: Die Prüfung sieht alle Objekte danach; ihr Fehler verhindert das Einfügen ohne Schritt.
func TestEinfuegenPrueft(t *testing.T) {
	s := dreiKaesten(t)
	q := kopiere(t, s, 1, 2)
	stand := s.D.Stand()
	nein := errors.New("zu teuer")
	var gesehen int
	_, err := s.Einfuegen(q, func(d *document.Dokument) error { gesehen = len(d.Objekte()); return nein })
	if !errors.Is(err, nein) || gesehen != 5 || s.D.Stand() != stand || len(s.D.Objekte()) != 3 {
		t.Errorf("Fehler %v, geprüft %d Objekte, Stand geändert %v", err, gesehen, s.D.Stand() != stand)
	}
}

// ADR-0045 Punkt 3: Die Mitte ist die des Fensterausschnitts, nicht die des Bilds.
func TestEinfuegenMitteDesFensters(t *testing.T) {
	quelle := dreiKaesten(t)
	q := kopiere(t, quelle, 3) // x 120 … 140, liegt im kleinen Ziel nicht

	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 100, 60)))
	if err != nil {
		t.Fatal(err)
	}
	ziel := NeueSitzung(d)
	ziel.SetzeFenster(image.Pt(40, 40))
	ziel.Abb = Abbildung{Zoom: 1, Versatz: document.Pt(0, 0)} // Fenster zeigt (0,0)–(40,40), Mitte (20, 20)
	ids, err := ziel.Einfuegen(q, nil)
	if err != nil {
		t.Fatal(err)
	}
	o, _ := ziel.D.Objekt(ids[0])
	if g := o.Grenzen(); g.Min.X != 10 || g.Min.Y != 10 {
		t.Errorf("Grenzen %v, erwartet ab (10, 10)", g)
	}
}

// ADR-0045 Punkt 1: Während des Speicherns geht Kopieren, Ausschneiden nicht.
func TestAusschneidenNichtWaehrendSpeichern(t *testing.T) {
	s := dreiKaesten(t)
	s.setzeAuswahl(1, []int{1})
	if !s.KannAusschneiden() {
		t.Fatal("Vorbedingung")
	}
	stand := s.D.Stand()
	s.hg.laeuft = true
	if s.Ausschneiden([]int{1}, stand) {
		t.Error("Ausschneiden während des Speicherns")
	}
	if !s.KannObjekteKopieren() || s.KannAusschneiden() || s.KannEinfuegen() {
		t.Errorf("während des Speicherns: kopieren %v, ausschneiden %v, einfügen %v", s.KannObjekteKopieren(),
			s.KannAusschneiden(), s.KannEinfuegen())
	}
}

// F-985: Ein Stauch-Streifen des Ziels unter der alten Lage zählt als unsichtbar; das Eingefügte landet ganz sichtbar
// neben der Naht.
func TestEinfuegenMeidetStreifenDesZiels(t *testing.T) {
	quelle := dreiKaesten(t)
	q := kopiere(t, quelle, 1) // x 10 … 30

	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 400, 100)))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]document.Streifen{{Von: 11, Bis: 300}}); err != nil {
		t.Fatal(err)
	}
	ziel := NeueSitzung(d)
	ziel.SetzeFenster(image.Pt(400, 200))
	ids, err := ziel.Einfuegen(q, nil)
	if err != nil {
		t.Fatal(err)
	}
	o, _ := ziel.D.Objekt(ids[0])
	g := o.Grenzen()
	a, e := ziel.D.ZuAnsicht(g.Min), ziel.D.ZuAnsicht(g.Max)
	if e.X-a.X != g.Breite() || a.X < 0 || e.X > float64(ziel.D.Groesse().X) {
		t.Errorf("Grenzen %v, in der Ansicht %v … %v: nicht ganz sichtbar", g, a, e)
	}
}

// F-989: Ausschneiden löscht nur bei unverändertem Stand, in einem Schritt.
func TestAusschneidenMitStand(t *testing.T) {
	s := dreiKaesten(t)
	stand := s.D.Stand()
	if s.Ausschneiden([]int{1, 2}, stand+1) || len(s.D.Objekte()) != 3 {
		t.Error("bei geändertem Stand gelöscht")
	}
	if !s.Ausschneiden([]int{1, 2}, stand) || len(s.D.Objekte()) != 1 || s.AnzahlGewaehlt() != 0 {
		t.Fatalf("%d Objekte, %d gewählt", len(s.D.Objekte()), s.AnzahlGewaehlt())
	}
	if err := s.D.Rueckgaengig(); err != nil || len(s.D.Objekte()) != 3 {
		t.Errorf("Rückgängig: %v, %d Objekte", err, len(s.D.Objekte()))
	}
}

// F-985: Liegt auch die Mitte des Fensters über einer Naht, rückt das Eingefügte direkt daneben.
func TestEinfuegenMitteNebenDerNaht(t *testing.T) {
	quelle := dreiKaesten(t)
	q := kopiere(t, quelle, 3) // x 120 … 140

	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 400, 100)))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]document.Streifen{{Von: 130, Bis: 230}}); err != nil { // Naht bei x 130 der Ansicht
		t.Fatal(err)
	}
	ziel := NeueSitzung(d)
	ziel.SetzeFenster(image.Pt(260, 100))
	ziel.Abb = Abbildung{Zoom: 1} // Mitte des Fensters x 130: genau auf der Naht
	ids, err := ziel.Einfuegen(q, nil)
	if err != nil {
		t.Fatal(err)
	}
	o, _ := ziel.D.Objekt(ids[0])
	g := o.Grenzen()
	a, e := ziel.D.ZuAnsicht(g.Min), ziel.D.ZuAnsicht(g.Max)
	if e.X-a.X != g.Breite() {
		t.Errorf("Grenzen %v, in der Ansicht %v … %v: überdeckt den Streifen", g, a, e)
	}
}
