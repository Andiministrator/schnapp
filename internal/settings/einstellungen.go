// SPDX-License-Identifier: GPL-3.0-or-later

// Package settings liest die Einstellungsdatei `settings.ini` (ADR-0018) und liefert typisierte Werte.
//
// Der Leser schreibt nichts und beendet nie das Programm: Eine fehlende Datei ergibt die Voreinstellungen ohne
// Warnung; eine unlesbare Datei, kaputte Zeilen, unbekannte Schlüssel und ungültige Werte ergeben Warnungen, die
// als Liste zurückkommen (fertig übersetzt, eine pro Befund). Die Aufnahme darf an einer kaputten Datei nie
// scheitern. Die übrigen Pakete bekommen die Werte übergeben und lesen die Datei nie selbst.
//
// Format: Abschnitte `[name]`, Zeilen `schlüssel = wert`, Kommentare mit `#` oder `;` am Zeilenanfang (nach
// führendem Leerraum), Leerzeilen, UTF-8; Groß-/Kleinschreibung zählt. Wird ein Schlüssel im selben Abschnitt
// mehrfach gesetzt, gilt der letzte gültige Wert (mit Warnung); ein ungültiger späterer Wert lässt den bisherigen
// Stand (Voreinstellung oder früherer gültiger Wert) unverändert.
package settings

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/errprefix"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
)

const (
	// maxDateigroesse ist die größte gelesene Dateigröße; eine größere Datei gilt als unbrauchbar (F-90).
	maxDateigroesse = 64 << 10
	// maxWarnungen ist die Zahl der ausgegebenen Warnungen; weitere werden nur gezählt (F-91).
	maxWarnungen = 20
	// maxWertLaenge ist die Länge (in Zeichen), auf die Werte aus der Datei in Warnungen gekürzt werden (F-91).
	maxWertLaenge = 80
)

// maske macht einen Text aus der Datei oder von außen für die Ausgabe auf dem Terminal unschädlich: Steuerzeichen
// (auch 0x7f und C1) und nicht druckbare Zeichen erscheinen als Escape-Folge, nie roh (F-89). Die Anführungszeichen
// von strconv.Quote entfallen, sie stehen im Meldungstext.
func maske(s string) string {
	q := strconv.Quote(s)
	return q[1 : len(q)-1]
}

// kuerze kürzt s auf höchstens maxWertLaenge Zeichen und hängt dann „…“ an (F-91).
func kuerze(s string) string {
	if r := []rune(s); len(r) > maxWertLaenge {
		return string(r[:maxWertLaenge]) + "…"
	}
	return s
}

// Bildschirmwahl ist die Einstellung `[capture] screen`: was „Bildschirm aufnehmen“ ohne Monitornamen tut
// (ADR-0017 Nachtrag 3).
type Bildschirmwahl string

// Werte von Bildschirmwahl.
const (
	WahlKlick  Bildschirmwahl = "pick"    // Monitor per Klick im Overlay wählen (Voreinstellung)
	WahlZeiger Bildschirmwahl = "pointer" // Monitor unter dem Mauszeiger, sofort
	WahlAlle   Bildschirmwahl = "all"     // alle Monitore
)

// sprachcode meldet, ob w ein Sprachcode der Form ll, lll, ll_CC oder lll_CC ist (wie in LANG ohne Kodierung).
func sprachcode(w string) bool {
	sprache, land, mitLand := strings.Cut(w, "_")
	klein := func(s string) bool {
		for _, r := range s {
			if r < 'a' || r > 'z' {
				return false
			}
		}
		return true
	}
	gross := func(s string) bool {
		for _, r := range s {
			if r < 'A' || r > 'Z' {
				return false
			}
		}
		return true
	}
	if len(sprache) < 2 || len(sprache) > 3 || !klein(sprache) {
		return false
	}
	return !mitLand || (len(land) == 2 && gross(land))
}

// Einstellungen sind die gelesenen Werte samt Voreinstellungen.
type Einstellungen struct {
	// Sprache ist [general] language: leer bei „auto“ (Sprache aus der Umgebung), sonst ein Sprachcode wie „de“ oder
	// „pt_BR“ (ADR-0018 Nachtrag 2, ADR-0014 Nachtrag 2). Gibt es die Sprache nicht, gilt die Umgebung.
	Sprache    string
	Bildschirm Bildschirmwahl // [capture] screen
	Editor     bool           // [capture] open-editor: Editor nach der Aufnahme öffnen (ADR-0020)
	// Kante ist [capture] seam: Form der Stauch-Kante neuer Dokumente (Aufnahme, Öffnen eines einfachen PNG/JPEG),
	// tear (Voreinstellung), wave oder zigzag (ADR-0023 Nachtrag 6, ADR-0018 Nachtrag 3).
	Kante      document.Kante
	Bibliothek library.Einstellung // [library] cleanup, cleanup-days
	Agenten    Agenten             // [agents] (ADR-0030 Nachtrag 2)
}

