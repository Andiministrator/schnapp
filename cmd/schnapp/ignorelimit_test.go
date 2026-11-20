// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/document"
	"github.com/Andiministrator/schnapp/fileformat"
	"github.com/Andiministrator/schnapp/i18n"
	"github.com/Andiministrator/schnapp/render"
	"github.com/Andiministrator/schnapp/settings"
)

// schreibeDatei legt d als Arbeitsdatei unter name im Testordner ab und liefert den Pfad.
func schreibeDatei(t *testing.T, dir, name string, d *document.Dokument) string {
	t.Helper()
	pfad := filepath.Join(dir, name)
	f, err := os.Create(pfad)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := fileformat.SchreibeArbeitsdatei(f, d, d.Ansichtsbild(), fileformat.Metadaten{}); err != nil {
		t.Fatal(err)
	}
	return pfad
}

// ueberBudget liefert eine Datei, deren Rastern das Budget überschritte, aber alle Grenzen beim Lesen einhält.
func ueberBudget(t *testing.T, dir string) (pfad string, n int) {
	t.Helper()
	// Je Glyphe „W“ im Grad 4096 kostet das Rastern 4096² Einheiten (wie TestOeffnenLehntZuAufwendigeDateiAb).
	budget := render.GlyphBudget
	n = int(budget/(4096*4096)) + 1
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 10, 10)))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < n; i++ {
		if _, err := d.Hinzufuegen(document.Text{Ort: document.Pt(0, 0), Inhalt: "W", Schriftgrad: 4096,
			Farbe: color.NRGBA{R: 255, A: 255}}); err != nil {
			t.Fatal(err)
		}
	}
	return schreibeDatei(t, dir, "ueber-budget.png", d), n
}

// F-146: Ohne --ignore-limit lehnt das Öffnen eine Datei über dem Rechenbudget ab, und die Meldung nennt den Schalter;
// mit dem Schalter öffnet sie, mit allen Objekten.
func TestOeffnenOhneGrenze(t *testing.T) {
	i18n.Setze(nil)
	pfad, n := ueberBudget(t, t.TempDir())

	_, _, _, err := ladeFuerEditor(pfad, false)
	if !errors.Is(err, render.ErrZuAufwendig) {
		t.Fatalf("ohne Schalter: %v, Soll render.ErrZuAufwendig", err)
	}
	if got := nutzerFehler(err); !strings.Contains(got, "--ignore-limit") {
		t.Errorf("Meldung %q nennt --ignore-limit nicht", got)
	}

	_, g, _, err := ladeFuerEditor(pfad, true)
	if err != nil {
		t.Fatalf("mit Schalter: %v", err)
	}
	if got := len(g.Dokument.Objekte()); got != n {
		t.Errorf("mit Schalter %d Objekte, Soll %d", got, n)
	}
}

// ersetzeSVGBlock ersetzt in einer PNG-Datei den Block snAP durch daten (mit neuer CRC).
func ersetzeSVGBlock(t *testing.T, png []byte, daten []byte) []byte {
	t.Helper()
	var aus bytes.Buffer
	aus.Write(png[:8])
	gefunden := false
	for p := 8; p < len(png); {
		laenge := int(binary.BigEndian.Uint32(png[p:]))
		typ := string(png[p+4 : p+8])
		inhalt := png[p+8 : p+8+laenge]
		if typ == "snAP" {
			inhalt, gefunden = daten, true
		}
		var kopf [4]byte
		binary.BigEndian.PutUint32(kopf[:], uint32(len(inhalt)))
		aus.Write(kopf[:])
		aus.WriteString(typ)
		aus.Write(inhalt)
		crc := crc32.NewIEEE()
		crc.Write([]byte(typ))
		crc.Write(inhalt)
		binary.BigEndian.PutUint32(kopf[:], crc.Sum32())
		aus.Write(kopf[:])
		p += 12 + laenge
	}
	if !gefunden {
		t.Fatal("kein snAP-Block")
	}
	return aus.Bytes()
}

