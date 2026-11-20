// SPDX-License-Identifier: GPL-3.0-or-later

//go:build probe

package main

import (
	"fmt"
	"strings"
	"unicode"
)

// Tastentabelle: evdev-Codes (linux/input-event-codes.h) nach US-Belegung. Nur für Probeläufe im virtuellen KWin.

const (
	keyStrg     = 29
	keyUmschalt = 42
	keyAlt      = 56
	keyMeta     = 125
)

// buchstaben: Tastencodes der Buchstaben a bis z.
var buchstaben = map[rune]uint32{
	'q': 16, 'w': 17, 'e': 18, 'r': 19, 't': 20, 'y': 21, 'u': 22, 'i': 23, 'o': 24, 'p': 25,
	'a': 30, 's': 31, 'd': 32, 'f': 33, 'g': 34, 'h': 35, 'j': 36, 'k': 37, 'l': 38,
	'z': 44, 'x': 45, 'c': 46, 'v': 47, 'b': 48, 'n': 49, 'm': 50,
}

// ziffern: Tastencodes von 1 bis 9 und 0 in der Zifferreihe.
var ziffern = map[rune]uint32{'1': 2, '2': 3, '3': 4, '4': 5, '5': 6, '6': 7, '7': 8, '8': 9, '9': 10, '0': 11}

// satzzeichen: Zeichen ohne Umschalten und das Zeichen mit Umschalten auf derselben Taste (US-Belegung).
var satzzeichen = map[rune]struct {
	code    uint32
	shifted rune
}{
	'-': {12, '_'}, '=': {13, '+'}, '[': {26, '{'}, ']': {27, '}'}, ';': {39, ':'}, '\'': {40, '"'},
	'`': {41, '~'}, '\\': {43, '|'}, ',': {51, '<'}, '.': {52, '>'}, '/': {53, '?'},
}

// umgeschaltet: Zeichen der Zifferreihe mit Umschalten.
var umgeschaltet = map[rune]rune{'!': '1', '@': '2', '#': '3', '$': '4', '%': '5', '^': '6', '&': '7', '*': '8', '(': '9', ')': '0'}

// benannt sind die Tasten mit Namen (klein geschrieben, deutsch und englisch).
var benannt = map[string]uint32{
	"esc": 1, "escape": 1, "enter": 28, "return": 28, "tab": 15, "ruecktaste": 14, "rücktaste": 14, "backspace": 14,
	"entf": 111, "del": 111, "delete": 111, "leertaste": 57, "space": 57, "einfg": 110, "insert": 110,
	"pos1": 102, "home": 102, "ende": 107, "end": 107, "bildauf": 104, "pageup": 104, "bildab": 109, "pagedown": 109,
	"hoch": 103, "up": 103, "links": 105, "left": 105, "rechts": 106, "right": 106, "runter": 108, "down": 108,
	"minus": 12, "plus": 78, // Plus: Ziffernblock-Taste (wie Strg+Plus zum Zoomen)
	"f1": 59, "f2": 60, "f3": 61, "f4": 62, "f5": 63, "f6": 64, "f7": 65, "f8": 66, "f9": 67, "f10": 68, "f11": 87, "f12": 88,
}

// modifikatoren sind die Namen der Umschalttasten.
var modifikatoren = map[string]uint32{
	"strg": keyStrg, "ctrl": keyStrg, "control": keyStrg,
	"umschalt": keyUmschalt, "shift": keyUmschalt,
	"alt": keyAlt, "meta": keyMeta, "super": keyMeta,
}

// Kombi ist eine Tastenkombination: erst die Modifikatoren drücken, dann die Taste, dann alles lösen.
type Kombi struct {
	Mods  []uint32
	Taste uint32
}

// ParseKombi liest „Strg+Umschalt+Z“, „B“, „Esc“ usw. Die Groß- und Kleinschreibung der Namen ist gleichgültig.
func ParseKombi(s string) (Kombi, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Kombi{}, fmt.Errorf("leere Tastenkombination")
	}
	teile := strings.Split(s, "+")
	var k Kombi
	for i, t := range teile {
		t = strings.TrimSpace(t)
		if t == "" {
			return Kombi{}, fmt.Errorf("leerer Teil in %q (Plus-Taste heißt „Plus“)", s)
		}
		name := strings.ToLower(t)
		if i < len(teile)-1 {
			m, ok := modifikatoren[name]
			if !ok {
				return Kombi{}, fmt.Errorf("unbekannter Modifikator %q in %q", t, s)
			}
			k.Mods = append(k.Mods, m)
			continue
		}
		code, ok := taste(name)
		if !ok {
			return Kombi{}, fmt.Errorf("unbekannte Taste %q in %q", t, s)
		}
		k.Taste = code
	}
	return k, nil
}

// taste liefert den Code einer Taste nach Namen (klein geschrieben).
func taste(name string) (uint32, bool) {
	if c, ok := benannt[name]; ok {
		return c, true
	}
	if r := []rune(name); len(r) == 1 {
		if c, ok := buchstaben[r[0]]; ok {
			return c, true
		}
		if c, ok := ziffern[r[0]]; ok {
			return c, true
		}
		if s, ok := satzzeichen[r[0]]; ok {
			return s.code, true
		}
	}
	if c, ok := modifikatoren[name]; ok { // eine Umschalttaste allein
		return c, true
	}
	return 0, false
}

// ZeichenZuTaste liefert für ein ASCII-Zeichen Tastencode und ob Umschalt nötig ist (US-Belegung).
func ZeichenZuTaste(r rune) (code uint32, umschalt bool, ok bool) {
	switch {
	case r == ' ':
		return 57, false, true
	case r == '\t':
		return 15, false, true
	case r == '\n':
		return 28, false, true
	case r >= 'a' && r <= 'z':
		return buchstaben[r], false, true
	case r >= 'A' && r <= 'Z':
		return buchstaben[unicode.ToLower(r)], true, true
	case r >= '0' && r <= '9':
		return ziffern[r], false, true
	}
	if base, ok := umgeschaltet[r]; ok {
		return ziffern[base], true, true
	}
	if s, ok := satzzeichen[r]; ok {
		return s.code, false, true
	}
	for _, s := range satzzeichen {
		if s.shifted == r {
			return s.code, true, true
		}
	}
	return 0, false, false
}

// Keysym liefert das X11-Keysym eines Zeichens: Latin-1 unverändert, sonst 0x01000000 + Codepunkt. Steuerzeichen haben
// keinen Unicode-Keysym: Tab, Zeilenende (\n und \r) und Rücktaste (0x08) werden zu den Tasten-Keysyms, alle anderen
// Steuerzeichen (unter 0x20, 0x7f bis 0x9f) sind ein Fehler.
func Keysym(r rune) (uint32, error) {
	switch {
	case r == '\t':
		return 0xff09, nil
	case r == '\n' || r == '\r':
		return 0xff0d, nil
	case r == 0x08:
		return 0xff08, nil
	case r < 0x20 || (r >= 0x7f && r <= 0x9f):
		return 0, fmt.Errorf("Steuerzeichen U+%04X lässt sich nicht schreiben", r)
	case r > 0x10ffff || (r >= 0xd800 && r <= 0xdfff):
		return 0, fmt.Errorf("ungültiges Zeichen U+%04X", r)
	case r <= 0x7e || (r >= 0xa0 && r <= 0xff):
		return uint32(r), nil
	}
	return 0x01000000 + uint32(r), nil
}
