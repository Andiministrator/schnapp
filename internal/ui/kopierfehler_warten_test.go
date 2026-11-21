// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Prüfer-Tests aus Verify 5 (Fable, über 3a01c98): Schließen-Anfrage, die auf ein Kopieren wartet, neben
// gescheitertem Speichern, Hinweis, Fenster-Warnung und Signal (F-352, F-360).

import (
	"errors"
	"strings"
	"testing"
	"time"

	"gioui.org/layout"

	"github.com/Andiministrator/schnapp/internal/document"
)

func v5Voll(*document.Dokument) (Gespeichert, error) { return Gespeichert{}, errors.New("voll") }

func v5Hinweis(*document.Dokument) (Gespeichert, error) {
	return Gespeichert{Hinweis: "HINWEIS-NEU", Warnung: true}, nil
}

// v5NurKopieMerker: Dokument gespeichert, ein Kopieren läuft, eine Anfrage wartet darauf (schliessenNachKopie).
func v5NurKopieMerker(t *testing.T, e *editor) {
	t.Helper()
	e.s.Gespeichert()
	if e.schliessen() {
		t.Fatal("schloss trotz Kopieren")
	}
	if !e.schliessenNachKopie {
		t.Fatal("Vorbedingung: schliessenNachKopie fehlt")
	}
}

// (a) Kopierfrist: Nach dem gescheiterten Speichern läuft die Kopierfrist ab, ohne dass eine Antwort kam. Die Anfrage
// wird nicht von selbst neu gestellt; erst die Anfrage des Nutzers verwirft.
func TestKopierWartenFristKeinSelbstVerwerfen(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	e.speichern = v5Voll
	v4BeideMerker(t, e)
	warteSpeichern(t, e)
	e.kopierStart = time.Now().Add(-2 * kopierFrist) // Frist um, Antwort bleibt aus
	if e.nachKopieSchliessen() {
		t.Fatalf("Anfrage zur Kopierfrist von selbst neu gestellt; schliessen()=%v", e.schliessen())
	}
	if *zu != 0 || !e.s.Geaendert() || !strings.Contains(e.meldung.Text, "Close again to discard") {
		t.Errorf("zu=%d geändert=%v Meldung=%q", *zu, e.s.Geaendert(), e.meldung.Text)
	}
	if !e.schliessen() {
		t.Error("die Anfrage des Nutzers verwirft nicht")
	}
}

// (a) Reihenfolge umgekehrt: Die Kopierantwort kommt vor dem gescheiterten Speicher-Ende.
func TestKopierWartenKopieVorSpeicherFehler(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	e.speichern = v5Voll
	v4BeideMerker(t, e)
	v4KopieGelingt(e)
	if e.nachKopieSchliessen() {
		t.Fatal("stellte die Anfrage während des Speicherns neu")
	}
	warteSpeichern(t, e)
	if e.nachKopieSchliessen() || *zu != 0 {
		t.Fatalf("Anfrage von selbst neu gestellt oder geschlossen: zu=%d", *zu)
	}
	if !strings.Contains(e.meldung.Text, "Close again to discard") {
		t.Errorf("Meldung %q", e.meldung.Text)
	}
	if e.schliessenNachKopie || e.schliessenNachSpeichern || e.kopierFehlerHaelt {
		t.Errorf("Merker stehen: %v %v %v", e.schliessenNachKopie, e.schliessenNachSpeichern, e.kopierFehlerHaelt)
	}
	if !e.schliessen() {
		t.Error("die Anfrage des Nutzers verwirft nicht")
	}
}

// (c) Nach dem gescheiterten Speichern läuft das Kopieren noch: Die verwerfende Anfrage des Nutzers wartet auf das
// Kopieren (der Halter muss starten, F-281), danach schließt die neu gestellte Anfrage.
func TestKopierWartenVerwerfenWartetAufKopie(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	e.speichern = v5Voll
	v4BeideMerker(t, e)
	warteSpeichern(t, e)
	if e.schliessenNachKopie {
		t.Fatal("schliessenNachKopie steht nach dem Fehler noch")
	}
	if e.schliessen() {
		t.Fatal("verwerfende Anfrage schloss, obwohl das Kopieren läuft")
	}
	if !e.schliessenNachKopie || *zu != 0 {
		t.Fatalf("wartet nicht auf das Kopieren: Merker %v zu=%d", e.schliessenNachKopie, *zu)
	}
	v4KopieGelingt(e)
	if !e.nachKopieSchliessen() {
		t.Fatal("nach der Kopierantwort wird die (vom Nutzer bestätigte) Anfrage nicht neu gestellt")
	}
	if !e.schliessen() {
		t.Error("die neu gestellte Anfrage schließt nicht")
	}
}

