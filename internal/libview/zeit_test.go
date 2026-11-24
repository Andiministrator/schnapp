// SPDX-License-Identifier: GPL-3.0-or-later

package libview

import (
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/po"
)

// ADR-0036 Punkt 1: heute die Uhrzeit, gestern „yesterday“, sonst Wochentag und Datum, in einem anderen Jahr mit Jahr;
// Sprache über den Katalog. Die Grenzen liegen an Mitternacht der Zone von jetzt, nicht 24 Stunden zurück.
func TestZeitangabe(t *testing.T) {
	zone := time.FixedZone("Test", 2*3600)
	jetzt := time.Date(2026, 11, 24, 0, 30, 0, 0, zone) // Dienstag, kurz nach Mitternacht
	faelle := []struct {
		zeit   time.Time
		en, de string
	}{
		{time.Date(2026, 11, 24, 0, 10, 0, 0, zone), "00:10", "00:10"},
		{time.Date(2026, 11, 23, 22, 5, 0, 0, time.UTC), "00:05", "00:05"},                // in UTC gestern, in der Zone heute
		{time.Date(2026, 11, 23, 23, 50, 0, 0, zone), "yesterday", "gestern"},             // 40 Minuten her, aber gestern
		{time.Date(2026, 11, 21, 22, 50, 0, 0, time.UTC), "Sun Nov 22", "So 22.11."},      // in der Zone von jetzt schon der 22.
		{time.Date(2026, 11, 23, 0, 0, 0, 0, zone), "yesterday", "gestern"},               // Anfang von gestern
		{time.Date(2026, 11, 22, 23, 59, 0, 0, zone), "Sun Nov 22", "So 22.11."},          // Ende von vorgestern
		{time.Date(2025, 12, 31, 12, 0, 0, 0, zone), "Wed Dec 31, 2025", "Mi 31.12.2025"}, // anderes Jahr
		{time.Date(2026, 11, 25, 9, 0, 0, 0, zone), "Wed Nov 25", "Mi 25.11."},            // Uhr vorgegangen: Datum
	}
	for _, sprache := range []string{"en", "de"} {
		k, err := i18n.Lade(po.Dateien, sprache)
		if err != nil {
			t.Fatal(err)
		}
		i18n.Setze(k)
		for _, f := range faelle {
			soll := f.en
			if sprache == "de" {
				soll = f.de
			}
			if ist := Zeitangabe(f.zeit, jetzt); ist != soll {
				t.Errorf("%s %v: %q, Soll %q", sprache, f.zeit, ist, soll)
			}
		}
	}
	i18n.Setze(nil)
	// Alle Wochentage haben einen eigenen Namen.
	namen := map[string]bool{}
	for w := time.Sunday; w <= time.Saturday; w++ {
		namen[wochentag(w)] = true
	}
	if len(namen) != 7 {
		t.Errorf("Wochentage: %v", namen)
	}
}
