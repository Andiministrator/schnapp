// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"sync/atomic"
	"testing"
	"time"
)

// F-258: Ohne Quittung (Fenster unsichtbar) fällt der Takt zurück; die Zahl der Weckrufe bis zur Obergrenze bleibt
// weit unter der Zahl bei festem Takt. Messung im Test: takt 5 ms, Grenze 1 s ergäbe bei festem Takt rund 200 Rufe.
func TestWeckerLeerlaufWeniger(t *testing.T) {
	const takt, grenze = 5 * time.Millisecond, time.Second
	var n atomic.Int32
	w := NeuerWecker(func() { n.Add(1) }, takt, grenze)
	defer w.Ende()
	w.Anfordern()
	time.Sleep(grenze + 300*time.Millisecond)
	if w.Offen() {
		t.Fatal("Wecker offen nach der Obergrenze")
	}
	got := int(n.Load())
	fest := int(grenze / takt) // Rufe bei festem Takt
	t.Logf("Weckrufe bis zur Obergrenze: %d (fester Takt: rund %d)", got, fest)
	if got > fest/2 {
		t.Errorf("%d Weckrufe, erwartet deutlich unter %d (fester Takt rund %d)", got, fest/2, fest)
	}
	if got < schnelleTakte {
		t.Errorf("nur %d Weckrufe, die ersten %d im vollen Takt erwartet", got, schnelleTakte)
	}
}

// F-231 bleibt: Auch im zurückgefallenen Takt weckt der Wecker weiter, bis quittiert wird, und eine neue Anforderung
// beginnt wieder im vollen Takt. Ein Bild nach langem Leerlauf (Fenster wieder sichtbar) holt die Antwort ab: Die
// Schleife quittiert, der Wecker ruht.
func TestWeckerZurueckgefallenWecktWeiterUndNeuerStart(t *testing.T) {
	const takt = 2 * time.Millisecond
	var n atomic.Int32
	w := NeuerWecker(func() { n.Add(1) }, takt, time.Minute)
	defer w.Ende()
	w.Anfordern()
	// Bis weit hinter die schnellen Takte: 20 Takte zu 2 ms, danach 4, 8, 16, 32 ms.
	time.Sleep(400 * time.Millisecond)
	vor := n.Load()
	if int(vor) <= schnelleTakte {
		t.Fatalf("nur %d Rufe nach 400 ms", vor)
	}
	time.Sleep(200 * time.Millisecond)
	if n.Load() == vor {
		t.Fatal("zurückgefallener Wecker weckt nicht mehr (F-231)")
	}
	// Neue Anforderung im Leerlauf: sofort ein Ruf, dann wieder der volle Takt.
	vor = n.Load()
	w.Anfordern()
	time.Sleep(60 * time.Millisecond)
	if d := int(n.Load() - vor); d < 8 {
		t.Errorf("nach neuer Anforderung nur %d Rufe in 60 ms, erwartet vollen Takt", d)
	}
	// Das Bild kommt: Quittung, danach Ruhe.
	w.Quittieren()
	time.Sleep(150 * time.Millisecond) // ein laufender Takt darf auslaufen
	vor = n.Load()
	time.Sleep(150 * time.Millisecond)
	if n.Load() != vor {
		t.Errorf("Weckrufe nach Quittieren: %d -> %d", vor, n.Load())
	}
	if w.Offen() {
		t.Error("Offen nach Quittieren")
	}
}