// (c) Dasselbe nach einem Hinweis: Die nächste Anfrage wartet auf das laufende Kopieren und schließt danach.
func TestKopierWartenNachHinweisWartetAufKopie(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	e.speichern = v5Hinweis
	v4BeideMerker(t, e)
	warteSpeichern(t, e)
	if *zu != 0 || e.schliessenNachKopie || !strings.Contains(e.meldung.Text, "HINWEIS-NEU") {
		t.Fatalf("zu=%d Merker=%v Meldung=%q", *zu, e.schliessenNachKopie, e.meldung.Text)
	}
	if e.schliessen() {
		t.Fatal("schloss, obwohl das Kopieren läuft")
	}
	v4KopieGelingt(e)
	if !e.nachKopieSchliessen() || !e.schliessen() {
		t.Error("schließt nach der Kopierantwort nicht")
	}
}

// (c) Kopierfehler nach dem gescheiterten Speichern, auf den die verwerfende Anfrage wartet: einmal offen, Fehler in
// Leiste und Fehlerausgabe, die nächste Anfrage schließt.
func TestKopierWartenVerwerfenDannKopierfehler(t *testing.T) {
	e, zu := haltStart(t)
	aus := fehlerAbfangen(t)
	e.speichern = v5Voll
	v4BeideMerker(t, e)
	warteSpeichern(t, e)
	if e.schliessen() {
		t.Fatal("schloss, obwohl das Kopieren läuft")
	}
	haltKopierFehler(e)
	if e.nachKopieSchliessen() || *zu != 0 {
		t.Fatal("kein Halt nach Kopierfehler")
	}
	if !strings.Contains(e.meldung.Text, "kaputt") || strings.Count(aus.String(), "kaputt") != 1 {
		t.Errorf("Meldung=%q stderr=%q", e.meldung.Text, aus.String())
	}
	if !e.schliessen() {
		t.Error("nächste Anfrage schließt nicht")
	}
}

// (a) Strg+S statt Schließen: Anfrage wartet auf das Kopieren, Änderung, Strg+S scheitert. Die neu gestellte Anfrage
// nach der Kopierantwort verwirft nicht, sondern speichert erneut; scheitert auch das, steht die Verwerfen-Meldung,
// und erst die Anfrage des Nutzers verwirft.
func TestKopierWartenStrgSFehlerKeinSelbstVerwerfen(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	e.speichern = v5Voll
	v5NurKopieMerker(t, e)
	haltAendere(t, e)
	e.speichernJetzt()
	warteSpeichern(t, e)
	if strings.Contains(e.meldung.Text, "discard") || !strings.Contains(e.meldung.Text, "voll") {
		t.Fatalf("Meldung nach Strg+S %q", e.meldung.Text)
	}
	v4KopieGelingt(e)
	if !e.nachKopieSchliessen() {
		t.Fatal("Anfrage nach der Kopierantwort nicht neu gestellt (sie ist noch unbeantwortet)")
	}
	if e.schliessen() {
		t.Fatal("neu gestellte Anfrage schloss mit ungesicherter Änderung")
	}
	warteSpeichern(t, e)
	if *zu != 0 || !e.s.Geaendert() || !strings.Contains(e.meldung.Text, "Close again to discard") {
		t.Fatalf("zu=%d geändert=%v Meldung=%q", *zu, e.s.Geaendert(), e.meldung.Text)
	}
	if e.nachKopieSchliessen() {
		t.Error("stellte die Anfrage ein zweites Mal von selbst")
	}
	if !e.schliessen() {
		t.Error("die Anfrage des Nutzers verwirft nicht")
	}
}

// (a) Strg+S läuft, eine zweite Anfrage des Nutzers kommt während des Speicherns, es scheitert: Die wartende
// Kopier-Anfrage ist weg, die Kopierantwort stellt nichts neu.
func TestKopierWartenStrgSMitAnfrageFehler(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	frei := make(chan struct{})
	e.speichern = func(*document.Dokument) (Gespeichert, error) {
		<-frei
		return Gespeichert{}, errors.New("voll")
	}
	v5NurKopieMerker(t, e)
	haltAendere(t, e)
	e.speichernJetzt()
	if e.schliessen() {
		t.Fatal("schloss während des Speicherns")
	}
	close(frei)
	warteSpeichern(t, e)
	if e.schliessenNachKopie || !strings.Contains(e.meldung.Text, "Close again to discard") {
		t.Fatalf("Merker=%v Meldung=%q", e.schliessenNachKopie, e.meldung.Text)
	}
	v4KopieGelingt(e)
	if e.nachKopieSchliessen() || *zu != 0 || !e.s.Geaendert() {
		t.Errorf("selbst neu gestellt oder zu: zu=%d", *zu)
	}
}

