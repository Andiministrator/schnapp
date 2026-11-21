// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"io"
	"path/filepath"
	"testing"
)

// F-313: schreibeAtomar reicht der Schreibfunktion die Datei selbst, nicht eine Hülle: Nur so kann
// fileformat.SchreibeArbeitsdatei je Stück synchronisieren und pulsen, wo die Zeit eines langsamen Mediums vergeht.
func TestSchreibeAtomarReichtSynchronisierbarenSchreiberDurch(t *testing.T) {
	ziel := filepath.Join(t.TempDir(), "ziel.bin")
	gesehen := false
	err := schreibeAtomar(ziel, func(w io.Writer) error {
		s, ok := w.(interface{ Sync() error })
		gesehen = ok
		if ok {
			if _, err := io.WriteString(w, "x"); err != nil {
				return err
			}
			return s.Sync()
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !gesehen {
		t.Error("die Schreibfunktion bekam einen Schreiber ohne Sync; die Arbeitsdatei pulst dann nicht je abgeflossenem Stück (F-313)")
	}
}
