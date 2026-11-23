// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

func testbild(b, h int) *image.NRGBA {
	m := image.NewNRGBA(image.Rect(0, 0, b, h))
	for y := 0; y < h; y++ {
		for x := 0; x < b; x++ {
			m.SetNRGBA(x, y, color.NRGBA{uint8(x * 3), uint8(y * 5), 90, 255})
		}
	}
	return m
}

var rot = document.Strich{Farbe: color.NRGBA{220, 30, 30, 255}, Breite: 3.5}

// volles baut ein Dokument mit allen Objektarten, Zuschnitt, Streifen und einer Schwärzung.
func volles(t *testing.T) *document.Dokument {
	t.Helper()
	d, err := document.Neu(testbild(80, 60))
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range []document.Objekt{
		document.GeraderPfeil(document.Pt(1, 2), document.Pt(70.25, 40), rot).MitGriff(document.Pt(30, 1e-3)),
		document.Kasten{Bereich: document.R(document.Pt(5, 5), document.Pt(25.5, 15)), Strich: rot,
			Fuellung: color.NRGBA{255, 255, 0, 64}, Eckradius: 2},
		document.Text{Ort: document.Pt(10, 30), Inhalt: "Zeile <1> & \"2\"\nÄß\tTab", Schriftgrad: 14,
			Farbe: color.NRGBA{0, 0, 0, 255}, Hintergrund: color.NRGBA{255, 255, 255, 200}, BreiteGrenze: 50},
		document.Nummer{Mitte: document.Pt(60, 10), Wert: 3, Radius: 9, Farbe: color.NRGBA{30, 90, 200, 255}},
	} {
		if _, err := d.Hinzufuegen(o); err != nil {
			t.Fatal(err)
		}
	}
	if err := d.Schwaerzen(image.Rect(40, 40, 50, 50), document.Schwaerzung{Farbe: color.NRGBA{0, 0, 0, 255}}); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeZuschnitt(image.Rect(2, 3, 78, 58)); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 20, Bis: 25}, {Zeilen: false, Von: 30, Bis: 33}}); err != nil {
		t.Fatal(err)
	}
	return d
}

var meta = Metadaten{Autor: "Jo <Test> & \"Co\"", Erstellt: time.Date(2026, 11, 19, 14, 35, 12, 0, time.UTC),
	Quelle: "Beispiel", Schlagworte: []string{"a&b", "<x>", "Leer zeichen"}, Bearbeitet: true}

func schreibe(t *testing.T, d *document.Dokument, m Metadaten) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := SchreibeArbeitsdatei(&buf, d, d.Ansichtsbild(), m); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func blockDaten(t *testing.T, daten []byte, typ string) []byte {
	t.Helper()
	bl, err := leseBloecke(bytes.NewReader(daten))
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range bl {
		if b.typ == typ {
			return b.daten
		}
	}
	t.Fatalf("Block %s fehlt", typ)
	return nil
}

func TestRundlauf(t *testing.T) {
	d := volles(t)
	daten := schreibe(t, d, meta)
	g, err := LeseArbeitsdatei(bytes.NewReader(daten))
	if err != nil {
		t.Fatal(err)
	}
	if g.Fremd {
		t.Error("eigene Datei als fremd erkannt")
	}
	if !reflect.DeepEqual(g.Dokument.Objekte(), d.Objekte()) {
		t.Errorf("Objekte:\n got %#v\nwant %#v", g.Dokument.Objekte(), d.Objekte())
	}
	if g.Dokument.Zuschnitt() != d.Zuschnitt() || !reflect.DeepEqual(g.Dokument.Streifen(), d.Streifen()) {
		t.Errorf("Ansicht: %v %v", g.Dokument.Zuschnitt(), g.Dokument.Streifen())
	}
	if !bytes.Equal(g.Dokument.Original().Pix, d.Original().Pix) {
		t.Error("Original verändert")
	}
	if !reflect.DeepEqual(g.Meta, meta) {
		t.Errorf("Metadaten:\n got %#v\nwant %#v", g.Meta, meta)
	}
	if g.Dokument.KannRueckgaengig() {
		t.Error("geladenes Dokument hat Verlauf")
	}
	// Zweiter Rundlauf ergibt dieselben Bytes (stabile Ausgabe).
	if !bytes.Equal(schreibe(t, g.Dokument, g.Meta), daten) {
		t.Error("zweiter Rundlauf schreibt andere Bytes")
	}
}

