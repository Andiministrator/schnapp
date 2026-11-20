// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/canvas"
	"github.com/Andiministrator/schnapp/document"
	"github.com/Andiministrator/schnapp/fileformat"
	"github.com/Andiministrator/schnapp/i18n"
	"github.com/Andiministrator/schnapp/library"
	"github.com/Andiministrator/schnapp/render"
	"github.com/Andiministrator/schnapp/ui"
)

// geheimBild ist ein graues Bild 120×80 mit einem „geheimen“ Schachbrett aus Magenta und Grün bei (30..90, 20..60):
// Farben, die sonst nirgends vorkommen und die Schwärzen wie Verpixeln (Mosaik: Gesamtmittel ~ Grau ±48) nie erzeugen.
func geheimBild() *image.NRGBA {
	b := image.NewNRGBA(image.Rect(0, 0, 120, 80))
	for y := 0; y < 80; y++ {
		for x := 0; x < 120; x++ {
			c := color.NRGBA{0x80, 0x80, 0x80, 0xFF}
			if x >= 30 && x < 90 && y >= 20 && y < 60 {
				c = color.NRGBA{0xFF, 0x00, 0xFF, 0xFF}
				if (x/3+y/3)%2 == 0 {
					c = color.NRGBA{0x00, 0xFF, 0x00, 0xFF}
				}
			}
			b.SetNRGBA(x, y, c)
		}
	}
	return b
}

// geheimePixel zählt Pixel, die nach Magenta oder dem Grün des Musters aussehen (mit Spielraum für JPG).
func geheimePixel(b image.Image) int {
	n := 0
	r := b.Bounds()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			c := color.NRGBAModel.Convert(b.At(x, y)).(color.NRGBA)
			magenta := c.R > 200 && c.G < 70 && c.B > 200
			gruen := c.R < 70 && c.G > 200 && c.B < 70
			if magenta || gruen {
				n++
			}
		}
	}
	return n
}

var datenURI = regexp.MustCompile(`data:image/png;base64,([A-Za-z0-9+/=]+)`)

// ADR-0022 Punkt 5, Kernprinzip 2: Nach Schwärzen bzw. Verpixeln steht das Muster in keinem Ausgang — Arbeitsdatei
// (sichtbares Bild und snAO-Original), Vorschaubild der Bibliothek, SVG-Export (alle eingebetteten Kacheln), flaches
// PNG und JPG, PNG der Zwischenablage. Das Muster ist vorher in jedem Ausgang nachweisbar (Gegenprobe), damit der Test
// nicht nur deshalb grün ist, weil er nichts findet.
func TestSchwaerzenInKeinemAusgang(t *testing.T) {
	for name, sw := range map[string]document.Schwaerzung{
		"schwärzen": {Farbe: color.NRGBA{A: 255}},
		"verpixeln": {Block: 12, Mosaik: 24},
	} {
		t.Run(name, func(t *testing.T) {
			b := testBibliothek(t)
			d, err := document.Neu(geheimBild())
			if err != nil {
				t.Fatal(err)
			}
			// Ein Kasten über dem Muster und ein Stauch-Streifen außerhalb: Objekte und Kacheln im Export.
			mitKasten(t, d)
			if err := d.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 70, Bis: 75}}); err != nil {
				t.Fatal(err)
			}
			ausgaenge := func() map[string]image.Image {
				t.Helper()
				sichtbar, err := render.Bild(d)
				if err != nil {
					t.Fatal(err)
				}
				e, err := b.Ablegen(d, sichtbar, fileformat.Metadaten{})
				if err != nil {
					t.Fatal(err)
				}
				abs := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
				roh, err := os.ReadFile(abs)
				if err != nil {
					t.Fatal(err)
				}
				m := map[string]image.Image{}
				if m["Arbeitsdatei sichtbar"], err = fileformat.LeseBild(bytes.NewReader(roh)); err != nil {
					t.Fatal(err)
				}
				g, err := fileformat.Oeffne(abs)
				if err != nil {
					t.Fatal(err)
				}
				m["Arbeitsdatei snAO"] = g.Dokument.Original()
				v, err := os.ReadFile(b.VorschauPfad(e.Pfad))
				if err != nil {
					t.Fatal(err)
				}
				if m["Vorschau"], err = png.Decode(bytes.NewReader(v)); err != nil {
					t.Fatal(err)
				}
				var svg bytes.Buffer
				if err := fileformat.ExportiereSVG(&svg, d); err != nil {
					t.Fatal(err)
				}
				kacheln := datenURI.FindAllStringSubmatch(svg.String(), -1)
				if len(kacheln) == 0 {
					t.Fatal("SVG ohne eingebettete Kacheln")
				}
				for i, k := range kacheln {
					daten, err := base64.StdEncoding.DecodeString(k[1])
					if err != nil {
						t.Fatal(err)
					}
					if m["SVG-Kachel "+string(rune('0'+i))], err = png.Decode(bytes.NewReader(daten)); err != nil {
						t.Fatal(err)
					}
				}
				var flach bytes.Buffer
				if err := fileformat.ExportierePNG(&flach, sichtbar); err != nil {
					t.Fatal(err)
				}
				if m["PNG"], err = png.Decode(&flach); err != nil {
					t.Fatal(err)
				}
				var jpg bytes.Buffer
				if err := fileformat.ExportiereJPG(&jpg, sichtbar, 90); err != nil {
					t.Fatal(err)
				}
				if m["JPG"], err = jpeg.Decode(&jpg); err != nil {
					t.Fatal(err)
				}
				zw, err := render.PNG(d)
				if err != nil {
					t.Fatal(err)
				}
				if m["Zwischenablage"], err = png.Decode(bytes.NewReader(zw)); err != nil {
					t.Fatal(err)
				}
				return m
			}
			svg := 0 // die Kacheln unter dem Stauch-Streifen enthalten das Muster nicht; zusammen schon
			for a, bild := range ausgaenge() {
				n := geheimePixel(bild)
				if strings.HasPrefix(a, "SVG-Kachel") {
					svg += n
				} else if n == 0 {
					t.Errorf("Gegenprobe: Muster fehlt schon vor dem Schwärzen in %s", a)
				}
			}
			if svg == 0 {
				t.Error("Gegenprobe: Muster fehlt schon vor dem Schwärzen im SVG")
			}
			if err := d.Schwaerzen(image.Rect(28, 18, 92, 62), sw); err != nil {
				t.Fatal(err)
			}
			for a, bild := range ausgaenge() {
				if n := geheimePixel(bild); n != 0 {
					t.Errorf("%s: %d Pixel des Musters nach dem Schwärzen", a, n)
				}
			}
		})
	}
}

