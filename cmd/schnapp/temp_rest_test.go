// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/document"
	"github.com/Andiministrator/schnapp/fileformat"
	"github.com/Andiministrator/schnapp/i18n"
	"github.com/Andiministrator/schnapp/library"
	"github.com/Andiministrator/schnapp/ui"
)

// Z-4 (Verify Fable zu #114): Liegen gebliebene temporäre Kopien (library.ErrTempRest, F-274) dürfen nirgends
// verschluckt werden. Die Tests bauen den Rest im eigenen Testverzeichnis: eine eigene temporäre Datei mit Zeit weit in
// der Zukunft im Vorschau-Ordner (die Bibliothek räumt sie nicht und meldet sie).

// zukunftsRest legt im Vorschau-Ordner der Bibliothek einen Rest mit Änderungszeit in 48 Stunden an.
func zukunftsRest(t *testing.T, b *library.Bibliothek) {
	t.Helper()
	if err := os.MkdirAll(b.Orte.Vorschau, 0o700); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(b.Orte.Vorschau, ".schnapp-tmp-zukunft")
	if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	z := time.Now().Add(48 * time.Hour)
	if err := os.Chtimes(p, z, z); err != nil {
		t.Fatal(err)
	}
}

// Speichern an Ort und Stelle: Bleibt nur eine temporäre Kopie, sagt der Hinweis es; bleibt auch die alte Vorschau
// liegen, nennt er beides (früher nannte er nur die Vorschau).
func TestSpeichernNenntTempRestUndVorschau(t *testing.T) {
	b, rel, _, g, speichern := bibliotheksAufnahme(t)
	zukunftsRest(t, b)
	if err := g.Dokument.Schwaerzen(geheimRahmen, schwarz); err != nil {
		t.Fatal(err)
	}
	erg, err := speichern(g.Dokument)
	if err != nil {
		t.Fatalf("Speichern: %v", err)
	}
	if !erg.Warnung || erg.Hinweis != "Saved, but a leftover temporary copy of this capture could not be removed and may still show the unredacted content." {
		t.Errorf("nur Rest: %+v", erg)
	}

	blockiereVorschau(t, b, rel)
	mitKasten(t, g.Dokument)
	erg, err = speichern(g.Dokument)
	if err != nil {
		t.Fatalf("Speichern: %v", err)
	}
	for _, soll := range []string{"old preview of this capture could not be removed", "leftover temporary copy of this capture could not be removed"} {
		if !erg.Warnung || !strings.Contains(erg.Hinweis, soll) {
			t.Errorf("Hinweis %q nennt nicht %q", erg.Hinweis, soll)
		}
	}
	if erg.Datei != "" {
		t.Errorf("Ergebnis %+v: Soll an Ort und Stelle gespeichert, ohne neue Datei", erg)
	}
}

// Ohne Schwärzung ist der Hinweis nüchterner, mit beiden Meldungen bleibt er vollständig.
func TestSpeichernNenntTempRestOhneSchwaerzen(t *testing.T) {
	i18n.Setze(nil)
	b := testBibliothek(t)
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 5, 5)))
	if err != nil {
		t.Fatal(err)
	}
	e, err := b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{})
	if err != nil {
		t.Fatal(err)
	}
	abs := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
	g, err := fileformat.Oeffne(abs)
	if err != nil {
		t.Fatal(err)
	}
	zukunftsRest(t, b)
	mitKasten(t, g.Dokument)
	erg, err := speichernIn(b, abs, g, standVon(t, abs))(g.Dokument)
	if err != nil || !erg.Warnung || erg.Hinweis != "Saved, but a leftover temporary copy of this capture could not be removed." {
		t.Errorf("nur Rest: %+v, %v", erg, err)
	}
	blockiereVorschau(t, b, e.Pfad)
	mitKasten(t, g.Dokument)
	erg, err = speichernIn(b, abs, g, standVon(t, abs))(g.Dokument)
	want := "Saved, but the old preview of this capture could not be removed. Also, a leftover temporary copy of this capture could not be removed."
	if err != nil || erg.Hinweis != want {
		t.Errorf("beide: %+v, %v (Soll %q)", erg, err, want)
	}
}

// Das Ablegen beim Ausweichen (Datei von außen): Eine liegen gebliebene Kopie der neuen Aufnahme wird genannt.
func TestAblegenBeimAusweichenNenntTempRest(t *testing.T) {
	i18n.Setze(nil)
	b := testBibliothek(t)
	aussen := filepath.Join(t.TempDir(), "fremd.png")
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, 6, 4))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(aussen, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	g, err := fileformat.Oeffne(aussen)
	if err != nil {
		t.Fatal(err)
	}
	zukunftsRest(t, b)
	mitKasten(t, g.Dokument)
	erg, err := speichernIn(b, aussen, g, standVon(t, aussen))(g.Dokument)
	if err != nil {
		t.Fatal(err)
	}
	if !erg.Warnung || erg.Datei == "" || !strings.Contains(erg.Hinweis, "A temporary copy of the new capture could not be removed.") {
		t.Errorf("Ergebnis %+v (Soll: neue Aufnahme und Warnung zur Kopie)", erg)
	}
}