// Das sichtbare Bild lesen alle PNG-Programme; die Blöcke sind zusätzlich, privat und nicht kopiersicher.
func TestBloeckeNachSpezifikation(t *testing.T) {
	d := volles(t)
	daten := schreibe(t, d, Metadaten{})
	bild, err := png.Decode(bytes.NewReader(daten))
	if err != nil {
		t.Fatal(err)
	}
	if !gleich(bild, d.Ansichtsbild()) {
		t.Error("sichtbares Bild ist nicht die Ansicht")
	}
	bl, _ := leseBloecke(bytes.NewReader(daten))
	var typen []string
	for _, b := range bl {
		typen = append(typen, b.typ)
	}
	n := len(typen)
	if typen[n-3] != blockSVG || typen[n-2] != blockOriginal || typen[n-1] != "IEND" {
		t.Errorf("Reihenfolge %v", typen)
	}
	for _, name := range []string{blockSVG, blockOriginal} {
		klein := func(c byte) bool { return c&0x20 != 0 }
		if !klein(name[0]) || !klein(name[1]) || klein(name[2]) || klein(name[3]) {
			t.Errorf("%s: zusätzlich, privat, reserviert, nicht kopiersicher verlangt", name)
		}
	}
	if b, err := LeseBild(bytes.NewReader(daten)); err != nil || b.Bounds() != d.Ansichtsbild().Bounds() {
		t.Errorf("LeseBild: %v %v", b, err)
	}
}

// gleich vergleicht zwei Bilder Pixel für Pixel (NRGBA), unabhängig vom Bildtyp.
func gleich(a, b image.Image) bool {
	if a.Bounds() != b.Bounds() {
		return false
	}
	r := a.Bounds()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if color.NRGBAModel.Convert(a.At(x, y)) != color.NRGBAModel.Convert(b.At(x, y)) {
				return false
			}
		}
	}
	return true
}

