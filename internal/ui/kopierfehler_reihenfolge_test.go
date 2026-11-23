// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Prüfer-Tests aus Verify 4 (Fable, nach ce586c6): Reihenfolgen des Kopierfehler-Halts, F-351 bis F-355.

import (
	"errors"
	"strings"
	"testing"
	"time"

	"gioui.org/layout"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/saveprogress"
)

// v4BeideMerker: Eine Anfrage hält wegen des Kopierens (schliessenNachKopie), nach einer Änderung startet eine zweite
// das Speichern (schliessenNachSpeichern).
func v4BeideMerker(t *testing.T, e *editor) {
	t.Helper()
	e.s.Gespeichert()
	if e.schliessen() {
		t.Fatal("schloss trotz Kopieren")
	}
	haltAendere(t, e)
	if e.schliessen() {
		t.Fatal("schloss vor dem Speichern")
	}
	if !e.schliessenNachKopie || !e.schliessenNachSpeichern {
		t.Fatalf("Vorbedingung: Merker %v %v", e.schliessenNachKopie, e.schliessenNachSpeichern)
	}
}

func v4KopieGelingt(e *editor) {
	e.kopierAntwort <- kopierErgebnis{}
	e.kopierenAbholen(layout.Context{})
}

// F-346: Nach dem Halt (Fenster einmal offen) schließt ein Signal sofort.
func TestKopierHaltSignalNachHaltSchliesst(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	e.schliessen()
	haltKopierFehler(e)
	warteSpeichern(t, e)
	if *zu != 0 || !strings.Contains(e.meldung.Text, "kaputt") {
		t.Fatalf("Halt fehlt: zu=%d Meldung=%q", *zu, e.meldung.Text)
	}
	e.nachSignal()
	if *zu != 1 || !e.erzwungen {
		t.Errorf("Signal nach dem Halt schließt nicht: zu=%d erzwungen=%v", *zu, e.erzwungen)
	}
}

// F-346: Signal vor dem Kopierfehler: kein Merker, Fehler auf der Fehlerausgabe, Fenster schließt.
func TestKopierHaltSignalVorKopierfehler(t *testing.T) {
	e, zu := haltStart(t)
	aus := fehlerAbfangen(t)
	e.schliessen()
	e.nachSignal()
	haltKopierFehler(e)
	if e.kopierFehlerHaelt {
		t.Error("Merker trotz Signal gesetzt")
	}
	if strings.Count(aus.String(), "kaputt") != 1 {
		t.Errorf("Fehlerausgabe %q", aus.String())
	}
	warteSpeichern(t, e)
	if *zu != 1 {
		t.Errorf("zu=%d", *zu)
	}
}

// F-346: Signal ohne Schließen-Anfrage, Kopierfehler während des Signal-Speicherns.
func TestKopierHaltNurSignalKopierfehler(t *testing.T) {
	e, zu := haltStart(t)
	aus := fehlerAbfangen(t)
	e.nachSignal()
	haltKopierFehler(e)
	warteSpeichern(t, e)
	if *zu != 1 || e.kopierFehlerHaelt || !strings.Contains(aus.String(), "kaputt") {
		t.Errorf("zu=%d haelt=%v stderr=%q", *zu, e.kopierFehlerHaelt, aus.String())
	}
}

// F-346: Beide Merker, Kopierfehler, Signal: schließt.
func TestKopierHaltSignalBeideMerker(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	v4BeideMerker(t, e)
	haltKopierFehler(e)
	e.nachSignal()
	warteSpeichern(t, e)
	if *zu != 1 {
		t.Errorf("zu=%d", *zu)
	}
	if e.nachKopieSchliessen() && *zu != 1 {
		t.Error("unerwartet")
	}
}

// F-346: Speichern scheitert immer; Schließen, Kopierfehler, Signal: zwei Läufe, dann schließt es mit Fehler.
func TestKopierHaltSignalErneutScheitert(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	e.speichern = func(*document.Dokument) (Gespeichert, error) { return Gespeichert{}, errors.New("voll") }
	e.schliessen()
	haltKopierFehler(e)
	e.nachSignal()
	warteSpeichern(t, e)
	if *zu != 0 {
		t.Fatalf("schloss vor dem Neuversuch (%d)", *zu)
	}
	warteSpeichern(t, e)
	if *zu != 1 || e.schlussErr == nil || e.kopierFehlerHaelt || e.schliessenNachSpeichern {
		t.Errorf("zu=%d schlussErr=%v haelt=%v nachSpeichern=%v", *zu, e.schlussErr, e.kopierFehlerHaelt, e.schliessenNachSpeichern)
	}
}

