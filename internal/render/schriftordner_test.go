// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"os"
	"testing"

	"github.com/Andiministrator/schnapp/internal/fonts"
)

// TestMain macht die Tests unabhängig von den Schriften des Rechners (Sitzungs-QA B-12): Eine fremde Familie in einem
// Test löst sonst die Suche in HOME und /usr/share/fonts aus; Ergebnis und Dauer hingen vom Rechner ab.
func TestMain(m *testing.M) {
	fonts.SetzeOrdner(nil)
	os.Exit(m.Run())
}
