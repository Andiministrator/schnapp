// SPDX-License-Identifier: GPL-3.0-or-later

//go:build probe

package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseZeile(t *testing.T) {
	for _, f := range []struct {
		zeile string
		art   string
		zahl  []int
		rest  string
	}{
		{"warte 250", "warte", []int{250}, ""},
		{"  Zeiger 10 20  ", "zeiger", []int{10, 20}, ""},
		{"klick", "klick", nil, ""},
		{"klick 5 6", "klick", []int{5, 6}, ""},
		{"doppelklick 5 6", "doppelklick", []int{5, 6}, ""},
		{"rechtsklick 1 2", "rechtsklick", []int{1, 2}, ""},
		{"mittelklick", "mittelklick", nil, ""},
		{"ziehe 1 2 3 4", "ziehe", []int{1, 2, 3, 4}, ""},
		{"ziehe-mitte 1 2 3 4 20", "ziehe-mitte", []int{1, 2, 3, 4, 20}, ""},
		{"rad 100 200 -3", "rad", []int{100, 200, -3}, ""},
		{"foto vor-allem", "foto", nil, "vor-allem"},
		{"schreibe  Hallo  Welt ", "schreibe", nil, "Hallo  Welt "},
		{"taste Strg+Z", "taste", nil, ""},
	} {
		b, err := ParseZeile(7, f.zeile)
		if err != nil || b == nil {
			t.Errorf("%q: %v", f.zeile, err)
			continue
		}
		if b.Art != f.art || !reflect.DeepEqual(b.Zahl, f.zahl) || b.Rest != f.rest || b.Zeile != 7 {
			t.Errorf("%q: %+v", f.zeile, b)
		}
	}
}

func TestParseZeileLeerUndKommentar(t *testing.T) {
	for _, z := range []string{"", "   ", "# nichts", "  # nichts"} {
		if b, err := ParseZeile(1, z); b != nil || err != nil {
			t.Errorf("%q: %v %v", z, b, err)
		}
	}
}

func TestParseZeileFehler(t *testing.T) {
	for _, z := range []string{
		"tanze", "warte", "warte x", "warte -5", "zeiger 1", "klick 1", "ziehe 1 2 3", "ziehe 1 2 3 4 0", "rad 1 2",
		"schreibe", "foto", "foto ../x", "foto a/b", "foto a b c", "taste", "taste Strg+", "taste Foo", "taste Quatsch+A",
	} {
		if b, err := ParseZeile(1, z); err == nil {
			t.Errorf("%q: kein Fehler (%+v)", z, b)
		}
	}
}

func TestParseDrehbuchZeilennummer(t *testing.T) {
	bs, err := ParseDrehbuch("# Kopf\n\nwarte 10\nklick 1 2\r\nquatsch\n")
	if err == nil || !strings.Contains(err.Error(), "Zeile 5") {
		t.Fatalf("Fehler %v, erwartet mit „Zeile 5“", err)
	}
	bs, err = ParseDrehbuch("# Kopf\n\nwarte 10\nklick 1 2\r\n")
	if err != nil || len(bs) != 2 || bs[0].Zeile != 3 || bs[1].Zeile != 4 {
		t.Errorf("%v %+v", err, bs)
	}
}

func TestParseKombi(t *testing.T) {
	for _, f := range []struct {
		s     string
		mods  []uint32
		taste uint32
	}{
		{"B", nil, 48},
		{"b", nil, 48},
		{"Esc", nil, 1},
		{"Strg+Z", []uint32{29}, 44},
		{"strg+umschalt+z", []uint32{29, 42}, 44},
		{"Ctrl+Shift+E", []uint32{29, 42}, 18},
		{"Entf", nil, 111},
		{"Del", nil, 111},
		{"Enter", nil, 28},
		{"Return", nil, 28},
		{"Tab", nil, 15},
		{"Strg+0", []uint32{29}, 11},
		{"Strg+1", []uint32{29}, 2},
		{"Strg+W", []uint32{29}, 17},
		{"Strg+S", []uint32{29}, 31},
		{"Strg+C", []uint32{29}, 46},
		{"Alt+F4", []uint32{56}, 62},
		{"Strg+Plus", []uint32{29}, 78},
		{"Rücktaste", nil, 14},
		{"Leertaste", nil, 57},
	} {
		k, err := ParseKombi(f.s)
		if err != nil || !reflect.DeepEqual(k.Mods, f.mods) || k.Taste != f.taste {
			t.Errorf("%q: %+v, %v", f.s, k, err)
		}
	}
	for _, s := range []string{"", "+", "Strg+", "Foo+Z", "Strg+Quatsch", "ß"} {
		if k, err := ParseKombi(s); err == nil {
			t.Errorf("%q: kein Fehler (%+v)", s, k)
		}
	}
}

func TestZeichenZuTaste(t *testing.T) {
	for _, f := range []struct {
		r        rune
		code     uint32
		umschalt bool
	}{
		{'a', 30, false}, {'A', 30, true}, {'z', 44, false}, {'1', 2, false}, {'0', 11, false}, {'!', 2, true},
		{' ', 57, false}, {'-', 12, false}, {'_', 12, true}, {'?', 53, true}, {'/', 53, false}, {'"', 40, true},
		{'(', 10, true}, {')', 11, true}, {'\n', 28, false},
	} {
		c, u, ok := ZeichenZuTaste(f.r)
		if !ok || c != f.code || u != f.umschalt {
			t.Errorf("%q: %d %v %v", f.r, c, u, ok)
		}
	}
	if _, _, ok := ZeichenZuTaste('ä'); ok {
		t.Error("ä ist auf der US-Belegung keine Taste")
	}
}

// TestJedesAsciiZeichenHatEineTaste: alle druckbaren ASCII-Zeichen sind über die Tabelle schreibbar.
func TestJedesAsciiZeichenHatEineTaste(t *testing.T) {
	for r := rune(0x20); r <= 0x7e; r++ {
		if _, _, ok := ZeichenZuTaste(r); !ok {
			t.Errorf("%q fehlt", r)
		}
	}
}

func TestKeysym(t *testing.T) {
	for _, f := range []struct {
		r    rune
		want uint32
		fehl bool
	}{
		{'A', 0x41, false}, {'ä', 0xe4, false}, {'€', 0x010020ac, false}, {' ', 0x20, false}, {'~', 0x7e, false},
		{'\t', 0xff09, false}, {'\n', 0xff0d, false}, {'\r', 0xff0d, false}, {0x08, 0xff08, false},
		{0x00, 0, true}, {0x1b, 0, true}, {0x7f, 0, true}, {0x85, 0, true}, {0x9f, 0, true},
		{0xa0, 0xa0, false}, {0xd800, 0, true}, {0x110000, 0, true},
	} {
		got, err := Keysym(f.r)
		if (err != nil) != f.fehl || got != f.want {
			t.Errorf("Keysym(%U) = %#x, %v; erwartet %#x, Fehler %v", f.r, got, err, f.want, f.fehl)
		}
	}
}