// Agenten ist der Abschnitt [agents]: Einwilligungen für den MCP-Server `schnapp mcp` (ADR-0030 Nachtrag 2). Werte
// yes/no (auch true/false). Ein ungültiger Wert lässt die Voreinstellung stehen — für Aufnehmen und Bibliothek also
// „aus“.
type Agenten struct {
	Aufnehmen   bool // capture: Agenten dürfen aufnehmen (Voreinstellung no)
	Bibliothek  bool // library: Agenten dürfen die Bibliothek durchsuchen und lesen (Voreinstellung no)
	Bestaetigen bool // confirm: vor jeder Agenten-Aufnahme ein Dialog (Voreinstellung yes)
	Bild        bool // image: das Bild geht als Bildinhalt an den Agenten (Voreinstellung yes)
	Bearbeiten  bool // edit: Agenten dürfen Dateien beschriften und schwärzen (Voreinstellung no, ADR-0030 Nachtrag 10)
}

// janein liest yes/no/true/false.
func janein(w string) (wert, ok bool) {
	switch w {
	case "yes", "true":
		return true, true
	case "no", "false":
		return false, true
	}
	return false, false
}

// Voreinstellungen liefert die Werte, die gelten, wenn nichts eingestellt ist.
func Voreinstellungen() Einstellungen {
	return Einstellungen{Bildschirm: WahlKlick, Editor: true, Kante: document.KanteAbriss, Bibliothek: library.Voreinstellung,
		Agenten: Agenten{Bestaetigen: true, Bild: true}}
}

// Pfad liefert den Ort der Datei: $XDG_CONFIG_HOME/schnapp/settings.ini, ohne gesetztes oder mit relativem
// XDG_CONFIG_HOME (laut Spezifikation ignoriert) $HOME/.config/schnapp/settings.ini. Ist auch das Heimverzeichnis
// unbekannt, kommt ein Fehler.
func Pfad() (string, error) {
	p, err := ort()
	return p, errprefix.Wrap("settings", err) // ADR-0019 Nachtrag 2, F-379
}

// ort ist Pfad ohne Paketpräfix im Fehler: Lade setzt die Ursache in eine übersetzte Warnung, dort gehört der
// Paketname nicht hin (KA-8).
func ort() (string, error) {
	if w := os.Getenv("XDG_CONFIG_HOME"); filepath.IsAbs(w) {
		return filepath.Join(filepath.Clean(w), "schnapp", "settings.ini"), nil
	}
	heim, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(heim, ".config", "schnapp", "settings.ini"), nil
}

// Lade liest die Einstellungsdatei am Ort von Pfad. Es gibt immer brauchbare Einstellungen zurück; die Warnungen
// sind fertig übersetzte Zeilen, die der Aufrufer ausgibt (der Leser selbst schreibt nichts).
func Lade() (Einstellungen, []string) {
	pfad, err := ort()
	if err != nil {
		return Voreinstellungen(), []string{fmt.Sprintf(i18n.T("Cannot locate the settings file (%v); using the defaults."), maske(err.Error()))}
	}
	return LadeDatei(pfad)
}

// LadeDatei liest die Einstellungsdatei pfad. Fehlt sie, gelten die Voreinstellungen ohne Warnung.
func LadeDatei(pfad string) (Einstellungen, []string) {
	// Ohne Blockieren öffnen: ein FIFO als Datei darf die Aufnahme nicht aufhalten (F-90).
	f, err := os.OpenFile(pfad, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return Voreinstellungen(), nil
		}
		return Voreinstellungen(), []string{fmt.Sprintf(i18n.T("%s: cannot read the settings file (%v); using the defaults."), maske(pfad), maske(err.Error()))}
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return Voreinstellungen(), []string{fmt.Sprintf(i18n.T("%s: cannot read the settings file (%v); using the defaults."), maske(pfad), maske(err.Error()))}
	}
	if !info.Mode().IsRegular() {
		return Voreinstellungen(), []string{fmt.Sprintf(i18n.T("%s: the settings file is not a regular file; using the defaults."), maske(pfad))}
	}
	daten, err := io.ReadAll(io.LimitReader(f, maxDateigroesse+1))
	if err != nil {
		return Voreinstellungen(), []string{fmt.Sprintf(i18n.T("%s: cannot read the settings file (%v); using the defaults."), maske(pfad), maske(err.Error()))}
	}
	if len(daten) > maxDateigroesse {
		return Voreinstellungen(), []string{fmt.Sprintf(i18n.T("%s: the settings file is larger than 64 KiB; using the defaults."), maske(pfad))}
	}
	return Parse(pfad, string(daten))
}

