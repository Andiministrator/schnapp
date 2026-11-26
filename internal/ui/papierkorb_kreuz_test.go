// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Karte #312 (ADR-0036 Zusatz): Ein ✕ oben rechts in der Vorschau legt die Aufnahme in den Papierkorb — sichtbar unter
// dem Zeiger oder nach langem Druck, in der Leiste und in der vollen Ansicht. Entf an einer Vorschau der Leiste tut
// dasselbe. Karte #347: auch an der offenen Aufnahme — der Editor wechselt erst zur Nachbarin, dann geht sie in den
// Papierkorb; die Zwischenablage bleibt.

import (
	"errors"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/widget"
)

func papierkorbDateien(t *testing.T, l *leistenLauf) int {
	t.Helper()
	papier, _ := os.ReadDir(filepath.Join(l.b.Orte.Papierkorb, "files"))
	return len(papier)
}

func TestLeisteKreuzLegtInDenPapierkorb(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 4)
	k := &l.e.unten
	if !k.l.IstAktuell(1) {
		t.Fatal("Vorbedingung: Stelle 1 ist die offene Aufnahme")
	}
	// Ohne Zeiger kein ✕.
	for i, r := range k.loeschRahmen {
		if !r.Empty() {
			t.Fatalf("✕ an Stelle %d ohne Zeiger", i)
		}
	}
	// Zeiger über der offenen: auch dort das ✕ (Karte #347).
	l.maus(pointer.Move, l.kachel(1), 0)
	l.setle()
	if k.loeschRahmen[1].Empty() {
		t.Error("kein ✕ an der offenen Aufnahme")
	}
	// Zeiger über Stelle 2: ✕ oben rechts in der Vorschau.
	l.maus(pointer.Move, l.kachel(2), 0)
	l.setle()
	lr, kr := k.loeschRahmen[2], k.kacheln[2]
	if lr.Empty() || !lr.In(kr) || lr.Max.X < kr.Max.X-8 || lr.Min.Y > kr.Min.Y+8 {
		t.Fatalf("✕ %v nicht oben rechts in %v", lr, kr)
	}
	// Zeiger auf das ✕: Hinweis, dann Klick legt in den Papierkorb, die Aufnahme öffnet sich nicht.
	l.maus(pointer.Move, mittenPunkt(lr), 0)
	l.setle()
	if k.hinweis != loeschHinweis() {
		t.Errorf("Hinweis am ✕: %q", k.hinweis)
	}
	weg := l.pfade[2]
	l.klick(mittenPunkt(k.loeschRahmen[2]))
	l.warte(func() bool { return !l.e.bibl.arbeitLaeuft && !k.indexLaeuft })
	if _, err := os.Stat(weg); !os.IsNotExist(err) {
		t.Fatalf("Datei noch da: %v", err)
	}
	if n := papierkorbDateien(t, l); n != 1 {
		t.Errorf("Papierkorb hat %d Dateien", n)
	}
	if len(l.offen) != 0 {
		t.Errorf("✕ öffnete eine Aufnahme: %v", l.offen)
	}
	l.warte(func() bool { return len(k.l.Eintraege()) == 3 })
	if !strings.Contains(l.e.meldung.Text, "trash") || l.e.meldung.Fehler {
		t.Errorf("Meldung %q", l.e.meldung.Text)
	}
}

func TestLeisteEntfLegtInDenPapierkorb(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	k := &l.e.unten
	l.fokusiere(&k.knoepfe[2])
	l.taste(key.NameDeleteForward, 0)
	l.warte(func() bool { return !l.e.bibl.arbeitLaeuft && !k.indexLaeuft })
	if _, err := os.Stat(l.pfade[2]); !os.IsNotExist(err) {
		t.Fatalf("Entf an der Vorschau: Datei noch da (%v)", err)
	}
	// An der offenen Aufnahme: Wechsel zur Nachbarin (die jüngere, die ältere ist weg), dann Papierkorb (Karte #347).
	l.warte(func() bool { return len(k.l.Eintraege()) == 2 })
	l.setle()
	l.fokusiere(&k.knoepfe[1])
	l.taste(key.NameDeleteForward, 0)
	l.warte(func() bool { return !l.e.wechsel.laedt && !l.e.bibl.arbeitLaeuft && !k.indexLaeuft })
	if _, err := os.Stat(l.pfade[1]); !os.IsNotExist(err) {
		t.Errorf("Entf an der offenen: Datei noch da (%v)", err)
	}
	if a := l.aufrufe(); len(a) != 1 || a[0] != l.pfade[0] {
		t.Errorf("geladen: %v, Soll %s", a, l.pfade[0])
	}
}

