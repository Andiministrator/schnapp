// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Tests aus Verify 7 (Fable, V7-8, F-385): Die Rettungskopie hängt an den richtigen Stellen des Editors (Aufbau, Schleife,
// Antworten des Speicherns, Zerstören). Die Mutationsprobe scripts/mutanten/rescue.json prüft, dass sie es tragen.

import (
	"errors"
	"image"
	"os"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
)

// meldendeRettung ist eine Attrappe, deren Wiederherstellen eine Meldung liefert.
type meldendeRettung struct {
	rettungAttrappe
	text   string
	fehler bool
}

func (r *meldendeRettung) Wiederherstellen(*document.Dokument) (string, bool) {
	return r.text, r.fehler
}

// U7, U7b, U8: Der Aufbau des Editors wendet die Rettungskopie an und zeigt ihre Meldung in der Fußzeile.
func TestNeuerEditorZeigtWiederherstellung(t *testing.T) {
	i18n.Setze(nil)
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 40, 30)))
	if err != nil {
		t.Fatal(err)
	}
	r := &meldendeRettung{text: "Restored unsaved changes", fehler: true}
	e := neuerEditor("datei.png", d, nil, r)
	if e.meldung.Text != r.text || !e.meldung.Fehler {
		t.Errorf("Meldung %q (Fehler %v), erwartet %q als Fehler", e.meldung.Text, e.meldung.Fehler, r.text)
	}
}

// U9: Die Schleife prüft nach jedem Ereignis die Rettungskopie (nachEreignis).
func TestNachEreignisSichert(t *testing.T) {
	jetzt := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	alt := rettungUhr
	rettungUhr = func() time.Time { return jetzt }
	t.Cleanup(func() { rettungUhr = alt })
	e := testEditor(t)
	e.weitereSignale = make(chan struct{}, 1)
	e.wache = canvas.NeueSignalwache(make(chan os.Signal), func() {}, func(os.Signal, bool) {})
	t.Cleanup(e.wache.Ende)
	r := &rettungAttrappe{}
	e.rettung = r
	e.rettungStart()
	haltAendere(t, e)
	e.nachEreignis()
	jetzt = jetzt.Add(rettungRuhe)
	e.nachEreignis()
	if r.gesichert != 1 {
		t.Errorf("%d Sicherungen nach der Ruhezeit, erwartet 1", r.gesichert)
	}
}

// U1, U3: Während eines Speicherns und nach der Freigabe zum Schließen ruht die Rettung, auch wenn sich der Stand
// ändert und die Bündelung um ist.
func TestRettungRuhtBeiSpeichernUndErzwungen(t *testing.T) {
	jetzt := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	alt := rettungUhr
	rettungUhr = func() time.Time { return jetzt }
	t.Cleanup(func() { rettungUhr = alt })
	fehlerAbfangen(t)

	e := geaenderterEditor(t)
	v := neuerSperrSpeicher()
	einmal := freigebenBeiEnde(t, v)
	e.speichern = v.speichern
	r := &rettungAttrappe{}
	e.rettung = r
	e.rettungStart()
	e.speichernJetzt()
	vorher := r.gesichert
	haltAendere(t, e)  // am Dokument vorbei an der Sperre der Oberfläche: Die Rettung selbst muss ruhen
	e.rettungPruefen() // sieht die Änderung
	jetzt = jetzt.Add(time.Minute)
	e.rettungPruefen() // Bündelung um
	if r.gesichert != vorher {
		t.Errorf("während des Speicherns gesichert: %d, vorher %d", r.gesichert, vorher)
	}
	einmal.Do(func() { close(v.frei) })
	warteSpeichern(t, e)

	f := testEditor(t)
	rf := &rettungAttrappe{}
	f.rettung = rf
	f.rettungStart()
	f.erzwungen = true
	haltAendere(t, f)
	f.rettungPruefen()
	jetzt = jetzt.Add(time.Minute)
	f.rettungPruefen()
	if rf.gesichert != 0 {
		t.Errorf("nach der Freigabe zum Schließen %d Sicherungen, erwartet 0", rf.gesichert)
	}
}

// U14: Ein gescheitertes Speichern verwirft die Rettungskopie nicht; ein gelungenes schon.
func TestSpeichernFertigVerwirftNurNachErfolg(t *testing.T) {
	fehlerAbfangen(t)
	for _, fall := range []struct {
		name   string
		fehler error
		soll   int
	}{{"scheitert", errors.New("platte voll"), 0}, {"gelingt", nil, 1}} {
		e := geaenderterEditor(t)
		v := neuerSperrSpeicher()
		v.fehler = []error{fall.fehler}
		e.speichern = v.speichern
		r := &rettungAttrappe{}
		e.rettung = r
		e.rettungStart()
		e.speichernJetzt()
		close(v.frei)
		warteAntwort(t, e)
		if !e.speichernAbholen() {
			t.Fatalf("%s: Antwort nicht abgeholt", fall.name)
		}
		if r.verworfen != fall.soll {
			t.Errorf("%s: %d-mal verworfen, erwartet %d", fall.name, r.verworfen, fall.soll)
		}
	}
}

// U12: speichernAbwarten (beim Zerstören) verwirft nach gelungenem Speichern, nach gescheitertem nicht.
func TestSpeichernAbwartenVerwirft(t *testing.T) {
	fehlerAbfangen(t)
	for _, fall := range []struct {
		name   string
		fehler error
		soll   int
	}{{"scheitert", errors.New("platte voll"), 0}, {"gelingt", nil, 1}} {
		e := geaenderterEditor(t)
		v := neuerSperrSpeicher()
		v.fehler = []error{fall.fehler}
		e.speichern = v.speichern
		r := &rettungAttrappe{}
		e.rettung = r
		e.rettungStart()
		e.speichernJetzt()
		close(v.frei)
		e.speichernAbwarten()
		if r.verworfen != fall.soll {
			t.Errorf("%s: %d-mal verworfen, erwartet %d", fall.name, r.verworfen, fall.soll)
		}
	}
}

// U11: Kommt die Antwort eines aufgegebenen Speicherns doch noch und gelang es, wird verworfen.
func TestSpaeteAntwortVerwirft(t *testing.T) {
	fehlerAbfangen(t)
	e := geaenderterEditor(t)
	v := neuerSperrSpeicher()
	e.speichern = v.speichern
	r := &rettungAttrappe{}
	e.rettung = r
	e.rettungStart()
	e.speichernJetzt()
	e.speichernAufgeben()
	close(v.frei)
	warteAntwort(t, e)
	e.spaeteAntwort(<-e.speicherAntwort)
	if r.verworfen != 1 {
		t.Errorf("%d-mal verworfen, erwartet 1", r.verworfen)
	}
}

// U13: Zerstören räumt die Rettung auf (wartet auf ihre Schreibaufträge).
func TestZerstoertBeendetRettung(t *testing.T) {
	e := testEditor(t)
	r := &rettungAttrappe{}
	e.rettung = r
	e.rettungStart()
	if err := e.zerstoert(nil); err != nil {
		t.Fatal(err)
	}
	if r.ende != 1 || r.verworfen != 1 {
		t.Errorf("Zerstören ohne Änderung: %+v, erwartet einmal Ende und einmal Verwirf", r)
	}
}
