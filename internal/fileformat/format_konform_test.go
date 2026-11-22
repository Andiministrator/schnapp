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
//
// Die Byte-Gleichheit der Beispiel-PNG hängt an `compress/zlib` und `image/png` der Go-Version: ein anderer Deflate-
// Encoder liefert andere Bytes bei gleichem Inhalt. Die CI nimmt Go nach go.mod; bei einem Go-Update `-update` fahren
// und den Diff der `*.snap.svg` prüfen (er muss leer sein, nur die `*.png` dürfen sich ändern), erst dann
// einchecken. Das Format selbst ändert sich dadurch nicht.

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
	return pngMitSVGStufe(t, png, svg, zlib.DefaultCompression)
}

// pngMitSVGStufe wie pngMitSVG, mit der zlib-Stufe level für den Block.
func pngMitSVGStufe(t *testing.T, png []byte, svg string, level int) []byte {
	t.Helper()
	bl, err := leseBloecke(bytes.NewReader(png))
	if err != nil {
		t.Fatal(err)
	}
	var z bytes.Buffer
	zw, err := zlib.NewWriterLevel(&z, level)
	if err != nil {
		t.Fatal(err)
	}
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

// beispielSVG liest das PNG und das SVG eines Beispiels.
func beispielSVG(t *testing.T, name string) (png []byte, svg string) {
	t.Helper()
	png, err := os.ReadFile(filepath.Join(beispielOrdner, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	return png, string(entpackeSVG(t, png))
}

// urteil ist, was ein Leser über eine Datei sagt: ok, neuer, zu groß oder beschädigt.
func urteil(err error) string {
	switch {
	case err == nil:
		return "ok"
	case errors.Is(err, ErrFormatNeuer):
		return "neuer"
	case errors.Is(err, ErrZuGross):
		return "zu groß"
	case errors.Is(err, ErrBeschaedigt):
		return "beschädigt"
	}
	return "anders: " + err.Error()
}

// TestFormatLeserRegeln: die Regeln für Leser aus docs/format.md (Abschnitte 5.2, 7, 8, 11), an den Beispielen
// durchgespielt. Jede Ablehnung prüft einen Teil des Fehlertexts; das volle Lesen und LeseMetadaten fällen dasselbe
// Urteil (Regel 4 und 7).
func TestFormatLeserRegeln(t *testing.T) {
	png, svg := beispielSVG(t, "objects")
	viewPNG, viewSVG := beispielSVG(t, "view")
	lies := func(png []byte, s string) error {
		_, err := LeseArbeitsdatei(bytes.NewReader(pngMitSVG(t, png, s)))
		return err
	}
	// gleich: LeseMetadaten urteilt wie LeseArbeitsdatei.
	gleich := func(name string, png []byte, s string) {
		t.Helper()
		_, kopf := LeseMetadaten(bytes.NewReader(pngMitSVG(t, png, s)))
		if voll := lies(png, s); urteil(voll) != urteil(kopf) {
			t.Errorf("%s: LeseArbeitsdatei %q, LeseMetadaten %q", name, urteil(voll), urteil(kopf))
		}
	}
	if err := lies(png, svg); err != nil {
		t.Fatal(err)
	}
	ersetzeIn := func(vorlage, alt, neu string) string {
		if !strings.Contains(vorlage, alt) {
			t.Fatalf("Vorlage ohne %q", alt)
		}
		return strings.Replace(vorlage, alt, neu, 1)
	}
	ersetze := func(alt, neu string) string { return ersetzeIn(svg, alt, neu) }
	ersetzeV := func(alt, neu string) string { return ersetzeIn(viewSVG, alt, neu) }
	version := `s:format="` + strconv.Itoa(FormatVersion) + `"`
	const streifenV = `<s:streifen s:zeilen="false" s:von="30" s:bis="33"/><s:streifen s:zeilen="true" s:von="20" s:bis="25"/>`
	const metaV = `<s:meta s:erstellt="2026-01-02T03:04:05Z" s:bearbeitet="false">`

	type fall struct {
		png []byte
		svg string
	}
	// Angenommen: unbekannter Inhalt ist ohne Wirkung (Regel 3), dazu die Schreibweisen, die 5.2 und 7 erlauben.
	angenommen := map[string]fall{
		"Attribut am Objekt":          {png, ersetze(`s:typ="kasten"`, `s:typ="kasten" s:zukunft="1"`)},
		"Attribut an der Wurzel":      {png, ersetze(version, version+` s:zukunft="1"`)},
		"Element in der Ansicht":      {png, ersetze(`</s:ansicht>`, `<s:zukunft a="b"/></s:ansicht>`)},
		"Element neben Objekten":      {png, ersetze(`<g s:ebene="objekte">`, `<desc>x</desc><g s:ebene="objekte">`)},
		"fremdes Attribut":            {png, ersetze(`<image `, `<image data-x="y" `)},
		"Metadaten ohne Zeit":         {png, ersetze(` s:erstellt="2026-01-02T03:04:05Z"`, ``)},
		"Farbe in Großbuchstaben":     {png, ersetze(`#1e5ac8ff`, `#1E5AC8FF`)},
		"Zahl mit Exponent":           {png, ersetze(`s:radius="7"`, `s:radius="0.7e1"`)},
		"Exponent mit Plus":           {png, ersetze(`s:eckradius="3"`, `s:eckradius="3e+00"`)},
		"Exponent mit Minus":          {png, ersetze(`s:eckradius="3"`, `s:eckradius="30e-1"`)},
		"Zuschnitt, Ecken vertauscht": {viewPNG, ersetzeV(`s:zuschnitt="4 3 60 45"`, `s:zuschnitt="60 45 4 3"`)},
		"Streifen unsortiert": {viewPNG, ersetzeV(streifenV,
			`<s:streifen s:zeilen="true" s:von="30" s:bis="33"/><s:streifen s:zeilen="false" s:von="30" s:bis="33"/><s:streifen s:zeilen="true" s:von="20" s:bis="25"/>`)},
		"Streifen aneinander": {viewPNG, ersetzeV(`s:zeilen="false" s:von="30" s:bis="33"`, `s:zeilen="true" s:von="25" s:bis="33"`)},
	}
	for name, f := range angenommen {
		if err := lies(f.png, f.svg); err != nil {
			t.Errorf("%s: %v", name, err)
		}
		gleich(name, f.png, f.svg)
	}
	// Ein anderer zlib-Level ändert die Bytes des Blocks, nicht seinen Inhalt.
	for _, level := range []int{zlib.NoCompression, zlib.BestSpeed, zlib.BestCompression} {
		if _, err := LeseArbeitsdatei(bytes.NewReader(pngMitSVGStufe(t, png, svg, level))); err != nil {
			t.Errorf("zlib-Stufe %d: %v", level, err)
		}
	}
	// Unsortierte Streifen liest der Leser sortiert (Spalten zuerst, je Achse aufsteigend).
	if g, err := LeseArbeitsdatei(bytes.NewReader(pngMitSVG(t, viewPNG, angenommen["Streifen unsortiert"].svg))); err != nil {
		t.Error(err)
	} else if want := []document.Streifen{{Zeilen: false, Von: 30, Bis: 33}, {Zeilen: true, Von: 20, Bis: 25},
		{Zeilen: true, Von: 30, Bis: 33}}; !reflect.DeepEqual(g.Dokument.Streifen(), want) {
		t.Errorf("Streifen unsortiert: %v, erwartet %v", g.Dokument.Streifen(), want)
	}

	// Ortsfreiheit (Regel 3): `s:streifen` und `s:schlagwort` zählen nur unter `s:ansicht` bzw. `s:meta`; an anderer
	// Stelle sind sie unbekannter Inhalt und ohne Wirkung.
	fremdStreifen := `<s:streifen s:zeilen="true" s:von="1" s:bis="2"/>`
	for name, s := range map[string]string{
		"Streifen unter s:meta":       ersetzeV(metaV+`</s:meta>`, metaV+fremdStreifen+`</s:meta>`),
		"Streifen in der Objektebene": ersetzeV(`<g s:ebene="objekte">`, `<g s:ebene="objekte">`+fremdStreifen),
		"Streifen unter der Wurzel":   ersetzeV(`</svg>`, fremdStreifen+`</svg>`),
	} {
		g, err := LeseArbeitsdatei(bytes.NewReader(pngMitSVG(t, viewPNG, s)))
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if len(g.Dokument.Streifen()) != 2 {
			t.Errorf("%s: %d Streifen, erwartet 2 (der fremde zählt nicht)", name, len(g.Dokument.Streifen()))
		}
		gleich(name, viewPNG, s)
	}
	for name, s := range map[string]string{
		"Schlagwort unter s:ansicht":  ersetze(`</s:ansicht>`, `<s:schlagwort>fremd</s:schlagwort></s:ansicht>`),
		"Schlagwort unter der Wurzel": ersetze(`</svg>`, `<s:schlagwort>fremd</s:schlagwort></svg>`),
	} {
		g, err := LeseArbeitsdatei(bytes.NewReader(pngMitSVG(t, png, s)))
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if !reflect.DeepEqual(g.Meta.Schlagworte, beispielMeta().Schlagworte) {
			t.Errorf("%s: Schlagworte %v", name, g.Meta.Schlagworte)
		}
		gleich(name, png, s)
	}

	// Neuere Version: abgelehnt, die Datei bleibt als Bild lesbar.
	neuer := ersetze(version, `s:format="`+strconv.Itoa(FormatVersion+1)+`"`)
	if err := lies(png, neuer); !errors.Is(err, ErrFormatNeuer) {
		t.Errorf("neuere Version: %v", err)
	}
	gleich("neuere Version", png, neuer)
	if _, err := LeseBild(bytes.NewReader(png)); err != nil {
		t.Error(err)
	}

	// Beschädigt oder zu groß: Sentinel und ein Teil des Fehlertexts.
	abgelehnt := []struct {
		name string
		png  []byte
		svg  string
		art  error
		teil string
	}{
		{"Version fehlt", png, ersetze(version, ``), ErrBeschaedigt, "format"},
		{"Version mit Plus", png, ersetze(version, `s:format="+1"`), ErrBeschaedigt, "format"},
		{"Ansicht fehlt", png, strings.Replace(ersetze(`<s:ansicht`, `<s:ansichtx`), `</s:ansicht>`, `</s:ansichtx>`, 1), ErrBeschaedigt, "Ansicht fehlt"},
		{"unbekannter Objekttyp", png, ersetze(`s:typ="kasten"`, `s:typ="ellipse"`), ErrBeschaedigt, "unbekannter Objekttyp"},
		{"Pflichtattribut fehlt", png, ersetze(` s:radius="7"`, ``), ErrBeschaedigt, "radius fehlt"},
		{"Zahl NaN", png, ersetze(`s:radius="7"`, `s:radius="NaN"`), ErrBeschaedigt, "Zahl radius"},
		{"Entität", png, strings.Replace(ersetze(`<s:meta`, `<s:meta s:autor="&x;"`), ` s:autor="Example Author"`, ``, 1), ErrBeschaedigt, "SVG"},
		{"Größe passt nicht", png, ersetze(`width="64" height="48" viewBox`, `width="65" height="48" viewBox`), ErrBeschaedigt, "passt nicht"},
		{"Wert außerhalb Grenzen", png, ersetze(`s:radius="7"`, `s:radius="1e9"`), ErrBeschaedigt, ""},
		// Farbe: genau # und acht Hexziffern
		{"Farbe zu kurz", png, ersetze(`#1e5ac8ff`, `#1e5ac8`), ErrBeschaedigt, "Farbe farbe"},
		{"Farbe mit Leerraum", png, ersetze(`#1e5ac8ff`, `# a b c d`), ErrBeschaedigt, "Farbe farbe"},
		{"Farbe mit Leerzeichen am Ende", png, ersetze(`#1e5ac8ff`, `#1e5ac8f `), ErrBeschaedigt, "Farbe farbe"},
		{"Farbe ohne Raute", png, ersetze(`#1e5ac8ff`, `x1e5ac8ffx`), ErrBeschaedigt, "Farbe farbe"},
		{"Farbe mit Vorzeichen", png, ersetze(`#1e5ac8ff`, `#+e5ac8ff`), ErrBeschaedigt, "Farbe farbe"},
		// Zahlen: Dezimalschreibweise
		{"Zahl hexadezimal", png, ersetze(`s:radius="7"`, `s:radius="0x1p3"`), ErrBeschaedigt, "Zahl radius"},
		{"Zahl mit Unterstrich", png, ersetze(`s:radius="7"`, `s:radius="1_0"`), ErrBeschaedigt, "Zahl radius"},
		{"Zahl mit Plus", png, ersetze(`s:radius="7"`, `s:radius="+7"`), ErrBeschaedigt, "Zahl radius"},
		{"Zahl Inf", png, ersetze(`s:radius="7"`, `s:radius="Inf"`), ErrBeschaedigt, "Zahl radius"},
		{"Zahl mit Leerraum", png, ersetze(`s:radius="7"`, `s:radius=" 7"`), ErrBeschaedigt, "Zahl radius"},
		{"Punkt mit Plus", png, ersetze(`s:mitte="52,10"`, `s:mitte="+52,10"`), ErrBeschaedigt, "Punkt mitte"},
		{"Ganzzahl mit Plus", png, ersetze(`s:wert="3"`, `s:wert="+3"`), ErrBeschaedigt, "Ganzzahl wert"},
		{"Ganzzahl mit Dezimalpunkt", png, ersetze(`s:wert="3"`, `s:wert="3.0"`), ErrBeschaedigt, "Ganzzahl wert"},
		{"Breite mit Plus", png, ersetze(`width="64" height="48" viewBox`, `width="+64" height="48" viewBox`), ErrBeschaedigt, "Ganzzahl"},
		// s:typ
		{"leeres s:typ", png, ersetze(`s:typ="kasten"`, `s:typ=""`), ErrBeschaedigt, "leeres s:typ"},
		{"s:typ an s:meta", png, ersetze(`<s:meta`, `<s:meta s:typ="kasten"`), ErrBeschaedigt, "Strukturelement"},
		{"s:typ an s:ansicht", png, ersetze(`<s:ansicht`, `<s:ansicht s:typ="kasten"`), ErrBeschaedigt, "Strukturelement"},
		{"s:typ an s:streifen", viewPNG, ersetzeV(`<s:streifen`, `<s:streifen s:typ="kasten"`), ErrBeschaedigt, "Strukturelement"},
		{"s:typ an s:schlagwort", png, ersetze(`<s:schlagwort>`, `<s:schlagwort s:typ="kasten">`), ErrBeschaedigt, "Strukturelement"},
		{"s:typ an der Wurzel", png, ersetze(version, version+` s:typ="kasten"`), ErrBeschaedigt, "Strukturelement"},
		// Zuschnitt (5.2 rectangle, 7.1)
		{"Zuschnitt mit Rest", viewPNG, ersetzeV(`s:zuschnitt="4 3 60 45"`, `s:zuschnitt="4 3 60 45 muell"`), ErrBeschaedigt, "Zuschnitt ungültig"},
		{"Zuschnitt mit doppeltem Leerzeichen", viewPNG, ersetzeV(`s:zuschnitt="4 3 60 45"`, `s:zuschnitt="4  3 60 45"`), ErrBeschaedigt, "Zuschnitt ungültig"},
		{"Zuschnitt mit Plus", viewPNG, ersetzeV(`s:zuschnitt="4 3 60 45"`, `s:zuschnitt="+4 3 60 45"`), ErrBeschaedigt, "Zuschnitt ungültig"},
		{"Zuschnitt mit Leerraum am Rand", viewPNG, ersetzeV(`s:zuschnitt="4 3 60 45"`, `s:zuschnitt=" 4 3 60 45"`), ErrBeschaedigt, "Zuschnitt ungültig"},
		{"Zuschnitt zu kurz", viewPNG, ersetzeV(`s:zuschnitt="4 3 60 45"`, `s:zuschnitt="4 3 60"`), ErrBeschaedigt, "Zuschnitt ungültig"},
		{"Zuschnitt dezimal", viewPNG, ersetzeV(`s:zuschnitt="4 3 60 45"`, `s:zuschnitt="4 3 60 45.5"`), ErrBeschaedigt, "Zuschnitt ungültig"},
		{"Zuschnitt außerhalb", viewPNG, ersetzeV(`s:zuschnitt="4 3 60 45"`, `s:zuschnitt="4 3 60 49"`), ErrBeschaedigt, "außerhalb des Bilds"},
		{"Zuschnitt negativ", viewPNG, ersetzeV(`s:zuschnitt="4 3 60 45"`, `s:zuschnitt="-1 3 60 45"`), ErrBeschaedigt, "außerhalb des Bilds"},
		{"Zuschnitt leer", viewPNG, ersetzeV(`s:zuschnitt="4 3 60 45"`, `s:zuschnitt="4 3 4 45"`), ErrBeschaedigt, "außerhalb des Bilds"},
		// Streifen (7.2, 7.3)
		{"Streifen überlappen", viewPNG, ersetzeV(`s:zeilen="false" s:von="30" s:bis="33"`, `s:zeilen="true" s:von="24" s:bis="27"`), ErrBeschaedigt, "überlappen"},
		{"Streifen von gleich bis", viewPNG, ersetzeV(`s:von="20" s:bis="25"`, `s:von="25" s:bis="25"`), ErrBeschaedigt, "ungültig"},
		{"Streifen von größer als bis", viewPNG, ersetzeV(`s:von="20" s:bis="25"`, `s:von="25" s:bis="20"`), ErrBeschaedigt, "ungültig"},
		{"Streifen über den Rand", viewPNG, ersetzeV(`s:von="20" s:bis="25"`, `s:von="20" s:bis="49"`), ErrBeschaedigt, "ungültig"},
		{"Streifen ohne Zahl", viewPNG, ersetzeV(`s:von="20"`, `s:von="zwanzig"`), ErrBeschaedigt, "Ganzzahl von"},
		{"leere Ansicht", viewPNG, ersetzeV(`s:von="20" s:bis="25"`, `s:von="0" s:bis="48"`), ErrBeschaedigt, "Ansicht wäre leer"},
	}
	for _, f := range abgelehnt {
		err := lies(f.png, f.svg)
		if !errors.Is(err, f.art) || !strings.Contains(err.Error(), f.teil) {
			t.Errorf("%s: %v (erwartet %v mit %q)", f.name, err, f.art, f.teil)
		}
		gleich(f.name, f.png, f.svg)
	}
}

// TestFormatMetadatenBeispiele: LeseMetadaten liest alle Beispiele mit demselben Ergebnis wie das volle Lesen
// (Regel 4 und 7).
func TestFormatMetadatenBeispiele(t *testing.T) {
	for _, b := range beispiele() {
		daten, err := os.ReadFile(filepath.Join(beispielOrdner, b.name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		k, err := LeseMetadaten(bytes.NewReader(daten))
		if err != nil {
			t.Fatalf("%s: %v", b.name, err)
		}
		g, err := LeseArbeitsdatei(bytes.NewReader(daten))
		if err != nil {
			t.Fatalf("%s: %v", b.name, err)
		}
		if !reflect.DeepEqual(k.Meta, g.Meta) || image.Pt(k.Breite, k.Hoehe) != g.Dokument.Groesse() {
			t.Errorf("%s: LeseMetadaten %+v %dx%d, voll %+v %v", b.name, k.Meta, k.Breite, k.Hoehe, g.Meta, g.Dokument.Groesse())
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
	// Nur Abschnitt 10: andere Tabellen (etwa die Beispielnamen in Abschnitt 14) haben ebenfalls Zeilen der Form `x` | y.
	von, bis := strings.Index(spez, "\n## 10."), strings.Index(spez, "\n## 11.")
	if von < 0 || bis < von {
		t.Fatal("docs/format.md: Abschnitte 10 und 11 nicht gefunden")
	}
	zeile := regexp.MustCompile("(?m)^\\| `([a-z_]+)` \\| ([^|]+?) \\|")
	werte := map[string]string{}
	for _, m := range zeile.FindAllStringSubmatch(spez[von:bis], -1) {
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
	// Rückrichtung: Jeder Schlüssel der Tabelle ist hier einer Zuordnung zum Code zugeordnet; sonst stünde in der
	// Spezifikation ein Wert, den kein Test festhält.
	for k := range werte {
		_, istZahl := zahlen[k]
		_, istText := texte[k]
		if !istZahl && !istText {
			t.Errorf("docs/format.md: Schlüssel `%s` der Tabelle Constants ist keinem Wert im Code zugeordnet", k)
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
	wortliste := []string{"format", "meta", "autor", "erstellt", "quelle", "bearbeitet", "schlagwort", "ansicht", "zuschnitt",
		"streifen", "zeilen", "von", "bis", "ebene", "typ", "a", "b", "k", "strich", "bereich", "fuellung", "eckradius",
		"ort", "inhalt", "grad", "farbe", "hintergrund", "umbruch", "mitte", "wert", "radius"}
	for _, w := range wortliste {
		if !gesehen[w] {
			t.Errorf("Beispiele enthalten s:%s nicht (Beispielabdeckung)", w)
		}
	}
	// Rückrichtung: Jedes `s:name` der Spezifikation steht in der Wortliste; ein Name, den die Spezifikation einführt
	// und den weder Schreiber noch Beispiele noch Leser kennen, fiele sonst nicht auf. `s:name` ist der Platzhalter
	// der Einleitung von Abschnitt 5.
	bekannt := map[string]bool{"name": true}
	for _, w := range wortliste {
		bekannt[w] = true
	}
	for _, m := range regexp.MustCompile("`s:([a-z]+)`").FindAllStringSubmatch(spez, -1) {
		if !bekannt[m[1]] {
			t.Errorf("docs/format.md nennt `s:%s`, das die Wortliste des Tests nicht kennt (Schreiber, Beispiele, Leser prüfen)", m[1])
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
