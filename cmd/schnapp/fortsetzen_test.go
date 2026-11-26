// SPDX-License-Identifier: GPL-3.0-or-later

package main

// ADR-0036 Zusatz Karte #356: fortsetzerFuer nimmt das gemerkte Dokument nur bei unverändertem Dateistand, und das
// Speichern danach schreibt in dieselbe Aufnahme zurück.

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
)

func fortsetzenUmgebung(t *testing.T) (*library.Bibliothek, string) {
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
	b := testBibliothek(t)
	d0, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 40, 30)))
	if err != nil {
		t.Fatal(err)
	}
	e, err := b.Ablegen(d0, d0.Ansichtsbild(), fileformat.Metadaten{})
	if err != nil {
		t.Fatal(err)
	}
	return b, filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
}

func fortsetzenKasten(x int) document.Kasten {
	return document.Kasten{Bereich: document.R(document.Pt(float64(x), 2), document.Pt(float64(x+8), 12)),
		Strich: document.Strich{Farbe: color.NRGBA{R: 255, A: 255}, Breite: 2}}
}

func objekteInDatei(t *testing.T, abs string) int {
	t.Helper()
	f, err := os.Open(abs)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	g, err := fileformat.LeseArbeitsdatei(f)
	if err != nil {
		t.Fatal(err)
	}
	return len(g.Dokument.Objekte())
}

func TestFortsetzenNimmtGemerktesDokument(t *testing.T) {
	b, abs := fortsetzenUmgebung(t)
	geo, err := oeffnerFuer(b, document.KanteAbriss)(abs)
	if err != nil {
		t.Fatal(err)
	}
	d := geo.Dokument
	if _, err := d.Hinzufuegen(fortsetzenKasten(2)); err != nil {
		t.Fatal(err)
	}
	if _, err := geo.Speichern(d); err != nil {
		t.Fatal(err)
	}
	geo.Rettung.Ende()
	stand, err := library.StandVon(abs)
	if err != nil {
		t.Fatal(err)
	}

	fort, err := fortsetzerFuer(b, document.KanteAbriss)(abs, d, stand)
	if err != nil {
		t.Fatal(err)
	}
	if !fort.Fortgesetzt || fort.Dokument != d {
		t.Fatalf("unveränderte Datei: fortgesetzt %v, gemerktes Dokument %v", fort.Fortgesetzt, fort.Dokument == d)
	}
	if !d.KannRueckgaengig() {
		t.Error("Verlauf verloren")
	}
	if _, err := d.Hinzufuegen(fortsetzenKasten(14)); err != nil {
		t.Fatal(err)
	}
	if _, err := fort.Speichern(d); err != nil {
		t.Fatal(err)
	}
	fort.Rettung.Ende()
	if fort.Pfad() != abs || objekteInDatei(t, abs) != 2 || len(ablageDateien(t, b)) != 1 {
		t.Errorf("Speichern nach dem Fortsetzen: Pfad %s, %d Objekte, %d Dateien", fort.Pfad(), objekteInDatei(t, abs),
			len(ablageDateien(t, b)))
	}
}

func TestFortsetzenNachFremderAenderungLiestNeu(t *testing.T) {
	b, abs := fortsetzenUmgebung(t)
	geo, err := oeffnerFuer(b, document.KanteAbriss)(abs)
	if err != nil {
		t.Fatal(err)
	}
	d := geo.Dokument
	if _, err := d.Hinzufuegen(fortsetzenKasten(2)); err != nil {
		t.Fatal(err)
	}
	if _, err := geo.Speichern(d); err != nil {
		t.Fatal(err)
	}
	geo.Rettung.Ende()
	stand, err := library.StandVon(abs)
	if err != nil {
		t.Fatal(err)
	}
	spaeter := time.Now().Add(time.Hour)
	if err := os.Chtimes(abs, spaeter, spaeter); err != nil {
		t.Fatal(err)
	}
	fort, err := fortsetzerFuer(b, document.KanteAbriss)(abs, d, stand)
	if err != nil {
		t.Fatal(err)
	}
	defer fort.Rettung.Ende()
	if fort.Fortgesetzt || fort.Dokument == d || fort.Dokument.KannRueckgaengig() {
		t.Errorf("geänderte Datei: fortgesetzt %v, gemerktes Dokument %v", fort.Fortgesetzt, fort.Dokument == d)
	}
	if n := len(fort.Dokument.Objekte()); n != 1 {
		t.Errorf("frisch gelesen: %d Objekte, Soll 1", n)
	}
}

func TestAnbindungHatFortsetzen(t *testing.T) {
	if anbindungFuer(nil, document.KanteAbriss).Fortsetzen == nil {
		t.Error("Anbindung ohne Fortsetzen: der Verlauf ginge beim Wechsel verloren")
	}
}
