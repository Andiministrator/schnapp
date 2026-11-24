// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"errors"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/libview"
)

// leistenRettung ist eine Rettung, die Wiederherstellen und Ende zählt (nur aus der Ereignisschleife gerufen).
type leistenRettung struct{ wieder, ende int }

func (r *leistenRettung) Wiederherstellen(*document.Dokument) (string, bool) { r.wieder++; return "", false }
func (r *leistenRettung) Sichere(*document.Dokument)                         {}
func (r *leistenRettung) Verwirf()                                           {}
func (r *leistenRettung) Ende()                                              { r.ende++ }
func (r *leistenRettung) Lage() (string, bool, uint64)                       { return "", false, 0 }

// leistenLauf ist ein Editor mit Bibliothek in einem Testordner und einer Ladefunktion, die mitschreibt.
type leistenLauf struct {
	*menueLauf
	b      *library.Bibliothek
	pfade  []string // absolute Pfade der Aufnahmen, neueste zuerst (ohne Agenten-Aufnahme)
	mu     sync.Mutex
	offen  []string // Aufrufe der Ladefunktion
	fehler error    // Ladefunktion scheitert
	tor    chan struct{}
	neu    map[string]*document.Dokument // geladene Dokumente je Pfad
	rett   map[string]*leistenRettung
	alt    *leistenRettung // Rettung des ersten Dokuments
}

// legeAufnahmeAb legt eine Aufnahme 40 × 30 in der Farbe c mit Zeit zeit ab und liefert den absoluten Pfad.
func legeAufnahmeAb(t *testing.T, b *library.Bibliothek, c color.NRGBA, zeit time.Time, agent bool) string {
	t.Helper()
	bild := image.NewNRGBA(image.Rect(0, 0, 40, 30))
	for i := 0; i < len(bild.Pix); i += 4 {
		bild.Pix[i], bild.Pix[i+1], bild.Pix[i+2], bild.Pix[i+3] = c.R, c.G, c.B, 255
	}
	d, err := document.Neu(bild)
	if err != nil {
		t.Fatal(err)
	}
	ablegen := b.Ablegen
	if agent {
		ablegen = b.AblegenAgent
	}
	e, err := ablegen(d, d.Original(), fileformat.Metadaten{Erstellt: zeit})
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
}

func neuerLeistenLauf(t *testing.T, fenster image.Point, anzahl int) *leistenLauf {
	t.Helper()
	ml := neuerMenueLauf(t, fenster)
	heim := t.TempDir()
	b := library.Neu(library.Orte{Ablage: filepath.Join(heim, "Bilder", "schnapp"), Index: filepath.Join(heim, "daten", "bibliothek.jsonl"),
		Vorschau: filepath.Join(heim, "cache", "vorschau"), Papierkorb: filepath.Join(heim, "daten", "Trash")})
	l := &leistenLauf{menueLauf: ml, b: b, neu: map[string]*document.Dokument{}, rett: map[string]*leistenRettung{}}
	t0 := time.Date(2026, 11, 24, 8, 0, 0, 0, time.UTC)
	farben := []color.NRGBA{{R: 200}, {G: 200}, {B: 200}}
	for i := range anzahl {
		l.pfade = append([]string{legeAufnahmeAb(t, b, farben[i%3], t0.Add(time.Duration(i)*time.Hour), false)}, l.pfade...)
	}
	legeAufnahmeAb(t, b, color.NRGBA{R: 90}, t0.Add(100*time.Hour), true) // Agenten-Aufnahme, die neueste
	alteLade := ladeLeiste
	ladeLeiste = func() bool { return true }
	t.Cleanup(func() { ladeLeiste = alteLade })
	l.alt = &leistenRettung{}
	l.e.rettung = l.alt
	offenPfad := ""
	if len(l.pfade) > 1 {
		offenPfad = l.pfade[1]
	}
	l.e.pfad = func() string { return offenPfad }
	l.e.bib = &Anbindung{Bibliothek: b, Oeffnen: l.oeffnen}
	l.e.leisteStart()
	l.warte(func() bool { return !l.e.unten.indexLaeuft && len(l.e.unten.l.Eintraege()) == anzahl })
	return l
}