func TestFremdesPNG(t *testing.T) {
	var buf bytes.Buffer
	png.Encode(&buf, testbild(7, 4))
	g, err := LeseArbeitsdatei(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if !g.Fremd || g.Dokument.Original().Rect.Dx() != 7 || len(g.Dokument.Objekte()) != 0 {
		t.Errorf("fremdes PNG: %+v", g)
	}
}

// ersetzeBlock tauscht die Daten eines Blocks (mit neuer, gültiger Prüfsumme).
func ersetzeBlock(t *testing.T, daten []byte, typ string, neu []byte) []byte {
	t.Helper()
	bl, _ := leseBloecke(bytes.NewReader(daten))
	for i := range bl {
		if bl[i].typ == typ {
			bl[i].daten = neu
		}
	}
	var buf bytes.Buffer
	schreibeBloecke(&buf, bl)
	return buf.Bytes()
}

func zlibVon(s string) []byte {
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	zw.Write([]byte(s))
	zw.Close()
	return buf.Bytes()
}

func TestBeschaedigt(t *testing.T) {
	d := volles(t)
	gut := schreibe(t, d, meta)
	svg := func(s string) []byte { return ersetzeBlock(t, gut, blockSVG, zlibVon(s)) }
	kopf := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:s="` + ns + `" s:format="1" width="80" height="60">`
	ansicht := `<metadata><s:ansicht s:zuschnitt="0 0 80 60"/></metadata>`

	falscheCRC := append([]byte(nil), gut...)
	falscheCRC[len(falscheCRC)-5] ^= 1 // letzte Prüfsumme vor IEND gehört zu snAO

	riesig := new(bytes.Buffer)
	png.Encode(riesig, testbild(2, 2))
	ihdr := riesig.Bytes()[8+8 : 8+8+13]
	binary.BigEndian.PutUint32(ihdr[0:4], document.MaxSeite+1)
	crc := crc32.NewIEEE()
	crc.Write([]byte("IHDR"))
	crc.Write(ihdr)
	binary.BigEndian.PutUint32(riesig.Bytes()[8+8+13:], crc.Sum32())

	nurSVG := func() []byte {
		bl, _ := leseBloecke(bytes.NewReader(gut))
		var buf bytes.Buffer
		schreibeBloecke(&buf, ohneBloecke(bl, blockOriginal))
		return buf.Bytes()
	}()

	// Bis auf den geprüften Punkt gültig, damit der Fall genau die Grenze trifft.
	tief := kopf + ansicht + strings.Repeat("<g>", 100) + strings.Repeat("</g>", 100) + `</svg>`
	text := `<g s:typ="text" s:ort="1,1" s:grad="10" s:farbe="#000000ff" s:hintergrund="#00000000" s:umbruch="0" s:inhalt=`

	for name, daten := range map[string][]byte{
		"kein PNG":          []byte("GIF89a…"),
		"abgeschnitten":     gut[:len(gut)/2],
		"Prüfsumme falsch":  falscheCRC,
		"nur ein Block":     nurSVG,
		"kein zlib":         ersetzeBlock(t, gut, blockSVG, []byte("kein zlib")),
		"kein XML":          svg("<svg"),
		"Wurzel falsch":     svg(`<html/>`),
		"Version fehlt":     svg(`<svg xmlns="http://www.w3.org/2000/svg" width="80" height="60">` + ansicht + `</svg>`),
		"Version neuer":     svg(strings.Replace(kopf, `s:format="1"`, `s:format="3"`, 1) + ansicht + `</svg>`),
		"Ansicht fehlt":     svg(kopf + `</svg>`),
		"unbekannter Typ":   svg(kopf + ansicht + `<g s:typ="stempel"/></svg>`),
		"Farbe kaputt":      svg(kopf + ansicht + `<g s:typ="nummer" s:mitte="1,1" s:wert="1" s:radius="2" s:farbe="rot"/></svg>`),
		"Zahl NaN":          svg(kopf + ansicht + `<g s:typ="nummer" s:mitte="1,1" s:wert="1" s:radius="NaN" s:farbe="#00000000"/></svg>`),
		"Größe passt nicht": svg(strings.Replace(kopf, `width="80"`, `width="81"`, 1) + ansicht + `</svg>`),
		"Zuschnitt außen":   svg(kopf + `<metadata><s:ansicht s:zuschnitt="0 0 99 60"/></metadata></svg>`),
		"zu tief":           svg(tief),
		"Entität":           svg(`<!DOCTYPE svg [<!ENTITY x "y">]>` + kopf + ansicht + text + `"&x;"/></svg>`),
		"Original zu groß":  ersetzeBlock(t, gut, blockOriginal, riesig.Bytes()),
		"Original kein PNG": ersetzeBlock(t, gut, blockOriginal, []byte("x")),
		"Block doppelt":     doppelt(t, gut),
		"Daten nach IEND":   append(append([]byte(nil), gut...), 'x'),
	} {
		if _, err := LeseArbeitsdatei(bytes.NewReader(daten)); err == nil {
			t.Errorf("%s: kein Fehler", name)
		} else if name == "Version neuer" {
			if !errors.Is(err, ErrFormatNeuer) || IstBeschaedigt(err) {
				t.Errorf("%s: %v ist nicht ErrFormatNeuer", name, err)
			}
		} else if !IstBeschaedigt(err) {
			t.Errorf("%s: Fehler %v nicht als beschädigt erkennbar", name, err)
		}
	}
	// Gegenstücke: dieselben Fälle ohne den geprüften Fehler sind gültig.
	for name, daten := range map[string][]byte{
		"flach statt tief":  svg(kopf + ansicht + strings.Repeat("<g>", 60) + strings.Repeat("</g>", 60) + `</svg>`),
		"Text ohne Entität": svg(kopf + ansicht + text + `"y"/></svg>`),
	} {
		if _, err := LeseArbeitsdatei(bytes.NewReader(daten)); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	// Eine beschädigte Arbeitsdatei lässt sich als einfaches Bild öffnen.
	if _, err := LeseBild(bytes.NewReader(nurSVG)); err != nil {
		t.Errorf("LeseBild einer beschädigten Datei: %v", err)
	}
}

// Keine Spuren nach außen (Kernprinzip 3, ADR-0008): weder Rechnername noch Heimatordner noch Werkzeugnamen in den
// Bytes oder im entpackten SVG.
func TestKeineSpuren(t *testing.T) {
	d := volles(t)
	daten := schreibe(t, d, Metadaten{})
	zr, err := zlib.NewReader(bytes.NewReader(blockDaten(t, daten, blockSVG)))
	if err != nil {
		t.Fatal(err)
	}
	svg, _ := io.ReadAll(zr)
	var exp bytes.Buffer
	ExportiereSVG(&exp, d)
	verboten := []string{"golang", "Go-http", "/home/", "/tmp/", "Software", "Creator", "Producer"}
	if h, err := os.Hostname(); err == nil && len(h) > 2 {
		verboten = append(verboten, h)
	}
	if h, err := os.UserHomeDir(); err == nil && len(h) > 1 {
		verboten = append(verboten, h)
	}
	for _, quelle := range map[string][]byte{"Datei": daten, "SVG": svg, "Export": exp.Bytes()} {
		for i, v := range verboten {
			if bytes.Contains(quelle, []byte(v)) {
				t.Errorf("verbotene Angabe Nr. %d gefunden", i) // den Wert selbst nicht ausgeben (Rechnername)
			}
		}
	}
	// Ohne Autor und Datum stehen keine Attribute dafür im SVG.
	if bytes.Contains(svg, []byte("s:autor")) || bytes.Contains(svg, []byte("s:erstellt")) {
		t.Error("leere Metadaten geschrieben")
	}
}

// muster ist eine Farbe, die nur im zu schwärzenden Bereich vorkommt.
var muster = color.NRGBA{13, 37, 201, 255}

func enthaeltFarbe(b image.Image, c color.NRGBA) bool {
	r := b.Bounds()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if color.NRGBAModel.Convert(b.At(x, y)).(color.NRGBA) == c {
				return true
			}
		}
	}
	return false
}

// alleBilder liefert jedes Bild, das in einer der Ausgaben steckt: sichtbares PNG, snAO-Original, flaches PNG und
// das im SVG-Export eingebettete Original. JPG wird verlustbehaftet geprüft (kein Pixel nahe am Muster).
// svgKacheln liefert alle im SVG eingebetteten PNG-Bilder.
func svgKacheln(t *testing.T, s string) []image.Image {
	t.Helper()
	const kopf = "data:image/png;base64,"
	var res []image.Image
	for {
		i := strings.Index(s, kopf)
		if i < 0 {
			return res
		}
		s = s[i+len(kopf):]
		j := strings.Index(s, `"`)
		roh, err := base64.StdEncoding.DecodeString(s[:j])
		if err != nil {
			t.Fatal(err)
		}
		b, err := png.Decode(bytes.NewReader(roh))
		if err != nil {
			t.Fatal(err)
		}
		res = append(res, b)
		s = s[j:]
	}
}

