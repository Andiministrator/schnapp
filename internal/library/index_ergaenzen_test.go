// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Andiministrator/schnapp/internal/fileformat"
)

// Aufnahmen ohne Indexzeile (F-700, ADR-0016 Nachtrag 7): Ablegen an einen fehlenden Index baut ihn aus der Ablage neu
// auf, Abgleich ergänzt fehlende Zeilen.

func pfadeImIndex(t *testing.T, b *Bibliothek) map[string]bool {
	t.Helper()
	g, err := b.Laden()
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]bool{}
	for _, e := range g.Eintraege {
		if m[e.Pfad] {
			t.Errorf("doppelte Zeile für %s", e.Pfad)
		}
		m[e.Pfad] = true
	}
	return m
}

// Der übliche Start ist eine Aufnahme: Ablegen an einen beiseitegelegten Index darf ihn nicht mit nur einer Zeile
// anlegen.
func TestAblegenOhneIndexBautIhnNeuAuf(t *testing.T) {
	b, _ := neueBibliothek(t)
	alt := []Eintrag{
		ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-10-01T10:00:00Z")}),
		ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-10-02T10:00:00Z")}),
		ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-10-03T10:00:00Z")}),
	}
	if err := os.Rename(b.Orte.Index, b.Orte.Index+".beiseite"); err != nil {
		t.Fatal(err)
	}
	// Vorschau einer alten Aufnahme fehlt: der Neuaufbau beim Ablegen heilt sie nicht im Vordergrund (F-756).
	if err := os.Remove(b.VorschauPfad(alt[0].Pfad)); err != nil {
		t.Fatal(err)
	}
	neu := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-24T10:00:00Z")})
	m := pfadeImIndex(t, b)
	if len(m) != 4 || !m[neu.Pfad] {
		t.Fatalf("Index nach Ablegen ohne Index: %v, erwartet 4 Zeilen samt %s", m, neu.Pfad)
	}
	for _, e := range alt {
		if !m[e.Pfad] {
			t.Errorf("alte Aufnahme %s fehlt im Index", e.Pfad)
		}
	}
	if _, err := os.Stat(b.VorschauPfad(alt[0].Pfad)); err == nil {
		t.Error("Vorschau der alten Aufnahme im Vordergrund geheilt, Soll: fehlt")
	}
	if _, err := os.Stat(b.VorschauPfad(neu.Pfad)); err != nil {
		t.Errorf("Vorschau der neuen Aufnahme fehlt: %v", err)
	}
	// Der Abgleich danach ändert nichts mehr.
	if n, err := b.Abgleich(); err != nil || n != 0 {
		t.Errorf("Abgleich danach: %d, %v", n, err)
	}
	if len(pfadeImIndex(t, b)) != 4 {
		t.Error("Abgleich danach hat den Index verändert")
	}
}

// Die allererste Aufnahme legt den Index mit genau ihrer Zeile an.
func TestErstesAblegenLegtIndexAn(t *testing.T) {
	b, _ := neueBibliothek(t)
	e := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-10-01T10:00:00Z")})
	if m := pfadeImIndex(t, b); len(m) != 1 || !m[e.Pfad] {
		t.Fatalf("Index nach erster Aufnahme: %v", m)
	}
}

