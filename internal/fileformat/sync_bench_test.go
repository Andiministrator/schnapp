// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkSyncJeStueck misst, was ein Sync je Stück gegenüber einem einzigen Sync am Ende kostet (F-313): 32 MB
// Rauschen auf die Platte unter TMPDIR (kein Race-Bau; Zahlen aus -race taugen nicht).
func BenchmarkSyncJeStueck(b *testing.B) {
	daten := make([]byte, 32<<20)
	rand.New(rand.NewSource(313)).Read(daten)
	for _, stueck := range []int{0, 1 << 20, 4 << 20, 8 << 20} {
		b.Run(fmt.Sprintf("stueck=%dMiB", stueck>>20), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				f, err := os.Create(filepath.Join(b.TempDir(), "x"))
				if err != nil {
					b.Fatal(err)
				}
				rest := daten
				for len(rest) > 0 {
					n := len(rest)
					if stueck > 0 {
						n = min(n, stueck)
					}
					f.Write(rest[:n])
					rest = rest[n:]
					if stueck > 0 {
						f.Sync()
					}
				}
				f.Sync()
				f.Close()
			}
		})
	}
}
