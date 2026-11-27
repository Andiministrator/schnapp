// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// ADR-0036 Zusatz Karte #361: mehrere Vorschauen markieren (Strg/Umschalt-Klick, Strg+Leertaste, Umschalt+Pfeile,
// Strg+A, Esc) in Leiste und voller Ansicht; Entf und Kontextmenü gelten der Markierung, das ✕ nur seiner Vorschau.

import (
	"errors"
	"image"
	"image/color"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// klickMit klickt mit der linken Taste und den Umschaltern mods an p.
func (l *menueLauf) klickMit(p image.Point, mods key.Modifiers) {
	pos := f32.Pt(float32(p.X), float32(p.Y))
	l.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: pos})
	l.setle()
	l.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: pos,
		Modifiers: mods})
	l.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: pos, Modifiers: mods})
	l.setle()
}

// leisteMarkiert sind die markierten Stellen der Leiste.
func (l *leistenLauf) leisteMarkiert() []int {
	var aus []int
	for i, e := range l.e.unten.l.Eintraege() {
		if l.e.unten.l.Markierung().Enthaelt(e.Pfad) {
			aus = append(aus, i)
		}
	}
	return aus
}

// bibMarkiert sind die markierten Stellen der vollen Ansicht.
func (l *leistenLauf) bibMarkiert() []int {
	var aus []int
	for i, e := range l.e.bibl.r.Eintraege() {
		if l.e.bibl.r.Markierung().Enthaelt(e.Pfad) {
			aus = append(aus, i)
		}
	}
	return aus
}

func imPapierkorb(t *testing.T, pfade ...string) {
	t.Helper()
	for _, p := range pfade {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%s nicht im Papierkorb (%v)", p, err)
		}
	}
}

func nochDa(t *testing.T, pfade ...string) {
	t.Helper()
	for _, p := range pfade {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("%s fehlt: %v", p, err)
		}
	}
}

// Punkt 2: Strg+Klick schaltet und öffnet nicht, Umschalt+Klick markiert den Bereich vom Anker und ersetzt,
// Strg+Umschalt+Klick fügt hinzu; ein Klick ohne Taste öffnet wie bisher und hebt die Markierung auf.
func TestLeisteMarkierenPerKlick(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 5)
	l.klickMit(l.kachel(0), key.ModShortcut)
	l.klickMit(l.kachel(2), key.ModShortcut)
	if got := l.leisteMarkiert(); !slices.Equal(got, []int{0, 2}) {
		t.Fatalf("Strg+Klick 0 und 2: %v", got)
	}
	if a := l.aufrufe(); len(a) != 0 {
		t.Fatalf("Strg+Klick öffnete %v", a)
	}
	if l.e.unten.l.Fokus() != 2 {
		t.Errorf("Fokus %d, Soll 2", l.e.unten.l.Fokus())
	}
	l.klickMit(l.kachel(2), key.ModShortcut) // wieder heraus
	if got := l.leisteMarkiert(); !slices.Equal(got, []int{0}) {
		t.Fatalf("zweites Strg+Klick auf 2: %v", got)
	}
	l.klickMit(l.kachel(3), key.ModShift) // Anker 2 bis 3, ersetzt
	if got := l.leisteMarkiert(); !slices.Equal(got, []int{2, 3}) {
		t.Fatalf("Umschalt+Klick 3 ab Anker 2: %v", got)
	}
	l.klickMit(l.kachel(0), key.ModShortcut|key.ModShift) // Anker 2 bis 0, dazu
	if got := l.leisteMarkiert(); !slices.Equal(got, []int{0, 1, 2, 3}) {
		t.Fatalf("Strg+Umschalt+Klick 0: %v", got)
	}
	if a := l.aufrufe(); len(a) != 0 {
		t.Fatalf("Markieren öffnete %v", a)
	}
	// Klick ohne Taste: öffnet und hebt auf.
	l.klick(l.kachel(4))
	l.ruhig()
	if len(l.leisteMarkiert()) != 0 {
		t.Errorf("Klick ohne Taste ließ die Markierung stehen: %v", l.leisteMarkiert())
	}
	if a := l.aufrufe(); len(a) != 1 || a[0] != l.pfade[4] {
		t.Errorf("Klick ohne Taste öffnete %v", a)
	}
}