// oeffnen ist die Ladefunktion des Tests: ein neues Dokument je Pfad, mit eigener Rettung.
func (l *leistenLauf) oeffnen(pfad string) (Geoeffnet, error) {
	l.mu.Lock()
	l.offen = append(l.offen, pfad)
	tor, fehler := l.tor, l.fehler
	l.mu.Unlock()
	if tor != nil {
		<-tor
	}
	if fehler != nil {
		return Geoeffnet{}, fehler
	}
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 60, 20)))
	if err != nil {
		return Geoeffnet{}, err
	}
	r := &leistenRettung{}
	l.mu.Lock()
	l.neu[pfad], l.rett[pfad] = d, r
	l.mu.Unlock()
	return Geoeffnet{Titel: filepath.Base(pfad), Dokument: d, Rettung: r, Pfad: func() string { return pfad },
		Speichern: func(*document.Dokument) (Gespeichert, error) { return Gespeichert{}, nil }}, nil
}

func (l *leistenLauf) aufrufe() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.offen...)
}

// warte zeichnet Bilder, bis f gilt (Antworten aus Goroutinen kommen über die Ereignisschleife), höchstens 5 s.
func (l *leistenLauf) warte(f func() bool) {
	l.t.Helper()
	ende := time.Now().Add(5 * time.Second)
	for !f() {
		if time.Now().After(ende) {
			l.t.Fatal("Zeit abgelaufen")
		}
		l.bild()
		time.Sleep(time.Millisecond)
	}
	l.setle()
}

// ruhig wartet, bis weder Speichern noch Wechsel noch Lesen des Index laufen.
func (l *leistenLauf) ruhig() {
	l.t.Helper()
	l.warte(func() bool { return !l.e.s.SpeichernLaeuft() && !l.e.wechsel.laedt && !l.e.unten.indexLaeuft })
}

// kachel ist die Mitte der gezeichneten Vorschau an der Stelle i.
func (l *leistenLauf) kachel(i int) image.Point {
	l.t.Helper()
	k := &l.e.unten
	if i < k.von || i >= k.bis {
		l.t.Fatalf("Stelle %d nicht gezeichnet (%d bis %d)", i, k.von, k.bis)
	}
	r := k.kacheln[i-k.von]
	return r.Min.Add(r.Size().Div(2))
}

// ADR-0036 Punkte 1 und 2: Die Leiste liegt unter der Leinwand und nimmt ihr 104 dp; unter 420 dp verbleibender
// Leinwandhöhe, ausgeschaltet oder ohne Bibliothek fehlt sie.
func TestLeisteFlaeche(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	if f := l.e.leinwandFlaeche(l.gtx()); f.Max.Y != 800-fusszeileDp-leisteUntenDp {
		t.Errorf("Leinwand mit Leiste endet bei %d", f.Max.Y)
	}
	if r := l.e.unten.rahmen; r != image.Rect(werkzeugeDp, 800-fusszeileDp-leisteUntenDp, 1200-seiteDp, 800-fusszeileDp) {
		t.Errorf("Lage der Leiste %v", r)
	}
	// Genau an der Schwelle noch da, ein Pixel darunter nicht.
	schwelle := menuezeileDp + fusszeileDp + leisteUntenDp + leisteMinLeinwandDp
	l.fenst = image.Pt(1200, schwelle)
	l.setle()
	if l.e.unten.rahmen.Empty() || l.e.leinwandFlaeche(l.gtx()).Dy() != leisteMinLeinwandDp {
		t.Errorf("an der Schwelle: Leiste %v, Leinwand %v", l.e.unten.rahmen, l.e.leinwandFlaeche(l.gtx()))
	}
	l.fenst = image.Pt(1200, schwelle-1)
	l.setle()
	if !l.e.unten.rahmen.Empty() || l.e.leinwandFlaeche(l.gtx()).Max.Y != schwelle-1-fusszeileDp {
		t.Errorf("unter der Schwelle: Leiste %v", l.e.unten.rahmen)
	}
	if !l.e.unten.an {
		t.Error("das Ausblenden ändert die Einstellung nicht")
	}
	l.fenst = image.Pt(1200, 800)
	l.e.unten.an = false
	l.setle()
	if !l.e.unten.rahmen.Empty() || l.e.leinwandFlaeche(l.gtx()).Max.Y != 800-fusszeileDp {
		t.Error("ausgeschaltet: Leiste noch da")
	}
	ohne := neuerMenueLauf(t, image.Pt(1200, 800))
	ohne.e.unten.an = true
	ohne.setle()
	if !ohne.e.unten.rahmen.Empty() || ohne.e.leinwandFlaeche(ohne.gtx()).Max.Y != 800-fusszeileDp {
		t.Error("ohne Bibliothek: Leiste da")
	}
}

