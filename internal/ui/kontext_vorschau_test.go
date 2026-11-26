// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"context"
	"errors"
	"image"
	"os"
	"slices"
	"sync"
	"testing"

	"gioui.org/io/key"

	"github.com/Andiministrator/schnapp/internal/portal"
)

// aktionen sind die Handlungen der offenen Liste in ihrer Reihenfolge.
func (l *menueLauf) aktionen() []aktion {
	var a []aktion
	for _, en := range l.e.menueEintraege() {
		a = append(a, en.akt)
	}
	return a
}

// eintragKlick klickt den Eintrag mit der Handlung a der offenen Liste.
func (l *menueLauf) eintragKlick(a aktion) {
	l.t.Helper()
	i := l.eintragMit(a)
	if i < 0 {
		l.t.Fatalf("Eintrag %d fehlt in %v", a, l.aktionen())
	}
	l.klick(mitte(l.e.menue.rahmen[i]))
}

// dateimanagerAttrappe ersetzt den Busaufruf und zählt mit.
type dateimanagerAttrappe struct {
	mu     sync.Mutex
	pfade  []string
	fehler error
}

func (d *dateimanagerAttrappe) gerufen() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return slices.Clone(d.pfade)
}

func mitDateimanager(t *testing.T, fehler error) *dateimanagerAttrappe {
	d := &dateimanagerAttrappe{fehler: fehler}
	alt := dateimanagerZeigen
	dateimanagerZeigen = func(_ context.Context, pfad string) error {
		d.mu.Lock()
		defer d.mu.Unlock()
		d.pfade = append(d.pfade, pfad)
		return d.fehler
	}
	t.Cleanup(func() { dateimanagerZeigen = alt })
	return d
}

// ADR-0038 Punkt 3: Ein Rechtsklick auf eine Vorschau der Leiste öffnet am Klickpunkt das Kontextmenü dieser Vorschau
// (Öffnen, Im Dateimanager zeigen, Papierkorb; keine Schlagworte) und gibt ihr den Fokus; Öffnen wechselt zu ihr.
func TestKontextVorschauLeisteOeffnet(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	p := l.kachel(0).Add(image.Pt(7, 5))
	l.rechtsklick(p)
	m := &l.e.menue
	if !m.offen || m.welches != listeKontext || l.e.kontext.vorschau != l.rel(0) || !l.e.kontext.inLeiste {
		t.Fatalf("offen %v, Liste %d, Vorschau %q (Soll %q), Leiste %v", m.offen, m.welches, l.e.kontext.vorschau, l.rel(0),
			l.e.kontext.inLeiste)
	}
	if l.e.kontext.anker.Min != p {
		t.Errorf("Anker %v, Soll %v", l.e.kontext.anker.Min, p)
	}
	if a := l.aktionen(); !slices.Equal(a, []aktion{aktVorschauOeffnen, aktVorschauZeigen, aktVorschauPapierkorb}) {
		t.Errorf("Einträge %v", a)
	}
	for _, en := range l.e.menueEintraege() {
		if !en.an {
			t.Errorf("Eintrag %q grau", en.text)
		}
	}
	if !l.fokus(&l.e.unten.knoepfe[0]) {
		t.Error("die Vorschau hat keinen Fokus")
	}
	if len(l.aufrufe()) != 0 {
		t.Fatal("der Rechtsklick hat geöffnet")
	}
	l.eintragKlick(aktVorschauOeffnen)
	l.ruhig()
	if a := l.aufrufe(); !slices.Equal(a, []string{l.pfade[0]}) {
		t.Errorf("geöffnet %q, Soll %q", a, l.pfade[0])
	}
}

