// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/saveprogress"
)

// Tests zu F-298 (ADR-0020 Nachtrag 9): Die Frist für hängendes Speichern zählt ab dem letzten Fortschritt.
//
// F-317: Die Fristentscheidungen laufen auf einer steuerbaren Uhr (saveprogress.SetzeUhr) und mit einem Speicher, der
// nur auf das Freigeben des Tests wartet; kein Test schläft, um eine Frist zu überschreiten, und keine Goroutine pulst
// nach dem Ende des Tests weiter. Nur TestFristPruefungWiederholtSich braucht echte Timer und wartet mit Obergrenze
// statt mit fester Dauer.

// steuerUhr ist eine Uhr, die nur der Test vorstellt.
type steuerUhr struct {
	mu sync.Mutex
	t  time.Time
}

func (u *steuerUhr) jetzt() time.Time {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.t
}

// vor stellt die Uhr um d vor.
func (u *steuerUhr) vor(d time.Duration) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.t = u.t.Add(d)
}

// vorMitPuls stellt die Uhr in Schritten von schritt bis gesamt vor und meldet nach jedem Schritt Fortschritt; kein
// Schritt darf allein schon die Frist überschreiten (das prüft der Aufrufer über speichernHaengt).
func (u *steuerUhr) vorMitPuls(schritt, gesamt time.Duration, nachSchritt func()) {
	for d := time.Duration(0); d < gesamt; d += schritt {
		u.vor(schritt)
		saveprogress.Puls()
		if nachSchritt != nil {
			nachSchritt()
		}
	}
}

// uhrTest setzt für den Test eine steuerbare Uhr als Zeitquelle von Fortschritt und Fristen ein.
func uhrTest(t *testing.T) *steuerUhr {
	t.Helper()
	u := &steuerUhr{t: saveprogress.Basis().Add(time.Hour)}
	alt := saveprogress.SetzeUhr(u.jetzt)
	t.Cleanup(func() { saveprogress.SetzeUhr(alt) })
	return u
}

// fristFalle ersetzt den Timer der äußersten Grenze beim Zerstören: Er merkt sich die angeforderte Restzeit und lässt
// die Grenze sofort ablaufen.
type fristFalle struct {
	rest []time.Duration
}

func fristFalleTest(t *testing.T) *fristFalle {
	t.Helper()
	f := &fristFalle{}
	alt := fristTimer
	t.Cleanup(func() { fristTimer = alt })
	fristTimer = func(rest time.Duration) (<-chan time.Time, func()) {
		f.rest = append(f.rest, rest)
		c := make(chan time.Time, 1)
		c <- time.Time{}
		return c, func() {}
	}
	return f
}

// fristNieTest lässt die äußerste Grenze beim Zerstören nie ablaufen.
func fristNieTest(t *testing.T) {
	t.Helper()
	alt := fristTimer
	t.Cleanup(func() { fristTimer = alt })
	fristTimer = func(time.Duration) (<-chan time.Time, func()) { return nil, func() {} }
}

// fristTest verkürzt speichernFrist für einen Test.
func fristTest(t *testing.T, d time.Duration) {
	t.Helper()
	alt := speichernFrist
	t.Cleanup(func() { speichernFrist = alt })
	speichernFrist = d
}

