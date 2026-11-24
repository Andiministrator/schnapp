// SPDX-License-Identifier: GPL-3.0-or-later

package settings

// Eigene Schnellstile (ADR-0037 Punkt 6): `styles.ini` im Konfigurationsordner, neben `settings.ini`. Anders als diese
// gehört die Datei dem Programm: Schnapp schreibt sie jedes Mal ganz neu (Kopfkommentar sagt das), der Leser folgt
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
	"strconv"
	"strings"
	"syscall"
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
)

// MaxEigeneStile ist die Grenze eigener Stile je Objektart (ADR-0037 Punkt 5).
const MaxEigeneStile = 16

// Werte, die gelten, wenn ein Schlüssel fehlt (Faktor 1, wie die Voreinstellung in canvas.StandardStil).
const (
	stilBreite      = 4.0
	stilSchriftgrad = 20.0
	stilRadius      = 14.0
)

// Fehler des Schreibens (der Leser meldet Warnungen statt Fehler).
var (
	ErrStileZuGross    = errors.New("the styles file would be larger than 64 KiB")
	ErrStileKeineDatei = errors.New("the styles file is not a regular file")
	ErrStileVerweis    = errors.New("the styles file is a link to nothing")
	ErrStileKeinUTF8   = errors.New("a style contains text that is not valid UTF-8")
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
}

// Schluessel je Art, in der Reihenfolge, in der sie geschrieben werden.
func schluesselFuer(art string) []string {
	switch art {
	case ArtPfeil, ArtLupe:
		return []string{"color", "width"}
	case ArtKasten:
		return []string{"color", "fill", "width"}
	case ArtText, ArtSprechblase:
		return []string{"color", "fill", "font", "size", "bold", "italic", "underline", "strike", "align"}
	case ArtNummer:
		return []string{"color", "radius"}
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
		}
	}
	return nil
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
			var w string
			switch k {
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
	ziel := pfad
	if info, err := os.Lstat(pfad); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			z, err := filepath.EvalSymlinks(pfad)
			if err != nil {
				if os.IsNotExist(err) {
					return ErrStileVerweis
				}
				return err
			}
			ziel = z
			if info, err = os.Stat(ziel); err != nil {
				return err
			}
		}
		if !info.Mode().IsRegular() {
			return ErrStileKeineDatei
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
	return r
}

// Bereinigt ist s ohne die Angaben, die für seine Art nicht gelten (so liest ParseStile einen Stil).
func (s Stil) Bereinigt() Stil { return bereinigt(s) }
