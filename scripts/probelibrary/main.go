// SPDX-License-Identifier: GPL-3.0-or-later

// Command probelibrary legt die Probe-Bibliothek für die Sichtprobe der Leiste der letzten Aufnahmen an
// (scripts/drehbuecher/bibliothek-174-leiste.txt, ADR-0036, F-659): 14 synthetische Aufnahmen 1200 × 800 über heute,
// gestern, die letzten Wochen und das Vorjahr, dazu eine Agenten-Aufnahme (magenta), die in der Leiste nicht erscheinen
// darf. Die Orte sind die der Probe-Umgebung von scripts/probe.sh unter HEIM (Bilder/schnapp, .local/share/schnapp,
// .cache/schnapp/vorschau). Am Ende steht der Pfad der zweitneuesten Aufnahme auf stdout (für --bild).
//
//	go run ./scripts/probelibrary -heim tmp/probe/home
//
// Nur für Probeläufe; schreibt nie in das echte Heim (HOME wird abgelehnt) und legt nur neue Dateien an.
package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/library"
)

func main() {
	heim := flag.String("heim", filepath.Join("tmp", "probe", "home"), "Heim der Probe-Umgebung")
	flag.Parse()
	pfad, err := lege(*heim, time.Now())
	if err != nil {
		fmt.Fprintln(os.Stderr, "probelibrary:", err)
		os.Exit(1)
	}
	fmt.Println(pfad)
}

// lege legt die Aufnahmen unter heim an und liefert den absoluten Pfad der zweitneuesten (ohne Agenten-Aufnahme).
func lege(heim string, jetzt time.Time) (string, error) {
	abs, err := filepath.Abs(heim)
	if err != nil {
		return "", err
	}
	if echt := os.Getenv("HOME"); echt != "" && filepath.Clean(echt) == abs {
		return "", errors.New("das echte Heim ist kein Ort für die Probe-Bibliothek")
	}
	b := library.Neu(library.Orte{
		Ablage:     filepath.Join(abs, "Bilder", "schnapp"),
		Index:      filepath.Join(abs, ".local", "share", "schnapp", "bibliothek.jsonl"),
		Vorschau:   filepath.Join(abs, ".cache", "schnapp", "vorschau"),
		Papierkorb: filepath.Join(abs, ".local", "share", "Trash"),
	})
	b.Zone = time.Local
	tag := func(t time.Time, tage int, stunde int) time.Time {
		t = t.AddDate(0, 0, tage)
		return time.Date(t.Year(), t.Month(), t.Day(), stunde, 17, 0, 0, time.Local)
	}
	zeiten := []time.Time{
		jetzt.Add(-10 * time.Minute), jetzt.Add(-40 * time.Minute), jetzt.Add(-70 * time.Minute), // heute
		tag(jetzt, -1, 16), tag(jetzt, -1, 9), // gestern
		tag(jetzt, -3, 14), tag(jetzt, -5, 11), tag(jetzt, -9, 10), tag(jetzt, -14, 15), tag(jetzt, -21, 8), tag(jetzt, -30, 13),
		jetzt.AddDate(-1, 0, 0), jetzt.AddDate(-1, -1, 0), jetzt.AddDate(-1, -2, 0), // Vorjahr
	}
	var pfade []string
	for i, z := range zeiten {
		p, err := ablegen(b, bild(i, farbe(i)), z, false)
		if err != nil {
			return "", err
		}
		pfade = append(pfade, p)
	}
	// Agenten-Aufnahme, die neueste: darf in der Leiste nicht erscheinen.
	if _, err := ablegen(b, bild(99, color.NRGBA{R: 0xE0, B: 0xE0, A: 0xFF}), jetzt.Add(-time.Minute), true); err != nil {
		return "", err
	}
	return pfade[1], nil
}

// farbe ist die Grundfarbe der Aufnahme i (gedämpft, gut unterscheidbar).
func farbe(i int) color.NRGBA {
	tafel := []color.NRGBA{{0x4A, 0x7C, 0xB8, 0xFF}, {0x5B, 0xA0, 0x5B, 0xFF}, {0xC8, 0x8A, 0x3C, 0xFF},
		{0x8A, 0x5B, 0xA8, 0xFF}, {0x3C, 0xA0, 0xA0, 0xFF}, {0xB8, 0x4A, 0x4A, 0xFF}, {0x70, 0x70, 0x70, 0xFF}}
	return tafel[i%len(tafel)]
}

// bild ist eine synthetische Aufnahme: Grundfläche, ein heller Kasten und i + 1 dunkle Balken (die Nummer).
func bild(i int, c color.NRGBA) *image.NRGBA {
	m := image.NewNRGBA(image.Rect(0, 0, 1200, 800))
	fuelle := func(r image.Rectangle, c color.NRGBA) {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				m.SetNRGBA(x, y, c)
			}
		}
	}
	fuelle(m.Rect, c)
	fuelle(image.Rect(150, 150, 1050, 650), color.NRGBA{0xF0, 0xF0, 0xF0, 0xFF})
	for k := 0; k <= i%15; k++ {
		fuelle(image.Rect(200+k*55, 250, 240+k*55, 550), color.NRGBA{0x20, 0x20, 0x20, 0xFF})
	}
	return m
}

func ablegen(b *library.Bibliothek, m *image.NRGBA, zeit time.Time, agent bool) (string, error) {
	d, err := document.Neu(m)
	if err != nil {
		return "", err
	}
	ab := b.Ablegen
	if agent {
		ab = b.AblegenAgent
	}
	e, err := ab(d, d.Original(), fileformat.Metadaten{Erstellt: zeit})
	if err != nil {
		return "", err
	}
	return filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad)), nil
}
