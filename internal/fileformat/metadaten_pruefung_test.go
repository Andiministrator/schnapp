// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

// Tests zu F-272 (LeseMetadaten urteilt und misst wie das volle Lesen) und F-273 (angekündigte Blocklängen).

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"io"
	"regexp"
	"runtime"
	"testing"
)

func ersetzeBlockMit(t *testing.T, daten []byte, typ string, f func([]byte) []byte) []byte {
	t.Helper()
	bl, err := leseBloecke(bytes.NewReader(daten))
	if err != nil {
		t.Fatal(err)
	}
	for i := range bl {
		if bl[i].typ == typ {
			bl[i].daten = f(bl[i].daten)
		}
	}
	var buf bytes.Buffer
	if err := schreibeBloecke(&buf, bl); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func ersetzeSVG(t *testing.T, daten []byte, f func(string) string) []byte {
	t.Helper()
	return ersetzeBlockMit(t, daten, blockSVG, func(alt []byte) []byte {
		zr, err := zlib.NewReader(bytes.NewReader(alt))
		if err != nil {
			t.Fatal(err)
		}
		roh, err := io.ReadAll(zr)
		if err != nil {
			t.Fatal(err)
		}
		neu := f(string(roh))
		if neu == string(roh) {
			t.Fatalf("SVG unverändert, Muster passt nicht:\n%.600s", roh)
		}
		var buf bytes.Buffer
		zw := zlib.NewWriter(&buf)
		zw.Write([]byte(neu))
		zw.Close()
		return buf.Bytes()
	})
}

// angefordert misst die Speicheranforderung von f in Bytes.
func angefordert(f func()) uint64 {
	var a, b runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&a)
	f()
	runtime.ReadMemStats(&b)
	return b.TotalAlloc - a.TotalAlloc
}

// Bei einer von Schnapp geschriebenen Datei mit Zuschnitt und Streifen meldet der Index die Größe der Ansicht.
func TestLeseMetadatenGroesseWieVollesLesen(t *testing.T) {
	d := volles(t)
	if d.Groesse() == d.OriginalBild().Bounds().Size() {
		t.Fatal("Testdokument ohne Zuschnitt oder Streifen")
	}
	gut := schreibe(t, d, meta)
	k, err := LeseMetadaten(bytes.NewReader(gut))
	voll, err2 := LeseArbeitsdatei(bytes.NewReader(gut))
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	if g := voll.Dokument.Groesse(); k.Breite != g.X || k.Hoehe != g.Y {
		t.Errorf("Metadaten %dx%d, volles Lesen %v", k.Breite, k.Hoehe, g)
	}
}

// Das sichtbare Bild einer fremd nachbearbeiteten Datei hat eine andere Größe als die Ansicht: Das volle Lesen
// nimmt die Größe des Dokuments, der Index muss dieselbe melden (F-272 c).
func TestLeseMetadatenGroesseUnabhaengigVomSichtbarenBild(t *testing.T) {
	d := volles(t)
	var buf bytes.Buffer
	if err := SchreibeArbeitsdatei(&buf, d, testbild(7, 5), meta); err != nil {
		t.Fatal(err)
	}
	k, err := LeseMetadaten(bytes.NewReader(buf.Bytes()))
	voll, err2 := LeseArbeitsdatei(bytes.NewReader(buf.Bytes()))
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	if g := voll.Dokument.Groesse(); k.Breite != g.X || k.Hoehe != g.Y {
		t.Errorf("Index-Größe %dx%d, volles Lesen %v", k.Breite, k.Hoehe, g)
	}
}

// Was das volle Lesen ohne Pixel ablehnt (Zuschnitt, Streifen, Objekte), lehnt LeseMetadaten ebenfalls ab (F-272 a).
func TestLeseMetadatenSVGInhaltWieVollesLesen(t *testing.T) {
	gut := schreibe(t, volles(t), meta)
	zuschnitt := regexp.MustCompile(`zuschnitt="[^"]*"`)
	streifen := regexp.MustCompile(`von="\d+"`)
	for name, kaputt := range map[string][]byte{
		"Zuschnitt außerhalb": ersetzeSVG(t, gut, func(s string) string { return zuschnitt.ReplaceAllString(s, `zuschnitt="0 0 9999 9999"`) }),
		"Zuschnitt leer":      ersetzeSVG(t, gut, func(s string) string { return zuschnitt.ReplaceAllString(s, `zuschnitt="5 5 5 5"`) }),
		"Streifen ungültig":   ersetzeSVG(t, gut, func(s string) string { return streifen.ReplaceAllString(s, `von="-3"`) }),
	} {
		_, errVoll := LeseArbeitsdatei(bytes.NewReader(kaputt))
		_, errMeta := LeseMetadaten(bytes.NewReader(kaputt))
		if errVoll == nil {
			t.Errorf("%s: volles Lesen müsste ablehnen (Testaufbau)", name)
		}
		if errVoll != nil && (errMeta == nil || IstBeschaedigt(errVoll) != IstBeschaedigt(errMeta)) {
			t.Errorf("%s: volles Lesen: %v, Metadaten: %v", name, errVoll, errMeta)
		}
	}
}

