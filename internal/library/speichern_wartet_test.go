// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// F-310, F-331: Speichern kehrt erst zurück, wenn die nebenläufige Verkleinerung des Vorschaubilds beendet ist, auch
// wenn es vorher mit einem Fehler abbricht. Ein großes sichtbares Bild macht die Verkleinerung deutlich länger als den
// frühen Abbruch (fremde Datei); bliebe die Goroutine zurück, zählte NumGoroutine nach der Rückkehr eine mehr.
func TestSpeichernKehrtNachVerkleinerungZurueck(t *testing.T) {
	b, _ := neueBibliothek(t)
	ordner := filepath.Join(b.Orte.Ablage, "2026-11")
	if err := os.MkdirAll(ordner, 0o700); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(ordner, "fremd.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, muster(16, 16)); err != nil {
		t.Fatal(err)
	}
	f.Close()
	d := neuesDokument(t, 16, 16)
	gross := image.NewNRGBA(image.Rect(0, 0, 6000, 4000))
	stand := standIn(t, b, "2026-11/fremd.png")
	for i := 0; i < 3; i++ {
		vorher := runtime.NumGoroutine()
		_, _, err = b.Speichern("2026-11/fremd.png", stand, d, gross)
		if !errors.Is(err, ErrFremd) {
			t.Fatalf("Fehler %v, erwartet ErrFremd", err)
		}
		// Die Goroutine signalisiert ihr Ende mit close (defer) und lebt danach noch einen Augenblick; auf einem langsamen
		// Rechner (CI mit -race) zählt NumGoroutine sie dann noch mit. Ein kurzes Nachlauffenster trennt das vom echten
		// Fehler: Eine nicht abgewartete Verkleinerung von 6000x4000 Pixeln läuft deutlich länger (Gegenprobe ohne das
		// Warten in Speichern: rot).
		nachher := runtime.NumGoroutine()
		for frist := time.Now().Add(nachlauf); nachher > vorher && time.Now().Before(frist); nachher = runtime.NumGoroutine() {
			time.Sleep(time.Millisecond)
		}
		if nachher > vorher {
			t.Errorf("Durchgang %d: nach der Rückkehr laufen %d statt %d Goroutinen (Verkleinerung nicht abgewartet)", i, nachher, vorher)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// nachlauf ist das Fenster, in dem eine eben beendete Goroutine noch gezählt werden darf (CI-Wackler 22.11.2026).
const nachlauf = 20 * time.Millisecond
