// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/library"
)

// ADR-0016 Nachtrag 10 (Karte #345) und Nachtrag 11 (Karte #348, F-825): Eine Datei von außen wird mit dem ersten
// Speichern (der Editor speichert sie bei der ersten Änderung) eine Aufnahme der Bibliothek; ohne Änderung bleibt die
// Bibliothek leer. Ein früher übernommener Inhalt öffnet gleich seine Aufnahme.

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

func TestOeffnenUebernimmtErstBeimSpeichern(t *testing.T) {
	_, g, b := importLauf(t)
	d0, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 40, 30)))
	if err != nil {
		t.Fatal(err)
	}
	mitKasten(t, d0)
	arbeit := schreibeDatei(t, t.TempDir(), "mit-kasten.png", d0) // Arbeitsdatei von außen mit einem Objekt
	abs, _ := filepath.Abs(arbeit)
	vorher, _ := os.ReadFile(arbeit)
	if err := oeffneImEditor(arbeit, false, document.KanteWelle); err != nil {
		t.Fatal(err)
	}
	// Ohne Änderung: die Datei selbst, nichts in der Bibliothek, der Editor soll bei der ersten Änderung speichern.
	if g.geoeffnet.Pfad() != abs || !g.geoeffnet.Aussen || aufnahmen(t, b) != 0 {
		t.Fatalf("beim Öffnen: Pfad %q, Aussen %v, %d Aufnahmen", g.geoeffnet.Pfad(), g.geoeffnet.Aussen, aufnahmen(t, b))
	}
	if len(g.dok.Objekte()) != 1 {
		t.Errorf("Objekte der Arbeitsdatei fehlen: %d", len(g.dok.Objekte()))
	}
	// Das Speichern nach der ersten Änderung legt sie als Aufnahme ab und merkt den Inhalt.
	if _, err := g.speichern(g.dok); err != nil {
		t.Fatal(err)
	}
	rel, ok := b.Relativ(g.geoeffnet.Pfad())
	if !ok || aufnahmen(t, b) != 1 {
		t.Fatalf("nach dem Speichern: Pfad %q, %d Aufnahmen", g.geoeffnet.Pfad(), aufnahmen(t, b))
	}
	if nachher, _ := os.ReadFile(arbeit); !bytes.Equal(vorher, nachher) {
		t.Error("die Datei von außen wurde verändert")
	}
	g.rettung.Ende()
	// Dieselbe unveränderte Datei: die vorhandene Aufnahme samt Objekt, keine zweite, kein erneutes Speichern.
	if err := oeffneImEditor(arbeit, false, document.KanteWelle); err != nil {
		t.Fatal(err)
	}
	if r2, _ := b.Relativ(g.geoeffnet.Pfad()); r2 != rel || g.geoeffnet.Aussen || aufnahmen(t, b) != 1 || len(g.dok.Objekte()) != 1 {
		t.Errorf("zweites Öffnen: %q statt %q, Aussen %v, %d Aufnahmen, %d Objekte", r2, rel, g.geoeffnet.Aussen,
			aufnahmen(t, b), len(g.dok.Objekte()))
	}
	g.rettung.Ende()
	// Ein einfaches Bild über die Ladefunktion von Datei → Öffnen…: nicht übernommen, Kantenvorgabe gesetzt.
	einfach := schreibeEinfachesPNG(t, t.TempDir(), "einfach.png")
	geo, err := oeffnerFuer(b, document.KanteWelle)(einfach)
	if err != nil {
		t.Fatal(err)
	}
	einfachAbs, _ := filepath.Abs(einfach)
	if geo.Pfad() != einfachAbs || !geo.Aussen || aufnahmen(t, b) != 1 || geo.Dokument.Kante() != document.KanteWelle {
		t.Errorf("Datei → Öffnen…: Pfad %q, Aussen %v, %d Aufnahmen, Kante %v", geo.Pfad(), geo.Aussen, aufnahmen(t, b), geo.Dokument.Kante())
	}
	if _, err := geo.Speichern(geo.Dokument); err != nil {
		t.Fatal(err)
	}
	if _, ok := b.Relativ(geo.Pfad()); !ok || aufnahmen(t, b) != 2 {
		t.Errorf("Datei → Öffnen… gespeichert: Pfad %q, %d Aufnahmen", geo.Pfad(), aufnahmen(t, b))
	}
	geo.Rettung.Ende()
	// Eine Aufnahme der Ablage kommt nicht von außen.
	aufnahme := filepath.Join(b.Orte.Ablage, filepath.FromSlash(rel))
	if err := oeffneImEditor(aufnahme, false, document.KanteWelle); err != nil {
		t.Fatal(err)
	}
	if g.geoeffnet.Aussen || aufnahmen(t, b) != 2 {
		t.Errorf("Aufnahme der Ablage: Aussen %v, %d Aufnahmen", g.geoeffnet.Aussen, aufnahmen(t, b))
	}
	g.rettung.Ende()
	// Liegt die übernommene Aufnahme nicht mehr in der Ablage (Papierkorb), öffnet sich wieder die Datei selbst.
	if _, err := b.InPapierkorb(rel, b.Jetzt()); err != nil {
		t.Fatal(err)
	}
	if err := oeffneImEditor(arbeit, false, document.KanteWelle); err != nil {
		t.Fatal(err)
	}
	if g.geoeffnet.Pfad() != abs || !g.geoeffnet.Aussen {
		t.Errorf("nach dem Papierkorb: %q, Aussen %v", g.geoeffnet.Pfad(), g.geoeffnet.Aussen)
	}
}

