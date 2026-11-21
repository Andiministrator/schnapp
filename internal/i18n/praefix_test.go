// SPDX-License-Identifier: GPL-3.0-or-later

package i18n

import (
	"strings"
	"testing"
	"testing/fstest"
)

// einmalPraefix prüft die Fehlertext-Regel (ADR-0019 Nachtrag 2, F-339): Präfix "i18n: " genau einmal am Anfang.
func einmalPraefix(t *testing.T, name string, err error) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: kein Fehler", name)
		return
	}
	s := err.Error()
	if !strings.HasPrefix(s, "i18n: ") || strings.Count(s, "i18n:") != 1 {
		t.Errorf("%s: Fehlertext %q ohne genau ein Präfix \"i18n: \"", name, s)
	}
}

func TestFehlertextPraefix(t *testing.T) {
	// PO-Leser: Fehler in der Zeile, am Dateiende, beim Escape.
	for name, text := range map[string]string{
		"msgid ohne msgstr":  "msgid \"a\"\n",
		"Dateiende":          "msgid \"a\"\nmsgstr \"b\"\nmsgid \"c\"\n",
		"Zeichenkette lose":  "\"lose\"\n",
		"unbekanntes Escape": "msgid \"a\\q\"\nmsgstr \"b\"\n",
		"unbekanntes Wort":   "msgfoo \"a\"\n",
		"Index ungültig":     "msgid \"a\"\nmsgid_plural \"b\"\nmsgstr[x] \"c\"\n",
		"Datei zu groß":      strings.Repeat("# x\n", maxPO/4+10),
		"Eintrag doppelt":    "msgid \"a\"\nmsgstr \"b\"\n\nmsgid \"a\"\nmsgstr \"c\"\n",
	} {
		_, err := LesePO(strings.NewReader(text))
		einmalPraefix(t, "LesePO "+name, err)
	}
	// Mehrzahlregel.
	for name, kopf := range map[string]string{
		"nplurals": "nplurals=x; plural=n;", "unvollständig": "nplurals=2;", "unerwartet": "nplurals=2; plural=n n;",
		"Klammer": "nplurals=2; plural=(n;", "Doppelpunkt": "nplurals=2; plural=n?1;",
	} {
		_, err := LeseMehrzahlregel(kopf)
		einmalPraefix(t, "LeseMehrzahlregel "+name, err)
	}
	// Katalog: Kopf mit kaputter Regel und kaputte Datei über Lade/Starte; fehlende Datei.
	kaputtKopf := "msgid \"\"\nmsgstr \"Plural-Forms: nplurals=x; plural=n;\\n\"\n"
	_, err := NeuerKatalog("de", mustPO(t, kaputtKopf))
	einmalPraefix(t, "NeuerKatalog", err)
	fsys := fstest.MapFS{"de.po": {Data: []byte("msgid \"a\"\n")}, "fr.po": {Data: []byte(kaputtKopf)}}
	_, err = Lade(fsys, "de")
	einmalPraefix(t, "Lade PO-Fehler", err)
	_, err = Lade(fsys, "fr")
	einmalPraefix(t, "Lade Kopf-Fehler", err)
	_, err = Lade(fsys, "es")
	einmalPraefix(t, "Lade fehlende Datei", err)
	_, err = Starte(fsys, "de")
	einmalPraefix(t, "Starte", err)
	Setze(nil)
}

func mustPO(t *testing.T, text string) []Eintrag {
	t.Helper()
	e, err := LesePO(strings.NewReader(text))
	if err != nil {
		t.Fatal(err)
	}
	return e
}
