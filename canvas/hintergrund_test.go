// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"errors"
	"image"
	"image/color"
	"testing"

	"github.com/Andiministrator/schnapp/document"
)

var errPlatte = errors.New("Platte voll")

// Strg+S im Hintergrund (ADR-0020 Nachtrag 6): Ohne Änderung kein Abzug; mit Änderung ein Abzug des aktuellen Stands,
// ein zweites Speichern wird abgelehnt, bis die Antwort da ist. Gelungen gilt der Stand des Abzugs als gespeichert.
func TestSpeichernJetztImHintergrund(t *testing.T) {
	s, _, _ := sitzung(t)
	if _, err := s.SpeichernJetzt(); !errors.Is(err, ErrNichtsZuSpeichern) {
		t.Fatalf("ohne Änderung: %v", err)
	}
	verschiebe(t, s, 10)
	abzug, err := s.SpeichernJetzt()
	if err != nil || abzug == nil || abzug == s.D || abzug.Stand() != s.D.Stand() || !s.SpeichernLaeuft() {
		t.Fatalf("Abzug %v, %v, läuft %v", abzug, err, s.SpeichernLaeuft())
	}
	if _, err := s.SpeichernJetzt(); !errors.Is(err, ErrSpeichernLaeuft) {
		t.Errorf("zweites Speichern: %v", err)
	}
	if e := s.SpeichernEnde(false, nil); e.Schliessen || e.Erneut != nil {
		t.Errorf("Ende %+v", e)
	}
	if s.SpeichernLaeuft() || s.Geaendert() {
		t.Error("nach dem Ende läuft es noch oder gilt als geändert")
	}
	// Gescheitert: Der Stand bleibt ungesichert, das Fenster bleibt (ohne Schließen-Anfrage) unberührt.
	verschiebe(t, s, 10)
	if _, err := s.SpeichernJetzt(); err != nil {
		t.Fatal(err)
	}
	if e := s.SpeichernEnde(false, errPlatte); e.Schliessen {
		t.Errorf("Ende nach Fehler: %+v", e)
	}
	if !s.Geaendert() || !s.Ungesichert() {
		t.Error("gescheitertes Speichern gilt als gespeichert oder verworfen")
	}
}

// Während eines Ziehens speichert Strg+S nicht (wie bisher), eine laufende Texteingabe wird bestätigt.
func TestSpeichernJetztNichtImZug(t *testing.T) {
	s, _, _ := sitzung(t)
	s.Druecken(f(30, 30), TastePrimaer)
	s.Ziehen(f(40, 30))
	if _, err := s.SpeichernJetzt(); !errors.Is(err, ErrZugLaeuft) || s.SpeichernLaeuft() {
		t.Errorf("im Zug: %v", err)
	}
}

// Schließen ohne Änderung schließt sofort; mit Änderung beginnt ein Speichern, und die Antwort entscheidet wie
// bisher (F-107, ADR-0020 Nachtrag 2 und 4): zu, einmal offen mit Hinweis, offen mit Fehler — dann verwirft die
// nächste Anfrage.
func TestSchliessenAnfrageImHintergrund(t *testing.T) {
	s, _, _ := sitzung(t)
	if z, a := s.SchliessenAnfrage(); z != SchlussZu || a != nil {
		t.Fatalf("ohne Änderung: %v %v", z, a)
	}
	for _, fall := range []struct {
		name    string
		warnung bool
		err     error
		zu      bool
		schluss Schluss
	}{
		{"gelungen", false, nil, true, SchlussZu},
		{"Hinweis", true, nil, false, SchlussHinweis},
		{"Fehler", false, errPlatte, false, SchlussFehler},
	} {
		s, _, _ := sitzung(t)
		verschiebe(t, s, 10)
		z, a := s.SchliessenAnfrage()
		if z != SchlussWarten || a == nil || !s.SpeichernLaeuft() {
			t.Fatalf("%s: %v %v", fall.name, z, a)
		}
		e := s.SpeichernEnde(fall.warnung, fall.err)
		if e.Schliessen != fall.zu || e.Schluss != fall.schluss {
			t.Errorf("%s: Ende %+v", fall.name, e)
		}
		if fall.err != nil {
			// Die nächste Anfrage ohne neue Änderung verwirft.
			if z, a := s.SchliessenAnfrage(); z != SchlussZu || a != nil {
				t.Errorf("%s: zweite Anfrage %v %v", fall.name, z, a)
			}
		}
	}
}

