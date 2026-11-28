// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"slices"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

// gruppiert sind dreiKaesten mit 1 und 2 als Gruppe (Strg+G über die Mehrfachauswahl).
func gruppiert(t *testing.T) *Sitzung {
	t.Helper()
	s := dreiKaesten(t)
	s.setzeAuswahl(2, []int{1})
	if !s.KannGruppieren() || !s.Gruppieren() {
		t.Fatal("Gruppieren")
	}
	if s.GruppeVon(1) == 0 || s.GruppeVon(1) != s.GruppeVon(2) || s.GruppeVon(3) != 0 {
		t.Fatalf("Gruppen %d %d %d", s.GruppeVon(1), s.GruppeVon(2), s.GruppeVon(3))
	}
	s.waehleNur(0)
	return s
}

// ADR-0044 Punkt 2 und 3: Gruppieren ist ein Schritt; ein Klick auf ein Mitglied wählt die Gruppe, das angeklickte wird
// Hauptauswahl; Ziehen verschiebt die Gruppe.
func TestGruppeKlickWaehltGanz(t *testing.T) {
	s := gruppiert(t)
	klick(s, f(65, 15), false)
	if got := s.Gewaehlt(); !slices.Equal(got, []int{1, 2}) || s.Auswahl != 2 {
		t.Fatalf("Klick: %v Haupt %d", got, s.Auswahl)
	}
	klick(s, f(125, 15), false)
	if got := s.Gewaehlt(); !slices.Equal(got, []int{3}) {
		t.Fatalf("Klick auf ungruppiertes: %v", got)
	}
	// Ziehen an einem nicht gewählten Mitglied verschiebt die ganze Gruppe.
	x1, x2 := minX(t, s, 1), minX(t, s, 2)
	s.Druecken(f(15, 15), TastePrimaer)
	s.Ziehen(f(25, 15))
	s.Loslassen(f(25, 15))
	if minX(t, s, 1) != x1+10 || minX(t, s, 2) != x2+10 || s.Auswahl != 1 {
		t.Errorf("Ziehen: %v %v Haupt %d", minX(t, s, 1), minX(t, s, 2), s.Auswahl)
	}
	s.Rueckgaengig()
	if minX(t, s, 1) != x1 || minX(t, s, 2) != x2 {
		t.Error("Verschieben war nicht ein Schritt")
	}
	s.Rueckgaengig()
	if s.GruppeVon(1) != 0 {
		t.Error("Rückgängig hebt das Gruppieren nicht auf")
	}
}

// Umschalt+Klick nimmt die Gruppe dazu und heraus; der Rahmen nimmt eine Gruppe nur ganz.
func TestGruppeUmschaltUndRahmen(t *testing.T) {
	s := gruppiert(t)
	klick(s, f(125, 15), false)
	klick(s, f(15, 15), true)
	if got := s.Gewaehlt(); !slices.Equal(got, []int{1, 2, 3}) || s.Auswahl != 1 {
		t.Fatalf("Umschalt dazu: %v Haupt %d", got, s.Auswahl)
	}
	klick(s, f(65, 15), true)
	if got := s.Gewaehlt(); !slices.Equal(got, []int{3}) {
		t.Fatalf("Umschalt heraus: %v", got)
	}
	// Rahmen um Kasten 2 und 3: die Gruppe ist nicht ganz drin, nur 3 wird gewählt.
	s.waehleNur(0)
	s.Druecken(f(45, 70), TastePrimaer)
	s.Ziehen(f(150, 2))
	s.Loslassen(f(150, 2))
	if got := s.Gewaehlt(); !slices.Equal(got, []int{3}) {
		t.Errorf("Rahmen über halbe Gruppe: %v", got)
	}
	s.waehleNur(0)
	s.Druecken(f(2, 70), TastePrimaer)
	s.Ziehen(f(95, 2))
	s.Loslassen(f(95, 2))
	if got := s.Gewaehlt(); !slices.Equal(got, []int{1, 2}) {
		t.Errorf("Rahmen um die Gruppe: %v", got)
	}
}