// Punkt 3: Strg+Leertaste schaltet die Vorschau mit Fokus, Umschalt+Pfeil markiert vom Anker bis zum Fokus, Strg+A
// alle, Esc hebt auf (ein Esc ohne Markierung geht weiter).
func TestLeisteMarkierenPerTaste(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 4)
	k := &l.e.unten
	l.fokusiere(&k.knoepfe[0])
	l.taste(key.NameSpace, key.ModShortcut)
	if got := l.leisteMarkiert(); !slices.Equal(got, []int{0}) {
		t.Fatalf("Strg+Leertaste: %v", got)
	}
	l.taste(key.NameRightArrow, key.ModShift)
	l.setle()
	l.taste(key.NameRightArrow, key.ModShift)
	l.setle()
	if got := l.leisteMarkiert(); !slices.Equal(got, []int{0, 1, 2}) {
		t.Fatalf("Umschalt+Rechts zweimal: %v", got)
	}
	if !l.fokus(&k.knoepfe[2]) {
		t.Error("Fokus wanderte nicht mit auf Stelle 2")
	}
	l.taste(key.NameLeftArrow, key.ModShift)
	l.setle()
	if got := l.leisteMarkiert(); !slices.Equal(got, []int{0, 1}) {
		t.Fatalf("Umschalt+Links: %v", got)
	}
	l.taste("A", key.ModShortcut)
	if got := l.leisteMarkiert(); !slices.Equal(got, []int{0, 1, 2, 3}) {
		t.Fatalf("Strg+A: %v", got)
	}
	if a := l.aufrufe(); len(a) != 0 {
		t.Fatalf("Tasten öffneten %v", a)
	}
	l.taste(key.NameEscape, 0)
	if n := k.l.Markierung().Anzahl(); n != 0 {
		t.Fatalf("Esc ließ %d markiert", n)
	}
	// Ein Pfeil ohne Umschalt setzt den Anker (Fokus 1 → 2); Umschalt+Pfeil misst von dort (2 → 1).
	l.taste(key.NameRightArrow, 0)
	l.setle()
	l.taste(key.NameLeftArrow, key.ModShift)
	l.setle()
	if got := l.leisteMarkiert(); !slices.Equal(got, []int{1, 2}) {
		t.Fatalf("Anker nach Pfeil: %v", got)
	}
}

// Umschalt+Pfeil ohne Anker misst ab der Vorschau mit Fokus (per Tab dorthin, noch nichts geklickt).
func TestLeisteUmschaltPfeilOhneAnker(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 4)
	k := &l.e.unten
	l.fokusiere(&k.knoepfe[1])
	if k.l.Markierung().Anker() != "" {
		t.Fatal("Vorbedingung: kein Anker")
	}
	l.taste(key.NameRightArrow, key.ModShift)
	l.setle()
	if got := l.leisteMarkiert(); !slices.Equal(got, []int{1, 2}) {
		t.Fatalf("Umschalt+Rechts ab Fokus 1: %v", got)
	}
}

// Punkte 5 und 7: Entf an einer markierten Vorschau legt alle markierten in den Papierkorb; die offene bleibt
// unberührt, wenn sie nicht markiert ist. Die Meldung nennt die Zahl.
func TestLeisteEntfMarkierte(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 4)
	k := &l.e.unten
	l.klickMit(l.kachel(2), key.ModShortcut)
	l.klickMit(l.kachel(3), key.ModShortcut)
	l.fokusiere(&k.knoepfe[3])
	l.taste(key.NameDeleteForward, 0)
	l.ruhig()
	imPapierkorb(t, l.pfade[2], l.pfade[3])
	nochDa(t, l.pfade[0], l.pfade[1])
	if n := papierkorbDateien(t, l); n != 2 {
		t.Errorf("Papierkorb hat %d Dateien", n)
	}
	if a := l.aufrufe(); len(a) != 0 {
		t.Errorf("Wechsel ohne die offene: %v", a)
	}
	if !strings.Contains(l.e.meldung.Text, "2 captures") || l.e.meldung.Fehler {
		t.Errorf("Meldung %q", l.e.meldung.Text)
	}
	l.warte(func() bool { return len(k.l.Eintraege()) == 2 })
	if k.l.Markierung().Anzahl() != 0 {
		t.Errorf("Markierung der weggeworfenen blieb: %d", k.l.Markierung().Anzahl())
	}
}

// Entf an einer nicht markierten Vorschau gilt nur ihr, auch wenn andere markiert sind; das ✕ gilt immer nur seiner
// Vorschau (Punkt 5).
func TestLeisteEntfUndKreuzNurEine(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 4)
	k := &l.e.unten
	l.klickMit(l.kachel(0), key.ModShortcut)
	l.klickMit(l.kachel(2), key.ModShortcut)
	l.maus(pointer.Move, l.kachel(2), 0)
	l.setle()
	l.klick(mittenPunkt(k.loeschRahmen[2]))
	l.ruhig()
	imPapierkorb(t, l.pfade[2])
	nochDa(t, l.pfade[0], l.pfade[3])
	l.warte(func() bool { return len(k.l.Eintraege()) == 3 })
	if got := l.leisteMarkiert(); !slices.Equal(got, []int{0}) {
		t.Errorf("Markierung nach dem ✕: %v", got)
	}
	l.e.bibl.loeschRuheBis = leisteUhr() // Ruhe nach dem Löschen abkürzen
	l.fokusiere(&k.knoepfe[2])           // jetzt die ehemals vierte, nicht markiert
	l.taste(key.NameDeleteForward, 0)
	l.ruhig()
	imPapierkorb(t, l.pfade[3])
	nochDa(t, l.pfade[0])
}

// Punkt 7: Ist die offene markiert, wechselt der Editor zuerst zu einer nicht markierten Nachbarin und legt danach die
// offene und die übrigen markierten in den Papierkorb.
func TestLeisteEntfMarkierteMitOffener(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 4)
	k := &l.e.unten
	if !k.l.IstAktuell(1) {
		t.Fatal("Vorbedingung: Stelle 1 ist offen")
	}
	l.klickMit(l.kachel(1), key.ModShortcut)
	l.klickMit(l.kachel(2), key.ModShortcut)
	l.fokusiere(&k.knoepfe[1])
	l.taste(key.NameDeleteForward, 0)
	l.ruhig()
	// Nachbarinnen der offenen (1) nach Abstand: 2 ist markiert, also 0.
	if a := l.aufrufe(); len(a) != 1 || a[0] != l.pfade[0] {
		t.Fatalf("geladen %v, Soll die nicht markierte Nachbarin %s", a, l.pfade[0])
	}
	imPapierkorb(t, l.pfade[1], l.pfade[2])
	nochDa(t, l.pfade[0], l.pfade[3])
	if !strings.Contains(l.e.meldung.Text, "2 captures") {
		t.Errorf("Meldung %q", l.e.meldung.Text)
	}
}

