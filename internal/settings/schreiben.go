// SPDX-License-Identifier: GPL-3.0-or-later

package settings

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

	"github.com/Andiministrator/schnapp/internal/errprefix"
)

// Aenderung ist ein Schlüssel, den der Einstellungsdialog schreibt (ADR-0034 Punkt 5).
type Aenderung struct {
	Abschnitt, Schluessel, Wert string
}

// Fehler des Schreibers. Bei jedem wird nichts geschrieben.
var (
	ErrZuGross     = errors.New("the settings file is larger than 64 KiB")
	ErrKeineDatei  = errors.New("the settings file is not a regular file")
	ErrKeinUTF8    = errors.New("the settings file is not valid UTF-8")
	ErrUngueltig   = errors.New("invalid setting")
	ErrVerweisLeer = errors.New("the settings file is a link to nothing")
)

const (
	// kopfNeueDatei ist die erste Zeile einer Datei, die der Dialog neu anlegt.
	kopfNeueDatei = "# Schnapp settings (see the documentation); edited by hand or in Settings"
	// tempPraefixName ist der Anfang des Namens der temporären Datei beim atomaren Schreiben.
	tempPraefixName = ".settings.ini.schnapp-tmp-"
)

// Werte liefert alle Schlüssel der Einstellungen in der Normalform, die der Dialog schreibt (ADR-0034 Punkt 5), in
// der Reihenfolge der Tafel. Jeder Schlüssel, den Parse kennt, steht hier (Test TestWerteDeckenParse).
func Werte(e Einstellungen) []Aenderung {
	wahr := func(b bool) string { return strconv.FormatBool(b) }
	ja := func(b bool) string {
		if b {
			return "yes"
		}
		return "no"
	}
	sprache := e.Sprache
	if sprache == "" {
		sprache = "auto"
	}
	return []Aenderung{
		{"general", "language", sprache},
		{"capture", "screen", string(e.Bildschirm)},
		{"capture", "open-editor", wahr(e.Editor)},
		{"capture", "seam", e.Kante.Name()},
		{"library", "bar", ja(e.Leiste)},
		{"library", "cleanup", wahr(e.Bibliothek.An)},
		{"library", "cleanup-days", strconv.Itoa(e.Bibliothek.Tage)},
		{"agents", "capture", ja(e.Agenten.Aufnehmen)},
		{"agents", "confirm", ja(e.Agenten.Bestaetigen)},
		{"agents", "image", ja(e.Agenten.Bild)},
		{"agents", "library", ja(e.Agenten.Bibliothek)},
		{"agents", "edit", ja(e.Agenten.Bearbeiten)},
	}
}

// Unterschiede liefert die Schlüssel, deren Wert in neu anders ist als in alt — nur diese schreibt der Dialog.
func Unterschiede(alt, neu Einstellungen) []Aenderung {
	a, n := Werte(alt), Werte(neu)
	var r []Aenderung
	for i := range n {
		if a[i].Wert != n[i].Wert {
			r = append(r, n[i])
		}
	}
	return r
}

// pruefeAenderung stellt sicher, dass der Leser den Schlüssel kennt und den Wert annimmt.
func pruefeAenderung(a Aenderung) error {
	if strings.ContainsAny(a.Abschnitt+a.Schluessel, "\r\n[]=#; \t") || strings.ContainsAny(a.Wert, "\r\n") {
		return fmt.Errorf("%w: %s.%s", ErrUngueltig, a.Abschnitt, a.Schluessel)
	}
	_, warn := Parse("", "["+a.Abschnitt+"]\n"+a.Schluessel+" = "+a.Wert+"\n")
	if len(warn) > 0 {
		return fmt.Errorf("%w: %s.%s = %q", ErrUngueltig, a.Abschnitt, a.Schluessel, a.Wert)
	}
	return nil
}

