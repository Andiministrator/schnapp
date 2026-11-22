// SPDX-License-Identifier: GPL-3.0-or-later

package main

// Tests des Schriftlesers ohne Systemschrift (laufen auch in der CI; Sitzungs-QA 22.11., QB-3 und QB-4).

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// Die Zeichenlogik wie fontTools Glyph.draw hinter SVGPathPen: Kontur auf einen Punkt auf der Kurve gedreht, H/V,
// implizite Mittelpunkte zwischen zwei Punkten außerhalb der Kurve, Z am Ende; y gespiegelt.
func TestZeichneWieFontTools(t *testing.T) {
	g := &glyphe{konturen: [][]roh{{
		{0, 0, true}, {100, 0, true}, {100, 100, false}, {0, 100, false},
	}}}
	st := &svgStift{}
	if err := st.zeichne(g, 1, 0, 0); err != nil {
		t.Fatal(err)
	}
	soll := "M0.0 0.0H100.0Q100.0 -100.0 50.0 -100.0Q0.0 -100.0 0.0 0.0Z"
	if ist := joinBefehle(st); ist != soll {
		t.Errorf("\nist  %s\nsoll %s", ist, soll)
	}
	// Beginnt die Kontur mit einem Punkt außerhalb der Kurve, wird sie gedreht; schräge Linien nach M ohne „L“ (fontTools
	// merkt sich „M“ als letzten Befehl, bis ein H, V, L oder Q kommt), doppelter Punkt fällt weg.
	g = &glyphe{konturen: [][]roh{{
		{5, 5, false}, {10, 0, true}, {20, 10, true}, {20, 10, true}, {30, 30, true},
	}}}
	st = &svgStift{}
	if err := st.zeichne(g, 0.5, 1, 2); err != nil {
		t.Fatal(err)
	}
	soll = "M6.0 2.0 11.0 -3.0 16.0 -13.0Q3.5 -0.5 6.0 2.0Z" // von fontTools so bestätigt
	if ist := joinBefehle(st); ist != soll {
		t.Errorf("gedreht:\nist  %s\nsoll %s", ist, soll)
	}
	if err := (&svgStift{}).zeichne(&glyphe{konturen: [][]roh{{{1, 1, false}, {2, 2, false}}}}, 1, 0, 0); err == nil {
		t.Error("Kontur ohne Punkt auf der Kurve angenommen")
	}
}

func joinBefehle(st *svgStift) string {
	s := ""
	for _, b := range st.befehle {
		s += b
	}
	return s
}

// cmap Format 4 mit idRangeOffset ≠ 0 (Glyphen aus glyphIdArray) und mit idDelta.
func TestCmapFormat4(t *testing.T) {
	be := binary.BigEndian
	sub := make([]byte, 14)
	be.PutUint16(sub[0:], 4)
	be.PutUint16(sub[6:], 4) // segCountX2: zwei Segmente
	for _, v := range []uint16{0x43, 0xFFFF, 0, 0x41, 0xFFFF, 0, 1, 4, 0, 7, 8, 9} {
		sub = be.AppendUint16(sub, v) // endCode, Pad, startCode, idDelta, idRangeOffset, glyphIdArray
	}
	// Ende ist endCode[2] + Pad: oben steht nach endCode (0x43, 0xFFFF) das Pad 0, dann startCode (0x41, 0xFFFF),
	// idDelta (0, 1), idRangeOffset (4, 0), glyphIdArray (7, 8, 9).
	be.PutUint16(sub[2:], uint16(len(sub)))
	c := make([]byte, 12)
	be.PutUint16(c[2:], 1)
	be.PutUint16(c[4:], 3)
	be.PutUint16(c[6:], 1)
	be.PutUint32(c[8:], 12)
	c = append(c, sub...)
	s := &schrift{tabellen: map[string][]byte{"cmap": c}}
	if err := s.leseCmap(); err != nil {
		t.Fatal(err)
	}
	for z, soll := range map[rune]int{'A': 7, 'B': 8, 'C': 9} {
		if s.zeichen[z] != soll {
			t.Errorf("%q → %d, soll %d", z, s.zeichen[z], soll)
		}
	}
}

// Beschädigte Dateien: Fehler, keine Panik (QB-3).
func TestLadeSchriftKaputt(t *testing.T) {
	dir := t.TempDir()
	be := binary.BigEndian
	kopf := func(tabellen ...string) []byte {
		d := make([]byte, 12+16*len(tabellen))
		be.PutUint16(d[4:], uint16(len(tabellen)))
		for i, tag := range tabellen {
			copy(d[12+16*i:], tag)
			be.PutUint32(d[12+16*i+8:], uint32(len(d))) // jede Tabelle leer, am Ende der Datei
		}
		return d
	}
	faelle := map[string][]byte{
		"leer":             nil,
		"zu kurz":          {0, 1, 0, 0},
		"Verzeichnis lügt": {0, 1, 0, 0, 0, 40, 0, 0, 0, 0, 0, 0},
		"Tabellen leer":    kopf("head", "hhea", "hmtx", "maxp", "cmap", "loca", "glyf"),
		"Tabelle fehlt":    kopf("head"),
	}
	for name, d := range faelle {
		p := filepath.Join(dir, "kaputt.ttf")
		if err := os.WriteFile(p, d, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := ladeSchrift(p); err == nil {
			t.Errorf("%s: angenommen", name)
		}
	}
	// Glyphe außerhalb, kaputte glyf-Daten: Fehler, nichts im Speicher.
	s := &schrift{anzahl: 2, glyphen: map[int]*glyphe{}, vorschub: []int{0, 0}, kurzLoca: true,
		tabellen: map[string][]byte{"loca": {0, 0, 0, 5, 0, 5}, "glyf": {0, 1, 0, 0}}}
	if _, err := s.glyphe(5); err == nil {
		t.Error("Glyphe außerhalb angenommen")
	}
	if _, err := s.glyphe(0); err == nil {
		t.Error("abgeschnittene Glyphe angenommen")
	}
	if len(s.glyphen) != 0 {
		t.Errorf("%d Glyphen im Speicher trotz Fehler", len(s.glyphen))
	}
}