// Z-2: Das Entfernen der alten Aufnahme nach dem Ausweichen meldet liegen gebliebene Kopien der alten Datei: Der
// Hinweis bleibt „entfernt“ und nennt die Kopie zusätzlich (statt „Delete it yourself“ für eine Datei, die weg ist).
func TestEntfernenMitTempRestWirdGenannt(t *testing.T) {
	_, _, abs, g, speichern := bibliotheksAufnahme(t)
	vorher := entfernen
	entfernen = func(*library.Bibliothek, string, library.Stand) (bool, error) {
		return true, errors.Join(library.ErrTempRest, library.ErrIndex)
	}
	t.Cleanup(func() { entfernen = vorher })
	if err := g.Dokument.Schwaerzen(geheimRahmen, schwarz); err != nil {
		t.Fatal(err)
	}
	sperre(t, abs)
	erg, err := speichern(g.Dokument)
	if err != nil {
		t.Fatal(err)
	}
	if !erg.Warnung || !strings.Contains(erg.Hinweis, "so it was removed") ||
		!strings.Contains(erg.Hinweis, "A temporary copy of the previous file could not be removed") ||
		strings.Contains(erg.Hinweis, "Delete it yourself") {
		t.Errorf("Ergebnis %+v", erg)
	}
}

// Z-2: Dasselbe im Rückblick auf früher liegen gebliebene Fassungen (ADR-0022 Nachtrag 2).
func TestRueckblickNenntTempRest(t *testing.T) {
	i18n.Setze(nil)
	vorher := entfernen
	entfernen = func(*library.Bibliothek, string, library.Stand) (bool, error) {
		return true, library.ErrTempRest
	}
	t.Cleanup(func() { entfernen = vorher })
	d, err := document.Neu(geheimBild())
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Schwaerzen(geheimRahmen, schwarz); err != nil {
		t.Fatal(err)
	}
	reste := []rest{{rel: "2026-11/alt.png", ort: "schnapp/2026-11/alt.png", bei: 0}}
	erg, bleiben := rueckblick(nil, d, reste, "2026-11/neu.png", ui.Gespeichert{})
	if !erg.Warnung || !strings.Contains(erg.Hinweis, "still showed the unredacted content and was removed") ||
		!strings.Contains(erg.Hinweis, "A temporary copy of the earlier file schnapp/2026-11/alt.png could not be removed") {
		t.Errorf("Ergebnis %+v", erg)
	}
	if len(bleiben) != 0 {
		t.Errorf("Fassung bleibt vorgemerkt: %+v", bleiben)
	}
}

// Z-4: Das echte Ablegen der Aufnahme reicht ErrTempRest zusammen mit dem Pfad weiter (statt es zu verschlucken); jeder
// andere Folgefehler (Index, Vorschau) bleibt still. Eigene Umgebung unter t.TempDir().
func TestEchtesAblegenReichtTempRestWeiter(t *testing.T) {
	heim := t.TempDir()
	t.Setenv("HOME", heim)
	for _, v := range []string{"XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME"} {
		t.Setenv(v, "")
	}
	orte, err := library.OrteAus(os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	zukunftsRest(t, library.Neu(orte))
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 4, 3)))
	if err != nil {
		t.Fatal(err)
	}
	pfad, _, err := echteAnschluesse().ablegen(d)
	if pfad == "" || !errors.Is(err, library.ErrTempRest) {
		t.Fatalf("Pfad %q, Fehler %v (Soll: Pfad der Aufnahme und ErrTempRest)", pfad, err)
	}
	if _, serr := os.Stat(pfad); serr != nil {
		t.Errorf("Aufnahme fehlt: %v", serr)
	}
}

// Z-4: Die Aufnahme meldet eine liegen gebliebene Kopie auf stderr; Pfad und Exit-Code bleiben die einer gelungenen
// Aufnahme.
func TestAufnahmeWarntBeiTempRest(t *testing.T) {
	i18n.Setze(nil)
	l := neuerLauf(t)
	l.an.ablegen = func(d *document.Dokument) (string, library.Stand, error) {
		l.abgelegt = append(l.abgelegt, d)
		return "/ablage/2026-11/datei.png", l.stand, errors.Join(library.ErrTempRest, errors.New("Entfernen scheitert"))
	}
	code, aus, fehler := l.fuehre(Auftrag{Modus: ModusFenster})
	if code != exitOK || aus != "/ablage/2026-11/datei.png\n" {
		t.Errorf("Code %d, aus %q (Soll: Aufnahme gilt)", code, aus)
	}
	if !strings.Contains(fehler, "Warning: a temporary copy of the screenshot could not be removed") || !strings.Contains(fehler, "Entfernen scheitert") {
		t.Errorf("stderr %q (Soll: Warnung mit dem Grund)", fehler)
	}
	if strings.Contains(fehler, "Could not save the screenshot") {
		t.Errorf("stderr %q: Soll kein Speicherfehler sein", fehler)
	}
}