// Parse wertet den Inhalt einer Einstellungsdatei aus; name erscheint in den Warnungen.
func Parse(name, inhalt string) (Einstellungen, []string) {
	e := Voreinstellungen()
	var warn []string
	zuviele := 0 // Warnungen über der Grenze, nur gezählt
	melde := func(zeile int, format string, a ...any) {
		if len(warn) >= maxWarnungen {
			zuviele++
			return
		}
		warn = append(warn, fmt.Sprintf("%s:%d: ", maske(name), zeile)+fmt.Sprintf(format, a...))
	}

	abschnitt := ""              // aktueller Abschnitt
	imAbschnitt := false         // schon ein Abschnittskopf gesehen (gültig oder nicht)?
	bekannt := false             // ist der aktuelle Abschnitt bekannt?
	gesetzt := map[string]bool{} // "abschnitt\x00schlüssel" bereits gesehen
	inhalt = strings.TrimPrefix(inhalt, "\xef\xbb\xbf")
	for i, zeile := range strings.Split(inhalt, "\n") {
		nr := i + 1
		zeile = strings.TrimSpace(zeile)
		if zeile == "" || zeile[0] == '#' || zeile[0] == ';' {
			continue
		}
		if zeile[0] == '[' {
			if !strings.HasSuffix(zeile, "]") {
				melde(nr, "%s", i18n.T("Broken section header; its settings up to the next section are ignored."))
				abschnitt, bekannt, imAbschnitt = "", false, true // Folgezeilen still übergehen (F-92)
				continue
			}
			abschnitt = strings.TrimSpace(zeile[1 : len(zeile)-1])
			imAbschnitt = true
			bekannt = abschnitt == "general" || abschnitt == "capture" || abschnitt == "library" || abschnitt == "agents"
			if !bekannt {
				melde(nr, i18n.T("Unknown section [%s]; its keys are ignored."), maske(kuerze(abschnitt)))
			}
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
		if !bekannt {
			continue // schon mit dem Abschnitt gemeldet
		}
		kennung := abschnitt + "\x00" + schluessel
		wiederholt := gesetzt[kennung]
		var gueltig, bekannterSchluessel bool
		switch abschnitt + "." + schluessel {
		case "general.language":
			bekannterSchluessel = true
			if wert == "auto" {
				e.Sprache, gueltig = "", true
			} else if sprachcode(wert) {
				e.Sprache, gueltig = wert, true
			}
		case "capture.screen":
			bekannterSchluessel = true
			switch w := Bildschirmwahl(wert); w {
			case WahlKlick, WahlZeiger, WahlAlle:
				e.Bildschirm, gueltig = w, true
			}
		case "capture.seam":
			bekannterSchluessel = true
			if k, ok := document.KanteAus(wert); ok {
				e.Kante, gueltig = k, true
			}
		case "capture.open-editor":
			bekannterSchluessel = true
			switch wert {
			case "true":
				e.Editor, gueltig = true, true
			case "false":
				e.Editor, gueltig = false, true
			}
		case "library.cleanup":
			bekannterSchluessel = true
			switch wert {
			case "true":
				e.Bibliothek.An, gueltig = true, true
			case "false":
				e.Bibliothek.An, gueltig = false, true
			}
		case "agents.capture", "agents.library", "agents.confirm", "agents.image", "agents.edit":
			bekannterSchluessel = true
			if w, ok := janein(wert); ok {
				gueltig = true
				switch schluessel {
				case "capture":
					e.Agenten.Aufnehmen = w
				case "library":
					e.Agenten.Bibliothek = w
				case "confirm":
					e.Agenten.Bestaetigen = w
				case "image":
					e.Agenten.Bild = w
				case "edit":
					e.Agenten.Bearbeiten = w
				}
			}
		case "library.cleanup-days":
			bekannterSchluessel = true
			if n, err := strconv.Atoi(wert); err == nil && n >= 1 && n <= library.MaxTage {
				e.Bibliothek.Tage, gueltig = n, true
			}
		}
		switch {
		case !bekannterSchluessel:
			melde(nr, i18n.T("Unknown key %q in section [%s]; ignored."), kuerze(schluessel), abschnitt)
		case !gueltig:
			melde(nr, i18n.T("Invalid value %q for key %q in section [%s]; ignored."), kuerze(wert), schluessel, abschnitt)
		case wiederholt:
			melde(nr, i18n.T("Key %q in section [%s] is set again; the last value counts."), schluessel, abschnitt)
		}
		if gueltig { // nur ein gültiger Wert zählt als gesetzt (F-93)
			gesetzt[kennung] = true
		}
	}
	if zuviele > 0 {
		warn = append(warn, fmt.Sprintf(i18n.N("… and %d more warning", "… and %d more warnings", zuviele), zuviele))
	}
	return e, warn
}
