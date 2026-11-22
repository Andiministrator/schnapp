// SPDX-License-Identifier: GPL-3.0-or-later

package i18n

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// aktualisieren schreibt internal/po/schnapp.pot neu: go test ./internal/i18n -run TestVorlage -update
var aktualisieren = flag.Bool("update", false, "internal/po/schnapp.pot aus den Texten im Code neu schreiben")

// vorlagePfad ist die Vorlage für Übersetzer (ADR-0014 Nachtrag 2), relativ zu diesem Paket.
var vorlagePfad = filepath.Join("..", "po", "schnapp.pot")

// vorlageKopf ist der Kopf der Vorlage: ohne Pfade, Rechner- oder Personennamen und ohne Datum (Kernprinzip 3; ein
// Datum änderte die Datei bei jedem Lauf).
const vorlageKopf = `# Translation template for Schnapp.
# SPDX-License-Identifier: GPL-3.0-or-later
#
# Generated from the i18n.T, i18n.TK and i18n.N calls in the code; do not edit by hand. Regenerate with
#   go test ./internal/i18n -run TestVorlage -update
# To start a translation, copy this file to <language>.po (for example fr.po), fill in Language and Plural-Forms
# and translate every msgstr. Entries with the context "date format" are Go time layouts (reference time
# 2006-01-02 15:04); keep the digits, change only their order and separators. See CONTRIBUTING.md.
msgid ""
msgstr ""
"Project-Id-Version: Schnapp\n"
"Language: \n"
"MIME-Version: 1.0\n"
"Content-Type: text/plain; charset=UTF-8\n"
"Content-Transfer-Encoding: 8bit\n"
"Plural-Forms: nplurals=INTEGER; plural=EXPRESSION;\n"
`

// poZeichenkette setzt s als PO-Zeichenkette in Anführungszeichen (Escapes wie C; alles andere bleibt UTF-8).
func poZeichenkette(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// baueVorlage erzeugt die Vorlage: je Text (Kontext und msgid) ein Eintrag in der Reihenfolge des ersten Vorkommens,
// mit allen Dateien als Fundstelle (ohne Zeilennummer, damit die Vorlage nicht bei jeder Codeänderung veraltet; ohne
// Testdateien, wenn der Text auch im Programm steht).
func baueVorlage(texte []text) []byte {
	type eintrag struct {
		t       text
		dateien []string
	}
	var liste []*eintrag
	nach := map[schluessel]*eintrag{}
	for _, t := range texte {
		s := schluessel{t.kontext, t.id}
		e, ok := nach[s]
		if !ok {
			e = &eintrag{t: t}
			nach[s] = e
			liste = append(liste, e)
		}
		if !slices.Contains(e.dateien, t.datei) {
			e.dateien = append(e.dateien, t.datei)
		}
	}
	var b bytes.Buffer
	b.WriteString(vorlageKopf)
	for _, e := range liste {
		// Testdateien sind keine Fundstelle für Übersetzer, außer ein Text steht nur dort.
		orte := slices.DeleteFunc(slices.Clone(e.dateien), func(d string) bool { return strings.HasSuffix(d, "_test.go") })
		if len(orte) == 0 {
			orte = e.dateien
		}
		b.WriteString("\n#: " + strings.Join(orte, " ") + "\n")
		if e.t.kontext != "" {
			b.WriteString("msgctxt " + poZeichenkette(e.t.kontext) + "\n")
		}
		b.WriteString("msgid " + poZeichenkette(e.t.id) + "\n")
		if e.t.mehrzahl != "" {
			b.WriteString("msgid_plural " + poZeichenkette(e.t.mehrzahl) + "\n")
			b.WriteString("msgstr[0] \"\"\nmsgstr[1] \"\"\n")
		} else {
			b.WriteString("msgstr \"\"\n")
		}
	}
	return b.Bytes()
}

// TestVorlage hält internal/po/schnapp.pot aktuell: Sie enthält genau die Texte im Code (ADR-0014 Nachtrag 2) und
// ist für LesePO lesbar. Weicht sie ab, schreibt -update sie neu.
func TestVorlage(t *testing.T) {
	texte, probleme, err := sammleTexte(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range probleme {
		t.Error(p)
	}
	soll := baueVorlage(texte)
	if *aktualisieren {
		if err := os.WriteFile(vorlagePfad, soll, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ist, err := os.ReadFile(vorlagePfad)
	if err != nil {
		t.Fatalf("%v (anlegen mit: go test ./internal/i18n -run TestVorlage -update)", err)
	}
	if !bytes.Equal(ist, soll) {
		t.Errorf("%s ist veraltet; neu schreiben mit: go test ./internal/i18n -run TestVorlage -update", vorlagePfad)
	}
	eintraege, err := LesePO(bytes.NewReader(ist))
	if err != nil {
		t.Fatalf("Vorlage nicht lesbar: %v", err)
	}
	// LesePO lehnt doppelte Einträge ab; dass sie gelesen wurde, belegt also auch: jeder Text genau einmal. Jeder Text im Code steht in der Vorlage, jeder Eintrag der Vorlage im Code, und alle sind leer.
	imCode := map[schluessel]string{}
	for _, tx := range texte {
		imCode[schluessel{tx.kontext, tx.id}] = tx.mehrzahl
	}
	if len(eintraege)-1 != len(imCode) {
		t.Errorf("Vorlage hat %d Einträge, im Code stehen %d verschiedene Texte", len(eintraege)-1, len(imCode))
	}
	for _, e := range eintraege[1:] {
		m, ok := imCode[schluessel{e.Kontext, e.ID}]
		if !ok || m != e.Mehrzahl {
			t.Errorf("Eintrag %q (Kontext %q) passt nicht zum Code", e.ID, e.Kontext)
		}
		for _, w := range e.Texte {
			if w != "" {
				t.Errorf("Eintrag %q ist nicht leer", e.ID)
			}
		}
	}
	// Kernprinzip 3: keine absoluten Pfade, kein Benutzerverzeichnis.
	if bytes.Contains(ist, []byte("#: /")) || bytes.Contains(ist, []byte("/home/")) {
		t.Error("Vorlage enthält absolute Pfade")
	}
}

// Escapes der Vorlage: Was poZeichenkette schreibt, liest LesePO als denselben Text.
func TestVorlageEscapes(t *testing.T) {
	for _, s := range []string{`a "b" \c`, "Zeile 1\nZeile 2\tTab\r", "Größe × 100 %"} {
		e, err := LesePO(strings.NewReader("msgid " + poZeichenkette(s) + "\nmsgstr \"\"\n"))
		if err != nil || len(e) != 1 || e[0].ID != s {
			t.Errorf("%q: %v, %+v", s, err, e)
		}
	}
}