// Schreibe setzt die Schlüssel in der Einstellungsdatei am Ort von Pfad (ADR-0034 Punkt 5). Die übrigen Zeilen
// bleiben unverändert. Nichts wird halb geschrieben.
func Schreibe(aenderungen []Aenderung) error {
	pfad, err := ort()
	if err != nil {
		return errprefix.Wrap("settings", err)
	}
	return SchreibeDatei(pfad, aenderungen)
}

// SchreibeDatei ist Schreibe für die Datei pfad.
func SchreibeDatei(pfad string, aenderungen []Aenderung) error {
	return errprefix.Wrap("settings", schreibeDatei(pfad, aenderungen))
}

func schreibeDatei(pfad string, aenderungen []Aenderung) error {
	if len(aenderungen) == 0 {
		return nil
	}
	for _, a := range aenderungen {
		if err := pruefeAenderung(a); err != nil {
			return err
		}
	}
	ziel := pfad
	if info, err := os.Lstat(pfad); err == nil && info.Mode()&os.ModeSymlink != 0 {
		// Verweis (Dotfile-Sammlung): das Ziel schreiben, der Verweis bleibt.
		z, err := filepath.EvalSymlinks(pfad)
		if err != nil {
			if os.IsNotExist(err) {
				return ErrVerweisLeer
			}
			return err
		}
		ziel = z
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	inhalt, modus, gibt, err := leseZumSchreiben(ziel)
	if err != nil {
		return err
	}
	ordner := filepath.Dir(ziel)
	if !gibt {
		if err := os.MkdirAll(ordner, 0o700); err != nil {
			return err
		}
		inhalt = kopfNeueDatei + "\n"
		modus = 0o600
	}
	neu := setzeZeilen(inhalt, aenderungen)
	if len(neu) > maxDateigroesse {
		// Größer als der Leser annimmt: Danach gälten die Voreinstellungen, und jede Korrektur scheiterte (F-629).
		return ErrZuGross
	}
	return schreibeAtomar(ordner, ziel, []byte(neu), modus)
}

// leseZumSchreiben liest die Datei mit denselben Grenzen wie der Leser. gibt ist false, wenn sie fehlt.
func leseZumSchreiben(pfad string) (inhalt string, modus os.FileMode, gibt bool, err error) {
	f, err := os.OpenFile(pfad, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return "", 0, false, nil
		}
		return "", 0, false, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return "", 0, false, err
	}
	if !info.Mode().IsRegular() {
		return "", 0, false, ErrKeineDatei
	}
	daten, err := io.ReadAll(io.LimitReader(f, maxDateigroesse+1))
	if err != nil {
		return "", 0, false, err
	}
	if len(daten) > maxDateigroesse {
		return "", 0, false, ErrZuGross
	}
	if !utf8.Valid(daten) {
		return "", 0, false, ErrKeinUTF8
	}
	return string(daten), info.Mode().Perm(), true, nil
}

// schreibeAtomar schreibt daten über eine temporäre Datei im Ordner und benennt sie auf ziel um.
func schreibeAtomar(ordner, ziel string, daten []byte, modus os.FileMode) (err error) {
	tmp, err := os.CreateTemp(ordner, tempPraefixName+"*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(tmpName)
		}
	}()
	if err = tmp.Chmod(modus); err != nil {
		return err
	}
	if _, err = tmp.Write(daten); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmpName, ziel); err != nil {
		return err
	}
	if d, e := os.Open(ordner); e == nil {
		d.Sync()
		d.Close()
	}
	return nil
}

