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

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/render"
	"github.com/Andiministrator/schnapp/internal/ui"
)

// Nach der Aufnahme öffnet der Editor mit dem abgelegten Pfad, erst nachdem der Pfad auf stdout steht. Er bekommt das
// abgelegte Dokument selbst und startet erst, wenn die Verbindung zur Aufnahmequelle zu ist (F-112); den Weg über die
// Datei (editor) nimmt er nicht.
func TestEditorNachDerAufnahme(t *testing.T) {
	i18n.Setze(nil)
	for _, mitEditor := range []bool{true, false} {
		l := neuerLauf(t)
		var geoeffnet []string
		var dok *document.Dokument
		l.an.editor = func(p string, _ bool) error { t.Errorf("Editor über die Datei geöffnet: %q", p); return nil }
		// Der Stand kommt vom Ablegen, nicht vom Start des Editors (F-110, Verify W-5).
		datei := filepath.Join(t.TempDir(), "stand.png")
		if err := os.WriteFile(datei, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		l.stand = standVon(t, datei)
		l.an.bearbeiten = func(a abgelegt) error {
			geoeffnet = append(geoeffnet, a.pfad)
			dok = a.d
			if a.stand != l.stand {
				t.Error("Editor bekommt nicht den Stand vom Ablegen")
			}
			if !l.q.geschlossen {
				t.Error("Editor startet, während die Aufnahmequelle noch offen ist (F-112)")
			}
			return nil
		}
		l.overlayErg = ui.OverlayErgebnis{Monitor: 0, Bereich: image.Rect(0, 0, 2, 2)}
		code, aus, fehler := l.fuehre(Auftrag{Modus: ModusBereich, Editor: mitEditor})
		if code != exitOK || aus != "/ablage/2026-11/datei.png\n" || fehler != "" {
			t.Fatalf("Code %d, aus %q, fehler %q", code, aus, fehler)
		}
		if mitEditor && (len(geoeffnet) != 1 || geoeffnet[0] != "/ablage/2026-11/datei.png") {
			t.Errorf("mit Editor: %q", geoeffnet)
		}
		if mitEditor && (len(l.abgelegt) != 1 || dok != l.abgelegt[0]) {
			t.Error("Editor bekommt nicht das abgelegte Dokument")
		}
		if !mitEditor && len(geoeffnet) != 0 {
			t.Errorf("ohne Editor geöffnet: %q", geoeffnet)
		}
	}
}

// Das Dokument für den Editor hält das Monitorbild nicht fest (F-112): Sein Original ist eine eigene Kopie des
// Bereichs, nicht ein Ausschnitt, der den ganzen Monitor im Speicher hielte.
func TestEditorDokumentOhneMonitorbild(t *testing.T) {
	i18n.Setze(nil)
	l := neuerLauf(t)
	var dok *document.Dokument
	l.an.bearbeiten = func(a abgelegt) error { dok = a.d; return nil }
	l.overlayErg = ui.OverlayErgebnis{Monitor: 1, Bereich: image.Rect(1, 1, 3, 3)}
	if code, _, fehler := l.fuehre(Auftrag{Modus: ModusBereich, Editor: true}); code != exitOK || dok == nil {
		t.Fatalf("Code %d, fehler %q", code, fehler)
	}
	o := dok.OriginalBild()
	if o.Rect != image.Rect(0, 0, 2, 2) || len(o.Pix) != 2*2*4 || cap(o.Pix) != 2*2*4 {
		t.Errorf("Original %v mit %d/%d Bytes, Soll eigene Kopie 2×2", o.Rect, len(o.Pix), cap(o.Pix))
	}
}

// Scheitert der Editor nach der Aufnahme, ist das nur eine Warnung: Die Aufnahme ist gespeichert.
func TestEditorFehlerNachDerAufnahme(t *testing.T) {
	i18n.Setze(nil)
	l := neuerLauf(t)
	l.an.bearbeiten = func(abgelegt) error { return errors.New("kein Wayland") }
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
	l.an.editor = func(p string, _ bool) error { geoeffnet = p; return nil }
	code, aus, fehler := l.fuehre(Auftrag{Modus: ModusOeffnen, Datei: "bild.png"})
	if code != exitOK || aus != "" || fehler != "" || geoeffnet != "bild.png" {
		t.Errorf("Code %d, aus %q, fehler %q, geöffnet %q", code, aus, fehler, geoeffnet)
	}
	l.an.editor = func(string, bool) error { return errors.New("gibt es nicht") }
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

// standVon merkt den Stand einer Datei wie ladeFuerEditor.
func standVon(t *testing.T, pfad string) library.Stand {
	t.Helper()
	s, err := library.StandVon(pfad)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// F-112: Der Editor nach der Aufnahme bekommt das Dokument aus dem Speicher statt der neu gelesenen Datei. Speichern
// verhält sich wie nach ladeFuerEditor: Es schreibt die Aufnahme an ihren Platz zurück; ist sie inzwischen weg, weicht
// es auf eine neue Aufnahme aus.
func TestSpeichernNachDerAufnahmeOhneNeuLesen(t *testing.T) {
	i18n.Setze(nil)
	b := testBibliothek(t)
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 8, 8)))
	if err != nil {
		t.Fatal(err)
	}
	e, err := b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{})
	if err != nil {
		t.Fatal(err)
	}
	abs := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
	speichern := speichernIn(b, abs, &fileformat.Gelesen{Dokument: d}, standVon(t, abs))
	mitKasten(t, d)
	if erg, err := speichern(d); err != nil || erg != (ui.Gespeichert{}) {
		t.Fatalf("Zurückschreiben: %+v, %v (Soll: gewöhnlich an den Platz)", erg, err)
	}
	if g, err := fileformat.Oeffne(abs); err != nil || g.Fremd || len(g.Dokument.Objekte()) != 1 {
		t.Fatalf("zurückgeschriebene Datei: %v", err)
	}
	if err := os.Remove(abs); err != nil {
		t.Fatal(err)
	}
	mitKasten(t, d)
	erg, err := speichern(d)
	if err != nil || erg.Datei == "" || !strings.Contains(erg.Hinweis, "saved as a new capture") {
		t.Fatalf("Ausweichen: %+v, %v", erg, err)
	}
	g, err := fileformat.Oeffne(filepath.Join(filepath.Dir(abs), erg.Datei))
	if err != nil || len(g.Dokument.Objekte()) != 2 || !g.Meta.Bearbeitet || g.Meta.Erstellt.IsZero() {
		t.Errorf("neue Aufnahme: %+v, %v", g, err)
	}
}

