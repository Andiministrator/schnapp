// SPDX-License-Identifier: GPL-3.0-or-later

package settings

// Eigene Schnellstile (ADR-0037 Punkt 6): `styles.ini` im Konfigurationsordner, neben `settings.ini`. Anders als diese
// gehört die Datei dem Programm: Schnapp schreibt sie jedes Mal ganz neu, aber aus dem frisch gelesenen Stand unter einer Sperre (F-627; Kopfkommentar sagt das), der Leser folgt
// aber denselben Regeln wie bei `settings.ini` (höchstens 64 KiB, Warnungen statt Abbruch, ohne Blockieren öffnen).
//
// Format: ein Abschnitt je Stil, Name = Objektart (`[arrow]`, `[box]`, `[text]`, `[callout]`, `[step]`,
// `[magnifier]`), Reihenfolge der Abschnitte = Reihenfolge der Kacheln. Schlüssel englisch: `color`, `fill`, `width`,
// `font`, `size`, `bold`, `italic`, `underline`, `strike`, `align`, `radius`; Farben `#RRGGBBAA` (beim Lesen auch
// `#RRGGBB`). Größen stehen auf Faktor 1 (ADR-0037 Punkt 1). Ein Stil mit unbekanntem oder doppeltem Schlüssel,
// ungültigem Wert oder ohne `color` fällt mit einer Warnung weg; die übrigen bleiben.
//
// Das Paket kennt `canvas` nicht (sonst ein Zyklus): Der Datentyp Stil ist neutral, `canvas` wandelt um.

import (
	"errors"
	"fmt"
	"image/color"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/errprefix"
	"github.com/Andiministrator/schnapp/internal/fonts"
	"github.com/Andiministrator/schnapp/internal/i18n"
)

// Namen der Objektarten in styles.ini (Abschnittsnamen).
const (
	ArtPfeil       = "arrow"
	ArtKasten      = "box"
	ArtText        = "text"
	ArtSprechblase = "callout"
	ArtNummer      = "step"
	ArtLupe        = "magnifier"
	ArtEllipse     = "ellipse"     // ADR-0047 Punkte 4 und 5
	ArtTextmarker  = "highlighter" // ADR-0048 Punkte 5 und 6
	ArtFreihand    = "freehand"    // ADR-0049 Punkt 4
)

// MaxEigeneStile ist die Grenze eigener Stile je Objektart (ADR-0037 Punkt 5).
const MaxEigeneStile = 16

// Werte, die gelten, wenn ein Schlüssel fehlt (Faktor 1, wie die Voreinstellung in canvas.StandardStil).
const (
	stilBreite      = 6.0
	stilSchriftgrad = 20.0
	stilRadius      = 14.0
	// textmarkerBreite ist canvas.TextmarkerBreite (ADR-0048 Punkt 5).
	textmarkerBreite = 20.0
)

// Fehler des Schreibens (der Leser meldet Warnungen statt Fehler).
var (
	ErrStileZuGross    = errors.New("the styles file would be larger than 64 KiB")
	ErrStileKeineDatei = errors.New("the styles file is not a regular file")
	ErrStileVerweis    = errors.New("the styles file is a link to nothing")
	ErrStileKeinUTF8   = errors.New("a style contains text that is not valid UTF-8")
	// Zusammenführen (F-627, F-629 b): Die vorhandene Datei wird nie stillschweigend ersetzt.
	ErrStileUnlesbar  = errors.New("the existing styles file cannot be read, so it is left as it is")
	ErrStileSicherung = errors.New("the existing styles file has parts that cannot be read and styles.ini.bak already exists, so it is left as it is")
	ErrStileGesperrt  = errors.New("the styles file is locked by another window")
)

// Stil ist ein Schnellstil, wie er in styles.ini steht. Nicht alle Felder gelten für jede Art (schluesselFuer).
type Stil struct {
	Art         string // einer der Namen Art…
	Farbe       color.NRGBA
	Fuellung    color.NRGBA // Kasten, Text, Sprechblase; Deckkraft 0 = keine
	Breite      float64     // Strich von Pfeil, Kasten, Lupe (Faktor 1)
	Schriftgrad float64     // Text, Sprechblase (Faktor 1)
	Radius      float64     // Nummer (Faktor 1)
	Schrift     string      // leer = Go
	Fett        bool
	Kursiv      bool
	Unterstr    bool
	Durchgestr  bool
	Ausrichtung document.Ausrichtung
	// Schatten (ADR-0021 Nachtrag 26 Punkt 8, ADR-0037 Nachtrag 1): An = der Stil trägt einen; Länge und Weich auf
	// Faktor 1. Schlüssel shadow = WINKEL LÄNGE WEICH #RRGGBBAA, nur geschrieben, wenn An.
	Schatten document.Schatten
}