func alleBilder(t *testing.T, d *document.Dokument) map[string]image.Image {
	t.Helper()
	res := map[string]image.Image{}
	daten := schreibe(t, d, Metadaten{})
	res["sichtbar"], _ = png.Decode(bytes.NewReader(daten))
	res["snAO"], _ = png.Decode(bytes.NewReader(blockDaten(t, daten, blockOriginal)))
	var flach bytes.Buffer
	ExportierePNG(&flach, d.Ansichtsbild())
	res["flach"], _ = png.Decode(&flach)
	var exp bytes.Buffer
	ExportiereSVG(&exp, d)
	for i, k := range svgKacheln(t, exp.String()) {
		res["svg-"+strconv.Itoa(i)] = k
	}
	var jpg bytes.Buffer
	ExportiereJPG(&jpg, d.Ansichtsbild(), 100)
	res["jpg"], _ = jpeg.Decode(&jpg)
	for name, b := range res {
		if b == nil {
			t.Fatalf("Bild %s nicht lesbar", name)
		}
	}
	return res
}

func nahAmMuster(b image.Image) bool {
	r := b.Bounds()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			c := color.NRGBAModel.Convert(b.At(x, y)).(color.NRGBA)
			if abs(int(c.R)-int(muster.R)) < 12 && abs(int(c.G)-int(muster.G)) < 12 && abs(int(c.B)-int(muster.B)) < 12 {
				return true
			}
		}
	}
	return false
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// [S] Geschwärzter Inhalt steckt in keiner Ausgabe — auch nicht im mitgespeicherten Original und auch dann nicht,
// wenn der geschwärzte Bereich weggeschnitten oder weggestaucht ist (ADR-0008, Kernprinzip 2).
//
// Schwarz füllen: eine einfarbige Fläche (Muster) darf nirgends übrig bleiben, auch im JPG nicht annähernd.
// Verpixeln: Eine einfarbige Fläche behielte ihre Farbe (Mittelwert), das ist kein Leck. Geprüft wird deshalb ein
// Schachbrett aus zwei Farben wie die Kanten von Schrift: danach darf keine der beiden Farben übrig sein.
func TestSchwaerzenUeberAlleWege(t *testing.T) {
	zweite := color.NRGBA{240, 230, 20, 255}
	faelle := []struct {
		name        string
		s           document.Schwaerzung
		schachbrett bool
	}{
		{"schwarz", document.Schwaerzung{Farbe: color.NRGBA{0, 0, 0, 255}}, false},
		{"verpixelt", document.Schwaerzung{Block: document.MinBlock}, true},
	}
	for _, f := range faelle {
		quelle := testbild(80, 60)
		for y := 10; y < 20; y++ {
			for x := 60; x < 75; x++ {
				c := muster
				if f.schachbrett && (x+y)%2 == 1 {
					c = zweite
				}
				quelle.SetNRGBA(x, y, c)
			}
		}
		pruefe := func(name string, b image.Image) bool {
			switch {
			case name == "jpg" && f.schachbrett:
				return false // Schachbrett übersteht JPG ohnehin nicht pixelgenau; geprüft in den verlustfreien Wegen
			case name == "jpg":
				return nahAmMuster(b)
			case f.schachbrett:
				return enthaeltFarbe(b, muster) || enthaeltFarbe(b, zweite)
			}
			return enthaeltFarbe(b, muster)
		}
		d, err := document.Neu(quelle)
		if err != nil {
			t.Fatal(err)
		}
		// Gegenprobe der Prüfung selbst: vor dem Schwärzen findet sie das Muster (außer im JPG-Sonderfall).
		for name, b := range alleBilder(t, d) {
			if !pruefe(name, b) && !(name == "jpg" && f.schachbrett) {
				t.Fatalf("%s: Prüfung findet das Muster in %s nicht einmal vor dem Schwärzen", f.name, name)
			}
		}
		if err := d.Schwaerzen(image.Rect(60, 10, 75, 20), f.s); err != nil {
			t.Fatal(err)
		}
		// Erst mit sichtbarem Bereich (alle Wege zeigen ihn, keiner darf das Muster tragen), dann mit weggestauchtem
		// Bereich (das mitgespeicherte Original darf es trotzdem nicht zeigen).
		for _, st := range [][]document.Streifen{nil, {{Zeilen: true, Von: 5, Bis: 25}}} {
			if err := d.SetzeStreifen(st); err != nil {
				t.Fatal(err)
			}
			for name, b := range alleBilder(t, d) {
				if pruefe(name, b) {
					t.Errorf("%s, Streifen %v: Muster in %s noch vorhanden", f.name, st, name)
				}
			}
		}
	}
}

