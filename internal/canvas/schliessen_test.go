// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"errors"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// verschiebe zieht den Kasten (10,10)–(50,50) um dx nach rechts.
func verschiebe(t *testing.T, s *Sitzung, dx float64) {
	t.Helper()
	s.Druecken(f(30, 30), TastePrimaer)
	s.Ziehen(f(30+dx, 30))
	s.Loslassen(f(30+dx, 30))
	if s.Zieht() {
		t.Fatal("Zug läuft noch")
	}
}

// schliesse führt eine Schließen-Anfrage auf dem Weg der Oberfläche aus (F-284): SchliessenAnfrage, dann die
// Speicherfunktion mit dem Abzug (hier nur als Zeitpunkt, gespeichert wird nichts), dann SpeichernEnde. Zurück kommt der
// Schluss und der Fehler der Speicherfunktion.
func schliesse(s *Sitzung, speichern func() (warnung bool, err error)) (Schluss, error) {
	z, abzug := s.SchliessenAnfrage()
	if abzug == nil {
		return z, nil
	}
	warnung, err := speichern()
	return s.SpeichernEnde(warnung, err).Schluss, err
}

// schliesseNachSignal führt ein Signal auf dem Weg der Oberfläche aus (F-284): SignalAnfrage, Speicherfunktion,
// SpeichernEnde. Ein weiterer Versuch (Ende.Erneut) nach einem gescheiterten Strg+S oder Schließen läuft ebenso. Zurück
// kommt der Fehler des letzten Versuchs.
func schliesseNachSignal(s *Sitzung, speichern func() error) error {
	_, abzug := s.SignalAnfrage()
	var err error
	for abzug != nil {
		err = speichern()
		abzug = s.SpeichernEnde(false, err).Erneut
	}
	return err
}

// F-284: Ein angenommenes Signal bleibt vorgemerkt, auch nachdem sein Speichern entschieden ist: Die Oberfläche fragt
// SignalVorgemerkt nach jedem Ereignis, um das Signal nicht noch einmal anzunehmen, und schließt in jedem Fall. Ein
// späteres Schließen kommt nicht mehr vor; der Test hält fest, was die Sitzung dann täte (kein Verwerfen vortäuschen,
// sondern ein weiterer Versuch als Signal), damit eine Änderung daran auffällt.
func TestSignalBleibtNachEntscheidungVorgemerkt(t *testing.T) {
	voll := errors.New("voll")
	s, _, _ := sitzung(t)
	verschiebe(t, s, 10)
	if err := schliesseNachSignal(s, func() error { return voll }); !errors.Is(err, voll) {
		t.Fatalf("Signal: %v", err)
	}
	if !s.SignalVorgemerkt() || s.SpeichernLaeuft() {
		t.Errorf("nach dem Signal: vorgemerkt %v, läuft %v (Soll true, false)", s.SignalVorgemerkt(), s.SpeichernLaeuft())
	}
	verschiebe(t, s, 10)
	z, abzug := s.SchliessenAnfrage()
	if z != SchlussWarten || abzug == nil {
		t.Fatalf("Schließen nach dem Signal: %v, Abzug %v", z, abzug != nil)
	}
	e := s.SpeichernEnde(false, voll)
	if e.Schluss == SchlussFehler || e.Erneut == nil {
		t.Errorf("Ende nach Signal: %+v (Soll weiterer Versuch als Signal, kein SchlussFehler)", e)
	}
}