// Verify W-5 (F-112, F-110): Ändert ein anderer Prozess die eben abgelegte Aufnahme, bevor der Editor startet (etwa
// ein Skript, das den Pfad von stdout liest), überschreibt Speichern sie nicht still: Der Stand stammt vom Ablegen.
func TestAenderungZwischenAblegenUndEditor(t *testing.T) {
	i18n.Setze(nil)
	b := testBibliothek(t)
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 8, 8)))
	if err != nil {
		t.Fatal(err)
	}
	e, err := b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{})
	if err != nil {
		t.Fatal(err)
	}
	abs := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
	stand := standVon(t, abs) // wie der Anschluss ablegen, unmittelbar nach dem Schreiben

	// Der andere schreibt einen Kasten in die Datei.
	a2, g2, s2, err := ladeFuerEditor(abs, false)
	if err != nil {
		t.Fatal(err)
	}
	mitKasten(t, g2.Dokument)
	if erg, err := speichernIn(b, a2, g2, s2)(g2.Dokument); err != nil || erg != (ui.Gespeichert{}) {
		t.Fatalf("anderer: %+v, %v", erg, err)
	}

	// Jetzt startet der Editor mit dem Dokument aus dem Speicher und speichert eine eigene Änderung.
	mitKasten(t, d)
	mitKasten(t, d)
	erg, err := speichernIn(b, abs, &fileformat.Gelesen{Dokument: d}, stand)(d)
	if err != nil || !erg.Warnung || !strings.Contains(erg.Hinweis, "changed elsewhere") {
		t.Fatalf("Speichern: %+v, %v (Soll: Konflikt erkannt)", erg, err)
	}
	if g, err := fileformat.Oeffne(abs); err != nil || len(g.Dokument.Objekte()) != 1 {
		t.Errorf("Fassung des anderen überschrieben: %v", err)
	}
}