// (a) fensterWarnen: Das Kopieren endet mit „nur das Fenster hält“. Nach dem gescheiterten Speichern stellt nichts die
// Anfrage neu; die Anfrage des Nutzers warnt einmal (F-295), die nächste schließt.
func TestKopierWartenFensterWarnungNachFehler(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	e.speichern = v5Voll
	v4BeideMerker(t, e)
	warteSpeichern(t, e)
	e.kopierAntwort <- kopierErgebnis{daten: []byte("png")}
	e.kopierenAbholen(layout.Context{})
	if !e.fensterHaeltKopie {
		t.Fatal("Vorbedingung: Fenster hält die Kopie nicht")
	}
	if e.nachKopieSchliessen() || *zu != 0 {
		t.Fatal("Anfrage von selbst neu gestellt")
	}
	if e.schliessen() {
		t.Fatal("Anfrage des Nutzers schließt ohne Fenster-Warnung")
	}
	if !strings.Contains(e.meldung.Text, "clipboard holds") {
		t.Errorf("Meldung %q", e.meldung.Text)
	}
	if !e.schliessen() {
		t.Error("nächste Anfrage schließt nicht")
	}
}

// (d) Signal nach dem gescheiterten Speichern (beide Merker): speichert noch einmal und schließt in jedem Fall.
func TestKopierWartenSignalNachFehler(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	e.speichern = v5Voll
	v4BeideMerker(t, e)
	warteSpeichern(t, e)
	e.nachSignal()
	if *zu != 0 {
		t.Fatal("schloss vor dem Speichern nach dem Signal")
	}
	warteSpeichern(t, e)
	if *zu != 1 || !e.erzwungen || e.schlussErr == nil {
		t.Errorf("zu=%d erzwungen=%v schlussErr=%v", *zu, e.erzwungen, e.schlussErr)
	}
}

// (d) Signal während des Speicherns bei beiden Merkern, das Speichern scheitert: Neuversuch, dann zu. Das neue
// Zurücksetzen greift im Erneut-Fall nicht ein.
func TestKopierWartenSignalWaehrendSpeichernFehler(t *testing.T) {
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
	v4BeideMerker(t, e)
	e.nachSignal()
	warteSpeichern(t, e)
	if *zu != 0 || !e.s.SpeichernLaeuft() {
		t.Fatalf("kein Neuversuch: zu=%d läuft=%v", *zu, e.s.SpeichernLaeuft())
	}
	warteSpeichern(t, e)
	if *zu != 1 || e.schlussErr != nil || e.s.Geaendert() {
		t.Errorf("zu=%d schlussErr=%v geändert=%v", *zu, e.schlussErr, e.s.Geaendert())
	}
}

// (d) Signal, während nur die Kopier-Anfrage wartet (nichts zu speichern): schließt sofort, zerstoert wartet das
// Kopieren ab.
func TestKopierWartenSignalNurKopieMerker(t *testing.T) {
	e, zu := haltStart(t)
	aus := fehlerAbfangen(t)
	v5NurKopieMerker(t, e)
	e.nachSignal()
	if *zu != 1 || !e.erzwungen {
		t.Fatalf("zu=%d erzwungen=%v", *zu, e.erzwungen)
	}
	e.kopierAntwort <- kopierErgebnis{err: errors.New("kaputt")}
	if err := e.zerstoert(nil); err != nil {
		t.Errorf("zerstoert: %v", err)
	}
	if strings.Count(aus.String(), "kaputt") != 1 {
		t.Errorf("stderr=%q", aus.String())
	}
}

// (d) Signal nach dem Hinweis-Halt (beide Merker): schließt sofort.
func TestKopierWartenSignalNachHinweis(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	e.speichern = v5Hinweis
	v4BeideMerker(t, e)
	warteSpeichern(t, e)
	e.nachSignal()
	if *zu != 1 || !e.erzwungen {
		t.Errorf("zu=%d erzwungen=%v", *zu, e.erzwungen)
	}
}