// F-109: Geaendert folgt dem Verlaufsstand, nicht der bloßen Tatsache, dass etwas geschah.
func TestGeaendertFolgtVerlaufsstand(t *testing.T) {
	s, _, _ := sitzung(t)
	verschiebe(t, s, 10)
	if !s.Geaendert() {
		t.Fatal("nach Verschieben nicht geändert")
	}
	s.Rueckgaengig()
	if s.Geaendert() {
		t.Error("Verschieben → Rückgängig: Geaendert == true (Soll false)")
	}
	s.Wiederholen()
	if !s.Geaendert() {
		t.Error("Wiederholen: Geaendert == false (Soll true)")
	}
	// Gespeichert beim verschobenen Stand: Rückgängig weicht davon ab, Wiederholen kehrt zurück.
	s.Gespeichert()
	if s.Geaendert() {
		t.Error("nach Gespeichert geändert")
	}
	s.Rueckgaengig()
	if !s.Geaendert() {
		t.Error("Rückgängig hinter den gespeicherten Stand: Geaendert == false (Soll true)")
	}
	s.Wiederholen()
	if s.Geaendert() {
		t.Error("Wiederholen zum gespeicherten Stand: Geaendert == true (Soll false)")
	}
	// Ein anderer Schritt nach Rückgängig bekommt eine neue Kennung: gleich viele Schritte, aber nicht der gespeicherte
	// Stand.
	s.Rueckgaengig()
	verschiebe(t, s, 20)
	if !s.Geaendert() {
		t.Error("anderer Schritt anstelle des gespeicherten: Geaendert == false (Soll true)")
	}
	// Ein Zug ohne Wirkung (Klick, wählt den Kasten bei (30,10)–(70,50) aus) ändert nichts.
	s.Gespeichert()
	s.Druecken(f(50, 30), TastePrimaer)
	s.Loslassen(f(50, 30))
	if s.Geaendert() || s.Auswahl == 0 {
		t.Errorf("Klick ohne Bewegung: geändert %v, Auswahl %d", s.Geaendert(), s.Auswahl)
	}
	// Löschen ändert, Rückgängig hebt es auf.
	if !s.Loeschen() || !s.Geaendert() {
		t.Error("Löschen: nicht geändert")
	}
	s.Rueckgaengig()
	if s.Geaendert() {
		t.Error("Löschen → Rückgängig: geändert")
	}
	// Nur die Ansicht verschieben ändert nichts.
	s.Druecken(f(5, 90), TastePrimaer)
	s.Ziehen(f(40, 90))
	s.Loslassen(f(40, 90))
	if s.Geaendert() {
		t.Error("Ansicht verschoben: geändert")
	}
}

// F-107: Schließen ohne Änderung speichert nicht; mit Änderung wird gespeichert.
func TestSchliessenSpeichertNurGeaendertes(t *testing.T) {
	s, _, _ := sitzung(t)
	aufrufe := 0
	speichern := func() (bool, error) { aufrufe++; return false, nil }
	if z, err := schliesse(s, speichern); z != SchlussZu || err != nil || aufrufe != 0 {
		t.Errorf("ungeändert: %v, %v, %d Aufrufe (Soll SchlussZu, nil, 0)", z, err, aufrufe)
	}
	verschiebe(t, s, 10)
	if z, err := schliesse(s, speichern); z != SchlussZu || err != nil || aufrufe != 1 {
		t.Errorf("geändert: %v, %v, %d Aufrufe (Soll SchlussZu, nil, 1)", z, err, aufrufe)
	}
	if s.Geaendert() || s.Ungesichert() {
		t.Error("nach Speichern beim Schließen noch geändert")
	}
}

// F-107: Scheitert das Speichern beim Schließen, bleibt das Fenster offen; die zweite Anfrage ohne neue Änderung
// schließt (Verwerfen), eine neue Änderung hebt das wieder auf.
func TestSchliessenMitScheiterndemSpeichern(t *testing.T) {
	s, k, _ := sitzung(t)
	verschiebe(t, s, 10)
	kaputt := errors.New("Platte voll")
	aufrufe := 0
	scheitert := func() (bool, error) { aufrufe++; return false, kaputt }
	z, err := schliesse(s, scheitert)
	if z != SchlussFehler || !errors.Is(err, kaputt) || aufrufe != 1 {
		t.Fatalf("erste Anfrage: %v, %v, %d (Soll SchlussFehler, Platte voll, 1)", z, err, aufrufe)
	}
	if !s.Geaendert() {
		t.Error("nach gescheitertem Speichern nicht mehr geändert")
	}
	if s.Ungesichert() {
		t.Error("Ungesichert nach der Fehlermeldung (Soll: zum Verwerfen freigegeben)")
	}
	z, err = schliesse(s, scheitert)
	if z != SchlussZu || err != nil || aufrufe != 1 {
		t.Errorf("zweite Anfrage: %v, %v, %d (Soll SchlussZu, nil, kein weiterer Versuch)", z, err, aufrufe)
	}
	// Neue Änderung nach dem Fehler: wieder ungesichert, das nächste Schließen versucht es erneut.
	verschiebe(t, s, 5)
	if !s.Ungesichert() {
		t.Fatal("neue Änderung nicht ungesichert")
	}
	if z, _ := schliesse(s, scheitert); z != SchlussFehler || aufrufe != 2 {
		t.Errorf("nach neuer Änderung: %v, %d Aufrufe (Soll SchlussFehler, 2)", z, aufrufe)
	}
	// Gelingt es danach, ist alles gesichert.
	if z, err := schliesse(s, func() (bool, error) { return false, nil }); z != SchlussZu || err != nil {
		t.Errorf("gelungenes Speichern: %v, %v (Soll SchlussZu, nil)", z, err)
	}
	o, _ := s.D.Objekt(k)
	if b := o.(document.Kasten).Bereich; !nah(b.Min, document.Pt(25, 10)) {
		t.Errorf("Kasten bei %v (Soll (25,10))", b.Min)
	}
}

