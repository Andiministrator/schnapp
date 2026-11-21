// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/canvas"
	"github.com/Andiministrator/schnapp/document"
	"github.com/Andiministrator/schnapp/internal/speicherschritt"
)

// Tests zu F-298 (ADR-0020 Nachtrag 9): Die Frist für hängendes Speichern zählt ab dem letzten Fortschritt.

// fristTest verkürzt speichernFrist für einen Test.
func fristTest(t *testing.T, d time.Duration) {
	t.Helper()
	alt := speichernFrist
	t.Cleanup(func() { speichernFrist = alt })
	speichernFrist = d
}

// geaenderterEditor liefert einen Testeditor mit einer Änderung und weitereSignale.
func geaenderterEditor(t *testing.T) *editor {
	t.Helper()
	e := testEditor(t)
	e.weitereSignale = make(chan struct{}, 1)
	if _, err := e.s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(1, 1), document.Pt(5, 5))}); err != nil {
		t.Fatal(err)
	}
	return e
}

// pulsSpeicher meldet alle takt Fortschritt und gelingt nach dauer.
func pulsSpeicher(takt, dauer time.Duration, aufrufe *atomic.Int32) Speichern {
	return func(*document.Dokument) (Gespeichert, error) {
		aufrufe.Add(1)
		ende := time.Now().Add(dauer)
		for time.Now().Before(ende) {
			time.Sleep(takt)
			speicherschritt.Puls()
		}
		return Gespeichert{}, nil
	}
}

// (a) Ein Speichern, das länger als die Frist dauert, aber Fortschritt meldet, wird weder durch eine weitere
// Schließen-Anfrage noch durch ein weiteres Signal aufgegeben; Zerstören wartet auf seine Antwort.
func TestSpeichernMitFortschrittNieAufgegeben(t *testing.T) {
	fristTest(t, 50*time.Millisecond)
	for _, weg := range []string{"Schließen", "Signal"} {
		e := geaenderterEditor(t)
		var aufrufe atomic.Int32
		e.speichern = pulsSpeicher(10*time.Millisecond, 250*time.Millisecond, &aufrufe)
		e.speichernJetzt()
		if e.schliessen() {
			t.Fatalf("%s: schließt sofort", weg)
		}
		time.Sleep(120 * time.Millisecond) // länger als die Frist, aber mit Fortschritt
		if e.speichernHaengt() || e.speichernHaengtMelden() {
			t.Fatalf("%s: arbeitendes Speichern gilt als hängend", weg)
		}
		switch weg {
		case "Schließen":
			if e.schliessen() {
				t.Fatalf("%s: zweite Anfrage gibt ein arbeitendes Speichern auf", weg)
			}
		case "Signal":
			e.weitereSignale <- struct{}{}
			if e.weiteresSignalSchliesst() {
				t.Fatalf("%s: weiteres Signal gibt ein arbeitendes Speichern auf", weg)
			}
		}
		if err := e.zerstoert(nil); err != nil {
			t.Errorf("%s: Rückgabe %v", weg, err)
		}
		if e.s.Ungesichert() || e.speichernAufgegeben || aufrufe.Load() != 1 {
			t.Errorf("%s: ungesichert %v, aufgegeben %v, %d Aufrufe", weg, e.s.Ungesichert(), e.speichernAufgegeben,
				aufrufe.Load())
		}
	}
}

// Kommt nach der Meldung „antwortet nicht“ wieder Fortschritt, nimmt die Statusleiste sie zurück, und die nächste
// Schließen-Anfrage hält das Fenster wieder offen.
func TestHaengtMeldungZurueckBeiFortschritt(t *testing.T) {
	fristTest(t, 50*time.Millisecond)
	e := geaenderterEditor(t)
	v := neuerSperrSpeicher()
	t.Cleanup(func() { close(v.frei) })
	e.speichern = v.speichern
	e.speichernJetzt()
	time.Sleep(60 * time.Millisecond)
	if !e.speichernHaengtMelden() || !e.meldung.Fehler {
		t.Fatalf("Stillstand nicht gemeldet: %q", e.meldung.Text)
	}
	speicherschritt.Puls()
	if !e.speichernHaengtMelden() || e.meldung.Fehler || e.haengtGemeldet {
		t.Fatalf("Meldung nach Fortschritt nicht zurückgenommen: %q, Fehler %v", e.meldung.Text, e.meldung.Fehler)
	}
	if e.speichernHaengtMelden() {
		t.Error("Rücknahme doppelt gemeldet")
	}
	if e.schliessen() {
		t.Error("Anfrage nach Fortschritt gibt das Speichern auf")
	}
}