// Schluessel je Art, in der Reihenfolge, in der sie geschrieben werden.
func schluesselFuer(art string) []string {
	switch art {
	case ArtPfeil, ArtLupe, ArtTextmarker, ArtFreihand:
		return []string{"color", "width", "shadow"}
	case ArtKasten, ArtEllipse:
		return []string{"color", "fill", "width", "shadow"}
	case ArtText, ArtSprechblase:
		return []string{"color", "fill", "font", "size", "bold", "italic", "underline", "strike", "align", "shadow"}
	case ArtNummer:
		return []string{"color", "radius", "shadow"}
	}
	return nil
}

// StilePfad liefert den Ort der Datei: <Konfigurationsordner>/styles.ini (Ordner wie bei Pfad).
func StilePfad() (string, error) {
	p, err := stileOrt()
	return p, errprefix.Wrap("settings", err)
}

func stileOrt() (string, error) {
	o, err := konfigOrdner()
	if err != nil {
		return "", err
	}
	return filepath.Join(o, "styles.ini"), nil
}

// LadeStile liest die Datei am Ort von StilePfad. Fehlt sie, gibt es keine eigenen Stile und keine Warnung.
func LadeStile() ([]Stil, []string) {
	pfad, err := stileOrt()
	if err != nil {
		return nil, []string{fmt.Sprintf(i18n.T("Cannot locate the styles file (%v); using no custom styles."), maske(err.Error()))}
	}
	return LadeStileDatei(pfad)
}

// LadeStileDatei liest die Stildatei pfad.
func LadeStileDatei(pfad string) ([]Stil, []string) {
	f, err := os.OpenFile(pfad, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []string{fmt.Sprintf(i18n.T("%s: cannot read the styles file (%v); using no custom styles."), maske(pfad), maske(err.Error()))}
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, []string{fmt.Sprintf(i18n.T("%s: cannot read the styles file (%v); using no custom styles."), maske(pfad), maske(err.Error()))}
	}
	if !info.Mode().IsRegular() {
		return nil, []string{fmt.Sprintf(i18n.T("%s: the styles file is not a regular file; using no custom styles."), maske(pfad))}
	}
	daten, err := io.ReadAll(io.LimitReader(f, maxDateigroesse+1))
	if err != nil {
		return nil, []string{fmt.Sprintf(i18n.T("%s: cannot read the styles file (%v); using no custom styles."), maske(pfad), maske(err.Error()))}
	}
	if len(daten) > maxDateigroesse {
		return nil, []string{fmt.Sprintf(i18n.T("%s: the styles file is larger than 64 KiB; using no custom styles."), maske(pfad))}
	}
	return ParseStile(pfad, string(daten))
}

