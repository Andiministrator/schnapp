// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"compress/zlib"
	"errors"
	"flag"
	"image"
	"image/color"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Konformität zu docs/format.md (E-15): Die Beispieldateien unter testdata/format/ werden vom Schreiber erzeugt
// (`go test ./internal/fileformat -run TestFormatBeispiele -update`), der Leser liest sie mit dem erwarteten Ergebnis,
// und ein Test hält die Konstanten der Spezifikation mit denen des Codes zusammen. Die Spezifikation und FormatVersion
// ändern sich immer im selben Commit.

var aktualisiereBeispiele = flag.Bool("update", false, "Beispieldateien unter testdata/format neu schreiben")

const beispielOrdner = "../../testdata/format"

// beispielBild ist ein kleines, deterministisches Bild ohne Bezug zu echten Aufnahmen.
func beispielBild(b, h int) *image.NRGBA {
	m := image.NewNRGBA(image.Rect(0, 0, b, h))
	for y := 0; y < h; y++ {
		for x := 0; x < b; x++ {
			m.SetNRGBA(x, y, color.NRGBA{uint8(40 + x*3), uint8(60 + y*4), uint8(200 - x - y), 255})
		}
	}
	return m
}

type beispiel struct {
	name string
	bau  func(t *testing.T) *document.Dokument
	meta Metadaten
}

func beispielMeta() Metadaten {
	return Metadaten{Autor: "Example Author", Erstellt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		Quelle: "Example App", Schlagworte: []string{"example", "format spec"}, Bearbeitet: true}
}

func beispielDok(t *testing.T, b, h int) *document.Dokument {
	t.Helper()
	d, err := document.Neu(beispielBild(b, h))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func fuege(t *testing.T, d *document.Dokument, os ...document.Objekt) {
	t.Helper()
	for _, o := range os {
		if _, err := d.Hinzufuegen(o); err != nil {
			t.Fatal(err)
		}
	}
}

var (
	beispielRot  = document.Strich{Farbe: color.NRGBA{220, 30, 30, 255}, Breite: 2.5}
	beispielBlau = color.NRGBA{30, 90, 200, 255}
)

func beispiele() []beispiel {
	return []beispiel{
		{"objects", func(t *testing.T) *document.Dokument {
			d := beispielDok(t, 64, 48)
			fuege(t, d,
				document.GeraderPfeil(document.Pt(4, 6), document.Pt(40, 20), beispielRot).MitGriff(document.Pt(16, 6)),
				document.Kasten{Bereich: document.R(document.Pt(8, 24), document.Pt(30.5, 40)), Strich: beispielRot,
					Fuellung: color.NRGBA{255, 255, 0, 64}, Eckradius: 3},
				document.Text{Ort: document.Pt(34, 26), Inhalt: "Line <1> & \"2\"\nTab\there", Schriftgrad: 8,
					Farbe: color.NRGBA{0, 0, 0, 255}, Hintergrund: color.NRGBA{255, 255, 255, 200}},
				document.Nummer{Mitte: document.Pt(52, 10), Wert: 3, Radius: 7, Farbe: beispielBlau})
			return d
		}, beispielMeta()},
		{"view", func(t *testing.T) *document.Dokument {
			d := beispielDok(t, 64, 48)
			fuege(t, d, document.Kasten{Bereich: document.R(document.Pt(10, 8), document.Pt(50, 40)), Strich: beispielRot})
			if err := d.SetzeZuschnitt(image.Rect(4, 3, 60, 45)); err != nil {
				t.Fatal(err)
			}
			if err := d.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 20, Bis: 25}, {Zeilen: false, Von: 30, Bis: 33}}); err != nil {
				t.Fatal(err)
			}
			return d
		}, Metadaten{Erstellt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}},
		{"redaction", func(t *testing.T) *document.Dokument {
			d := beispielDok(t, 64, 48)
			if err := d.Schwaerzen(image.Rect(8, 8, 24, 20), document.Schwaerzung{Farbe: color.NRGBA{0, 0, 0, 255}}); err != nil {
				t.Fatal(err)
			}
			if err := d.Schwaerzen(image.Rect(32, 24, 56, 40), document.Schwaerzung{Block: 8}); err != nil {
				t.Fatal(err)
			}
			return d
		}, Metadaten{Bearbeitet: true}},
		{"minimal", func(t *testing.T) *document.Dokument { return beispielDok(t, 32, 24) }, Metadaten{}},
	}
}