// An der offenen Aufnahme der Leiste ist Öffnen grau mit Erklärung; Esc gibt den Fokus an die Vorschau zurück.
func TestKontextVorschauLeisteOffeneAufnahme(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	l.rechtsklick(l.kachel(1)) // pfade[1] ist offen
	i := l.eintragMit(aktVorschauOeffnen)
	if i < 0 {
		t.Fatal("Öffnen fehlt")
	}
	if en := l.e.menueEintraege()[i]; en.an || en.erklaerung == "" {
		t.Errorf("Öffnen an %v, Erklärung %q", en.an, en.erklaerung)
	}
	if j := l.eintragMit(aktVorschauPapierkorb); j < 0 || !l.e.menueEintraege()[j].an {
		t.Error("Papierkorb an der offenen mit Nachbarinnen grau")
	}
	l.taste(key.NameEscape, 0)
	l.setle()
	if l.e.menue.offen || !l.fokus(&l.e.unten.knoepfe[1]) {
		t.Errorf("offen %v, Fokus an der Vorschau %v", l.e.menue.offen, l.fokus(&l.e.unten.knoepfe[1]))
	}
}

// „Im Dateimanager zeigen“ ruft den Dateimanager mit dem absoluten Pfad der Vorschau; fehlt der Dienst, sagt es die
// Fußzeile.
func TestKontextVorschauDateimanager(t *testing.T) {
	d := mitDateimanager(t, nil)
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	l.rechtsklick(l.kachel(2))
	l.eintragKlick(aktVorschauZeigen)
	l.warte(func() bool { return l.e.dateimanagerAbholen() })
	if g := d.gerufen(); !slices.Equal(g, []string{l.pfade[2]}) {
		t.Errorf("gerufen %q, Soll %q", g, l.pfade[2])
	}
	if l.e.dateimanager.laeuft || l.e.meldung.Text != "" && l.e.meldung.Fehler {
		t.Errorf("läuft %v, Meldung %q", l.e.dateimanager.laeuft, l.e.meldung.Text)
	}

	d.fehler = errors.Join(portal.ErrKeinDateimanager)
	l.rechtsklick(l.kachel(2))
	l.eintragKlick(aktVorschauZeigen)
	l.warte(func() bool { return l.e.dateimanagerAbholen() })
	if !l.e.meldung.Fehler || l.e.meldung.Text != "No file manager available to show the file (org.freedesktop.FileManager1 missing)." {
		t.Errorf("Meldung %q (Fehler %v)", l.e.meldung.Text, l.e.meldung.Fehler)
	}

	d.fehler = errors.New("anders")
	l.rechtsklick(l.kachel(2))
	l.eintragKlick(aktVorschauZeigen)
	l.warte(func() bool { return l.e.dateimanagerAbholen() })
	if !l.e.meldung.Fehler || l.e.meldung.Text != "Could not show the file in the file manager." {
		t.Errorf("Meldung %q (Fehler %v)", l.e.meldung.Text, l.e.meldung.Fehler)
	}
}

// Während ein Aufruf des Dateimanagers läuft, ist der Eintrag grau, und ein zweiter Aufruf tut nichts.
func TestKontextVorschauDateimanagerEinmal(t *testing.T) {
	tor := make(chan struct{})
	var n int
	var mu sync.Mutex
	alt := dateimanagerZeigen
	dateimanagerZeigen = func(context.Context, string) error {
		mu.Lock()
		n++
		mu.Unlock()
		<-tor
		return nil
	}
	t.Cleanup(func() { dateimanagerZeigen = alt })
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	l.rechtsklick(l.kachel(2))
	l.eintragKlick(aktVorschauZeigen)
	l.rechtsklick(l.kachel(0))
	if i := l.eintragMit(aktVorschauZeigen); i < 0 || l.e.menueEintraege()[i].an {
		t.Error("Eintrag während des Aufrufs nicht grau")
	}
	l.e.dateimanagerStarten(l.pfade[0])
	close(tor)
	l.warte(func() bool { return l.e.dateimanagerAbholen() })
	mu.Lock()
	defer mu.Unlock()
	if n != 1 {
		t.Errorf("%d Aufrufe, Soll 1", n)
	}
}

