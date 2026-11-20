// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"errors"
	"image/png"
	"testing"
)

// kaputtePixel baut aus einer Arbeitsdatei eine, deren Pixelteil (IDAT des sichtbaren Bilds) und deren snAO-Block
// hinter dem IHDR Müll sind — Blockprüfsummen stimmen, dekodieren lässt sich nichts.
func kaputtePixel(t *testing.T, daten []byte) []byte {
	t.Helper()
	bl, err := leseBloecke(bytes.NewReader(daten))
	if err != nil {
		t.Fatal(err)
	}
	for i := range bl {
		switch bl[i].typ {
		case "IDAT":
			bl[i].daten = bytes.Repeat([]byte{0xAB}, len(bl[i].daten))
		case blockOriginal:
			// Signatur + Blockkopf + IHDR samt Prüfsumme (8+8+13+4) bleiben, danach 3 MB Müll.
			bl[i].daten = append(append([]byte{}, bl[i].daten[:8+8+13+4]...), bytes.Repeat([]byte{0x5A}, 3<<20)...)
		}
	}
	var buf bytes.Buffer
	if err := schreibeBloecke(&buf, bl); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestLeseMetadatenOhnePixel(t *testing.T) {
	d := volles(t)
	gut := schreibe(t, d, meta)
	k, err := LeseMetadaten(bytes.NewReader(gut))
	if err != nil {
		t.Fatal(err)
	}
	g := d.Groesse()
	if k.Fremd || k.Breite != g.X || k.Hoehe != g.Y {
		t.Errorf("Größe/Fremd: %+v, erwartet %v", k, g)
	}
	if k.Meta.Autor != meta.Autor || !k.Meta.Erstellt.Equal(meta.Erstellt) || k.Meta.Quelle != meta.Quelle ||
		!k.Meta.Bearbeitet || len(k.Meta.Schlagworte) != 3 {
		t.Errorf("Metadaten: %+v", k.Meta)
	}
	// Gleiche Angaben wie das volle Lesen.
	voll, err := LeseArbeitsdatei(bytes.NewReader(gut))
	if err != nil {
		t.Fatal(err)
	}
	if voll.Dokument.Groesse().X != k.Breite || voll.Meta.Autor != k.Meta.Autor {
		t.Errorf("weicht vom vollen Lesen ab")
	}

	// Kaputter Pixelteil und riesiger snAO: Die volle Lesung scheitert, die Metadaten kommen.
	kaputt := kaputtePixel(t, gut)
	if _, err := LeseArbeitsdatei(bytes.NewReader(kaputt)); err == nil {
		t.Fatal("volles Lesen müsste an den kaputten Pixeln scheitern")
	}
	k2, err := LeseMetadaten(bytes.NewReader(kaputt))
	if err != nil {
		t.Fatalf("Metadaten trotz kaputter Pixel: %v", err)
	}
	if k2.Meta.Autor != meta.Autor || k2.Breite != g.X || k2.Hoehe != g.Y {
		t.Errorf("Metadaten bei kaputten Pixeln: %+v", k2)
	}
}

func TestLeseMetadatenFremdUndFehler(t *testing.T) {
	var fremd bytes.Buffer
	png.Encode(&fremd, testbild(5, 4))
	k, err := LeseMetadaten(bytes.NewReader(fremd.Bytes()))
	if err != nil || !k.Fremd || k.Breite != 5 || k.Hoehe != 4 {
		t.Errorf("fremdes PNG: %+v %v", k, err)
	}
	if _, err := LeseMetadaten(bytes.NewReader([]byte("kein PNG, nur Text"))); !errors.Is(err, ErrKeinPNG) {
		t.Errorf("kein PNG: %v", err)
	}
	gut := schreibe(t, volles(t), meta)
	// abgeschnitten
	if _, err := LeseMetadaten(bytes.NewReader(gut[:len(gut)/2])); !IstBeschaedigt(err) {
		t.Errorf("abgeschnitten: %v", err)
	}
	// Daten nach IEND
	if _, err := LeseMetadaten(bytes.NewReader(append(append([]byte{}, gut...), '\n'))); !IstBeschaedigt(err) {
		t.Errorf("Anhang: %v", err)
	}
	// falsche Prüfsumme in einem übersprungenen Block (IDAT)
	bl, _ := leseBloecke(bytes.NewReader(gut))
	var buf bytes.Buffer
	schreibeBloecke(&buf, bl)
	roh := buf.Bytes()
	i := bytes.Index(roh, []byte("IDAT"))
	roh[i+5] ^= 0xFF
	if _, err := LeseMetadaten(bytes.NewReader(roh)); !errors.Is(err, ErrBeschaedigt) {
		t.Errorf("Prüfsumme: %v", err)
	}
	// nur einer der beiden Blöcke
	var einer bytes.Buffer
	schreibeBloecke(&einer, ohneBloecke(bl, blockOriginal))
	if _, err := LeseMetadaten(&einer); !errors.Is(err, ErrBeschaedigt) {
		t.Errorf("nur snAP: %v", err)
	}
	// doppelter snAP
	var dop []block
	for _, b := range bl {
		dop = append(dop, b)
		if b.typ == blockSVG {
			dop = append(dop, b)
		}
	}
	var dbuf bytes.Buffer
	schreibeBloecke(&dbuf, dop)
	if _, err := LeseMetadaten(&dbuf); !errors.Is(err, ErrBeschaedigt) {
		t.Errorf("doppelter snAP: %v", err)
	}
}

func TestLeseMetadatenGrenzen(t *testing.T) {
	// Block über maxBlock wird vor dem Lesen abgelehnt (wie leseBloecke, ohne Speicher dafür anzulegen).
	var buf bytes.Buffer
	buf.Write(pngSignatur)
	buf.Write([]byte{0xFF, 0xFF, 0xFF, 0xFF, 'I', 'H', 'D', 'R'})
	if _, err := LeseMetadaten(&buf); !errors.Is(err, ErrZuGross) {
		t.Errorf("Riesenblock: %v", err)
	}
	// Original in anderer Größe als im SVG: beschädigt.
	gut := schreibe(t, volles(t), meta)
	bl, _ := leseBloecke(bytes.NewReader(gut))
	var anders bytes.Buffer
	png.Encode(&anders, testbild(7, 7))
	for i := range bl {
		if bl[i].typ == blockOriginal {
			bl[i].daten = anders.Bytes()
		}
	}
	var out bytes.Buffer
	schreibeBloecke(&out, bl)
	if _, err := LeseMetadaten(&out); !errors.Is(err, ErrBeschaedigt) {
		t.Errorf("Original passt nicht: %v", err)
	}
}