// setzeZeilen setzt die Schlüssel in inhalt (ADR-0034 Punkt 5): letzte Zeile des Schlüssels im letzten
// gleichnamigen Abschnitt ersetzen, sonst hinter die letzte Schlüsselzeile dieses Abschnitts einfügen, sonst den
// Abschnitt am Ende anhängen. Alle übrigen Zeilen bleiben Byte für Byte.
func setzeZeilen(inhalt string, aenderungen []Aenderung) string {
	nl := "\n"
	if strings.Contains(inhalt, "\r\n") {
		nl = "\r\n"
	}
	zeilen := strings.SplitAfter(inhalt, "\n")
	if n := len(zeilen); n > 0 && zeilen[n-1] == "" {
		zeilen = zeilen[:n-1]
	}
	// Gliederung wie Parse: bekannte Abschnitte, letzte Zeile je Schlüssel, letzte Schlüssel- oder Kopfzeile je
	// Abschnitt (im letzten gleichnamigen Abschnitt).
	schluesselZeile := map[string]int{}
	abschnittEnde := map[string]int{}
	abschnitt, bekannt, imAbschnitt := "", false, false
	for i, z := range zeilen {
		if i == 0 {
			z = strings.TrimPrefix(z, "\xef\xbb\xbf")
		}
		z = strings.TrimSpace(z)
		if z == "" || z[0] == '#' || z[0] == ';' {
			continue
		}
		if z[0] == '[' {
			imAbschnitt = true
			if !strings.HasSuffix(z, "]") {
				abschnitt, bekannt = "", false
				continue
			}
			abschnitt = strings.TrimSpace(z[1 : len(z)-1])
			bekannt = abschnitt == "general" || abschnitt == "capture" || abschnitt == "library" || abschnitt == "agents"
			if bekannt {
				abschnittEnde[abschnitt] = i
			}
			continue
		}
		s, _, ok := strings.Cut(z, "=")
		s = strings.TrimSpace(s)
		if !ok || s == "" || !imAbschnitt || !bekannt {
			continue
		}
		schluesselZeile[abschnitt+"\x00"+s] = i
		abschnittEnde[abschnitt] = i
	}

	ersetzt := map[int]string{}
	danach := map[int][]string{}
	var neueAbschnitte []string
	neueZeilen := map[string][]string{}
	for _, a := range aenderungen {
		zeile := a.Schluessel + " = " + a.Wert
		if i, ok := schluesselZeile[a.Abschnitt+"\x00"+a.Schluessel]; ok {
			ersetzt[i] = zeile
			continue
		}
		if i, ok := abschnittEnde[a.Abschnitt]; ok {
			danach[i] = append(danach[i], zeile)
			continue
		}
		if _, ok := neueZeilen[a.Abschnitt]; !ok {
			neueAbschnitte = append(neueAbschnitte, a.Abschnitt)
		}
		neueZeilen[a.Abschnitt] = append(neueZeilen[a.Abschnitt], zeile)
	}

	var b strings.Builder
	endeMitUmbruch := func(z string) string {
		if strings.HasSuffix(z, "\n") {
			return z
		}
		return z + nl
	}
	for i, z := range zeilen {
		if neu, ok := ersetzt[i]; ok {
			ende := ""
			switch {
			case strings.HasSuffix(z, "\r\n"):
				ende = "\r\n"
			case strings.HasSuffix(z, "\n"):
				ende = "\n"
			}
			if i == 0 && strings.HasPrefix(z, "\xef\xbb\xbf") {
				neu = "\xef\xbb\xbf" + neu
			}
			z = neu + ende
		}
		if extra, ok := danach[i]; ok {
			b.WriteString(endeMitUmbruch(z))
			for _, e := range extra {
				b.WriteString(e + nl)
			}
			continue
		}
		b.WriteString(z)
	}
	if len(neueAbschnitte) > 0 {
		s := b.String()
		if s != "" && !strings.HasSuffix(s, "\n") {
			b.WriteString(nl)
		}
		for _, ab := range neueAbschnitte {
			if b.Len() > 0 {
				b.WriteString(nl)
			}
			b.WriteString("[" + ab + "]" + nl)
			for _, z := range neueZeilen[ab] {
				b.WriteString(z + nl)
			}
		}
	}
	return b.String()
}