func TestLangerDruck(t *testing.T) {
	t0 := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	for _, f := range []struct {
		h    []widget.Press
		lang bool
	}{
		{nil, false},
		{[]widget.Press{{Start: t0, End: t0.Add(100 * time.Millisecond)}}, false},
		{[]widget.Press{{Start: t0, End: t0.Add(langDruckZeit)}}, true},
		{[]widget.Press{{Start: t0, End: t0.Add(time.Second), Cancelled: true}}, false},
		{[]widget.Press{{Start: t0}}, false}, // läuft noch
		{[]widget.Press{{Start: t0, End: t0.Add(time.Second)}, {Start: t0, End: t0.Add(time.Millisecond)}}, false},
	} {
		if got := langerDruck(f.h); got != f.lang {
			t.Errorf("%v: %v, Soll %v", f.h, got, f.lang)
		}
	}
}

func TestAnsichtKreuzUndLangerDruck(t *testing.T) {
	l := bibLauf(t, 4)
	b := &l.e.bibl
	// Zeiger über Stelle 2: ✕; über der offenen (1) auch (Karte #347).
	l.maus(pointer.Move, l.bibKachel(1), 0)
	l.setle()
	if b.loesch[1-b.von].Empty() {
		t.Error("kein ✕ an der offenen Aufnahme")
	}
	l.maus(pointer.Move, l.bibKachel(2), 0)
	l.setle()
	lr := b.loesch[2-b.von]
	if lr.Empty() {
		t.Fatal("kein ✕ unter dem Zeiger")
	}
	l.maus(pointer.Move, mittenPunkt(lr), 0)
	l.setle()
	if h := l.e.bibliothekHinweis(); h != loeschHinweis() {
		t.Errorf("Hinweis am ✕: %q", h)
	}
	weg := l.pfade[2]
	l.bibDruck(mittenPunkt(lr), time.Second)
	l.bibRuhig()
	if _, err := os.Stat(weg); !os.IsNotExist(err) {
		t.Fatalf("✕ in der Ansicht: Datei noch da (%v)", err)
	}
	if len(b.r.Eintraege()) != 3 {
		t.Errorf("%d Einträge nach dem ✕", len(b.r.Eintraege()))
	}
	// Langer Druck (Finger: kein Zeiger darüber) zeigt das ✕ der Kachel — nach der Ruhe, die dem Löschen folgt (K-2).
	b.loeschRuheBis = time.Time{}
	l.maus(pointer.Leave, image.Pt(0, 0), 0)
	l.setle()
	p := l.bibKachel(2)
	pos := f32.Pt(float32(p.X), float32(p.Y))
	l.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, Buttons: pointer.ButtonPrimary, Position: pos,
		Time: 10 * time.Second})
	l.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Touch, Position: pos,
		Time: 10*time.Second + langDruckZeit})
	l.setle()
	if b.langDruck != 2 || b.loesch[2-b.von].Empty() {
		t.Errorf("langer Druck: Stelle %d, ✕ %v", b.langDruck, b.loesch[2-b.von])
	}
	// Ein kurzer Druck zeigt es nicht.
	b.langDruck = -1
	l.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, Buttons: pointer.ButtonPrimary, Position: pos,
		Time: 20 * time.Second})
	l.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Touch, Position: pos, Time: 20*time.Second + 100*time.Millisecond})
	l.setle()
	if b.langDruck != -1 {
		t.Errorf("kurzer Druck zeigte das ✕ (Stelle %d)", b.langDruck)
	}
}

