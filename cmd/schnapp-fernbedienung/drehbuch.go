// SPDX-License-Identifier: GPL-3.0-or-later

//go:build probe

package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Befehl ist eine Zeile des Drehbuchs, gelesen und geprüft.
type Befehl struct {
	Zeile int    // Zeilennummer im Drehbuch (ab 1)
	Text  string // die Zeile, wie sie im Protokoll erscheint
	Art   string // kleingeschrieben: warte, zeiger, klick, …
	Zahl  []int  // Zahlenargumente
	Kombi Kombi  // bei „taste“
	Rest  string // bei „schreibe“ der Text, bei „foto“ der Name
	Extra string // bei „foto“ der optionale Bildschirmname
}

// Befehle des Drehbuchs: Name -> (kleinste, größte Zahl der Zahlenargumente).
var zahlenBefehle = map[string][2]int{
	"warte":       {1, 1},
	"zeiger":      {2, 2},
	"klick":       {0, 2},
	"rechtsklick": {0, 2},
	"mittelklick": {0, 2},
	"doppelklick": {2, 2},
	"ziehe":       {4, 5},
	"ziehe-mitte": {4, 5},
	"rad":         {3, 3},
}

var fotoName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// ParseZeile liest eine Zeile. Leere Zeilen und Kommentare (# …) liefern nil ohne Fehler.
func ParseZeile(nr int, zeile string) (*Befehl, error) {
	z := strings.TrimSpace(zeile)
	if z == "" || strings.HasPrefix(z, "#") {
		return nil, nil
	}
	name, rest, _ := strings.Cut(z, " ")
	name = strings.ToLower(name)
	rest = strings.TrimSpace(rest)
	b := &Befehl{Zeile: nr, Text: z, Art: name}
	switch name {
	case "taste":
		k, err := ParseKombi(rest)
		if err != nil {
			return nil, err
		}
		b.Kombi = k
		return b, nil
	case "schreibe":
		// Rest der Zeile, nur die Leerzeichen am Anfang gehören dem Befehl, nicht dem Text.
		_, text, _ := strings.Cut(strings.TrimLeft(zeile, " \t"), " ")
		b.Rest = strings.TrimLeft(strings.TrimRight(text, "\r\n"), " ")
		if b.Rest == "" {
			return nil, fmt.Errorf("schreibe braucht einen Text")
		}
		return b, nil
	case "foto":
		f := strings.Fields(rest)
		if len(f) < 1 || len(f) > 2 {
			return nil, fmt.Errorf("foto braucht NAME [BILDSCHIRM]")
		}
		if !fotoName.MatchString(f[0]) || strings.Contains(f[0], "..") {
			return nil, fmt.Errorf("ungültiger Fotoname %q (erlaubt: Buchstaben, Ziffern, . _ -)", f[0])
		}
		b.Rest = f[0]
		if len(f) == 2 {
			b.Extra = f[1]
		}
		return b, nil
	}
	grenzen, ok := zahlenBefehle[name]
	if !ok {
		return nil, fmt.Errorf("unbekannter Befehl %q", name)
	}
	for _, f := range strings.Fields(rest) {
		n, err := strconv.Atoi(f)
		if err != nil {
			return nil, fmt.Errorf("%s: %q ist keine ganze Zahl", name, f)
		}
		b.Zahl = append(b.Zahl, n)
	}
	if len(b.Zahl) < grenzen[0] || len(b.Zahl) > grenzen[1] {
		return nil, fmt.Errorf("%s: %d Zahlen angegeben, erwartet %d bis %d", name, len(b.Zahl), grenzen[0], grenzen[1])
	}
	switch {
	case (name == "klick" || name == "rechtsklick" || name == "mittelklick") && len(b.Zahl) == 1:
		return nil, fmt.Errorf("%s: entweder keine Koordinaten oder X und Y", name)
	case name == "warte" && b.Zahl[0] < 0:
		return nil, fmt.Errorf("warte: negative Zeit")
	case (name == "ziehe" || name == "ziehe-mitte") && len(b.Zahl) == 5 && b.Zahl[4] < 1:
		return nil, fmt.Errorf("%s: SCHRITTE muss mindestens 1 sein", name)
	}
	return b, nil
}

// ParseDrehbuch liest alle Zeilen; beim ersten Fehler kommt die Zeilennummer in den Fehler.
func ParseDrehbuch(text string) ([]*Befehl, error) {
	var res []*Befehl
	for i, zeile := range strings.Split(text, "\n") {
		b, err := ParseZeile(i+1, strings.TrimRight(zeile, "\r"))
		if err != nil {
			return nil, fmt.Errorf("Zeile %d: %w", i+1, err)
		}
		if b != nil {
			res = append(res, b)
		}
	}
	return res, nil
}