// Die Ebenenliste wählt zeilenweise; Aufheben nimmt die gewählten aus der Gruppe.
func TestGruppeEbenenlisteUndAufheben(t *testing.T) {
	s := gruppiert(t)
	s.WaehleEbene(1)
	if got := s.Gewaehlt(); !slices.Equal(got, []int{1}) {
		t.Fatalf("Liste: %v", got)
	}
	z := s.Ebenen()
	if z[0].Gruppe != 0 || z[1].Gruppe == 0 || z[1].Gruppe != z[2].Gruppe {
		t.Errorf("Gruppe in der Liste: %d %d %d", z[0].Gruppe, z[1].Gruppe, z[2].Gruppe)
	}
	if !s.KannGruppierungAufheben() || !s.GruppierungAufheben() {
		t.Fatal("Aufheben")
	}
	// Kasten 2 trägt die Kennung noch, ist aber allein: keine wirksame Gruppe.
	if s.GruppeVon(1) != 0 || s.GruppeVon(2) != 0 {
		t.Errorf("nach Aufheben: %d %d", s.GruppeVon(1), s.GruppeVon(2))
	}
	klick(s, f(65, 15), false)
	if got := s.Gewaehlt(); !slices.Equal(got, []int{2}) {
		t.Errorf("einzelnes Restmitglied wählt %v", got)
	}
	s.waehleNur(0)
	if s.KannGruppieren() {
		t.Error("Gruppieren ohne Auswahl möglich")
	}
	s.setzeAuswahl(2, []int{1})
	s.Gruppieren()
	s.waehleNur(0)
	klick(s, f(15, 15), false)
	if s.KannGruppieren() {
		t.Error("eine ganze Gruppe lässt sich noch einmal gruppieren")
	}
}

// Punkt 4: Strg+H und Strg+L an einer gewählten Gruppe wirken auf alle Mitglieder (ein Schritt); Doppeln gibt den
// Kopien eine neue Gruppe, die Kopie eines einzelnen Mitglieds steht allein.
func TestGruppeAusblendenSperrenDoppeln(t *testing.T) {
	s := gruppiert(t)
	klick(s, f(15, 15), false)
	g := s.GruppeVon(1)
	if !s.SchalteGesperrtGruppe(s.Auswahl) {
		t.Fatal("Sperren")
	}
	for _, id := range []int{1, 2} {
		if o, _ := s.D.Objekt(id); !o.Ebenenwerte().Gesperrt {
			t.Errorf("Mitglied %d nicht gesperrt", id)
		}
	}
	s.Rueckgaengig()
	if o, _ := s.D.Objekt(2); o.Ebenenwerte().Gesperrt {
		t.Error("Sperren der Gruppe war nicht ein Schritt")
	}
	klick(s, f(15, 15), false)
	s.SchalteAusgeblendetGruppe(s.Auswahl)
	if o, _ := s.D.Objekt(2); !o.Ebenenwerte().Ausgeblendet {
		t.Error("Mitglied 2 nicht ausgeblendet")
	}
	s.Rueckgaengig()
	klick(s, f(15, 15), false)
	n := len(s.D.Objekte())
	if !s.Doppeln() || len(s.D.Objekte()) != n+2 {
		t.Fatal("Doppeln der Gruppe")
	}
	ids := s.Gewaehlt()
	g2 := s.GruppeVon(ids[0])
	if g2 == 0 || g2 == g || s.GruppeVon(ids[1]) != g2 {
		t.Errorf("Kopien: Gruppen %d %d, Original %d", g2, s.GruppeVon(ids[1]), g)
	}
	s.WaehleEbene(1)
	s.Doppeln()
	if s.GruppeVon(s.Auswahl) != 0 {
		t.Error("Kopie eines einzelnen Mitglieds gehört zur Gruppe")
	}
	if o, _ := s.D.Objekt(s.Auswahl); o.Ebenenwerte().Gruppe != 0 {
		t.Error("Kopie trägt noch die Kennung")
	}
}

