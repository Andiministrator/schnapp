// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/document"
	"github.com/Andiministrator/schnapp/fileformat"
	"github.com/Andiministrator/schnapp/i18n"
	"github.com/Andiministrator/schnapp/library"
	"github.com/Andiministrator/schnapp/ui"
)

// Nach der Aufnahme öffnet der Editor mit dem abgelegten Pfad, erst nachdem der Pfad auf stdout steht.
func TestEditorNachDerAufnahme(t *testing.T) {
	i18n.Setze(nil)
	for _, mitEditor := range []bool{true, false} {
		l := neuerLauf(t)
		var geoeffnet []string
		l.an.editor = func(p string) error { geoeffnet = append(geoeffnet, p); return nil }
		l.overlayErg = ui.OverlayErgebnis{Monitor: 0, Bereich: image.Rect(0, 0, 2, 2)}
		code, aus, fehler := l.fuehre(Auftrag{Modus: ModusBereich, Editor: mitEditor})
		if code != exitOK || aus != "/ablage/2026-11/datei.png\n" || fehler != "" {
			t.Fatalf("Code %d, aus %q, fehler %q", code, aus, fehler)
		}
		if mitEditor && (len(geoeffnet) != 1 || geoeffnet[0] != "/ablage/2026-11/datei.png") {
			t.Errorf("mit Editor: %q", geoeffnet)
		}
		if !mitEditor && len(geoeffnet) != 0 {
			t.Errorf("ohne Editor geöffnet: %q", geoeffnet)
		}
	}
}

// Scheitert der Editor nach der Aufnahme, ist das nur eine Warnung: Die Aufnahme ist gespeichert.
func TestEditorFehlerNachDerAufnahme(t *testing.T) {
	i18n.Setze(nil)
	l := neuerLauf(t)
	l.an.editor = func(string) error { return errors.New("kein Wayland") }
	l.overlayErg = ui.OverlayErgebnis{Monitor: 0, Bereich: image.Rect(0, 0, 2, 2)}
	code, aus, fehler := l.fuehre(Auftrag{Modus: ModusBereich, Editor: true})
	if code != exitOK || aus == "" || !strings.Contains(fehler, "The editor could not be opened or saved: kein Wayland") {
		t.Errorf("Code %d, aus %q, fehler %q", code, aus, fehler)
	}
}

// schnapp DATEI öffnet nur den Editor: keine Aufnahme, nichts auf stdout.
func TestOeffnenOhneAufnahme(t *testing.T) {
	i18n.Setze(nil)
	l := neuerLauf(t)
	l.an.quelle = nil // eine Aufnahme würde hier abstürzen
	var geoeffnet string
	l.an.editor = func(p string) error { geoeffnet = p; return nil }
	code, aus, fehler := l.fuehre(Auftrag{Modus: ModusOeffnen, Datei: "bild.png"})
	if code != exitOK || aus != "" || fehler != "" || geoeffnet != "bild.png" {
		t.Errorf("Code %d, aus %q, fehler %q, geöffnet %q", code, aus, fehler, geoeffnet)
	}
	l.an.editor = func(string) error { return errors.New("gibt es nicht") }
	if code, _, fehler := l.fuehre(Auftrag{Modus: ModusOeffnen, Datei: "x"}); code != exitFehler || !strings.Contains(fehler, "Cannot open x: gibt es nicht") {
		t.Errorf("Fehlerfall: Code %d, fehler %q", code, fehler)
	}
}

func testBibliothek(t *testing.T) *library.Bibliothek {
	t.Helper()
	heim := t.TempDir()
	env := map[string]string{"HOME": heim}
	o, err := library.OrteAus(func(k string) string { return env[k] })
	if err != nil {
		t.Fatal(err)
	}
	b := library.Neu(o)
	b.Jetzt = func() time.Time { return time.Date(2026, 11, 19, 20, 0, 0, 0, time.UTC) }
	return b
}

func mitKasten(t *testing.T, d *document.Dokument) {
	t.Helper()
	if _, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(1, 1), document.Pt(3, 3)),
		Strich: document.Strich{Farbe: color.NRGBA{R: 255, A: 255}, Breite: 1}}); err != nil {
		t.Fatal(err)
	}
}

// Eine Datei außerhalb der Bibliothek bleibt unberührt; das erste Speichern legt eine neue Aufnahme an, jedes weitere
// schreibt diese fort.
func TestSpeichernDateiVonAussen(t *testing.T) {
	b := testBibliothek(t)
	aussen := filepath.Join(t.TempDir(), "fremd.png")
	f, err := os.Create(aussen)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewNRGBA(image.Rect(0, 0, 6, 4))); err != nil {
		t.Fatal(err)
	}
	f.Close()
	vorher, _ := os.ReadFile(aussen)
	g, err := fileformat.Oeffne(aussen)
	if err != nil {
		t.Fatal(err)
	}
	speichern := speichernIn(b, aussen, g)
	mitKasten(t, g.Dokument)
	erg, err := speichern(g.Dokument)
	if err != nil {
		t.Fatal(err)
	}
	// Erstes Speichern einer Datei von außen: neue Aufnahme, sichtbar genannt, keine Warnung (F-116).
	if erg.Datei == "" || erg.Warnung || !strings.Contains(erg.Hinweis, erg.Datei) {
		t.Errorf("Ergebnis %+v (Soll: neuer Name im Hinweis, keine Warnung)", erg)
	}
	mitKasten(t, g.Dokument)
	if erg, err := speichern(g.Dokument); err != nil || erg != (ui.Gespeichert{}) {
		t.Fatalf("zweites Speichern: %+v, %v (Soll: gewöhnlich zurückgeschrieben)", erg, err)
	}
	if nachher, _ := os.ReadFile(aussen); string(nachher) != string(vorher) {
		t.Error("Datei außerhalb verändert")
	}
	idx, err := b.Laden()
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.Eintraege) != 1 || !idx.Eintraege[0].Bearbeitet {
		t.Fatalf("Index %+v (Soll: genau eine neue, bearbeitete Aufnahme)", idx.Eintraege)
	}
	neu, err := fileformat.Oeffne(filepath.Join(b.Orte.Ablage, filepath.FromSlash(idx.Eintraege[0].Pfad)))
	if err != nil || neu.Fremd || len(neu.Dokument.Objekte()) != 2 {
		t.Errorf("neue Aufnahme: %+v, %v", neu, err)
	}
}