// Punkt 7: Sind alle markiert, bleibt die offene, die übrigen gehen, und die Meldung sagt es.
func TestLeisteAlleMarkiertOffeneBleibt(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	k := &l.e.unten
	l.fokusiere(&k.knoepfe[0])
	l.taste("A", key.ModShortcut)
	l.taste(key.NameDeleteForward, 0)
	l.ruhig()
	imPapierkorb(t, l.pfade[0], l.pfade[2])
	nochDa(t, l.pfade[1])
	if a := l.aufrufe(); len(a) != 0 {
		t.Errorf("Wechsel ohne nicht markierte Nachbarin: %v", a)
	}
	if !strings.Contains(l.e.meldung.Text, "2 captures") || !strings.Contains(l.e.meldung.Text, "stays") {
		t.Errorf("Meldung %q", l.e.meldung.Text)
	}
}

// Scheitert der Wechsel vor dem Papierkorb, geht keine der markierten, und die Meldung sagt es in der Mehrzahl.
func TestLeisteMarkierteWechselScheitert(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	k := &l.e.unten
	l.fehler = os.ErrPermission
	l.klickMit(l.kachel(1), key.ModShortcut)
	l.klickMit(l.kachel(2), key.ModShortcut)
	l.fokusiere(&k.knoepfe[1])
	l.taste(key.NameDeleteForward, 0)
	l.ruhig()
	nochDa(t, l.pfade...)
	if !strings.Contains(l.e.meldung.Text, "selected captures were not moved") {
		t.Errorf("Meldung %q", l.e.meldung.Text)
	}
}

// Punkt 6: Das Kontextmenü an einer markierten Vorschau unter mehreren zeigt Dateimanager und Papierkorb für alle; der
// Dateimanager bekommt alle in einem Aufruf. Ein Rechtsklick auf eine nicht markierte hebt die Markierung auf.
func TestKontextMarkierteLeiste(t *testing.T) {
	d := mitDateimanager(t, nil)
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 4)
	l.klickMit(l.kachel(0), key.ModShortcut)
	l.klickMit(l.kachel(2), key.ModShortcut)
	l.rechtsklick(l.kachel(2))
	if got := l.aktionen(); !slices.Equal(got, []aktion{aktVorschauZeigen, aktVorschauPapierkorb}) {
		t.Fatalf("Einträge %v", got)
	}
	for _, en := range l.e.menueEintraege() {
		if !strings.Contains(en.text, "2 captures") || !en.an {
			t.Errorf("Eintrag %q (an %v)", en.text, en.an)
		}
	}
	l.eintragKlick(aktVorschauZeigen)
	l.warte(func() bool { return l.e.dateimanagerAbholen() })
	d.mu.Lock()
	rufe := d.rufe
	d.mu.Unlock()
	if got := d.gerufen(); !slices.Equal(got, []string{l.pfade[0], l.pfade[2]}) || rufe != 1 {
		t.Errorf("Dateimanager %v in %d Aufrufen", got, rufe)
	}
	l.rechtsklick(l.kachel(2))
	l.eintragKlick(aktVorschauPapierkorb)
	l.ruhig()
	imPapierkorb(t, l.pfade[0], l.pfade[2])
	nochDa(t, l.pfade[1], l.pfade[3])
	// Rechtsklick auf eine nicht markierte: Markierung weg, Menü der einen.
	l.warte(func() bool { return len(l.e.unten.l.Eintraege()) == 2 })
	l.e.bibl.loeschRuheBis = leisteUhr()
	l.klickMit(l.kachel(0), key.ModShortcut)
	l.rechtsklick(l.kachel(1))
	if l.e.unten.l.Markierung().Anzahl() != 0 {
		t.Error("Rechtsklick auf eine nicht markierte ließ die Markierung stehen")
	}
	if got := l.aktionen(); !slices.Contains(got, aktVorschauOeffnen) {
		t.Errorf("Menü der einen erwartet, Einträge %v", got)
	}
}