// Punkt 3: Rechtsklick auf ein nicht gewähltes Mitglied wählt die ganze Gruppe.
func TestGruppeRechtsklick(t *testing.T) {
	s := gruppiert(t)
	if _, objekt := s.KontextBei(f(15, 15)); !objekt {
		t.Fatal("kein Objekt getroffen")
	}
	if got := s.Gewaehlt(); !slices.Equal(got, []int{1, 2}) || s.Auswahl != 1 {
		t.Errorf("Rechtsklick: %v Haupt %d", got, s.Auswahl)
	}
}

// F-958: Strg+H viermal an einer Gruppe — der Zustand von 2 folgt immer dem von 1; ebenso Strg+L.
func TestGruppeSchalterSymmetrisch(t *testing.T) {
	for _, fall := range []struct {
		name    string
		schalte func(*Sitzung, int) bool
		wert    func(*Sitzung, int) bool
	}{
		{"ausblenden", (*Sitzung).SchalteAusgeblendetGruppe, func(s *Sitzung, id int) bool {
			o, _ := s.D.Objekt(id)
			return o.Ebenenwerte().Ausgeblendet
		}},
		{"sperren", (*Sitzung).SchalteGesperrtGruppe, func(s *Sitzung, id int) bool {
			o, _ := s.D.Objekt(id)
			return o.Ebenenwerte().Gesperrt
		}},
	} {
		s := gruppiert(t)
		klick(s, f(15, 15), false)
		for i := 1; i <= 4; i++ {
			fall.schalte(s, s.Auswahl)
			if a, b := fall.wert(s, 1), fall.wert(s, 2); a != b || a != (i%2 == 1) {
				t.Errorf("%s, Druck %d: 1=%v 2=%v", fall.name, i, a, b)
			}
			if i%2 == 0 && !slices.Equal(s.Gewaehlt(), []int{1, 2}) { // wieder sichtbar/offen: ganze Gruppe gewählt
				t.Errorf("%s, Druck %d: gewählt %v", fall.name, i, s.Gewaehlt())
			}
		}
		if s.gewaehlteGruppe(s.Auswahl) == 0 {
			t.Errorf("%s: nach vier Drücken ist die Gruppe nicht gewählt: %v", fall.name, s.Gewaehlt())
		}
	}
}

// F-959: Ein einzeln per Liste gewähltes Mitglied schaltet nur sich, auch wenn der Partner gesperrt ist.
func TestGruppeEinzelwahlMitGesperrtemPartner(t *testing.T) {
	s := gruppiert(t)
	s.SchalteGesperrt(2)
	s.WaehleEbene(1)
	if s.GruppeSchaltet(SchalterAusblenden) {
		t.Fatal("Einzelwahl gilt als Gruppenwahl")
	}
	s.SchalteAusgeblendetGruppe(1)
	if o, _ := s.D.Objekt(2); o.Ebenenwerte().Ausgeblendet {
		t.Error("gesperrter Partner mit ausgeblendet")
	}
}

// F-960: Eine ganz gewählte Gruppe mit gesperrtem Mitglied lässt sich nicht noch einmal gruppieren.
func TestGruppeMitGesperrtemNichtSpalten(t *testing.T) {
	s := dreiKaesten(t)
	s.WaehleAlle()
	s.Gruppieren()
	s.SchalteGesperrt(3)
	s.waehleNur(0)
	klick(s, f(15, 15), false)
	if got := s.Gewaehlt(); !slices.Equal(got, []int{1, 2}) {
		t.Fatalf("Klick: %v", got)
	}
	if s.KannGruppieren() {
		t.Error("Gruppieren spaltet die Gruppe")
	}
	if !s.KannGruppierungAufheben() {
		t.Error("Aufheben nicht möglich")
	}
}