// Kritik K-2: Der zweite Klick eines Doppelklicks auf das ✕ legt nicht die nachgerückte Aufnahme in den Papierkorb und
// öffnet sie auch nicht.
func TestLeisteKreuzDoppelklickLoeschtNurEine(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 4)
	k := &l.e.unten
	l.maus(pointer.Move, l.kachel(2), 0)
	l.setle()
	lr := k.loeschRahmen[2]
	l.maus(pointer.Move, mittenPunkt(lr), 0)
	l.setle()
	l.klick(mittenPunkt(lr))
	l.warte(func() bool { return !l.e.bibl.arbeitLaeuft && !k.indexLaeuft && len(k.l.Eintraege()) == 3 })
	l.klick(mittenPunkt(lr)) // sofort danach, an derselben Stelle steht jetzt die nächste Aufnahme
	l.warte(func() bool { return !l.e.bibl.arbeitLaeuft && !k.indexLaeuft })
	if n := papierkorbDateien(t, l); n != 1 {
		t.Errorf("Papierkorb hat %d Dateien nach Doppelklick, Soll 1", n)
	}
	if len(l.aufrufe()) != 0 {
		t.Errorf("zweiter Klick öffnete %v", l.aufrufe())
	}
}

// Kritik K-3: Eine frisch geöffnete Ansicht zeigt ohne Zeiger kein ✕, auch nicht an Kachel 0.
func TestAnsichtOhneZeigerOhneKreuz(t *testing.T) {
	l := bibLauf(t, 3)
	for i, r := range l.e.bibl.loesch {
		if !r.Empty() {
			t.Errorf("✕ an Stelle %d ohne Zeiger", i+l.e.bibl.von)
		}
	}
}

// Karte #347: Das ✕ an der offenen Aufnahme speichert (hier unverändert: nichts zu tun), lädt die nächst ältere und legt
// erst danach die bisher offene in den Papierkorb. Die Kopie, die das Fenster hält (Rückfall ohne Halter), bleibt.
func TestLeisteKreuzAnDerOffenen(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 4)
	k := &l.e.unten
	l.e.fensterHaeltKopie = true // Strg+C vorher, ohne Halter: das Fenster bietet die Kopie an
	l.tor = make(chan struct{})
	l.maus(pointer.Move, l.kachel(1), 0)
	l.setle()
	lr := k.loeschRahmen[1]
	l.maus(pointer.Move, mittenPunkt(lr), 0)
	l.setle()
	l.klick(mittenPunkt(lr))
	l.warte(func() bool { return len(l.aufrufe()) == 1 })
	// Während des Ladens liegt die offene noch an ihrem Platz.
	if _, err := os.Stat(l.pfade[1]); err != nil {
		t.Fatalf("offene vor der Übernahme verschoben: %v", err)
	}
	close(l.tor)
	l.warte(func() bool {
		return !l.e.wechsel.laedt && !l.e.bibl.arbeitLaeuft && !k.indexLaeuft && len(k.l.Eintraege()) == 3
	})
	if a := l.aufrufe(); a[0] != l.pfade[2] {
		t.Errorf("geladen %s, Soll die nächst ältere %s", a[0], l.pfade[2])
	}
	if _, err := os.Stat(l.pfade[1]); !os.IsNotExist(err) {
		t.Errorf("bisher offene noch da: %v", err)
	}
	if n := papierkorbDateien(t, l); n != 1 {
		t.Errorf("Papierkorb hat %d Dateien", n)
	}
	if l.e.pfad() != l.pfade[2] {
		t.Errorf("offen ist %s", l.e.pfad())
	}
	if !l.e.fensterHaeltKopie {
		t.Error("die Kopie des Fensters ging verloren")
	}
	if l.alt.verworfen == 0 {
		t.Error("Rettungskopie der weggeworfenen nicht verworfen")
	}
}

// Die älteste offene: Nachbarin ist die jüngere; scheitert das Laden, bleibt sie offen und im Bestand.
func TestLeisteKreuzAnDerOffenenLadenScheitert(t *testing.T) {
	if nachbarStelle(2, 3) != 1 || nachbarStelle(0, 3) != 1 || nachbarStelle(0, 1) != -1 {
		t.Fatal("nachbarStelle")
	}
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	k := &l.e.unten
	l.fehler = errors.New("kaputt")
	l.e.leistePapierkorb(1)
	l.warte(func() bool { return !l.e.wechsel.laedt && !l.e.bibl.arbeitLaeuft && !k.indexLaeuft })
	if _, err := os.Stat(l.pfade[1]); err != nil {
		t.Errorf("offene trotz gescheitertem Laden verschoben: %v", err)
	}
	if !strings.Contains(l.e.meldung.Text, "not moved to the trash") {
		t.Errorf("Meldung %q", l.e.meldung.Text)
	}
	// Der nächste Wechsel ohne ✕ wirft nichts weg.
	l.fehler = nil
	l.e.leisteOeffne(0)
	l.warte(func() bool { return !l.e.wechsel.laedt && !l.e.bibl.arbeitLaeuft && !k.indexLaeuft })
	if _, err := os.Stat(l.pfade[1]); err != nil {
		t.Errorf("gewöhnlicher Wechsel verschob die alte: %v", err)
	}
}

