// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"gioui.org/layout"
)

// F-329: Ein Signal schließt auch, wenn beim Speichern noch ein Kopieren läuft.
func TestSignalSchliesstTrotzLaufendemKopieren(t *testing.T) {
	e, zu := editorMitFensterKopie(t)
	e.fensterHaeltKopie, e.fensterWarnung = false, false
	e.kopierLaeuft, e.kopierStart = true, time.Now()
	e.schliessen()
	e.nachSignal()
	warteSpeichern(t, e)
	if *zu != 1 {
		t.Errorf("%d Schließen-Aktionen, erwartet 1", *zu)
	}
}

// F-329: Übernimmt ein Halter die Kopie, schließt die erneute Anfrage ohne Warnung.
func TestSchliessenMitSpeichernNachKopieMitHalter(t *testing.T) {
	e, zu := editorMitFensterKopie(t)
	e.fensterHaeltKopie, e.fensterWarnung = false, false
	e.kopierLaeuft, e.kopierStart = true, time.Now()
	e.schliessen()
	warteSpeichern(t, e)
	if *zu != 0 {
		t.Fatalf("schloss vor der Kopierantwort (%d)", *zu)
	}
	e.kopierAntwort <- kopierErgebnis{}
	e.kopierenAbholen(layout.Context{})
	if !e.nachKopieSchliessen() || !e.schliessen() {
		t.Errorf("erneute Anfrage schließt nicht; Meldung %q", e.meldung.Text)
	}
}

// fehlerAbfangen leitet die Fehlerausgabe des Editors in einen Puffer um.
func fehlerAbfangen(t *testing.T) *bytes.Buffer {
	t.Helper()
	var b bytes.Buffer
	alt := fehlerAusgabe
	fehlerAusgabe = &b
	t.Cleanup(func() { fehlerAusgabe = alt })
	return &b
}

// F-337: Scheitert das Kopieren, auf das ein Schließen nach dem Speichern wartete, bleibt das Fenster einmal offen,
// zeigt den Fehler und schreibt ihn auf die Fehlerausgabe; die nächste Anfrage schließt.
func TestKopierfehlerNachSpeichernHaeltEinmalOffen(t *testing.T) {
	e, zu := editorMitFensterKopie(t)
	e.fensterHaeltKopie, e.fensterWarnung = false, false
	e.kopierLaeuft, e.kopierStart = true, time.Now()
	aus := fehlerAbfangen(t)
	e.schliessen()
	warteSpeichern(t, e)
	if *zu != 0 {
		t.Fatalf("schloss vor der Kopierantwort (%d)", *zu)
	}
	e.kopierAntwort <- kopierErgebnis{err: errors.New("kaputt")}
	e.kopierenAbholen(layout.Context{})
	if e.nachKopieSchliessen() {
		t.Error("stellte die Anfrage nach dem Kopierfehler erneut")
	}
	if !strings.Contains(e.meldung.Text, "kaputt") {
		t.Errorf("Meldung %q ohne Fehler", e.meldung.Text)
	}
	if !strings.Contains(aus.String(), "kaputt") {
		t.Errorf("Fehlerausgabe %q ohne Fehler", aus.String())
	}
	if *zu != 0 {
		t.Fatalf("schloss trotz Kopierfehler (%d)", *zu)
	}
	if !e.schliessen() {
		t.Error("die nächste Anfrage schließt nicht")
	}
}

// F-337, Weg ohne Speichern (SchlussZu): derselbe Ablauf.
func TestKopierfehlerOhneSpeichernHaeltEinmalOffen(t *testing.T) {
	e, _ := editorMitFensterKopie(t)
	e.fensterHaeltKopie, e.fensterWarnung = false, false
	e.s.Gespeichert() // nichts zu speichern: SchlussZu
	e.kopierLaeuft, e.kopierStart = true, time.Now()
	aus := fehlerAbfangen(t)
	if e.schliessen() {
		t.Fatal("schloss vor der Kopierantwort")
	}
	e.kopierAntwort <- kopierErgebnis{err: errors.New("kaputt")}
	e.kopierenAbholen(layout.Context{})
	if e.nachKopieSchliessen() {
		t.Error("stellte die Anfrage nach dem Kopierfehler erneut")
	}
	if !strings.Contains(e.meldung.Text, "kaputt") || !strings.Contains(aus.String(), "kaputt") {
		t.Errorf("Meldung %q, Fehlerausgabe %q", e.meldung.Text, aus.String())
	}
	if !e.schliessen() {
		t.Error("die nächste Anfrage schließt nicht")
	}
}
