// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/rescue"
)

// F-831: Fälle der Übernahme, die TestMain sonst abschaltet (ADR-0016 Nachtrag 11).

// Punkt 9: --ignore-limit mit einer Datei von außen öffnet sie ohne Kopie und ohne Budget; das erste Speichern legt sie
// als Aufnahme ab, das Budget bleibt aus.
func TestUebernahmeMitIgnoreLimit(t *testing.T) {
	aussen, g, b := importLauf(t)
	abs, _ := filepath.Abs(aussen)
	if err := oeffneImEditor(aussen, true, document.KanteWelle); err != nil {
		t.Fatal(err)
	}
	if g.geoeffnet.Pfad() != abs || !g.geoeffnet.Aussen || !g.geoeffnet.OhneGrenze || aufnahmen(t, b) != 0 {
		t.Fatalf("beim Öffnen: Pfad %q, Aussen %v, OhneGrenze %v, %d Aufnahmen", g.geoeffnet.Pfad(), g.geoeffnet.Aussen,
			g.geoeffnet.OhneGrenze, aufnahmen(t, b))
	}
	mitKasten(t, g.dok)
	if _, err := g.speichern(g.dok); err != nil {
		t.Fatal(err)
	}
	if _, ok := b.Relativ(g.geoeffnet.Pfad()); !ok || aufnahmen(t, b) != 1 {
		t.Errorf("nach dem Speichern: Pfad %q, %d Aufnahmen", g.geoeffnet.Pfad(), aufnahmen(t, b))
	}
}

// Punkt 2: Scheitert das erste Speichern (Ablage nicht beschreibbar), bleibt die Datei von außen offen und unverändert,
// nichts liegt in der Bibliothek, der Inhalt ist nicht als übernommen gemerkt; das nächste gewöhnliche Speichern legt ab.
func TestUebernahmeErstesSpeichernScheitert(t *testing.T) {
	aussen, g, b := importLauf(t)
	abs, _ := filepath.Abs(aussen)
	vorher, _ := os.ReadFile(aussen)
	if err := os.MkdirAll(filepath.Dir(b.Orte.Ablage), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b.Orte.Ablage, []byte("keine Ablage"), 0o600); err != nil { // eine Datei, wo der Ordner hingehört
		t.Fatal(err)
	}
	if err := oeffneImEditor(aussen, false, document.KanteWelle); err != nil {
		t.Fatal(err)
	}
	mitKasten(t, g.dok)
	if _, err := g.speichern(g.dok); err == nil {
		t.Fatal("Speichern trotz fehlender Ablage gelungen")
	}
	if g.geoeffnet.Pfad() != abs {
		t.Errorf("nach dem Fehlschlag: Pfad %q, Soll die Datei von außen", g.geoeffnet.Pfad())
	}
	if nachher, _ := os.ReadFile(aussen); string(nachher) != string(vorher) {
		t.Error("die Datei von außen wurde verändert")
	}
	if summe, err := dateiSumme(abs); err != nil {
		t.Fatal(err)
	} else if _, ok := b.Importiert(summe); ok {
		t.Error("Inhalt nach gescheitertem Speichern als übernommen gemerkt")
	}
	if err := os.Remove(b.Orte.Ablage); err != nil {
		t.Fatal(err)
	}
	if _, err := g.speichern(g.dok); err != nil {
		t.Fatal(err)
	}
	if _, ok := b.Relativ(g.geoeffnet.Pfad()); !ok || aufnahmen(t, b) != 1 {
		t.Errorf("zweites Speichern: Pfad %q, %d Aufnahmen", g.geoeffnet.Pfad(), aufnahmen(t, b))
	}
	if summe, _ := dateiSumme(abs); func() bool { _, ok := b.Importiert(summe); return !ok }() {
		t.Error("Inhalt nach dem zweiten Speichern nicht als übernommen gemerkt")
	}
}

// Punkt 4: Die Rettungskopie gilt bis zur ersten Änderung für die Datei von außen — eine liegende wird beim Öffnen
// angewendet, ohne dass etwas in die Bibliothek kommt —, danach für die Aufnahme.
func TestUebernahmeRettungskopieVorUndNach(t *testing.T) {
	aussen, g, b := importLauf(t)
	ort, err := rescue.Ort(rettungUmgebung)
	if err != nil {
		t.Fatal(err)
	}
	// Erste Sitzung: geändert, gesichert, nicht gespeichert (Absturz).
	if err := oeffneImEditor(aussen, false, document.KanteWelle); err != nil {
		t.Fatal(err)
	}
	if text, _ := g.rettung.Wiederherstellen(g.dok); text != "" {
		t.Fatalf("Meldung ohne Kopie: %q", text)
	}
	mitKasten(t, g.dok)
	g.rettung.Sichere(g.dok)
	g.rettung.Ende()
	alt := filepath.Join(ort, rescue.Name(aussen))
	if _, err := os.Stat(alt); err != nil {
		t.Fatalf("Kopie der Datei von außen fehlt: %v", err)
	}
	if aufnahmen(t, b) != 0 {
		t.Fatalf("schon vor dem Speichern %d Aufnahmen", aufnahmen(t, b))
	}
	// Zweite Sitzung: die Kopie wird angewendet (wie der Editor beim Start), die Datei bleibt außen.
	if err := oeffneImEditor(aussen, false, document.KanteWelle); err != nil {
		t.Fatal(err)
	}
	if text, fehler := g.rettung.Wiederherstellen(g.dok); !strings.Contains(text, "Restored unsaved changes") || fehler {
		t.Errorf("Meldung %q (Fehler %v)", text, fehler)
	}
	if len(g.dok.Objekte()) != 1 || !g.geoeffnet.Aussen || aufnahmen(t, b) != 0 {
		t.Fatalf("zweites Öffnen: %d Objekte, Aussen %v, %d Aufnahmen", len(g.dok.Objekte()), g.geoeffnet.Aussen,
			aufnahmen(t, b))
	}
	if _, err := g.speichern(g.dok); err != nil {
		t.Fatal(err)
	}
	g.rettung.Ende()
	if _, err := os.Stat(alt); !os.IsNotExist(err) {
		t.Errorf("Kopie der Datei von außen nach der Übernahme noch da: %v", err)
	}
	mitKasten(t, g.dok)
	g.rettung.Sichere(g.dok)
	g.rettung.Ende()
	if _, err := os.Stat(filepath.Join(ort, rescue.Name(g.geoeffnet.Pfad()))); err != nil {
		t.Errorf("keine Kopie unter dem Namen der Aufnahme: %v", err)
	}
}
