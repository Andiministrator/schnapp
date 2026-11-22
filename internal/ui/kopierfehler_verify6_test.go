// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Verify 6 über b93b502 (haltende Meldungen; Prüfer Fable). Die Tests TestHaltendeMeldungV6* ohne „Soll“ waren die roten
// Befund-Tests V6-1 bis V6-3; die übrigen bestätigen das Soll.

import (
	"errors"
	"strings"
	"testing"
	"time"

	"gioui.org/layout"

	"github.com/Andiministrator/schnapp/internal/document"
)

// V6-1: Die Verwerfen-Warnung steht (haltend). Das laufende Kopieren antwortet mit Daten für das Fenster (kein Halter),
// Gio meldet danach, dass es die Zwischenablage nicht schreiben konnte (zwischenablageFehlt, F-177). Die Schleife ersetzt
// die Warnung durch „Could not copy: no clipboard is available.“; die nächste Anfrage verwirft ohne sichtbare Warnung.
func TestHaltendeMeldungV6ZwischenablageFehltVerdraengtVerwerfenWarnung(t *testing.T) {
	e, _ := haltStart(t)
	fehlerAbfangen(t)
	e.speichern = v5Voll
	e.schliessen()
	warteSpeichern(t, e)
	if !strings.Contains(e.meldung.Text, "discard") {
		t.Fatalf("Vorbedingung: Meldung %q", e.meldung.Text)
	}
	e.kopierAntwort <- kopierErgebnis{daten: []byte("png")}
	e.kopierenAbholen(layout.Context{})
	if !strings.Contains(e.meldung.Text, "discard") {
		t.Fatalf("Vorbedingung: Kopierantwort verdrängt schon: %q", e.meldung.Text)
	}
	zwischenablageFehlt.Store(true) // das täte der Gio-Patch
	t.Cleanup(func() { zwischenablageFehlt.Store(false) })
	if !e.zwischenablageFehltMelden() {
		t.Fatal("Vorbedingung: Marke nicht angenommen")
	}
	if !e.s.Geaendert() {
		t.Fatal("Vorbedingung: Änderung noch ungesichert")
	}
	if !strings.Contains(e.meldung.Text, "discard") {
		t.Errorf("Fenster offen wegen des Speicherfehlers, Leiste zeigt aber %q", e.meldung.Text)
	}
	if !e.schliessen() {
		t.Error("Vorbedingung: die nächste Anfrage verwirft")
	}
}

// V6-1, zweiter Weg: Der Export-Dialog ist offen, das Fenster bekommt eine Schließen-Anfrage, das Speichern scheitert.
// Die Antwort des Dialogs (hier ein Fehler) ersetzt die Verwerfen-Warnung.
func TestHaltendeMeldungV6ExportAntwortVerdraengtVerwerfenWarnung(t *testing.T) {
	e, _ := haltStart(t)
	fehlerAbfangen(t)
	e.kopierLaeuft = false
	e.exportOffen = true
	e.speichern = v5Voll
	e.schliessen()
	warteSpeichern(t, e)
	if !strings.Contains(e.meldung.Text, "discard") {
		t.Fatalf("Vorbedingung: Meldung %q", e.meldung.Text)
	}
	e.exportOffen = false
	e.exportFertig(exportErgebnis{err: errors.New("Dialog kaputt")})
	if !strings.Contains(e.meldung.Text, "discard") {
		t.Errorf("Fenster offen wegen des Speicherfehlers, Leiste zeigt aber %q", e.meldung.Text)
	}
	if !e.schliessen() {
		t.Error("Vorbedingung: die nächste Anfrage verwirft")
	}
}

// V6-2: Gewöhnliches Strg+S ohne jede Schließen-Anfrage gelingt mit einem Hinweis (etwa „Saved as a new capture …“,
// Warnung false). Scheitert danach das noch laufende Kopieren, steht der Fehler nur auf der Fehlerausgabe; im Fenster
// erfährt niemand, dass die Zwischenablage leer blieb. Der Hinweis sagt hier nichts über die nächste Schließen-Anfrage.
func TestHaltendeMeldungV6StrgSHinweisVerschlucktKopierfehler(t *testing.T) {
	e, _ := haltStart(t)
	aus := fehlerAbfangen(t)
	e.speichern = func(*document.Dokument) (Gespeichert, error) {
		return Gespeichert{Hinweis: "HINWEIS-NEU"}, nil
	}
	e.speichernJetzt()
	warteSpeichern(t, e)
	if e.schliessenNachKopie || e.schliessenNachSpeichern || e.s.Geaendert() {
		t.Fatal("Vorbedingung: keine Anfrage wartet, gespeichert")
	}
	haltKopierFehler(e)
	if !strings.Contains(e.meldung.Text, "kaputt") {
		t.Errorf("Kopierfehler nur auf der Fehlerausgabe (%q), Leiste zeigt %q", aus.String(), e.meldung.Text)
	}
}

// V6-3: Das Fenster hält eine ältere Kopie, ein neues Kopieren läuft schon über die Frist. Die Anfrage warnt (haltend).
// Antwortet das Kopieren doch noch und ein Halter hat das Bild übernommen, bleibt die Warnung stehen, obwohl sie nicht
// mehr stimmt (fensterHaeltKopie ist false).
func TestHaltendeMeldungV6FensterWarnungUeberholt(t *testing.T) {
	e, _ := editorMitFensterKopie(t)
	fehlerAbfangen(t)
	e.s.Gespeichert()
	e.kopierLaeuft, e.kopierStart = true, time.Now().Add(-2*kopierFrist)
	if e.schliessen() {
		t.Fatal("Vorbedingung: Fenster-Warnung fehlt")
	}
	if !strings.Contains(e.meldung.Text, "only while this window is open") {
		t.Fatalf("Vorbedingung: Meldung %q", e.meldung.Text)
	}
	v4KopieGelingt(e) // ein Halter hat das Bild
	if e.fensterHaeltKopie {
		t.Fatal("Vorbedingung: Halter hat übernommen")
	}
	if strings.Contains(e.meldung.Text, "only while this window is open") {
		t.Errorf("Halter hält das Bild, Leiste warnt weiter: %q", e.meldung.Text)
	}
}

