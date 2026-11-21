// SPDX-License-Identifier: GPL-3.0-or-later

package saveprogress

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"
)

// F-298: Marke, Puls, Schreiber und Leser setzen den Zeitpunkt des letzten Fortschritts.
func TestFortschritt(t *testing.T) {
	for name, tu := range map[string]func(){
		"Marke": func() { Marke("probe") },
		"Puls":  Puls,
		"Schreiber": func() {
			var b bytes.Buffer
			if _, err := Schreiber(&b).Write([]byte("x")); err != nil || b.String() != "x" {
				t.Errorf("Schreiber reicht nicht weiter: %v %q", err, b.String())
			}
		},
		"Leser": func() {
			got, err := io.ReadAll(Leser(strings.NewReader("abc")))
			if err != nil || string(got) != "abc" {
				t.Errorf("Leser reicht nicht weiter: %v %q", err, got)
			}
		},
	} {
		vorher := time.Now()
		time.Sleep(time.Millisecond)
		tu()
		if !Zuletzt().After(vorher) {
			t.Errorf("%s: kein Fortschritt", name)
		}
	}
}

// Verify F-298 B-5: Zuletzt trägt den monotonen Anteil der Uhr, Vergleiche folgen keinem Sprung der Wanduhr.
func TestZuletztMonoton(t *testing.T) {
	Puls()
	z := Zuletzt()
	if !strings.Contains(z.String(), " m=") { // time.Time.String zeigt den monotonen Anteil als „m=±…“
		t.Errorf("Zuletzt ohne monotonen Anteil: %v", z)
	}
}
