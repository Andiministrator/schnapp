// SPDX-License-Identifier: GPL-3.0-or-later

package settings

// Eigene Farbfelder (ADR-0021 Nachtrag 24, Karte #364): `colors.ini` im Konfigurationsordner, neben `styles.ini`. Die
// Datei gehört dem Programm wie `styles.ini`: Schnapp schreibt sie ganz neu, aber aus dem frisch gelesenen Stand unter
// einer Sperre; gelesen wird mit denselben Regeln (höchstens 64 KiB, Warnungen statt Abbruch, ohne Blockieren öffnen).
//
// Format: ein Abschnitt `[swatches]`, darin je Farbfeld eine Zeile `color = #RRGGBBAA` (beim Lesen auch `#RRGGBB`), in
// der Reihenfolge der Felder. Fehlt die Datei, gelten die eingebauten Farbfelder (der Aufrufer kennt sie); eine Datei
// ohne Zeile heißt: keine Farbfelder.

import (
	"errors"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Andiministrator/schnapp/internal/errprefix"
	"github.com/Andiministrator/schnapp/internal/i18n"
)

// MaxFarbfelder ist die Grenze der eigenen Farbfelder.
const MaxFarbfelder = 24

// Fehler beim Ändern der Farbfelder.
var (
	ErrFarbenUnlesbar  = errors.New("the colour swatches file cannot be read, so it is left as it is")
	ErrFarbenGesperrt  = errors.New("the colour swatches file is locked by another window")
	ErrFarbenSicherung = errors.New("the colour swatches file has parts that cannot be read and colors.ini.bak already exists, so it is left as it is")
)

func farbenOrt() (string, error) {
	o, err := konfigOrdner()
	if err != nil {
		return "", err
	}
	return filepath.Join(o, "colors.ini"), nil
}

// LadeFarben liest die Farbfelder am Ort neben styles.ini. da meldet, ob es die Datei gibt (sonst gelten die
// eingebauten Farbfelder).
func LadeFarben() (farben []color.NRGBA, da bool, warn []string) {
	pfad, err := farbenOrt()
	if err != nil {
		return nil, false, []string{fmt.Sprintf(i18n.T("Cannot locate the colour swatches file (%v); using the built-in swatches."), maske(err.Error()))}
	}
	return LadeFarbenDatei(pfad)
}

// LadeFarbenDatei liest die Farbfelder aus pfad.
func LadeFarbenDatei(pfad string) (farben []color.NRGBA, da bool, warn []string) {
	daten, da, err := leseStileRoh(pfad)
	if err != nil {
		return nil, false, []string{fmt.Sprintf(i18n.T("%s: cannot read the colour swatches file (%v); using the built-in swatches."), maske(pfad), maske(err.Error()))}
	}
	if !da {
		return nil, false, nil
	}
	farben, warn = ParseFarben(pfad, string(daten))
	return farben, true, warn
}

// ParseFarben wertet den Inhalt einer Farbdatei aus; name erscheint in den Warnungen. Ungültige Zeilen und doppelte
// Farben fallen weg, über MaxFarbfelder hinaus wird abgeschnitten.
func ParseFarben(name, inhalt string) ([]color.NRGBA, []string) {
	var (
		farben []color.NRGBA
		warn   []string
	)
	melde := func(zeile int, format string, a ...any) {
		if len(warn) < maxWarnungen {
			warn = append(warn, fmt.Sprintf("%s:%d: ", maske(name), zeile)+fmt.Sprintf(format, a...))
		}
	}
	inhalt = strings.TrimPrefix(inhalt, "\xef\xbb\xbf")
	abschnitt := ""
	for i, zeile := range strings.Split(inhalt, "\n") {
		nr := i + 1
		z := strings.TrimSpace(strings.TrimSuffix(zeile, "\r"))
		switch {
		case z == "" || strings.HasPrefix(z, ";"):
			continue
		case strings.HasPrefix(z, "[") && strings.HasSuffix(z, "]"):
			abschnitt = strings.TrimSpace(z[1 : len(z)-1])
			if abschnitt != "swatches" {
				melde(nr, i18n.T("Unknown section [%s] ignored."), abschnitt)
			}
			continue
		}
		if abschnitt != "swatches" {
			if abschnitt == "" {
				melde(nr, "%s", i18n.T("Line outside of [swatches] ignored."))
			}
			continue
		}
		k, w, ok := strings.Cut(z, "=")
		k, w = strings.TrimSpace(k), strings.TrimSpace(w)
		if !ok || k != "color" {
			melde(nr, "%s", i18n.T("Line ignored: only “color = #RRGGBBAA” is allowed here."))
			continue
		}
		c, ok := farbeAus(w)
		if !ok {
			melde(nr, i18n.T("Colour %q ignored: not #RRGGBB or #RRGGBBAA."), w)
			continue
		}
		if slices.Contains(farben, c) {
			continue
		}
		if len(farben) >= MaxFarbfelder {
			melde(nr, i18n.T("Colour ignored: more than %d colour swatches."), MaxFarbfelder)
			continue
		}
		farben = append(farben, c)
	}
	return farben, warn
}