// Volle Ansicht: Strg/Umschalt-Druck markiert und öffnet nie (auch nicht als Doppelklick), Entf legt alle markierten in
// den Papierkorb, die Ansicht bleibt offen; Esc hebt erst auf, dann schließt es.
func TestBibliothekMarkieren(t *testing.T) {
	l := bibLauf(t, 5)
	b := &l.e.bibl
	l.klickMit(l.bibKachel(0), key.ModShortcut)
	l.klickMit(l.bibKachel(0), key.ModShortcut) // zweiter Druck: kein Doppelklick, wieder heraus
	l.klickMit(l.bibKachel(2), key.ModShortcut)
	l.klickMit(l.bibKachel(4), key.ModShift)
	if got := l.bibMarkiert(); !slices.Equal(got, []int{2, 3, 4}) {
		t.Fatalf("Markierung %v", got)
	}
	if a := l.aufrufe(); len(a) != 0 {
		t.Fatalf("Markieren öffnete %v", a)
	}
	if !strings.Contains(b.hinweis, "3 captures selected") {
		t.Errorf("Hinweis %q", b.hinweis)
	}
	l.taste(key.NameEscape, 0)
	if !b.offen || b.r.Markierung().Anzahl() != 0 {
		t.Fatalf("Esc: offen %v, markiert %d", b.offen, b.r.Markierung().Anzahl())
	}
	// Tastatur: Strg+Leertaste, Umschalt+Runter/Rechts, Strg+A.
	b.r.SetzeFokus(0)
	l.taste(key.NameSpace, key.ModShortcut)
	l.taste(key.NameRightArrow, key.ModShift)
	if got := l.bibMarkiert(); !slices.Equal(got, []int{0, 1}) {
		t.Fatalf("Strg+Leertaste, Umschalt+Rechts: %v", got)
	}
	l.taste("A", key.ModShortcut)
	if n := b.r.Markierung().Anzahl(); n != 5 {
		t.Fatalf("Strg+A: %d", n)
	}
	l.taste(key.NameEscape, 0)
	// Markieren und wegwerfen (ohne die offene, Stelle 1).
	l.klickMit(l.bibKachel(3), key.ModShortcut)
	l.klickMit(l.bibKachel(4), key.ModShortcut)
	l.taste(key.NameDeleteForward, 0)
	l.bibRuhig()
	imPapierkorb(t, l.pfade[3], l.pfade[4])
	nochDa(t, l.pfade[0], l.pfade[1], l.pfade[2])
	if !b.offen {
		t.Error("Ansicht schloss nach dem Papierkorb")
	}
	l.warte(func() bool { return len(b.r.Eintraege()) == 3 })
	// Ein Klick ins Leere hebt auf.
	l.klickMit(l.bibKachel(0), key.ModShortcut)
	l.klickMit(image.Pt(b.sicht.Max.X-20, b.sicht.Max.Y-20), 0)
	if n := b.r.Markierung().Anzahl(); n != 0 {
		t.Errorf("Klick ins Leere ließ %d markiert", n)
	}
	l.taste(key.NameEscape, 0)
	if b.offen {
		t.Error("Esc ohne Markierung schloss die Ansicht nicht")
	}
}

// Punkt 1: Die Ansicht beginnt ohne Markierung; Suche und Zeitraum tragen Markierungen nicht gezeigter Aufnahmen aus.
func TestBibliothekMarkierungNurGezeigt(t *testing.T) {
	l := bibLauf(t, 3)
	b := &l.e.bibl
	l.klickMit(l.bibKachel(0), key.ModShortcut)
	l.klickMit(l.bibKachel(2), key.ModShortcut)
	l.e.bibl.suche = "gibt-es-nicht"
	l.e.bibliothekNeuLesen()
	l.bibRuhig()
	if n := b.r.Markierung().Anzahl(); n != 0 {
		t.Errorf("gefiltert: %d markiert", n)
	}
	l.e.bibl.suche = ""
	l.e.bibliothekNeuLesen()
	l.bibRuhig()
	l.klickMit(l.bibKachel(0), key.ModShortcut)
	l.taste("L", key.ModShortcut|key.ModShift) // schließt mit Markierung (nicht über Esc, das sie zuerst leert)
	if b.offen {
		t.Fatal("Ansicht noch offen")
	}
	l.klickMit(l.kachel(0), key.ModShortcut) // Leiste: eigene Markierung
	l.taste("L", key.ModShortcut|key.ModShift)
	l.bibRuhig()
	if n := b.r.Markierung().Anzahl(); n != 0 {
		t.Errorf("neu geöffnete Ansicht mit %d markiert", n)
	}
	if n := l.e.unten.l.Markierung().Anzahl(); n != 1 {
		t.Errorf("Markierung der Leiste: %d", n)
	}
}

// Das Menü Bearbeiten der Ansicht nennt die Zahl, wenn die Kachel mit Fokus markiert ist.
func TestBibliothekMenueMarkierte(t *testing.T) {
	l := bibLauf(t, 3)
	l.klickMit(l.bibKachel(0), key.ModShortcut)
	l.klickMit(l.bibKachel(2), key.ModShortcut)
	for _, en := range l.e.bibliothekBearbeiten() {
		if en.akt == aktBibPapierkorb && (!strings.Contains(en.text, "2 captures") || !en.an) {
			t.Errorf("Eintrag %q (an %v)", en.text, en.an)
		}
	}
}

// --- Nach der Kritik zu Karte #361 (F-887, F-888, F-890 bis F-893) ---

// torAuf hält die Ladefunktion an, bis das gelieferte Tor schließt.
func (l *leistenLauf) torAuf() chan struct{} {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.tor = make(chan struct{})
	return l.tor
}

// F-887: Hebt Esc die Markierung auf, während die Nachbarin lädt, geht nach der Übernahme nur die bisher offene in den
// Papierkorb; die weitere bleibt.
func TestLeisteMitnehmenNachEsc(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 4)
	k := &l.e.unten
	l.klickMit(l.kachel(1), key.ModShortcut)
	l.klickMit(l.kachel(2), key.ModShortcut)
	l.fokusiere(&k.knoepfe[1])
	tor := l.torAuf()
	l.taste(key.NameDeleteForward, 0)
	l.warte(func() bool { return l.e.wechsel.laedt })
	l.taste(key.NameEscape, 0)
	if n := k.l.Markierung().Anzahl(); n != 0 {
		t.Fatalf("Esc während des Ladens ließ %d markiert", n)
	}
	close(tor)
	l.ruhig()
	imPapierkorb(t, l.pfade[1])
	nochDa(t, l.pfade[0], l.pfade[2], l.pfade[3])
}