// Eine Schließen-Anfrage während eines Strg+S wartet auf dessen Antwort und wird dann entschieden.
func TestSchliessenWaehrendStrgS(t *testing.T) {
	for _, fall := range []struct {
		name    string
		warnung bool
		err     error
		zu      bool
		schluss Schluss
	}{
		{"gelungen", false, nil, true, SchlussZu},
		{"Hinweis", true, nil, false, SchlussHinweis},
		{"Fehler", false, errPlatte, false, SchlussFehler},
	} {
		s, _, _ := sitzung(t)
		verschiebe(t, s, 10)
		if _, err := s.SpeichernJetzt(); err != nil {
			t.Fatal(err)
		}
		if z, a := s.SchliessenAnfrage(); z != SchlussWarten || a != nil {
			t.Fatalf("%s: Anfrage während Strg+S: %v %v", fall.name, z, a)
		}
		e := s.SpeichernEnde(fall.warnung, fall.err)
		if e.Schliessen != fall.zu || e.Schluss != fall.schluss || e.Erneut != nil {
			t.Errorf("%s: Ende %+v", fall.name, e)
		}
		// Danach: eine weitere Anfrage schließt (nach Hinweis gespeichert, nach Fehler Verwerfen bestätigt).
		if !fall.zu {
			if z, _ := s.SchliessenAnfrage(); z != SchlussZu {
				t.Errorf("%s: zweite Anfrage %v", fall.name, z)
			}
		}
	}
}

// Ein Signal speichert und schließt in jedem Fall; kommt es während eines Strg+S, das scheitert, versucht es ein
// weiteres Speichern (Ende.Erneut, auch nach einem gescheiterten Schließen).
func TestSignalImHintergrund(t *testing.T) {
	s, _, _ := sitzung(t)
	if zu, a := s.SignalAnfrage(); !zu || a != nil {
		t.Fatalf("ohne Änderung: %v %v", zu, a)
	}

	s, _, _ = sitzung(t)
	verschiebe(t, s, 10)
	zu, a := s.SignalAnfrage()
	if zu || a == nil {
		t.Fatalf("mit Änderung: %v %v", zu, a)
	}
	if e := s.SpeichernEnde(true, nil); !e.Schliessen || e.SignalFehler {
		t.Errorf("Signal gelungen (auch mit Hinweis zu): %+v", e)
	}

	s, _, _ = sitzung(t)
	verschiebe(t, s, 10)
	s.SignalAnfrage()
	if e := s.SpeichernEnde(false, errPlatte); !e.Schliessen || !e.SignalFehler {
		t.Errorf("Signal gescheitert: %+v", e)
	}

	// Während Strg+S: wartet; Strg+S scheitert → ein weiterer Versuch, danach zu.
	s, _, _ = sitzung(t)
	verschiebe(t, s, 10)
	if _, err := s.SpeichernJetzt(); err != nil {
		t.Fatal(err)
	}
	if zu, a := s.SignalAnfrage(); zu || a != nil || !s.SignalVorgemerkt() {
		t.Fatalf("Signal während Strg+S: %v %v", zu, a)
	}
	e := s.SpeichernEnde(false, errPlatte)
	if e.Schliessen || e.Erneut == nil || !s.SpeichernLaeuft() {
		t.Fatalf("nach gescheitertem Strg+S: %+v", e)
	}
	if e := s.SpeichernEnde(false, nil); !e.Schliessen || e.SignalFehler || s.Geaendert() {
		t.Errorf("zweiter Versuch: %+v", e)
	}

	// Signal nach einem gescheiterten Schließen (Verwerfen vorgemerkt): speichert trotzdem noch einmal.
	s, _, _ = sitzung(t)
	verschiebe(t, s, 10)
	s.SchliessenAnfrage()
	s.SpeichernEnde(false, errPlatte)
	if zu, a := s.SignalAnfrage(); zu || a == nil {
		t.Errorf("Signal nach gescheitertem Schließen: %v %v (Soll: neuer Versuch)", zu, a)
	}
}

// ADR-0022 Punkt 3: Nach einem gelungenen Speichern mit Schwärzung im Verlauf ist Rückgängig nicht mehr möglich — auch
// im Hintergrund, für genau den gespeicherten Stand. Scheitert es, bleibt der Verlauf.
func TestSpeichernImHintergrundLeertVerlaufNachSchwaerzung(t *testing.T) {
	for _, err := range []error{nil, errPlatte} {
		d, e := document.Neu(image.NewNRGBA(image.Rect(0, 0, 40, 40)))
		if e != nil {
			t.Fatal(e)
		}
		s := NeueSitzung(d)
		if e := d.Schwaerzen(image.Rect(0, 0, 10, 10), document.Schwaerzung{Farbe: color.NRGBA{A: 255}}); e != nil {
			t.Fatal(e)
		}
		if _, e := s.SpeichernJetzt(); e != nil {
			t.Fatal(e)
		}
		s.SpeichernEnde(false, err)
		if got := d.KannRueckgaengig(); got != (err != nil) {
			t.Errorf("Fehler %v: KannRueckgaengig %v", err, got)
		}
	}
}

// SpeichernEnde ohne laufendes Speichern tut nichts (doppelte Antwort, Fehler der Oberfläche).
func TestSpeichernEndeOhneSpeichern(t *testing.T) {
	s, _, _ := sitzung(t)
	verschiebe(t, s, 10)
	if e := s.SpeichernEnde(false, nil); e.Schliessen || !s.Geaendert() {
		t.Errorf("Ende ohne Speichern: %+v, geändert %v", e, s.Geaendert())
	}
}
