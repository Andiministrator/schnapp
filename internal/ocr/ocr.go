// SPDX-License-Identifier: GPL-3.0-or-later

// Package ocr erkennt Text in einem Bild über das Programm tesseract, wenn es installiert ist (ADR-0035). Das Bild
// geht als PNG über stdin, der Text kommt über stdout; es entsteht keine temporäre Datei und nichts verlässt den
// Rechner. Das Paket kennt weder Gio noch das Dokument: Der Aufrufer übergibt das sichtbare Bild (mit Schwärzungen),
// nie das Original.
package ocr

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"os/exec"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	xdraw "golang.org/x/image/draw"

	"github.com/Andiministrator/schnapp/internal/errprefix"
)

const (
	// Frist ist die längste Dauer einer Erkennung (ADR-0035 Punkt 2).
	Frist = 30 * time.Second
	// maxAusgabe begrenzt den gelesenen Text, maxFehlerausgabe die gelesene Fehlerausgabe.
	maxAusgabe       = 1 << 20
	maxFehlerausgabe = 64 << 10
	// kleinSeite: Ist die kürzere Seite kleiner, wird das Bild vor der Erkennung verdoppelt (ADR-0035 Punkt 4).
	kleinSeite = 600
	// MaxPixel ist die größte Pixelzahl, die an tesseract geht (nach dem Vergrößern).
	MaxPixel = 40_000_000
	// programmName ist der Name im PATH.
	programmName = "tesseract"
)

// Fehler der Erkennung.
var (
	ErrKeinProgramm = errors.New("tesseract is not installed")
	ErrKeineSprache = errors.New("tesseract has no language data")
	ErrZuGross      = errors.New("the area is too large for text recognition")
	ErrLeer         = errors.New("the area is empty")
	ErrFrist        = errors.New("text recognition took too long")
)

// Erkenner ist ein gefundenes tesseract mit seinen installierten Sprachen.
type Erkenner struct {
	programm string
	sprachen []string // installierte Sprachen ohne osd, in der Reihenfolge von --list-langs
}

// suchePfad ist exec.LookPath, für Tests austauschbar.
var suchePfad = exec.LookPath

// Suche findet tesseract im PATH und fragt seine Sprachen ab. Fehlt das Programm, kommt ErrKeinProgramm; hat es keine
// Sprache außer osd, ErrKeineSprache.
func Suche(ctx context.Context) (*Erkenner, error) {
	p, err := suchePfad(programmName)
	if err != nil {
		return nil, errprefix.Wrap("ocr", ErrKeinProgramm)
	}
	ctx, ende := context.WithTimeout(ctx, 10*time.Second)
	defer ende()
	aus, err := lauf(ctx, p, nil, "--list-langs")
	if err != nil {
		return nil, errprefix.Wrap("ocr", fmt.Errorf("list languages: %w", err))
	}
	e := &Erkenner{programm: p, sprachen: leseSprachen(aus)}
	if len(e.sprachen) == 0 {
		return nil, errprefix.Wrap("ocr", ErrKeineSprache)
	}
	return e, nil
}

// leseSprachen liest die Ausgabe von --list-langs: eine Kopfzeile („List of available languages …“), dann je Zeile ein
// Code. osd (Ausrichtung) ist keine Textsprache. Nur Codes aus Buchstaben, Ziffern, _ und - (kein Argument-Einschleusen).
func leseSprachen(aus string) []string {
	var r []string
	for _, z := range strings.Split(aus, "\n") {
		z = strings.TrimSpace(z)
		if z == "" || z == "osd" || strings.Contains(z, " ") || strings.HasPrefix(z, "-") {
			continue
		}
		gut := true
		for _, c := range z {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
				gut = false
				break
			}
		}
		if gut && !slices.Contains(r, z) {
			r = append(r, z)
		}
	}
	return r
}

// iso639 ordnet Sprachcodes der Oberfläche (ISO 639-1) den Tesseract-Codes (ISO 639-2/T) zu; nur die häufigen.
var iso639 = map[string]string{
	"de": "deu", "en": "eng", "fr": "fra", "es": "spa", "it": "ita", "pt": "por", "nl": "nld", "pl": "pol",
	"cs": "ces", "da": "dan", "sv": "swe", "nb": "nor", "no": "nor", "fi": "fin", "hu": "hun", "ru": "rus",
	"uk": "ukr", "tr": "tur", "el": "ell", "ja": "jpn", "ko": "kor", "zh": "chi_sim",
}