func beispielDatei(t *testing.T, b beispiel) (d *document.Dokument, png []byte) {
	t.Helper()
	d = b.bau(t)
	var buf bytes.Buffer
	// Sichtbar ist hier die Ansicht ohne Beschriftungen: Das Rendern der Schrift hängt von Fließkomma-Details ab und
	// gehört nicht zum Format (docs/format.md, „visible image“).
	if err := SchreibeArbeitsdatei(&buf, d, d.Ansichtsbild(), b.meta); err != nil {
		t.Fatal(err)
	}
	return d, buf.Bytes()
}

func entpackeSVG(t *testing.T, png []byte) []byte {
	t.Helper()
	zr, err := zlib.NewReader(bytes.NewReader(blockDaten(t, png, blockSVG)))
	if err != nil {
		t.Fatal(err)
	}
	s, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// TestFormatBeispiele: Der Schreiber erzeugt die Beispieldateien byte-gleich (und das entpackte SVG daneben als
// lesbare Fassung).
func TestFormatBeispiele(t *testing.T) {
	for _, b := range beispiele() {
		_, png := beispielDatei(t, b)
		svg := entpackeSVG(t, png)
		if !bytes.Equal(svg, svgArbeitsdatei(b.bau(t), b.meta)) {
			t.Errorf("%s: entpacktes SVG weicht von svgArbeitsdatei ab", b.name)
		}
		for ext, soll := range map[string][]byte{".png": png, ".snap.svg": svg} {
			pfad := filepath.Join(beispielOrdner, b.name+ext)
			if *aktualisiereBeispiele {
				if err := os.MkdirAll(beispielOrdner, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(pfad, soll, 0o644); err != nil {
					t.Fatal(err)
				}
				continue
			}
			ist, err := os.ReadFile(pfad)
			if err != nil {
				t.Fatalf("%v (mit -update erzeugen)", err)
			}
			if !bytes.Equal(ist, soll) {
				t.Errorf("%s weicht vom Schreiber ab (mit -update neu erzeugen, wenn die Änderung gewollt ist)", pfad)
			}
		}
	}
}

// TestFormatBeispieleLesen: Der Leser liest jede Beispieldatei mit dem erwarteten Ergebnis.
func TestFormatBeispieleLesen(t *testing.T) {
	for _, b := range beispiele() {
		daten, err := os.ReadFile(filepath.Join(beispielOrdner, b.name+".png"))
		if err != nil {
			t.Fatalf("%v (mit -update erzeugen)", err)
		}
		g, err := LeseArbeitsdatei(bytes.NewReader(daten))
		if err != nil {
			t.Fatalf("%s: %v", b.name, err)
		}
		soll := b.bau(t)
		if g.Fremd {
			t.Errorf("%s: als fremd erkannt", b.name)
		}
		if !reflect.DeepEqual(g.Dokument.Objekte(), soll.Objekte()) {
			t.Errorf("%s: Objekte\n got %#v\nwant %#v", b.name, g.Dokument.Objekte(), soll.Objekte())
		}
		if g.Dokument.Zuschnitt() != soll.Zuschnitt() || !reflect.DeepEqual(g.Dokument.Streifen(), soll.Streifen()) {
			t.Errorf("%s: Ansicht %v %v", b.name, g.Dokument.Zuschnitt(), g.Dokument.Streifen())
		}
		if !bytes.Equal(g.Dokument.Original().Pix, soll.Original().Pix) {
			t.Errorf("%s: Original weicht ab", b.name)
		}
		if !reflect.DeepEqual(g.Meta, b.meta) {
			t.Errorf("%s: Metadaten\n got %#v\nwant %#v", b.name, g.Meta, b.meta)
		}
		// Das sichtbare Bild ist ein gewöhnliches PNG in der Größe der Ansicht.
		bild, err := LeseBild(bytes.NewReader(daten))
		if err != nil {
			t.Fatalf("%s: %v", b.name, err)
		}
		if bild.Bounds().Size() != soll.Groesse() {
			t.Errorf("%s: sichtbares Bild %v, Ansicht %v", b.name, bild.Bounds().Size(), soll.Groesse())
		}
		// Die Datei trägt außer den zwei privaten Blöcken nur IHDR, IDAT und IEND (keine Spuren, Kernprinzip 3).
		bl, err := leseBloecke(bytes.NewReader(daten))
		if err != nil {
			t.Fatal(err)
		}
		var namen []string
		for _, k := range bl {
			if len(namen) == 0 || namen[len(namen)-1] != k.typ {
				namen = append(namen, k.typ)
			}
		}
		if got := strings.Join(namen, " "); got != "IHDR IDAT snAP snAO IEND" {
			t.Errorf("%s: Blockfolge %q", b.name, got)
		}
	}
}

// TestFormatSchwaerzung: Im Original der Beispieldatei stehen an den geschwärzten Stellen keine Ausgangspixel mehr
// (Kernprinzip 2), und die Datei enthält kein zweites Bild, aus dem sie sich holen ließen.
func TestFormatSchwaerzung(t *testing.T) {
	daten, err := os.ReadFile(filepath.Join(beispielOrdner, "redaction.png"))
	if err != nil {
		t.Fatal(err)
	}
	g, err := LeseArbeitsdatei(bytes.NewReader(daten))
	if err != nil {
		t.Fatal(err)
	}
	orig, ausgang := g.Dokument.Original(), beispielBild(64, 48)
	for y := 8; y < 20; y++ {
		for x := 8; x < 24; x++ {
			if orig.NRGBAAt(x, y) != (color.NRGBA{0, 0, 0, 255}) {
				t.Fatalf("(%d,%d) nicht geschwärzt: %v", x, y, orig.NRGBAAt(x, y))
			}
		}
	}
	gleich := 0
	for y := 24; y < 40; y++ {
		for x := 32; x < 56; x++ {
			if orig.NRGBAAt(x, y) == ausgang.NRGBAAt(x, y) {
				gleich++
			}
			if x%8 == 0 && y%8 == 0 && orig.NRGBAAt(x, y) != orig.NRGBAAt(x+1, y+1) {
				t.Fatalf("Block bei (%d,%d) nicht einfarbig", x, y)
			}
		}
	}
	if gleich > 8 {
		t.Errorf("%d Pixel der Verpixelung gleichen dem Ausgangsbild", gleich)
	}
	// Das sichtbare Bild stammt aus dem Original: gleiche Schwärzung.
	bild, err := LeseBild(bytes.NewReader(daten))
	if err != nil {
		t.Fatal(err)
	}
	if c := color.NRGBAModel.Convert(bild.At(10, 10)); c != (color.NRGBA{0, 0, 0, 255}) {
		t.Errorf("sichtbares Bild bei (10,10): %v", c)
	}
}

// pngMitSVG ersetzt den Block snAP einer Beispieldatei durch das SVG svg.
func pngMitSVG(t *testing.T, png []byte, svg string) []byte {
	t.Helper()
	bl, err := leseBloecke(bytes.NewReader(png))
	if err != nil {
		t.Fatal(err)
	}
	var z bytes.Buffer
	zw := zlib.NewWriter(&z)
	zw.Write([]byte(svg))
	zw.Close()
	for i := range bl {
		if bl[i].typ == blockSVG {
			bl[i].daten = z.Bytes()
		}
	}
	var buf bytes.Buffer
	if err := schreibeBloecke(&buf, bl); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestFormatLeserRegeln: die Regeln für Leser aus docs/format.md, an der Beispieldatei „objects“ durchgespielt.
func TestFormatLeserRegeln(t *testing.T) {
	png, err := os.ReadFile(filepath.Join(beispielOrdner, "objects.png"))
	if err != nil {
		t.Fatal(err)
	}
	svg := string(entpackeSVG(t, png))
	lies := func(s string) error {
		_, err := LeseArbeitsdatei(bytes.NewReader(pngMitSVG(t, png, s)))
		return err
	}
	if err := lies(svg); err != nil {
		t.Fatal(err)
	}
	ersetze := func(alt, neu string) string {
		if !strings.Contains(svg, alt) {
			t.Fatalf("Vorlage ohne %q", alt)
		}
		return strings.Replace(svg, alt, neu, 1)
	}
	version := `s:format="` + strconv.Itoa(FormatVersion) + `"`
	// Unbekannte Attribute und Elemente sind ohne Wirkung.
	for name, s := range map[string]string{
		"Attribut am Objekt":     ersetze(`s:typ="kasten"`, `s:typ="kasten" s:zukunft="1"`),
		"Attribut an der Wurzel": ersetze(version, version+` s:zukunft="1"`),
		"Element in der Ansicht": ersetze(`</s:ansicht>`, `<s:zukunft a="b"/></s:ansicht>`),
		"Element neben Objekten": ersetze(`<g s:ebene="objekte">`, `<desc>x</desc><g s:ebene="objekte">`),
		"fremdes Attribut":       ersetze(`<image `, `<image data-x="y" `),
		"Metadaten ohne Zeit":    ersetze(` s:erstellt="2026-01-02T03:04:05Z"`, ``),
	} {
		if err := lies(s); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	// Neuere Version: abgelehnt, die Datei bleibt als Bild lesbar.
	err = lies(ersetze(version, `s:format="`+strconv.Itoa(FormatVersion+1)+`"`))
	if !errors.Is(err, ErrFormatNeuer) {
		t.Errorf("neuere Version: %v", err)
	}
	if _, err := LeseBild(bytes.NewReader(png)); err != nil {
		t.Error(err)
	}
	// Beschädigt: fehlende Version, fehlende Ansicht, unbekannter Objekttyp, fehlendes Pflichtattribut, ungültiger
	// Wert, Entität, Größe passt nicht zum Original.
	for name, s := range map[string]string{
		"Version fehlt":          ersetze(version, ``),
		"Ansicht fehlt":          strings.Replace(ersetze(`<s:ansicht`, `<s:ansichtx`), `</s:ansicht>`, `</s:ansichtx>`, 1),
		"unbekannter Objekttyp":  ersetze(`s:typ="kasten"`, `s:typ="ellipse"`),
		"Pflichtattribut fehlt":  ersetze(` s:radius="7"`, ``),
		"Farbe ungültig":         ersetze(`#1e5ac8ff`, `#1e5ac8`),
		"Zahl ungültig":          ersetze(`s:radius="7"`, `s:radius="NaN"`),
		"Entität":                strings.Replace(ersetze(`<s:meta`, `<s:meta s:autor="&x;"`), ` s:autor="Example Author"`, ``, 1),
		"Größe passt nicht":      ersetze(`width="64" height="48" viewBox`, `width="65" height="48" viewBox`),
		"Wert außerhalb Grenzen": ersetze(`s:radius="7"`, `s:radius="1e9"`),
	} {
		err := lies(s)
		if err == nil || !errors.Is(err, ErrBeschaedigt) && !errors.Is(err, ErrZuGross) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// --- Spezifikation und Code zusammenhalten ---

func leseSpezifikation(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("../../docs/format.md")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestFormatSpezifikationKonstanten: Die Tabelle „Constants“ in docs/format.md nennt Formatversion, Namensraum,
// Blocknamen und Grenzen; jeder Wert muss dem Code entsprechen. Eine Änderung an einer Seite ohne die andere
// fällt hier auf.
func TestFormatSpezifikationKonstanten(t *testing.T) {
	spez := leseSpezifikation(t)
	zeile := regexp.MustCompile("(?m)^\\| `([a-z_]+)` \\| ([^|]+?) \\|")
	werte := map[string]string{}
	for _, m := range zeile.FindAllStringSubmatch(spez, -1) {
		werte[m[1]] = strings.Trim(m[2], "` ")
	}
	zahlen := map[string]float64{
		"format_version":   FormatVersion,
		"max_side":         document.MaxSeite,
		"max_pixels":       document.MaxPixel,
		"max_file_bytes":   maxDatei,
		"max_chunk_bytes":  maxBlock,
		"max_svg_bytes":    maxSVG,
		"max_svg_elements": maxElemente,
		"max_svg_depth":    maxTiefe,
		"max_objects":      document.MaxObjekte,
		"max_text_chars":   document.MaxTextZeichen,
		"max_stripes":      document.MaxStreifen,
		"max_tiles":        document.MaxKacheln,
		"max_coordinate":   document.MaxKoordinate,
		"max_size":         document.MaxGroesse,
		"max_font_size":    document.MaxSchrift,
		"max_number":       document.MaxNummer,
	}
	for k, soll := range zahlen {
		s, ok := werte[k]
		if !ok {
			t.Errorf("docs/format.md: Zeile `%s` fehlt in der Tabelle Constants", k)
			continue
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil || math.Abs(f-soll) > 0 {
			t.Errorf("docs/format.md: %s = %q, Code %v", k, s, soll)
		}
	}
	texte := map[string]string{
		"namespace":    ns,
		"chunk_svg":    blockSVG,
		"chunk_orig":   blockOriginal,
		"original_ref": "schnapp:original",
	}
	for k, soll := range texte {
		if s := werte[k]; s != soll {
			t.Errorf("docs/format.md: %s = %q, Code %q", k, s, soll)
		}
	}
	// Die Kopfzeile nennt die Version noch einmal im Fließtext.
	if !strings.Contains(spez, "**Format version: "+strconv.Itoa(FormatVersion)+"**") {
		t.Errorf("docs/format.md: Kopfzeile „**Format version: %d**“ fehlt", FormatVersion)
	}
}

// TestFormatSpezifikationAttribute: Jedes Attribut und Element mit Präfix `s:`, das der Schreiber erzeugt, ist in
// docs/format.md beschrieben, und jedes dort beschriebene wird geschrieben oder gelesen.
func TestFormatSpezifikationAttribute(t *testing.T) {
	spez := leseSpezifikation(t)
	re := regexp.MustCompile(`(?:^|[^a-z])s:([a-z]+)`)
	gesehen := map[string]bool{}
	for _, b := range beispiele() {
		d := b.bau(t)
		for _, m := range re.FindAllStringSubmatch(string(svgArbeitsdatei(d, b.meta)), -1) {
			gesehen[m[1]] = true
		}
	}
	for _, w := range []string{"format", "meta", "autor", "erstellt", "quelle", "bearbeitet", "schlagwort", "ansicht", "zuschnitt",
		"streifen", "zeilen", "von", "bis", "ebene", "typ", "a", "b", "k", "strich", "bereich", "fuellung", "eckradius",
		"ort", "inhalt", "grad", "farbe", "hintergrund", "umbruch", "mitte", "wert", "radius"} {
		if !gesehen[w] {
			t.Errorf("Beispiele enthalten s:%s nicht (Beispielabdeckung)", w)
		}
	}
	for n := range gesehen {
		if !strings.Contains(spez, "`s:"+n+"`") {
			t.Errorf("docs/format.md beschreibt s:%s nicht", n)
		}
	}
	for _, n := range []string{"pfeil", "kasten", "text", "nummer"} {
		if !strings.Contains(spez, `"`+n+`"`) {
			t.Errorf("docs/format.md nennt den Objekttyp %q nicht", n)
		}
	}
}
