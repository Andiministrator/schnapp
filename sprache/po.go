// SPDX-License-Identifier: GPL-3.0-or-later

package sprache

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Eintrag ist ein Eintrag einer PO-Datei.
type Eintrag struct {
	Kontext  string   // msgctxt, leer ohne Kontext
	ID       string   // msgid (englischer Ausgangstext)
	Mehrzahl string   // msgid_plural, leer ohne Mehrzahl
	Texte    []string // msgstr bzw. msgstr[0], msgstr[1], …
	Unscharf bool     // Markierung "#, fuzzy": Übersetzung ungeprüft, wird nicht verwendet
}

// LesePO liest eine PO-Datei. Veraltete Einträge ("#~") werden übergangen, der Kopfeintrag (leere msgid) bleibt
// in der Liste. Unterstützt werden msgctxt, msgid, msgid_plural, msgstr, msgstr[n], mehrzeilige Zeichenketten und
// die C-Escapes \n \t \" \\.
func LesePO(r io.Reader) ([]Eintrag, error) {
	var (
		eintraege []Eintrag
		akt       Eintrag
		begonnen  bool   // akt hat mindestens ein Schlüsselwort
		feld      string // zuletzt gelesenes Schlüsselwort, für Fortsetzungszeilen
		index     int    // Index bei msgstr[n]
		unscharf  bool   // "#, fuzzy" vor dem nächsten Eintrag
	)
	abschliessen := func() error {
		if begonnen {
			if feld != "msgstr" {
				return fmt.Errorf("Eintrag %q ohne msgstr", akt.ID)
			}
			eintraege = append(eintraege, akt)
		}
		akt, begonnen, feld = Eintrag{}, false, ""
		return nil
	}
	setze := func(wert string) error {
		switch feld {
		case "msgctxt":
			akt.Kontext += wert
		case "msgid":
			akt.ID += wert
		case "msgid_plural":
			akt.Mehrzahl += wert
		case "msgstr":
			for len(akt.Texte) <= index {
				akt.Texte = append(akt.Texte, "")
			}
			akt.Texte[index] += wert
		default:
			return fmt.Errorf("Zeichenkette ohne Schlüsselwort")
		}
		return nil
	}

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for nr := 1; sc.Scan(); nr++ {
		zeile := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(zeile, "#") {
			// Kommentare stehen nur vor einem Eintrag; sie beenden den vorigen.
			if feld != "veraltet" {
				if err := abschliessen(); err != nil {
					return nil, fmt.Errorf("Zeile %d: %w", nr, err)
				}
			}
			switch {
			case strings.HasPrefix(zeile, "#~"):
				// Veralteter Eintrag: wird samt seiner Markierungen übergangen.
				feld, unscharf = "veraltet", false
			case strings.HasPrefix(zeile, "#,"):
				feld = ""
				for _, f := range strings.Split(zeile[2:], ",") {
					if strings.TrimSpace(f) == "fuzzy" {
						unscharf = true
					}
				}
			default:
				feld = ""
			}
			continue
		}
		switch {
		case zeile == "":
			continue
		case strings.HasPrefix(zeile, `"`):
			if feld == "veraltet" {
				continue
			}
			wert, err := zeichenkette(zeile)
			if err != nil {
				return nil, fmt.Errorf("Zeile %d: %w", nr, err)
			}
			if err := setze(wert); err != nil {
				return nil, fmt.Errorf("Zeile %d: %w", nr, err)
			}
			continue
		}

		wort, rest, _ := strings.Cut(zeile, " ")
		neu := ""
		switch {
		case wort == "msgid" && feld == "msgctxt":
			neu = wort
		case wort == "msgctxt" || wort == "msgid":
			// Ein neuer Eintrag beginnt am Dateianfang, nach einem Kommentar oder nach einem msgstr.
			if feld != "" && feld != "msgstr" && feld != "veraltet" {
				return nil, fmt.Errorf("Zeile %d: %s nach %s", nr, wort, feld)
			}
			if feld == "msgstr" {
				if err := abschliessen(); err != nil {
					return nil, fmt.Errorf("Zeile %d: %w", nr, err)
				}
			}
			neu = wort
		case wort == "msgid_plural" && feld == "msgid":
			neu = wort
		case wort == "msgstr" && (feld == "msgid" || feld == "msgid_plural"):
			neu, index = "msgstr", 0
		case strings.HasPrefix(wort, "msgstr[") && strings.HasSuffix(wort, "]") &&
			(feld == "msgid_plural" || feld == "msgstr"):
			n, err := strconv.Atoi(wort[len("msgstr[") : len(wort)-1])
			if err != nil || n < 0 || n > 16 {
				return nil, fmt.Errorf("Zeile %d: ungültiger Index %q", nr, wort)
			}
			neu, index = "msgstr", n
		default:
			return nil, fmt.Errorf("Zeile %d: unerwartet %q", nr, wort)
		}
		if !begonnen {
			akt.Unscharf, unscharf = unscharf, false
			begonnen = true
		}
		feld = neu
		wert, err := zeichenkette(strings.TrimSpace(rest))
		if err != nil {
			return nil, fmt.Errorf("Zeile %d: %w", nr, err)
		}
		if err := setze(wert); err != nil {
			return nil, fmt.Errorf("Zeile %d: %w", nr, err)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if err := abschliessen(); err != nil {
		return nil, fmt.Errorf("Dateiende: %w", err)
	}
	return eintraege, nil
}

// zeichenkette liest eine PO-Zeichenkette in Anführungszeichen.
func zeichenkette(s string) (string, error) {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return "", fmt.Errorf("Zeichenkette erwartet: %q", s)
	}
	innen := s[1 : len(s)-1]
	var b strings.Builder
	for i := 0; i < len(innen); i++ {
		c := innen[i]
		if c == '"' {
			return "", fmt.Errorf("unmaskiertes Anführungszeichen in %q", s)
		}
		if c != '\\' {
			b.WriteByte(c)
			continue
		}
		i++
		if i == len(innen) {
			return "", fmt.Errorf("Escape am Ende von %q", s)
		}
		switch innen[i] {
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case '"':
			b.WriteByte('"')
		case '\\':
			b.WriteByte('\\')
		default:
			return "", fmt.Errorf("unbekanntes Escape \\%c in %q", innen[i], s)
		}
	}
	return b.String(), nil
}