// ADR-0036 Punkt 1: neueste links, ohne Agenten-Aufnahme, die offene grün umrandet; Vorschaubilder kommen aus dem Cache.
func TestLeisteZeigtBibliothek(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	es := l.e.unten.l.Eintraege()
	for i, e := range es {
		if filepath.Join(l.b.Orte.Ablage, filepath.FromSlash(e.Pfad)) != l.pfade[i] {
			t.Errorf("Stelle %d: %s", i, e.Pfad)
		}
	}
	if !l.e.unten.l.IstAktuell(1) || l.e.unten.l.IstAktuell(0) {
		t.Error("offene Aufnahme nicht erkannt")
	}
	if n := len(l.e.unten.kacheln); n != 3 {
		t.Fatalf("%d Vorschauen gezeichnet", n)
	}
	l.warte(func() bool {
		_, z := l.e.unten.v.Bild(es[2].Pfad)
		return z == libview.Da
	})
	if len(l.e.unten.ops) != 3 {
		t.Errorf("Bild-Ops: %d", len(l.e.unten.ops))
	}
	// Hinweis zur Vorschau unter dem Zeiger.
	p := l.kachel(1)
	l.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(float32(p.X), float32(p.Y))})
	l.setle()
	if !strings.Contains(l.e.hinweis(), "open now") {
		t.Errorf("Hinweis zur offenen: %q", l.e.hinweis())
	}
	// Verschwindet die Leiste unter dem Zeiger, verschwindet ihr Hinweis mit (Sichtprobe 07).
	l.e.unten.an = false
	l.setle()
	if h := l.e.hinweis(); strings.Contains(h, "Capture of") || len(l.e.unten.gezeigt) != 0 {
		t.Errorf("Hinweis ohne Leiste: %q", h)
	}
}

// ADR-0036 Punkt 2: Strg+Umschalt+R schaltet die Leiste und merkt es in settings.ini; ohne Bibliothek ist der
// Menüeintrag grau mit Erklärung.
func TestLeisteSchalter(t *testing.T) {
	pfad := eigeneEinstellungen(t, "")
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	l.taste("R", key.ModShortcut|key.ModShift)
	if l.e.unten.an || !l.e.unten.rahmen.Empty() {
		t.Fatal("nicht ausgeschaltet")
	}
	if b, _ := os.ReadFile(pfad); !strings.Contains(string(b), "bar = no") {
		t.Errorf("nicht gemerkt: %q", b)
	}
	for _, en := range l.e.menueEintraegeVon(menueAnsicht) {
		if en.akt == aktLeiste && (en.aktiv || !en.an) {
			t.Errorf("Menüeintrag ausgeschaltet: aktiv %v, an %v", en.aktiv, en.an)
		}
	}
	l.taste("R", key.ModShortcut|key.ModShift)
	if !l.e.unten.an || l.e.unten.rahmen.Empty() {
		t.Fatal("nicht eingeschaltet")
	}
	if b, _ := os.ReadFile(pfad); !strings.Contains(string(b), "bar = yes") {
		t.Errorf("nicht gemerkt: %q", b)
	}
	ohne := testEditor(t)
	for _, en := range ohne.menueEintraegeVon(menueAnsicht) {
		if en.akt == aktLeiste && (en.an || en.aktiv || en.erklaerung == "") {
			t.Errorf("ohne Bibliothek: an %v, aktiv %v, Erklärung %q", en.an, en.aktiv, en.erklaerung)
		}
	}
	if aktionAusfuehrbar(aktLeiste) == false {
		t.Error("aktLeiste ohne Wirkung")
	}
}

// ADR-0034 Nachtrag 2: Speichert die Tafel [library] bar, folgt die Leiste sofort.
func TestLeisteFolgtTafel(t *testing.T) {
	eigeneEinstellungen(t, "")
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	l.taste(",", key.ModShortcut)
	tf := &l.e.einst
	for z, zeile := range tafelZeilen {
		if zeile.abschnitt == "library" && zeile.schluessel == "bar" {
			tf.schalte(z)
		}
	}
	l.e.einstellungenSpeichern(l.gtx())
	if l.e.unten.an {
		t.Error("die Leiste folgt der Tafel nicht")
	}
}