// F-154 (3): Jede neue Bearbeitung nach dem gescheiterten Speichern hebt die Verwerfen-Marke auf — auch wenn Rückgängig
// danach denselben Stand wieder herstellt. Das nächste Schließen versucht das Speichern erneut.
func TestVerwerfenMarkeNachBearbeitungUndRueckgaengig(t *testing.T) {
	bearbeitungen := map[string]func(s *Sitzung, k int){
		"Nummer anlegen": func(s *Sitzung, k int) {
			s.SetzeWerkzeug(WerkzeugNummer)
			s.Druecken(f(150, 80), TastePrimaer)
			s.Loslassen(f(150, 80))
		},
		"Verschieben": func(s *Sitzung, k int) { verschiebe(t, s, 7) },
		"Löschen":     func(s *Sitzung, k int) { s.Auswahl = k; s.Loeschen() },
		"Umfärben": func(s *Sitzung, k int) {
			s.Auswahl = k
			s.SetzeFarbe(Beschriftungsfarben[1])
		},
		"Text": func(s *Sitzung, k int) {
			s.SetzeWerkzeug(WerkzeugText)
			s.Druecken(f(100, 10), TastePrimaer)
			s.Loslassen(f(100, 10))
			s.Tippe("x")
			s.Enter(false)
		},
	}
	for name, bearbeite := range bearbeitungen {
		s, k, _ := sitzung(t)
		verschiebe(t, s, 10)
		aufrufe := 0
		fehl := func() (bool, error) { aufrufe++; return false, errors.New("voll") }
		if z, _ := schliesse(s, fehl); z != SchlussFehler {
			t.Fatalf("%s: Schluss %v", name, z)
		}
		bearbeite(s, k)
		if !s.Rueckgaengig() {
			t.Fatalf("%s: Rückgängig", name)
		}
		if !s.Ungesichert() {
			t.Errorf("%s: Marke überlebt Bearbeitung + Rückgängig (Ungesichert false)", name)
		}
		if z, _ := schliesse(s, fehl); z != SchlussFehler || aufrufe != 2 {
			t.Errorf("%s: Schluss %v, %d Aufrufe (Soll SchlussFehler, 2)", name, z, aufrufe)
		}
	}
	// Ohne Bearbeitung bleibt es dabei: Rückgängig und Wiederholen allein heben die Marke nicht auf.
	s, _, _ := sitzung(t)
	verschiebe(t, s, 10)
	fehl := func() (bool, error) { return false, errors.New("voll") }
	schliesse(s, fehl)
	s.Rueckgaengig()
	s.Wiederholen()
	if s.Ungesichert() {
		t.Error("Rückgängig + Wiederholen ohne Bearbeitung hebt die Marke auf")
	}
}

// F-107/F-116: Ein Hinweis beim Speichern (Ausweichen) hält das Fenster einmal offen; gespeichert ist trotzdem.
func TestSchliessenMitHinweis(t *testing.T) {
	s, _, _ := sitzung(t)
	verschiebe(t, s, 10)
	if z, err := schliesse(s, func() (bool, error) { return true, nil }); z != SchlussHinweis || err != nil {
		t.Fatalf("%v, %v (Soll SchlussHinweis, nil)", z, err)
	}
	if s.Geaendert() {
		t.Error("nach Speichern mit Hinweis noch geändert")
	}
	aufrufe := 0
	if z, _ := schliesse(s, func() (bool, error) { aufrufe++; return true, nil }); z != SchlussZu || aufrufe != 0 {
		t.Errorf("zweite Anfrage: %v, %d Aufrufe (Soll SchlussZu, 0)", z, aufrufe)
	}
}

// Ein laufender Zug wird beim Schließen abgebrochen und nicht gespeichert.
func TestSchliessenBrichtZugAb(t *testing.T) {
	s, k, _ := sitzung(t)
	s.Druecken(f(30, 30), TastePrimaer)
	s.Ziehen(f(60, 30))
	aufrufe := 0
	if z, _ := schliesse(s, func() (bool, error) { aufrufe++; return false, nil }); z != SchlussZu || aufrufe != 0 {
		t.Errorf("%v, %d Aufrufe (Soll SchlussZu, 0)", z, aufrufe)
	}
	o, _ := s.D.Objekt(k)
	if s.Zieht() || !nah(o.(document.Kasten).Bereich.Min, document.Pt(10, 10)) {
		t.Errorf("Zug nicht abgebrochen: zieht %v, Kasten %v", s.Zieht(), o.Grenzen())
	}
}
