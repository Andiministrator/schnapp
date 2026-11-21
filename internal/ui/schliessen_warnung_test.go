// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"testing"

	"gioui.org/io/system"
	"gioui.org/layout"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
)

// warnungKopie ist die Meldung, die das Schließen einmal anhält, wenn nur das Fenster das Bild trägt (F-295).
func warnungKopie() string {
	return i18n.T("The clipboard holds the image only while this window is open. Close again to discard it.")
}

// editorMitFensterKopie baut einen Editor ohne Fenster, dessen Zwischenablage nur das Fenster trägt (Rückfall ohne
// Halter: WAYLAND_SOCKET gesetzt), mit einer Änderung nach dem Kopieren. zu zählt die Fensteraktionen „schließen“.
func editorMitFensterKopie(t *testing.T) (e *editor, zu *int) {
	t.Helper()
	t.Setenv("WAYLAND_SOCKET", "1")
	e = testEditor(t)
	zu = new(int)
	e.perform = func(a system.Action) {
		if a == system.ActionClose {
			*zu++
		}
	}
	e.speichern = func(*document.Dokument) (Gespeichert, error) { return Gespeichert{}, nil }
	gtx := layout.Context{}
	e.kopieren(gtx)
	warteKopieren(t, e, gtx)
	if !e.fensterHaeltKopie {
		t.Fatal("Vorbedingung: Das Fenster hält die Kopie nicht")
	}
	if _, err := e.s.D.Hinzufuegen(document.Text{Ort: document.Pt(5, 15), Inhalt: "x", Schriftgrad: 12}); err != nil {
		t.Fatal(err)
	}
	if !e.s.Geaendert() {
		t.Fatal("Vorbedingung: Dokument nicht geändert")
	}
	return e, zu
}

// F-321: Auch wenn die Schließen-Anfrage erst speichert (Hauptablauf: Strg+C, dann Schließen mit ungesicherter
// Änderung), hält das Fenster einmal an und warnt; die zweite Anfrage schließt.
func TestSchliessenMitSpeichernWarntOhneHalter(t *testing.T) {
	e, zu := editorMitFensterKopie(t)
	if e.schliessen() {
		t.Fatal("Schließen mit ungesicherter Änderung schließt sofort")
	}
	warteSpeichern(t, e)
	if *zu != 0 {
		t.Errorf("Das Fenster schloss nach dem Speichern ohne Warnung (%d Schließen-Aktionen)", *zu)
	}
	if e.meldung.Text != warnungKopie() {
		t.Errorf("Meldung %q, erwartet die Warnung", e.meldung.Text)
	}
	if e.s.Geaendert() {
		t.Error("Der Stand ist nach dem Speichern nicht gesichert")
	}
	if !e.schliessen() {
		t.Error("Die zweite Schließen-Anfrage schließt nicht")
	}
}

// F-321, Gegenprobe: Ohne Kopie im Fenster schließt das Speichern-beim-Schließen ohne Warnung.
func TestSchliessenMitSpeichernOhneKopieSchliesst(t *testing.T) {
	e, zu := editorMitFensterKopie(t)
	e.fensterHaeltKopie = false
	if e.schliessen() {
		t.Fatal("Schließen mit ungesicherter Änderung schließt sofort")
	}
	warteSpeichern(t, e)
	if *zu != 1 {
		t.Errorf("%d Schließen-Aktionen, erwartet 1", *zu)
	}
}

// F-321: Ein Signal schließt in jedem Fall, auch mit Kopie im Fenster; die Warnung hielte das Beenden auf.
func TestSignalSchliesstTrotzFensterKopie(t *testing.T) {
	e, zu := editorMitFensterKopie(t)
	e.nachSignal()
	warteSpeichern(t, e)
	if *zu != 1 {
		t.Errorf("Signal: %d Schließen-Aktionen, erwartet 1 (Meldung %q)", *zu, e.meldung.Text)
	}
}