// Die einzige Aufnahme bleibt offen; das Menü der Ansicht ist dann grau mit Erklärung.
func TestLeisteKreuzEinzigeBleibt(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 1)
	l.e.pfad = func() string { return l.pfade[0] }
	l.setle()
	l.e.leistePapierkorb(0)
	l.warte(func() bool { return !l.e.bibl.arbeitLaeuft })
	if _, err := os.Stat(l.pfade[0]); err != nil {
		t.Errorf("einzige verschoben: %v", err)
	}
	if len(l.aufrufe()) != 0 || !strings.Contains(l.e.meldung.Text, "only capture") {
		t.Errorf("Aufrufe %v, Meldung %q", l.aufrufe(), l.e.meldung.Text)
	}
}

// Andi, 26.11.2026: Nach dem Löschen rückt die nächste Aufnahme unter den stehenden Zeiger; ihr ✕ erscheint am Ende
// der Ruhe ohne Mausbewegung — das Fenster bestellt dafür ein Bild.
func TestLeisteKreuzNachRuheOhneBewegung(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 4)
	k := &l.e.unten
	l.maus(pointer.Move, l.kachel(2), 0)
	l.setle()
	lr := k.loeschRahmen[2]
	l.maus(pointer.Move, mittenPunkt(lr), 0)
	l.setle()
	l.klick(mittenPunkt(lr))
	l.warte(func() bool { return !l.e.bibl.arbeitLaeuft && !k.indexLaeuft && len(k.l.Eintraege()) == 3 })
	// Die Ruhe auf eine feste, ferne Zeit legen, damit ein langsamer Lauf sie nicht schon hinter sich hat.
	l.e.bibl.loeschRuheBis = leisteUhr().Add(time.Hour).Round(0)
	l.setle()
	if !k.loeschRahmen[2].Empty() {
		t.Error("✕ während der Ruhe")
	}
	wann, ok := l.r.WakeupTime()
	for range 20 { // Vorschauen laden noch und bestellen sofortige Bilder
		if !ok || !wann.IsZero() {
			break
		}
		time.Sleep(20 * time.Millisecond)
		l.bild()
		wann, ok = l.r.WakeupTime()
	}
	if !ok || !wann.Equal(l.e.bibl.loeschRuheBis) {
		t.Errorf("kein Bild zum Ende der Ruhe bestellt: %v %v, Soll %v", wann, ok, l.e.bibl.loeschRuheBis)
	}
}

// Kritik K-9: Wird der Wechsel vor dem Wegwerfen der offenen Aufnahme abgebrochen (hier: ein Export-Dialog ging
// während des Ladens auf), bleibt sie im Bestand, und die Meldung sagt auch, dass sie nicht im Papierkorb liegt.
func TestLeisteKreuzAnDerOffenenAbbruch(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	k := &l.e.unten
	l.tor = make(chan struct{})
	l.e.leistePapierkorb(1)
	l.warte(func() bool { return len(l.aufrufe()) == 1 })
	l.e.exportOffen = true
	close(l.tor)
	l.warte(func() bool { return !l.e.wechsel.laedt && !l.e.bibl.arbeitLaeuft && !k.indexLaeuft })
	l.e.exportOffen = false
	if _, err := os.Stat(l.pfade[1]); err != nil {
		t.Errorf("offene trotz Abbruch verschoben: %v", err)
	}
	if !strings.Contains(l.e.meldung.Text, "export") || !strings.Contains(l.e.meldung.Text, "not moved to the trash") {
		t.Errorf("Meldung %q", l.e.meldung.Text)
	}
}