// ParseStile wertet den Inhalt einer Stildatei aus; name erscheint in den Warnungen.
func ParseStile(name, inhalt string) ([]Stil, []string) {
	var (
		stile  []Stil
		warn   []string
		zuviel int
		anzahl = map[string]int{}
	)
	melde := func(zeile int, format string, a ...any) {
		if len(warn) >= maxWarnungen {
			zuviel++
			return
		}
		warn = append(warn, fmt.Sprintf("%s:%d: ", maske(name), zeile)+fmt.Sprintf(format, a...))
	}

	// Der Stil im Aufbau.
	var (
		cur       Stil
		offen     bool // ein Stil-Abschnitt läuft
		kopfZeile int
		verworfen bool // der laufende Stil ist schon ausgefallen (nur eine Warnung je Stil)
		gesehen   map[string]bool
	)
	abschliessen := func() {
		if !offen || verworfen {
			offen = false
			return
		}
		offen = false
		if !gesehen["color"] {
			melde(kopfZeile, i18n.T("Style [%s] ignored: key %q is missing."), cur.Art, "color")
			return
		}
		if anzahl[cur.Art] >= MaxEigeneStile {
			melde(kopfZeile, i18n.T("Style [%s] ignored: more than %d styles of this kind."), cur.Art, MaxEigeneStile)
			return
		}
		anzahl[cur.Art]++
		stile = append(stile, bereinigt(cur))
	}
	verwirf := func(zeile int, format string, a ...any) {
		if !verworfen {
			melde(zeile, format, a...)
		}
		verworfen = true
	}

	imAbschnitt := false // schon ein Abschnittskopf gesehen
	inhalt = strings.TrimPrefix(inhalt, "\xef\xbb\xbf")
	for i, zeile := range strings.Split(inhalt, "\n") {
		nr := i + 1
		zeile = strings.TrimSpace(zeile)
		if zeile == "" || zeile[0] == '#' || zeile[0] == ';' {
			continue
		}
		if zeile[0] == '[' {
			abschliessen()
			imAbschnitt = true
			if !strings.HasSuffix(zeile, "]") {
				melde(nr, "%s", i18n.T("Broken section header; its settings up to the next section are ignored."))
				continue
			}
			art := strings.TrimSpace(zeile[1 : len(zeile)-1])
			if schluesselFuer(art) == nil {
				melde(nr, i18n.T("Unknown section [%s]; its keys are ignored."), maske(kuerze(art)))
				continue
			}
			cur = Stil{Art: art, Breite: stilBreite, Schriftgrad: stilSchriftgrad, Radius: stilRadius}
			if art == ArtTextmarker {
				cur.Breite = textmarkerBreite // ADR-0048 Punkt 5
			}
			offen, verworfen, kopfZeile, gesehen = true, false, nr, map[string]bool{}
			continue
		}
		schluessel, wert, ok := strings.Cut(zeile, "=")
		schluessel, wert = strings.TrimSpace(schluessel), strings.TrimSpace(wert)
		if !ok || schluessel == "" {
			melde(nr, "%s", i18n.T("Line is neither a section, a comment nor \"key = value\"; ignored."))
			continue
		}
		if !imAbschnitt {
			melde(nr, i18n.T("Setting %q is outside of any section; ignored."), kuerze(schluessel))
			continue
		}
		if !offen || verworfen {
			continue // Abschnitt unbekannt oder Stil schon ausgefallen: schon gemeldet
		}
		erlaubt := false
		for _, k := range schluesselFuer(cur.Art) {
			if k == schluessel {
				erlaubt = true
			}
		}
		if !erlaubt {
			verwirf(nr, i18n.T("Style [%s] ignored: unknown key %q."), cur.Art, kuerze(schluessel))
			continue
		}
		if gesehen[schluessel] {
			verwirf(nr, i18n.T("Style [%s] ignored: key %q is set twice."), cur.Art, schluessel)
			continue
		}
		if !setzeStilWert(&cur, schluessel, wert) {
			verwirf(nr, i18n.T("Style [%s] ignored: invalid value %q for key %q."), cur.Art, kuerze(wert), schluessel)
			continue
		}
		gesehen[schluessel] = true
	}
	abschliessen()
	if zuviel > 0 {
		warn = append(warn, fmt.Sprintf(i18n.N("… and %d more warning", "… and %d more warnings", zuviel), zuviel))
	}
	return stile, warn
}

// setzeStilWert wertet wert für den Schlüssel k aus und setzt ihn in s (false: ungültig, s bleibt unverändert).
func setzeStilWert(s *Stil, k, wert string) bool {
	switch k {
	case "color":
		c, ok := farbeAus(wert)
		s.Farbe = pick(ok, c, s.Farbe)
		return ok
	case "fill":
		c, ok := farbeAus(wert)
		s.Fuellung = pick(ok, c, s.Fuellung)
		return ok
	case "width":
		g, ok := groesseAus(wert, document.MaxGroesse)
		s.Breite = pick(ok, g, s.Breite)
		return ok
	case "size":
		g, ok := groesseAus(wert, document.MaxSchrift)
		s.Schriftgrad = pick(ok, g, s.Schriftgrad)
		return ok
	case "radius":
		g, ok := groesseAus(wert, document.MaxSchrift)
		s.Radius = pick(ok, g, s.Radius)
		return ok
	case "font":
		if wert == fonts.Go {
			s.Schrift = ""
			return true
		}
		if !fonts.NameGueltig(wert) {
			return false
		}
		s.Schrift = wert
		return true
	case "bold", "italic", "underline", "strike":
		b, ok := janein(wert)
		if !ok {
			return false
		}
		switch k {
		case "bold":
			s.Fett = b
		case "italic":
			s.Kursiv = b
		case "underline":
			s.Unterstr = b
		default:
			s.Durchgestr = b
		}
		return true
	case "shadow":
		sch, ok := schattenAus(wert)
		s.Schatten = pick(ok, sch, s.Schatten)
		return ok
	case "align":
		switch wert {
		case "left":
			s.Ausrichtung = document.AusrichtungLinks
		case "center":
			s.Ausrichtung = document.AusrichtungMitte
		case "right":
			s.Ausrichtung = document.AusrichtungRechts
		default:
			return false
		}
		return true
	}
	return false
}