// Sprachen liefert das Argument für -l (ADR-0035 Punkt 3): die Sprache der Oberfläche (etwa „de“ oder „pt_BR“) und
// Englisch, soweit installiert; sonst die erste installierte Sprache.
func (e *Erkenner) Sprachen(oberflaeche string) string {
	var wahl []string
	code, _, _ := strings.Cut(oberflaeche, "_")
	for _, w := range []string{iso639[code], "eng"} {
		if w != "" && slices.Contains(e.sprachen, w) && !slices.Contains(wahl, w) {
			wahl = append(wahl, w)
		}
	}
	if len(wahl) == 0 {
		wahl = e.sprachen[:1]
	}
	return strings.Join(wahl, "+")
}

// Vorbereite vergrößert ein kleines Bild auf das Doppelte und prüft die Größe (ADR-0035 Punkt 4).
func Vorbereite(b image.Image) (image.Image, error) {
	r := b.Bounds()
	if r.Empty() {
		return nil, errprefix.Wrap("ocr", ErrLeer)
	}
	faktor := 1
	if min(r.Dx(), r.Dy()) < kleinSeite {
		faktor = 2
	}
	if int64(r.Dx())*int64(r.Dy())*int64(faktor*faktor) > MaxPixel {
		return nil, errprefix.Wrap("ocr", ErrZuGross)
	}
	if faktor == 1 {
		return b, nil
	}
	gross := image.NewNRGBA(image.Rect(0, 0, r.Dx()*faktor, r.Dy()*faktor))
	xdraw.BiLinear.Scale(gross, gross.Bounds(), b, r, xdraw.Src, nil)
	return gross, nil
}

// Erkenne liefert den Text im Bild b mit den Sprachen sprachen (Argument für -l, aus Sprachen). Der Text ist gültiges
// UTF-8 mit Zeilenenden \n, ohne Seitenvorschub und ohne Leerzeilen am Rand; leer, wenn nichts erkannt wurde.
func (e *Erkenner) Erkenne(ctx context.Context, b image.Image, sprachen string) (string, error) {
	text, err := e.erkenne(ctx, b, sprachen)
	return text, errprefix.Wrap("ocr", err)
}

func (e *Erkenner) erkenne(ctx context.Context, b image.Image, sprachen string) (string, error) {
	for _, s := range strings.Split(sprachen, "+") {
		if !slices.Contains(e.sprachen, s) {
			return "", fmt.Errorf("unknown language %q", s)
		}
	}
	bild, err := Vorbereite(b)
	if err != nil {
		return "", err
	}
	var eingabe bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(&eingabe, bild); err != nil {
		return "", err
	}
	ctx, ende := context.WithTimeout(ctx, Frist)
	defer ende()
	aus, err := lauf(ctx, e.programm, &eingabe, "-", "-", "-l", sprachen, "--psm", "3")
	if err != nil {
		return "", err
	}
	return bereinige(aus), nil
}

// lauf startet das Programm mit festen Argumenten (kein Shell) und liefert stdout; stdout und stderr sind begrenzt.
func lauf(ctx context.Context, programm string, eingabe io.Reader, argumente ...string) (string, error) {
	cmd := exec.CommandContext(ctx, programm, argumente...)
	cmd.Stdin = eingabe
	var aus, fehler grenzPuffer
	aus.grenze, fehler.grenze = maxAusgabe, maxFehlerausgabe
	cmd.Stdout, cmd.Stderr = &aus, &fehler
	cmd.WaitDelay = 2 * time.Second
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return "", ErrFrist
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if err != nil {
		meldung := strings.TrimSpace(strings.ToValidUTF8(fehler.String(), "?"))
		if i := strings.LastIndexByte(meldung, '\n'); i >= 0 {
			meldung = meldung[i+1:] // die letzte Zeile sagt meist, was fehlt
		}
		if meldung != "" {
			return "", fmt.Errorf("%s failed: %w (%s)", programmName, err, meldung)
		}
		return "", fmt.Errorf("%s failed: %w", programmName, err)
	}
	return aus.String(), nil
}

// grenzPuffer nimmt höchstens grenze Bytes auf und verwirft den Rest still (der Kindprozess soll nicht blockieren).
type grenzPuffer struct {
	bytes.Buffer
	grenze int
}

func (g *grenzPuffer) Write(p []byte) (int, error) {
	if frei := g.grenze - g.Len(); frei > 0 {
		g.Buffer.Write(p[:min(len(p), frei)])
	}
	return len(p), nil
}

// bereinige macht aus der Ausgabe von tesseract den Text für die Zwischenablage.
func bereinige(s string) string {
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "�")
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\f", "")
	zeilen := strings.Split(s, "\n")
	for i, z := range zeilen {
		zeilen[i] = strings.TrimRight(z, " \t")
	}
	return strings.Trim(strings.Join(zeilen, "\n"), "\n")
}