func TestExportSVG(t *testing.T) {
	d := volles(t)
	var buf bytes.Buffer
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	// Gültiges XML; eine Kachel je sichtbarem Rechteck: Spalten 2..30, 33..78 × Zeilen 3..20, 25..58 = 4.
	dec := xml.NewDecoder(&buf)
	dec.Strict = true
	var kacheln []string
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if e, ok := tok.(xml.StartElement); ok && e.Name.Local == "svg" {
			for _, a := range e.Attr {
				if a.Name.Local == "viewBox" {
					kacheln = append(kacheln, a.Value)
				}
			}
		}
	}
	// Kacheln ab 0,0, ohne Lage im Original (F-225); die Verschiebung steht in einem transform.
	want := []string{"0 0 73 50", "0 0 28 17", "0 0 45 17", "0 0 28 33", "0 0 45 33"}
	if !reflect.DeepEqual(kacheln, want) {
		t.Errorf("viewBoxen %v, want %v", kacheln, want)
	}
	if g := d.Groesse(); g != image.Pt(73, 50) {
		t.Errorf("Größe %v", g)
	}
}

func TestExportJPG(t *testing.T) {
	for _, q := range []int{0, 101} {
		if ExportiereJPG(io.Discard, testbild(2, 2), q) == nil {
			t.Errorf("Qualität %d ohne Fehler", q)
		}
	}
	// Transparenz wird Weiß.
	b := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	var buf bytes.Buffer
	if err := ExportiereJPG(&buf, b, 95); err != nil {
		t.Fatal(err)
	}
	j, _ := jpeg.Decode(&buf)
	if r, g, bb, _ := j.At(3, 3).RGBA(); r>>8 < 250 || g>>8 < 250 || bb>>8 < 250 {
		t.Errorf("transparent → %v", j.At(3, 3))
	}
}

