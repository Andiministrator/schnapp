// SPDX-License-Identifier: GPL-3.0-or-later

package i18n

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
// in der Liste. Unterstützt werden msgctxt, msgid, msgid_plural, msgstr, msgstr[n], mehrzeilige Zeichenketten, die
// C-Escapes (\n \t \r \a \b \f \v \" \\, oktal, \x) und ein Byte-Order-Mark am Anfang. Strenger als msgfmt:
// Kommentare stehen nur vor einem Eintrag, nicht zwischen msgid und msgstr. Fehler sind außerdem doppelte Einträge,
// msgid_plural ohne msgstr[n] und Dateien über 8 MiB.
func LesePO(r io.Reader) ([]Eintrag, error) {
	var (
		eintraege []Eintrag
		akt       Eintrag
		begonnen  bool   // akt hat mindestens ein Schlüsselwort
		feld      string // zuletzt gelesenes Schlüsselwort, für Fortsetzungszeilen
		index     int    // Index bei msgstr[n]
		unscharf  bool   // "#, fuzzy" vor dem nächsten Eintrag
		gesehen   = map[[2]string]bool{}
		groesse   int
	)
	abschliessen := func() error {
		if begonnen {
			if feld != "msgstr" {
				return fmt.Errorf("Eintrag %q ohne msgstr", akt.ID)
			}
			s := [2]string{akt.Kontext, akt.ID}
			if gesehen[s] {
				return fmt.Errorf("Eintrag %q (Kontext %q) doppelt", akt.ID, akt.Kontext)
			}
			gesehen[s] = true
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
		roh := sc.Text()
		if groesse += len(roh) + 1; groesse > maxPO {
			return nil, fmt.Errorf("Datei größer als %d Bytes", maxPO)
		}
		if nr == 1 {
			roh = strings.TrimPrefix(roh, "\uFEFF") // Byte-Order-Mark mancher Editoren
		}
		zeile := strings.TrimSpace(roh)
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
		case wort == "msgstr" && feld == "msgid":
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
		case 'r':
			b.WriteByte('\r')
		case 'a':
			b.WriteByte('\a')
		case 'b':
			b.WriteByte('\b')
		case 'f':
			b.WriteByte('\f')
		case 'v':
			b.WriteByte('\v')
		case '0', '1', '2', '3', '4', '5', '6', '7':
			// Oktal, höchstens drei Ziffern (C).
			wert, n := 0, 0
			for n < 3 && i+n < len(innen) && innen[i+n] >= '0' && innen[i+n] <= '7' {
				wert = wert*8 + int(innen[i+n]-'0')
				n++
			}
			if wert > 0xff {
				return "", fmt.Errorf("Oktal-Escape zu groß in %q", s)
			}
			b.WriteByte(byte(wert))
			i += n - 1
		case 'x':
			// Hexadezimal, beliebig viele Ziffern wie in C; der Wert muss in ein Byte passen.
			wert, n := 0, 0
			for i+1+n < len(innen) && hexwert(innen[i+1+n]) >= 0 {
				if wert = wert*16 + hexwert(innen[i+1+n]); wert > 0xff {
					return "", fmt.Errorf("Hex-Escape zu groß in %q", s)
				}
				n++
			}
			if n == 0 {
				return "", fmt.Errorf("\\x ohne Ziffer in %q", s)
			}
			b.WriteByte(byte(wert))
			i += n
		default:
			return "", fmt.Errorf("unbekanntes Escape \\%c in %q", innen[i], s)
		}
	}
	return b.String(), nil
}

// hexwert liefert den Wert einer Hex-Ziffer oder -1.
func hexwert(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}

// maxPO ist die größte erlaubte PO-Datei (eingebettete Dateien sind weit kleiner; schützt vor fremden Dateien).
const maxPO = 8 << 20
