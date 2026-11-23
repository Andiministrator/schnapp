// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/rescue"
)

// F-496: Ist die hinterlassene Kopie nicht beiseite zu legen und die Sitzung speichert unter demselben Namen, liegt die
// Kopie weiter im Weg: Die Sitzung bleibt aus, die Lage meldet nicht „geschützt“.
func TestRettungLageAusBleibtBeiGleicherDatei(t *testing.T) {
	i18n.Setze(nil)
	staat, abs1, _ := lageUmgebung(t)
	ort := filepath.Join(staat, "schnapp", "rescue")
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 40, 30)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(1, 1), document.Pt(5, 5))}); err != nil {
		t.Fatal(err)
	}
	a := rettungFuer(abs1, false)
	b := rettungFuer(abs1, false)
	defer b.Ende()
	a.Sichere(d)
	a.Ende()
	if err := os.WriteFile(filepath.Join(ort, sperrnameVon(abs1)), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(ort, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(ort, 0o700) })
	b.Sichere(d)
	b.s.Ende()
	if text, _, _ := b.Lage(); !strings.Contains(text, "could not be set aside") {
		t.Fatalf("Ausgangslage %q, erwartet „nicht beiseite zu legen“", text)
	}
	if err := os.Chmod(ort, 0o700); err != nil {
		t.Fatal(err)
	}
	b.nachSpeichern()(abs1) // dieselbe Datei
	b.s.Ende()
	text, fehler, _ := b.Lage()
	if !strings.Contains(text, "could not be set aside") || !fehler || !b.s.Lage().Aus {
		t.Errorf("Lage %q (Fehler %v, Aus %v): Die Kopie liegt weiter im Weg, die Sitzung ist nicht geschützt", text, fehler, b.s.Lage().Aus)
	}
}

// F-497: Eine beiseitegelegte Kopie steht mit vollem Dateinamen auf der Fehlerausgabe; die Statusleiste bleibt kurz.
func TestWiederherstellenBeiseiteVollerNameAufFehlerausgabe(t *testing.T) {
	i18n.Setze(nil)
	staat, abs1, _ := lageUmgebung(t)
	ort := filepath.Join(staat, "schnapp", "rescue")
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 40, 30)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(1, 1), document.Pt(5, 5))}); err != nil {
		t.Fatal(err)
	}
	a := rettungFuer(abs1, false)
	a.Sichere(d)
	a.Ende()
	if err := os.WriteFile(abs1, []byte("geaendert, und laenger als vorher"), 0o600); err != nil { // die Kopie passt nicht mehr
		t.Fatal(err)
	}
	var aus bytes.Buffer
	alt := rettungFehlerAusgabe
	rettungFehlerAusgabe = &aus
	t.Cleanup(func() { rettungFehlerAusgabe = alt })
	b := rettungFuer(abs1, false)
	defer b.Ende()
	neu, _ := document.Neu(image.NewNRGBA(image.Rect(0, 0, 40, 30)))
	text, fehler := b.Wiederherstellen(neu)
	if !fehler || !strings.Contains(text, "not restored") {
		t.Fatalf("Meldung %q (Fehler %v)", text, fehler)
	}
	eintraege, err := os.ReadDir(ort)
	if err != nil {
		t.Fatal(err)
	}
	voll := ""
	for _, e := range eintraege {
		if strings.HasSuffix(e.Name(), ".alt.json") {
			voll = e.Name()
		}
	}
	if voll == "" {
		t.Fatal("keine beiseitegelegte Kopie")
	}
	if !strings.Contains(aus.String(), "schnapp/rescue/"+voll) {
		t.Errorf("Fehlerausgabe %q enthält den vollen Namen %q nicht", aus.String(), voll)
	}
	if strings.Contains(text, voll) || !strings.Contains(text, strings.TrimSuffix(rescue.Name(abs1), ".json")[:8]+"…") {
		t.Errorf("Statusleiste %q soll kurz bleiben", text)
	}
}