// snAO ohne PNG-Signatur und IHDR gilt nicht als passendes Original (F-272 b).
func TestLeseMetadatenOriginalKeinPNG(t *testing.T) {
	d := volles(t)
	gut := schreibe(t, d, meta)
	o := d.OriginalBild().Bounds().Size()
	kaputt := ersetzeBlockMit(t, gut, blockOriginal, func([]byte) []byte {
		m := []byte("KEINPNG!") // statt der Signatur
		m = binary.BigEndian.AppendUint32(m, uint32(o.X))
		m = binary.BigEndian.AppendUint32(m, uint32(o.Y))
		return append(m, "nur Text, kein Bild, kein IHDR"...)
	})
	if _, err := LeseArbeitsdatei(bytes.NewReader(kaputt)); err == nil {
		t.Fatal("volles Lesen müsste ablehnen (Testaufbau)")
	}
	if _, err := LeseMetadaten(bytes.NewReader(kaputt)); !IstBeschaedigt(err) {
		t.Errorf("snAO ohne Signatur und IHDR: %v", err)
	}
	// Richtige Signatur und Größe, aber falsche IHDR-Prüfsumme.
	kaputt = ersetzeBlockMit(t, gut, blockOriginal, func(alt []byte) []byte {
		k := append([]byte{}, alt...)
		k[30] ^= 0xFF
		return k
	})
	if _, err := LeseMetadaten(bytes.NewReader(kaputt)); !IstBeschaedigt(err) {
		t.Errorf("snAO mit falscher IHDR-Prüfsumme: %v", err)
	}
}

// Eine kurze Datei, die einen riesigen Block ankündigt, lässt weder LeseMetadaten noch das volle Lesen (leseBloecke,
// auch LeseBild) den angekündigten Speicher anfordern (F-273, F-59, ADR-0008).
func TestAngekuendigteBlocklaengeReserviertNichts(t *testing.T) {
	block := func(typ string, daten []byte) []byte {
		b := binary.BigEndian.AppendUint32(nil, uint32(len(daten)))
		b = append(b, typ...)
		b = append(b, daten...)
		return binary.BigEndian.AppendUint32(b, crc32.ChecksumIEEE(b[4:]))
	}
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], 4)
	binary.BigEndian.PutUint32(ihdr[4:], 4)
	kopf := func(typ string) []byte {
		return append(binary.BigEndian.AppendUint32(nil, 500<<20), typ...)
	}
	for name, datei := range map[string][]byte{
		"IHDR 500 MB angekündigt": append(append([]byte{}, pngSignatur...), kopf("IHDR")...),
		"snAP 500 MB angekündigt": append(append(append([]byte{}, pngSignatur...), block("IHDR", ihdr)...), kopf(blockSVG)...),
		"snAO 500 MB angekündigt": append(append(append([]byte{}, pngSignatur...), block("IHDR", ihdr)...), kopf(blockOriginal)...),
		"IDAT 500 MB angekündigt": append(append(append([]byte{}, pngSignatur...), block("IHDR", ihdr)...), kopf("IDAT")...),
	} {
		for weg, lies := range map[string]func() error{
			"LeseMetadaten":    func() error { _, err := LeseMetadaten(bytes.NewReader(datei)); return err },
			"LeseArbeitsdatei": func() error { _, err := LeseArbeitsdatei(bytes.NewReader(datei)); return err },
			"LeseBild":         func() error { _, err := LeseBild(bytes.NewReader(datei)); return err },
		} {
			var err error
			n := angefordert(func() { err = lies() })
			t.Logf("%s, %s: %d KB angefordert", name, weg, n>>10)
			if err == nil {
				t.Errorf("%s, %s: kein Fehler", name, weg)
			}
			if n > 16<<20 {
				t.Errorf("%s, %s: %d Bytes Datei lassen %d MB anfordern", name, weg, len(datei), n>>20)
			}
		}
	}
}

// Ein IHDR mit anderer Länge als 13 ist beschädigt.
func TestLeseMetadatenIHDRLaenge(t *testing.T) {
	gut := schreibe(t, volles(t), meta)
	kaputt := ersetzeBlockMit(t, gut, "IHDR", func(alt []byte) []byte { return append(append([]byte{}, alt...), 0) })
	if _, err := LeseMetadaten(bytes.NewReader(kaputt)); !IstBeschaedigt(err) {
		t.Errorf("IHDR mit 14 Bytes: %v", err)
	}
}

// Große gültige Blöcke werden weiter gelesen (Wachsen des Puffers in leseDaten).
func TestLeseDatenGrosserBlock(t *testing.T) {
	roh := bytes.Repeat([]byte{7, 1, 9}, 700_000) // über 1 MB
	got, err := leseDaten(bytes.NewReader(roh), uint32(len(roh)))
	if err != nil || !bytes.Equal(got, roh) {
		t.Fatalf("%v, gleich: %v", err, bytes.Equal(got, roh))
	}
	if _, err := leseDaten(bytes.NewReader(roh[:len(roh)-1]), uint32(len(roh))); err == nil {
		t.Error("zu kurzer Strom: kein Fehler")
	}
}