// zerstoerenFristTest verkürzt zerstoerenFrist für einen Test.
func zerstoerenFristTest(t *testing.T, d time.Duration) {
	t.Helper()
	alt := zerstoerenFrist
	t.Cleanup(func() { zerstoerenFrist = alt })
	zerstoerenFrist = d
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

// wartePoll wartet höchstens 30 s darauf, dass cond gilt (nur Wartezeit, keine Prüfung einer Dauer).
func wartePoll(t *testing.T, was string, cond func() bool) {
	t.Helper()
	ende := time.Now().Add(30 * time.Second)
	for !cond() {
		if time.Now().After(ende) {
			t.Fatalf("Zeitüberschreitung: %s", was)
		}
		time.Sleep(time.Millisecond)
	}
}

// warteAntwort wartet, bis die Antwort des Speicherns im Kanal liegt.
func warteAntwort(t *testing.T, e *editor) {
	t.Helper()
	wartePoll(t, "Antwort des Speicherns", func() bool { return len(e.speicherAntwort) > 0 })
}

// freigebenBeiEnde lässt einen noch wartenden Speicher nach dem Test frei, damit keine Goroutine hängen bleibt.
func freigebenBeiEnde(t *testing.T, v *sperrSpeicher) *sync.Once {
	t.Helper()
	var einmal sync.Once
	t.Cleanup(func() { einmal.Do(func() { close(v.frei) }) })
	return &einmal
}

// (a) Ein Speichern, das länger als die Frist dauert, aber Fortschritt meldet, wird weder durch eine weitere
// Schließen-Anfrage noch durch ein weiteres Signal aufgegeben; Zerstören wartet auf seine Antwort.
func TestSpeichernMitFortschrittNieAufgegeben(t *testing.T) {
	fristTest(t, 50*time.Millisecond)
	uhr := uhrTest(t)
	fristNieTest(t) // die äußerste Grenze liegt hier außer Reichweite: Antwort und Grenze zugleich wären unbestimmt
	for _, weg := range []string{"Schließen", "Signal"} {
		e := geaenderterEditor(t)
		v := neuerSperrSpeicher()
		einmal := freigebenBeiEnde(t, v)
		e.speichern = v.speichern
		e.speichernJetzt()
		if e.schliessen() {
			t.Fatalf("%s: schließt sofort", weg)
		}
		// 120 ms, länger als die Frist von 50 ms, aber nie 50 ms ohne Fortschritt
		uhr.vorMitPuls(40*time.Millisecond, 120*time.Millisecond, func() {
			if e.speichernHaengt() || e.speichernHaengtMelden() {
				t.Fatalf("%s: arbeitendes Speichern gilt als hängend", weg)
			}
		})
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
		// Die Grenze soll gar nicht erst gefragt werden müssen: Antwort vor dem Zerstören.
		einmal.Do(func() { close(v.frei) })
		warteAntwort(t, e)
		if err := e.zerstoert(nil); err != nil {
			t.Errorf("%s: Rückgabe %v", weg, err)
		}
		if e.s.Ungesichert() || e.speichernAufgegeben || v.aufrufe.Load() != 1 {
			t.Errorf("%s: ungesichert %v, aufgegeben %v, %d Aufrufe", weg, e.s.Ungesichert(), e.speichernAufgegeben,
				v.aufrufe.Load())
		}
	}
}

// Kommt nach der Meldung „antwortet nicht“ wieder Fortschritt, nimmt die Fußzeile sie zurück, und die nächste
// Schließen-Anfrage hält das Fenster wieder offen.
func TestHaengtMeldungZurueckBeiFortschritt(t *testing.T) {
	fristTest(t, 50*time.Millisecond)
	uhr := uhrTest(t)
	e := geaenderterEditor(t)
	v := neuerSperrSpeicher()
	freigebenBeiEnde(t, v)
	e.speichern = v.speichern
	e.speichernJetzt()
	uhr.vor(60 * time.Millisecond)
	if !e.speichernHaengtMelden() || !e.meldung.Fehler {
		t.Fatalf("Stillstand nicht gemeldet: %q", e.meldung.Text)
	}
	saveprogress.Puls()
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
	uhr := uhrTest(t)
	spaet := errors.New("spät gescheitert")
	for _, fall := range []struct {
		name   string
		fehler error
	}{{"gelingt", nil}, {"scheitert", spaet}} {
		for _, imFenster := range []bool{false, true} {
			e := geaenderterEditor(t)
			v := neuerSperrSpeicher()
			v.fehler = []error{fall.fehler}
			einmal := freigebenBeiEnde(t, v)
			e.speichern = v.speichern
			e.speichernJetzt()
			uhr.vor(60 * time.Millisecond) // ohne Fortschritt über die Frist
			e.weitereSignale <- struct{}{}
			if !e.weiteresSignalSchliesst() || !e.speichernAufgegeben {
				t.Fatalf("%s: hängendes Speichern nicht aufgegeben", fall.name)
			}
			einmal.Do(func() { close(v.frei) })
			warteAntwort(t, e)                      // die Goroutine legt die Antwort nach dem Speichern ab
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
// weckt die Schleife später noch einmal, nach dem Ende des Speicherns nicht mehr. Echte Timer (Frist 40 ms, danach
// alle 10 ms); gewartet wird mit Obergrenze auf mindestens drei Weckrufe, nicht eine feste Dauer, und die Prüfung
// „nach dem Ende kein Weckruf mehr“ gilt für jede Verzögerung, denn ein späterer Timer sieht `fertig` und rührt
// nichts an.
func TestFristPruefungWiederholtSich(t *testing.T) {
	fristTest(t, 40*time.Millisecond)
	e := geaenderterEditor(t)
	var weckrufe atomic.Int32
	e.wecker = canvas.NeuerWecker(func() { weckrufe.Add(1) }, time.Hour, time.Hour) // je Anforderung ein Ruf
	t.Cleanup(e.wecker.Ende)
	v := neuerSperrSpeicher()
	einmal := freigebenBeiEnde(t, v)
	e.speichern = v.speichern
	e.speichernJetzt()
	wartePoll(t, "drei Weckrufe während des Speicherns", func() bool {
		e.wecker.Quittieren()
		return weckrufe.Load() >= 3
	})
	einmal.Do(func() { close(v.frei) })
	warteAntwort(t, e)
	// Ein Timer, der schon lief, als die Antwort kam, darf höchstens noch einmal wecken; danach ist Ruhe.
	time.Sleep(30 * time.Millisecond)
	e.wecker.Quittieren()
	vorher := weckrufe.Load()
	time.Sleep(100 * time.Millisecond) // zehn Takte ohne Weckruf
	if n := weckrufe.Load(); n != vorher {
		t.Errorf("Prüfung weckt nach dem Ende weiter: %d → %d", vorher, n)
	}
}

// Verify F-298 B-2 (Mutationen M1, M14): Die Schließen-Anfrage gibt nur auf, wenn die Fußzeile den Stillstand
// gemeldet hat UND das Speichern jetzt noch hängt — nicht vor der Meldung und nicht, wenn seit der Meldung Fortschritt
// kam, auch bevor die Schleife die Meldung zurückgenommen hat.
func TestSchliessenNurBeiGemeldetemStillstand(t *testing.T) {
	fristTest(t, 50*time.Millisecond)
	uhr := uhrTest(t)
	e := geaenderterEditor(t)
	v := neuerSperrSpeicher()
	freigebenBeiEnde(t, v)
	e.speichern = v.speichern
	e.speichernJetzt()
	uhr.vor(60 * time.Millisecond)
	if e.schliessen() {
		t.Fatal("gibt auf, bevor die Fußzeile den Stillstand gemeldet hat")
	}
	if !e.speichernHaengtMelden() {
		t.Fatal("Stillstand nicht gemeldet")
	}
	saveprogress.Puls() // Fortschritt, die Schleife hat die Meldung noch nicht zurückgenommen
	if e.schliessen() {
		t.Error("gibt trotz neuem Fortschritt auf")
	}
}

// Verify F-298 B-2 (Mutation M3): Ein weiteres Signal, während speichernAbwarten beim Zerstören auf ein Speichern mit
// Fortschritt wartet, gibt es nicht auf. Der Test gibt den Speicher erst frei, wenn speichernAbwarten das Signal aus
// dem Kanal genommen hat; danach darf nichts aufgegeben sein.
func TestAbwartenSignalMitFortschritt(t *testing.T) {
	fristTest(t, 50*time.Millisecond)
	uhr := uhrTest(t)
	fristNieTest(t) // die Grenze zerstoerenFrist soll nicht auslösen: Signal und Antwort entscheiden
	e := geaenderterEditor(t)
	v := neuerSperrSpeicher()
	einmal := freigebenBeiEnde(t, v)
	e.speichern = v.speichern
	e.speichernJetzt()
	uhr.vorMitPuls(40*time.Millisecond, 120*time.Millisecond, nil) // länger als die Frist, aber mit Fortschritt
	e.weitereSignale <- struct{}{}                                 // liegt im Kanal, wenn speichernAbwarten beginnt
	go func() {
		for len(e.weitereSignale) > 0 { // Signal angenommen und bewertet
			time.Sleep(time.Millisecond)
		}
		einmal.Do(func() { close(v.frei) })
	}()
	if err := e.zerstoert(nil); err != nil {
		t.Errorf("Rückgabe %v", err)
	}
	if e.speichernAufgegeben || e.s.Ungesichert() {
		t.Errorf("aufgegeben %v, ungesichert %v", e.speichernAufgegeben, e.s.Ungesichert())
	}
}

// Verify F-298 B-2 (Mutation M12): zerstoerenFrist zählt ab dem Beginn des Speicherns und bleibt äußerste Grenze,
// auch wenn es Fortschritt meldet (Andi, 21.11.2026, Nachtrag 9). Läuft die Grenze ab, gibt Zerstören auf.
func TestZerstoerenFristTrotzFortschritt(t *testing.T) {
	fristTest(t, 50*time.Millisecond)
	zerstoerenFristTest(t, 100*time.Millisecond)
	uhr := uhrTest(t)
	falle := fristFalleTest(t)
	e := geaenderterEditor(t)
	v := neuerSperrSpeicher()
	freigebenBeiEnde(t, v)
	e.speichern = v.speichern
	e.speichernJetzt()
	uhr.vorMitPuls(30*time.Millisecond, 90*time.Millisecond, nil) // Fortschritt bis kurz vor der Grenze
	err := e.zerstoert(nil)
	if !errors.Is(err, ErrSpeichernHaengt) || !e.speichernAufgegeben {
		t.Errorf("Rückgabe %v, aufgegeben %v", err, e.speichernAufgegeben)
	}
	if len(falle.rest) != 1 || falle.rest[0] != 10*time.Millisecond {
		t.Errorf("Restzeit der Grenze %v, erwartet [10ms] (100 ms ab Beginn, 90 ms vergangen)", falle.rest)
	}
}

// Kritiker Sprint 9 (F-314): Wird das Fenster erst zerstört, während das Speichern schon eine Weile mit Fortschritt
// läuft, zählt zerstoerenFrist trotzdem ab Beginn des Speicherns, nicht ab dem letzten Fortschritt. Der Test oben ruft
// zerstoert nach 90 von 100 ms und kann beides kaum unterscheiden; hier sind es 150 von 200 ms mit Fortschritt zuletzt
// bei 150 ms: ab Beginn bleiben 50 ms, ab letztem Fortschritt wären es 200 ms.
func TestZerstoerenFristAbBeginnBeiSpaetemZerstoeren(t *testing.T) {
	fristTest(t, 50*time.Millisecond)
	zerstoerenFristTest(t, 200*time.Millisecond)
	uhr := uhrTest(t)
	falle := fristFalleTest(t)
	e := geaenderterEditor(t)
	v := neuerSperrSpeicher()
	freigebenBeiEnde(t, v)
	e.speichern = v.speichern
	e.speichernJetzt()
	uhr.vorMitPuls(50*time.Millisecond, 150*time.Millisecond, nil) // das Fenster lebt noch, das Speichern meldet Fortschritt
	err := e.zerstoert(nil)
	if !errors.Is(err, ErrSpeichernHaengt) {
		t.Errorf("Rückgabe %v", err)
	}
	if len(falle.rest) != 1 || falle.rest[0] != 50*time.Millisecond {
		t.Errorf("Restzeit der Grenze %v, erwartet [50ms] ab Beginn des Speicherns", falle.rest)
	}
}

// Verify F-298 B-2 (Mutation M6b): Eine späte Antwort ersetzt nur ErrSpeichernHaengt, keinen Fehler, der schon vorher
// in der Rückgabe stand.
func TestSpaeteAntwortBehaeltFruehenFehler(t *testing.T) {
	fristTest(t, 50*time.Millisecond)
	uhr := uhrTest(t)
	frueh := errors.New("früher Fehler")
	e := geaenderterEditor(t)
	v := neuerSperrSpeicher()
	einmal := freigebenBeiEnde(t, v)
	e.speichern = v.speichern
	e.speichernJetzt()
	e.schlussErr = frueh
	uhr.vor(60 * time.Millisecond)
	e.weitereSignale <- struct{}{}
	if !e.weiteresSignalSchliesst() {
		t.Fatal("nicht aufgegeben")
	}
	einmal.Do(func() { close(v.frei) })
	warteAntwort(t, e)
	if err := e.zerstoert(nil); !errors.Is(err, frueh) {
		t.Errorf("Rückgabe %v, früher Fehler verloren", err)
	}
}

// Verify F-298 B-3: Strg+S hängt, ein Signal ist vorgemerkt, ein weiteres Signal gibt auf, danach scheitert das
// Speichern doch. Wie sonst nach einem Signal (Nachtrag 2) läuft der Neuversuch und wird abgewartet; nie zwei
// zugleich, die Sitzung meldet danach kein laufendes Speichern.
func TestNeuversuchNachSpaeterFehlantwort(t *testing.T) {
	fristTest(t, 50*time.Millisecond)
	uhr := uhrTest(t)
	for _, imFenster := range []bool{true, false} {
		e := geaenderterEditor(t)
		v := neuerSperrSpeicher()
		v.fehler = []error{errors.New("spät gescheitert")} // der zweite Aufruf gelingt
		einmal := freigebenBeiEnde(t, v)
		e.speichern = v.speichern
		e.speichernJetzt()
		if zu, abzug := e.s.SignalAnfrage(); zu || abzug != nil {
			t.Fatalf("SignalAnfrage: zu %v, Abzug %v", zu, abzug != nil)
		}
		uhr.vor(60 * time.Millisecond)
		e.weitereSignale <- struct{}{}
		if !e.weiteresSignalSchliesst() {
			t.Fatal("nicht aufgegeben")
		}
		einmal.Do(func() { close(v.frei) }) // auch der Neuversuch läuft gleich durch
		warteAntwort(t, e)
		if imFenster && !e.speichernAbholen() {
			t.Fatal("Antwort nicht abgeholt")
		}
		err := e.zerstoert(nil)
		if err != nil || e.s.Ungesichert() || e.s.SpeichernLaeuft() || v.aufrufe.Load() != 2 || v.doppelt.Load() {
			t.Errorf("Fenster %v: Rückgabe %v, ungesichert %v, läuft %v, %d Aufrufe, doppelt %v", imFenster, err,
				e.s.Ungesichert(), e.s.SpeichernLaeuft(), v.aufrufe.Load(), v.doppelt.Load())
		}
	}
}
