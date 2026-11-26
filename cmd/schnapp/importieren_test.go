// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/library"
)

// ADR-0016 Nachtrag 10 (Karte #345): Eine Datei von außen wird beim Öffnen eine Aufnahme der Bibliothek.

// importLauf schaltet die Übernahme für den Test ein (TestMain schaltet sie für die Tests des Rückfalls ab).
func importLauf(t *testing.T) (aussen string, g *gestartet, b *library.Bibliothek) {
	t.Helper()
	_, aussen, g = rettungsLauf(t)
	importBeimOeffnen = true
	t.Cleanup(func() { importBeimOeffnen = false })
	orte, err := library.OrteAus(os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	return aussen, g, library.Neu(orte)
}

// aufnahmen zählt die Aufnahmen in der Ablage.
func aufnahmen(t *testing.T, b *library.Bibliothek) int {
	t.Helper()
	n := 0
	_ = filepath.WalkDir(b.Orte.Ablage, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && filepath.Ext(p) == ".png" {
			n++
		}
		return nil
	})
	return n
}

func TestOeffnenUebernimmtDateiVonAussen(t *testing.T) {
	aussen, g, b := importLauf(t)
	d0, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 40, 30)))
	if err != nil {
		t.Fatal(err)
	}
	mitKasten(t, d0)
	arbeit := schreibeDatei(t, t.TempDir(), "mit-kasten.png", d0) // Arbeitsdatei von außen mit einem Objekt
	vorher, _ := os.ReadFile(arbeit)
	if err := oeffneImEditor(arbeit, false, document.KanteWelle); err != nil {
		t.Fatal(err)
	}
	rel, ok := b.Relativ(g.geoeffnet.Pfad())
	if !ok || aufnahmen(t, b) != 1 {
		t.Fatalf("nicht übernommen: Pfad %q, %d Aufnahmen", g.geoeffnet.Pfad(), aufnahmen(t, b))
	}
	if len(g.dok.Objekte()) != 1 {
		t.Errorf("Objekte der Arbeitsdatei nicht übernommen: %d", len(g.dok.Objekte()))
	}
	if nachher, _ := os.ReadFile(arbeit); !bytes.Equal(vorher, nachher) {
		t.Error("die Datei von außen wurde verändert")
	}
	// Dieselbe unveränderte Datei: die vorhandene Aufnahme, keine zweite.
	if err := oeffneImEditor(arbeit, false, document.KanteWelle); err != nil {
		t.Fatal(err)
	}
	if r2, _ := b.Relativ(g.geoeffnet.Pfad()); r2 != rel || aufnahmen(t, b) != 1 {
		t.Errorf("zweites Öffnen: %q statt %q, %d Aufnahmen", r2, rel, aufnahmen(t, b))
	}
	// Ein einfaches Bild über die Ladefunktion von Datei → Öffnen…: übernommen mit der Kantenvorgabe.
	einfach := schreibeEinfachesPNG(t, t.TempDir(), "einfach.png")
	geo, err := oeffnerFuer(b, document.KanteWelle)(einfach)
	if err != nil {
		t.Fatal(err)
	}
	defer geo.Rettung.Ende()
	if _, ok := b.Relativ(geo.Pfad()); !ok || aufnahmen(t, b) != 2 || geo.Dokument.Kante() != document.KanteWelle {
		t.Errorf("Datei → Öffnen…: Pfad %q, %d Aufnahmen, Kante %v", geo.Pfad(), aufnahmen(t, b), geo.Dokument.Kante())
	}
	// Eine Aufnahme der Ablage wird nicht noch einmal übernommen.
	if err := oeffneImEditor(g.geoeffnet.Pfad(), false, document.KanteWelle); err != nil {
		t.Fatal(err)
	}
	if aufnahmen(t, b) != 2 {
		t.Errorf("Aufnahme der Ablage übernommen: %d", aufnahmen(t, b))
	}
	// Liegt die übernommene Aufnahme nicht mehr in der Ablage (Papierkorb), wird neu übernommen.
	if _, err := b.InPapierkorb(rel, b.Jetzt()); err != nil {
		t.Fatal(err)
	}
	if err := oeffneImEditor(arbeit, false, document.KanteWelle); err != nil {
		t.Fatal(err)
	}
	// (Der Name ist sekundengenau und darf derselbe sein wie der der weggeworfenen.)
	if _, ok := b.Relativ(g.geoeffnet.Pfad()); !ok || aufnahmen(t, b) != 2 {
		t.Errorf("nach dem Papierkorb: %q, %d Aufnahmen", g.geoeffnet.Pfad(), aufnahmen(t, b))
	}
	if _, err := os.Stat(g.geoeffnet.Pfad()); err != nil {
		t.Errorf("neu übernommene Aufnahme fehlt: %v", err)
	}
	_ = aussen
}

// Scheitert die Übernahme (Ablage nicht anlegbar), öffnet der Editor die Datei von außen wie bisher.
func TestOeffnenOhneUebernahmeWieBisher(t *testing.T) {
	aussen, g, b := importLauf(t)
	if err := os.MkdirAll(filepath.Dir(b.Orte.Ablage), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b.Orte.Ablage, []byte("keine Ablage"), 0o600); err != nil { // eine Datei, wo der Ordner hingehört
		t.Fatal(err)
	}
	if err := oeffneImEditor(aussen, false, document.KanteWelle); err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(aussen)
	if g.geoeffnet.Pfad() != abs {
		t.Errorf("Pfad %q, Soll die Datei von außen %q", g.geoeffnet.Pfad(), abs)
	}
}