// svgBlock liefert den entpackten snAP-Block (das SVG) einer Arbeitsdatei.
func svgBlock(t *testing.T, png []byte) []byte {
	t.Helper()
	for p := 8; p < len(png); {
		laenge := int(binary.BigEndian.Uint32(png[p:]))
		if string(png[p+4:p+8]) == "snAP" {
			zr, err := zlib.NewReader(bytes.NewReader(png[p+8 : p+8+laenge]))
			if err != nil {
				t.Fatal(err)
			}
			svg, err := io.ReadAll(zr)
			if err != nil {
				t.Fatal(err)
			}
			return svg
		}
		p += 12 + laenge
	}
	t.Fatal("kein snAP-Block")
	return nil
}

// F-146: Der Schalter hebt nur die Budget-Ablehnung auf. Eine Datei mit mehr als MaxObjekte Objekten bleibt auch mit
// --ignore-limit unverändert behandelt: nicht als Arbeitsdatei gelesen, sondern nur als Bild (Grenze beim Lesen, F-127).
func TestOeffnenOhneGrenzeBehaeltLesegrenzen(t *testing.T) {
	i18n.Setze(nil)
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 80, 60)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(1, 1), document.Pt(5, 5)),
		Strich: document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 1}}); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	gut, err := os.ReadFile(schreibeDatei(t, dir, "gut.png", d))
	if err != nil {
		t.Fatal(err)
	}
	svg := svgBlock(t, gut)
	re := regexp.MustCompile(`<rect s:typ="kasten"[^>]*/>`)
	if len(re.FindAll(svg, -1)) != 1 {
		t.Fatalf("Kasten-Element nicht eindeutig im SVG: %s", svg)
	}
	mit := func(name string, n int) string {
		neu := re.ReplaceAllLiteral(svg, []byte(strings.Repeat(string(re.Find(svg)), n)))
		var z bytes.Buffer
		zw := zlib.NewWriter(&z)
		zw.Write(neu)
		zw.Close()
		pfad := filepath.Join(dir, name)
		if err := os.WriteFile(pfad, ersetzeSVGBlock(t, gut, z.Bytes()), 0o600); err != nil {
			t.Fatal(err)
		}
		return pfad
	}

	// Kontrolle: genau MaxObjekte (leichte Kästen, unter dem Budget) öffnet mit und ohne Schalter.
	for _, ohne := range []bool{false, true} {
		if _, g, _, err := ladeFuerEditor(mit("max.png", document.MaxObjekte), ohne); err != nil ||
			len(g.Dokument.Objekte()) != document.MaxObjekte {
			t.Fatalf("MaxObjekte (Schalter %v): %v", ohne, err)
		}
	}
	for _, ohne := range []bool{false, true} {
		// Die Grenze beim Lesen macht die Arbeitsdatei beschädigt; fileformat.Oeffne liest sie dann nur als Bild (ohne
		// die Beschriftungen, Fremd = true), der Schalter ändert daran nichts.
		_, g, _, err := ladeFuerEditor(mit("zuviel.png", document.MaxObjekte+1), ohne)
		if err != nil || !g.Fremd || len(g.Dokument.Objekte()) != 0 {
			t.Errorf("MaxObjekte+1 (Schalter %v): %v, Soll nur als Bild geöffnet (Fremd, ohne Objekte)", ohne, err)
		}
	}
}

// F-146: Der Schalter geht von der Kommandozeile über Auftrag und Ausführung bis zur Ladefunktion; Aufnahmen öffnen
// den Editor nie damit. Der Hilfetext nennt ihn.
func TestOhneGrenzeBisZumEditor(t *testing.T) {
	i18n.Setze(nil)
	a, err := Auswerten([]string{"--ignore-limit", "x.png"})
	if err != nil {
		t.Fatal(err)
	}
	for _, soll := range []bool{true, false} {
		a.OhneGrenze = soll
		l := neuerLauf(t)
		l.an.quelle = nil
		var gesehen []bool
		l.an.editor = func(_ string, ohneGrenze bool) error { gesehen = append(gesehen, ohneGrenze); return nil }
		if code, _, fehler := l.fuehre(plane(a, settings.Voreinstellungen())); code != exitOK || fehler != "" {
			t.Fatalf("Code %d, fehler %q", code, fehler)
		}
		if len(gesehen) != 1 || gesehen[0] != soll {
			t.Errorf("Editor mit ohneGrenze %v, Soll [%v]", gesehen, soll)
		}
	}
	if !strings.Contains(hilfe(), "--ignore-limit") {
		t.Error("Hilfetext nennt --ignore-limit nicht")
	}
}
