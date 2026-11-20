// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/fileformat"
)

// Tests zu den liegen gebliebenen temporären Dateien (`.schnapp-tmp-*`, F-274, Karte #114) und zur Nacharbeit nach dem
// Verify (Z-1 bis Z-3). Alles in eigenen Testverzeichnissen.

// F-274 (a): Scheitert das Entfernen der temporären Datei nach dem Link, ist die Aufnahme abgelegt und gültig, der
// Fehler aber nicht verschluckt (ErrTempRest).
func TestAblegenMeldetTempRest(t *testing.T) {
	b, _ := neueBibliothek(t)
	alt := entferneTemp
	entferneTemp = func(string) error { return errors.New("Entfernen scheitert") }
	t.Cleanup(func() { entferneTemp = alt })
	d := neuesDokument(t, 40, 30)
	e, err := b.Ablegen(d, d.Original(), fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")})
	if !errors.Is(err, ErrTempRest) || e.Pfad == "" {
		t.Fatalf("Eintrag %+v, Fehler %v (Soll: gültiger Eintrag und ErrTempRest)", e, err)
	}
	if g, err := fileformat.Oeffne(ablageDatei(b, e.Pfad)); err != nil || g.Fremd {
		t.Errorf("Aufnahme nicht lesbar: %v", err)
	}
	if idx, err := b.Laden(); err != nil || len(idx.Eintraege) != 1 {
		t.Errorf("Index %+v %v (Soll: Eintrag trotz Rest)", idx.Eintraege, err)
	}
}

// F-274: Ein zweiter Name derselben Aufnahme (`.schnapp-tmp-x`, Rest eines Absturzes) übersteht das Schwärzen und
// Speichern nicht: In keiner Datei des Monatsordners steht danach noch der alte Inhalt.
func TestSpeichernRaeumtTempRestMitAltemInhalt(t *testing.T) {
	b, _ := neueBibliothek(t)
	e := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")})
	voll := ablageDatei(b, e.Pfad)
	alt, err := os.ReadFile(voll)
	if err != nil {
		t.Fatal(err)
	}
	rest := filepath.Join(filepath.Dir(voll), tempPraefix+"x")
	if err := os.Link(voll, rest); err != nil {
		t.Fatal(err)
	}
	vor := time.Now().Add(-time.Hour)
	if err := os.Chtimes(rest, vor, vor); err != nil { // wirkt auf den gemeinsamen Inode
		t.Fatal(err)
	}
	neu := neuesDokument(t, 12, 9) // anderer Inhalt: „geschwärzter“ Stand
	if _, _, err := b.Speichern(e.Pfad, standIn(t, b, e.Pfad), neu, neu.Original()); err != nil {
		t.Fatalf("Speichern: %v", err)
	}
	if _, err := os.Lstat(rest); !os.IsNotExist(err) {
		t.Errorf("Rest bleibt liegen: %v", err)
	}
	eintraege, err := os.ReadDir(filepath.Dir(voll))
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range eintraege {
		roh, err := os.ReadFile(filepath.Join(filepath.Dir(voll), x.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Equal(roh, alt) {
			t.Errorf("%s enthält noch den alten Stand", x.Name())
		}
	}
}

// F-274: Lässt sich der Rest vor dem Speichern nicht entfernen, wird trotzdem gespeichert und ErrTempRest gemeldet.
func TestSpeichernMeldetTempRestOhneEntfernung(t *testing.T) {
	b, _ := neueBibliothek(t)
	e := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")})
	voll := ablageDatei(b, e.Pfad)
	rest := filepath.Join(filepath.Dir(voll), tempPraefix+"x")
	if err := os.Link(voll, rest); err != nil {
		t.Fatal(err)
	}
	vor := time.Now().Add(-time.Hour)
	if err := os.Chtimes(rest, vor, vor); err != nil {
		t.Fatal(err)
	}
	alt := entferneTemp
	entferneTemp = func(string) error { return errors.New("Entfernen scheitert") }
	t.Cleanup(func() { entferneTemp = alt })
	neu := neuesDokument(t, 12, 9)
	_, _, err := b.Speichern(e.Pfad, standIn(t, b, e.Pfad), neu, neu.Original())
	if !errors.Is(err, ErrTempRest) {
		t.Errorf("Fehler %v (Soll: ErrTempRest)", err)
	}
	if g, err := fileformat.Oeffne(voll); err != nil || g.Dokument.Groesse().X != 12 {
		t.Errorf("Speichern hat nicht geschrieben: %v", err)
	}
}

// F-274: Neuaufbau räumt alte eigene tmp-Dateien (reguläre Dateien, auch in Unterordnern), lässt junge, Symlinks,
// Aufnahmen und fremde Dateien unberührt.
func TestNeuaufbauRaeumtAlteTempReste(t *testing.T) {
	b, _ := neueBibliothek(t)
	e := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")})
	ordner := filepath.Dir(ablageDatei(b, e.Pfad))
	alt := filepath.Join(ordner, tempPraefix+"alt")
	jung := filepath.Join(ordner, tempPraefix+"jung")
	fremd := filepath.Join(ordner, "notiz.txt")
	ziel := filepath.Join(ordner, "ziel.dat")
	verweis := filepath.Join(ordner, tempPraefix+"verweis")
	unter := filepath.Join(ordner, "unter", tempPraefix+"alt")
	for _, p := range []string{alt, jung, fremd, ziel, unter} {
		schreibe(t, p, "x")
	}
	if err := os.Symlink(ziel, verweis); err != nil {
		t.Fatal(err)
	}
	vor := time.Now().Add(-time.Hour)
	for _, p := range []string{alt, unter, ziel} {
		if err := os.Chtimes(p, vor, vor); err != nil {
			t.Fatal(err)
		}
	}
	erg, err := b.Neuaufbau()
	if err != nil || erg.Aufgenommen != 1 || len(erg.TempReste) != 0 {
		t.Fatalf("Neuaufbau %+v %v", erg, err)
	}
	for _, p := range []string{alt, unter} {
		if _, err := os.Lstat(p); !os.IsNotExist(err) {
			t.Errorf("%s bleibt: %v", p, err)
		}
	}
	for _, p := range []string{jung, fremd, ziel, verweis, ablageDatei(b, e.Pfad)} {
		if _, err := os.Lstat(p); err != nil {
			t.Errorf("%s fehlt: %v", filepath.Base(p), err)
		}
	}
}

// F-274: Ein Rest, den der Neuaufbau nicht entfernen kann, steht im Ergebnis.
func TestNeuaufbauMeldetNichtEntfernteTempReste(t *testing.T) {
	b, _ := neueBibliothek(t)
	e := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")})
	rest := filepath.Join(filepath.Dir(ablageDatei(b, e.Pfad)), tempPraefix+"x")
	schreibe(t, rest, "x")
	vor := time.Now().Add(-time.Hour)
	if err := os.Chtimes(rest, vor, vor); err != nil {
		t.Fatal(err)
	}
	alt := entferneTemp
	entferneTemp = func(string) error { return errors.New("Entfernen scheitert") }
	t.Cleanup(func() { entferneTemp = alt })
	erg, err := b.Neuaufbau()
	if err != nil || len(erg.TempReste) != 1 || !strings.HasSuffix(erg.TempReste[0], tempPraefix+"x") {
		t.Errorf("Neuaufbau %+v %v (Soll: ein gemeldeter Rest)", erg, err)
	}
}