func pick[T any](ok bool, neu, alt T) T {
	if ok {
		return neu
	}
	return alt
}

// farbeAus liest #RRGGBBAA oder #RRGGBB.
func farbeAus(w string) (color.NRGBA, bool) {
	if len(w) != 7 && len(w) != 9 || w[0] != '#' {
		return color.NRGBA{}, false
	}
	for _, r := range w[1:] {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F') {
			return color.NRGBA{}, false
		}
	}
	n, err := strconv.ParseUint(w[1:], 16, 32)
	if err != nil {
		return color.NRGBA{}, false
	}
	if len(w) == 7 {
		return color.NRGBA{R: uint8(n >> 16), G: uint8(n >> 8), B: uint8(n), A: 0xFF}, true
	}
	return color.NRGBA{R: uint8(n >> 24), G: uint8(n >> 16), B: uint8(n >> 8), A: uint8(n)}, true
}

func farbeText(c color.NRGBA) string {
	return fmt.Sprintf("#%02X%02X%02X%02X", c.R, c.G, c.B, c.A)
}

// schattenAus liest „WINKEL LÄNGE WEICH #RRGGBBAA“ (Leerraum getrennt): Winkel endlich (auf 0 bis 360 gebracht), Länge
// und Weich endlich von 0 bis zu den Grenzen des Dokuments.
func schattenAus(w string) (document.Schatten, bool) {
	teile := strings.Fields(w)
	if len(teile) != 4 {
		return document.Schatten{}, false
	}
	var z [3]float64
	for i, t := range teile[:3] {
		g, err := strconv.ParseFloat(t, 64)
		if err != nil || math.IsNaN(g) || math.IsInf(g, 0) {
			return document.Schatten{}, false
		}
		z[i] = g
	}
	c, ok := farbeAus(teile[3])
	if !ok || z[1] < 0 || z[1] > document.MaxSchattenLaenge || z[2] < 0 || z[2] > document.MaxSchattenWeich {
		return document.Schatten{}, false
	}
	return document.Schatten{An: true, Winkel: document.NormWinkel(z[0]), Laenge: z[1], Weich: z[2], Farbe: c}, true
}

// schattenText ist die Form von schattenAus.
func schattenText(sch document.Schatten) string {
	return zahlText(sch.Winkel) + " " + zahlText(sch.Laenge) + " " + zahlText(sch.Weich) + " " + farbeText(sch.Farbe)
}

// groesseAus liest eine endliche Zahl 0 < g ≤ max.
func groesseAus(w string, grenze float64) (float64, bool) {
	g, err := strconv.ParseFloat(w, 64)
	if err != nil || math.IsNaN(g) || math.IsInf(g, 0) || !(g > 0) || g > grenze {
		return 0, false
	}
	return g, true
}

func zahlText(g float64) string { return strconv.FormatFloat(g, 'f', -1, 64) }