// (b, c) Ein Speichern ohne Fortschritt wird nach der Frist aufgegeben; kommt seine Antwort vor der Rückkehr doch,
// zählt sie: Gelang es, steht ErrSpeichernHaengt nicht mehr in der Rückgabe und der Stand gilt als gesichert;
// scheiterte es, steht sein Fehler dort.
func TestAufgegebenesSpeichernGelingtDoch(t *testing.T) {
	fristTest(t, 50*time.Millisecond)
	spaet := errors.New("spät gescheitert")
	for _, fall := range []struct {
		name   string
		fehler error
	}{{"gelingt", nil}, {"scheitert", spaet}} {
		for _, imFenster := range []bool{false, true} {
			e := geaenderterEditor(t)
			v := neuerSperrSpeicher()
			v.fehler = []error{fall.fehler}
			e.speichern = v.speichern
			e.speichernJetzt()
			time.Sleep(60 * time.Millisecond) // ohne Fortschritt über die Frist
			e.weitereSignale <- struct{}{}
			if !e.weiteresSignalSchliesst() || !e.speichernAufgegeben {
				t.Fatalf("%s: hängendes Speichern nicht aufgegeben", fall.name)
			}
			close(v.frei)
			// Warten, bis die Antwort im Kanal liegt (die Goroutine legt sie nach dem Speichern ab).
			for i := 0; len(e.speicherAntwort) == 0; i++ {
				if i > 500 {
					t.Fatalf("%s: keine Antwort", fall.name)
				}
				time.Sleep(2 * time.Millisecond)
			}
			if imFenster && !e.speichernAbholen() { // die Schleife holt sie noch vor dem Zerstören ab
				t.Fatalf("%s: Antwort nicht abgeholt", fall.name)
			}
			err := e.zerstoert(nil)
			if errors.Is(err, ErrSpeichernHaengt) {
				t.Errorf("%s (Fenster %v): ErrSpeichernHaengt bleibt stehen", fall.name, imFenster)
			}
			if fall.fehler == nil && (err != nil || e.s.Ungesichert()) {
				t.Errorf("%s (Fenster %v): Rückgabe %v, ungesichert %v", fall.name, imFenster, err, e.s.Ungesichert())
			}
			if fall.fehler != nil && !errors.Is(err, spaet) {
				t.Errorf("%s (Fenster %v): Rückgabe %v", fall.name, imFenster, err)
			}
			if v.aufrufe.Load() != 1 {
				t.Errorf("%s: %d Speicheraufrufe", fall.name, v.aufrufe.Load())
			}
		}
	}
}

// Die Prüfung der Frist wiederholt sich, solange das Speichern läuft: Ein Stillstand nach einer Phase mit Fortschritt
// weckt die Schleife später noch einmal, nach dem Ende des Speicherns nicht mehr.
func TestFristPruefungWiederholtSich(t *testing.T) {
	fristTest(t, 40*time.Millisecond)
	e := geaenderterEditor(t)
	var weckrufe atomic.Int32
	e.wecker = canvas.NeuerWecker(func() { weckrufe.Add(1) }, time.Hour, time.Hour) // je Anforderung ein Ruf
	t.Cleanup(e.wecker.Ende)
	v := neuerSperrSpeicher()
	e.speichern = v.speichern
	e.speichernJetzt()
	time.Sleep(150 * time.Millisecond) // Frist 40 ms, danach alle 10 ms
	e.wecker.Quittieren()
	if n := weckrufe.Load(); n < 3 {
		t.Errorf("nur %d Weckrufe während des Speicherns", n)
	}
	close(v.frei)
	for len(e.speicherAntwort) == 0 {
		time.Sleep(2 * time.Millisecond)
	}
	e.wecker.Quittieren()
	time.Sleep(30 * time.Millisecond)
	e.wecker.Quittieren()
	vorher := weckrufe.Load()
	time.Sleep(60 * time.Millisecond)
	if n := weckrufe.Load(); n != vorher {
		t.Errorf("Prüfung weckt nach dem Ende weiter: %d → %d", vorher, n)
	}
}
