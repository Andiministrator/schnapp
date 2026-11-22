// SPDX-License-Identifier: GPL-3.0-or-later

package document

// SetzeVerlaufGrenze setzt verlaufGrenze für Tests im Paket document_test (etwa die Rettungskopie bei unvollständigem
// Verlauf, F-385) und liefert die Rücknahme. Nur in Testbauten vorhanden.
func SetzeVerlaufGrenze(n int64) (zurueck func()) {
	alt := verlaufGrenze
	verlaufGrenze = n
	return func() { verlaufGrenze = alt }
}
