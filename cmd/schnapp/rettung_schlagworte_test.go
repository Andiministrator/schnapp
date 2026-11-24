// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/ui"
)

// F-703: Schlagworte in der Bibliotheksansicht ändern die Datei einer Aufnahme; eine liegende Rettungskopie zu ihr
// bleibt anwendbar, über den Weg, den die Ansicht nimmt (Anbindung.UmDatei).
func TestSchlagworteHaltenRettungskopieAnwendbar(t *testing.T) {
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

	var (
		rettung ui.Rettung
		dok     *document.Dokument
	)
	altStart := editorStarten
	editorStarten = func(g ui.Geoeffnet, _ *ui.Anbindung) error {
		rettung, dok = g.Rettung, g.Dokument
		return nil
	}
	t.Cleanup(func() { editorStarten = altStart })

	// Erster Lauf: bearbeiten, sichern, „abstürzen“ (Fenster zu, Sperre frei).
	if err := oeffneImEditor(abs, false, document.KanteAbriss); err != nil {
		t.Fatal(err)
	}
	mitKasten(t, dok)
	rettung.Sichere(dok)
	rettung.Ende()

	// Die Ansicht setzt ein Schlagwort, wie bibSchlagworteUebernehmen.
	um := anbindungFuer(b, document.KanteAbriss).UmDatei
	if um == nil {
		t.Fatal("Anbindung ohne UmDatei")
	}
	err = um(abs, func() error {
		_, err := b.SetzeSchlagworte(e.Pfad, []string{"wichtig"})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	// Zweiter Lauf: die Bearbeitung ist zurück.
	if err := oeffneImEditor(abs, false, document.KanteAbriss); err != nil {
		t.Fatal(err)
	}
	text, fehler := rettung.Wiederherstellen(dok)
	if !strings.Contains(text, "Restored unsaved changes") || fehler || len(dok.Objekte()) != 1 {
		t.Errorf("nicht wiederhergestellt: %q (Fehler %v), %d Objekte", text, fehler, len(dok.Objekte()))
	}
	rettung.Ende()
}
