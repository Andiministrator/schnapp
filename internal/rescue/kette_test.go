// SPDX-License-Identifier: GPL-3.0-or-later

package rescue

// Tests aus Karte 200 (F-456, F-457, F-460): Aufträge der Kette lesen den Stand der Kette, nicht den der Aufrufe, und
// die Übernahme der Sperre (F-436) im Normalfall und im Fehlerpfad. Die Mutationsprobe scripts/mutanten/rescue.json
// (G1, G3, G4, T4, T5; G2 äquivalent) prüft, dass sie die Fixes tragen. Die Tests ändern Felder der Sicherung und
// laufen nicht parallel.

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

// beiseiteGelegt zählt die beiseitegelegten Kopien im Ordner ort.
func beiseiteGelegt(t *testing.T, ort string) int {
	t.Helper()
	reste, err := os.ReadDir(ort)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, e := range reste {
		if strings.HasSuffix(e.Name(), ".alt.json") {
			n++
		}
	}
	return n
}

// F-456: Fremde Sitzung B, ein Sichere steht noch in der Kette, dann NeueBasis auf eine freie Datei. Der
// Sichere-Auftrag darf nicht die Sperre der neuen Datei nachholen (basis ist beim Aufruf schon gesetzt) und B damit
// „nicht fremd“ machen: Der NeueBasis-Auftrag löschte danach die Kopie der alten Datei, die Fenster A noch hält.
func TestFremdSichereDannNeueBasisLaesstKopieVonA(t *testing.T) {
	pfad, k := ausgang(t, "datei")
	ort := t.TempDir()
	d, _ := document.Neu(bild(64, 64))
	bearbeite(t, d)
	a := NeueSicherung(ort, k)
	defer a.Ende()
	a.Sichere(d)
	warteKette(t, a)
	kopieA, err := os.ReadFile(filepath.Join(ort, Name(pfad)))
	if err != nil {
		t.Fatal(err)
	}

	b := NeueSicherung(ort, k)
	neu, _ := document.Neu(bild(64, 64))
	if w := b.Wiederherstellen(neu, nil); !w.Fremd {
		t.Fatalf("B nicht fremd: %+v", w)
	}
	mitText(t, neu)
	// Kette von B anhalten (hängende Platte, noch nicht gelaufene Goroutine), damit NeueBasis vor dem
	// Sichere-Auftrag aufgerufen ist.
	halt := make(chan struct{})
	b.mu.Lock()
	b.kette = halt
	b.mu.Unlock()
	b.Sichere(neu)
	pfad2, k2 := ausgang(t, "neu")
	b.NeueBasis(k2)
	close(halt)
	b.Ende()
	nachher, err := os.ReadFile(filepath.Join(ort, Name(pfad)))
	if err != nil {
		t.Fatalf("Kopie von A ist weg, obwohl A die Sperre hält: %v", err)
	}
	if !bytes.Equal(nachher, kopieA) {
		t.Fatal("Kopie von A verändert")
	}
	if n := beiseiteGelegt(t, ort); n != 0 {
		t.Errorf("%d Kopien beiseitegelegt, obwohl A die Sperre hält", n)
	}
	// B hat die neue Datei übernommen und sichert dort.
	mitText(t, neu)
	b.Sichere(neu)
	b.Ende()
	if _, err := Lies(ort, pfad2); err != nil {
		t.Errorf("Kopie zur neuen Basis fehlt: %v", err)
	}
}

// F-456: Ein verfallener Sichere-Auftrag holt keine Sperre nach. A endet mit Kopie, während B (fremd) schon auf eine
// neue Datei wechselt: Die Kopie von A bleibt unter ihrem Namen liegen und wird beim nächsten Öffnen angewendet, statt
// von einer Übernahme beiseitegelegt zu werden, die B gleich wieder aufgibt.
func TestVerfallenerAuftragUebernimmtNicht(t *testing.T) {
	pfad, k := ausgang(t, "datei")
	ort := t.TempDir()
	d, _ := document.Neu(bild(64, 64))
	bearbeite(t, d)
	a := NeueSicherung(ort, k)
	a.Sichere(d)
	warteKette(t, a)
	kopieA, err := os.ReadFile(filepath.Join(ort, Name(pfad)))
	if err != nil {
		t.Fatal(err)
	}

	b := NeueSicherung(ort, k)
	neu, _ := document.Neu(bild(64, 64))
	if w := b.Wiederherstellen(neu, nil); !w.Fremd {
		t.Fatalf("B nicht fremd: %+v", w)
	}
	mitText(t, neu)
	halt := make(chan struct{})
	b.mu.Lock()
	b.kette = halt
	b.mu.Unlock()
	b.Sichere(neu)
	a.Ende() // die Sperre auf die alte Datei ist frei, die Kopie von A bleibt
	_, k2 := ausgang(t, "neu")
	b.NeueBasis(k2)
	close(halt)
	b.Ende()
	if nachher, err := os.ReadFile(filepath.Join(ort, Name(pfad))); err != nil || !bytes.Equal(nachher, kopieA) {
		t.Fatalf("Kopie von A nicht mehr an ihrem Platz: %v", err)
	}
	if n := beiseiteGelegt(t, ort); n != 0 {
		t.Errorf("%d Kopien beiseitegelegt", n)
	}
}