// F-962: Aufheben räumt die Kennung eines einzelnen Restmitglieds mit auf; Gruppieren eines Teils ebenso.
func TestGruppeKeineVerwaisteKennung(t *testing.T) {
	s := gruppiert(t)
	s.WaehleEbene(1)
	s.GruppierungAufheben()
	if o, _ := s.D.Objekt(2); o.Ebenenwerte().Gruppe != 0 {
		t.Errorf("Restmitglied trägt noch Kennung %d", o.Ebenenwerte().Gruppe)
	}
	s.Rueckgaengig()
	if s.GruppeVon(2) == 0 {
		t.Fatal("Rückgängig stellt die Gruppe nicht her (ein Schritt)")
	}
	// Gruppe {1,2}; 2 und 3 neu gruppieren: 1 bliebe allein mit der alten Kennung.
	s.setzeAuswahl(3, []int{2})
	s.Gruppieren()
	if o, _ := s.D.Objekt(1); o.Ebenenwerte().Gruppe != 0 {
		t.Errorf("alte Kennung an 1: %d", o.Ebenenwerte().Gruppe)
	}
	s.WaehleEbene(1)
	if s.KannGruppierungAufheben() {
		t.Error("Aufheben an einem ungruppierten Objekt")
	}
}

// F-963: Mitglieder, die im Stapel nicht benachbart sind; die Gruppe überlebt Abzug und Rettungs-Angaben.
func TestGruppeNichtBenachbartUndAbzug(t *testing.T) {
	s := dreiKaesten(t)
	s.setzeAuswahl(3, []int{1})
	s.Gruppieren()
	s.waehleNur(0)
	klick(s, f(125, 15), false)
	if got := s.Gewaehlt(); !slices.Equal(got, []int{1, 3}) {
		t.Errorf("nicht benachbarte Gruppe: %v", got)
	}
	a, err := s.D.Abzug()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int{1, 3} {
		o, ok := a.Objekt(id)
		if !ok || o.Ebenenwerte().Gruppe != s.GruppeVon(1) {
			t.Errorf("Abzug verliert die Gruppe von %d", id)
		}
	}
}

// F-961: Rahmen und KannGruppieren bleiben bei einer großen Gruppe schnell.
func TestGruppeVieleObjekteSchnell(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 2000, 2000)))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2000; i++ {
		x, y := float64(i%40)*48, float64(i/40)*38
		k := document.Kasten{Bereich: document.R(document.Pt(x, y), document.Pt(x+20, y+20)),
			Strich: document.Strich{Breite: 1}}
		k.Gruppe = 1
		if _, err := d.Hinzufuegen(k); err != nil {
			t.Fatal(err)
		}
	}
	_ = d.VergissVerlauf()
	s := NeueSitzung(d)
	s.SetzeFenster(image.Pt(800, 600))
	ids := make([]int, 0, 2000)
	for _, o := range d.Objekte() {
		ids = append(ids, o.Kennung())
	}
	start := time.Now()
	if got := s.nurGanzeGruppen(ids); len(got) != 2000 {
		t.Errorf("ganze Gruppe: %d", len(got))
	}
	if got := s.nurGanzeGruppen(ids[1:]); len(got) != 0 {
		t.Errorf("halbe Gruppe: %d", len(got))
	}
	s.WaehleAlle()
	for i := 0; i < 20; i++ {
		_ = s.KannGruppieren()
		_ = s.KannGruppierungAufheben()
		_ = s.GruppeSchaltet(SchalterAusblenden)
	}
	if d := time.Since(start); gruppenZeit > 0 && d > gruppenZeit {
		t.Errorf("2000 Objekte in einer Gruppe dauerten %v", d)
	} else {
		t.Logf("2000 Objekte in einer Gruppe: %v", d)
	}
}

// Ein Restmitglied nach dem Löschen des Partners verhält sich ungruppiert (ADR-0044 Punkt 1).
func TestGruppeRestNachLoeschen(t *testing.T) {
	s := gruppiert(t)
	s.WaehleEbene(2)
	if !s.Loeschen() {
		t.Fatal("Löschen")
	}
	if s.GruppeVon(1) != 0 {
		t.Errorf("Restmitglied in Gruppe %d", s.GruppeVon(1))
	}
	klick(s, f(15, 15), false)
	if got := s.Gewaehlt(); !slices.Equal(got, []int{1}) {
		t.Errorf("Klick auf das Restmitglied: %v", got)
	}
	if s.KannGruppierungAufheben() {
		t.Error("Aufheben an einem Restmitglied")
	}
}