// Eine Aufnahme der Bibliothek wird an ihren Platz zurückgeschrieben, kein zweiter Eintrag.
func TestSpeichernAufnahmeDerBibliothek(t *testing.T) {
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
	mitKasten(t, g.Dokument)
	if erg, err := speichernIn(b, abs, g)(g.Dokument); err != nil || erg != (ui.Gespeichert{}) {
		t.Fatalf("%+v, %v", erg, err)
	}
	idx, _ := b.Laden()
	if len(idx.Eintraege) != 1 || idx.Eintraege[0].Pfad != e.Pfad || !idx.Eintraege[0].Bearbeitet {
		t.Errorf("Index %+v", idx.Eintraege)
	}
	if neu, _ := fileformat.Oeffne(abs); len(neu.Dokument.Objekte()) != 1 {
		t.Error("Objekt nicht gespeichert")
	}
}

// Verschwindet die Bibliotheksdatei, während der Editor offen ist, landet die Bearbeitung als neue Aufnahme
// (Kritik A-1) — nichts geht verloren.
func TestSpeichernDateiWegWaehrendEditor(t *testing.T) {
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
	speichern := speichernIn(b, abs, g)
	if err := os.Rename(abs, abs+".weg"); err != nil {
		t.Fatal(err)
	}
	mitKasten(t, g.Dokument)
	erg, err := speichern(g.Dokument)
	if err != nil {
		t.Fatalf("Speichern: %v", err)
	}
	if erg.Datei == "" || !erg.Warnung || !strings.Contains(erg.Hinweis, erg.Datei) {
		t.Errorf("Ergebnis %+v (Soll: Warnung mit neuem Namen, F-116)", erg)
	}
	idx, _ := b.Laden()
	gefunden := false
	for _, ein := range idx.Eintraege {
		if !ein.Bearbeitet {
			continue
		}
		neu, err := fileformat.Oeffne(filepath.Join(b.Orte.Ablage, filepath.FromSlash(ein.Pfad)))
		if err == nil && len(neu.Dokument.Objekte()) == 1 {
			gefunden = true
		}
	}
	if !gefunden {
		t.Errorf("keine neue Aufnahme im Index: %+v", idx.Eintraege)
	}
}

// Bekannte Fehler erscheinen übersetzt statt als Paketfehler (Kritik A-9).
func TestNutzerFehler(t *testing.T) {
	i18n.Setze(nil)
	dir := t.TempDir()
	_, fehlt := fileformat.Oeffne(filepath.Join(dir, "fehlt.png"))
	_, ordner := fileformat.Oeffne(dir)
	text := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(text, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, keinBild := fileformat.Oeffne(text)
	for err, soll := range map[error]string{
		fehlt:              "The file does not exist.",
		ordner:             "This is not a regular file.",
		keinBild:           "This is neither a PNG nor a JPEG image.",
		ui.ErrOverlayOffen: "The selection overlay did not close in time.",
		speicherFehler(fmt.Errorf("x: %w", os.ErrPermission)):                             "No permission to write the file.",
		speicherFehler(errors.Join(errors.New("a"), fmt.Errorf("b: %w", syscall.ENOSPC))): "Not enough disk space.",
		speicherFehler(errors.New("/irgendwo/datei.png: kaputt")):                         "The file could not be written.",
		errors.New("x"): "x",
	} {
		if got := nutzerFehler(err); got != soll {
			t.Errorf("%v: %q, Soll %q", err, got, soll)
		}
	}
}

// Scheitert das Zurückschreiben aus anderem Grund als „Datei weg“ (hier: Monatsordner nicht beschreibbar), weicht
// Speichern auf eine neue Aufnahme aus und sagt es als Warnung mit dem neuen Namen (V-1, F-116); die alte Datei
// bleibt unverändert. Das Ausweichen bringt den eigenen Monatsordner dabei wieder auf 0700 (ADR-0016).
func TestSpeichernSchreibfehlerWeichtSichtbarAus(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("als root wirken Ordnerrechte nicht")
	}
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
	monat := filepath.Dir(abs)
	if err := os.Chmod(monat, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(monat, 0o700) })
	speichern := speichernIn(b, abs, g)
	mitKasten(t, g.Dokument)
	erg, err := speichern(g.Dokument)
	if err != nil {
		t.Fatalf("Speichern: %v", err)
	}
	if erg.Datei == "" || erg.Datei == filepath.Base(abs) || !erg.Warnung ||
		!strings.Contains(erg.Hinweis, "previous file could not be updated") || !strings.Contains(erg.Hinweis, erg.Datei) {
		t.Errorf("Ergebnis %+v (Soll: Warnung, neuer Name)", erg)
	}
	if alt, err := fileformat.Oeffne(abs); err != nil || len(alt.Dokument.Objekte()) != 0 {
		t.Errorf("alte Datei: %v (Soll: unverändert ohne Objekte)", err)
	}
}
