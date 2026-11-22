// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Sitzungs-QA 22.11.2026 über 226a63f (Kritiker Fable, Q-1 und Q-2); beide Tests waren rot.

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/portal"
)

// Q-1: Die Fenster-Warnung steht („The clipboard holds the image only while this window is open …“). Danach meldet
// Gio, dass es die Zwischenablage nicht schreiben konnte (zwischenablageFehlt, F-177): Das Fenster hält nichts mehr
// (fensterHaeltKopie = false), die Warnung behauptet es aber weiter; der Fehler steht nur auf der Fehlerausgabe.
// Vor 226a63f ersetzte der Fehler die Warnung. Gegenstück zu V6-3 (überholte Fenster-Warnung).
func TestHaltendeMeldungZwischenablageFehlerLoestFensterWarnungAb(t *testing.T) {
	e, _ := editorMitFensterKopie(t)
	aus := fehlerAbfangen(t)
	e.s.Gespeichert()
	if e.schliessen() {
		t.Fatal("Vorbedingung: Fenster-Warnung hält nicht")
	}
	if e.meldungHaelt != haltFenster || !strings.Contains(e.meldung.Text, "only while this window is open") {
		t.Fatalf("Vorbedingung: Meldung %q, Art %v", e.meldung.Text, e.meldungHaelt)
	}
	zwischenablageFehlt.Store(true)
	t.Cleanup(func() { zwischenablageFehlt.Store(false) })
	if !e.zwischenablageFehltMelden() {
		t.Fatal("Vorbedingung: Marke nicht angenommen")
	}
	if e.fensterHaeltKopie {
		t.Fatal("Vorbedingung: Fenster hält die Kopie noch")
	}
	text, _, _ := canvas.StatusZeile(&e.meldung, e.s.D.Stand(), e.s.Werkzeug, e.hinweis(), false)
	if strings.Contains(text, "holds the image") {
		t.Errorf("Zwischenablage leer, Leiste behauptet weiter: %q (Fehlerausgabe: %q)", text, aus.String())
	}
}

// Q-2: Die Verwerfen-Warnung steht. Der Nutzer öffnet den Export-Dialog (nahe liegend nach gescheitertem Speichern) und
// bricht ihn ab: exportStarten leert die Meldung unmittelbar (e.meldung = canvas.Meldung{}), ohne meldungHaelt
// zurückzusetzen; die Leiste zeigt kein „discard“ mehr, die nächste Anfrage verwirft. Das ist die
// Widerlegungsbedingung von F-356 („Fenster offen wegen Speicherfehler, Statusleiste ohne discard“).
func TestHaltendeMeldungExportStartLaesstVerwerfenWarnung(t *testing.T) {
	e, _ := haltStart(t)
	fehlerAbfangen(t)
	e.kopierLaeuft = false
	e.speichern = v5Voll
	e.schliessen()
	warteSpeichern(t, e)
	if !strings.Contains(e.meldung.Text, "discard") {
		t.Fatalf("Vorbedingung: Meldung %q", e.meldung.Text)
	}
	// Kein echter Portal-Dialog: Der Sitzungsbus zeigt auf einen Sockel, den es nicht gibt.
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path="+filepath.Join(t.TempDir(), "kein-bus"))
	e.wecker = canvas.NeuerWecker(func() {}, time.Hour, time.Hour)
	t.Cleanup(e.wecker.Ende)
	e.exportStarten()
	select {
	case <-e.exportAntwort: // Antwort der Goroutine (kein Portal) verwerfen
	case <-time.After(10 * time.Second):
		t.Fatal("keine Antwort des Exports")
	}
	e.exportOffen = false
	e.exportFertig(exportErgebnis{err: portal.ErrAbgebrochen}) // der Nutzer bricht den Dialog ab
	if e.meldungHaelt == keinHalt {
		t.Fatal("Vorbedingung: Halt wurde ordentlich abgelöst")
	}
	if !e.s.Geaendert() {
		t.Fatal("Vorbedingung: Änderung noch ungesichert")
	}
	text, _, _ := canvas.StatusZeile(&e.meldung, e.s.D.Stand(), e.s.Werkzeug, e.hinweis(), false)
	if !strings.Contains(text, "discard") {
		t.Errorf("Fenster offen wegen des Speicherfehlers, Leiste zeigt nach abgebrochenem Export %q; meldungHaelt=%v", text, e.meldungHaelt)
	}
	if !e.schliessen() {
		t.Error("Vorbedingung: die nächste Anfrage verwirft")
	}
}
