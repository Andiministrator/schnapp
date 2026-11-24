// SPDX-License-Identifier: GPL-3.0-or-later

// schnapp-ocr erkennt Text in einem Bild ohne installiertes tesseract (ADR-0035 Nachtrag 1, Karte #242): tesseract
// läuft als WebAssembly (gogosseract, wazero, ohne cgo), die Sprachdaten deu und eng aus tessdata_fast sind
// eingebaut. Die Schnittstelle ist die Teilmenge von tesseract, die Schnapp benutzt (internal/ocr):
//
//	schnapp-ocr --list-langs
//	schnapp-ocr - - -l deu [--psm 3] [--progress]
//
// Das Bild kommt als PNG (oder JPEG) über stdin, der Text geht über stdout. Mit --progress meldet das Programm den
// Fortschritt auf stderr als Zeilen „progress N“ (N von 0 bis 100). Je Aufruf genau eine Sprache: Die WASM-Fassung
// lädt Sprachdaten aus dem Speicher, und das kann tesseract nur für ein Modell.
package main

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"

	"github.com/danlock/gogosseract"
)

// version setzt install.sh per -ldflags.
var version = "0.0.0-dev"

//go:embed tessdata/deu.traineddata tessdata/eng.traineddata
var tessdata embed.FS

// sprachen sind die eingebauten Sprachen in der Reihenfolge von --list-langs.
var sprachen = []string{"deu", "eng"}

// maxEingabe begrenzt das gelesene Bild (internal/ocr schickt höchstens 40 Mpx als PNG).
const maxEingabe = 512 << 20

// auftrag ist ein gelesener Aufruf.
type auftrag struct {
	liste       bool
	ver         bool
	sprache     string
	psm         int
	fortschritt bool
}

var errAufruf = errors.New("usage: schnapp-ocr --list-langs | schnapp-ocr - - -l LANG [--psm N] [--progress]")

// leseArgumente versteht genau die Aufrufe oben; alles andere ist ein Fehler (kein stilles Übergehen).
func leseArgumente(args []string) (auftrag, error) {
	a := auftrag{psm: 3}
	if len(args) == 1 && args[0] == "--list-langs" {
		a.liste = true
		return a, nil
	}
	if len(args) == 1 && args[0] == "--version" {
		a.ver = true
		return a, nil
	}
	if len(args) < 2 || args[0] != "-" || args[1] != "-" {
		return a, errAufruf
	}
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "-l", "--psm":
			if i+1 >= len(args) {
				return a, errAufruf
			}
			wert := args[i+1]
			if args[i] == "-l" {
				a.sprache = wert
			} else {
				n, err := strconv.Atoi(wert)
				if err != nil || n < 0 || n > 13 {
					return a, fmt.Errorf("invalid page segmentation mode %q", wert)
				}
				a.psm = n
			}
			i++
		case "--progress":
			a.fortschritt = true
		default:
			return a, errAufruf
		}
	}
	if a.sprache == "" {
		a.sprache = "eng"
	}
	if !slices.Contains(sprachen, a.sprache) {
		return a, fmt.Errorf("language %q is not built in (available: deu, eng)", a.sprache)
	}
	return a, nil
}

func main() {
	a, err := leseArgumente(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "schnapp-ocr:", err)
		os.Exit(2)
	}
	switch {
	case a.liste:
		fmt.Printf("List of available languages in \"schnapp-ocr\" (%d):\n", len(sprachen))
		for _, s := range sprachen {
			fmt.Println(s)
		}
		return
	case a.ver:
		fmt.Println("schnapp-ocr", version)
		return
	}
	// Abbruch: SIGTERM/SIGINT beenden den Prozess (Voreinstellung von Go). Ein Kontext erreichte die laufende
	// Erkennung nicht, gogosseract startet wazero ohne CloseOnContextDone; internal/ocr beendet den Prozess.
	ctx := context.Background()
	bild, err := io.ReadAll(io.LimitReader(os.Stdin, maxEingabe+1))
	if err == nil && len(bild) > maxEingabe {
		err = errors.New("the image is too large")
	}
	if err == nil {
		var text string
		text, err = erkenne(ctx, a, bild, os.Stderr)
		if err == nil {
			_, err = io.WriteString(os.Stdout, text)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "schnapp-ocr:", err)
		os.Exit(1)
	}
}

// erkenne startet tesseract in WebAssembly mit der Sprache a.sprache, lädt das Bild und liefert den Text. Mit
// a.fortschritt gehen Zeilen „progress N“ an fehler.
func erkenne(ctx context.Context, a auftrag, bild []byte, fehler io.Writer) (string, error) {
	if len(bild) == 0 {
		return "", errors.New("no image on standard input")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	daten, err := tessdata.ReadFile("tessdata/" + a.sprache + ".traineddata")
	if err != nil {
		return "", err
	}
	cfg := gogosseract.Config{
		Language:     a.sprache,
		TrainingData: bytes.NewReader(daten),
		Variables:    map[string]string{"tessedit_pageseg_mode": strconv.Itoa(a.psm)},
	}
	// tesseract schreibt Hinweise (etwa „Estimating resolution“) auf seine Ausgaben; die gehören nicht in den Text.
	cfg.Stdout, cfg.Stderr = io.Discard, io.Discard
	t, err := gogosseract.New(ctx, cfg)
	if err != nil {
		return "", fmt.Errorf("start: %w", err)
	}
	defer t.Close(context.WithoutCancel(ctx))
	if err := t.LoadImage(ctx, bytes.NewReader(bild), gogosseract.LoadImageOptions{}); err != nil {
		return "", fmt.Errorf("load image: %w", err)
	}
	letzter := int32(-1)
	melde := func(p int32) {
		if a.fortschritt && p != letzter && p >= 0 && p <= 100 {
			letzter = p
			fmt.Fprintf(fehler, "progress %d\n", p)
		}
	}
	text, err := t.GetText(ctx, melde)
	if err != nil {
		return "", fmt.Errorf("recognize: %w", err)
	}
	return text, nil
}