// F-347: Strg+S ohne Schließen, Kopierfehler: kein Merker, nichts auf der Fehlerausgabe.
func TestKopierHaltStrgSOhneSchliessenKeinHalt(t *testing.T) {
	e, zu := haltStart(t)
	aus := fehlerAbfangen(t)
	e.speichernJetzt()
	haltKopierFehler(e)
	if e.kopierFehlerHaelt || aus.Len() != 0 {
		t.Errorf("haelt=%v stderr=%q", e.kopierFehlerHaelt, aus.String())
	}
	warteSpeichern(t, e)
	if *zu != 0 || e.kopierFehlerHaelt {
		t.Errorf("zu=%d haelt=%v", *zu, e.kopierFehlerHaelt)
	}
}

// F-347: Halt, danach Änderung und gewöhnliches Schließen: schließt nach dem Speichern, „Saved“ erscheint beim Strg+S.
func TestKopierHaltNachHaltAllesGewoehnlich(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	e.schliessen()
	haltKopierFehler(e)
	warteSpeichern(t, e)
	if *zu != 0 || e.kopierFehlerHaelt || e.schliessenNachSpeichern || e.schliessenNachKopie {
		t.Fatalf("nach Halt: zu=%d Merker %v %v %v", *zu, e.kopierFehlerHaelt, e.schliessenNachSpeichern, e.schliessenNachKopie)
	}
	haltAendere(t, e)
	e.speichernJetzt()
	warteSpeichern(t, e)
	if !strings.Contains(e.meldung.Text, "Saved") {
		t.Errorf("Meldung %q", e.meldung.Text)
	}
	haltAendere(t, e)
	e.schliessen()
	warteSpeichern(t, e)
	if *zu != 1 {
		t.Errorf("zu=%d", *zu)
	}
}

// F-347: Halt und Titel: Der Titel trägt die Datei, die Fußzeile den Kopierfehler.
func TestKopierHaltHaltSetztTitel(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	e.speichern = func(*document.Dokument) (Gespeichert, error) { return Gespeichert{Datei: "neu.png"}, nil }
	e.schliessen()
	haltKopierFehler(e)
	warteSpeichern(t, e)
	if *zu != 0 || e.titel != "neu.png" || !strings.Contains(e.meldung.Text, "kaputt") || !e.meldung.Fehler {
		t.Errorf("zu=%d titel=%q Meldung=%q", *zu, e.titel, e.meldung.Text)
	}
}

// F-348 (b): Das zweite Kopieren scheitert ebenfalls: einmal offen, Fehler in der Leiste, die nächste Anfrage schließt.
func TestKopierHaltZweitesKopierenScheitert(t *testing.T) {
	e, zu := haltStart(t)
	aus := fehlerAbfangen(t)
	e.schliessen()
	haltKopierFehler(e)
	e.kopierLaeuft, e.kopierStart = true, time.Now()
	warteSpeichern(t, e)
	if *zu != 0 {
		t.Fatal("schloss vor der zweiten Kopierantwort")
	}
	haltKopierFehler(e)
	if e.nachKopieSchliessen() {
		t.Error("stellte die Anfrage neu")
	}
	if strings.Count(aus.String(), "kaputt") != 2 || !strings.Contains(e.meldung.Text, "kaputt") {
		t.Errorf("stderr=%q Meldung=%q", aus.String(), e.meldung.Text)
	}
	if e.kopierFehlerHaelt || e.schliessenNachKopie {
		t.Errorf("Merker stehen: %v %v", e.kopierFehlerHaelt, e.schliessenNachKopie)
	}
	if !e.schliessen() {
		t.Error("nächste Anfrage schließt nicht")
	}
}

// F-348 (a): Beide Merker, das Kopieren gelingt während des Speicherns: Das Speicher-Ende schließt, kein Halt.
func TestKopierHaltBeideMerkerKopieGelingt(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	v4BeideMerker(t, e)
	v4KopieGelingt(e)
	if e.nachKopieSchliessen() {
		t.Error("stellte die Anfrage während des Speicherns neu")
	}
	warteSpeichern(t, e)
	if *zu != 1 {
		t.Errorf("zu=%d Meldung=%q", *zu, e.meldung.Text)
	}
}

