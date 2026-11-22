// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Haltende Meldungen in der Statusleiste (F-356 bis F-359, Verify 5 nach Sprint 10) und die Mutanten Q1, Q7, Q9 aus der
// Sitzungs-QA vom 22.11.2026 (F-370). Die ersten vier Tests waren die roten Befund-Tests des Prüfers.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gioui.org/layout"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/portal"
	"github.com/Andiministrator/schnapp/internal/saveprogress"
)

// F-356: Nach „Could not save … Close again to discard the changes.“ verdrängt die Kopierantwort die Warnung nicht;
// die nächste Anfrage verwirft erst, nachdem die Warnung zu sehen war.
func TestHaltendeMeldungKopierantwortVerdraengtVerwerfenWarnungNicht(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	e.speichern = v5Voll
	v4BeideMerker(t, e)
	warteSpeichern(t, e)
	if !strings.Contains(e.meldung.Text, "Close again to discard") {
		t.Fatalf("Vorbedingung: Meldung %q", e.meldung.Text)
	}
	v4KopieGelingt(e)
	if e.nachKopieSchliessen() || *zu != 0 {
		t.Fatal("Vorbedingung: Fenster bleibt offen")
	}
	if !strings.Contains(e.meldung.Text, "discard") {
		t.Errorf("Fenster offen wegen des Speicherfehlers, Leiste zeigt aber %q", e.meldung.Text)
	}
}

// F-356, einfachste Form ohne wartende Kopier-Anfrage: Schließen speichert und scheitert, das laufende Kopieren
// antwortet danach.
func TestHaltendeMeldungVerwerfenWarnungEinfach(t *testing.T) {
	e, _ := haltStart(t)
	fehlerAbfangen(t)
	e.speichern = v5Voll
	if e.schliessen() {
		t.Fatal("schloss vor dem Speichern")
	}
	warteSpeichern(t, e)
	v4KopieGelingt(e)
	if !strings.Contains(e.meldung.Text, "discard") {
		t.Errorf("Leiste zeigt %q statt der Verwerfen-Warnung", e.meldung.Text)
	}
}

// F-356, Gegenprobe: Ändert der Nutzer nach der Warnung etwas, gilt sie nicht mehr; die Kopierantwort meldet sich wie
// sonst.
func TestHaltendeMeldungNachAenderungAbgeloest(t *testing.T) {
	e, _ := haltStart(t)
	fehlerAbfangen(t)
	e.speichern = v5Voll
	e.schliessen()
	warteSpeichern(t, e)
	haltAendere(t, e)
	v4KopieGelingt(e)
	if !strings.Contains(e.meldung.Text, "Copied to clipboard") {
		t.Errorf("Leiste zeigt %q, erwartet die Kopierbestätigung", e.meldung.Text)
	}
}

// F-356, Kopierfehler statt Erfolg: Ohne wartende Anfrage bleibt die Verwerfen-Warnung stehen, der Kopierfehler geht
// genau einmal auf die Fehlerausgabe.
func TestHaltendeMeldungKopierfehlerAufStderr(t *testing.T) {
	e, zu := haltStart(t)
	aus := fehlerAbfangen(t)
	e.speichern = v5Voll
	e.schliessen()
	warteSpeichern(t, e)
	haltKopierFehler(e)
	if !strings.Contains(e.meldung.Text, "discard") || strings.Count(aus.String(), "kaputt") != 1 {
		t.Errorf("Meldung=%q stderr=%q", e.meldung.Text, aus.String())
	}
	if e.kopierFehlerHaelt || e.nachKopieSchliessen() || *zu != 0 {
		t.Errorf("Halt-Merker %v oder zu=%d", e.kopierFehlerHaelt, *zu)
	}
	if !e.schliessen() {
		t.Error("die Anfrage des Nutzers verwirft nicht")
	}
}

// F-356 für einen Hinweis aus dem Schließen: Die Kopierantwort lässt ihn stehen.
func TestHaltendeMeldungHinweisBleibt(t *testing.T) {
	e, _ := haltStart(t)
	fehlerAbfangen(t)
	e.speichern = v5Hinweis
	e.schliessen()
	warteSpeichern(t, e)
	v4KopieGelingt(e)
	if !strings.Contains(e.meldung.Text, "HINWEIS-NEU") {
		t.Errorf("Leiste zeigt %q statt des Hinweises", e.meldung.Text)
	}
}

// F-357: Anfrage wartet auf das Kopieren, Änderung, Strg+S gelingt mit Hinweis. Die Kopierantwort stellt die alte
// Anfrage nicht neu, der Hinweis bleibt stehen.
func TestHaltendeMeldungStrgSHinweisSchliesstNichtSelbst(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	e.speichern = v5Hinweis
	v5NurKopieMerker(t, e)
	haltAendere(t, e)
	e.speichernJetzt()
	warteSpeichern(t, e)
	if !strings.Contains(e.meldung.Text, "HINWEIS-NEU") {
		t.Fatalf("Vorbedingung: Meldung %q", e.meldung.Text)
	}
	v4KopieGelingt(e)
	if e.nachKopieSchliessen() || *zu != 0 {
		t.Fatalf("Anfrage von selbst neu gestellt; zu=%d", *zu)
	}
	if !strings.Contains(e.meldung.Text, "HINWEIS-NEU") {
		t.Errorf("Hinweis verdrängt: %q", e.meldung.Text)
	}
	if !e.schliessen() {
		t.Error("die Anfrage des Nutzers schließt nicht")
	}
}

