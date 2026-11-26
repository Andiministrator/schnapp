// SPDX-License-Identifier: GPL-3.0-or-later

package main

// F-784 (Verify Sprint 13, V-4): Lücken der Tests zu F-765, F-769 und F-770. Mutanten A1, A2, A3, A5 und A8 in
// scripts/mutanten/rescue.json. Umgebung und Hilfen: rettung_umdatei_laden_test.go.

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/ui"
)

// A1: Der Wechsel der Aufnahme im selben Fenster (oeffnerFuer) wartet wie das Öffnen von der Kommandozeile auf ein
// laufendes UmDatei, lädt danach und schreibt beim Speichern in die geöffnete Aufnahme zurück, mit Schlagwort und Kasten.
func TestWechselWartetAufUmDatei(t *testing.T) {
	b, rel, abs, _ := umDateiUmgebung(t)

	um := anbindungFuer(b, document.KanteAbriss).UmDatei
	if um == nil {
		t.Fatal("Anbindung ohne UmDatei") // sonst endete der Test mit einer Panik ohne Namen (Mutant N5)
	}
	drin, weiter := make(chan struct{}), make(chan struct{})
	fertig := make(chan error, 1)
	go func() {
		fertig <- um(abs, func() error {
			close(drin)
			<-weiter
			_, err := b.SetzeSchlagworte(rel, []string{"wichtig"})
			return err
		})
	}()
	<-drin
	oeffnen := oeffnerFuer(b, document.KanteAbriss)
	ergebnis := make(chan ui.Geoeffnet, 1)
	fehler := make(chan error, 1)
	go func() {
		g, err := oeffnen(abs)
		fehler <- err
		ergebnis <- g
	}()
	time.Sleep(300 * time.Millisecond)
	close(weiter)
	if err := <-fertig; err != nil {
		t.Fatal(err)
	}
	if err := <-fehler; err != nil {
		t.Fatal(err)
	}
	g := <-ergebnis
	text, fehlerhaft := g.Rettung.Wiederherstellen(g.Dokument)
	if !strings.Contains(text, "Restored unsaved changes") || fehlerhaft {
		t.Errorf("nicht wiederhergestellt: %q (Fehler %v)", text, fehlerhaft)
	}
	_, err := g.Speichern(g.Dokument)
	g.Rettung.Ende()
	if err != nil {
		t.Fatal(err)
	}
	if dateien := ablageDateien(t, b); len(dateien) != 1 || g.Pfad() != abs {
		t.Fatalf("Speichern nach dem Warten: %d Dateien %v, Pfad %s", len(dateien), dateien, g.Pfad())
	}
	f, err := os.Open(abs)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gl, err := fileformat.LeseArbeitsdatei(f)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(gl.Meta.Schlagworte, []string{"wichtig"}) || len(gl.Dokument.Objekte()) != 1 {
		t.Errorf("gespeichert: Schlagworte %v, %d Objekte", gl.Meta.Schlagworte, len(gl.Dokument.Objekte()))
	}
}

// A5: Wird das Fenster fremd, weil UmDatei beim Anlegen der Rettung die Sperre hält, lädt ladeMitRettung neu, auch wenn
// sich der Stand der Datei nicht geändert hat (F-770).
func TestFremdDurchUmDateiLaedtNeu(t *testing.T) {
	b, _, abs, _ := umDateiUmgebung(t)
	um := anbindungFuer(b, document.KanteAbriss).UmDatei
	versuche := 0
	fertig := make(chan error, 1)
	lade := func(p string, ohneGrenze bool) (string, *fileformat.Gelesen, library.Stand, error) {
		versuche++
		if versuche == 1 {
			drin, weiter := make(chan struct{}), make(chan struct{})
			go func() {
				fertig <- um(abs, func() error {
					close(drin)
					<-weiter
					return nil // die Datei bleibt unverändert: nur die Sperre macht das Fenster fremd
				})
			}()
			<-drin
			time.AfterFunc(300*time.Millisecond, func() { close(weiter) })
		}
		return ladeFuerEditor(p, ohneGrenze)
	}
	_, _, _, r, err := ladeMitRettung(abs, false, lade)
	if err != nil {
		t.Fatal(err)
	}
	defer r.ende()
	if err := <-fertig; err != nil {
		t.Fatal(err)
	}
	if versuche != 2 || r.s.Lage().Fremd {
		t.Errorf("%d Versuche, fremd %v", versuche, r.s.Lage().Fremd)
	}
}

// A8: Ändert sich die Datei bei jedem Versuch, lädt ladeMitRettung genau ladeVersuche (3) Mal und bleibt dann beim
// letzten Ergebnis (F-770).
func TestLadeVersucheGrenze(t *testing.T) {
	_, _, abs, _ := umDateiUmgebung(t)
	versuche := 0
	lade := func(p string, ohneGrenze bool) (string, *fileformat.Gelesen, library.Stand, error) {
		versuche++
		a, g, stand, err := ladeFuerEditor(p, ohneGrenze)
		// Nach dem Laden ändert sich der Stand: die Rettung passt nie zum geladenen Stand.
		if e := os.Chtimes(abs, time.Now(), time.Now().Add(time.Duration(versuche)*time.Hour)); e != nil {
			t.Fatal(e)
		}
		return a, g, stand, err
	}
	_, _, _, r, err := ladeMitRettung(abs, false, lade)
	if err != nil {
		t.Fatal(err)
	}
	r.ende()
	if versuche != 3 || ladeVersuche != 3 {
		t.Errorf("%d Versuche bei ladeVersuche %d, erwartet 3", versuche, ladeVersuche)
	}
}

// A2, A3: Scheitert VorgabeKante nach dem Anlegen der Rettung (ungültige Kantenform), geben der Wechsel und das Öffnen
// von der Kommandozeile die Sperre der Ausgangsdatei wieder frei: ein neues Öffnen ist nicht fremd.
func TestVorgabeKanteFehlerGibtSperreFrei(t *testing.T) {
	for _, weg := range []string{"wechsel", "oeffnen"} {
		t.Run(weg, func(t *testing.T) {
			rettungsLauf(t)
			einfach := schreibeEinfachesPNG(t, t.TempDir(), "einfach.png")
			abs, err := filepath.Abs(einfach)
			if err != nil {
				t.Fatal(err)
			}
			ungueltig := document.Kante(99)
			if weg == "wechsel" {
				_, err = oeffnerFuer(nil, ungueltig)(abs)
			} else {
				err = oeffneImEditor(abs, false, ungueltig)
			}
			if err == nil {
				t.Fatal("ungültige Kantenform nicht abgelehnt")
			}
			r := rettungFuer(abs, false)
			defer r.ende()
			if r.s.Lage().Fremd {
				t.Error("die Sperre der Ausgangsdatei blieb nach dem Fehler gehalten")
			}
		})
	}
}
