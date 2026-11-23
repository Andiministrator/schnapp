// SPDX-License-Identifier: GPL-3.0-or-later

//go:build probe

package main

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// aufzeichner ist eine Attrappe der Eingabe: sie merkt sich die Ereignisse als Text.
type aufzeichner struct{ ereignisse []string }

func (r *aufzeichner) Zeiger(x, y int) error {
	r.ereignisse = append(r.ereignisse, fmt.Sprintf("zeiger %d %d", x, y))
	return nil
}
func (r *aufzeichner) Maus(knopf, zustand uint32) error {
	r.ereignisse = append(r.ereignisse, fmt.Sprintf("maus %#x %d", knopf, zustand))
	return nil
}
func (r *aufzeichner) Rad(w float64) error { return nil }
func (r *aufzeichner) Tastatur(code, zustand uint32) error {
	r.ereignisse = append(r.ereignisse, fmt.Sprintf("taste %d %d", code, zustand))
	return nil
}
func (r *aufzeichner) Keysym(sym, zustand uint32) error { return nil }
func (r *aufzeichner) Fassung() uint32                  { return 6 }

// spiele führt ein Drehbuch gegen die Attrappe aus (ohne Foto) und liefert die Ereignisse und den ersten Fehler.
func spiele(t *testing.T, text string) ([]string, *abspieler, error) {
	t.Helper()
	befehle, err := ParseDrehbuch(text)
	if err != nil {
		t.Fatal(err)
	}
	r := &aufzeichner{}
	a := &abspieler{e: r}
	for _, b := range befehle {
		if err := a.fuehreAus(b); err != nil {
			return r.ereignisse, a, err
		}
	}
	return r.ereignisse, a, nil
}

func TestParseHalten(t *testing.T) {
	for _, f := range []struct {
		zeile string
		art   string
		zahl  []int
	}{
		{"druecke", "druecke", nil},
		{"druecke 10 20", "druecke", []int{10, 20}},
		{"druecke-rechts 1 2", "druecke-rechts", []int{1, 2}},
		{"druecke-mitte", "druecke-mitte", nil},
		{"bewege 5 6", "bewege", []int{5, 6}},
		{"bewege 5 6 20", "bewege", []int{5, 6, 20}},
		{"loslassen", "loslassen", nil},
		{"taste-halten Alt", "taste-halten", nil},
		{"taste-loslassen Strg+Alt", "taste-loslassen", nil},
	} {
		b, err := ParseZeile(1, f.zeile)
		if err != nil || b == nil || b.Art != f.art || !reflect.DeepEqual(b.Zahl, f.zahl) {
			t.Errorf("%q: %+v %v", f.zeile, b, err)
		}
	}
	for _, z := range []string{"druecke 1", "bewege 1", "bewege 1 2 0", "loslassen 1", "taste-halten", "taste-halten Foo", "taste-loslassen Strg+"} {
		if b, err := ParseZeile(1, z); err == nil {
			t.Errorf("%q: kein Fehler (%+v)", z, b)
		}
	}
}

// TestZugHalten: druecke, bewege, loslassen ergeben Drücken, Bewegen in Schritten, Lösen — in dieser Reihenfolge.
func TestZugHalten(t *testing.T) {
	ev, a, err := spiele(t, "zeiger 0 0\ndruecke 10 10\nbewege 30 10 2\nloslassen\n")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"zeiger 0 0", "zeiger 10 10", "maus 0x110 1", "zeiger 20 10", "zeiger 30 10", "maus 0x110 0"}
	if !reflect.DeepEqual(ev, want) {
		t.Errorf("Ereignisse %q, erwartet %q", ev, want)
	}
	if len(a.knoepfe) != 0 {
		t.Errorf("noch gehalten: %v", a.knoepfe)
	}
}

// TestTasteHalten: Modifikator zuerst drücken, zuletzt lösen; Fehler bei doppeltem Halten und Lösen ohne Halten.
func TestTasteHalten(t *testing.T) {
	ev, _, err := spiele(t, "taste-halten Alt\ntaste-halten Strg+B\ntaste-loslassen Strg+B\ntaste-loslassen Alt\n")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"taste 56 1", "taste 29 1", "taste 48 1", "taste 48 0", "taste 29 0", "taste 56 0"}
	if !reflect.DeepEqual(ev, want) {
		t.Errorf("Ereignisse %q, erwartet %q", ev, want)
	}
	if _, _, err := spiele(t, "taste-halten Alt\ntaste-halten Alt\n"); err == nil {
		t.Error("doppeltes Halten angenommen")
	}
	if _, _, err := spiele(t, "taste-loslassen Alt\n"); err == nil {
		t.Error("Lösen ohne Halten angenommen")
	}
	if _, _, err := spiele(t, "loslassen\n"); err == nil {
		t.Error("loslassen ohne Maustaste angenommen")
	}
	if _, _, err := spiele(t, "druecke\ndruecke\n"); err == nil {
		t.Error("doppeltes Drücken angenommen")
	}
}

// TestAlleLoslassen: was am Ende des Drehbuchs gehalten wird, löst alleLoslassen — Maus zuerst, Tasten umgekehrt.
func TestAlleLoslassen(t *testing.T) {
	befehle, err := ParseDrehbuch("taste-halten Alt\ntaste-halten Umschalt\ndruecke 1 2\n")
	if err != nil {
		t.Fatal(err)
	}
	r := &aufzeichner{}
	a := &abspieler{e: r}
	for _, b := range befehle {
		if err := a.fuehreAus(b); err != nil {
			t.Fatal(err)
		}
	}
	r.ereignisse = nil
	if err := a.alleLoslassen(); err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(r.ereignisse, ","), "maus 0x110 0,taste 42 0,taste 56 0"; got != want {
		t.Errorf("Ereignisse %q, erwartet %q", got, want)
	}
	r.ereignisse = nil
	if err := a.alleLoslassen(); err != nil || len(r.ereignisse) != 0 {
		t.Errorf("zweites Loslassen sendet %q (%v)", r.ereignisse, err)
	}
}

// TestBewegeOhneLage: ohne bekannte Zeigerlage springt bewege.
func TestBewegeOhneLage(t *testing.T) {
	ev, _, err := spiele(t, "bewege 40 50 5\n")
	if err != nil || !reflect.DeepEqual(ev, []string{"zeiger 40 50"}) {
		t.Errorf("%q %v", ev, err)
	}
}