// F-358: Meldet das Speichern zwischendurch, dass es nicht antwortet, zeigt der Halt nach dem gelungenen Speichern
// wieder den Kopierfehler statt der überholten Hänge-Meldung.
func TestHaltendeMeldungHaltZeigtKopierfehlerNachHaengen(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	var versatz time.Duration
	alt := saveprogress.SetzeUhr(func() time.Time { return time.Now().Add(versatz) })
	t.Cleanup(func() { saveprogress.SetzeUhr(alt) })
	frei := make(chan struct{})
	e.speichern = func(*document.Dokument) (Gespeichert, error) {
		<-frei
		return Gespeichert{}, nil
	}
	e.schliessen()
	haltKopierFehler(e)
	versatz = time.Hour
	if !e.speichernHaengtMelden() {
		t.Fatal("Vorbedingung: Hängen nicht gemeldet")
	}
	close(frei)
	warteSpeichern(t, e)
	if *zu != 0 || e.s.Geaendert() {
		t.Fatalf("Vorbedingung: Halt nach gelungenem Speichern, zu=%d geändert=%v", *zu, e.s.Geaendert())
	}
	if !strings.Contains(e.meldung.Text, "kaputt") || !e.meldung.Fehler {
		t.Errorf("Fenster bleibt wegen des Kopierfehlers offen, Leiste zeigt aber %q", e.meldung.Text)
	}
}

// F-359: Kopierfehler, neues Strg+C im selben Bild, Änderung, die zweite Anfrage speichert und gelingt, während das
// zweite Kopieren läuft: Die Leiste meldet das Speichern, statt bei „Saving…“ stehen zu bleiben.
func TestHaltendeMeldungSavingBleibtNichtStehen(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	v5NurKopieMerker(t, e)
	haltKopierFehler(e)
	e.kopierLaeuft, e.kopierStart = true, time.Now() // Strg+C im selben Bild
	if e.nachKopieSchliessen() {
		t.Fatal("stellte neu, obwohl ein Kopieren läuft")
	}
	haltAendere(t, e)
	if e.schliessen() {
		t.Fatal("schloss vor dem Speichern")
	}
	warteSpeichern(t, e)
	if *zu != 0 || e.s.SpeichernLaeuft() || e.s.Geaendert() {
		t.Fatalf("Vorbedingung: gespeichert, wartet auf das Kopieren; zu=%d", *zu)
	}
	if !strings.Contains(e.meldung.Text, "Saved") {
		t.Errorf("Speichern ist fertig, Leiste zeigt %q", e.meldung.Text)
	}
	v4KopieGelingt(e)
	if !e.nachKopieSchliessen() || !e.schliessen() {
		t.Error("schließt nach der Antwort des zweiten Kopierens nicht")
	}
}

// F-370 Q1: Ist das Schließen schon freigegeben (erzwungen, hier nach dem Aufgeben eines hängenden Speicherns), merkt
// ein Kopierfehler keinen Halt mehr; der Fehler steht trotzdem auf der Fehlerausgabe.
func TestKopierfehlerNachFreigabeOhneHalt(t *testing.T) {
	e, _ := haltStart(t)
	aus := fehlerAbfangen(t)
	frei := make(chan struct{})
	t.Cleanup(func() { close(frei) })
	e.speichern = func(*document.Dokument) (Gespeichert, error) {
		<-frei
		return Gespeichert{}, nil
	}
	e.schliessen()
	e.speichernAufgeben()
	haltKopierFehler(e)
	if e.kopierFehlerHaelt || e.meldungHaelt != keinHalt {
		t.Errorf("Halt gemerkt trotz freigegebenem Schließen: %v %v", e.kopierFehlerHaelt, e.meldungHaelt)
	}
	if strings.Count(aus.String(), "kaputt") != 1 {
		t.Errorf("stderr=%q", aus.String())
	}
}

// F-370 Q7: Ein neues Kopieren, das wieder nur das Fenster hält, warnt beim Schließen erneut, auch wenn die Warnung zum
// vorigen Kopieren schon kam (F-295).
func TestFensterWarnungNachNeuemKopierenErneut(t *testing.T) {
	e, zu := editorMitFensterKopie(t)
	e.s.Gespeichert()
	if e.schliessen() {
		t.Fatal("Vorbedingung: erste Fenster-Warnung fehlt")
	}
	e.kopierLaeuft, e.kopierStart = true, time.Now()
	e.kopierAntwort <- kopierErgebnis{daten: []byte("png")}
	e.kopierenAbholen(layout.Context{})
	if e.schliessen() {
		t.Fatalf("schloss ohne Warnung zum neuen Kopieren; zu=%d", *zu)
	}
	if !strings.Contains(e.meldung.Text, "clipboard holds") {
		t.Errorf("Meldung %q", e.meldung.Text)
	}
	if !e.schliessen() {
		t.Error("nächste Anfrage schließt nicht")
	}
}