func TestLeseBloeckeGrenzen(t *testing.T) {
	var buf bytes.Buffer
	buf.Write(pngSignatur)
	binary.Write(&buf, binary.BigEndian, uint32(maxBlock+1))
	buf.WriteString("IDAT")
	if _, err := leseBloecke(&buf); !errors.Is(err, ErrZuGross) {
		t.Errorf("übergroßer Block: %v", err)
	}
}

// doppelt liefert die Datei mit einem zweiten snAP-Block.
func doppelt(t *testing.T, daten []byte) []byte {
	t.Helper()
	bl, _ := leseBloecke(bytes.NewReader(daten))
	var neu []block
	for _, b := range bl {
		neu = append(neu, b)
		if b.typ == blockSVG {
			neu = append(neu, b)
		}
	}
	var buf bytes.Buffer
	schreibeBloecke(&buf, neu)
	return buf.Bytes()
}

// F-60: Der SVG-Export enthält keine weggeschnittenen oder weggestauchten Pixel; die Kacheln decken genau die Ansicht.
func TestExportSVGOhneWeggeschnittenes(t *testing.T) {
	quelle := testbild(80, 60)
	for _, p := range []image.Point{{1, 1}, {40, 30}} { // (1,1) weggeschnitten, (40,30) weggestaucht
		quelle.SetNRGBA(p.X, p.Y, muster)
	}
	d, err := document.Neu(quelle)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeZuschnitt(image.Rect(10, 10, 70, 50)); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 28, Bis: 33}}); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	flaeche := 0
	for _, k := range svgKacheln(t, buf.String()) {
		if enthaeltFarbe(k, muster) {
			t.Error("Kachel enthält weggeschnittenen oder weggestauchten Inhalt")
		}
		flaeche += k.Bounds().Dx() * k.Bounds().Dy()
	}
	if g := d.Groesse(); flaeche != g.X*g.Y {
		t.Errorf("Kacheln decken %d Pixel, Ansicht hat %d", flaeche, g.X*g.Y)
	}
}