// ADR-0036 Punkt 5: Ein Klick auf eine andere Vorschau speichert zuerst und öffnet sie dann im selben Fenster — neue
// Sitzung, Titel, Pfad, Rettungskopie; die alte Rettung endet, --ignore-limit gilt nicht weiter, die Schalter der
// Ansicht bleiben.
func TestLeisteKlickWechselt(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	var gespeichert int
	var mu sync.Mutex
	l.e.speichern = func(*document.Dokument) (Gespeichert, error) {
		mu.Lock()
		gespeichert++
		mu.Unlock()
		return Gespeichert{}, nil
	}
	exportOhneGrenze.Store(true)
	l.e.ohneGrenze = true
	t.Cleanup(func() { exportOhneGrenze.Store(false) })
	l.e.s.SchalteEinrasten()
	l.e.s.SchalteNavigator()
	altD := l.e.s.D
	l.klick(l.kachel(0))
	l.ruhig()
	mu.Lock()
	if gespeichert != 1 {
		t.Errorf("vorher gespeichert: %d", gespeichert)
	}
	mu.Unlock()
	if a := l.aufrufe(); len(a) != 1 || a[0] != l.pfade[0] {
		t.Fatalf("geladen: %v", a)
	}
	if l.e.s.D == altD || l.e.s.D != l.neu[l.pfade[0]] {
		t.Fatal("Dokument nicht gewechselt")
	}
	if l.e.titel != filepath.Base(l.pfade[0]) || l.e.pfad() != l.pfade[0] {
		t.Errorf("Titel %q, Pfad %q", l.e.titel, l.e.pfad())
	}
	if l.alt.ende != 1 || l.rett[l.pfade[0]].wieder != 1 {
		t.Errorf("Rettung: alt Ende %d, neu Wiederherstellen %d", l.alt.ende, l.rett[l.pfade[0]].wieder)
	}
	if l.e.ohneGrenze || exportOhneGrenze.Load() {
		t.Error("--ignore-limit gilt nach dem Wechsel weiter")
	}
	if l.e.s.EinrastenAn() || l.e.s.NavigatorAn() {
		t.Error("Schalter der Ansicht nicht übernommen")
	}
	if l.e.s.Geaendert() || l.e.s.D.KannRueckgaengig() {
		t.Error("neue Sitzung nicht frisch")
	}
	if !l.e.unten.l.IstAktuell(0) || l.e.unten.l.IstAktuell(1) {
		t.Error("grüner Rand nicht gewandert")
	}
	if b := l.e.bildOp.Size(); b != image.Pt(60, 20) {
		t.Errorf("Leinwandbild %v", b)
	}
	// Ein Klick auf die jetzt offene Aufnahme tut nichts.
	l.klick(l.kachel(0))
	l.ruhig()
	if a := l.aufrufe(); len(a) != 1 {
		t.Errorf("Klick auf die offene lädt: %v", a)
	}
}

// Ohne Änderung wird nicht gespeichert, sondern gleich geladen.
func TestLeisteWechselOhneAenderung(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	l.e.s.Gespeichert()
	l.e.speichern = func(*document.Dokument) (Gespeichert, error) {
		t.Error("ohne Änderung gespeichert")
		return Gespeichert{}, nil
	}
	l.klick(l.kachel(2))
	l.ruhig()
	if a := l.aufrufe(); len(a) != 1 || a[0] != l.pfade[2] || l.e.s.D != l.neu[l.pfade[2]] {
		t.Errorf("geladen: %v", a)
	}
}

// Scheitert das Speichern, bleibt die offene Aufnahme, und die Statuszeile sagt es; ein Hinweis beim Speichern hält
// den Wechsel einmal an, der nächste Klick wechselt.
func TestLeisteWechselSpeichernScheitert(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	altD := l.e.s.D
	l.e.speichern = func(*document.Dokument) (Gespeichert, error) { return Gespeichert{}, errors.New("Platte voll") }
	l.klick(l.kachel(0))
	l.ruhig()
	if len(l.aufrufe()) != 0 || l.e.s.D != altD {
		t.Fatal("trotz Fehler gewechselt")
	}
	if m := l.e.meldung; !m.Fehler || !strings.Contains(m.Text, "Platte voll") || !strings.Contains(m.Text, "not opened") {
		t.Errorf("Meldung %+v", m)
	}
	l.e.speichern = func(*document.Dokument) (Gespeichert, error) { return Gespeichert{Hinweis: "als neue Aufnahme"}, nil }
	l.klick(l.kachel(0))
	l.ruhig()
	if len(l.aufrufe()) != 0 || l.e.s.D != altD || !strings.Contains(l.e.meldung.Text, "als neue Aufnahme") {
		t.Fatalf("Hinweis: geladen %v, Meldung %q", l.aufrufe(), l.e.meldung.Text)
	}
	l.klick(l.kachel(0))
	l.ruhig()
	if a := l.aufrufe(); len(a) != 1 || l.e.s.D == altD {
		t.Errorf("zweiter Klick nach dem Hinweis: %v", a)
	}
}

