// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

// kritikUhr tauscht Uhr und Pause des Countdowns gegen eine künstliche Uhr.
func kritikUhr(t *testing.T) (jetzt *time.Time) {
	t.Helper()
	z := time.Unix(1000, 0)
	altJ, altS, altE, altC := countdownJetzt, countdownSchlaf, countdownErsetze, countdownSchliesse
	t.Cleanup(func() { countdownJetzt, countdownSchlaf, countdownErsetze, countdownSchliesse = altJ, altS, altE, altC })
	countdownJetzt = func() time.Time { return z }
	countdownSchlaf = func(d time.Duration) { z = z.Add(d) }
	return &z
}

// Kritik A: Ein hängender Benachrichtigungsdienst (jeder Aufruf läuft in die 5-s-Frist von notify) darf die
// Wartezeit nicht verlängern (ADR-0042 Punkt 4: „die Gesamtzeit bleibt N Sekunden“, „ohne Dienst wird trotzdem
// gewartet“).
func TestKritikHaengenderDienstVerlaengertNicht(t *testing.T) {
	jetzt := kritikUhr(t)
	start := *jetzt
	aufrufe := 0
	countdownErsetze = func(context.Context, uint32, string, string) (uint32, error) {
		aufrufe++
		*jetzt = jetzt.Add(5 * time.Second) // notify.frist
		return 0, errors.New("notify: context deadline exceeded")
	}
	countdownSchliesse = func(context.Context, uint32) error { return nil }
	countdown(3, io.Discard)
	t.Logf("3 s verlangt, gewartet %v, %d Aufrufe", jetzt.Sub(start), aufrufe)
	if aufrufe != 1 {
		t.Errorf("%d Aufrufe, Soll 1 (nach dem ersten Fehler keine weiteren Hinweise, F-938)", aufrufe)
	}
	if jetzt.Sub(start) > 3*time.Second+5*time.Second {
		t.Errorf("hängender Dienst: gewartet %v statt 3 s (höchstens eine Frist darüber)", jetzt.Sub(start))
	}
}

// Kritik B: Antwortet der Dienst langsam (700 ms je Aufruf), muss zwischen dem Schließen des Hinweises und der
// Aufnahme trotzdem hinweisWeg liegen, sonst steht der ausblendende Hinweis im Bild (F-490).
func TestKritikLangsamerDienstHaeltAbstand(t *testing.T) {
	jetzt := kritikUhr(t)
	var zu time.Time
	countdownErsetze = func(context.Context, uint32, string, string) (uint32, error) {
		*jetzt = jetzt.Add(700 * time.Millisecond)
		return 7, nil
	}
	countdownSchliesse = func(context.Context, uint32) error {
		*jetzt = jetzt.Add(700 * time.Millisecond)
		zu = *jetzt
		return nil
	}
	countdown(3, io.Discard)
	t.Logf("Abstand Schließen bis Aufnahme: %v", jetzt.Sub(zu))
	if jetzt.Sub(zu) < hinweisWeg {
		t.Errorf("Abstand zwischen Schließen und Aufnahme %v, Soll mindestens %v", jetzt.Sub(zu), hinweisWeg)
	}
}

// Kritik C: Hinkt die Schleife hinterher (Aufruf dauert 2,5 s), zeigt der Hinweis eine Zahl, die nicht mehr stimmt.
func TestKritikZahlStimmtBeiVerzug(t *testing.T) {
	jetzt := kritikUhr(t)
	start := *jetzt
	countdownErsetze = func(_ context.Context, _ uint32, _, text string) (uint32, error) {
		rest := 3*time.Second - jetzt.Sub(start)
		t.Logf("bei Rest %v gezeigt: %q", rest, text)
		*jetzt = jetzt.Add(2500 * time.Millisecond)
		return 7, nil
	}
	countdownSchliesse = func(context.Context, uint32) error { return nil }
	countdown(3, io.Discard)
	t.Logf("3 s verlangt, gewartet %v", jetzt.Sub(start))
	if jetzt.Sub(start) > 3*time.Second+2500*time.Millisecond {
		t.Errorf("gewartet %v statt 3 s", jetzt.Sub(start))
	}
}

// Kritik D: Scheitert ein späterer Hinweis, wird der zuvor gezeigte trotzdem geschlossen (Kennung bleibt).
func TestKritikKennungBleibtNachFehler(t *testing.T) {
	kritikUhr(t)
	n := 0
	countdownErsetze = func(context.Context, uint32, string, string) (uint32, error) {
		n++
		if n == 1 {
			return 7, nil
		}
		return 0, errors.New("weg")
	}
	var zu []uint32
	countdownSchliesse = func(_ context.Context, id uint32) error { zu = append(zu, id); return nil }
	countdown(3, io.Discard)
	if len(zu) != 1 || zu[0] != 7 {
		t.Errorf("geschlossen %v, Soll [7]", zu)
	}
}
