// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

// BenchmarkSuchen10000 misst, was die Leiste der letzten Aufnahmen nach jedem Speichern tut (F-638): den ganzen Index
// mit 10 000 Einträgen lesen, Agenten-Aufnahmen ausblenden, sortieren. Aufruf:
//
//	go test ./internal/library -run '^$' -bench Suchen10000 -benchmem
func BenchmarkSuchen10000(b *testing.B) {
	dir := b.TempDir()
	bib := Neu(Orte{
		Ablage:     filepath.Join(dir, "ablage"),
		Index:      filepath.Join(dir, "daten", "schnapp", "bibliothek.jsonl"),
		Vorschau:   filepath.Join(dir, "cache"),
		Papierkorb: filepath.Join(dir, "daten", "Trash"),
	})
	start := time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC)
	es := make([]Eintrag, 10000)
	for i := range es {
		t := start.Add(time.Duration(i) * 97 * time.Minute)
		pfad := fmt.Sprintf("%s/schnapp-%s.png", t.Format("2006-01"), t.Format("2006-01-02_15-04-05"))
		if i%10 == 0 {
			pfad = AgentenOrdner + "/" + pfad
		}
		es[i] = Eintrag{Pfad: pfad, Erstellt: t, Breite: 2560, Hoehe: 1440, Schlagworte: []string{"probe", "leiste"}, Quelle: "Konsole", Bearbeitet: i%3 == 0}
	}
	if err := bib.schreibeIndex(es); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for b.Loop() {
		got, err := bib.Suchen(Anfrage{OhneAgenten: true})
		if err != nil || len(got) != 9000 {
			b.Fatalf("%d Treffer, %v", len(got), err)
		}
	}
}
