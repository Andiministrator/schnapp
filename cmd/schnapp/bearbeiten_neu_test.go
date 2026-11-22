// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"image"
	"path/filepath"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/ui"
)

// F-282: bearbeiteNeu übergibt dem Editor den Stand der Datei vom Ablegen (a.stand), nicht den Nullwert. Mit dem
// Nullwert passte der Stand zu keiner Datei, und jedes Speichern wiche auf eine neue Aufnahme aus, statt die abgelegte
// zu fortzuschreiben. Das Fenster ist eine Attrappe (editorStarten); geprüft wird die Speicherfunktion, die sie bekommt.
func TestBearbeiteNeuGibtStandVomAblegenWeiter(t *testing.T) {
	i18n.Setze(nil)
	heim := t.TempDir()
	t.Setenv("HOME", heim)
	for _, name := range []string{"XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME"} {
		t.Setenv(name, "")
	}
	orte, err := library.OrteAus(func(k string) string { return map[string]string{"HOME": heim}[k] })
	if err != nil {
		t.Fatal(err)
	}
	b := library.Neu(orte)

	legeAb := func() (abgelegt, string) {
		d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 8, 8)))
		if err != nil {
			t.Fatal(err)
		}
		e, err := b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{})
		if err != nil {
			t.Fatal(err)
		}
		abs := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
		return abgelegt{pfad: abs, d: d, stand: standVon(t, abs)}, abs
	}
	// editor liefert die Speicherfunktion, die bearbeiteNeu dem (nicht gestarteten) Editor übergibt.
	var speichern ui.Speichern
	alt := editorStarten
	editorStarten = func(_ string, _ *document.Dokument, s ui.Speichern, _ ui.Rettung) error { speichern = s; return nil }
	t.Cleanup(func() { editorStarten = alt })

	// Unverändert: Speichern schreibt an den Platz der abgelegten Aufnahme zurück.
	a, abs := legeAb()
	if err := bearbeiteNeu(a); err != nil {
		t.Fatal(err)
	}
	mitKasten(t, a.d)
	if erg, err := speichern(a.d); err != nil || erg != (ui.Gespeichert{}) {
		t.Fatalf("Zurückschreiben an den Platz: %+v, %v (Soll: ohne Ausweichen)", erg, err)
	}
	if g, err := fileformat.Oeffne(abs); err != nil || len(g.Dokument.Objekte()) != 1 {
		t.Fatalf("zurückgeschriebene Datei: %v", err)
	}

	// Ein anderer Prozess ändert die Datei vor dem Start des Editors: nichts wird überschrieben, es kommt eine neue
	// Aufnahme (F-110).
	a, abs = legeAb()
	a2, g2, s2, err := ladeFuerEditor(abs, false)
	if err != nil {
		t.Fatal(err)
	}
	mitKasten(t, g2.Dokument)
	if erg, err := speichernIn(b, a2, g2, s2)(g2.Dokument); err != nil || erg != (ui.Gespeichert{}) {
		t.Fatalf("anderer: %+v, %v", erg, err)
	}
	if err := bearbeiteNeu(a); err != nil {
		t.Fatal(err)
	}
	erg, err := speichern(a.d)
	if err != nil || erg.Datei == "" {
		t.Fatalf("nach fremder Änderung: %+v, %v (Soll: neue Aufnahme)", erg, err)
	}
	if g, err := fileformat.Oeffne(abs); err != nil || len(g.Dokument.Objekte()) != 1 {
		t.Errorf("Datei des anderen überschrieben: %v", err)
	}
}
