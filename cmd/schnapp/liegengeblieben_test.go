// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"errors"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/ui"
)

// F-221: Fassungen, die bei einem früheren Speichern liegen blieben, zeigen eine spätere Schwärzung nicht. Der
// Rückblick nach dem Speichern entfernt eigene Aufnahmen (ADR-0022 Punkt 4) und warnt bei allen anderen.

// mussSpeichern speichert und bricht bei einem Fehler ab.
func mussSpeichern(t *testing.T, speichern ui.Speichern, g *fileformat.Gelesen) ui.Gespeichert {
	t.Helper()
	erg, err := speichern(g.Dokument)
	if err != nil {
		t.Fatal(err)
	}
	return erg
}

// Datei von außen: erst ohne Schwärzung als neue Aufnahme gespeichert, dann geschwärzt — die Warnung nennt die
// ursprüngliche Datei, einmal je neuer Schwärzung, und lässt sie unberührt.
func TestF221AussenSpaeterGeschwaerztWarnt(t *testing.T) {
	i18n.Setze(nil)
	b := testBibliothek(t)
	aussen := filepath.Join(t.TempDir(), "bild.png")
	var buf bytes.Buffer
	if err := png.Encode(&buf, geheimBild()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(aussen, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	g, err := fileformat.Oeffne(aussen)
	if err != nil {
		t.Fatal(err)
	}
	speichern := speichernIn(b, aussen, g, standVon(t, aussen))
	mitKasten(t, g.Dokument)
	if erg := mussSpeichern(t, speichern, g); erg.Warnung {
		t.Fatalf("erstes Speichern ohne Schwärzung warnt: %+v", erg)
	}
	if err := g.Dokument.Schwaerzen(geheimRahmen, schwarz); err != nil {
		t.Fatal(err)
	}
	erg := mussSpeichern(t, speichern, g)
	if !erg.Warnung || !strings.Contains(erg.Hinweis, "bild.png") || !strings.Contains(erg.Hinweis, "unredacted") {
		t.Errorf("zweites Speichern: %+v", erg)
	}
	if strings.Contains(erg.Hinweis, filepath.Dir(aussen)) {
		t.Errorf("Hinweis nennt den Ordner: %q", erg.Hinweis)
	}
	if erg := mussSpeichern(t, speichern, g); erg.Warnung {
		t.Errorf("drittes Speichern ohne neue Schwärzung warnt erneut: %+v", erg)
	}
	if roh, _ := os.ReadFile(aussen); !bytes.Equal(roh, buf.Bytes()) {
		t.Error("Datei von außen verändert")
	}
}

// Aufnahme der Bibliothek: Zurückschreiben scheitert ohne Schwärzung (Ausweichen, alte Datei bleibt), danach
// geschwärzt — die alte Datei und ihre Vorschau sind weg, die Warnung sagt es.
func TestF221AusgewichenSpaeterGeschwaerztEntfernt(t *testing.T) {
	b, alt, abs, g, speichern := bibliotheksAufnahme(t)
	vorschau := b.VorschauPfad(alt)
	sperre(t, abs)
	mitKasten(t, g.Dokument)
	erg := mussSpeichern(t, speichern, g)
	if erg.Datei == "" {
		t.Fatalf("erstes Speichern wich nicht aus: %+v", erg)
	}
	if err := os.Chmod(filepath.Dir(abs), 0o700); err != nil {
		t.Fatal(err)
	}
	if n := musterIn(t, abs); n == 0 {
		t.Fatal("Vorbedingung: alte Datei zeigt das Muster")
	}
	if err := g.Dokument.Schwaerzen(geheimRahmen, schwarz); err != nil {
		t.Fatal(err)
	}
	erg = mussSpeichern(t, speichern, g)
	if !erg.Warnung || !strings.Contains(erg.Hinweis, "was removed") || !strings.Contains(erg.Hinweis, alt) {
		t.Errorf("zweites Speichern: %+v", erg)
	}
	for _, p := range []string{abs, vorschau} {
		if _, err := os.Stat(p); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s noch da: %v", filepath.Base(p), err)
		}
	}
	idx, _ := b.Laden()
	if len(idx.Eintraege) != 1 || idx.Eintraege[0].Pfad == alt {
		t.Fatalf("Index %+v (Soll: nur die neue Aufnahme)", idx.Eintraege)
	}
	if n := musterIn(t, filepath.Join(b.Orte.Ablage, filepath.FromSlash(idx.Eintraege[0].Pfad))); n != 0 {
		t.Errorf("neue Aufnahme: %d Musterpixel", n)
	}
	if erg := mussSpeichern(t, speichern, g); erg.Warnung {
		t.Errorf("drittes Speichern warnt erneut: %+v", erg)
	}
}

// Schlägt das Entfernen fehl, bleibt die Fassung vorgemerkt: Die Warnung nennt sie, und erst eine weitere Schwärzung
// versucht es erneut. Schwärzen und Rückgängig vor dem Speichern lösen nichts aus (F-187).
func TestF221EntfernenScheitertBleibtVorgemerkt(t *testing.T) {
	_, alt, abs, g, speichern := bibliotheksAufnahme(t)
	var versuche []string
	vorher := entfernen
	entfernen = func(_ *library.Bibliothek, rel string, _ library.Stand) (bool, error) {
		versuche = append(versuche, rel)
		return false, os.ErrPermission
	}
	t.Cleanup(func() { entfernen = vorher })
	sperre(t, abs)
	mitKasten(t, g.Dokument)
	mussSpeichern(t, speichern, g)
	if err := g.Dokument.Schwaerzen(geheimRahmen, schwarz); err != nil {
		t.Fatal(err)
	}
	if err := g.Dokument.Rueckgaengig(); err != nil {
		t.Fatal(err)
	}
	if erg := mussSpeichern(t, speichern, g); erg.Warnung || len(versuche) != 0 {
		t.Fatalf("zurückgenommene Schwärzung: %+v, Versuche %v", erg, versuche)
	}
	if err := g.Dokument.Schwaerzen(geheimRahmen, schwarz); err != nil {
		t.Fatal(err)
	}
	erg := mussSpeichern(t, speichern, g)
	if !erg.Warnung || !strings.Contains(erg.Hinweis, "Delete it yourself") || !strings.Contains(erg.Hinweis, alt) {
		t.Errorf("Speichern nach Schwärzung: %+v", erg)
	}
	if erg := mussSpeichern(t, speichern, g); erg.Warnung || len(versuche) != 1 {
		t.Errorf("ohne neue Schwärzung: %+v, Versuche %v", erg, versuche)
	}
	if err := g.Dokument.Schwaerzen(geheimRahmen.Inset(5), schwarz); err != nil {
		t.Fatal(err)
	}
	if erg := mussSpeichern(t, speichern, g); !erg.Warnung || len(versuche) != 2 || versuche[1] != alt {
		t.Errorf("nach weiterer Schwärzung: %+v, Versuche %v", erg, versuche)
	}
}

// Konflikt ohne Schwärzung, danach geschwärzt: Die andere Fassung gehört nicht Schnapp allein — nicht entfernen,
// aber warnen.
func TestF221KonfliktSpaeterGeschwaerztWarnt(t *testing.T) {
	_, alt, abs, g, speichern := bibliotheksAufnahme(t)
	vorher := entfernen
	entfernen = func(*library.Bibliothek, string, library.Stand) (bool, error) {
		t.Error("andere Fassung eines Konflikts entfernt")
		return false, nil
	}
	t.Cleanup(func() { entfernen = vorher })
	spaeter := time.Now().Add(time.Hour)
	if err := os.Chtimes(abs, spaeter, spaeter); err != nil { // „von anderer Seite geändert“
		t.Fatal(err)
	}
	roh, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	mitKasten(t, g.Dokument)
	if erg := mussSpeichern(t, speichern, g); !strings.Contains(erg.Hinweis, "changed elsewhere") {
		t.Fatalf("Vorbedingung Konflikt: %+v", erg)
	}
	if err := g.Dokument.Schwaerzen(geheimRahmen, schwarz); err != nil {
		t.Fatal(err)
	}
	erg := mussSpeichern(t, speichern, g)
	if !erg.Warnung || !strings.Contains(erg.Hinweis, "may still show") || !strings.Contains(erg.Hinweis, alt) {
		t.Errorf("Speichern nach Schwärzung: %+v", erg)
	}
	if jetzt, _ := os.ReadFile(abs); !bytes.Equal(jetzt, roh) {
		t.Error("andere Fassung verändert")
	}
}
