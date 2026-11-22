// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Kopierfehler-Halt (F-341) in allen Ausgängen des Speicherns: F-346 bis F-349 (Verify 3 nach Sprint 10).

import (
	"errors"
	"strings"
	"testing"
	"time"

	"gioui.org/layout"

	"github.com/Andiministrator/schnapp/internal/document"
)

func haltAendere(t *testing.T, e *editor) {
	t.Helper()
	if _, err := e.s.D.Hinzufuegen(document.Text{Ort: document.Pt(5, 15), Inhalt: "y", Schriftgrad: 12}); err != nil {
		t.Fatal(err)
	}
}

// haltStart: Editor mit Änderung, ein Kopieren läuft, ein Halter hat die Kopie (keine Fenster-Warnung).
func haltStart(t *testing.T) (*editor, *int) {
	e, zu := editorMitFensterKopie(t)
	e.fensterHaeltKopie, e.fensterWarnung = false, false
	e.kopierLaeuft, e.kopierStart = true, time.Now()
	return e, zu
}

func haltKopierFehler(e *editor) {
	e.kopierAntwort <- kopierErgebnis{err: errors.New("kaputt")}
	e.kopierenAbholen(layout.Context{})
}

// F-346: Schließen, Kopierfehler während des Speicherns, dann ein Signal: Das Signal schließt (ADR-0020 Nachtrag 2).
func TestSignalNachKopierfehlerSchliesst(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	e.schliessen()
	haltKopierFehler(e)
	e.nachSignal()
	warteSpeichern(t, e)
	if *zu != 1 {
		t.Errorf("%d Schließen-Aktionen nach Signal, erwartet 1; kopierFehlerHaelt=%v", *zu, e.kopierFehlerHaelt)
	}
}

// F-346 mit Erneut: Das Speichern scheitert erst und wird nach dem Signal wiederholt.
func TestSignalNachKopierfehlerMitErneut(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	n := 0
	e.speichern = func(*document.Dokument) (Gespeichert, error) {
		n++
		if n == 1 {
			return Gespeichert{}, errors.New("voll")
		}
		return Gespeichert{}, nil
	}
	e.schliessen()
	haltKopierFehler(e)
	e.nachSignal()
	warteSpeichern(t, e) // scheitert, Erneut
	warteSpeichern(t, e)
	if *zu != 1 {
		t.Errorf("%d Schließen-Aktionen nach Signal, erwartet 1 (Speicherläufe %d)", *zu, n)
	}
}

// F-347 (a, b), F-349: Scheitert das Speichern, sind beide Merker weg; ein späteres Strg+S meldet „Saved“, ein späteres
// gewöhnliches Schließen schließt nach dem Speichern sofort.
func TestKopierfehlerMerkerNachGescheitertemSpeichern(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	ok := false
	e.speichern = func(*document.Dokument) (Gespeichert, error) {
		if !ok {
			return Gespeichert{}, errors.New("voll")
		}
		return Gespeichert{}, nil
	}
	e.schliessen()
	haltKopierFehler(e)
	warteSpeichern(t, e)
	if e.schliessenNachSpeichern {
		t.Error("schliessenNachSpeichern steht nach gescheitertem Speichern noch")
	}
	if e.kopierFehlerHaelt {
		t.Error("kopierFehlerHaelt steht nach gescheitertem Speichern noch")
	}
	ok = true
	e.speichernJetzt()
	warteSpeichern(t, e)
	if !strings.Contains(e.meldung.Text, "Saved") {
		t.Errorf("Meldung nach gelungenem Strg+S %q, erwartet Saved to the library", e.meldung.Text)
	}
	haltAendere(t, e)
	if e.schliessen() {
		t.Fatal("schloss vor dem Speichern")
	}
	warteSpeichern(t, e)
	if *zu != 1 {
		t.Errorf("späteres gewöhnliches Schließen hält an (%d Aktionen), Meldung %q", *zu, e.meldung.Text)
	}
}

// F-347 (c): Speichern mit Hinweis nach einem Kopierfehler: Der Hinweis steht in der Statusleiste, der Merker ist weg,
// die nächste Anfrage schließt.
func TestKopierfehlerNebenSpeicherHinweis(t *testing.T) {
	e, zu := haltStart(t)
	aus := fehlerAbfangen(t)
	e.speichern = func(*document.Dokument) (Gespeichert, error) {
		return Gespeichert{Hinweis: "HINWEIS-NEU", Warnung: true}, nil
	}
	e.schliessen()
	haltKopierFehler(e)
	warteSpeichern(t, e)
	if *zu != 0 {
		t.Errorf("schloss trotz Hinweis (%d)", *zu)
	}
	if !strings.Contains(e.meldung.Text, "HINWEIS-NEU") {
		t.Errorf("Hinweis fehlt in der Statusleiste: %q", e.meldung.Text)
	}
	if !strings.Contains(aus.String(), "kaputt") {
		t.Errorf("Kopierfehler fehlt auf der Fehlerausgabe: %q", aus.String())
	}
	if e.kopierFehlerHaelt {
		t.Error("kopierFehlerHaelt bleibt nach SchlussHinweis stehen")
	}
	if !e.schliessen() {
		t.Error("nächste Anfrage schließt nicht")
	}
}

