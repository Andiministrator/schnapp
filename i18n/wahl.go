// SPDX-License-Identifier: GPL-3.0-or-later

// Package sprache wählt die Oberflächensprache und übersetzt Texte aus gettext-PO-Dateien (ADR-0014).
//
// Ausgangssprache ist Englisch: der englische Text ist zugleich der Schlüssel (msgid) und der Rückfall, wenn eine
// Übersetzung fehlt.
package i18n

import "strings"

// Ausgangssprache ist die Sprache der Texte im Code; sie braucht keine PO-Datei.
const Ausgangssprache = "en"

// Waehle bestimmt die Oberflächensprache.
//
// Reihenfolge (ADR-0014, Nachtrag 1; wie GNU gettext):
//  1. einstellung (Einstellung in Schnapp), falls gesetzt und verfügbar,
//  2. die Locale ist die erste nicht leere der Variablen LC_ALL, LC_MESSAGES, LANG — nur diese eine zählt; ist sie
//     leer, "C" oder "C.<Kodierung>", gilt Englisch und LANGUAGE wird nicht beachtet (glibc intl/dcigettext.c,
//     guess_category_value; bei "POSIX" gilt LANGUAGE dort weiter),
//  3. sonst jeder Eintrag der Liste LANGUAGE (z. B. "de:en_US") der Reihe nach; ein Eintrag "C" oder "POSIX"
//     beendet die Liste mit Englisch,
//  4. sonst die Locale selbst, sonst Englisch.
//
// Aus "de_DE.UTF-8@euro" wird zuerst "de_DE", dann "de" gesucht.
// umgebung liefert den Wert einer Variablen (im Programm os.Getenv, im Test eine Tabelle). verfuegbar nennt die
// Sprachen mit PO-Datei; Englisch ist immer verfügbar.
func Waehle(umgebung func(string) string, einstellung string, verfuegbar []string) string {
	gibt := func(s string) bool {
		if s == Ausgangssprache {
			return true
		}
		for _, v := range verfuegbar {
			if v == s {
				return true
			}
		}
		return false
	}
	if s := passend(einstellung, gibt); s != "" {
		return s
	}
	locale := ""
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if locale = umgebung(name); locale != "" {
			break
		}
	}
	if l := strings.TrimSpace(locale); l == "" || l == "C" || strings.HasPrefix(l, "C.") {
		return Ausgangssprache
	}
	for _, eintrag := range strings.Split(umgebung("LANGUAGE"), ":") {
		if istC(eintrag) {
			return Ausgangssprache
		}
		if s := passend(eintrag, gibt); s != "" {
			return s
		}
	}
	if s := passend(locale, gibt); s != "" {
		return s
	}
	return Ausgangssprache
}

// passend zerlegt eine Locale-Angabe und liefert die erste verfügbare Form ("de_DE", dann "de"), sonst "".
func passend(angabe string, gibt func(string) bool) string {
	for _, s := range Kandidaten(angabe) {
		if gibt(s) {
			return s
		}
	}
	return ""
}

// Kandidaten zerlegt eine Locale-Angabe wie "de_DE.UTF-8@euro" in die gesuchten Sprachnamen ["de_DE", "de"].
// Leere Angaben, "C" und "POSIX" ergeben keine Kandidaten.
func Kandidaten(angabe string) []string {
	s := strings.TrimSpace(angabe)
	if i := strings.IndexAny(s, ".@"); i >= 0 {
		s = s[:i]
	}
	if s == "" || s == "C" || s == "POSIX" {
		return nil
	}
	if i := strings.IndexByte(s, '_'); i > 0 {
		return []string{s, s[:i]}
	}
	return []string{s}
}

// istC meldet, ob eine Locale-Angabe die C-Locale ist ("C", "C.UTF-8", "POSIX").
func istC(angabe string) bool {
	s := strings.TrimSpace(angabe)
	if i := strings.IndexAny(s, ".@"); i >= 0 {
		s = s[:i]
	}
	return s == "C" || s == "POSIX"
}