// „In den Papierkorb“ aus dem Kontextmenü der Leiste legt die Vorschau dieses Menüs weg, auch wenn der Fokus inzwischen
// woanders steht.
func TestKontextVorschauLeistePapierkorb(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	l.rechtsklick(l.kachel(2))
	// Fokus woanders und ohne Zeichnen dazwischen auslösen: ein Bild holte den Fokus der Liste wieder auf die Vorschau mit
	// Tastaturfokus (Mutant KV10 überlebte den Weg über den Klick).
	l.e.unten.l.SetzeFokus(0)
	l.e.vorschauWirkung(aktVorschauPapierkorb)(l.gtx())
	l.ruhig()
	if _, err := os.Stat(l.pfade[2]); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Stelle 2 liegt noch in der Ablage (%v)", err)
	}
	if _, err := os.Stat(l.pfade[0]); err != nil {
		t.Errorf("Stelle 0 fehlt: %v", err)
	}
	if n := papierkorbDateien(t, l); n != 1 {
		t.Errorf("Papierkorb hat %d Dateien", n)
	}
}

// Ist die Vorschau beim Auslösen nicht mehr in der Liste (Index neu gelesen), tut der Eintrag nichts.
func TestKontextVorschauWeg(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	l.rechtsklick(l.kachel(2))
	l.e.kontext.vorschau = "2000-01/gibt-es-nicht.png"
	for _, en := range l.e.menueEintraege() {
		if en.an {
			t.Errorf("Eintrag %q an, obwohl die Vorschau fehlt", en.text)
		}
	}
	l.e.vorschauWirkung(aktVorschauPapierkorb)(l.gtx())
	l.e.vorschauWirkung(aktVorschauOeffnen)(l.gtx())
	l.ruhig()
	if n := papierkorbDateien(t, l); n != 0 || len(l.aufrufe()) != 0 {
		t.Errorf("Papierkorb %d, geöffnet %q", n, l.aufrufe())
	}
}

// Ein zweiter Rechtsklick auf eine andere Vorschau öffnet das Menü dort für sie.
func TestKontextVorschauZweiterRechtsklick(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	l.rechtsklick(l.kachel(0))
	p := l.kachel(2)
	l.rechtsklick(p)
	if !l.e.menue.offen || l.e.kontext.vorschau != l.rel(2) || l.e.kontext.anker.Min != p {
		t.Errorf("offen %v, Vorschau %q (Soll %q), Anker %v", l.e.menue.offen, l.e.kontext.vorschau, l.rel(2),
			l.e.kontext.anker.Min)
	}
	// und von der Vorschau zurück auf die Leinwand: das Menü des Bilds
	q := l.e.leinwandRahmen.Max.Sub(image.Pt(5, 5))
	l.rechtsklick(q)
	if !l.e.menue.offen || l.e.kontext.vorschau != "" || l.e.kontext.anker.Min != q {
		t.Errorf("Leinwand: offen %v, Vorschau %q, Anker %v", l.e.menue.offen, l.e.kontext.vorschau, l.e.kontext.anker.Min)
	}
}

// ADR-0038 Punkt 6: Umschalt+F10 an einer Vorschau der Leiste öffnet ihr Kontextmenü in der Mitte der Kachel.
func TestKontextVorschauLeisteUmschaltF10(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	l.fokusiere(&l.e.unten.knoepfe[2])
	l.taste(key.NameF10, key.ModShift)
	if !l.e.menue.offen || l.e.kontext.vorschau != l.rel(2) || l.e.kontext.anker.Min != l.kachel(2) {
		t.Fatalf("offen %v, Vorschau %q, Anker %v (Soll %v)", l.e.menue.offen, l.e.kontext.vorschau, l.e.kontext.anker.Min,
			l.kachel(2))
	}
	if l.e.menue.cursor != 0 {
		t.Errorf("Tastaturfokus auf Eintrag %d, Soll 0", l.e.menue.cursor)
	}
}

// Volle Ansicht: Rechtsklick setzt den Fokus auf die Vorschau und zeigt auch „Schlagworte“; der Eintrag beginnt die
// Bearbeitung für diese Vorschau.
func TestKontextVorschauAnsichtSchlagworte(t *testing.T) {
	l := bibLauf(t, 3)
	p := l.bibKachel(2)
	l.rechtsklick(p)
	b := &l.e.bibl
	if !l.e.menue.offen || l.e.kontext.vorschau != l.rel(2) || l.e.kontext.inLeiste || b.r.Fokus() != 2 {
		t.Fatalf("offen %v, Vorschau %q, Leiste %v, Fokus %d", l.e.menue.offen, l.e.kontext.vorschau, l.e.kontext.inLeiste,
			b.r.Fokus())
	}
	if a := l.aktionen(); !slices.Equal(a, []aktion{aktVorschauOeffnen, aktVorschauZeigen, aktVorschauSchlagworte,
		aktVorschauPapierkorb}) {
		t.Errorf("Einträge %v", a)
	}
	b.r.SetzeFokus(0)
	l.eintragKlick(aktVorschauSchlagworte)
	if b.bearbeite != l.rel(2) {
		t.Errorf("bearbeitet %q, Soll %q", b.bearbeite, l.rel(2))
	}
}

