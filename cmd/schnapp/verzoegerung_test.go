// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/capture"
	"github.com/Andiministrator/schnapp/internal/settings"
)

// ADR-0042: --delay=SEK nur mit Gleichheitszeichen, ganze Sekunden 0 bis 60, nur bei Aufnahmen; mehrfach nur gleich.
func TestAuswertenDelay(t *testing.T) {
	for _, f := range []struct {
		args   []string
		sek    int
		fehler bool
	}{
		{[]string{"--delay=5"}, 5, false},
		{[]string{"--screen", "--delay=0"}, 0, false},
		{[]string{"--delay=60", "--window", "--delay=60"}, 60, false},
		{[]string{"--delay"}, 0, true},
		{[]string{"--delay="}, 0, true},
		{[]string{"--delay=61"}, 0, true},
		{[]string{"--delay=-1"}, 0, true},
		{[]string{"--delay=2.5"}, 0, true},
		{[]string{"--delay=3", "--delay=4"}, 0, true},
		{[]string{"--delay=3", "bild.png"}, 0, true},
		{[]string{"--delay=3", "--list-screens"}, 0, true},
	} {
		a, err := Auswerten(f.args)
		if f.fehler {
			if err == nil {
				t.Errorf("%v: kein Fehler", f.args)
			}
			continue
		}
		if err != nil || !a.VerzoegerungGesetzt || a.Verzoegerung != f.sek {
			t.Errorf("%v: %+v, %v", f.args, a, err)
		}
	}
	if a, err := Auswerten(nil); err != nil || a.VerzoegerungGesetzt {
		t.Errorf("ohne --delay: %+v, %v", a, err)
	}
}

// ADR-0042: --delay vor [capture] delay, --delay=0 hebt die Einstellung auf; ohne Aufnahme keine Wartezeit.
func TestPlaneVerzoegerung(t *testing.T) {
	e := settings.Voreinstellungen()
	e.Verzoegerung = 5
	for _, f := range []struct {
		a    Aufruf
		soll int
	}{
		{Aufruf{Modus: ModusBereich}, 5},
		{Aufruf{Modus: ModusBereich, Verzoegerung: 3, VerzoegerungGesetzt: true}, 3},
		{Aufruf{Modus: ModusFenster, VerzoegerungGesetzt: true}, 0},
		{Aufruf{Modus: ModusOeffnen, Datei: "x.png"}, 0},
	} {
		if got := plane(f.a, e).Verzoegerung; got != f.soll {
			t.Errorf("%+v: %d, Soll %d", f.a, got, f.soll)
		}
	}
}

// ADR-0042: Gewartet wird vor der Verbindung zu KWin, ohne Verzögerung gar nicht.
func TestAufnahmeWartetVorDerQuelle(t *testing.T) {
	l := neuerLauf(t)
	var folge []string
	q := l.an.quelle
	l.an.quelle = func() (capture.Quelle, error) { folge = append(folge, "quelle"); return q() }
	l.an.warte = func(sek int, _ io.Writer) { folge = append(folge, "warte "+string(rune('0'+sek))) }
	l.fuehre(Auftrag{Modus: ModusFenster, Verzoegerung: 3})
	if strings.Join(folge, ",") != "warte 3,quelle" {
		t.Errorf("Reihenfolge %v", folge)
	}
	folge = nil
	l.fuehre(Auftrag{Modus: ModusFenster})
	if strings.Join(folge, ",") != "quelle" {
		t.Errorf("ohne Verzögerung: %v", folge)
	}
}

// ADR-0042: Der Countdown zeigt jede Sekunde die verbleibende Zahl (ersetzt den vorigen Hinweis), schließt ihn
// hinweisWeg vor dem Ende und kehrt genau nach sek Sekunden zurück; ohne Benachrichtigungsdienst wartet er trotzdem.
func TestCountdown(t *testing.T) {
	jetzt := time.Unix(1000, 0)
	altJ, altS, altE, altC := countdownJetzt, countdownSchlaf, countdownErsetze, countdownSchliesse
	t.Cleanup(func() { countdownJetzt, countdownSchlaf, countdownErsetze, countdownSchliesse = altJ, altS, altE, altC })
	countdownJetzt = func() time.Time { return jetzt }
	countdownSchlaf = func(d time.Duration) { jetzt = jetzt.Add(d) }
	var texte []string
	var ersetzt []uint32
	var zuBei time.Duration
	start := jetzt
	countdownErsetze = func(_ context.Context, id uint32, _, text string) (uint32, error) {
		texte = append(texte, text)
		ersetzt = append(ersetzt, id)
		return 7, nil
	}
	countdownSchliesse = func(_ context.Context, id uint32) error {
		if id != 7 {
			t.Errorf("schließt %d", id)
		}
		zuBei = jetzt.Sub(start)
		return nil
	}
	countdown(3, io.Discard)
	if jetzt.Sub(start) != 3*time.Second {
		t.Errorf("wartete %v, Soll 3 s", jetzt.Sub(start))
	}
	if len(texte) != 3 || !strings.Contains(texte[0], "3") || !strings.Contains(texte[2], "1") {
		t.Errorf("Texte %q", texte)
	}
	if ersetzt[0] != 0 || ersetzt[1] != 7 || ersetzt[2] != 7 {
		t.Errorf("ersetzt %v (der erste neu, dann derselbe)", ersetzt)
	}
	if zuBei != 3*time.Second-hinweisWeg {
		t.Errorf("geschlossen nach %v, Soll %v", zuBei, 3*time.Second-hinweisWeg)
	}

	// Ohne Dienst: kein Schließen, trotzdem volle Wartezeit.
	start = jetzt
	countdownErsetze = func(context.Context, uint32, string, string) (uint32, error) { return 0, errors.New("kein Dienst") }
	countdownSchliesse = func(context.Context, uint32) error { t.Error("schließt ohne Hinweis"); return nil }
	countdown(2, io.Discard)
	if jetzt.Sub(start) != 2*time.Second {
		t.Errorf("ohne Dienst wartete %v", jetzt.Sub(start))
	}
}