// FormatFarben ist der Inhalt einer Farbdatei für farben.
func FormatFarben(farben []color.NRGBA) string {
	var b strings.Builder
	b.WriteString("; Schnapp colour swatches. Written by Schnapp: changes made here while Schnapp is open may be lost.\n")
	b.WriteString("[swatches]\n")
	for _, c := range farben {
		fmt.Fprintf(&b, "color = %s\n", farbeText(c))
	}
	return b.String()
}

// AendereFarben liest die Farbfelder unter einer Sperre `colors.ini.lock` neu, wendet aend an (alt und da wie bei
// LadeFarben) und schreibt das Ergebnis atomar (Ordner 0700, Datei 0600) — so gehen Änderungen eines zweiten Fensters
// nicht verloren. Enthielt die Datei Unlesbares, legt der Schreibzug vorher `colors.ini.bak` an; gibt es sie schon,
// bleibt alles, wie es ist. Liefert die geschriebenen Farbfelder.
func AendereFarben(aend func(alt []color.NRGBA, da bool) []color.NRGBA) ([]color.NRGBA, error) {
	pfad, err := farbenOrt()
	if err != nil {
		return nil, errprefix.Wrap("settings", err)
	}
	return AendereFarbenDatei(pfad, aend)
}

// AendereFarbenDatei ist AendereFarben für die Datei pfad.
func AendereFarbenDatei(pfad string, aend func(alt []color.NRGBA, da bool) []color.NRGBA) ([]color.NRGBA, error) {
	neu, err := aendereFarbenDatei(pfad, aend)
	return neu, errprefix.Wrap("settings", err)
}

func aendereFarbenDatei(pfad string, aend func([]color.NRGBA, bool) []color.NRGBA) ([]color.NRGBA, error) {
	if err := os.MkdirAll(filepath.Dir(pfad), 0o700); err != nil {
		return nil, err
	}
	sp, err := stileSperreNehmen(pfad + ".lock")
	if errors.Is(err, ErrStileGesperrt) {
		return nil, ErrFarbenGesperrt
	}
	if err != nil {
		return nil, err
	}
	if sp != nil {
		defer sp.Close()
	}
	daten, da, err := leseStileRoh(pfad)
	if err != nil {
		return nil, errors.Join(ErrFarbenUnlesbar, err)
	}
	var alt []color.NRGBA
	if da {
		var warn []string
		alt, warn = ParseFarben(pfad, string(daten))
		if len(warn) > 0 {
			if err := sichereStile(pfad+".bak", daten); err != nil {
				if errors.Is(err, ErrStileSicherung) {
					return nil, ErrFarbenSicherung
				}
				return nil, err
			}
		}
	}
	neu := aend(slices.Clone(alt), da)
	if len(neu) > MaxFarbfelder {
		neu = neu[:MaxFarbfelder]
	}
	ziel := pfad
	if info, err := os.Lstat(pfad); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			z, err := filepath.EvalSymlinks(pfad)
			if err != nil {
				return nil, errors.Join(ErrFarbenUnlesbar, err)
			}
			ziel = z
		}
	}
	if err := schreibeAtomar(filepath.Dir(ziel), ziel, []byte(FormatFarben(neu)), 0o600); err != nil {
		return nil, err
	}
	return neu, nil
}

// FarbeHinzu hängt c an farben, wenn sie nicht schon darin steht, nicht durchsichtig ist und Platz bleibt; liefert eine
// neue Liste.
func FarbeHinzu(farben []color.NRGBA, c color.NRGBA) []color.NRGBA {
	if c.A == 0 || slices.Contains(farben, c) || len(farben) >= MaxFarbfelder {
		return slices.Clone(farben)
	}
	return append(slices.Clone(farben), c)
}

// FarbeEntfernt entfernt c aus farben; fehlt sie, bleibt die Liste.
func FarbeEntfernt(farben []color.NRGBA, c color.NRGBA) []color.NRGBA {
	if i := slices.Index(farben, c); i >= 0 {
		return slices.Delete(slices.Clone(farben), i, i+1)
	}
	return slices.Clone(farben)
}
