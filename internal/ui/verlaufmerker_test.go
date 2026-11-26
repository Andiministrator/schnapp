// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// ADR-0036 Zusatz Karte #356: Verlauf je Aufnahme beim Wechsel im selben Fenster.

import (
	"errors"
	"image"
	"os"
	"sync"
	"testing"
	"time"

	"gioui.org/io/pointer"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/library"
)

// fortsetzerLauf hängt an einen Leistenlauf ein Fortsetzen wie cmd/schnapp: gemerktes Dokument nur bei gleichem
// Dateistand, sonst das frisch geladene.
type fortsetzerLauf struct {
	mu      sync.Mutex
	aufrufe int
}

// Wie cmd/schnapp führt jedes Geoeffnet den Dateistand vom Laden (Geoeffnet.Stand, F-834); die Speicherfunktion des
// Tests schreibt nicht, der Stand bleibt also der vom Laden.
func mitStand(g Geoeffnet, pfad string) Geoeffnet {
	st, _ := library.StandVon(pfad)
	g.Stand = func() library.Stand { return st }
	return g
}

func (f *fortsetzerLauf) anbinden(l *leistenLauf) {
	if p := l.e.pfad(); p != "" {
		st, _ := library.StandVon(p)
		l.e.stand = func() library.Stand { return st }
	}
	l.e.bib.Oeffnen = func(pfad string) (Geoeffnet, error) {
		g, err := l.oeffnen(pfad)
		return mitStand(g, pfad), err
	}
	l.e.bib.Fortsetzen = func(pfad string, d *document.Dokument, stand library.Stand) (Geoeffnet, error) {
		f.mu.Lock()
		f.aufrufe++
		f.mu.Unlock()
		g, err := l.oeffnen(pfad)
		if err != nil {
			return g, err
		}
		if st, err := library.StandVon(pfad); err == nil && st == stand {
			g.Dokument, g.Fortgesetzt = d, true
		}
		return mitStand(g, pfad), nil
	}
}

func (f *fortsetzerLauf) zahl() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.aufrufe
}

func TestWechselZurueckBehaeltVerlauf(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	f := &fortsetzerLauf{}
	f.anbinden(l)
	l.e.s.Gespeichert()
	docA := l.e.s.D
	if _, err := docA.Hinzufuegen(wzKasten()); err != nil {
		t.Fatal(err)
	}
	l.ruhig()
	l.klick(l.kachel(0)) // weg von der offenen (Stelle 1): speichern, dann laden
	l.ruhig()
	if l.e.s.D == docA {
		t.Fatal("kein Wechsel")
	}
	if len(l.e.verlaeufe.liste) != 1 {
		t.Fatalf("%d gemerkt, Soll 1", len(l.e.verlaeufe.liste))
	}
	l.klick(l.kachel(1)) // zurück
	l.ruhig()
	if f.zahl() != 1 {
		t.Fatalf("Fortsetzen %d-mal gerufen, Soll 1", f.zahl())
	}
	if l.e.s.D != docA {
		t.Fatal("zurück: nicht das gemerkte Dokument")
	}
	if l.e.s.Geaendert() || !l.e.s.D.KannRueckgaengig() {
		t.Fatalf("zurück: geändert %v, Rückgängig %v", l.e.s.Geaendert(), l.e.s.D.KannRueckgaengig())
	}
	if len(l.e.verlaeufe.liste) != 0 {
		// die andere (Stelle 0) hatte keinen Verlauf und ist nicht gemerkt; der Eintrag von A ist verbraucht
		t.Errorf("%d gemerkt nach dem Zurück, Soll 0", len(l.e.verlaeufe.liste))
	}
	n := len(docA.Objekte())
	if err := l.e.s.D.Rueckgaengig(); err != nil {
		t.Fatal(err)
	}
	if len(docA.Objekte()) != n-1 || !l.e.s.Geaendert() {
		t.Errorf("Rückgängig nach dem Zurück: %d Objekte (vorher %d), geändert %v", len(docA.Objekte()), n, l.e.s.Geaendert())
	}
}

