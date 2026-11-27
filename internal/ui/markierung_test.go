// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// ADR-0036 Zusatz Karte #361: mehrere Vorschauen markieren (Strg/Umschalt-Klick, Strg+Leertaste, Umschalt+Pfeile,
// Strg+A, Esc) in Leiste und voller Ansicht; Entf und Kontextmenü gelten der Markierung, das ✕ nur seiner Vorschau.

import (
	"image"
	"os"
	"slices"
	"strings"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
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