// F-353, Gegenprobe: Strg+S scheitert neben Kopierfehler und wartender Anfrage: Das Fenster bleibt offen, die Leiste
// nennt den Speicherfehler (er ist der wichtigere), der Kopierfehler steht auf der Fehlerausgabe, Merker verbraucht.
func TestKopierWartenStrgSFehlerNebenKopierfehler(t *testing.T) {
	e, zu := haltStart(t)
	aus := fehlerAbfangen(t)
	e.speichern = v5Voll
	v5NurKopieMerker(t, e)
	haltAendere(t, e)
	e.speichernJetzt()
	haltKopierFehler(e)
	warteSpeichern(t, e)
	if e.nachKopieSchliessen() || *zu != 0 {
		t.Fatalf("kein Halt: zu=%d", *zu)
	}
	if !strings.Contains(e.meldung.Text, "voll") || !e.meldung.Fehler || !strings.Contains(aus.String(), "kaputt") {
		t.Errorf("Meldung=%q stderr=%q", e.meldung.Text, aus.String())
	}
	if e.kopierFehlerHaelt || e.schliessenNachKopie {
		t.Errorf("Merker stehen: %v %v", e.kopierFehlerHaelt, e.schliessenNachKopie)
	}
}

// F-354, Gegenprobe: Auch die Antwort „nur das Fenster hält“ (daten) löscht den Merker eines früheren Kopierfehlers.
func TestKopierWartenZweitesKopierenFensterHaelt(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	v5NurKopieMerker(t, e)
	haltKopierFehler(e)
	e.kopierLaeuft, e.kopierStart = true, time.Now()
	if e.nachKopieSchliessen() {
		t.Fatal("stellte neu, obwohl ein Kopieren läuft")
	}
	e.kopierAntwort <- kopierErgebnis{daten: []byte("png")}
	e.kopierenAbholen(layout.Context{})
	if e.kopierFehlerHaelt {
		t.Error("Merker steht nach gelungenem Kopieren")
	}
	if !e.nachKopieSchliessen() {
		t.Fatal("Anfrage nicht neu gestellt")
	}
	if e.schliessen() || !strings.Contains(e.meldung.Text, "clipboard holds") {
		t.Errorf("Fenster-Warnung fehlt: zu=%d Meldung=%q", *zu, e.meldung.Text)
	}
	if !e.schliessen() {
		t.Error("nächste Anfrage schließt nicht")
	}
}

// Testlücke (Mutanten V13, V15): Strg+S gelingt, während eine Anfrage auf das Kopieren wartet und KEIN Kopierfehler
// vorliegt: „Saved“ erscheint, die Anfrage bleibt bestehen und schließt nach der Kopierantwort.
func TestKopierWartenStrgSNebenWartenderAnfrageMeldetSaved(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	v5NurKopieMerker(t, e)
	haltAendere(t, e)
	e.speichernJetzt()
	warteSpeichern(t, e)
	if !strings.Contains(e.meldung.Text, "Saved") {
		t.Errorf("Meldung %q, erwartet Saved", e.meldung.Text)
	}
	if !e.schliessenNachKopie || *zu != 0 {
		t.Fatalf("wartende Anfrage verloren: Merker %v zu=%d", e.schliessenNachKopie, *zu)
	}
	v4KopieGelingt(e)
	if !e.nachKopieSchliessen() || !e.schliessen() {
		t.Error("schließt nach der Kopierantwort nicht")
	}
}

// Testlücke (Mutanten V21, V24): Der Halt-Zweig verbraucht beide Merker sofort; ein späteres gewöhnliches Schließen mit
// Speichern schließt, statt noch einmal anzuhalten.
func TestKopierWartenHaltVerbrauchtBeideMerkerSofort(t *testing.T) {
	e, zu := haltStart(t)
	fehlerAbfangen(t)
	v4BeideMerker(t, e)
	haltKopierFehler(e)
	warteSpeichern(t, e)
	if e.kopierFehlerHaelt || e.schliessenNachKopie || e.schliessenNachSpeichern {
		t.Errorf("Merker stehen nach dem Halt: %v %v %v", e.kopierFehlerHaelt, e.schliessenNachKopie, e.schliessenNachSpeichern)
	}
	if e.nachKopieSchliessen() {
		t.Fatal("stellte die Anfrage neu")
	}
	haltAendere(t, e)
	if e.schliessen() {
		t.Fatal("schloss vor dem Speichern")
	}
	warteSpeichern(t, e)
	if *zu != 1 {
		t.Errorf("späteres gewöhnliches Schließen hält an: zu=%d Meldung=%q", *zu, e.meldung.Text)
	}
}