func TestWechselZurueckNachFremderAenderungLaedtNeu(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	f := &fortsetzerLauf{}
	f.anbinden(l)
	l.e.s.Gespeichert()
	docA := l.e.s.D
	if _, err := docA.Hinzufuegen(wzKasten()); err != nil {
		t.Fatal(err)
	}
	l.ruhig()
	l.klick(l.kachel(0))
	l.ruhig()
	// Die Datei von A ändert sich, während A verlassen ist (anderes Programm, Schlagworte in der Ansicht).
	spaeter := time.Now().Add(time.Hour)
	if err := os.Chtimes(l.pfade[1], spaeter, spaeter); err != nil {
		t.Fatal(err)
	}
	l.klick(l.kachel(1))
	l.ruhig()
	if f.zahl() != 1 {
		t.Fatalf("Fortsetzen %d-mal gerufen, Soll 1", f.zahl())
	}
	if l.e.s.D == docA || l.e.s.D.KannRueckgaengig() {
		t.Error("nach fremder Änderung das gemerkte Dokument genommen")
	}
}

// Punkt 3: Mit einer Schwärzung im Verlauf, ohne Verlauf, als Datei von außen oder ohne Fortsetzen wird nicht gemerkt.
func TestMerkeVerlasseneNurWennSicher(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 2)
	f := &fortsetzerLauf{}
	f.anbinden(l)
	if err := l.e.s.D.VergissVerlauf(); err != nil { // die Vorrichtung legt einen Text an
		t.Fatal(err)
	}
	l.e.s.Gespeichert()
	l.e.merkeVerlassene()
	if len(l.e.verlaeufe.liste) != 0 {
		t.Fatal("ohne Verlauf gemerkt")
	}
	if _, err := l.e.s.D.Hinzufuegen(wzKasten()); err != nil {
		t.Fatal(err)
	}
	l.e.merkeVerlassene()
	if len(l.e.verlaeufe.liste) != 0 {
		t.Fatal("ungespeichert gemerkt")
	}
	l.e.s.Gespeichert()
	l.e.aussen = true
	l.e.merkeVerlassene()
	if len(l.e.verlaeufe.liste) != 0 {
		t.Fatal("Datei von außen gemerkt")
	}
	l.e.aussen = false
	fort := l.e.bib.Fortsetzen
	l.e.bib.Fortsetzen = nil
	l.e.merkeVerlassene()
	if len(l.e.verlaeufe.liste) != 0 {
		t.Fatal("ohne Fortsetzen gemerkt")
	}
	l.e.bib.Fortsetzen = fort
	l.e.merkeVerlassene()
	if len(l.e.verlaeufe.liste) != 1 {
		t.Fatal("gespeichert mit Verlauf nicht gemerkt")
	}
	l.e.verlaeufe = verlaufMerker{}
	if err := l.e.s.D.Schwaerzen(image.Rect(0, 0, 10, 10), document.Schwaerzung{Farbe: canvas.Schwaerzfarbe}); err != nil {
		t.Fatal(err)
	}
	l.e.s.Gespeichert()
	l.e.merkeVerlassene()
	if len(l.e.verlaeufe.liste) != 0 {
		t.Error("mit Schwärzung im Verlauf gemerkt")
	}
}

