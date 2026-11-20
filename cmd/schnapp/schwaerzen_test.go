// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/document"
	"github.com/Andiministrator/schnapp/fileformat"
	"github.com/Andiministrator/schnapp/i18n"
	"github.com/Andiministrator/schnapp/library"
	"github.com/Andiministrator/schnapp/render"
	"github.com/Andiministrator/schnapp/ui"
)

// geheimBild ist ein graues Bild 120×80 mit einem „geheimen“ Schachbrett aus Magenta und Grün bei (30..90, 20..60):
// Farben, die sonst nirgends vorkommen und die Schwärzen wie Verpixeln (Mittel ~ Grau, Rauschen ±24) nie erzeugen.
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
		"verpixeln": {Block: 12, Rauschen: 24},
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

// ADR-0022 Punkt 4 (F-126): Scheitert das Zurückschreiben nach einer Schwärzung, wird die bisherige Datei samt
// Vorschau entfernt; die neue Aufnahme ist geschwärzt, die Warnung sagt es.
func TestF126AlteDateiNachSchwaerzenEntfernt(t *testing.T) {
	alt, abs, b, speichern, g := ausweichenVorbereiten(t)
	vorschau := b.VorschauPfad(alt)
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
		t.Errorf("Index %+v (Soll: nur die neue Aufnahme)", idx.Eintraege)
	}
	neu, err := fileformat.Oeffne(filepath.Join(b.Orte.Ablage, filepath.FromSlash(idx.Eintraege[0].Pfad)))
	if err != nil {
		t.Fatal(err)
	}
	if n := geheimePixel(neu.Dokument.Original()); n != 0 {
		t.Errorf("neue Aufnahme: %d Pixel des Musters", n)
	}
}

// Lässt sich die alte Datei nicht entfernen, nennt die Warnung sie als ungeschwärzt.
func TestF126EntfernenScheitertWarnt(t *testing.T) {
	alt, abs, _, speichern, g := ausweichenVorbereiten(t)
	vorher := entfernen
	entfernen = func(*library.Bibliothek, string, library.Stand) error { return os.ErrPermission }
	t.Cleanup(func() { entfernen = vorher })
	erg, err := speichern(g.Dokument)
	if err != nil {
		t.Fatal(err)
	}
	if !erg.Warnung || !strings.Contains(erg.Hinweis, "still shows the unredacted content") ||
		!strings.Contains(erg.Hinweis, alt) {
		t.Errorf("Ergebnis %+v", erg)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Errorf("alte Datei: %v", err)
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
	if err := g.Dokument.Schwaerzen(image.Rect(28, 18, 92, 62), document.Schwaerzung{Farbe: color.NRGBA{A: 255}}); err != nil {
		t.Fatal(err)
	}
	erg, err := speichernIn(b, aussen, g, standVon(t, aussen))(g.Dokument)
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

// ausweichenVorbereiten legt eine Aufnahme mit Muster ab, öffnet sie, schwärzt und sperrt den Monatsordner, sodass das
// Zurückschreiben scheitert (wie TestSpeichernSchreibfehlerWeichtSichtbarAus). Liefert den relativen Pfad der alten
// Datei, ihren absoluten Pfad, die Bibliothek, die Speicherfunktion und das geöffnete Dokument.
func ausweichenVorbereiten(t *testing.T) (string, string, *library.Bibliothek, ui.Speichern, *fileformat.Gelesen) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("als root wirken Ordnerrechte nicht")
	}
	i18n.Setze(nil)
	b := testBibliothek(t)
	d, err := document.Neu(geheimBild())
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
	if err := g.Dokument.Schwaerzen(image.Rect(28, 18, 92, 62), document.Schwaerzung{Farbe: color.NRGBA{A: 255}}); err != nil {
		t.Fatal(err)
	}
	monat := filepath.Dir(abs)
	if err := os.Chmod(monat, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(monat, 0o700) })
	return e.Pfad, abs, b, speichernIn(b, abs, g, standVon(t, abs)), g
}