// Beide Merker, Kopierfehler, Speichern mit Hinweis im selben Durchlauf: offen, Merker danach weg.
func TestKopierHaltBeideMerkerHinweis(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	e.speichern = func(*document.Dokument) (Gespeichert, error) {
		return Gespeichert{Hinweis: "HINWEIS-NEU", Warnung: true}, nil
	}
	v4BeideMerker(t, e)
	haltKopierFehler(e)
	warteSpeichern(t, e)
	if e.nachKopieSchliessen() {
		t.Error("stellte die Anfrage neu")
	}
	if *zu != 0 || e.kopierFehlerHaelt || e.schliessenNachKopie || e.schliessenNachSpeichern {
		t.Errorf("zu=%d Merker %v %v %v", *zu, e.kopierFehlerHaelt, e.schliessenNachKopie, e.schliessenNachSpeichern)
	}
	if !e.schliessen() {
		t.Error("nächste Anfrage schließt nicht")
	}
}

// Halt, dann hält das Fenster noch eine ältere Kopie: zweite Anfrage warnt (F-295), dritte schließt.
func TestKopierHaltHaltDannFensterWarnung(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	e.fensterHaeltKopie = true
	e.schliessen()
	haltKopierFehler(e)
	warteSpeichern(t, e)
	if *zu != 0 || !strings.Contains(e.meldung.Text, "kaputt") {
		t.Fatalf("zu=%d Meldung=%q", *zu, e.meldung.Text)
	}
	if e.schliessen() {
		t.Error("zweite Anfrage schließt ohne Fenster-Warnung")
	}
	if !strings.Contains(e.meldung.Text, "clipboard holds") {
		t.Errorf("Meldung %q", e.meldung.Text)
	}
	if !e.schliessen() {
		t.Error("dritte Anfrage schließt nicht")
	}
}

// Zerstören ohne Anfrage (speichernAbwarten) nach Schließen und Kopierfehler: kehrt zurück, kein Fehler.
func TestKopierHaltZerstoertNachKopierfehler(t *testing.T) {
	e, _ := haltStart(t)
	aus := fehlerAbfangen(t)
	e.schliessen()
	haltKopierFehler(e)
	if err := e.zerstoert(nil); err != nil {
		t.Errorf("zerstoert: %v", err)
	}
	if e.s.SpeichernLaeuft() || e.s.Ungesichert() {
		t.Error("nicht gespeichert")
	}
	if strings.Count(aus.String(), "kaputt") != 1 {
		t.Errorf("stderr=%q", aus.String())
	}
}

// Hängen-Frist: Schließen, Kopierfehler, das Speichern hängt, zweite Anfrage gibt auf und schließt; die späte Antwort
// nimmt den Hänge-Fehler zurück.
func TestKopierHaltHaengenNachKopierfehler(t *testing.T) {
	e, _ := haltStart(t)
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
	if !e.kopierFehlerHaelt {
		t.Fatal("Vorbedingung: Merker fehlt")
	}
	versatz = time.Hour
	if !e.speichernHaengtMelden() {
		t.Fatal("Hängen nicht gemeldet")
	}
	if !e.schliessen() {
		t.Fatal("zweite Anfrage schließt nicht trotz hängendem Speichern")
	}
	if !e.erzwungen || !errors.Is(e.schlussErr, ErrSpeichernHaengt) {
		t.Errorf("erzwungen=%v schlussErr=%v", e.erzwungen, e.schlussErr)
	}
	close(frei)
	select {
	case r := <-e.speicherAntwort:
		e.speicherAntwort <- r
	case <-time.After(10 * time.Second):
		t.Fatal("keine Antwort")
	}
	if err := e.zerstoert(nil); err != nil {
		t.Errorf("zerstoert: %v", err)
	}
}

// F-351: Beide Merker, Kopierfehler und Speicher-Ende im selben Schleifendurchlauf (die Schleife ruft
// speichernAbholen vor nachKopieSchliessen): Der Halt beantwortet auch die wartende Anfrage, nachKopieSchliessen
// stellt sie nicht neu, das Fenster bleibt einmal offen.
func TestKopierHaltImSelbenDurchlauf(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	v4BeideMerker(t, e)
	haltKopierFehler(e)  // im Bild
	warteSpeichern(t, e) // speichernAbholen, gleicher Durchlauf
	if *zu != 0 || !strings.Contains(e.meldung.Text, "kaputt") {
		t.Fatalf("Halt fehlt schon am Speicher-Ende: zu=%d Meldung=%q", *zu, e.meldung.Text)
	}
	if e.nachKopieSchliessen() { // Ende desselben Durchlaufs
		t.Errorf("nachKopieSchliessen stellt die Anfrage neu; schliessen()=%v — das Fenster schließt sofort trotz Kopierfehler", e.schliessen())
	}
}