// F-348 (a): Eine Anfrage hält wegen des Kopierens, eine zweite startet das Speichern, das Kopieren scheitert währenddessen:
// nachKopieSchliessen verbraucht den Halt nicht, das Fenster bleibt nach dem Speichern einmal offen mit dem Fehler.
func TestKopierfehlerBeideMerker(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	e.s.Gespeichert()
	if e.schliessen() {
		t.Fatal("schloss trotz Kopieren")
	}
	haltAendere(t, e)
	if e.schliessen() {
		t.Fatal("schloss vor dem Speichern")
	}
	haltKopierFehler(e)
	if e.nachKopieSchliessen() { // die Schleife ruft das nach jedem Ereignis
		t.Error("stellte die Anfrage neu")
	}
	warteSpeichern(t, e)
	if *zu != 0 {
		t.Errorf("Fenster schloss nach dem Speichern, ohne einmal für den Kopierfehler offen zu bleiben (%d)", *zu)
	}
	if !strings.Contains(e.meldung.Text, "kaputt") {
		t.Errorf("Meldung %q, Kopierfehler verdrängt", e.meldung.Text)
	}
	if !e.schliessen() {
		t.Error("nächste Anfrage schließt nicht")
	}
}

// F-349: Nach gescheitertem Speichern beim Schließen ist ein späterer gewöhnlicher Kopierfehler kein Halt.
func TestGewoehnlicherKopierfehlerNachGescheitertemSpeichern(t *testing.T) {
	e, _ := haltStart(t)
	aus := fehlerAbfangen(t)
	e.kopierLaeuft = false
	e.speichern = func(*document.Dokument) (Gespeichert, error) { return Gespeichert{}, errors.New("voll") }
	e.schliessen()
	warteSpeichern(t, e)
	e.kopierLaeuft, e.kopierStart = true, time.Now()
	haltKopierFehler(e)
	// Das Kopieren ist hier ohne seine Meldung „Copying…“ gesetzt, die Verwerfen-Warnung steht also noch: Sie bleibt, und
	// der Kopierfehler geht genau einmal auf die Fehlerausgabe (F-356) — ein Halt ist er nicht.
	if e.kopierFehlerHaelt || strings.Count(aus.String(), "kaputt") != 1 || e.schliessenNachSpeichern {
		t.Errorf("gewöhnlicher Kopierfehler wird Halt: haelt=%v stderr=%q nachSpeichern=%v", e.kopierFehlerHaelt, aus.String(), e.schliessenNachSpeichern)
	}
	if !strings.Contains(e.meldung.Text, "discard") {
		t.Errorf("Verwerfen-Warnung verdrängt: %q", e.meldung.Text)
	}
	if !e.schliessen() {
		t.Error("zweite Anfrage verwirft nicht")
	}
}

// Strg+S läuft, Schließen kommt dazu, Kopierfehler, Speichern gelingt: einmal offen, dann zu; der Fehler steht genau
// einmal auf der Fehlerausgabe.
func TestKopierfehlerStrgSDannSchliessen(t *testing.T) {
	e, zu := haltStart(t)
	aus := fehlerAbfangen(t)
	e.speichernJetzt()
	if e.schliessen() {
		t.Fatal("schloss")
	}
	haltKopierFehler(e)
	warteSpeichern(t, e)
	if *zu != 0 || strings.Count(aus.String(), "kaputt") != 1 || !strings.Contains(e.meldung.Text, "kaputt") {
		t.Errorf("zu=%d stderr=%q Meldung=%q", *zu, aus.String(), e.meldung.Text)
	}
	if e.kopierFehlerHaelt || e.schliessenNachSpeichern {
		t.Errorf("Merker stehen: %v %v", e.kopierFehlerHaelt, e.schliessenNachSpeichern)
	}
	if !e.schliessen() {
		t.Error("nächste Anfrage schließt nicht")
	}
	e.kopierenAbwarten()
	if strings.Count(aus.String(), "kaputt") != 1 {
		t.Errorf("doppelte Zeile: %q", aus.String())
	}
}

// F-348 (b): Nach einem Kopierfehler kopiert der Nutzer erneut, das Speichern endet währenddessen, das zweite Kopieren
// gelingt: Die Anfrage wird danach neu gestellt, kein Halt ohne Fehler.
func TestZweitesKopierenGelingtNachKopierfehler(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	e.schliessen()
	haltKopierFehler(e)
	e.kopierLaeuft, e.kopierStart = true, time.Now() // zweites Kopieren läuft
	warteSpeichern(t, e)
	if *zu != 0 {
		t.Fatalf("schloss vor der zweiten Kopierantwort")
	}
	e.kopierAntwort <- kopierErgebnis{}
	e.kopierenAbholen(layout.Context{})
	if !e.nachKopieSchliessen() {
		t.Errorf("nach gelungenem zweiten Kopieren wird die Anfrage nicht neu gestellt; Meldung %q", e.meldung.Text)
	}
}

// F-349: kopierenAbwarten (Zerstören des Fensters) schreibt einen Kopierfehler und ein nur vom Fenster gehaltenes Bild
// auf die umleitbare Fehlerausgabe.
func TestKopierenAbwartenMeldetAufFehlerausgabe(t *testing.T) {
	for _, f := range []struct {
		name   string
		r      kopierErgebnis
		muster string
	}{
		{"Fehler", kopierErgebnis{err: errors.New("kaputt")}, "kaputt"},
		{"nur Fenster", kopierErgebnis{daten: []byte{1}}, "clipboard may be empty"},
	} {
		t.Run(f.name, func(t *testing.T) {
			e, _ := haltStart(t)
			aus := fehlerAbfangen(t)
			e.kopierAntwort <- f.r
			e.kopierenAbwarten()
			if !strings.Contains(aus.String(), f.muster) {
				t.Errorf("Fehlerausgabe %q ohne %q", aus.String(), f.muster)
			}
			if e.kopierLaeuft {
				t.Error("kopierLaeuft steht noch")
			}
		})
	}
}