// Scheitert das Laden, bleibt die offene Aufnahme und die Statuszeile nennt den Grund.
func TestLeisteLadenScheitert(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	l.e.s.Gespeichert()
	altD := l.e.s.D
	l.fehler = errors.New("Das Bild ist beschädigt.")
	l.klick(l.kachel(2))
	l.ruhig()
	if l.e.s.D != altD || !l.e.meldung.Fehler || !strings.Contains(l.e.meldung.Text, "beschädigt") {
		t.Errorf("Meldung %+v", l.e.meldung)
	}
	if l.alt.ende != 0 {
		t.Error("die Rettung der offenen endet trotz Fehler")
	}
}

// Wird die offene Aufnahme geändert, während die andere lädt, bleibt sie offen; die geladene gibt ihre Rettung frei.
func TestLeisteGeaendertWaehrendLaden(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	l.e.s.Gespeichert()
	altD := l.e.s.D
	l.tor = make(chan struct{})
	l.klick(l.kachel(0))
	l.warte(func() bool { return len(l.aufrufe()) == 1 })
	if !l.e.wechsel.laedt || !strings.Contains(l.e.meldung.Text, "Opening") {
		t.Fatalf("lädt nicht: %+v", l.e.meldung)
	}
	if _, err := l.e.s.D.Hinzufuegen(document.Text{Ort: document.Pt(5, 15), Inhalt: "neu", Schriftgrad: 12, Farbe: color.NRGBA{A: 255}}); err != nil {
		t.Fatal(err)
	}
	close(l.tor)
	l.ruhig()
	if l.e.s.D != altD {
		t.Fatal("trotz Änderung gewechselt")
	}
	if r := l.rett[l.pfade[0]]; r == nil || r.ende != 1 || r.wieder != 0 {
		t.Errorf("Rettung der geladenen: %+v", r)
	}
	if l.alt.ende != 0 {
		t.Error("Rettung der offenen beendet")
	}
}

// ADR-0036 Punkt 1: Tastatur über die Fokusfahrt — Pfeile, Pos1 und Ende wandern, Enter öffnet.
func TestLeisteTastatur(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	l.e.s.Gespeichert()
	k := &l.e.unten
	l.fokusiere(&k.knoepfe[0])
	if !l.fokus(&k.knoepfe[0]) {
		t.Fatal("kein Fokus")
	}
	if !strings.Contains(l.e.hinweis(), "Enter opens") {
		t.Errorf("Hinweis zur Vorschau mit Fokus: %q", l.e.hinweis())
	}
	l.taste(key.NameRightArrow, 0)
	if !l.fokus(&k.knoepfe[1]) {
		t.Error("Pfeil rechts")
	}
	l.taste(key.NameEnd, 0)
	if !l.fokus(&k.knoepfe[2]) {
		t.Error("Ende")
	}
	l.taste(key.NameHome, 0)
	if !l.fokus(&k.knoepfe[0]) {
		t.Error("Pos1")
	}
	l.taste(key.NameLeftArrow, 0) // am Anfang: bleibt
	if !l.fokus(&k.knoepfe[0]) {
		t.Error("Pfeil links am Anfang")
	}
	l.taste(key.NameEnd, 0)
	l.taste(key.NameReturn, 0)
	l.ruhig()
	if a := l.aufrufe(); len(a) != 1 || a[0] != l.pfade[2] {
		t.Errorf("Enter öffnet: %v", a)
	}
}

// ADR-0036 Punkt 1: Die Leertaste wirkt an einer Vorschau wie Enter.
func TestLeisteLeertaste(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	l.e.s.Gespeichert()
	l.fokusiere(&l.e.unten.knoepfe[2])
	l.taste(key.NameSpace, 0)
	l.ruhig()
	if a := l.aufrufe(); len(a) != 1 || a[0] != l.pfade[2] {
		t.Errorf("Leertaste öffnet: %v", a)
	}
}