var schwarz = document.Schwaerzung{Farbe: color.NRGBA{A: 255}}

// geheimRahmen umfasst das Muster aus geheimBild.
var geheimRahmen = image.Rect(28, 18, 92, 62)

// bibliotheksAufnahme legt geheimBild als Aufnahme ab und öffnet sie wie der Editor: Die Speicherfunktion entsteht
// vor jeder Bearbeitung.
func bibliotheksAufnahme(t *testing.T) (b *library.Bibliothek, rel, abs string, g *fileformat.Gelesen, speichern ui.Speichern) {
	t.Helper()
	i18n.Setze(nil)
	b = testBibliothek(t)
	d, err := document.Neu(geheimBild())
	if err != nil {
		t.Fatal(err)
	}
	e, err := b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{})
	if err != nil {
		t.Fatal(err)
	}
	abs = filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
	if g, err = fileformat.Oeffne(abs); err != nil {
		t.Fatal(err)
	}
	return b, e.Pfad, abs, g, speichernIn(b, abs, g, standVon(t, abs))
}

// musterIn liest eine Arbeitsdatei und zählt die Musterpixel in sichtbarem Bild und snAO-Original.
func musterIn(t *testing.T, abs string) int {
	t.Helper()
	roh, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	sichtbar, err := fileformat.LeseBild(bytes.NewReader(roh))
	if err != nil {
		t.Fatal(err)
	}
	g, err := fileformat.Oeffne(abs)
	if err != nil {
		t.Fatal(err)
	}
	return geheimePixel(sichtbar) + geheimePixel(g.Dokument.Original())
}

// sperre macht den Ordner von abs nicht beschreibbar, sodass das Zurückschreiben scheitert.
func sperre(t *testing.T, abs string) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("als root wirken Ordnerrechte nicht")
	}
	monat := filepath.Dir(abs)
	if err := os.Chmod(monat, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(monat, 0o700) })
}