// F-887: In der vollen Ansicht gilt dasselbe für eine Suche, die während des Ladens eine markierte ausblendet.
func TestBibliothekMitnehmenNachSuche(t *testing.T) {
	l := bibLauf(t, 5)
	b := &l.e.bibl
	if _, err := l.b.SetzeSchlagworte(l.rel(2), []string{"bleibt-gezeigt"}); err != nil {
		t.Fatal(err)
	}
	l.klickMit(l.bibKachel(1), key.ModShortcut) // die offene
	l.klickMit(l.bibKachel(2), key.ModShortcut)
	l.klickMit(l.bibKachel(3), key.ModShortcut)
	tor := l.torAuf()
	l.taste(key.NameDeleteForward, 0)
	l.warte(func() bool { return l.e.wechsel.laedt })
	b.suche = "bleibt"
	l.e.bibliothekNeuLesen()
	l.warte(func() bool { return !b.indexLaeuft && len(b.r.Eintraege()) == 1 })
	close(tor)
	l.ruhig()
	imPapierkorb(t, l.pfade[1], l.pfade[2])
	nochDa(t, l.pfade[0], l.pfade[3], l.pfade[4])
}

// F-888: Liest die Leiste zwischen Zeichnen und Taste neu, gelten Strg+Leertaste und Umschalt+Pfeil der gezeigten
// Aufnahme, nicht der, die jetzt an ihrer Stelle steht.
func TestLeisteMarkierTasteNachNeulesen(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 4)
	k := &l.e.unten
	l.fokusiere(&k.knoepfe[2])
	gezeigt := k.gezeigt[2]
	es := k.l.Eintraege()
	k.l.Setze(es[1:]) // neuer Stand ohne die erste: alle rücken um eins, gezeichnet ist noch der alte
	l.r.Queue(key.Event{Name: key.NameSpace, Modifiers: key.ModShortcut, State: key.Press})
	l.bild()
	if m := k.l.Markierung(); !m.Enthaelt(gezeigt) || m.Anzahl() != 1 {
		t.Fatalf("Strg+Leertaste markierte %v, Soll %s", m.Pfade(k.l.Eintraege()), gezeigt)
	}
	// Umschalt+Pfeil: Anker ist die gezeigte, der Bereich reicht bis zu ihrer rechten Nachbarin im neuen Stand.
	k.l.Markierung().Leere()
	k.l.Markierung().SetzeAnker("")
	l.setle()
	l.fokusiere(&k.knoepfe[1]) // im neuen Stand die ehemals dritte
	gezeigt = k.gezeigt[1]
	k.l.Setze(es) // die erste ist wieder da, wieder ohne neues Bild
	l.r.Queue(key.Event{Name: key.NameRightArrow, Modifiers: key.ModShift, State: key.Press})
	l.bild()
	if got := k.l.Markierung().Pfade(k.l.Eintraege()); !slices.Equal(got, []string{gezeigt, es[3].Pfad}) {
		t.Errorf("Umschalt+Rechts markierte %v, Soll %s und %s", got, gezeigt, es[3].Pfad)
	}
}

// F-890: Esc hebt die Markierung der Leiste auch mit Fokus auf der Leinwand auf; erst das nächste Esc wirkt dort wie
// bisher (Esc in Stufen der Sitzung).
func TestLeisteEscOhneFokusHebtAuf(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 4)
	k := &l.e.unten
	l.klickMit(l.kachel(2), key.ModShortcut)
	l.klickMit(l.kachel(3), key.ModShortcut)
	l.e.s.SetzeWerkzeug(canvas.WerkzeugPfeil)
	l.fokusiere(l.e)
	vorA, vorW := l.e.s.Auswahl, l.e.s.Werkzeug
	l.taste(key.NameEscape, 0)
	if n := k.l.Markierung().Anzahl(); n != 0 {
		t.Fatalf("Esc mit Fokus auf der Leinwand ließ %d markiert", n)
	}
	if l.e.s.Auswahl != vorA || l.e.s.Werkzeug != vorW {
		t.Error("das erste Esc wirkte auch auf die Leinwand")
	}
	for range 2 { // Auswahl aufheben, zurück zum Auswählen
		l.taste(key.NameEscape, 0)
	}
	if l.e.s.Auswahl == vorA && l.e.s.Werkzeug == vorW {
		t.Errorf("weitere Esc erreichten die Leinwand nicht: Auswahl %d, Werkzeug %v", l.e.s.Auswahl, l.e.s.Werkzeug)
	}
}

