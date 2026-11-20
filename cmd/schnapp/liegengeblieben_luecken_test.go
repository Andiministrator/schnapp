// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/document"
	"github.com/Andiministrator/schnapp/fileformat"
	"github.com/Andiministrator/schnapp/i18n"
	"github.com/Andiministrator/schnapp/library"
	"github.com/Andiministrator/schnapp/ui"
)

// F-243: Restzweige von F-221 (ADR-0022 Punkt 4) — Vormerken nach gescheitertem Entfernen im selben Speichern und
// „die aktuelle Aufnahme wird nie entfernt“.

// Scheitert im selben Speichern das Zurückschreiben und danach das Entfernen der alten Datei, bleibt sie vorgemerkt:
// Erst eine weitere Schwärzung versucht es erneut, ein Speichern ohne neue Schwärzung nicht.
func TestF243EntfernenScheitertImSelbenSpeichernBleibtVorgemerkt(t *testing.T) {
	_, alt, abs, g, speichern := bibliotheksAufnahme(t)
	var versuche []string
	vorher := entfernen
	entfernen = func(_ *library.Bibliothek, rel string, _ library.Stand) (bool, error) {
		versuche = append(versuche, rel)
		return false, os.ErrPermission
	}
	t.Cleanup(func() { entfernen = vorher })
	sperre(t, abs)
	if err := g.Dokument.Schwaerzen(geheimRahmen, schwarz); err != nil {
		t.Fatal(err)
	}
	erg := mussSpeichern(t, speichern, g)
	if !erg.Warnung || !strings.Contains(erg.Hinweis, "Delete it yourself") || len(versuche) != 1 || versuche[0] != alt {
		t.Fatalf("erstes Speichern: %+v, Versuche %v", erg, versuche)
	}
	if erg := mussSpeichern(t, speichern, g); erg.Warnung || len(versuche) != 1 {
		t.Errorf("ohne neue Schwärzung: %+v, Versuche %v", erg, versuche)
	}
	if err := g.Dokument.Schwaerzen(geheimRahmen.Inset(5), schwarz); err != nil {
		t.Fatal(err)
	}
	erg = mussSpeichern(t, speichern, g)
	if len(versuche) != 2 || versuche[1] != alt {
		t.Fatalf("weitere Schwärzung versucht das Entfernen nicht erneut: Versuche %v", versuche)
	}
	if !erg.Warnung || !strings.Contains(erg.Hinweis, alt) || !strings.Contains(erg.Hinweis, "Delete it yourself") {
		t.Errorf("weitere Schwärzung: %+v", erg)
	}
}

// Fehlt die bisherige Datei und bekommt die neue Aufnahme denselben Namen, entfernt das Speichern nach der Schwärzung
// nichts: Der Name gehört jetzt der gerade geschriebenen Aufnahme.
func TestF243NeueAufnahmeMitAltemNamenBleibt(t *testing.T) {
	i18n.Setze(nil)
	b := testBibliothek(t)
	d, err := document.Neu(geheimBild())
	if err != nil {
		t.Fatal(err)
	}
	// Erstellt zur Zeit der Uhr der Bibliothek: Eine neue Aufnahme bekäme denselben Namen, sobald er frei ist.
	e, err := b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{Erstellt: b.Jetzt()})
	if err != nil {
		t.Fatal(err)
	}
	abs := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
	g, err := fileformat.Oeffne(abs)
	if err != nil {
		t.Fatal(err)
	}
	speichern := speichernIn(b, abs, g, standVon(t, abs))
	var versuche []string
	vorher := entfernen
	entfernen = func(bb *library.Bibliothek, rel string, st library.Stand) (bool, error) {
		versuche = append(versuche, rel)
		return vorher(bb, rel, st)
	}
	t.Cleanup(func() { entfernen = vorher })
	if err := os.Remove(abs); err != nil { // weg (Papierkorb, anderes Programm)
		t.Fatal(err)
	}
	if err := g.Dokument.Schwaerzen(geheimRahmen, schwarz); err != nil {
		t.Fatal(err)
	}
	erg := mussSpeichern(t, speichern, g)
	if erg.Datei != filepath.Base(abs) {
		t.Fatalf("Vorbedingung: neue Aufnahme %q trägt nicht den alten Namen %q (%+v)", erg.Datei, filepath.Base(abs), erg)
	}
	if len(versuche) != 0 {
		t.Errorf("Entfernen versucht für %v, obwohl der Name der neuen Aufnahme gehört", versuche)
	}
	if n := musterIn(t, abs); n != 0 {
		t.Errorf("neue Aufnahme zeigt %d Musterpixel", n)
	}
}

// Der Rückblick entfernt nie die gerade geschriebene Aufnahme, auch wenn eine vorgemerkte Fassung ihren Namen trägt
// (Name wiederverwendet), und vergisst die Vormerkung dann.
func TestF243RueckblickEntferntNieDieAktuelle(t *testing.T) {
	i18n.Setze(nil)
	b := testBibliothek(t)
	d, err := document.Neu(geheimBild())
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Schwaerzen(geheimRahmen, schwarz); err != nil {
		t.Fatal(err)
	}
	vorher := entfernen
	entfernen = func(_ *library.Bibliothek, rel string, _ library.Stand) (bool, error) {
		t.Errorf("Rückblick entfernt %q", rel)
		return true, nil
	}
	t.Cleanup(func() { entfernen = vorher })
	reste := []rest{{rel: "2026-11/aktuell.png", ort: "schnapp/2026-11/aktuell.png"}} // bei 0: vor der Schwärzung
	erg, bleiben := rueckblick(b, d, reste, "2026-11/aktuell.png", ui.Gespeichert{})
	if erg.Warnung || erg.Hinweis != "" || len(bleiben) != 0 {
		t.Errorf("Ergebnis %+v, bleiben %v", erg, bleiben)
	}
}