// Das Mausrad rollt waagerecht; die Pfeiltaste rollt eine verdeckte Vorschau ins Bild und gibt ihr den Fokus.
func TestLeisteRollen(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(900, 800), 12)
	k := &l.e.unten
	if k.l.RollMax(k.masse) == 0 || k.bis >= 12 {
		t.Fatalf("zu breit für den Test: %d bis %d", k.von, k.bis)
	}
	p := l.kachel(1)
	for range 4 { // in Stufen wie ein echtes Rad (Lehre aus F-598)
		l.r.Queue(pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Position: f32.Pt(float32(p.X), float32(p.Y)), Scroll: f32.Pt(0, 30)})
		l.setle()
	}
	if r := k.l.Rollen(k.masse); r != 120 {
		t.Errorf("gerollt %d", r)
	}
	if x0, _ := k.l.Lage(0, k.masse); x0 != leisteRandDp-120 {
		t.Errorf("Lage nach dem Rollen %d", x0)
	}
	l.r.Queue(pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Position: f32.Pt(float32(p.X), float32(p.Y)), Scroll: f32.Pt(0, -1000)})
	l.setle()
	if r := k.l.Rollen(k.masse); r != 0 {
		t.Errorf("zurück: %d", r)
	}
	l.fokusiere(&k.knoepfe[0])
	l.taste(key.NameEnd, 0)
	l.setle()
	if !l.fokus(&k.knoepfe[11]) || k.bis != 12 {
		t.Errorf("Ende: Fokus auf der letzten %v, gezeichnet %d bis %d", l.fokus(&k.knoepfe[11]), k.von, k.bis)
	}
	if x0, x1 := k.l.Lage(11, k.masse); x0 < 0 || x1 > k.masse.Breite {
		t.Errorf("letzte nicht sichtbar: %d bis %d", x0, x1)
	}
}

// ADR-0036 Punkt 7: Nach dem eigenen Speichern liest die Leiste den Index neu (eine Aufnahme aus einem zweiten Fenster
// erscheint) und lädt die Vorschau der offenen neu.
func TestLeisteNachSpeichernNeu(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	aktuell := l.e.unten.l.Aktuell()
	l.warte(func() bool { _, z := l.e.unten.v.Bild(aktuell); return z == libview.Da })
	legeAufnahmeAb(t, l.b, color.NRGBA{R: 10, G: 10}, time.Date(2026, 11, 24, 20, 0, 0, 0, time.UTC), false)
	l.taste("S", key.ModShortcut)
	l.ruhig()
	if n := len(l.e.unten.l.Eintraege()); n != 4 {
		t.Errorf("nach dem Speichern %d Aufnahmen", n)
	}
	if !l.e.unten.l.IstAktuell(2) {
		t.Error("offene Aufnahme nach dem Neulesen")
	}
}

// ADR-0036 Punkt 1: Speichert der Editor als neue Aufnahme (die offene Datei ließ sich nicht fortschreiben), wandert
// der grüne Rand zur neuen Datei.
func TestLeisteNachSpeichernAlsNeue(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	neu := legeAufnahmeAb(t, l.b, color.NRGBA{B: 90}, time.Date(2026, 11, 24, 21, 0, 0, 0, time.UTC), false)
	var mu sync.Mutex
	pfad := l.pfade[1]
	l.e.pfad = func() string { mu.Lock(); defer mu.Unlock(); return pfad }
	l.e.speichern = func(*document.Dokument) (Gespeichert, error) {
		mu.Lock()
		pfad = neu
		mu.Unlock()
		return Gespeichert{}, nil
	}
	l.taste("S", key.ModShortcut)
	l.ruhig()
	if !l.e.unten.l.IstAktuell(0) || l.e.unten.l.IstAktuell(2) {
		t.Errorf("grüner Rand nach dem Speichern als neue Aufnahme: %q", l.e.unten.l.Aktuell())
	}
}

// Leere Bibliothek: ein Satz statt Vorschauen.
func TestLeisteLeer(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 0)
	if len(l.e.unten.kacheln) != 0 || l.e.unten.rahmen.Empty() {
		t.Errorf("leer: %d Vorschauen, Leiste %v", len(l.e.unten.kacheln), l.e.unten.rahmen)
	}
}
