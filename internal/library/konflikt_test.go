// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"errors"
	"image/color"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/saveprogress"
)

// standIn liefert den Stand einer Aufnahme der Ablage (wie der Editor beim Öffnen).
func standIn(t *testing.T, b *Bibliothek, rel string) Stand {
	t.Helper()
	s, err := StandVon(filepath.Join(b.Orte.Ablage, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func kasten(t *testing.T, d *document.Dokument, x float64) {
	t.Helper()
	if _, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(x, 1), document.Pt(x+3, 4)),
		Strich: document.Strich{Farbe: color.NRGBA{R: 255, A: 255}, Breite: 1}}); err != nil {
		t.Fatal(err)
	}
}

// F-110: Zwei Editoren öffnen dieselbe Aufnahme; der erste speichert, der zweite bekommt ErrGeaendert und die Datei
// behält den Stand des ersten. Der erste kann mit dem zurückgegebenen Stand weiter speichern.
func TestSpeichernZweiEditorenKonflikt(t *testing.T) {
	b, _ := neueBibliothek(t)
	d := neuesDokument(t, 20, 20)
	e, err := b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{})
	if err != nil {
		t.Fatal(err)
	}
	voll := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
	oeffne := func() (*document.Dokument, Stand) {
		s := standIn(t, b, e.Pfad) // vor dem Lesen, wie ladeFuerEditor
		g, err := fileformat.Oeffne(voll)
		if err != nil {
			t.Fatal(err)
		}
		return g.Dokument, s
	}
	d1, s1 := oeffne()
	d2, s2 := oeffne()
	if s1 != s2 || s1 == (Stand{}) {
		t.Fatalf("Stände beim Öffnen %+v / %+v (Soll gleich, nicht Null)", s1, s2)
	}
	kasten(t, d1, 1)
	_, s1neu, err := b.Speichern(e.Pfad, s1, d1, d1.Ansichtsbild())
	if err != nil {
		t.Fatalf("erster Editor: %v", err)
	}
	if s1neu == s1 || s1neu != standIn(t, b, e.Pfad) {
		t.Errorf("neuer Stand %+v (Soll: anders als beim Öffnen und gleich der Datei)", s1neu)
	}
	nachErstem, _ := os.ReadFile(voll)

	kasten(t, d2, 5)
	kasten(t, d2, 9)
	_, s2neu, err := b.Speichern(e.Pfad, s2, d2, d2.Ansichtsbild())
	if !errors.Is(err, ErrGeaendert) || s2neu != (Stand{}) {
		t.Fatalf("zweiter Editor: %v, Stand %+v (Soll ErrGeaendert, Nullstand)", err, s2neu)
	}
	if jetzt, _ := os.ReadFile(voll); string(jetzt) != string(nachErstem) {
		t.Error("zweiter Editor hat die Datei überschrieben")
	}
	g, err := fileformat.Oeffne(voll)
	if err != nil || len(g.Dokument.Objekte()) != 1 {
		t.Errorf("Datei nach dem Konflikt: %v Objekte, %v (Soll 1)", len(g.Dokument.Objekte()), err)
	}

	// Der erste Editor speichert mit seinem neuen Stand ohne Konflikt weiter.
	kasten(t, d1, 12)
	if _, _, err := b.Speichern(e.Pfad, s1neu, d1, d1.Ansichtsbild()); err != nil {
		t.Errorf("erster Editor, zweites Speichern: %v", err)
	}
	// Mit dem alten Stand dagegen nicht mehr.
	if _, _, err := b.Speichern(e.Pfad, s1, d1, d1.Ansichtsbild()); !errors.Is(err, ErrGeaendert) {
		t.Errorf("veralteter Stand: %v (Soll ErrGeaendert)", err)
	}
	// Der Nullwert passt zu keiner Datei.
	if _, _, err := b.Speichern(e.Pfad, Stand{}, d1, d1.Ansichtsbild()); !errors.Is(err, ErrGeaendert) {
		t.Errorf("Nullstand: %v (Soll ErrGeaendert)", err)
	}
}

// Auch eine Änderung an Ort und Stelle (gleicher Inode, andere Zeit) gilt als Konflikt.
func TestSpeichernKonfliktBeiAenderungAnOrtUndStelle(t *testing.T) {
	b, _ := neueBibliothek(t)
	d := neuesDokument(t, 20, 20)
	e, err := b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{})
	if err != nil {
		t.Fatal(err)
	}
	vorher := standIn(t, b, e.Pfad)
	voll := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
	spaeter := time.Unix(0, vorher.zeit).Add(3 * time.Second)
	if err := os.Chtimes(voll, spaeter, spaeter); err != nil {
		t.Fatal(err)
	}
	if nachher := standIn(t, b, e.Pfad); nachher.inode != vorher.inode || nachher == vorher {
		t.Fatalf("Probe: Inode %d→%d, Stand gleich %v", vorher.inode, nachher.inode, nachher == vorher)
	}
	kasten(t, d, 1)
	if _, _, err := b.Speichern(e.Pfad, vorher, d, d.Ansichtsbild()); !errors.Is(err, ErrGeaendert) {
		t.Errorf("Fehler %v (Soll ErrGeaendert)", err)
	}
}

// Wird die Datei während des Kodierens ersetzt (zweiter Vergleich kurz vor dem Umbenennen), bleibt die neue Fassung
// und keine temporäre Datei liegt herum.
func TestSpeichernKonfliktKurzVorDemErsetzen(t *testing.T) {
	b, _ := neueBibliothek(t)
	d := neuesDokument(t, 20, 20)
	e, err := b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{})
	if err != nil {
		t.Fatal(err)
	}
	s := standIn(t, b, e.Pfad)
	voll := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
	fremd := []byte("zwischendurch ersetzt")
	// Die Datei wird nach dem Lesen der alten Fassung ausgetauscht — nach dem ersten Vergleich, vor dem zweiten. Über den
	// Haken der Marke, nicht über den ersten Pixelzugriff: Den macht seit F-310 auch die nebenläufige Vorschau, mitunter
	// vor dem Lesen der alten Datei, und dann meldet Speichern zu Recht ErrFremd (CI-Lauf zu e9b4cef).
	var einmal sync.Once
	t.Cleanup(func() { saveprogress.SetzeHaken(nil) })
	saveprogress.SetzeHaken(func(schritt string) {
		if schritt != "alte-datei-gelesen" {
			return
		}
		einmal.Do(func() {
			if err := os.WriteFile(voll+".neu", fremd, 0o600); err != nil {
				t.Error(err)
			}
			if err := os.Rename(voll+".neu", voll); err != nil {
				t.Error(err)
			}
		})
	})
	kasten(t, d, 1)
	if _, _, err := b.Speichern(e.Pfad, s, d, d.Ansichtsbild()); !errors.Is(err, ErrGeaendert) {
		t.Fatalf("Fehler %v (Soll ErrGeaendert)", err)
	}
	if jetzt, _ := os.ReadFile(voll); string(jetzt) != string(fremd) {
		t.Error("die zwischendurch geschriebene Fassung wurde überschrieben")
	}
	reste, _ := filepath.Glob(filepath.Join(filepath.Dir(voll), tempPraefix+"*"))
	if len(reste) != 0 {
		t.Errorf("temporäre Dateien liegen noch: %v", reste)
	}
}
