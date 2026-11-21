// SPDX-License-Identifier: GPL-3.0-or-later

package i18n

import (
	"reflect"
	"strings"
	"testing"
)

const probePO = `# Kommentar
msgid ""
msgstr ""
"Language: de\n"
"Plural-Forms: nplurals=2; plural=(n != 1);\n"

#: datei.go:10
msgid "Save"
msgstr "Speichern"

msgctxt "menu"
msgid "Open"
msgstr "Öffnen"

#, fuzzy
msgid "Close"
msgstr "Schliessen"

msgid ""
"Two "
"lines"
msgstr "Zwei\n"
"Zeilen \"zitiert\" \\ \t."

msgid "%d image"
msgid_plural "%d images"
msgstr[0] "%d Bild"
msgstr[1] "%d Bilder"

#, fuzzy
#~ msgid "Old"
#~ msgstr "Alt"

msgid "Empty"
msgstr ""
`

func TestLesePO(t *testing.T) {
	got, err := LesePO(strings.NewReader(probePO))
	if err != nil {
		t.Fatal(err)
	}
	want := []Eintrag{
		{ID: "", Texte: []string{"Language: de\nPlural-Forms: nplurals=2; plural=(n != 1);\n"}},
		{ID: "Save", Texte: []string{"Speichern"}},
		{Kontext: "menu", ID: "Open", Texte: []string{"Öffnen"}},
		{ID: "Close", Texte: []string{"Schliessen"}, Unscharf: true},
		{ID: "Two lines", Texte: []string{"Zwei\nZeilen \"zitiert\" \\ \t."}},
		{ID: "%d image", Mehrzahl: "%d images", Texte: []string{"%d Bild", "%d Bilder"}},
		// Die fuzzy-Markierung des veralteten Eintrags "Old" darf nicht auf "Empty" übergehen.
		{ID: "Empty", Texte: []string{""}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("LesePO:\n got %#v\nwant %#v", got, want)
	}
}

func TestLesePOFehler(t *testing.T) {
	for name, text := range map[string]string{
		"msgid ohne msgstr":              "msgid \"a\"\n",
		"msgid ohne msgstr, dann neuer":  "msgid \"a\"\nmsgid \"b\"\nmsgstr \"c\"\n",
		"Kommentar mitten im Eintrag":    "msgid \"a\"\n# x\nmsgstr \"b\"\n",
		"msgctxt nach msgid":             "msgid \"a\"\nmsgctxt \"k\"\n",
		"Zeichenkette ohne Wort":         "\"lose\"\n",
		"Anführungszeichen fehlt":        "msgid \"a\nmsgstr \"b\"\n",
		"unmaskiertes Anführungszeichen": "msgid \"a\"b\"\nmsgstr \"c\"\n",
		"unbekanntes Escape":             "msgid \"a\\q\"\nmsgstr \"b\"\n",
		"Escape am Ende":                 "msgid \"a\\\"\nmsgstr \"b\"\n",
		"unbekanntes Wort":               "msgfoo \"a\"\n",
		"Index ungültig":                 "msgid \"a\"\nmsgid_plural \"b\"\nmsgstr[x] \"c\"\n",
		"msgstr[n] ohne Mehrzahl":        "msgid \"a\"\nmsgstr[0] \"c\"\n",
	} {
		if _, err := LesePO(strings.NewReader(text)); err == nil {
			t.Errorf("%s: kein Fehler", name)
		}
	}
}

// Byte-Order-Mark und alle C-Escapes (F-37).
func TestLesePOEscapesUndBOM(t *testing.T) {
	text := "\uFEFFmsgid \"a\"\nmsgstr \"\\r\\a\\b\\f\\v|\\101\\0|\\x41\\x4a\\x041|\\7z\"\n"
	got, err := LesePO(strings.NewReader(text))
	if err != nil {
		t.Fatal(err)
	}
	if want := "\r\a\b\f\v|A\x00|AJA|\x07z"; len(got) != 1 || got[0].Texte[0] != want {
		t.Errorf("got %#v, want %q", got, want)
	}
}

func TestLesePOFehlerStreng(t *testing.T) {
	for name, text := range map[string]string{
		"doppelt":                      "msgid \"a\"\nmsgstr \"b\"\n\nmsgid \"a\"\nmsgstr \"c\"\n",
		"doppelt mit Kontext":          "msgctxt \"k\"\nmsgid \"a\"\nmsgstr \"b\"\n\nmsgctxt \"k\"\nmsgid \"a\"\nmsgstr \"c\"\n",
		"Mehrzahl mit msgstr ohne [n]": "msgid \"a\"\nmsgid_plural \"b\"\nmsgstr \"c\"\n",
		"Oktal zu groß":                "msgid \"a\"\nmsgstr \"\\777\"\n",
		"\\x ohne Ziffer":              "msgid \"a\"\nmsgstr \"\\xg\"\n",
		"Hex zu groß (drei Ziffern)":   "msgid \"a\"\nmsgstr \"\\x414\"\n",
		"zu groß":                      "msgid \"a\"\nmsgstr \"\"\n" + strings.Repeat("\""+strings.Repeat("x", 1000)+"\"\n", 9000),
	} {
		if _, err := LesePO(strings.NewReader(text)); err == nil {
			t.Errorf("%s: kein Fehler", name)
		}
	}
	// Gleicher Text in verschiedenen Kontexten ist erlaubt.
	if _, err := LesePO(strings.NewReader("msgid \"a\"\nmsgstr \"b\"\n\nmsgctxt \"k\"\nmsgid \"a\"\nmsgstr \"c\"\n")); err != nil {
		t.Error(err)
	}
}