// F-110: Zwei Editoren auf derselben Aufnahme. Der erste schreibt zurück; der zweite überschreibt das nicht still,
// sondern weicht auf eine neue Aufnahme aus und sagt es als Warnung. Danach schreibt jeder seine Datei fort.
func TestZweiEditorenUeberschreibenSichNicht(t *testing.T) {
	i18n.Setze(nil)
	b := testBibliothek(t)
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 8, 8)))
	if err != nil {
		t.Fatal(err)
	}
	e, err := b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{})
	if err != nil {
		t.Fatal(err)
	}
	abs := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
	abs1, g1, s1, err := ladeFuerEditor(abs, false)
	if err != nil {
		t.Fatal(err)
	}
	abs2, g2, s2, err := ladeFuerEditor(abs, false)
	if err != nil {
		t.Fatal(err)
	}
	speichern1 := speichernIn(b, abs1, g1, s1)
	speichern2 := speichernIn(b, abs2, g2, s2)

	mitKasten(t, g1.Dokument)
	if erg, err := speichern1(g1.Dokument); err != nil || erg != (ui.Gespeichert{}) {
		t.Fatalf("erster Editor: %+v, %v (Soll: gewöhnlich zurückgeschrieben)", erg, err)
	}
	mitKasten(t, g2.Dokument)
	mitKasten(t, g2.Dokument)
	erg, err := speichern2(g2.Dokument)
	if err != nil {
		t.Fatalf("zweiter Editor: %v", err)
	}
	if erg.Datei == "" || erg.Datei == filepath.Base(abs) || !erg.Warnung ||
		!strings.Contains(erg.Hinweis, "changed elsewhere since it was opened") || !strings.Contains(erg.Hinweis, erg.Datei) {
		t.Fatalf("zweiter Editor: %+v (Soll: Warnung „changed elsewhere“, neue Datei)", erg)
	}
	if alt, err := fileformat.Oeffne(abs); err != nil || len(alt.Dokument.Objekte()) != 1 {
		t.Errorf("Datei des ersten Editors: %v (Soll: 1 Objekt, nicht überschrieben)", err)
	}
	neu := filepath.Join(filepath.Dir(abs), erg.Datei)
	if g, err := fileformat.Oeffne(neu); err != nil || len(g.Dokument.Objekte()) != 2 {
		t.Errorf("neue Aufnahme des zweiten: %v (Soll: 2 Objekte)", err)
	}

	// Danach schreibt jeder seine eigene Datei ohne Konflikt fort.
	mitKasten(t, g1.Dokument)
	if erg, err := speichern1(g1.Dokument); err != nil || erg != (ui.Gespeichert{}) {
		t.Errorf("erster Editor, zweites Speichern: %+v, %v", erg, err)
	}
	mitKasten(t, g2.Dokument)
	if erg, err := speichern2(g2.Dokument); err != nil || erg != (ui.Gespeichert{}) {
		t.Errorf("zweiter Editor, zweites Speichern: %+v, %v (Soll: in die neue Aufnahme)", erg, err)
	}
	if g, _ := fileformat.Oeffne(abs); g == nil || len(g.Dokument.Objekte()) != 2 {
		t.Error("erste Datei nach dem zweiten Speichern: Soll 2 Objekte")
	}
	if g, _ := fileformat.Oeffne(neu); g == nil || len(g.Dokument.Objekte()) != 3 {
		t.Error("neue Aufnahme nach dem zweiten Speichern: Soll 3 Objekte")
	}
	idx, err := b.Laden()
	if err != nil || len(idx.Eintraege) != 2 {
		t.Errorf("Index: %d Einträge, %v (Soll 2)", len(idx.Eintraege), err)
	}
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
	speichern := speichernIn(b, aussen, g, standVon(t, aussen))
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