func TestVerlaufMerkerGrenzen(t *testing.T) {
	var m verlaufMerker
	for i := range merkerAnzahl + 2 {
		m.merke(gemerkteSitzung{pfad: string(rune('a' + i)), bedarf: 1})
	}
	if len(m.liste) != merkerAnzahl || m.liste[0].pfad != "c" {
		t.Fatalf("Anzahl: %d Einträge, ältester %q", len(m.liste), m.liste[0].pfad)
	}
	m.merke(gemerkteSitzung{pfad: "c", bedarf: 1}) // erneut: wandert ans Ende, kein Doppel
	if len(m.liste) != merkerAnzahl || m.liste[len(m.liste)-1].pfad != "c" || m.liste[0].pfad != "d" {
		t.Fatalf("erneut gemerkt: %+v", m.liste)
	}
	m.merke(gemerkteSitzung{pfad: "gross", bedarf: merkerGrenze + 1})
	if len(m.liste) != merkerAnzahl {
		t.Errorf("ein Dokument über der Grenze verdrängte die anderen: %d Einträge", len(m.liste))
	}
	if _, ok := m.nimm("gross"); ok {
		t.Error("Dokument über der Grenze gemerkt")
	}
	m.merke(gemerkteSitzung{pfad: "halb", bedarf: merkerGrenze - 2})
	summe := int64(0)
	for _, e := range m.liste {
		summe += e.bedarf
	}
	if summe > merkerGrenze || m.liste[len(m.liste)-1].pfad != "halb" {
		t.Errorf("Speichergrenze: Summe %d, Liste %+v", summe, m.liste)
	}
	if _, ok := m.nimm("halb"); !ok {
		t.Fatal("nimm fand den Eintrag nicht")
	}
	if _, ok := m.nimm("halb"); ok {
		t.Error("nimm entfernte den Eintrag nicht")
	}
}

func TestPapierkorbVergisstGemerkte(t *testing.T) {
	var m verlaufMerker
	m.merke(gemerkteSitzung{pfad: "/a/b/../c.png", bedarf: 1})
	m.vergiss("/a/c.png")
	if len(m.liste) != 0 {
		t.Error("merke bereinigt den Pfad nicht")
	}
	m.merke(gemerkteSitzung{pfad: "/a/c.png", bedarf: 1})
	m.vergiss("/a/./c.png")
	if len(m.liste) != 0 {
		t.Error("vergiss mit anderer Schreibweise desselben Pfads wirkungslos")
	}
}

// Punkt 3: Geht die offene mit dem Wechsel in den Papierkorb (✕ an der offenen, Karte #347), bleibt ihr Dokument nicht
// gemerkt.
func TestKreuzAnDerOffenenMerktNicht(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 4)
	f := &fortsetzerLauf{}
	f.anbinden(l)
	k := &l.e.unten
	l.e.s.Gespeichert() // die Vorrichtung hat einen Verlauf (Text): gespeichert wäre sie merkbar
	l.maus(pointer.Move, l.kachel(1), 0)
	l.setle()
	lr := k.loeschRahmen[1]
	l.maus(pointer.Move, mittenPunkt(lr), 0)
	l.setle()
	l.klick(mittenPunkt(lr))
	l.warte(func() bool {
		return !l.e.wechsel.laedt && !l.e.bibl.arbeitLaeuft && !k.indexLaeuft && len(k.l.Eintraege()) == 3
	})
	if l.e.pfad() != l.pfade[2] {
		t.Fatalf("offen ist %s, Soll die Nachbarin", l.e.pfad())
	}
	if len(l.e.verlaeufe.liste) != 0 {
		t.Errorf("die weggeworfene ist gemerkt: %+v", l.e.verlaeufe.liste)
	}
}