// F-370 Q9: Bleibt die Antwort des Kopierens über die Frist aus, stellt die Schleife die wartende Anfrage neu, und sie
// schließt (F-281).
func TestNachKopieSchliessenNachFrist(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	v5NurKopieMerker(t, e)
	if e.nachKopieSchliessen() {
		t.Fatal("stellte vor der Frist neu")
	}
	e.kopierStart = time.Now().Add(-2 * kopierFrist)
	if !e.nachKopieSchliessen() {
		t.Fatal("Anfrage nach abgelaufener Kopierfrist nicht neu gestellt")
	}
	if !e.schliessen() {
		t.Errorf("neu gestellte Anfrage schließt nicht; zu=%d", *zu)
	}
}

// Kopierantwort und Speicher-Ende im selben Durchlauf der Schleife (kopierenAbholen im Bild, speichernAbholen danach,
// nachKopieSchliessen zuletzt): Ohne Kopierfehler meldet das Strg+S „Saved“, die wartende Anfrage schließt danach
// (Testlücke K17/K18 nach F-359).
func TestHaltendeMeldungKopieDannSpeicherEndeMeldetSaved(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	frei := make(chan struct{})
	e.speichern = func(*document.Dokument) (Gespeichert, error) {
		<-frei
		return Gespeichert{}, nil
	}
	v5NurKopieMerker(t, e)
	haltAendere(t, e)
	e.speichernJetzt()
	v4KopieGelingt(e)
	close(frei)
	warteSpeichern(t, e)
	if !strings.Contains(e.meldung.Text, "Saved") {
		t.Errorf("Leiste zeigt %q, erwartet Saved", e.meldung.Text)
	}
	if !e.nachKopieSchliessen() || !e.schliessen() {
		t.Errorf("wartende Anfrage schließt nicht; zu=%d", *zu)
	}
}

// Ein neues Strg+C löst eine haltende Meldung ab (melde): Seine Antwort meldet sich wie sonst (Mutante N8).
func TestHaltendeMeldungNeuesKopierenMeldetSich(t *testing.T) {
	e, _ := haltStart(t)
	fehlerAbfangen(t)
	e.speichern = v5Hinweis
	e.schliessen()
	warteSpeichern(t, e)
	v4KopieGelingt(e)
	if !strings.Contains(e.meldung.Text, "HINWEIS-NEU") {
		t.Fatalf("Vorbedingung: Meldung %q", e.meldung.Text)
	}
	gtx := layout.Context{}
	e.kopieren(gtx)
	warteKopieren(t, e, gtx)
	if !strings.Contains(e.meldung.Text, "Copied to clipboard") {
		t.Errorf("Leiste zeigt %q, erwartet die Kopierbestätigung", e.meldung.Text)
	}
}

// V6-3, Gegenprobe (Mutanten A3, A11): Hält auch das spät antwortende Kopieren das Bild nur im Fenster, stimmt die
// Fenster-Warnung weiter und bleibt stehen.
func TestHaltendeMeldungFensterWarnungBleibtBeiFensterKopie(t *testing.T) {
	e, _ := editorMitFensterKopie(t)
	fehlerAbfangen(t)
	e.s.Gespeichert()
	e.kopierLaeuft, e.kopierStart = true, time.Now().Add(-2*kopierFrist)
	if e.schliessen() {
		t.Fatal("Vorbedingung: Fenster-Warnung fehlt")
	}
	e.kopierAntwort <- kopierErgebnis{daten: []byte("png")}
	e.kopierenAbholen(layout.Context{})
	if !e.fensterHaeltKopie || !strings.Contains(e.meldung.Text, "only while this window is open") {
		t.Errorf("Fenster hält die Kopie (%v), Leiste zeigt aber %q", e.fensterHaeltKopie, e.meldung.Text)
	}
}

// V6-1 für einen gelungenen Export (Mutante A13): „Exported to …“ verdrängt die Verwerfen-Warnung nicht, die Datei ist
// trotzdem geschrieben.
func TestHaltendeMeldungExportErfolgVerdraengtNicht(t *testing.T) {
	e, _ := haltStart(t)
	fehlerAbfangen(t)
	e.kopierLaeuft = false
	e.speichern = v5Voll
	e.schliessen()
	warteSpeichern(t, e)
	ziel := filepath.Join(t.TempDir(), "export.png")
	e.exportFertig(exportErgebnis{ergebnis: portal.Ergebnis{URI: "file://" + ziel, Filter: 0}})
	if _, err := os.Stat(ziel); err != nil {
		t.Fatalf("Export nicht geschrieben: %v", err)
	}
	if !strings.Contains(e.meldung.Text, "discard") {
		t.Errorf("Leiste zeigt %q statt der Verwerfen-Warnung", e.meldung.Text)
	}
}