// F-66: Beschriftungen ganz außerhalb der Ansicht stehen nicht im SVG-Export; angeschnittene bleiben.
func TestExportSVGOhneObjekteAusserhalb(t *testing.T) {
	d, err := document.Neu(testbild(80, 60))
	if err != nil {
		t.Fatal(err)
	}
	schwarz := color.NRGBA{0, 0, 0, 255}
	for _, o := range []document.Objekt{
		document.Text{Ort: document.Pt(2, 2), Inhalt: "GEHEIMTEXT", Schriftgrad: 5, Farbe: schwarz},
		document.Text{Ort: document.Pt(30, 30), Inhalt: "WEGGESTAUCHT", Schriftgrad: 4, Farbe: schwarz},
		document.Text{Ort: document.Pt(45, 12), Inhalt: "SICHTBAR", Schriftgrad: 5, Farbe: schwarz},
		// Angeschnitten: beginnt links außerhalb des Zuschnitts (x 10) und reicht hinein — muss bleiben.
		document.Text{Ort: document.Pt(6, 20), Inhalt: "ANGESCHNITTEN", Schriftgrad: 5, Farbe: schwarz},
	} {
		if _, err := d.Hinzufuegen(o); err != nil {
			t.Fatal(err)
		}
	}
	if err := d.SetzeZuschnitt(image.Rect(10, 10, 70, 50)); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 28, Bis: 40}}); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	for _, weg := range []string{"GEHEIMTEXT", "WEGGESTAUCHT"} {
		if strings.Contains(s, weg) {
			t.Errorf("%s steht im Export", weg)
		}
	}
	for _, da := range []string{"SICHTBAR", "ANGESCHNITTEN"} {
		if !strings.Contains(s, da) {
			t.Errorf("%s fehlt im Export", da)
		}
	}
}

// F-69: Die Pixelgrenze greift schon vor dem Dekodieren (PNG-Kopf 32768×16384, beide Seiten erlaubt, Fläche nicht);
// ein doppelter snAO ist so beschädigt wie ein doppelter snAP.
func TestGrenzenVorDemDekodieren(t *testing.T) {
	var buf bytes.Buffer
	png.Encode(&buf, testbild(2, 2))
	roh := buf.Bytes()
	ihdr := roh[8+8 : 8+8+13]
	binary.BigEndian.PutUint32(ihdr[0:4], document.MaxSeite)
	binary.BigEndian.PutUint32(ihdr[4:8], document.MaxPixel/document.MaxSeite+1)
	crc := crc32.NewIEEE()
	crc.Write([]byte("IHDR"))
	crc.Write(ihdr)
	binary.BigEndian.PutUint32(roh[8+8+13:], crc.Sum32())
	if _, err := decodiere(roh); !errors.Is(err, ErrZuGross) {
		t.Errorf("Fläche über MaxPixel: %v", err)
	}
	gut := schreibe(t, volles(t), Metadaten{})
	bl, _ := leseBloecke(bytes.NewReader(gut))
	var neu []block
	for _, b := range bl {
		neu = append(neu, b)
		if b.typ == blockOriginal {
			neu = append(neu, b)
		}
	}
	var d bytes.Buffer
	schreibeBloecke(&d, neu)
	if _, err := LeseArbeitsdatei(&d); !IstBeschaedigt(err) {
		t.Errorf("doppelter snAO: %v", err)
	}
}