// Ohne Übernahme (Tests des Rückfalls, Drehbuch-Lauf) speichert der Editor nicht bei der ersten Änderung.
func TestOhneUebernahmeKeinSpeichernBeiAenderung(t *testing.T) {
	aussen, g, b := importLauf(t)
	importBeimOeffnen = false
	if err := oeffneImEditor(aussen, false, document.KanteWelle); err != nil {
		t.Fatal(err)
	}
	if g.geoeffnet.Aussen || aufnahmen(t, b) != 0 {
		t.Errorf("Aussen %v, %d Aufnahmen", g.geoeffnet.Aussen, aufnahmen(t, b))
	}
}

// Kritik K-1 (ADR-0016 Nachtrag 10 Punkt 6): Wird in der übernommenen Aufnahme geschwärzt, warnt das Speichern, dass
// die Datei von außen das Geschwärzte noch zeigt — wie vor der Übernahme (ADR-0022 Punkt 4, F-221). Auch beim
// Wiederfinden über die Summe.
func TestUebernahmeWarntVorUngeschwaerztemOriginal(t *testing.T) {
	aussen, g, _ := importLauf(t)
	for runde := range 2 {
		if err := oeffneImEditor(aussen, false, document.KanteWelle); err != nil {
			t.Fatal(err)
		}
		if err := g.dok.Schwaerzen(image.Rect(2, 2, 12+runde, 12), document.Schwaerzung{Farbe: color.NRGBA{A: 255}}); err != nil {
			t.Fatal(err)
		}
		erg, err := g.speichern(g.dok)
		if err != nil {
			t.Fatal(err)
		}
		// Runde 0 legt die Datei von außen ab („The original file is unchanged …“), Runde 1 findet die Aufnahme über die
		// Summe und nennt die Datei von außen mit Namen.
		nennt := "original file"
		if runde == 1 {
			nennt = filepath.Base(aussen)
		}
		if !erg.Warnung || !strings.Contains(erg.Hinweis, nennt) || !strings.Contains(erg.Hinweis, "unredacted") {
			t.Errorf("Runde %d: keine Warnung zur Datei von außen: %q", runde, erg.Hinweis)
		}
		g.rettung.Ende()
	}
}

// Kritik K-3: Eine FIFO oder ein Gerät wird nicht gehasht (blockierte sonst); der gewöhnliche Weg meldet den Fehler.
func TestUebernahmeNurRegulaereDatei(t *testing.T) {
	_, _, b := importLauf(t)
	fifo := filepath.Join(t.TempDir(), "rohr")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Skip(err)
	}
	fertig := make(chan string, 1)
	go func() {
		p, _ := vorOeffnen(b, fifo)
		fertig <- p
	}()
	select {
	case p := <-fertig:
		if p != fifo {
			t.Errorf("FIFO umgelenkt: %q", p)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("vorOeffnen blockiert an einer FIFO")
	}
}
