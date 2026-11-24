// SPDX-License-Identifier: GPL-3.0-or-later

// schnapp-ocr erkennt Text in einem Bild ohne installiertes tesseract (ADR-0035 Nachtrag 1, Karte #242): tesseract
// läuft als WebAssembly (gogosseract, wazero, ohne cgo), die Sprachdaten deu und eng aus tessdata_fast sind
// eingebaut. Die Schnittstelle ist die Teilmenge von tesseract, die Schnapp benutzt (internal/ocr):
//
//	schnapp-ocr --list-langs
//	schnapp-ocr - - -l deu [--psm 3] [--progress]
//
// Das Bild kommt als PNG oder JPEG über stdin, der Text geht über stdout. Mit --progress meldet das Programm den
// Fortschritt auf stderr als Zeilen „progress N“ (N von 0 bis 100). Je Aufruf genau eine Sprache: Die WASM-Fassung
// lädt Sprachdaten aus dem Speicher, und das kann tesseract nur für ein Modell. Exit 0: erkannt, 1: Fehler beim Bild
// oder bei der Erkennung, 2: falscher Aufruf.
package main

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // Kopf von JPEG für die Pixelgrenze
	_ "image/png"  // Kopf von PNG für die Pixelgrenze
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

// Grenzen des Bilds (F-646): dieselben wie in internal/ocr (MaxPixel, MaxSeite), damit ein direkter Aufruf mit einem
// kleinen, aber riesig angekündigten Bild nicht Gigabyte belegt (WASM bis 4 GiB). Geprüft wird der Kopf, vor dem
// Laden. maxEingabe begrenzt die gelesenen Bytes: ein PNG von 40 Mio. Pixeln mit BestSpeed aus internal/ocr ist
// höchstens rund 160 MB groß (4 Byte je Pixel, kaum komprimiert).
const (
	maxPixel   = 40_000_000
	maxSeite   = 32767
	maxEingabe = 192 << 20
)

// Kurze eigene Fehlertexte (F-649): Die Fehler von gogosseract tragen Go-Interna („gogosseract.(*Tesseract)…“,
// „%!w(<nil>)“), die als letzte Zeile bis in die Fußzeile von Schnapp gingen.
var (
	errAufruf       = errors.New("usage: schnapp-ocr --list-langs | schnapp-ocr - - -l LANG [--psm N] [--progress]")
	errKeinBild     = errors.New("no image on standard input")
	errZuGrossDatei = errors.New("the image is too large")
	errZuGross      = errors.New("the image is too large for text recognition (at most 40 million pixels, 32767 per side)")
	errUnlesbar     = errors.New("cannot read the image (PNG or JPEG expected)")
	errStart        = errors.New("cannot start text recognition")
	errErkennung    = errors.New("text recognition failed")
)

// auftrag ist ein gelesener Aufruf.
type auftrag struct {
	liste       bool
	ver         bool
	sprache     string
	psm         int
	fortschritt bool
}

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
	os.Exit(lauf(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// lauf ist das Programm ohne Prozess: liest args und das Bild aus ein, schreibt Text nach aus und Meldungen nach
// fehler, liefert den Exit-Code.
func lauf(args []string, ein io.Reader, aus, fehler io.Writer) int {
	a, err := leseArgumente(args)
	if err != nil {
		fmt.Fprintln(fehler, "schnapp-ocr:", err)
		return 2
	}
	switch {
	case a.liste:
		fmt.Fprintf(aus, "List of available languages in \"schnapp-ocr\" (%d):\n", len(sprachen))
		for _, s := range sprachen {
			fmt.Fprintln(aus, s)
		}
		return 0
	case a.ver:
		fmt.Fprintln(aus, "schnapp-ocr", version)
		return 0
	}
	// Abbruch: SIGTERM/SIGINT beenden den Prozess (Voreinstellung von Go). Ein Kontext erreichte die laufende
	// Erkennung nicht, gogosseract startet wazero ohne CloseOnContextDone; internal/ocr beendet den Prozess.
	bild, err := leseBild(ein, maxEingabe)
	if err == nil {
		err = pruefeBild(bild)
	}
	if err == nil {
		var text string
		text, err = erkenne(context.Background(), a, bild, fehler)
		if err == nil {
			_, err = io.WriteString(aus, text)
		}
	}
	if err != nil {
		fmt.Fprintln(fehler, "schnapp-ocr:", err)
		return 1
	}
	return 0
}

// leseBild liest höchstens grenze Bytes; ein größeres Bild ist ein Fehler.
func leseBild(ein io.Reader, grenze int64) ([]byte, error) {
	bild, err := io.ReadAll(io.LimitReader(ein, grenze+1))
	if err != nil {
		return nil, errUnlesbar
	}
	if int64(len(bild)) > grenze {
		return nil, errZuGrossDatei
	}
	return bild, nil
}

// pruefeBild liest nur den Kopf (PNG oder JPEG) und lehnt ein Bild über den Grenzen ab, bevor tesseract es lädt.
func pruefeBild(bild []byte) error {
	if len(bild) == 0 {
		return errKeinBild
	}
	k, _, err := image.DecodeConfig(bytes.NewReader(bild))
	if err != nil || k.Width <= 0 || k.Height <= 0 {
		return errUnlesbar
	}
	if k.Width > maxSeite || k.Height > maxSeite || int64(k.Width)*int64(k.Height) > maxPixel {
		return errZuGross
	}
	return nil
}

// konfig ist die Einstellung von tesseract für den Auftrag a mit den Sprachdaten daten. Die Ausgaben von tesseract
// selbst (etwa „Estimating resolution“) werden verworfen: Sie gehören weder in den Text noch in die Meldung.
func konfig(a auftrag, daten []byte) gogosseract.Config {
	cfg := gogosseract.Config{
		Language:     a.sprache,
		TrainingData: bytes.NewReader(daten),
		Variables:    map[string]string{"tessedit_pageseg_mode": strconv.Itoa(a.psm)},
	}
	cfg.Stdout, cfg.Stderr = io.Discard, io.Discard
	return cfg
}

// erkenne startet tesseract in WebAssembly mit der Sprache a.sprache, lädt das Bild und liefert den Text. Mit
// a.fortschritt gehen Zeilen „progress N“ an fehler.
func erkenne(ctx context.Context, a auftrag, bild []byte, fehler io.Writer) (string, error) {
	if len(bild) == 0 {
		return "", errKeinBild
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	daten, err := tessdata.ReadFile("tessdata/" + a.sprache + ".traineddata")
	if err != nil {
		return "", errStart
	}
	t, err := gogosseract.New(ctx, konfig(a, daten))
	if err != nil {
		return "", errStart
	}
	defer t.Close(context.WithoutCancel(ctx))
	if err := t.LoadImage(ctx, bytes.NewReader(bild), gogosseract.LoadImageOptions{}); err != nil {
		return "", errUnlesbar
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
		return "", errErkennung
	}
	return text, nil
}
