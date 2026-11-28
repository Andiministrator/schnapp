// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

// Feld Schatten und Gruppen-Schalter im Zusammenspiel (Sitzungs-Kritik 28.11., F-970 bis F-973).

import (
	"image/color"
	"slices"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

var (
	sRot   = color.NRGBA{R: 200, A: 255}
	sGruen = color.NRGBA{G: 200, A: 255}
)

func ebenenZustand(s *Sitzung, id int) string {
	o, ok := s.D.Objekt(id)
	if !ok {
		return "fehlt"
	}
	e := o.Ebenenwerte()
	r := "sichtbar"
	if e.Ausgeblendet {
		r = "ausgeblendet"
	}
	if e.Gesperrt {
		r += "+gesperrt"
	}
	return r
}

// F-970: Feld Schatten bei mehreren gewählt (Hauptauswahl Text ohne Schatten), Texteingabe an der Hauptauswahl, Farbe,
// bestätigen: Kehrt das Feld Schatten ohne Klick zurück?
func TestFeldSchattenNachTexteingabe(t *testing.T) {
	s := gemischt(t)
	if err := s.D.Ersetzen(mitSchatten(objekt(t, s, 2), document.StandardSchatten(1))); err != nil {
		t.Fatal(err)
	}
	s.setzeAuswahl(3, []int{2}) // Text (Haupt, ohne Schatten), Kasten mit Schatten
	if !s.SetzeFeld(FeldSchatten) {
		t.Fatal("Feld Schatten nicht wählbar")
	}
	if !s.Enter(false) || !s.TextLaeuft() {
		t.Fatal("keine Texteingabe")
	}
	t.Logf("in der Texteingabe: Feld %v", s.AktivesFeld())
	s.SetzeFarbe(sRot) // färbt den Text (Vordergrund)
	s.TextBestaetigen()
	t.Logf("nach Bestätigen: gewählt %v, Feld %v", s.Gewaehlt(), s.AktivesFeld())
	if s.AktivesFeld() == FeldSchatten {
		t.Errorf("Feld Schatten kehrt nach der Texteingabe ohne Klick zurück")
	}
	vor := objekt(t, s, 2).Ebenenwerte().Schatten.Farbe
	s.SetzeFarbe(sGruen)
	if nach := objekt(t, s, 2).Ebenenwerte().Schatten.Farbe; nach != vor {
		t.Errorf("die nächste Farbwahl färbt den Schatten des Kastens: %v -> %v", vor, nach)
	}
}

// F-970: Feld Schatten bei mehreren, dann auf die Hauptauswahl allein verkleinert, gefärbt, Schatten eingeschaltet.
func TestFeldSchattenNachVerkleinern(t *testing.T) {
	s := gemischt(t)
	if err := s.D.Ersetzen(mitSchatten(objekt(t, s, 2), document.StandardSchatten(1))); err != nil {
		t.Fatal(err)
	}
	s.setzeAuswahl(1, []int{2}) // Pfeil (Haupt, ohne Schatten), Kasten mit Schatten
	if !s.SetzeFeld(FeldSchatten) {
		t.Fatal("Feld Schatten nicht wählbar")
	}
	s.SchalteGewaehlt(2) // Kasten heraus: Pfeil allein
	if got := s.Gewaehlt(); !slices.Equal(got, []int{1}) {
		t.Fatalf("gewählt %v", got)
	}
	s.SetzeFarbe(sRot) // färbt den Pfeil; gibt das Feld Schatten auf?
	t.Logf("nach Färben allein: feld=%v feldSchattenID=%d", s.feld, s.feldSchattenID)
	s.SchalteSchatten()
	if !objekt(t, s, 1).Ebenenwerte().Schatten.An {
		t.Fatal("Schatten nicht an")
	}
	if s.AktivesFeld() == FeldSchatten {
		t.Errorf("Feld Schatten kehrt am Pfeil ohne Klick zurück")
	}
}

// F-972: Gruppe {1,2} + Fremdtext 3; Strg+H; Texteingabe am Fremdtext, leeren; Strg+H.
func TestSchalterInTexteingabeLeer(t *testing.T) {
	s := gemischt(t)
	s.setzeAuswahl(2, []int{1})
	if !s.Gruppieren() {
		t.Fatal("Gruppieren")
	}
	s.setzeAuswahl(1, []int{2, 3})
	s.SchalteAusgeblendetGruppe(s.Auswahl)
	t.Logf("nach Strg+H: Haupt %d gewählt %v; 1 %s, 2 %s", s.Auswahl, s.Gewaehlt(), ebenenZustand(s, 1), ebenenZustand(s, 2))
	if !s.Enter(false) || !s.TextLaeuft() {
		t.Fatal("keine Texteingabe")
	}
	for i := 0; i < 5; i++ {
		s.TextZurueck()
	}
	ok := s.SchalteAusgeblendetGruppe(s.Auswahl)
	t.Logf("Strg+H in leerer Texteingabe: ok=%v Haupt %d gewählt %v; 1 %s, 2 %s, 3 %s", ok, s.Auswahl, s.Gewaehlt(),
		ebenenZustand(s, 1), ebenenZustand(s, 2), ebenenZustand(s, 3))
	if _, da := s.D.Objekt(s.Auswahl); s.Auswahl != 0 && !da {
		t.Errorf("Hauptauswahl %d zeigt auf ein Objekt, das es nicht mehr gibt (gewählt %v)", s.Auswahl, s.Gewaehlt())
	}
}

// F-972: wie K-C, aber Text bleibt (ein Zeichen getippt): Stellt der zweite Druck die Auswahl von vorher her?
func TestSchalterInTexteingabe(t *testing.T) {
	s := gemischt(t)
	s.setzeAuswahl(2, []int{1})
	if !s.Gruppieren() {
		t.Fatal("Gruppieren")
	}
	s.setzeAuswahl(1, []int{2, 3})
	s.SchalteAusgeblendetGruppe(s.Auswahl)
	if !s.Enter(false) || !s.TextLaeuft() {
		t.Fatal("keine Texteingabe")
	}
	s.Tippe("x")
	s.SchalteAusgeblendetGruppe(s.Auswahl)
	t.Logf("Haupt %d gewählt %v; 1 %s, 2 %s", s.Auswahl, s.Gewaehlt(), ebenenZustand(s, 1), ebenenZustand(s, 2))
	if got := s.Gewaehlt(); !slices.Equal(got, []int{1, 2, 3}) || s.Auswahl != 1 {
		t.Errorf("Auswahl nach dem Zurückschalten aus der Texteingabe: %v Haupt %d, Soll [1 2 3] Haupt 1", got, s.Auswahl)
	}
}

// F-971: Anker der Ebenenliste nach einer Ebenenänderung ohne Auswahländerung (Umbenennen).
func TestAnkerNachUmbenennen(t *testing.T) {
	s := dreiKaesten(t)
	s.WaehleEbene(1)
	s.WaehleEbeneMehr(3, false) // 1..3, Anker 1
	if got := s.Gewaehlt(); !slices.Equal(got, []int{1, 2, 3}) {
		t.Fatalf("Bereich %v", got)
	}
	if !s.Umbenennen(2, "x") {
		t.Fatal("Umbenennen")
	}
	if got := s.Gewaehlt(); !slices.Equal(got, []int{1, 2, 3}) {
		t.Fatalf("Auswahl nach Umbenennen %v", got)
	}
	s.WaehleEbeneMehr(2, false)
	if got := s.Gewaehlt(); !slices.Equal(got, []int{1, 2}) {
		t.Errorf("Umschalt+Klick nach Umbenennen: %v, Soll [1 2] ab dem Anker 1", got)
	}
}

// F-971: Gedächtnis des Schalters überlebt eine Stiländerung und ein Gruppieren-Rückgängig? Strg+H an Gruppe + Fremd,
// dann Deckkraft des Fremdobjekts über die Leiste (aendereEbene bereinigt die Auswahl), dann Strg+H.
func TestSchalterNachDeckkraft(t *testing.T) {
	s := gruppiert(t)
	s.setzeAuswahl(1, []int{2, 3})
	s.SchalteAusgeblendetGruppe(s.Auswahl)
	s.SetzeDeckkraft(s.Auswahl, 50)
	t.Logf("Menü Gruppe nach Deckkraft: %v", s.GruppeSchaltet(SchalterAusblenden))
	s.SchalteAusgeblendetGruppe(s.Auswahl)
	t.Logf("1 %s, 2 %s, 3 %s", ebenenZustand(s, 1), ebenenZustand(s, 2), ebenenZustand(s, 3))
	if ebenenZustand(s, 1) != "sichtbar" || ebenenZustand(s, 3) != "sichtbar" {
		t.Errorf("nach Deckkraft am Fremdobjekt schaltet Strg+H das Fremdobjekt statt der Gruppe: 1 %s, 3 %s",
			ebenenZustand(s, 1), ebenenZustand(s, 3))
	}
}

// F-973: Gruppe + Fremdobjekt: Strg+H, Strg+L, Strg+H, Strg+H, Strg+L — kommt die Gruppe über die Tasten zurück?
func TestSchalterWechsel(t *testing.T) {
	s := gruppiert(t)
	s.setzeAuswahl(1, []int{2, 3})
	s.SchalteAusgeblendetGruppe(s.Auswahl)
	for i, a := range []string{"L", "H", "H", "L", "H"} {
		if a == "L" {
			s.SchalteGesperrtGruppe(s.Auswahl)
		} else {
			s.SchalteAusgeblendetGruppe(s.Auswahl)
		}
		t.Logf("%d Strg+%s: Haupt %d gewählt %v; 1 %s, 2 %s, 3 %s", i, a, s.Auswahl, s.Gewaehlt(), ebenenZustand(s, 1),
			ebenenZustand(s, 2), ebenenZustand(s, 3))
	}
	if ausgeblendet(t, s, 1) || ausgeblendet(t, s, 2) {
		t.Errorf("Gruppe bleibt ausgeblendet: 1 %s, 2 %s", ebenenZustand(s, 1), ebenenZustand(s, 2))
	}
}