// F-834: Trägt die Datei beim Verlassen einen anderen Stand als den der eigenen Speicherfunktion (ein anderes Fenster
// oder `--annotate` hat sie geändert, etwa geschwärzt), wird nicht gemerkt: der Rückwechsel liest neu.
func TestMerkeVerlasseneNichtBeiFremdemStand(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 2)
	f := &fortsetzerLauf{}
	f.anbinden(l)
	l.e.s.Gespeichert() // gespeichert, mit Verlauf (die Vorrichtung legt einen Text an)
	spaeter := time.Now().Add(time.Hour)
	if err := os.Chtimes(l.e.pfad(), spaeter, spaeter); err != nil {
		t.Fatal(err)
	}
	l.e.merkeVerlassene()
	if len(l.e.verlaeufe.liste) != 0 {
		t.Error("gemerkt, obwohl die Datei einen fremden Stand trägt")
	}
	l.e.stand = nil
	if err := os.Chtimes(l.e.pfad(), time.Now(), time.Now()); err != nil {
		t.Fatal(err)
	}
	l.e.merkeVerlassene()
	if len(l.e.verlaeufe.liste) != 0 {
		t.Error("ohne eigenen Stand gemerkt")
	}
}

// F-835 (K-2): Schwärzen und Rückgängig ohne Speichern ist ungeändert, hält die Schwärzung aber im Verlauf
// (zurückgenommen, wiederholbar) — nicht merken.
func TestMerkeVerlasseneNichtMitZurueckgenommenerSchwaerzung(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 2)
	f := &fortsetzerLauf{}
	f.anbinden(l)
	l.e.s.Gespeichert()
	if err := l.e.s.D.Schwaerzen(image.Rect(0, 0, 10, 10), document.Schwaerzung{Farbe: canvas.Schwaerzfarbe}); err != nil {
		t.Fatal(err)
	}
	if err := l.e.s.D.Rueckgaengig(); err != nil {
		t.Fatal(err)
	}
	if l.e.s.Geaendert() || !l.e.s.D.VerlaufMitSchwaerzung() {
		t.Fatalf("Vorbedingung: geändert %v, Schwärzung im Verlauf %v", l.e.s.Geaendert(), l.e.s.D.VerlaufMitSchwaerzung())
	}
	l.e.merkeVerlassene()
	if len(l.e.verlaeufe.liste) != 0 {
		t.Error("mit zurückgenommener Schwärzung im Verlauf gemerkt")
	}
}

// F-838: Scheitert das Laden beim Rückwechsel, bleibt der Eintrag gemerkt.
func TestWechselZurueckLadenScheitertBehaeltEintrag(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	f := &fortsetzerLauf{}
	f.anbinden(l)
	l.e.s.Gespeichert()
	docA := l.e.s.D
	l.klick(l.kachel(0))
	l.ruhig()
	if len(l.e.verlaeufe.liste) != 1 {
		t.Fatalf("%d gemerkt, Soll 1", len(l.e.verlaeufe.liste))
	}
	l.mu.Lock()
	l.fehlerFuer = map[string]error{l.pfade[1]: errors.New("kaputt")}
	l.mu.Unlock()
	l.klick(l.kachel(1))
	l.ruhig()
	if l.e.s.D == docA {
		t.Fatal("trotz Ladefehler gewechselt")
	}
	if len(l.e.verlaeufe.liste) != 1 || l.e.verlaeufe.liste[0].d != docA {
		t.Errorf("nach gescheitertem Laden: %d gemerkt", len(l.e.verlaeufe.liste))
	}
}

// F-838: Eine nicht offene, gemerkte Aufnahme, die in den Papierkorb geht, wird vergessen.
func TestPapierkorbNichtOffeneVergisst(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	f := &fortsetzerLauf{}
	f.anbinden(l)
	l.e.s.Gespeichert()
	l.klick(l.kachel(0))
	l.ruhig()
	if len(l.e.verlaeufe.liste) != 1 {
		t.Fatalf("%d gemerkt, Soll 1", len(l.e.verlaeufe.liste))
	}
	rel, ok := l.b.Relativ(l.pfade[1])
	if !ok {
		t.Fatal("Pfad nicht in der Ablage")
	}
	l.e.papierkorbAnfrage(rel, false, nil)
	if len(l.e.verlaeufe.liste) != 0 {
		t.Error("die weggeworfene, nicht offene Aufnahme ist noch gemerkt")
	}
	l.ruhig()
}