// F-168 (Lücke 1): Zurückschreiben an Ort und Stelle — Datei und Vorschau ohne Muster.
func TestSchwaerzenZurueckschreiben(t *testing.T) {
	b, rel, abs, g, speichern := bibliotheksAufnahme(t)
	if err := g.Dokument.Schwaerzen(geheimRahmen, schwarz); err != nil {
		t.Fatal(err)
	}
	if erg, err := speichern(g.Dokument); err != nil || erg != (ui.Gespeichert{}) {
		t.Fatalf("%+v, %v", erg, err)
	}
	if n := musterIn(t, abs); n != 0 {
		t.Errorf("Datei: %d Musterpixel", n)
	}
	v, err := os.ReadFile(b.VorschauPfad(rel))
	if err != nil {
		t.Fatal(err)
	}
	vb, err := png.Decode(bytes.NewReader(v))
	if err != nil {
		t.Fatal(err)
	}
	if n := geheimePixel(vb); n != 0 {
		t.Errorf("Vorschau: %d Musterpixel", n)
	}
}

// F-168 (Lücke 2): Ein Stauch-Streifen mitten im Rahmen wird bis in die Datei mitgeschwärzt.
func TestSchwaerzenUeberStreifenInDatei(t *testing.T) {
	_, _, abs, g, speichern := bibliotheksAufnahme(t)
	d := g.Dokument
	if err := d.SetzeStreifen([]document.Streifen{{Von: 50, Bis: 70}}); err != nil {
		t.Fatal(err)
	}
	s := canvas.NeueSitzung(d)
	s.SetzeFenster(image.Pt(148, 128)) // Ansicht 100×80, Zoom 1, Versatz 24
	s.SetzeWerkzeug(canvas.WerkzeugSchwaerzen)
	// Ansicht 28..72 = Original 28..50, verborgen 50..70, 70..92.
	s.Druecken(document.Pt(24+28, 24+18), canvas.TastePrimaer)
	s.Ziehen(document.Pt(24+50, 24+40))
	s.Loslassen(document.Pt(24+72, 24+62))
	if _, err := speichern(d); err != nil {
		t.Fatal(err)
	}
	if n := musterIn(t, abs); n != 0 {
		t.Errorf("Datei: %d Musterpixel (verborgener Streifen nicht geschwärzt?)", n)
	}
}

// F-168 (Lücke 3): Schwärzen → Speichern → Rückgängig → Speichern über die Sitzung: Das Rückgängig greift nicht mehr,
// die Datei bleibt geschwärzt.
func TestSchwaerzenRueckgaengigNachSpeichern(t *testing.T) {
	_, _, abs, g, speichern := bibliotheksAufnahme(t)
	s := canvas.NeueSitzung(g.Dokument)
	s.SetzeFenster(image.Pt(168, 128))
	s.SetzeWerkzeug(canvas.WerkzeugSchwaerzen)
	s.Druecken(document.Pt(24+28, 24+18), canvas.TastePrimaer)
	s.Ziehen(document.Pt(24+60, 24+40))
	s.Loslassen(document.Pt(24+92, 24+62))
	if _, err := speichern(s.D); err != nil {
		t.Fatal(err)
	}
	s.Gespeichert()
	if s.Rueckgaengig() {
		t.Error("Rückgängig nach dem Speichern holt die Schwärzung zurück")
	}
	if s.Geaendert() {
		if _, err := speichern(s.D); err != nil {
			t.Fatal(err)
		}
	}
	if n := musterIn(t, abs); n != 0 {
		t.Errorf("Datei: %d Musterpixel", n)
	}
}

// F-168 (Lücke 4): Konflikt mit Schwärzung — die andere Fassung bleibt und die Warnung sagt, dass sie den Inhalt
// zeigen kann.
func TestSchwaerzenKonflikt(t *testing.T) {
	_, _, abs, g, speichern := bibliotheksAufnahme(t)
	vorher, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	spaeter := time.Now().Add(time.Hour)
	if err := os.Chtimes(abs, spaeter, spaeter); err != nil { // „von anderer Seite geändert“
		t.Fatal(err)
	}
	if err := g.Dokument.Schwaerzen(geheimRahmen, schwarz); err != nil {
		t.Fatal(err)
	}
	erg, err := speichern(g.Dokument)
	if err != nil {
		t.Fatal(err)
	}
	if !erg.Warnung || !strings.Contains(erg.Hinweis, "changed elsewhere") || !strings.Contains(erg.Hinweis, "may still show") {
		t.Errorf("Ergebnis %+v", erg)
	}
	if roh, _ := os.ReadFile(abs); !bytes.Equal(roh, vorher) {
		t.Error("andere Fassung verändert oder entfernt")
	}
}