// F-891: Der Hinweis nennt Entf nur an der Vorschau mit Tastaturfokus; über einer markierten unter dem bloßen Zeiger
// sagt er Zahl und Esc (Leiste und volle Ansicht).
func TestMarkierHinweisNurMitFokusMitEntf(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 4)
	l.klickMit(l.kachel(0), key.ModShortcut)
	l.klickMit(l.kachel(2), key.ModShortcut) // Fokus an 2
	if h := l.e.hinweis(); !strings.Contains(h, "2 captures selected") || !strings.Contains(h, "Del") {
		t.Errorf("Hinweis mit Fokus: %q", h)
	}
	l.fokusiere(l.e)
	l.maus(pointer.Move, l.kachel(0), 0)
	l.setle()
	if h := l.e.hinweis(); !strings.Contains(h, "2 captures selected") || strings.Contains(h, "Del") {
		t.Errorf("Hinweis unter dem Zeiger: %q", h)
	}

	v := bibLauf(t, 4)
	v.klickMit(v.bibKachel(0), key.ModShortcut)
	v.klickMit(v.bibKachel(2), key.ModShortcut) // Fokus an 2
	if h := v.e.bibliothekHinweis(); !strings.Contains(h, "Del") {
		t.Errorf("Ansicht, Zeiger auf der Kachel mit Fokus: %q", h)
	}
	v.maus(pointer.Move, v.bibKachel(0), 0)
	v.setle()
	if h := v.e.bibliothekHinweis(); !strings.Contains(h, "2 captures selected") || strings.Contains(h, "Del") {
		t.Errorf("Ansicht, Zeiger auf einer anderen markierten: %q", h)
	}
}

// F-892 (Mutant A): Lässt sich die erste nicht markierte Nachbarin nicht laden, nimmt die Kette der Nachbarinnen die
// übrigen markierten mit.
func TestLeisteMitnehmenUeberKette(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 5)
	k := &l.e.unten
	l.mu.Lock()
	l.fehlerFuer = map[string]error{l.pfade[0]: os.ErrPermission}
	l.mu.Unlock()
	l.klickMit(l.kachel(1), key.ModShortcut)
	l.klickMit(l.kachel(2), key.ModShortcut)
	l.fokusiere(&k.knoepfe[1])
	l.taste(key.NameDeleteForward, 0)
	l.ruhig()
	if a := l.aufrufe(); !slices.Equal(a, []string{l.pfade[0], l.pfade[3]}) {
		t.Errorf("geladen %v", a)
	}
	imPapierkorb(t, l.pfade[1], l.pfade[2])
	nochDa(t, l.pfade[0], l.pfade[3], l.pfade[4])
}

// F-892 (Mutant R): Liegen markierte um die offene, sucht die Nachbarsuche weiter als Abstand 1.
func TestLeisteNachbarinHinterMarkierten(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 5)
	k := &l.e.unten
	for _, i := range []int{0, 1, 2} {
		l.klickMit(l.kachel(i), key.ModShortcut)
	}
	l.fokusiere(&k.knoepfe[1])
	l.taste(key.NameDeleteForward, 0)
	l.ruhig()
	if a := l.aufrufe(); !slices.Equal(a, []string{l.pfade[3]}) {
		t.Errorf("geladen %v, Soll %s", a, l.pfade[3])
	}
	imPapierkorb(t, l.pfade[0], l.pfade[1], l.pfade[2])
	nochDa(t, l.pfade[3], l.pfade[4])
}

// F-892 (Mutant B): Zeigt die gefilterte Ansicht nur markierte, ist die Nachbarin die der Leiste.
func TestBibliothekGefiltertMarkierteNachbarinDerLeiste(t *testing.T) {
	l := bibLauf(t, 4)
	b := &l.e.bibl
	for _, i := range []int{1, 2} {
		if _, err := l.b.SetzeSchlagworte(l.rel(i), []string{"gruppe"}); err != nil {
			t.Fatal(err)
		}
	}
	b.suche = "gruppe"
	l.e.bibliothekNeuLesen()
	l.bibRuhig()
	if len(b.r.Eintraege()) != 2 {
		t.Fatalf("gefiltert %d Einträge", len(b.r.Eintraege()))
	}
	l.klickMit(l.bibKachel(0), key.ModShortcut) // die offene
	l.klickMit(l.bibKachel(1), key.ModShortcut)
	l.taste(key.NameDeleteForward, 0)
	l.ruhig()
	if a := l.aufrufe(); !slices.Equal(a, []string{l.pfade[0]}) {
		t.Errorf("geladen %v, Soll die Nachbarin der Leiste %s", a, l.pfade[0])
	}
	imPapierkorb(t, l.pfade[1], l.pfade[2])
	nochDa(t, l.pfade[0], l.pfade[3])
}

// F-892 (Mutant C): Läuft noch eine Arbeit, legt papierkorbMehrere nichts in den Papierkorb und sagt warum.
func TestPapierkorbMehrereGesperrt(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 4)
	l.klickMit(l.kachel(2), key.ModShortcut)
	l.klickMit(l.kachel(3), key.ModShortcut)
	l.e.bibl.arbeitLaeuft = true
	l.e.papierkorbMarkiert(true)
	l.e.bibl.arbeitLaeuft = false
	l.ruhig()
	nochDa(t, l.pfade...)
	if !strings.Contains(l.e.meldung.Text, "still working") {
		t.Errorf("Meldung %q", l.e.meldung.Text)
	}
}

// F-892 (Mutanten D, E): Fehlt eine markierte schon, meldet der Papierkorb „1 of 2“ als Fehler, und der Abgleich trägt
// die fehlende aus der Leiste aus.
func TestLeistePapierkorbTeilfehler(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 4)
	k := &l.e.unten
	l.klickMit(l.kachel(2), key.ModShortcut)
	l.klickMit(l.kachel(3), key.ModShortcut)
	if err := os.Remove(l.pfade[3]); err != nil {
		t.Fatal(err)
	}
	l.fokusiere(&k.knoepfe[3])
	l.taste(key.NameDeleteForward, 0)
	l.ruhig()
	if !strings.Contains(l.e.meldung.Text, "1 of 2") || !l.e.meldung.Fehler {
		t.Errorf("Meldung %+v", l.e.meldung)
	}
	imPapierkorb(t, l.pfade[2])
	l.warte(func() bool { return !k.indexLaeuft && len(k.l.Eintraege()) == 2 })
}