// F-155: Nach dem ersten Speichern einer Datei von außen ist die neue Bibliotheksdatei das Arbeitsziel: Der Titel
// bekommt ihren Namen, die Meldung nennt den Ort (relativ zum Bilderordner, ohne Benutzernamen), und jedes weitere
// Speichern schreibt in dieselbe Datei — keine zweite und dritte Aufnahme, die Ausgangsdatei bleibt unberührt.
func TestSpeichernDateiVonAussenArbeitsziel(t *testing.T) {
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
	speichern := speichernIn(b, aussen, g, standVon(t, aussen))
	mitKasten(t, g.Dokument)
	erg, err := speichern(g.Dokument)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := b.Laden()
	if err != nil || len(idx.Eintraege) != 1 {
		t.Fatalf("Index %+v, %v", idx.Eintraege, err)
	}
	pfad := idx.Eintraege[0].Pfad
	if erg.Datei != filepath.Base(filepath.FromSlash(pfad)) {
		t.Errorf("Titel %q, Soll Name der neuen Datei", erg.Datei)
	}
	soll := "Saved as a new capture in the library: schnapp/" + pfad + " (the original file is unchanged)"
	if erg.Hinweis != soll || strings.Contains(erg.Hinweis, b.Orte.Ablage) || erg.Warnung {
		t.Errorf("Hinweis %q (Soll %q, relativ, keine Warnung)", erg.Hinweis, soll)
	}
	// F-158: Der Hinweis mit dem Ablageort hält das Fenster beim Schließen einmal offen (Warnung ist er nicht).
	if !erg.HaeltOffen() {
		t.Error("der Hinweis auf die neue Aufnahme hält das Fenster nicht offen (F-158)")
	}
	if (ui.Gespeichert{}).HaeltOffen() {
		t.Error("ein gewöhnliches Speichern hält das Fenster offen")
	}
	for i := 0; i < 2; i++ { // zweites und drittes Strg+S
		mitKasten(t, g.Dokument)
		if erg, err := speichern(g.Dokument); err != nil || erg != (ui.Gespeichert{}) {
			t.Fatalf("weiteres Speichern %d: %+v, %v", i, erg, err)
		}
	}
	idx, err = b.Laden()
	if err != nil || len(idx.Eintraege) != 1 || idx.Eintraege[0].Pfad != pfad {
		t.Fatalf("Index %+v, %v (Soll: weiter genau die eine Aufnahme)", idx.Eintraege, err)
	}
	neu, err := fileformat.Oeffne(filepath.Join(b.Orte.Ablage, filepath.FromSlash(pfad)))
	if err != nil || len(neu.Dokument.Objekte()) != 3 {
		t.Errorf("neue Aufnahme: %v (Soll: 3 Objekte)", err)
	}
	if nachher, _ := os.ReadFile(aussen); string(nachher) != string(vorher) {
		t.Error("Datei außerhalb verändert")
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
	if erg, err := speichernIn(b, abs, g, standVon(t, abs))(g.Dokument); err != nil || erg != (ui.Gespeichert{}) {
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
	speichern := speichernIn(b, abs, g, standVon(t, abs))
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
		fmt.Errorf("x: %w", render.ErrZuAufwendig):                                        "This file has too many or too large annotations to open. To open it anyway, use --ignore-limit.",
		speicherFehler(errors.New("/irgendwo/datei.png: kaputt")):                         "The file could not be written.",
		errors.New("x"): "x",
	} {
		if got := nutzerFehler(err); got != soll {
			t.Errorf("%v: %q, Soll %q", err, got, soll)
		}
	}
}

// F-125, F-131, F-132: Eine Arbeitsdatei, deren Rastern das Rechenbudget überschritte, öffnet der Editor nicht
// (Meldung übersetzt); knapp darunter öffnet sie. Abgelehnt wird nur beim Öffnen; Speichern warnt nur (F-138).
// wOrt setzt ein „W“ im Grad 4096 so, dass seine Grundlinie bei y = 6 liegt und seine Tinte (sie beginnt rund 49 Pixel
// rechts vom Ort) ins Bild (10×10) reicht. Bei Ort (0, 0) läge sie ganz unter und rechts neben dem Bild, wäre
// unsichtbar (ObjektSichtbar nach der echten Tinte, F-181) und kostete nichts.
var wOrt = document.Pt(-45, 6-4096)

func TestOeffnenLehntZuAufwendigeDateiAb(t *testing.T) {
	i18n.Setze(nil)
	dir := t.TempDir()
	// Je Glyphe „W“ im Grad 4096 kostet das Rastern 4096² Einheiten; über GlyphBudget wird abgelehnt.
	budget := render.GlyphBudget
	grenze := int(budget / (4096 * 4096))
	for _, fall := range []struct {
		n         int
		abgelehnt bool
	}{{grenze + 1, true}, {grenze - 5, false}} {
		d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 10, 10)))
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < fall.n; i++ {
			if _, err := d.Hinzufuegen(document.Text{Ort: wOrt, Inhalt: "W", Schriftgrad: 4096,
				Farbe: color.NRGBA{R: 255, A: 255}}); err != nil {
				t.Fatal(err)
			}
		}
		pfad := filepath.Join(dir, fmt.Sprintf("w%d.png", fall.n))
		f, err := os.Create(pfad)
		if err != nil {
			t.Fatal(err)
		}
		if err := fileformat.SchreibeArbeitsdatei(f, d, d.Ansichtsbild(), fileformat.Metadaten{}); err != nil {
			t.Fatal(err)
		}
		f.Close()
		_, g, _, err := ladeFuerEditor(pfad, false)
		if !fall.abgelehnt {
			if err != nil || len(g.Dokument.Objekte()) != fall.n {
				t.Errorf("%d Glyphen (unter dem Budget): %v", fall.n, err)
			}
			continue
		}
		if !errors.Is(err, render.ErrZuAufwendig) {
			t.Fatalf("%d Glyphen: %v, Soll render.ErrZuAufwendig", fall.n, err)
		}
		if got, soll := nutzerFehler(err), "This file has too many or too large annotations to open. To open it anyway, use --ignore-limit."; got != soll {
			t.Errorf("Meldung %q, Soll %q", got, soll)
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
	speichern := speichernIn(b, abs, g, standVon(t, abs))
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

// F-138: Wächst ein Dokument im Editor über das Rechenbudget, wird es trotzdem gespeichert (F-131), aber mit Warnung,
// weil ladeFuerEditor es beim nächsten Öffnen ablehnte. Knapp darunter speichert es ohne Hinweis und öffnet wieder.
func TestSpeichernUeberDemBudgetWarnt(t *testing.T) {
	i18n.Setze(nil)
	warnung := "Saved, but the file has too many or too large annotations to be opened again. Remove some before closing."
	// Je Glyphe „W“ im Grad 4096 kostet das Rastern 4096² Einheiten; über GlyphBudget lehnt Pruefe ab.
	budget := render.GlyphBudget
	grenze := int(budget / (4096 * 4096))
	w := document.Text{Ort: wOrt, Inhalt: "W", Schriftgrad: 4096, Farbe: color.NRGBA{R: 255, A: 255}}
	for _, fall := range []struct {
		n      int
		warnen bool
	}{{grenze - 5, false}, {grenze + 1, true}} {
		b := testBibliothek(t)
		d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 10, 10)))
		if err != nil {
			t.Fatal(err)
		}
		e, err := b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{})
		if err != nil {
			t.Fatal(err)
		}
		abs := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
		abs, g, stand, err := ladeFuerEditor(abs, false)
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < fall.n; i++ {
			if _, err := g.Dokument.Hinzufuegen(w); err != nil {
				t.Fatal(err)
			}
		}
		erg, err := speichernIn(b, abs, g, stand)(g.Dokument)
		if err != nil {
			t.Fatalf("%d Glyphen: Speichern %v", fall.n, err)
		}
		_, wieder, _, oeffnen := ladeFuerEditor(abs, false)
		if !fall.warnen {
			if erg != (ui.Gespeichert{}) || oeffnen != nil || len(wieder.Dokument.Objekte()) != fall.n {
				t.Errorf("%d Glyphen (unter dem Budget): %+v, Öffnen %v (Soll: ohne Hinweis, öffnet wieder)", fall.n, erg, oeffnen)
			}
			continue
		}
		if erg.Hinweis != warnung || !erg.Warnung || erg.Datei != "" {
			t.Errorf("%d Glyphen: %+v, Soll Warnung %q ohne neue Datei", fall.n, erg, warnung)
		}
		// Gespeichert ist es (F-131) — nur das Öffnen lehnt ab, wie die Warnung sagt.
		if !errors.Is(oeffnen, render.ErrZuAufwendig) {
			t.Errorf("%d Glyphen: Öffnen %v, Soll render.ErrZuAufwendig", fall.n, oeffnen)
		}
		if neu, err := fileformat.Oeffne(abs); err != nil || len(neu.Dokument.Objekte()) != fall.n {
			t.Errorf("%d Glyphen: nicht gespeichert (%v)", fall.n, err)
		}
	}
	// Muss Speichern ausweichen, steht die Budget-Warnung hinter dem Hinweis auf die neue Aufnahme.
	b := testBibliothek(t)
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 10, 10)))
	if err != nil {
		t.Fatal(err)
	}
	aussen := filepath.Join(t.TempDir(), "aussen.png")
	f, err := os.Create(aussen)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, d.Ansichtsbild()); err != nil {
		t.Fatal(err)
	}
	f.Close()
	g, err := fileformat.Oeffne(aussen)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i <= grenze; i++ {
		if _, err := g.Dokument.Hinzufuegen(w); err != nil {
			t.Fatal(err)
		}
	}
	erg, err := speichernIn(b, aussen, g, standVon(t, aussen))(g.Dokument)
	if err != nil || erg.Datei == "" || !erg.Warnung ||
		!strings.HasPrefix(erg.Hinweis, "Saved as a new capture in the library: schnapp/20") ||
		!strings.HasSuffix(erg.Hinweis, erg.Datei+" (the original file is unchanged) "+warnung) {
		t.Errorf("von außen: %+v, %v", erg, err)
	}
}
