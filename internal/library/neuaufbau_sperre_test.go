// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
)

// sperreFrei sagt, ob ein eigener Deskriptor die Sperre des Index jetzt bekäme (niemand hält sie).
func sperreFrei(t *testing.T, b *Bibliothek) bool {
	t.Helper()
	f, err := os.OpenFile(b.sperrPfad(), os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("Sperrdatei: %v", err)
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return false
	}
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return true
}

// F-691: Der Neuaufbau heilt fehlende Vorschaubilder erst nach der Freigabe der Index-Sperre. Ein zweiter Prozess, der
// währenddessen ablegt, bekommt seine Indexzeile sofort durch, und sie bleibt im neuen Index stehen. Geprüft für beide
// Wege: Neuaufbau und den Neuaufbau aus Abgleich bei fehlender Indexdatei.
func TestNeuaufbauHeiltVorschauenOhneSperre(t *testing.T) {
	for _, weg := range []string{"Neuaufbau", "Abgleich"} {
		t.Run(weg, func(t *testing.T) {
			b, _ := neueBibliothek(t)
			var es []Eintrag
			for i := range 3 {
				es = append(es, ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-05-01T10:00:00Z").Add(time.Duration(i) * time.Hour)}))
			}
			for _, e := range es {
				if err := os.Remove(b.VorschauPfad(e.Pfad)); err != nil {
					t.Fatal(err)
				}
			}
			if weg == "Abgleich" {
				if err := os.Remove(b.Orte.Index); err != nil {
					t.Fatal(err)
				}
			}
			zweiter := Neu(b.Orte)
			fremd := Eintrag{Pfad: "2026-05/zweiter-prozess.png", Erstellt: zeit("2026-11-19T12:00:00Z"), Breite: 4, Hoehe: 3}
			var frei []bool
			alt := heileNachNeuaufbau
			heileNachNeuaufbau = func(bib *Bibliothek, pfad, rel string, stand Stand) {
				frei = append(frei, sperreFrei(t, bib))
				if len(frei) == 1 {
					start := time.Now()
					if err := zweiter.haengeAn(fremd); err != nil {
						t.Errorf("zweiter Prozess: %v", err)
					}
					if d := time.Since(start); d > time.Second {
						t.Errorf("zweiter Prozess wartete %v auf die Sperre", d)
					}
				}
				alt(bib, pfad, rel, stand)
			}
			t.Cleanup(func() { heileNachNeuaufbau = alt })
			if weg == "Neuaufbau" {
				if erg, err := b.Neuaufbau(); err != nil || erg.Aufgenommen != 3 {
					t.Fatalf("Neuaufbau = %+v, %v", erg, err)
				}
			} else if n, err := b.Abgleich(); err != nil || n != 0 {
				t.Fatalf("Abgleich = %d, %v", n, err)
			}
			if len(frei) != len(es) {
				t.Fatalf("%d Heilungen, soll %d", len(frei), len(es))
			}
			for i, ok := range frei {
				if !ok {
					t.Errorf("Heilung %d lief unter der Sperre des Index", i)
				}
			}
			if n := vorschauZahl(t, b); n != len(es) {
				t.Errorf("%d Vorschaubilder nach dem Neuaufbau, soll %d", n, len(es))
			}
			p := indexPfade(t, b)
			if !p[fremd.Pfad] || !p[es[0].Pfad] || !p[es[1].Pfad] || !p[es[2].Pfad] {
				t.Errorf("Index nach %s %v", weg, p)
			}
		})
	}
}

// Liegen alle Vorschaubilder, gibt es nichts zu heilen: Der Neuaufbau ruft heileVorschau dann gar nicht auf.
func TestNeuaufbauOhneFehlendeVorschauHeiltNicht(t *testing.T) {
	b, _ := neueBibliothek(t)
	ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-05-01T10:00:00Z")})
	alt := heileNachNeuaufbau
	n := 0
	heileNachNeuaufbau = func(*Bibliothek, string, string, Stand) { n++ }
	t.Cleanup(func() { heileNachNeuaufbau = alt })
	if _, err := b.Neuaufbau(); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("%d Heilungen bei vollständigen Vorschaubildern", n)
	}
}

// BenchmarkNeuaufbau10000 misst den Neuaufbau über 10 000 Arbeitsdateien mit vorhandenen Vorschaubildern: genau die
// Zeit, in der er die Sperre des Index hält (Ablaufen, Metadaten lesen, Index schreiben; F-691, Frist 10 s). Gemessen
// am 24.11.2026 auf lokaler SSD mit Arbeitsdateien 2560×1440: rund 0,76 s. Aufruf:
//
//	GOFLAGS=-tags=novulkan,nox11 go test ./internal/library -run '^$' -bench Neuaufbau10000 -benchtime 3x
func BenchmarkNeuaufbau10000(b *testing.B) {
	dir := b.TempDir()
	bib := Neu(Orte{
		Ablage:     filepath.Join(dir, "ablage"),
		Index:      filepath.Join(dir, "daten", "schnapp", "bibliothek.jsonl"),
		Vorschau:   filepath.Join(dir, "cache"),
		Papierkorb: filepath.Join(dir, "daten", "Trash"),
	})
	d, err := document.Neu(muster(2560, 1440))
	if err != nil {
		b.Fatal(err)
	}
	e, err := bib.Ablegen(d, d.Original(), fileformat.Metadaten{Erstellt: time.Date(2023, 12, 1, 8, 0, 0, 0, time.UTC)})
	if err != nil {
		b.Fatal(err)
	}
	inhalt, err := os.ReadFile(filepath.Join(bib.Orte.Ablage, filepath.FromSlash(e.Pfad)))
	if err != nil {
		b.Fatal(err)
	}
	vorschau, err := os.ReadFile(bib.VorschauPfad(e.Pfad))
	if err != nil {
		b.Fatal(err)
	}
	start := time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC)
	for i := range 10000 {
		t := start.Add(time.Duration(i) * 97 * time.Minute)
		pfad := fmt.Sprintf("%s/schnapp-%s.png", t.Format("2006-01"), t.Format("2006-01-02_15-04-05"))
		voll := filepath.Join(bib.Orte.Ablage, filepath.FromSlash(pfad))
		if err := os.MkdirAll(filepath.Dir(voll), 0o700); err != nil {
			b.Fatal(err)
		}
		if err := os.WriteFile(voll, inhalt, 0o600); err != nil {
			b.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(bib.VorschauPfad(pfad)), 0o700); err != nil {
			b.Fatal(err)
		}
		if err := os.WriteFile(bib.VorschauPfad(pfad), vorschau, 0o600); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for b.Loop() {
		if erg, err := bib.Neuaufbau(); err != nil || erg.Aufgenommen != 10001 {
			b.Fatalf("%+v, %v", erg, err)
		}
	}
}