// F-892 (Mutant G): Der Papierkorb mehrerer vergisst den gemerkten Verlauf jeder weggeworfenen Aufnahme (F-838).
func TestPapierkorbMehrereVergisstVerlauf(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 4)
	k := &l.e.unten
	for _, i := range []int{2, 3} {
		l.e.verlaeufe.merke(gemerkteSitzung{pfad: l.pfade[i], d: l.e.s.D, bedarf: 1})
	}
	l.klickMit(l.kachel(2), key.ModShortcut)
	l.klickMit(l.kachel(3), key.ModShortcut)
	l.fokusiere(&k.knoepfe[3])
	l.taste(key.NameDeleteForward, 0)
	l.ruhig()
	for _, i := range []int{2, 3} {
		if _, ok := l.e.verlaeufe.hole(l.pfade[i]); ok {
			t.Errorf("Verlauf von %s noch gemerkt", l.pfade[i])
		}
	}
}

// F-892 (Mutanten H, F-893): Scheitert das Speichern vor dem Wechsel, sagt die Meldung in der Mehrzahl, dass keine
// markierte in den Papierkorb ging, und der Wechsel vergisst alle Merker. Ebenso bei einem Hinweis des Speicherns.
func TestLeisteMarkierteSpeichernScheitert(t *testing.T) {
	for _, fall := range []struct {
		name  string
		fehlt error
		g     Gespeichert
	}{
		{"Fehler", errors.New("Platte voll"), Gespeichert{}},
		{"Hinweis", nil, Gespeichert{Hinweis: "als neue Aufnahme gespeichert"}},
	} {
		t.Run(fall.name, func(t *testing.T) {
			l := neuerLeistenLauf(t, image.Pt(1200, 800), 4)
			k := &l.e.unten
			l.e.speichern = func(*document.Dokument) (Gespeichert, error) { return fall.g, fall.fehlt }
			l.klickMit(l.kachel(1), key.ModShortcut)
			l.klickMit(l.kachel(2), key.ModShortcut)
			l.fokusiere(&k.knoepfe[1])
			l.taste(key.NameDeleteForward, 0)
			l.ruhig()
			nochDa(t, l.pfade...)
			if len(l.aufrufe()) != 0 {
				t.Errorf("geladen %v", l.aufrufe())
			}
			if !strings.Contains(l.e.meldung.Text, "selected captures were not moved") {
				t.Errorf("Meldung %q", l.e.meldung.Text)
			}
			if w := l.e.wechsel; w.ziel != "" || w.papierkorb || w.mitnehmen != nil || w.weitere != nil || w.ausLeiste {
				t.Errorf("Merker des Wechsels nach dem Abbruch: %+v", w)
			}
		})
	}
}

// F-892 (Mutant I): Wird die offene geändert, während die Nachbarin lädt, sagt die Meldung in der Mehrzahl, dass keine
// markierte ging.
func TestLeisteMarkierteGeaendertWaehrendLaden(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 4)
	k := &l.e.unten
	l.klickMit(l.kachel(1), key.ModShortcut)
	l.klickMit(l.kachel(2), key.ModShortcut)
	l.fokusiere(&k.knoepfe[1])
	tor := l.torAuf()
	l.taste(key.NameDeleteForward, 0)
	l.warte(func() bool { return l.e.wechsel.laedt })
	if _, err := l.e.s.D.Hinzufuegen(document.Text{Ort: document.Pt(5, 15), Inhalt: "neu", Schriftgrad: 12,
		Farbe: color.NRGBA{A: 255}}); err != nil {
		t.Fatal(err)
	}
	close(tor)
	l.ruhig()
	nochDa(t, l.pfade...)
	if !strings.Contains(l.e.meldung.Text, "changed meanwhile") ||
		!strings.Contains(l.e.meldung.Text, "selected captures were not moved") {
		t.Errorf("Meldung %q", l.e.meldung.Text)
	}
}

// F-892 (Mutant T): Schon beim Start des Wechsels ruht der Papierkorb (loeschRuhe), nicht erst nach dem Laden.
func TestLeisteMarkierteWechselRuht(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 4)
	k := &l.e.unten
	l.klickMit(l.kachel(1), key.ModShortcut)
	l.klickMit(l.kachel(2), key.ModShortcut)
	l.fokusiere(&k.knoepfe[1])
	tor := l.torAuf()
	l.taste(key.NameDeleteForward, 0)
	l.warte(func() bool { return l.e.wechsel.laedt })
	if !l.e.loeschRuht() {
		t.Error("während des Wechsels ruht der Papierkorb nicht")
	}
	close(tor)
	l.ruhig()
}