func ausgeblendet(t *testing.T, s *Sitzung, id int) bool {
	t.Helper()
	o, ok := s.D.Objekt(id)
	if !ok {
		t.Fatalf("Objekt %d fehlt", id)
	}
	return o.Ebenenwerte().Ausgeblendet
}

// F-964: Gruppe plus Fremdobjekt — zweimal Strg+H stellt Sichtbarkeit und Auswahl von vorher her.
func TestGruppeSchalterMitFremdobjekt(t *testing.T) {
	s := gruppiert(t)
	s.setzeAuswahl(1, []int{2, 3})
	s.SchalteAusgeblendetGruppe(s.Auswahl)
	if !ausgeblendet(t, s, 1) || !ausgeblendet(t, s, 2) || ausgeblendet(t, s, 3) {
		t.Fatalf("erster Druck: %v %v %v", ausgeblendet(t, s, 1), ausgeblendet(t, s, 2), ausgeblendet(t, s, 3))
	}
	if !s.GruppeSchaltet(SchalterAusblenden) {
		t.Error("Menü nennt nach dem ersten Druck keine Gruppe")
	}
	s.SchalteAusgeblendetGruppe(s.Auswahl)
	if ausgeblendet(t, s, 1) || ausgeblendet(t, s, 2) || ausgeblendet(t, s, 3) {
		t.Errorf("zweiter Druck: %v %v %v", ausgeblendet(t, s, 1), ausgeblendet(t, s, 2), ausgeblendet(t, s, 3))
	}
	if got := s.Gewaehlt(); !slices.Equal(got, []int{1, 2, 3}) || s.Auswahl != 1 {
		t.Errorf("Auswahl nach dem Zurückschalten: %v Haupt %d", got, s.Auswahl)
	}
}

// F-965: Nach Strg+H, Rückgängig, Wiederholen blendet Strg+H die ganze Gruppe wieder ein.
func TestGruppeSchalterNachWiederholen(t *testing.T) {
	s := gruppiert(t)
	klick(s, f(15, 15), false)
	s.SchalteAusgeblendetGruppe(s.Auswahl)
	s.Rueckgaengig()
	s.Wiederholen()
	if !ausgeblendet(t, s, 1) || !ausgeblendet(t, s, 2) {
		t.Fatal("Wiederholen blendet nicht wieder aus")
	}
	s.SchalteAusgeblendetGruppe(s.Auswahl)
	if ausgeblendet(t, s, 1) || ausgeblendet(t, s, 2) {
		t.Errorf("Strg+H nach Wiederholen: %v %v", ausgeblendet(t, s, 1), ausgeblendet(t, s, 2))
	}
	if got := s.Gewaehlt(); !slices.Equal(got, []int{1, 2}) {
		t.Errorf("nach dem Einblenden gewählt %v, Soll die Gruppe", got)
	}
	// Das Gedächtnis gilt je Schalter: Strg+H, dann Strg+L sperrt nur die (ausgeblendete) Hauptauswahl.
	s.SchalteAusgeblendetGruppe(s.Auswahl)
	s.SchalteGesperrtGruppe(s.Auswahl)
	if o, _ := s.D.Objekt(2); o.Ebenenwerte().Gesperrt {
		t.Error("Strg+L nutzt das Gedächtnis von Strg+H")
	}
}

// F-966: Ein vorher einzeln ausgeblendetes Mitglied bleibt nach zweimal Strg+H ausgeblendet.
func TestGruppeSchalterLaesstEinzelnAusgeblendetes(t *testing.T) {
	s := dreiKaesten(t)
	s.WaehleAlle()
	s.Gruppieren()
	s.SchalteAusgeblendet(3)
	s.waehleNur(0)
	klick(s, f(15, 15), false)
	s.SchalteAusgeblendetGruppe(s.Auswahl)
	s.SchalteAusgeblendetGruppe(s.Auswahl)
	if ausgeblendet(t, s, 1) || ausgeblendet(t, s, 2) || !ausgeblendet(t, s, 3) {
		t.Errorf("nach zwei Drücken: %v %v %v", ausgeblendet(t, s, 1), ausgeblendet(t, s, 2), ausgeblendet(t, s, 3))
	}
}
