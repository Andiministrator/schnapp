// SPDX-License-Identifier: GPL-3.0-or-later

package settings

// Stil je Werkzeug (ADR-0041): `tools.ini` im Konfigurationsordner, neben `settings.ini` und `styles.ini`. Die Datei
// gehört dem Programm: Schnapp schreibt sie beim Schließen eines Editorfensters ganz neu. Der Leser folgt denselben
// Regeln wie bei `styles.ini` (höchstens 64 KiB, ohne Blockieren öffnen, Warnungen statt Abbruch), ist aber
// nachsichtiger: Ein ungültiger oder unbekannter Schlüssel fällt allein weg, der Rest des Abschnitts gilt.
//
// Format: ein Abschnitt je Werkzeug (`[arrow]`, `[box]`, `[text]`, `[step]`, `[magnifier]`; die Sprechblase teilt den
// Abschnitt des Texts), Schlüssel wie in styles.ini, dazu `line_style`, `start_head`, `end_head`, `corner_radius` und
// `shadow_on`. Größen auf Faktor 1.

import (
	"errors"
	"fmt"
	"io"
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

// Fehler des Schreibens von tools.ini.
var (
	ErrWerkzeugeZuGross    = errors.New("the tools file would be larger than 64 KiB")
	ErrWerkzeugeKeineDatei = errors.New("the tools file is not a regular file")
	ErrWerkzeugeVerweis    = errors.New("the tools file is a link to nothing")
	ErrWerkzeugeKeinUTF8   = errors.New("a tool setting contains text that is not valid UTF-8")
)

// WerkzeugStil ist der gemerkte Stil eines Werkzeugs. Die Felder von Stil gelten wie in styles.ini; Gesetzt nennt die
// Schlüssel, die in der Datei standen (beim Lesen), die übrigen behalten die Voreinstellung des Werkzeugs.
type WerkzeugStil struct {
	Stil
	Strichart  document.Strichart
	KopfEnde   document.Kopfart
	KopfAnfang document.Kopfart // KopfKeine = keine Spitze am Anfang
	Eckradius  float64
	// Gesetzt sind die Schlüssel, die beim Lesen gültig in der Datei standen (nil beim Schreiben: alle).
	Gesetzt map[string]bool
}

// werkzeugSchluessel sind die Schlüssel je Werkzeug, in der Reihenfolge, in der sie geschrieben werden.
func werkzeugSchluessel(art string) []string {
	switch art {
	case ArtPfeil:
		return []string{"color", "width", "line_style", "start_head", "end_head", "shadow_on", "shadow"}
	case ArtKasten:
		return []string{"color", "fill", "width", "corner_radius", "line_style", "shadow_on", "shadow"}
	case ArtText:
		return []string{"color", "fill", "font", "size", "bold", "italic", "underline", "strike", "align", "shadow_on", "shadow"}
	case ArtNummer:
		return []string{"color", "radius", "shadow_on", "shadow"}
	case ArtLupe:
		return []string{"color", "width", "line_style", "shadow_on", "shadow"}
	case ArtEllipse: // wie der Kasten ohne Eckradius (ADR-0047 Punkt 4)
		return []string{"color", "fill", "width", "line_style", "shadow_on", "shadow"}
	}
	return nil
}

func werkzeugeOrt() (string, error) {
	o, err := konfigOrdner()
	if err != nil {
		return "", err
	}
	return filepath.Join(o, "tools.ini"), nil
}

// LadeWerkzeugStile liest tools.ini im Konfigurationsordner. Fehlt die Datei, gibt es nichts und keine Warnung.
func LadeWerkzeugStile() ([]WerkzeugStil, []string) {
	pfad, err := werkzeugeOrt()
	if err != nil {
		return nil, []string{fmt.Sprintf(i18n.T("Cannot locate the tools file (%v); tools start with their defaults."), maske(err.Error()))}
	}
	return LadeWerkzeugStileDatei(pfad)
}

// LadeWerkzeugStileDatei liest die Datei pfad.
func LadeWerkzeugStileDatei(pfad string) ([]WerkzeugStil, []string) {
	f, err := os.OpenFile(pfad, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []string{fmt.Sprintf(i18n.T("%s: cannot read the tools file (%v); tools start with their defaults."), maske(pfad), maske(err.Error()))}
	}
	defer f.Close()
	info, err := f.Stat()
	if err == nil && !info.Mode().IsRegular() {
		return nil, []string{fmt.Sprintf(i18n.T("%s: the tools file is not a regular file; tools start with their defaults."), maske(pfad))}
	}
	var daten []byte
	if err == nil {
		daten, err = io.ReadAll(io.LimitReader(f, maxDateigroesse+1))
	}
	if err != nil {
		return nil, []string{fmt.Sprintf(i18n.T("%s: cannot read the tools file (%v); tools start with their defaults."), maske(pfad), maske(err.Error()))}
	}
	if len(daten) > maxDateigroesse {
		return nil, []string{fmt.Sprintf(i18n.T("%s: the tools file is larger than 64 KiB; tools start with their defaults."), maske(pfad))}
	}
	return ParseWerkzeugStile(pfad, string(daten))
}

// ParseWerkzeugStile wertet den Inhalt von tools.ini aus; name erscheint in den Warnungen. Je Werkzeug gilt der erste
// Abschnitt, ein zweiter fällt mit einer Warnung weg.
func ParseWerkzeugStile(name, inhalt string) ([]WerkzeugStil, []string) {
	var (
		stile  []WerkzeugStil
		warn   []string
		zuviel int
		cur    *WerkzeugStil
		da     = map[string]bool{}
		im     bool // schon ein Abschnittskopf gesehen
	)
	melde := func(zeile int, format string, a ...any) {
		if len(warn) >= maxWarnungen {
			zuviel++
			return
		}
		warn = append(warn, fmt.Sprintf("%s:%d: ", maske(name), zeile)+fmt.Sprintf(format, a...))
	}
	inhalt = strings.TrimPrefix(inhalt, "\xef\xbb\xbf")
	for i, zeile := range strings.Split(inhalt, "\n") {
		nr := i + 1
		zeile = strings.TrimSpace(zeile)
		if zeile == "" || zeile[0] == '#' || zeile[0] == ';' {
			continue
		}
		if zeile[0] == '[' {
			im, cur = true, nil
			if !strings.HasSuffix(zeile, "]") {
				melde(nr, "%s", i18n.T("Broken section header; its settings up to the next section are ignored."))
				continue
			}
			art := strings.TrimSpace(zeile[1 : len(zeile)-1])
			switch {
			case werkzeugSchluessel(art) == nil:
				melde(nr, i18n.T("Unknown section [%s]; its keys are ignored."), maske(kuerze(art)))
			case da[art]:
				melde(nr, i18n.T("Section [%s] appears twice; the second one is ignored."), art)
			default:
				da[art] = true
				stile = append(stile, WerkzeugStil{Stil: Stil{Art: art}, KopfAnfang: document.KopfKeine, Gesetzt: map[string]bool{}})
				cur = &stile[len(stile)-1]
			}
			continue
		}
		schluessel, wert, ok := strings.Cut(zeile, "=")
		schluessel, wert = strings.TrimSpace(schluessel), strings.TrimSpace(wert)
		if !ok || schluessel == "" {
			melde(nr, "%s", i18n.T("Line is neither a section, a comment nor \"key = value\"; ignored."))
			continue
		}
		if !im {
			melde(nr, i18n.T("Setting %q is outside of any section; ignored."), kuerze(schluessel))
			continue
		}
		if cur == nil {
			continue // Abschnitt unbekannt oder doppelt: schon gemeldet
		}
		erlaubt := false
		for _, k := range werkzeugSchluessel(cur.Art) {
			erlaubt = erlaubt || k == schluessel
		}
		switch {
		case !erlaubt:
			melde(nr, i18n.T("Unknown key %q in section [%s]; ignored."), kuerze(schluessel), cur.Art)
		case cur.Gesetzt[schluessel]:
			melde(nr, i18n.T("Key %q is set twice in section [%s]; the second one is ignored."), schluessel, cur.Art)
		case !setzeWerkzeugWert(cur, schluessel, wert):
			melde(nr, i18n.T("Invalid value %q for key %q in section [%s]; ignored."), kuerze(wert), schluessel, cur.Art)
		default:
			cur.Gesetzt[schluessel] = true
		}
	}
	// shadow_on ohne shadow: der Schalter gilt mit den Werten der Voreinstellung; das setzt canvas.
	if zuviel > 0 {
		warn = append(warn, fmt.Sprintf(i18n.N("… and %d more warning", "… and %d more warnings", zuviel), zuviel))
	}
	return stile, warn
}

// setzeWerkzeugWert wertet wert für den Schlüssel k aus (false: ungültig, s bleibt unverändert).
func setzeWerkzeugWert(s *WerkzeugStil, k, wert string) bool {
	switch k {
	case "line_style":
		a, ok := document.StrichartAus(wert)
		s.Strichart = pick(ok, a, s.Strichart)
		return ok
	case "start_head", "end_head":
		kopf, ok := document.KopfartAus(wert)
		if !ok {
			return false
		}
		if k == "start_head" {
			s.KopfAnfang = kopf
		} else {
			s.KopfEnde = kopf
		}
		return true
	case "corner_radius":
		g, err := strconv.ParseFloat(wert, 64)
		if err != nil || !(g >= 0) || g > document.MaxGroesse {
			return false
		}
		s.Eckradius = g
		return true
	case "shadow_on":
		b, ok := janein(wert)
		if ok {
			s.Schatten.An = b
		}
		return ok
	case "shadow":
		sch, ok := schattenAus(wert)
		if ok {
			an := s.Schatten.An // shadow_on davor gelesen
			s.Schatten = sch
			s.Schatten.An = an
		}
		return ok
	}
	return setzeStilWert(&s.Stil, k, wert)
}

// kopfWerkzeuge steht am Anfang der geschriebenen Datei.
const kopfWerkzeuge = `# Last settings of each drawing tool of Schnapp (ADR-0041).
# Schnapp writes this file when an editor window closes and replaces it as a whole: changes made by hand while a
# window is open are lost. Sizes are given for a 1080-pixel image and grow with the image size.
`

// FormatWerkzeugStile ist der Inhalt von tools.ini für stile (je Werkzeug höchstens einer, sonst gilt der erste).
func FormatWerkzeugStile(stile []WerkzeugStil) (string, error) {
	var b strings.Builder
	b.WriteString(kopfWerkzeuge)
	da := map[string]bool{}
	for _, s := range stile {
		keys := werkzeugSchluessel(s.Art)
		if keys == nil || da[s.Art] {
			continue
		}
		da[s.Art] = true
		b.WriteString("\n[" + s.Art + "]\n")
		for _, k := range keys {
			var w string
			switch k {
			case "line_style":
				w = s.Strichart.Name()
			case "start_head":
				w = s.KopfAnfang.Name()
			case "end_head":
				w = s.KopfEnde.Name()
			case "corner_radius":
				w = zahlText(s.Eckradius)
			case "shadow_on":
				w = jaNein(s.Schatten.An)
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
				w = [...]string{"left", "center", "right"}[min(int(s.Ausrichtung), 2)]
			}
			b.WriteString(k + " = " + w + "\n")
		}
	}
	if !utf8.ValidString(b.String()) {
		return "", ErrWerkzeugeKeinUTF8
	}
	if b.Len() > maxDateigroesse {
		return "", ErrWerkzeugeZuGross
	}
	return b.String(), nil
}

// SchreibeWerkzeugStile schreibt tools.ini im Konfigurationsordner ganz neu (ADR-0041 Punkt 4).
func SchreibeWerkzeugStile(stile []WerkzeugStil) error {
	pfad, err := werkzeugeOrt()
	if err != nil {
		return errprefix.Wrap("settings", err)
	}
	return SchreibeWerkzeugStileDatei(pfad, stile)
}

// SchreibeWerkzeugStileDatei schreibt die Datei pfad ganz neu (atomar, 0600, Ordner 0700).
func SchreibeWerkzeugStileDatei(pfad string, stile []WerkzeugStil) error {
	inhalt, err := FormatWerkzeugStile(stile)
	if err != nil {
		return errprefix.Wrap("settings", err)
	}
	return errprefix.Wrap("settings", schreibeEigeneDatei(pfad, inhalt, ErrWerkzeugeVerweis, ErrWerkzeugeKeineDatei))
}