// F-892 (Mutanten J und Umschalt+F10): Rechtsklick und Umschalt+F10 an einer markierten Vorschau der vollen Ansicht
// bzw. der Leiste zeigen das Menü der Markierung.
func TestKontextMarkierteAnsichtUndUmschaltF10(t *testing.T) {
	mehrere := []aktion{aktVorschauZeigen, aktVorschauPapierkorb}
	v := bibLauf(t, 4)
	v.klickMit(v.bibKachel(0), key.ModShortcut)
	v.klickMit(v.bibKachel(2), key.ModShortcut)
	v.rechtsklick(v.bibKachel(2))
	if got := v.aktionen(); !slices.Equal(got, mehrere) {
		t.Errorf("Rechtsklick in der Ansicht: %v", got)
	}
	v.taste(key.NameEscape, 0) // Liste zu
	v.e.bibl.r.SetzeFokus(2)
	v.taste(key.NameF10, key.ModShift)
	if got := v.aktionen(); !v.e.menue.offen || !slices.Equal(got, mehrere) {
		t.Errorf("Umschalt+F10 in der Ansicht: offen %v, %v", v.e.menue.offen, got)
	}

	l := neuerLeistenLauf(t, image.Pt(1200, 800), 4)
	l.klickMit(l.kachel(0), key.ModShortcut)
	l.klickMit(l.kachel(2), key.ModShortcut)
	l.fokusiere(&l.e.unten.knoepfe[2])
	l.taste(key.NameF10, key.ModShift)
	if got := l.aktionen(); !l.e.menue.offen || !slices.Equal(got, mehrere) {
		t.Errorf("Umschalt+F10 in der Leiste: offen %v, %v", l.e.menue.offen, got)
	}
}

// F-892 (Mutanten O, P): Das Menü mehrerer zählt beim Auslösen, wie die Markierung jetzt steht, und sein Papierkorb
// wirkt in der Ruhe nach einem Löschen nicht.
func TestKontextMehrereZahlUndRuhe(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 4)
	l.klickMit(l.kachel(0), key.ModShortcut)
	l.klickMit(l.kachel(2), key.ModShortcut)
	l.rechtsklick(l.kachel(2))
	l.e.unten.l.Markierung().Umschalte(l.rel(3))
	for _, en := range l.e.menueEintraege() {
		if !strings.Contains(en.text, "3 captures") {
			t.Errorf("Eintrag %q, Soll mit der Zahl von jetzt", en.text)
		}
	}
	l.e.bibl.loeschRuheBis = leisteUhr().Add(time.Hour)
	l.e.vorschauWirkung(aktVorschauPapierkorb)(l.gtx())
	l.ruhig()
	nochDa(t, l.pfade...)
}

// F-892 (Mutant Q): Das ✕ einer markierten Kachel der vollen Ansicht nimmt nur sie.
func TestBibliothekKreuzNurEine(t *testing.T) {
	l := bibLauf(t, 4)
	b := &l.e.bibl
	l.klickMit(l.bibKachel(2), key.ModShortcut)
	l.klickMit(l.bibKachel(3), key.ModShortcut)
	l.maus(pointer.Move, l.bibKachel(3), 0)
	l.setle()
	lr := b.loesch[3-b.von]
	if lr.Empty() {
		t.Fatal("kein ✕ unter dem Zeiger")
	}
	l.bibDruck(mittenPunkt(lr), time.Second)
	l.bibRuhig()
	imPapierkorb(t, l.pfade[3])
	nochDa(t, l.pfade[0], l.pfade[1], l.pfade[2])
}

// F-892 (Mutanten L, M, N): Anker der vollen Ansicht — das Öffnen beginnt ohne, ein Pfeil ohne Umschalt setzt ihn,
// Umschalt+Pfeil ohne Anker misst ab dem Fokus.
func TestBibliothekAnker(t *testing.T) {
	l := bibLauf(t, 5)
	b := &l.e.bibl
	// N: ohne Anker ab dem Fokus.
	b.r.SetzeFokus(1)
	l.taste(key.NameRightArrow, key.ModShift)
	if got := l.bibMarkiert(); !slices.Equal(got, []int{1, 2}) {
		t.Errorf("Umschalt+Rechts ohne Anker ab Fokus 1: %v", got)
	}
	l.taste(key.NameEscape, 0)
	// M: Strg+Leertaste setzt den Anker auf 0; zwei Pfeile ohne Umschalt setzen ihn auf 2.
	b.r.SetzeFokus(0)
	l.taste(key.NameSpace, key.ModShortcut)
	l.taste(key.NameRightArrow, 0)
	l.taste(key.NameRightArrow, 0)
	l.taste(key.NameRightArrow, key.ModShift)
	if got := l.bibMarkiert(); !slices.Equal(got, []int{2, 3}) {
		t.Errorf("Umschalt+Rechts nach Pfeilen: %v, Soll ab dem neuen Anker 2", got)
	}
	// L: Anker 2 aus Strg+Klick, Ansicht zu und wieder auf: Umschalt+Klick markiert nur die eine.
	l.taste(key.NameEscape, 0)
	l.klickMit(l.bibKachel(2), key.ModShortcut)
	l.taste("L", key.ModShortcut|key.ModShift)
	l.taste("L", key.ModShortcut|key.ModShift)
	l.warte(func() bool { return b.offen && !b.indexLaeuft && len(b.r.Eintraege()) == 5 })
	l.klickMit(l.bibKachel(4), key.ModShift)
	if got := l.bibMarkiert(); !slices.Equal(got, []int{4}) {
		t.Errorf("Umschalt+Klick nach neuem Öffnen: %v, Soll nur 4", got)
	}
}
