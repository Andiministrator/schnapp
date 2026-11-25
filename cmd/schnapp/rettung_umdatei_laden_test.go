// SPDX-License-Identifier: GPL-3.0-or-later

package main

// F-769, F-770 (ADR-0029 Nachtrag 12): Das Öffnen wartet vor dem Laden auf ein laufendes UmDatei und prüft nach dem
// Anlegen der Rettung, dass sie zum geladenen Stand gehört. Mutanten: scripts/mutanten/rescue.json (N10, N14).

import (
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/ui"
)

// umDateiUmgebung richtet Zustandsordner, Testbibliothek (auch für oeffneImEditor über os.Getenv) und eine Aufnahme mit
// liegender Rettungskopie (ein Kasten) ein.
func umDateiUmgebung(t *testing.T) (b *library.Bibliothek, rel, abs string, offen chan ui.Geoeffnet) {
	t.Helper()
	i18n.Setze(nil)
	staat := t.TempDir()
	alt := rettungUmgebung
	rettungUmgebung = func(k string) string {
		if k == "XDG_STATE_HOME" {
			return staat
		}
		return os.Getenv(k)
	}
	t.Cleanup(func() { rettungUmgebung = alt })

	b = testBibliothek(t)
	t.Setenv("HOME", filepath.Dir(filepath.Dir(b.Orte.Ablage)))
	t.Setenv("XDG_PICTURES_DIR", "")
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	if o, err := library.OrteAus(os.Getenv); err != nil || o.Ablage != b.Orte.Ablage {
		t.Fatalf("Orte der Testbibliothek nicht nachstellbar: %v, %q gegen %q", err, o.Ablage, b.Orte.Ablage)
	}
	d0, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 40, 30)))
	if err != nil {
		t.Fatal(err)
	}
	e, err := b.Ablegen(d0, d0.Ansichtsbild(), fileformat.Metadaten{})
	if err != nil {
		t.Fatal(err)
	}
	rel = e.Pfad
	abs = filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))

	offen = make(chan ui.Geoeffnet, 2)
	altStart := editorStarten
	editorStarten = func(g ui.Geoeffnet, _ *ui.Anbindung) error {
		offen <- g
		return nil
	}
	t.Cleanup(func() { editorStarten = altStart })

	// Erster Lauf: bearbeiten, sichern, „abstürzen“ (Fenster zu, Sperre frei).
	if err := oeffneImEditor(abs, false, document.KanteAbriss); err != nil {
		t.Fatal(err)
	}
	g := <-offen
	mitKasten(t, g.Dokument)
	g.Rettung.Sichere(g.Dokument)
	g.Rettung.Ende()
	return b, rel, abs, offen
}

// ablageDateien listet die Dateien der Ablage (relativ).
func ablageDateien(t *testing.T, b *library.Bibliothek) []string {
	t.Helper()
	var dateien []string
	_ = filepath.Walk(b.Orte.Ablage, func(p string, fi os.FileInfo, err error) error {
		if err == nil && fi.Mode().IsRegular() {
			rel, _ := filepath.Rel(b.Orte.Ablage, p)
			dateien = append(dateien, rel)
		}
		return nil
	})
	return dateien
}

// Öffnet ein Fenster die Aufnahme, während ein anderes ihre Schlagworte schreibt, lädt es erst danach: Es wendet die
// Kopie an, und das Speichern schreibt in die geöffnete Aufnahme zurück, mit Schlagwort und Kasten (F-769).
func TestNachWartenSpeichernTrifftNeuenStand(t *testing.T) {
	b, rel, abs, offen := umDateiUmgebung(t)

	um := anbindungFuer(b, document.KanteAbriss).UmDatei
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
	geoeffnetFehler := make(chan error, 1)
	go func() { geoeffnetFehler <- oeffneImEditor(abs, false, document.KanteAbriss) }()
	time.Sleep(300 * time.Millisecond)
	close(weiter)
	if err := <-fertig; err != nil {
		t.Fatal(err)
	}
	if err := <-geoeffnetFehler; err != nil {
		t.Fatal(err)
	}
	g := <-offen
	text, fehler := g.Rettung.Wiederherstellen(g.Dokument)
	if !strings.Contains(text, "Restored unsaved changes") || fehler {
		t.Errorf("nicht wiederhergestellt: %q (Fehler %v)", text, fehler)
	}
	erg, err := g.Speichern(g.Dokument)
	g.Rettung.Ende()
	if err != nil {
		t.Fatal(err)
	}
	dateien := ablageDateien(t, b)
	if len(dateien) != 1 || g.Pfad() != abs {
		t.Fatalf("Speichern nach dem Warten: %d Dateien %v, Pfad %s, Hinweis %q", len(dateien), dateien, g.Pfad(), erg.Hinweis)
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

// Beginnt UmDatei erst nach dem Warten und endet vor dem Anlegen der Rettung (Gegenrichtung, F-770), lädt
// ladeMitRettung neu: Dokument und Stand gehören zum neuen Stand, die nachgezogene Kopie passt.
func TestLadenNachSpaetemUmDateiLaedtNeu(t *testing.T) {
	b, rel, abs, _ := umDateiUmgebung(t)
	um := anbindungFuer(b, document.KanteAbriss).UmDatei
	versuche := 0
	lade := func(p string, ohneGrenze bool) (string, *fileformat.Gelesen, library.Stand, error) {
		versuche++
		a, g, stand, err := ladeFuerEditor(p, ohneGrenze)
		if versuche == 1 {
			if err := um(abs, func() error {
				_, err := b.SetzeSchlagworte(rel, []string{"spaet"})
				return err
			}); err != nil {
				t.Fatal(err)
			}
		}
		return a, g, stand, err
	}
	_, g, stand, r, err := ladeMitRettung(abs, false, lade)
	if err != nil {
		t.Fatal(err)
	}
	defer r.ende()
	jetzt, _ := library.StandVon(abs)
	if versuche != 2 || stand != jetzt || !slices.Equal(g.Meta.Schlagworte, []string{"spaet"}) {
		t.Errorf("%d Versuche, Stand aktuell %v, Schlagworte %v", versuche, stand == jetzt, g.Meta.Schlagworte)
	}
	if text, fehler := r.Wiederherstellen(g.Dokument); !strings.Contains(text, "Restored unsaved changes") || fehler {
		t.Errorf("nicht wiederhergestellt: %q (Fehler %v)", text, fehler)
	}
}