// Soll: Ein Kopierfehler neben einer haltenden Meldung steht genau einmal auf der Fehlerausgabe, auch nach dem Zerstören.
func TestHaltendeMeldungV6SollKopierfehlerGenauEinmal(t *testing.T) {
	e, _ := haltStart(t)
	aus := fehlerAbfangen(t)
	e.speichern = v5Voll
	e.schliessen()
	warteSpeichern(t, e)
	haltKopierFehler(e)
	e.kopierenAbwarten()
	if n := strings.Count(aus.String(), "kaputt"); n != 1 {
		t.Errorf("%d Zeilen, erwartet 1: %q", n, aus.String())
	}
	if e.kopierFehlerText == "" {
		t.Error("kopierFehlerText leer")
	}
}

// Soll: Ein Signal schließt auch, wenn ein Strg+S mit Hinweis die wartende Kopier-Anfrage verwirft.
func TestHaltendeMeldungV6SollSignalNachStrgSHinweisSchliesst(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	e.speichern = v5Hinweis
	v5NurKopieMerker(t, e)
	haltAendere(t, e)
	e.speichernJetzt()
	e.nachSignal()
	warteSpeichern(t, e)
	if *zu != 1 {
		t.Errorf("Signal schließt nicht: zu=%d Meldung=%q", *zu, e.meldung.Text)
	}
}

// Soll (F-357, andere Reihenfolge): Die Kopierantwort kommt einen Durchlauf vor dem Speicher-Ende; die neu gestellte
// Anfrage wartet auf das Speichern, der Hinweis hält das Fenster einmal offen.
func TestHaltendeMeldungV6SollKopierantwortVorStrgSHinweis(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	frei := make(chan struct{})
	e.speichern = func(*document.Dokument) (Gespeichert, error) {
		<-frei
		return Gespeichert{Hinweis: "HINWEIS-NEU", Warnung: true}, nil
	}
	v5NurKopieMerker(t, e)
	haltAendere(t, e)
	e.speichernJetzt()
	v4KopieGelingt(e)
	if !e.nachKopieSchliessen() {
		t.Fatal("Anfrage nach der Kopierantwort nicht neu gestellt")
	}
	if e.schliessen() {
		t.Fatal("schloss während des Speicherns")
	}
	close(frei)
	warteSpeichern(t, e)
	if *zu != 0 || !strings.Contains(e.meldung.Text, "HINWEIS-NEU") {
		t.Errorf("zu=%d Meldung=%q", *zu, e.meldung.Text)
	}
	if !e.schliessen() {
		t.Error("nächste Anfrage schließt nicht")
	}
}

// Soll: Ein Hinweis ohne Warnung gilt nur bis zum Werkzeugwechsel; danach meldet sich die Kopierantwort. Eine Warnung
// (Fehler) bleibt über den Werkzeugwechsel stehen.
func TestHaltendeMeldungV6SollHinweisUndWerkzeugwechsel(t *testing.T) {
	for _, warnung := range []bool{false, true} {
		e, _ := haltStart(t)
		fehlerAbfangen(t)
		e.speichern = func(*document.Dokument) (Gespeichert, error) {
			return Gespeichert{Hinweis: "HINWEIS-NEU", Warnung: warnung}, nil
		}
		e.schliessen()
		warteSpeichern(t, e)
		e.s.Werkzeug++
		v4KopieGelingt(e)
		if got := strings.Contains(e.meldung.Text, "HINWEIS-NEU"); got != warnung {
			t.Errorf("Warnung=%v: Leiste zeigt %q", warnung, e.meldung.Text)
		}
	}
}

// Soll: Nach der Verwerfen-Warnung bestätigt die zweite Anfrage das Verwerfen, wartet aber auf das laufende Kopieren;
// dessen Antwort stellt die Anfrage neu, und sie schließt.
func TestHaltendeMeldungV6SollVerwerfenBestaetigtWartetAufKopie(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	e.speichern = v5Voll
	e.schliessen()
	warteSpeichern(t, e)
	if e.schliessen() {
		t.Fatal("schloss trotz laufendem Kopieren")
	}
	v4KopieGelingt(e)
	if !e.nachKopieSchliessen() || !e.schliessen() || *zu != 0 {
		t.Errorf("schließt nach der Kopierantwort nicht; zu=%d", *zu)
	}
}

// Soll: Scheitert das Kopieren, während die bestätigte Verwerfen-Anfrage darauf wartet, zeigt die Leiste den
// Kopierfehler, das Fenster bleibt einmal offen, die Zeile steht einmal auf der Fehlerausgabe.
func TestHaltendeMeldungV6SollVerwerfenBestaetigtKopierfehler(t *testing.T) {
	e, _ := haltStart(t)
	aus := fehlerAbfangen(t)
	e.speichern = v5Voll
	e.schliessen()
	warteSpeichern(t, e)
	e.schliessen()
	haltKopierFehler(e)
	if e.nachKopieSchliessen() {
		t.Error("stellte trotz Kopierfehler neu")
	}
	if !strings.Contains(e.meldung.Text, "kaputt") || strings.Count(aus.String(), "kaputt") != 1 {
		t.Errorf("Meldung=%q stderr=%q", e.meldung.Text, aus.String())
	}
	if !e.schliessen() {
		t.Error("nächste Anfrage schließt nicht")
	}
}