// Volle Ansicht, offene Aufnahme: Schlagworte grau mit Erklärung, Öffnen führt zurück zum Bild.
func TestKontextVorschauAnsichtOffene(t *testing.T) {
	l := bibLauf(t, 3)
	l.rechtsklick(l.bibKachel(1))
	i := l.eintragMit(aktVorschauSchlagworte)
	if en := l.e.menueEintraege()[i]; en.an || en.erklaerung == "" {
		t.Errorf("Schlagworte an %v, Erklärung %q", en.an, en.erklaerung)
	}
	l.eintragKlick(aktVorschauOeffnen)
	if l.e.bibl.offen || len(l.aufrufe()) != 0 {
		t.Errorf("Ansicht offen %v, geöffnet %q", l.e.bibl.offen, l.aufrufe())
	}
}

// Volle Ansicht: „In den Papierkorb“ wirkt auf die Vorschau des Menüs.
func TestKontextVorschauAnsichtPapierkorb(t *testing.T) {
	l := bibLauf(t, 3)
	l.rechtsklick(l.bibKachel(2))
	l.e.bibl.r.SetzeFokus(0)
	l.eintragKlick(aktVorschauPapierkorb)
	l.ruhig()
	if _, err := os.Stat(l.pfade[2]); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Stelle 2 liegt noch in der Ablage (%v)", err)
	}
	if _, err := os.Stat(l.pfade[0]); err != nil {
		t.Errorf("Stelle 0 fehlt: %v", err)
	}
}

// ADR-0038 Punkt 6 (löst Kritik K-1 #313a ab): In der vollen Ansicht öffnet Umschalt+F10 das Kontextmenü der Vorschau
// mit Fokus, F10 allein weiter das Menü Datei; ohne Fokus im Raster öffnet Umschalt+F10 nichts.
func TestKontextmenueBibliothekUmschaltF10(t *testing.T) {
	l := bibLauf(t, 3)
	b := &l.e.bibl
	b.r.SetzeFokus(2)
	l.setle()
	l.taste(key.NameF10, key.ModShift)
	if !l.e.menue.offen || l.e.menue.welches != listeKontext || l.e.kontext.vorschau != l.rel(2) ||
		l.e.kontext.anker.Min != mitte(b.kacheln[2-b.von]) {
		t.Fatalf("offen %v, Liste %d, Vorschau %q, Anker %v", l.e.menue.offen, l.e.menue.welches, l.e.kontext.vorschau,
			l.e.kontext.anker.Min)
	}
	l.taste(key.NameEscape, 0)
	if l.e.menue.offen || !l.e.bibl.offen {
		t.Fatalf("nach Esc: Menü %v, Ansicht %v", l.e.menue.offen, l.e.bibl.offen)
	}
	b.ziel = bibZielSuche
	l.taste(key.NameF10, key.ModShift)
	if l.e.menue.offen {
		t.Errorf("Umschalt+F10 im Suchfeld öffnet Menü %d", l.e.menue.welches)
	}
	l.taste(key.NameF10, 0)
	if !l.e.menue.offen || l.e.menue.welches != menueDatei {
		t.Errorf("F10 in der Bibliothek: offen %v, Menü %d", l.e.menue.offen, l.e.menue.welches)
	}
}

// Ein Rechtsklick auf die leere Fläche der Ansicht öffnet nichts.
func TestKontextVorschauAnsichtLeer(t *testing.T) {
	l := bibLauf(t, 3)
	l.rechtsklick(l.e.bibl.sicht.Max.Sub(image.Pt(3, 3)))
	if l.e.menue.offen {
		t.Errorf("Menü %d offen", l.e.menue.welches)
	}
}