// F-65: Daten nach IEND machen nur eine Arbeitsdatei beschädigt; fremde PNG und LeseBild bleiben lesbar.
func TestDatenNachIEND(t *testing.T) {
	var fremd bytes.Buffer
	png.Encode(&fremd, testbild(5, 4))
	fremd.WriteString("\n")
	if g, err := LeseArbeitsdatei(bytes.NewReader(fremd.Bytes())); err != nil || !g.Fremd {
		t.Errorf("fremdes PNG mit Anhang: %v", err)
	}
	if _, err := LeseBild(bytes.NewReader(fremd.Bytes())); err != nil {
		t.Errorf("LeseBild fremd mit Anhang: %v", err)
	}
	arbeit := append(schreibe(t, volles(t), Metadaten{}), '\n')
	if _, err := LeseArbeitsdatei(bytes.NewReader(arbeit)); !IstBeschaedigt(err) {
		t.Errorf("Arbeitsdatei mit Anhang: %v", err)
	}
	if _, err := LeseBild(bytes.NewReader(arbeit)); err != nil {
		t.Errorf("LeseBild Arbeitsdatei mit Anhang: %v", err)
	}
}

// F-68: Text mit CRLF übersteht den Rundlauf unverändert.
func TestRundlaufCRLF(t *testing.T) {
	d, _ := document.Neu(testbild(10, 10))
	text := document.Text{Ort: document.Pt(1, 1), Inhalt: "Windows\r\nZeile\x7f", Schriftgrad: 8, Farbe: color.NRGBA{0, 0, 0, 255}}
	if _, err := d.Hinzufuegen(text); err != nil {
		t.Fatal(err)
	}
	g, err := LeseArbeitsdatei(bytes.NewReader(schreibe(t, d, Metadaten{})))
	if err != nil {
		t.Fatal(err)
	}
	if o := g.Dokument.Objekte()[0].(document.Text); o.Inhalt != text.Inhalt {
		t.Errorf("Inhalt %q, want %q", o.Inhalt, text.Inhalt)
	}
}

// F-118: Eine Arbeitsdatei mit zu langem Text wird beim Lesen abgelehnt (die Prüfung sitzt in document und greift über
// Hinzufuegen); genau MaxTextZeichen bleibt lesbar.
func TestLesenZuLangerText(t *testing.T) {
	d, err := document.Neu(testbild(80, 60))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Hinzufuegen(document.Text{Ort: document.Pt(1, 1), Inhalt: "QQQ", Schriftgrad: 10,
		Farbe: color.NRGBA{0, 0, 0, 255}}); err != nil {
		t.Fatal(err)
	}
	gut := schreibe(t, d, meta)
	zr, err := zlib.NewReader(bytes.NewReader(blockDaten(t, gut, blockSVG)))
	if err != nil {
		t.Fatal(err)
	}
	svg, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Count(svg, []byte("QQQ")) < 1 {
		t.Fatalf("Platzhalter fehlt im SVG: %s", svg)
	}
	mit := func(n int) []byte {
		neu := bytes.Replace(svg, []byte("QQQ"), bytes.Repeat([]byte("Q"), n), -1)
		return ersetzeBlock(t, gut, blockSVG, zlibVon(string(neu)))
	}
	if _, err := LeseArbeitsdatei(bytes.NewReader(mit(document.MaxTextZeichen))); err != nil {
		t.Errorf("Text mit genau MaxTextZeichen abgelehnt: %v", err)
	}
	if _, err := LeseArbeitsdatei(bytes.NewReader(mit(document.MaxTextZeichen + 1))); err == nil {
		t.Error("Text mit MaxTextZeichen+1 ohne Fehler gelesen")
	}
}