// F-457: Eine gestörte Sitzung (Symlink am Sperrnamen) wechselt per NeueBasis auf eine freie Datei; ein Sichere direkt
// danach fällt nicht weg. gestoert entscheidet wie aus erst die Kette, solange ein Wechsel aussteht.
func TestGestoertSichereDirektNachNeueBasis(t *testing.T) {
	pfad, k := ausgang(t, "datei")
	ort := t.TempDir()
	if err := os.Symlink(filepath.Join(ort, "woanders"), filepath.Join(ort, sperrName(pfad))); err != nil {
		t.Fatal(err)
	}
	s := NeueSicherung(ort, k)
	s.mu.Lock()
	g := s.gestoert
	s.mu.Unlock()
	if !g {
		t.Fatal("Sitzung nicht gestört")
	}
	pfad2, k2 := ausgang(t, "neu")
	d, _ := document.Neu(bild(64, 64))
	bearbeite(t, d)
	s.NeueBasis(k2)
	s.Sichere(d) // ohne auf die Kette zu warten
	s.Ende()
	if _, err := Lies(ort, pfad2); err != nil {
		t.Errorf("Kopie zur neuen Basis fehlt: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(ort, "woanders")); err == nil {
		t.Error("Symlink am Sperrnamen verfolgt")
	}
}

// F-460, F-436: Fenster A endet sauber, ohne Kopie. Das fremde Fenster B holt die Sperre beim nächsten Sichern nach
// und sichert; es gibt nichts beiseitezulegen, und das Fehlen einer Kopie macht B nicht stumm.
func TestUebernahmeOhneKopieSichert(t *testing.T) {
	pfad, k := ausgang(t, "datei")
	ort := t.TempDir()
	a := NeueSicherung(ort, k)
	b := NeueSicherung(ort, k)
	var g diagnosen
	g.fuer(b)
	neu, _ := document.Neu(bild(64, 64))
	if w := b.Wiederherstellen(neu, nil); !w.Fremd {
		t.Fatalf("B nicht fremd: %+v", w)
	}
	a.Ende()

	bearbeite(t, neu)
	b.Sichere(neu)
	b.Ende()
	kp, err := Lies(ort, pfad)
	if err != nil {
		t.Fatalf("Kopie von B fehlt: %v", err)
	}
	leer, _ := document.Neu(bild(64, 64))
	if err := Anwenden(leer, kp, nil); err != nil || len(leer.Objekte()) != len(neu.Objekte()) {
		t.Errorf("Kopie von B unvollständig: %d Objekte, %v", len(leer.Objekte()), err)
	}
	if n := beiseiteGelegt(t, ort); n != 0 {
		t.Errorf("%d Kopien beiseitegelegt, obwohl A keine hinterließ", n)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, e := range g.list {
		if !strings.Contains(e.Error(), "anderen Fenster") {
			t.Errorf("unerwartete Diagnose: %v", e)
		}
	}
}

// F-460, F-436: Lässt sich die Kopie, die A hinterlassen hat, bei der Übernahme nicht beiseitelegen, wird B stumm: Es
// überschreibt und löscht die Kopie von A nicht, und die Diagnose meldet den Fehler.
func TestUebernahmeBeiseiteScheitertMachtStumm(t *testing.T) {
	pfad, k := ausgang(t, "datei")
	ort := t.TempDir()
	d, _ := document.Neu(bild(64, 64))
	bearbeite(t, d)
	a := NeueSicherung(ort, k)
	a.Sichere(d)
	warteKette(t, a)
	kopieA, err := os.ReadFile(filepath.Join(ort, Name(pfad)))
	if err != nil {
		t.Fatal(err)
	}

	b := NeueSicherung(ort, k)
	var g diagnosen
	g.fuer(b)
	neu, _ := document.Neu(bild(64, 64))
	if w := b.Wiederherstellen(neu, nil); !w.Fremd {
		t.Fatalf("B nicht fremd: %+v", w)
	}
	// Ein Verzeichnis unter dem Namen, den legeBeiseite wählt, lässt das Umbenennen scheitern (EISDIR).
	fest := time.Date(2026, 1, 2, 3, 4, 5, 6, time.UTC)
	b.mu.Lock()
	b.jetzt = func() time.Time { return fest }
	b.mu.Unlock()
	ziel := strings.TrimSuffix(Name(pfad), ".json") + fmt.Sprintf(".%d.alt.json", fest.UnixNano())
	if err := os.Mkdir(filepath.Join(ort, ziel), 0o700); err != nil {
		t.Fatal(err)
	}
	a.Ende() // A endet mit ungesicherten Änderungen: seine Kopie bleibt liegen, die Sperre ist frei

	mitText(t, neu)
	b.Sichere(neu)
	warteKette(t, b)
	b.Verwirf()
	b.Ende()
	nachher, err := os.ReadFile(filepath.Join(ort, Name(pfad)))
	if err != nil {
		t.Fatalf("Kopie von A gelöscht: %v", err)
	}
	if !bytes.Equal(nachher, kopieA) {
		t.Error("Kopie von A überschrieben")
	}
	g.mu.Lock()
	n := len(g.list)
	g.mu.Unlock()
	if n == 0 {
		t.Error("gescheitertes Beiseitelegen nicht gemeldet")
	}
}
