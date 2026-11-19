// SPDX-License-Identifier: GPL-3.0-or-later

package sprache

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