// Eine Aufnahme, deren Zeile fehlt (Absturz zwischen Datei und Zeile, ErrIndex, zurückgespielter Ordner), holt der
// Abgleich nach; danach sieht sie auch das Aufräumen.
func TestAbgleichErgaenztFehlendeZeile(t *testing.T) {
	b, _ := neueBibliothek(t)
	e1 := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-10-01T10:00:00Z")})
	e2 := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-10-02T10:00:00Z")})
	if err := b.schreibeIndex([]Eintrag{e1}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(b.VorschauPfad(e2.Pfad)); err != nil {
		t.Fatal(err)
	}
	n, err := b.Abgleich()
	if err != nil || n != 0 {
		t.Fatalf("Abgleich: entfernt %d, %v", n, err)
	}
	m := pfadeImIndex(t, b)
	if len(m) != 2 || !m[e2.Pfad] {
		t.Fatalf("Index nach Abgleich: %v, erwartet auch %s", m, e2.Pfad)
	}
	g, _ := b.Laden()
	if g.Eintraege[0].Pfad != e1.Pfad || g.Eintraege[1].Pfad != e2.Pfad {
		t.Errorf("Index nicht nach Erstellt sortiert: %v", g.Eintraege)
	}
	if !g.Eintraege[1].Erstellt.Equal(e2.Erstellt) || g.Eintraege[1].Breite != 40 || g.Eintraege[1].Hoehe != 30 {
		t.Errorf("ergänzte Zeile trägt falsche Angaben: %+v", g.Eintraege[1])
	}
	if _, err := os.Stat(b.VorschauPfad(e2.Pfad)); err != nil {
		t.Errorf("Vorschau der ergänzten Aufnahme nicht geheilt: %v", err)
	}
	erg, err := b.Aufraeumen(zeit("2027-06-01T10:00:00Z"), Einstellung{An: true, Tage: 90})
	if err != nil {
		t.Fatal(err)
	}
	if len(erg.Verschoben) != 2 {
		t.Errorf("Aufräumen verschob %d von 2 alten Aufnahmen", len(erg.Verschoben))
	}
}

// Auch ein vorhandener, aber leerer Index wird ergänzt (vorher löste nur das Fehlen der Datei den Neuaufbau aus).
func TestAbgleichFuelltLeerenIndex(t *testing.T) {
	b, _ := neueBibliothek(t)
	e := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-10-01T10:00:00Z")})
	if err := b.schreibeIndex(nil); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Abgleich(); err != nil {
		t.Fatal(err)
	}
	if m := pfadeImIndex(t, b); len(m) != 1 || !m[e.Pfad] {
		t.Errorf("leerer Index nicht ergänzt: %v", m)
	}
}

// Fremde PNG-Dateien und eigene temporäre Reste werden nie ergänzt; ohne Änderung schreibt der Abgleich nicht.
func TestAbgleichErgaenztNurArbeitsdateien(t *testing.T) {
	b, _ := neueBibliothek(t)
	e := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-10-01T10:00:00Z")})
	ordner := filepath.Dir(filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad)))
	inhalt, err := os.ReadFile(filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad)))
	if err != nil {
		t.Fatal(err)
	}
	// Ein Rest mit dem Präfix der temporären Dateien und gültigem Inhalt, dazu eine fremde Datei mit Endung .png.
	if err := os.WriteFile(filepath.Join(ordner, tempPraefix+"rest.png"), inhalt, 0o600); err != nil {
		t.Fatal(err)
	}
	schreibe(t, filepath.Join(ordner, "fremd.png"), "kein PNG")
	vorher, err := os.Stat(b.Orte.Index)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.Abgleich(); err != nil {
		t.Fatal(err)
	}
	if m := pfadeImIndex(t, b); len(m) != 1 || !m[e.Pfad] {
		t.Errorf("Index nach Abgleich: %v", m)
	}
	nachher, err := os.Stat(b.Orte.Index)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(vorher, nachher) {
		t.Error("Abgleich ohne Änderung hat den Index neu geschrieben")
	}
}

// Eine Aufnahme, die sich zwischen dem Lesen ihrer Metadaten und der Sperre ändert, wird nicht mit veralteten Angaben
// ergänzt; der nächste Abgleich holt sie.
func TestAbgleichUebergehtGeaenderteKandidaten(t *testing.T) {
	b, _ := neueBibliothek(t)
	e1 := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-10-01T10:00:00Z")})
	e2 := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-10-02T10:00:00Z")})
	if err := b.schreibeIndex([]Eintrag{e1}); err != nil {
		t.Fatal(err)
	}
	voll := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e2.Pfad))
	ks := b.ohneIndexzeile(map[string]bool{e1.Pfad: true})
	if len(ks) != 1 || ks[0].eintrag.Pfad != e2.Pfad {
		t.Fatalf("Kandidaten: %+v", ks)
	}
	// Datei ändert sich nach dem Lesen (anderer Inode wie beim Speichern).
	inhalt, _ := os.ReadFile(voll)
	if err := os.Rename(voll, voll+".alt"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(voll, inhalt, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(voll + ".alt"); err != nil {
		t.Fatal(err)
	}
	alt := ohneIndexzeileFn
	ohneIndexzeileFn = func(*Bibliothek, map[string]bool) []kandidat { return ks }
	t.Cleanup(func() { ohneIndexzeileFn = alt })
	if _, err := b.Abgleich(); err != nil {
		t.Fatal(err)
	}
	if m := pfadeImIndex(t, b); m[e2.Pfad] {
		t.Error("geänderte Datei mit veraltetem Stand ergänzt")
	}
	ohneIndexzeileFn = alt
	if _, err := b.Abgleich(); err != nil {
		t.Fatal(err)
	}
	if m := pfadeImIndex(t, b); !m[e2.Pfad] {
		t.Error("nächster Abgleich holt die geänderte Datei nicht")
	}
}

// Hat ein anderer Prozess die Zeile zwischen dem Ablaufen und der Sperre angelegt, ergänzt der Abgleich sie nicht ein
// zweites Mal.
func TestAbgleichErgaenztKeineZeileDoppelt(t *testing.T) {
	b, _ := neueBibliothek(t)
	e1 := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-10-01T10:00:00Z")})
	e2 := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-10-02T10:00:00Z")})
	if err := b.schreibeIndex([]Eintrag{e1}); err != nil {
		t.Fatal(err)
	}
	alt := ohneIndexzeileFn
	t.Cleanup(func() { ohneIndexzeileFn = alt })
	ohneIndexzeileFn = func(b *Bibliothek, bekannt map[string]bool) []kandidat {
		ks := alt(b, bekannt)
		// Der andere Prozess legt die Zeile jetzt an, vor der Sperre dieses Abgleichs.
		if err := b.schreibeIndex([]Eintrag{e1, e2}); err != nil {
			t.Fatal(err)
		}
		return ks
	}
	if _, err := b.Abgleich(); err != nil {
		t.Fatal(err)
	}
	if m := pfadeImIndex(t, b); len(m) != 2 {
		t.Errorf("Index nach Abgleich: %v", m)
	}
}