func jaNein(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// kopfStile steht am Anfang der geschriebenen Datei.
const kopfStile = `# Quick styles of Schnapp (custom styles of the editor's right-hand panel).
# Schnapp writes this file and replaces it as a whole whenever a style is added or removed:
# changes made by hand are lost. One section per style, in the order of the tiles.
# Sizes are given for a 1080-pixel image and grow with the image size.
`

// PruefeStil meldet, warum s nicht in die Datei geschrieben werden kann (nil: in Ordnung). Dieselbe Prüfung wie beim
// Schreiben, damit die Oberfläche keinen Stil annimmt, der jedes weitere Speichern blockierte (F-628).
func PruefeStil(s Stil) error { return pruefeStil(s) }

// pruefeStil meldet, warum s nicht geschrieben werden kann (nil: in Ordnung).
func pruefeStil(s Stil) error {
	if schluesselFuer(s.Art) == nil {
		return fmt.Errorf("unknown style kind %q", s.Art)
	}
	ok := func(g, grenze float64) bool { return g > 0 && g <= grenze }
	for _, k := range schluesselFuer(s.Art) {
		switch {
		case k == "width" && !ok(s.Breite, document.MaxGroesse),
			k == "size" && !ok(s.Schriftgrad, document.MaxSchrift),
			k == "radius" && !ok(s.Radius, document.MaxSchrift):
			return fmt.Errorf("style [%s]: invalid %s", s.Art, k)
		case k == "font" && s.Schrift != "" &&
			(!fonts.NameGueltig(s.Schrift) || strings.TrimSpace(s.Schrift) != s.Schrift || s.Schrift == fonts.Go):
			return fmt.Errorf("style [%s]: invalid font name", s.Art)
		case k == "align" && s.Ausrichtung > document.AusrichtungRechts:
			return fmt.Errorf("style [%s]: invalid alignment", s.Art)
		case k == "shadow" && s.Schatten.An && !schattenGueltig(s.Schatten):
			return fmt.Errorf("style [%s]: invalid shadow", s.Art)
		}
	}
	return nil
}

// schattenGueltig meldet, ob sch so geschrieben und wieder gelesen werden kann (schattenAus).
func schattenGueltig(sch document.Schatten) bool {
	_, ok := schattenAus(schattenText(sch))
	return ok && sch.Winkel >= 0 && sch.Winkel < 360
}

// FormatStile liefert den Inhalt der Datei für stile (ohne zu prüfen, ob die Zahl je Art in Ordnung ist).
func FormatStile(stile []Stil) (string, error) {
	var b strings.Builder
	b.WriteString(kopfStile)
	for _, s := range stile {
		if err := pruefeStil(s); err != nil {
			return "", err
		}
		b.WriteString("\n[" + s.Art + "]\n")
		for _, k := range schluesselFuer(s.Art) {
			if k == "shadow" && !s.Schatten.An {
				continue // ohne Schatten fehlt der Schlüssel
			}
			var w string
			switch k {
			case "shadow":
				w = schattenText(s.Schatten)
			case "color":
				w = farbeText(s.Farbe)
			case "fill":
				w = farbeText(s.Fuellung)
			case "width":
				w = zahlText(s.Breite)
			case "size":
				w = zahlText(s.Schriftgrad)
			case "radius":
				w = zahlText(s.Radius)
			case "font":
				w = s.Schrift
				if w == "" {
					w = fonts.Go
				}
			case "bold":
				w = jaNein(s.Fett)
			case "italic":
				w = jaNein(s.Kursiv)
			case "underline":
				w = jaNein(s.Unterstr)
			case "strike":
				w = jaNein(s.Durchgestr)
			case "align":
				w = [...]string{"left", "center", "right"}[s.Ausrichtung]
			}
			b.WriteString(k + " = " + w + "\n")
		}
	}
	if !utf8.ValidString(b.String()) {
		return "", ErrStileKeinUTF8
	}
	if b.Len() > maxDateigroesse {
		return "", ErrStileZuGross
	}
	return b.String(), nil
}

// SchreibeStile schreibt die Datei am Ort von StilePfad ganz neu.
func SchreibeStile(stile []Stil) error {
	pfad, err := stileOrt()
	if err != nil {
		return errprefix.Wrap("settings", err)
	}
	return SchreibeStileDatei(pfad, stile)
}

// SchreibeStileDatei schreibt die Stildatei pfad ganz neu: Ordner 0700, Datei 0600, atomar über eine temporäre Datei im
// selben Ordner. Ein Verweis als Datei bleibt, das Ziel wird geschrieben; ein anderes als reguläres Ziel ist ein Fehler.
func SchreibeStileDatei(pfad string, stile []Stil) error {
	return errprefix.Wrap("settings", schreibeStileDatei(pfad, stile))
}

func schreibeStileDatei(pfad string, stile []Stil) error {
	inhalt, err := FormatStile(stile)
	if err != nil {
		return err
	}
	return schreibeEigeneDatei(pfad, inhalt, ErrStileVerweis, ErrStileKeineDatei)
}

// schreibeEigeneDatei schreibt eine Datei, die dem Programm gehört (styles.ini, tools.ini), ganz neu: Ordner 0700,
// Datei 0600, atomar; ein Verweis bleibt und sein Ziel wird geschrieben (errVerweis: Ziel fehlt), ein anderes als
// reguläres Ziel ist errKeineDatei.
func schreibeEigeneDatei(pfad, inhalt string, errVerweis, errKeineDatei error) error {
	ziel := pfad
	if info, err := os.Lstat(pfad); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			z, err := filepath.EvalSymlinks(pfad)
			if err != nil {
				if os.IsNotExist(err) {
					return errVerweis
				}
				return err
			}
			ziel = z
			if info, err = os.Stat(ziel); err != nil {
				return err
			}
		}
		if !info.Mode().IsRegular() {
			return errKeineDatei
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	ordner := filepath.Dir(ziel)
	if err := os.MkdirAll(ordner, 0o700); err != nil {
		return err
	}
	return schreibeAtomar(ordner, ziel, []byte(inhalt), 0o600)
}

// bereinigt nullt die Felder, die für die Art von s nicht in der Datei stehen: Zwei gleiche Stile sind dann auch
// als Werte gleich.
func bereinigt(s Stil) Stil {
	hat := map[string]bool{}
	for _, k := range schluesselFuer(s.Art) {
		hat[k] = true
	}
	r := Stil{Art: s.Art}
	if hat["color"] {
		r.Farbe = s.Farbe
	}
	if hat["fill"] {
		r.Fuellung = s.Fuellung
	}
	if hat["width"] {
		r.Breite = s.Breite
	}
	if hat["size"] {
		r.Schriftgrad = s.Schriftgrad
	}
	if hat["radius"] {
		r.Radius = s.Radius
	}
	if hat["font"] {
		r.Schrift, r.Fett, r.Kursiv, r.Unterstr, r.Durchgestr = s.Schrift, s.Fett, s.Kursiv, s.Unterstr, s.Durchgestr
		r.Ausrichtung = s.Ausrichtung
	}
	if hat["shadow"] && s.Schatten.An {
		r.Schatten = s.Schatten
	}
	return r
}

// Bereinigt ist s ohne die Angaben, die für seine Art nicht gelten (so liest ParseStile einen Stil).
func (s Stil) Bereinigt() Stil { return bereinigt(s) }

// StilHinzu hängt s an liste, wenn kein gleicher Stil (nach Bereinigen) der Art darin steht und die Art noch Platz hat
// (MaxEigeneStile); sonst bleibt liste. Liefert eine neue Liste.
func StilHinzu(liste []Stil, s Stil) []Stil {
	s = s.Bereinigt()
	n := 0
	for _, e := range liste {
		if e.Art != s.Art {
			continue
		}
		if e.Bereinigt() == s {
			return slices.Clone(liste)
		}
		n++
	}
	if n >= MaxEigeneStile {
		return slices.Clone(liste)
	}
	return append(slices.Clone(liste), s)
}

// StilEntfernt entfernt den ersten Stil aus liste, der s gleicht (nach Bereinigen); fehlt er, bleibt liste.
func StilEntfernt(liste []Stil, s Stil) []Stil {
	s = s.Bereinigt()
	for i, e := range liste {
		if e.Bereinigt() == s {
			return slices.Delete(slices.Clone(liste), i, i+1)
		}
	}
	return slices.Clone(liste)
}

// stileFlock ist das Sperren selbst (Naht für Tests).
var stileFlock = syscall.Flock

// AendereStile liest die Datei am Ort von StilePfad unter einer Sperre neu, wendet aend an und schreibt das Ergebnis
// (F-627). Liefert den Stand der Datei danach und ob die alte Datei Unlesbares enthielt und als styles.ini.bak gesichert
// wurde (F-629 b).
func AendereStile(aend func([]Stil) []Stil) ([]Stil, bool, error) {
	pfad, err := stileOrt()
	if err != nil {
		return nil, false, errprefix.Wrap("settings", err)
	}
	return AendereStileDatei(pfad, aend)
}

// AendereStileDatei ist AendereStile für die Stildatei pfad. Unter einer flock-Sperre `<pfad>.lock` (0600, neben der
// Datei; ohne flock-Unterstützung ohne Sperre) wird die Datei neu gelesen, aend auf die gelesenen Stile angewandt und
// geschrieben wie bei SchreibeStileDatei; so gehen Änderungen eines zweiten Fensters nicht verloren. Ist die Datei
// nicht lesbar (keine reguläre Datei, zu groß, Lesefehler), bleibt sie unberührt (ErrStileUnlesbar). Wurden beim Lesen
// Stile oder Zeilen verworfen, legt der erste Schreibzug vorher eine Sicherung `<pfad>.bak` an (exklusiv, 0600); gibt es
// sie schon, wird nichts geschrieben (ErrStileSicherung): Eine frühere Sicherung geht nie verloren.
func AendereStileDatei(pfad string, aend func([]Stil) []Stil) ([]Stil, bool, error) {
	neu, gesichert, err := aendereStileDatei(pfad, aend)
	return neu, gesichert, errprefix.Wrap("settings", err)
}

func aendereStileDatei(pfad string, aend func([]Stil) []Stil) (_ []Stil, gesichert bool, _ error) {
	ordner := filepath.Dir(pfad)
	if err := os.MkdirAll(ordner, 0o700); err != nil {
		return nil, false, err
	}
	sp, err := stileSperreNehmen(pfad + ".lock")
	if err != nil {
		return nil, false, err
	}
	if sp != nil {
		defer sp.Close()
	}
	var alt []Stil
	daten, da, err := leseStileRoh(pfad)
	if err != nil {
		return nil, false, errors.Join(ErrStileUnlesbar, err)
	}
	if da {
		var warn []string
		alt, warn = ParseStile(pfad, string(daten))
		if len(warn) > 0 {
			if err := sichereStile(pfad+".bak", daten); err != nil {
				return nil, false, err
			}
			gesichert = true
		}
	}
	neu := aend(slices.Clone(alt))
	if err := schreibeStileDatei(pfad, neu); err != nil {
		return nil, false, err
	}
	return neu, gesichert, nil
}

// leseStileRoh liest die Bytes der Stildatei (da: sie existiert); jedes Hindernis außer „fehlt“ ist ein Fehler.
func leseStileRoh(pfad string) (daten []byte, da bool, _ error) {
	f, err := os.OpenFile(pfad, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, false, err
	}
	if !info.Mode().IsRegular() {
		return nil, false, ErrStileKeineDatei
	}
	daten, err = io.ReadAll(io.LimitReader(f, maxDateigroesse+1))
	if err != nil {
		return nil, false, err
	}
	if len(daten) > maxDateigroesse {
		return nil, false, ErrStileZuGross
	}
	return daten, true, nil
}

// sichereStile legt die Sicherung name neu an (exklusiv, 0600, ohne Verweis zu folgen) und schreibt daten hinein.
func sichereStile(name string, daten []byte) error {
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return ErrStileSicherung
		}
		return err
	}
	_, err = f.Write(daten)
	if err == nil {
		err = f.Sync()
	}
	return errors.Join(err, f.Close())
}

// stileSperreNehmen nimmt die flock-Sperre name (wartet bis 3 s auf ein anderes Fenster). nil ohne Fehler: das
// Dateisystem kennt kein flock, es geht ohne Sperre weiter.
func stileSperreNehmen(name string) (*os.File, error) {
	f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0o600)
	if err != nil {
		return nil, err
	}
	if st, err := f.Stat(); err != nil || !st.Mode().IsRegular() {
		_ = f.Close()
		return nil, errors.Join(err, errors.New("the lock file is not a regular file"))
	}
	ende := time.Now().Add(3 * time.Second)
	for {
		err := stileFlock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		switch {
		case err == nil:
			return f, nil
		case errors.Is(err, syscall.ENOLCK), errors.Is(err, syscall.ENOTSUP), errors.Is(err, syscall.EOPNOTSUPP):
			_ = f.Close()
			return nil, nil
		case errors.Is(err, syscall.EWOULDBLOCK):
			if time.Now().After(ende) {
				_ = f.Close()
				return nil, ErrStileGesperrt
			}
			time.Sleep(20 * time.Millisecond)
		default:
			_ = f.Close()
			return nil, err
		}
	}
}