// ADR-0022 Punkt 4 (F-126): Scheitert das Zurückschreiben nach einer Schwärzung, wird die bisherige Datei samt
// Vorschau entfernt; die neue Aufnahme ist geschwärzt, die Warnung sagt es.
func TestF126AlteDateiNachSchwaerzenEntfernt(t *testing.T) {
	b, alt, abs, g, speichern := bibliotheksAufnahme(t)
	vorschau := b.VorschauPfad(alt)
	if err := g.Dokument.Schwaerzen(geheimRahmen, schwarz); err != nil {
		t.Fatal(err)
	}
	sperre(t, abs)
	erg, err := speichern(g.Dokument)
	if err != nil {
		t.Fatal(err)
	}
	if !erg.Warnung || !strings.Contains(erg.Hinweis, "so it was removed") || !strings.Contains(erg.Hinweis, erg.Datei) {
		t.Errorf("Ergebnis %+v", erg)
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
}

// Lässt sich die alte Datei nicht entfernen, nennt die Warnung sie als ungeschwärzt; bleibt nur die Vorschau, nennt
// sie die Vorschau (F-164).
func TestF126EntfernenScheitertWarnt(t *testing.T) {
	for name, fall := range map[string]struct {
		weg  bool
		err  error
		soll string
	}{
		"Datei bleibt":    {false, os.ErrPermission, "Delete it yourself"},
		"Vorschau bleibt": {true, library.ErrVorschauAlt, "its preview could not be removed"},
	} {
		t.Run(name, func(t *testing.T) {
			_, alt, abs, g, speichern := bibliotheksAufnahme(t)
			vorher := entfernen
			entfernen = func(*library.Bibliothek, string, library.Stand) (bool, error) { return fall.weg, fall.err }
			t.Cleanup(func() { entfernen = vorher })
			if err := g.Dokument.Schwaerzen(geheimRahmen, schwarz); err != nil {
				t.Fatal(err)
			}
			sperre(t, abs)
			erg, err := speichern(g.Dokument)
			if err != nil {
				t.Fatal(err)
			}
			if !erg.Warnung || !strings.Contains(erg.Hinweis, fall.soll) || !strings.Contains(erg.Hinweis, "unredacted") {
				t.Errorf("Ergebnis %+v", erg)
			}
			if !fall.weg && !strings.Contains(erg.Hinweis, alt) {
				t.Errorf("Hinweis nennt die alte Datei nicht: %q", erg.Hinweis)
			}
		})
	}
}

// F-167: Fehlt die alte Datei schon (verschoben, Papierkorb), behauptet die Meldung kein Entfernen.
func TestF126DateiSchonWeg(t *testing.T) {
	_, alt, abs, g, speichern := bibliotheksAufnahme(t)
	anderswo := filepath.Join(t.TempDir(), "verschoben.png")
	if err := os.Rename(abs, anderswo); err != nil {
		t.Fatal(err)
	}
	if err := g.Dokument.Schwaerzen(geheimRahmen, schwarz); err != nil {
		t.Fatal(err)
	}
	erg, err := speichern(g.Dokument)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(erg.Hinweis, "removed") || !strings.Contains(erg.Hinweis, "no longer there") ||
		!strings.Contains(erg.Hinweis, alt) {
		t.Errorf("Ergebnis %+v", erg)
	}
}

// F-169: Wurde seit dem letzten gelungenen Schreiben nicht geschwärzt, ist die alte Datei schon geschwärzt; scheitert
// dann ein Zurückschreiben, bleibt sie stehen.
func TestF126AlteDateiSchonGeschwaerzt(t *testing.T) {
	_, _, abs, g, speichern := bibliotheksAufnahme(t)
	if err := g.Dokument.Schwaerzen(geheimRahmen, schwarz); err != nil {
		t.Fatal(err)
	}
	if _, err := speichern(g.Dokument); err != nil {
		t.Fatal(err)
	}
	mitKasten(t, g.Dokument)
	sperre(t, abs)
	erg, err := speichern(g.Dokument)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(erg.Hinweis, "unredacted") || !strings.Contains(erg.Hinweis, "previous file could not be updated") {
		t.Errorf("Ergebnis %+v", erg)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Errorf("schon geschwärzte alte Datei entfernt: %v", err)
	}
}

// F-187: Schwärzen A, Speichern, Schwärzen B, Rückgängig B — die Datei entspricht dem Stand (A ist darin geschwärzt).
// Scheitert dann das Zurückschreiben, bleibt sie stehen, und die Meldung nennt keinen ungeschwärzten Inhalt.
func TestF126SchwaerzenRueckgaengigVorAusweichen(t *testing.T) {
	_, _, abs, g, speichern := bibliotheksAufnahme(t)
	d := g.Dokument
	if err := d.Schwaerzen(geheimRahmen, schwarz); err != nil {
		t.Fatal(err)
	}
	if _, err := speichern(d); err != nil {
		t.Fatal(err)
	}
	if err := d.Schwaerzen(image.Rect(0, 0, 10, 10), schwarz); err != nil {
		t.Fatal(err)
	}
	if err := d.Rueckgaengig(); err != nil {
		t.Fatal(err)
	}
	mitKasten(t, d)
	sperre(t, abs)
	erg, err := speichern(d)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(erg.Hinweis, "unredacted") || !strings.Contains(erg.Hinweis, "previous file could not be updated") {
		t.Errorf("Ergebnis %+v", erg)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Errorf("Datei mit dem geschwärzten Stand entfernt: %v", err)
	}
}

// F-186: Fehlte die alte Datei schon und scheitert nur das Entfernen der Vorschau bzw. nur der Index, nennt die
// Meldung die Vorschau bzw. sagt „nicht mehr da“ — nie „Delete it yourself“ für eine Datei, die es nicht gibt.
func TestF126DateiWegNebenschrittScheitert(t *testing.T) {
	for name, fall := range map[string]struct {
		err        error
		soll       []string
		vorschauDa bool
	}{
		"Vorschau bleibt": {fmt.Errorf("%w: %w: x", library.ErrVorschau, library.ErrVorschauAlt),
			[]string{"no longer there", "preview could not be removed", "unredacted"}, true},
		"nur Index": {fmt.Errorf("%w: x", library.ErrIndex), []string{"no longer there", "unredacted"}, false},
		"Index und Vorschau": {errors.Join(fmt.Errorf("%w: %w: x", library.ErrVorschau, library.ErrVorschauAlt),
			fmt.Errorf("%w: x", library.ErrIndex)), []string{"no longer there", "preview could not be removed"}, true},
	} {
		t.Run(name, func(t *testing.T) {
			_, alt, abs, g, speichern := bibliotheksAufnahme(t)
			vorher := entfernen
			entfernen = func(*library.Bibliothek, string, library.Stand) (bool, error) { return false, fall.err }
			t.Cleanup(func() { entfernen = vorher })
			if err := g.Dokument.Schwaerzen(geheimRahmen, schwarz); err != nil {
				t.Fatal(err)
			}
			sperre(t, abs)
			erg, err := speichern(g.Dokument)
			if err != nil {
				t.Fatal(err)
			}
			if !erg.Warnung || strings.Contains(erg.Hinweis, "Delete it yourself") || strings.Contains(erg.Hinweis, "was removed") ||
				!strings.Contains(erg.Hinweis, alt) {
				t.Errorf("Ergebnis %+v", erg)
			}
			for _, s := range fall.soll {
				if !strings.Contains(erg.Hinweis, s) {
					t.Errorf("Hinweis ohne %q: %q", s, erg.Hinweis)
				}
			}
			if !fall.vorschauDa && strings.Contains(erg.Hinweis, "preview") {
				t.Errorf("Hinweis nennt eine Vorschau: %q", erg.Hinweis)
			}
		})
	}
}

// Eine Datei von außen bleibt unberührt; nach einer Schwärzung sagt die Warnung, dass sie den Inhalt noch zeigt.
func TestSchwaerzenDateiVonAussenWarnt(t *testing.T) {
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
	if err := g.Dokument.Schwaerzen(geheimRahmen, schwarz); err != nil {
		t.Fatal(err)
	}
	erg, err := speichern(g.Dokument)
	if err != nil {
		t.Fatal(err)
	}
	if !erg.Warnung || !strings.Contains(erg.Hinweis, "still shows the unredacted content") {
		t.Errorf("Ergebnis %+v", erg)
	}
	if roh, _ := os.ReadFile(aussen); !bytes.Equal(roh, buf.Bytes()) {
		t.Error("Datei von außen verändert")
	}
}
