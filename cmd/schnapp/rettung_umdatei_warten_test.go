// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/ui"
)

// F-757 (Nachtrag aus Karte #288): Öffnet ein Fenster eine Aufnahme, während ein anderes über die Bibliotheksansicht
// ihre Schlagworte schreibt (Anbindung.UmDatei), wartet rettungFuer (Öffnen und Wechsel der Aufnahme) auf das Ende und
// wendet die liegende Kopie danach an, statt fremd zu werden.
func TestOeffnenWaehrendSchlagwortenWartetUndWendetAn(t *testing.T) {
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

	b := testBibliothek(t)
	d0, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 40, 30)))
	if err != nil {
		t.Fatal(err)
	}
	e, err := b.Ablegen(d0, d0.Ansichtsbild(), fileformat.Metadaten{})
	if err != nil {
		t.Fatal(err)
	}
	abs := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))

	type geoeffnet struct {
		rettung ui.Rettung
		dok     *document.Dokument
	}
	offen := make(chan geoeffnet, 2)
	altStart := editorStarten
	editorStarten = func(g ui.Geoeffnet, _ *ui.Anbindung) error {
		offen <- geoeffnet{g.Rettung, g.Dokument}
		return nil
	}
	t.Cleanup(func() { editorStarten = altStart })

	// Erster Lauf: bearbeiten, sichern, „abstürzen“ (Fenster zu, Sperre frei).
	if err := oeffneImEditor(abs, false, document.KanteAbriss); err != nil {
		t.Fatal(err)
	}
	g := <-offen
	mitKasten(t, g.dok)
	g.rettung.Sichere(g.dok)
	g.rettung.Ende()

	// Ein anderes Fenster schreibt Schlagworte; mitten darin öffnet dieses Fenster die Aufnahme.
	um := anbindungFuer(b, document.KanteAbriss).UmDatei
	drin, weiter := make(chan struct{}), make(chan struct{})
	fertig := make(chan error, 1)
	go func() {
		fertig <- um(abs, func() error {
			close(drin)
			<-weiter
			_, err := b.SetzeSchlagworte(e.Pfad, []string{"wichtig"})
			return err
		})
	}()
	<-drin
	geoeffnetFehler := make(chan error, 1)
	go func() { geoeffnetFehler <- oeffneImEditor(abs, false, document.KanteAbriss) }()
	select {
	case <-offen:
		t.Fatal("das Fenster hat nicht auf das Schreiben der Schlagworte gewartet")
	case <-time.After(300 * time.Millisecond):
	}
	close(weiter)
	if err := <-fertig; err != nil {
		t.Fatal(err)
	}
	if err := <-geoeffnetFehler; err != nil {
		t.Fatal(err)
	}
	g = <-offen
	text, fehler := g.rettung.Wiederherstellen(g.dok)
	if !strings.Contains(text, "Restored unsaved changes") || fehler || len(g.dok.Objekte()) != 1 {
		t.Errorf("nicht wiederhergestellt: %q (Fehler %v), %d Objekte", text, fehler, len(g.dok.Objekte()))
	}
	g.rettung.Ende()
}