// F-352: Anfrage hält wegen des Kopierens, Änderung, zweite Anfrage speichert und scheitert („Close again to
// discard“), danach kommt die Kopierantwort: Die Anfrage wird nicht von selbst neu gestellt, sonst gälte sie als
// Bestätigung des Verwerfens und die Änderung ginge ohne Zutun verloren.
func TestKopierHaltKeinSelbstVerwerfenNachKopie(t *testing.T) {
	e, _ := haltStart(t)
	fehlerAbfangen(t)
	e.speichern = func(*document.Dokument) (Gespeichert, error) { return Gespeichert{}, errors.New("voll") }
	v4BeideMerker(t, e)
	warteSpeichern(t, e)
	if !strings.Contains(e.meldung.Text, "Close again to discard") {
		t.Fatalf("Meldung %q", e.meldung.Text)
	}
	v4KopieGelingt(e)
	if e.nachKopieSchliessen() {
		zu := e.schliessen() // die Schleife setzt ActionClose ab, Gio fragt schliessen
		t.Errorf("Anfrage von selbst neu gestellt; schliessen()=%v, geändert=%v — Verwerfen ohne zweite Anfrage des Nutzers", zu, e.s.Geaendert())
	}
}

// F-352, Hinweis: wie oben, das Speichern gelingt mit Hinweis. Die Kopierantwort danach stellt die Anfrage nicht neu,
// sonst schlösse das Fenster, bevor der Hinweis gelesen ist; die nächste Anfrage des Nutzers schließt.
func TestKopierHaltKeinSelbstSchliessenVorHinweis(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	e.speichern = func(*document.Dokument) (Gespeichert, error) {
		return Gespeichert{Hinweis: "HINWEIS-NEU", Warnung: true}, nil
	}
	v4BeideMerker(t, e)
	warteSpeichern(t, e)
	if *zu != 0 || !strings.Contains(e.meldung.Text, "HINWEIS-NEU") {
		t.Fatalf("zu=%d Meldung=%q", *zu, e.meldung.Text)
	}
	v4KopieGelingt(e)
	if e.nachKopieSchliessen() {
		t.Error("Anfrage von selbst neu gestellt, der Hinweis wäre ungelesen weg")
	}
	if !e.schliessen() {
		t.Error("nächste Anfrage schließt nicht")
	}
}

// F-353:Anfrage hält wegen des Kopierens, Änderung, Strg+S; Kopierfehler und Speicher-Ende im selben Durchlauf:
// Das Fenster bleibt wegen des Kopierfehlers offen, und „Saved“ verdrängt dessen Meldung nicht.
func TestKopierHaltSavedVerdraengtKopierfehlerNicht(t *testing.T) {
	e, zu := haltStart(t)
	aus := fehlerAbfangen(t)
	e.s.Gespeichert()
	if e.schliessen() {
		t.Fatal("schloss trotz Kopieren")
	}
	haltAendere(t, e)
	e.speichernJetzt()
	haltKopierFehler(e)
	warteSpeichern(t, e)
	if e.nachKopieSchliessen() || *zu != 0 {
		t.Fatalf("kein Halt: zu=%d", *zu)
	}
	if !strings.Contains(aus.String(), "kaputt") {
		t.Errorf("stderr=%q", aus.String())
	}
	if !strings.Contains(e.meldung.Text, "kaputt") {
		t.Errorf("Fenster bleibt wegen des Kopierfehlers offen, Meldung aber %q", e.meldung.Text)
	}
}

// F-354: Anfrage hält wegen des Kopierens; die Antwort ist ein Fehler, im selben Bild startet Strg+C ein neues
// Kopieren (kopierenAbholen läuft zu Beginn des Bilds, die Tasten danach). Gelingt das zweite, wird die Anfrage neu
// gestellt; der Merker aus dem ersten Fehler hält nicht mehr.
func TestKopierHaltZweitesKopierenOhneSpeichern(t *testing.T) {
	e, _ := haltStart(t)
	fehlerAbfangen(t)
	e.s.Gespeichert()
	if e.schliessen() {
		t.Fatal("schloss trotz Kopieren")
	}
	haltKopierFehler(e)
	e.kopierLaeuft, e.kopierStart = true, time.Now() // Strg+C im selben Bild
	if e.nachKopieSchliessen() {
		t.Fatal("stellte die Anfrage neu, obwohl ein Kopieren läuft")
	}
	v4KopieGelingt(e)
	if !e.nachKopieSchliessen() {
		t.Errorf("nach gelungenem zweiten Kopieren wird die Anfrage nicht neu gestellt; Meldung %q", e.meldung.Text)
	}
}
