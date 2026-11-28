// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/widget"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/libview"
)

// bibLauf öffnet die volle Ansicht per Strg+Umschalt+L und wartet auf den Index. Die Uhr steht fest (Zeitraum).
func bibLauf(t *testing.T, anzahl int) *leistenLauf {
	t.Helper()
	alt := leisteUhr
	leisteUhr = func() time.Time { return time.Date(2026, 11, 24, 18, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { leisteUhr = alt })
	l := neuerLeistenLauf(t, image.Pt(1200, 800), anzahl)
	l.taste("L", key.ModShortcut|key.ModShift)
	l.warte(func() bool { return l.e.bibl.offen && !l.e.bibl.indexLaeuft && len(l.e.bibl.r.Eintraege()) == anzahl })
	return l
}

// bibKachel ist die Mitte der gezeichneten Vorschau der Stelle i in der Ansicht.
func (l *leistenLauf) bibKachel(i int) image.Point {
	l.t.Helper()
	b := &l.e.bibl
	if i < b.von || i >= b.bis {
		l.t.Fatalf("Stelle %d nicht gezeichnet (%d bis %d)", i, b.von, b.bis)
	}
	return mitte(b.kacheln[i-b.von])
}

// bibDruck drückt und löst die linke Taste an p zur Zeit zeit (Doppelklick: zweimal mit kleinem Abstand).
func (l *leistenLauf) bibDruck(p image.Point, zeit time.Duration) {
	pos := f32.Pt(float32(p.X), float32(p.Y))
	l.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: pos, Time: zeit})
	l.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: pos, Time: zeit})
	l.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: pos, Time: zeit})
	l.setle()
}

// bibRuhig wartet, bis Index, Arbeit, Speichern und Wechsel ruhen.
func (l *leistenLauf) bibRuhig() {
	l.t.Helper()
	l.warte(func() bool {
		b := &l.e.bibl
		return !b.indexLaeuft && !b.arbeitLaeuft && !l.e.s.SpeichernLaeuft() && !l.e.wechsel.laedt && !l.e.unten.indexLaeuft
	})
}

func (l *leistenLauf) rel(i int) string {
	r, ok := l.b.Relativ(l.pfade[i])
	if !ok {
		l.t.Fatalf("Pfad %d nicht in der Ablage", i)
	}
	return r
}

// ADR-0036 Punkt 3: Strg+Umschalt+L öffnet die Ansicht statt Werkzeugleiste, Leinwand und Leisten; ohne
// Agenten-Aufnahme, neueste zuerst nach Monaten; die offene Aufnahme grün und mit Fokus. Esc führt zurück, der Fokus
// geht an die Leinwand.
func TestBibliothekOeffnenUndSchliessen(t *testing.T) {
	l := bibLauf(t, 4)
	b := &l.e.bibl
	for i, e := range b.r.Eintraege() {
		if e.Pfad != l.rel(i) {
			t.Errorf("Stelle %d: %s", i, e.Pfad)
		}
	}
	if len(b.r.Gruppen()) != 1 || b.r.Gruppen()[0].Anzahl() != 4 {
		t.Errorf("Monate %+v", b.r.Gruppen())
	}
	if !b.r.IstAktuell(1) || b.r.Fokus() != 1 || b.ziel != bibZielRaster {
		t.Errorf("offene Aufnahme %v, Fokus %d, Ziel %d", b.r.IstAktuell(1), b.r.Fokus(), b.ziel)
	}
	if !l.fokus(b) {
		t.Error("die Ansicht hat den Tastaturfokus nicht")
	}
	if !l.e.unten.rahmen.Empty() || len(b.kacheln) != 4 {
		t.Errorf("Leiste %v, Kacheln %d", l.e.unten.rahmen, len(b.kacheln))
	}
	if b.sicht.Min.Y != menuezeileDp+bibKopfDp || b.sicht.Max.Y != 800-fusszeileDp || b.sicht.Dx() != 1200 {
		t.Errorf("Raster %v", b.sicht)
	}
	if !strings.Contains(l.e.hinweis(), "open now") {
		t.Errorf("Hinweis %q", l.e.hinweis())
	}
	for _, en := range l.e.menueEintraegeVon(menueDatei) {
		if en.akt == aktBibliothek && (!en.aktiv || !en.an) {
			t.Error("Menüeintrag Library ohne Haken")
		}
	}
	l.taste(key.NameEscape, 0)
	if b.offen || !l.fok(l.e) {
		t.Fatalf("Esc: offen %v, Fokus an der Leinwand %v", b.offen, l.fok(l.e))
	}
	if l.e.unten.rahmen.Empty() {
		t.Error("Leiste kommt nicht zurück")
	}
	// Knopf „Library“ der Leiste öffnet, „Back to image“ schließt.
	l.klick(mitte(l.e.unten.rahmen.Intersect(image.Rect(werkzeugeDp, 0, werkzeugeDp+leisteKnopfRand+leisteKnopfB, 800))))
	l.warte(func() bool { return b.offen && !b.indexLaeuft })
	l.klick(mitte(b.zurueckRahmen))
	if b.offen {
		t.Error("„Back to image“ schließt nicht")
	}
	// Menü Datei → Library öffnet, Strg+Umschalt+L schließt wieder.
	l.zuMenue(aktBibliothek)
	l.oeffneMitTaste()
	l.e.knopfPerTaste(&l.e.menue.eintraege[l.index(aktBibliothek)])
	l.setle()
	if !b.offen {
		t.Fatal("Menüeintrag öffnet nicht")
	}
	l.taste("L", key.ModShortcut|key.ModShift)
	if b.offen {
		t.Error("Strg+Umschalt+L schließt nicht")
	}
}

// Ohne Bibliothek ist der Eintrag grau mit Erklärung, und das Kürzel öffnet nichts.
func TestBibliothekOhneBibliothek(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.taste("L", key.ModShortcut|key.ModShift)
	if l.e.bibl.offen {
		t.Error("ohne Bibliothek geöffnet")
	}
	for _, en := range l.e.menueEintraegeVon(menueDatei) {
		if en.akt == aktBibliothek && (en.an || en.erklaerung == "") {
			t.Errorf("Eintrag ohne Bibliothek: an %v, Erklärung %q", en.an, en.erklaerung)
		}
	}
}

// ADR-0036 Punkt 4: Solange die Ansicht offen ist, wirken Werkzeugtasten, Leinwand-Kürzel und getippter Text nicht;
// Entf trifft die Ansicht, nicht das gewählte Objekt. Das Menü bleibt bedienbar, Bild-Einträge sind grau mit Erklärung.
func TestBibliothekSperrtDasBild(t *testing.T) {
	l := bibLauf(t, 3)
	vorher := len(l.e.s.D.Objekte())
	for _, k := range []struct {
		n key.Name
		m key.Modifiers
	}{{"R", 0}, {"B", 0}, {"T", 0}, {"X", key.ModShortcut | key.ModShift}, {"Z", key.ModShortcut},
		{key.NameDeleteBackward, 0}, {"D", key.ModShortcut}, {"H", key.ModShortcut}, {"B", key.ModShortcut | key.ModShift},
		{"R", key.ModShortcut | key.ModShift}, {"0", key.ModShortcut}, {"1", key.ModShortcut}} {
		l.taste(k.n, k.m)
	}
	l.tippe("Rx")
	// Entf an der offenen Aufnahme wechselt seit Karte #347 zur Nachbarin (TestBibliothekPapierkorb); hier zählt nur, dass
	// es das gewählte Objekt nicht trifft — an einer anderen Kachel.
	l.taste(key.NameRightArrow, 0)
	l.e.bibl.loeschRuheBis = leisteUhr().Add(time.Hour) // das ✕ ruht: Entf legt nichts in den Papierkorb
	l.e.bibl.arbeitLaeuft = true
	l.taste(key.NameDeleteForward, 0)
	l.e.bibl.arbeitLaeuft = false
	l.taste(key.NameLeftArrow, 0)
	if l.e.s.Werkzeug != canvas.WerkzeugAuswahl || len(l.e.s.D.Objekte()) != vorher || l.e.s.Auswahl != l.id ||
		l.inhalt() != "Hallo" || l.e.seiteZu || !l.e.unten.an {
		t.Errorf("Bild verändert: Werkzeug %v, Objekte %d, Auswahl %d, Inhalt %q, Seite zu %v, Leiste %v",
			l.e.s.Werkzeug, len(l.e.s.D.Objekte()), l.e.s.Auswahl, l.inhalt(), l.e.seiteZu, l.e.unten.an)
	}
	if !l.e.bibl.offen || !strings.Contains(l.e.meldung.Text, "still working") {
		t.Errorf("offen %v, Meldung %q", l.e.bibl.offen, l.e.meldung.Text)
	}
	for _, d := range l.pfade {
		if _, err := os.Stat(d); err != nil {
			t.Errorf("Aufnahme fehlt: %v", err)
		}
	}
	// Ein Klick in die Fläche, wo sonst die Leinwand liegt, zeichnet nichts.
	l.e.s.SetzeWerkzeug(canvas.WerkzeugKasten)
	l.maus(pointer.Move, image.Pt(700, 600), 0)
	l.maus(pointer.Press, image.Pt(700, 600), pointer.ButtonPrimary)
	l.maus(pointer.Move, image.Pt(900, 700), pointer.ButtonPrimary)
	l.maus(pointer.Release, image.Pt(900, 700), 0)
	l.setle()
	if len(l.e.s.D.Objekte()) != vorher {
		t.Error("Zug in der Ansicht hat gezeichnet")
	}
	l.e.s.SetzeWerkzeug(canvas.WerkzeugAuswahl)
	// Die Felder Farbe und Zoom der Fußzeile öffnen keine Liste.
	for _, a := range []canvas.FeldArt{canvas.FeldFarbe, canvas.FeldZoom} {
		r, ok := l.e.fussFeldRahmen[a]
		if !ok {
			t.Fatalf("Feld %v nicht gezeichnet", a)
		}
		l.klick(mitte(r))
		if l.e.menue.offen {
			t.Errorf("Feld %v öffnet in der Ansicht eine Liste", a)
			l.taste(key.NameEscape, 0)
		}
	}
	// Menü: Bild und Werkzeuge grau mit Erklärung, Bearbeiten bedient die Ansicht, Datei → Settings bleibt.
	for _, m := range []int{menueBild, menueWerkzeuge, menueEbene, menueAnsicht} {
		for _, en := range l.e.menueEintraegeVon(m) {
			if en.an || en.erklaerung == "" {
				t.Errorf("Menü %d, %s: an %v, Erklärung %q", m, en.text, en.an, en.erklaerung)
			}
		}
	}
	var akt []aktion
	for _, en := range l.e.menueEintraegeVon(menueBearbeiten) {
		akt = append(akt, en.akt)
		if menueKuerzel(en) == "" {
			t.Errorf("%s ohne Kürzel", en.text)
		}
	}
	if !slices.Equal(akt, []aktion{aktBibOeffnen, aktBibSchlagworte, aktBibPapierkorb, aktBibSuchen}) {
		t.Errorf("Bearbeiten: %v", akt)
	}
	for _, en := range l.e.menueEintraegeVon(menueDatei) {
		if gilt := en.akt == aktEinstellungen || en.akt == aktSchliessen || en.akt == aktSpeichern || en.akt == aktBibliothek ||
			en.akt == aktSpeichernUnter || en.akt == aktNeueAufnahme || en.akt == aktNeueAufnahme3 || en.akt == aktNeueAufnahme5 ||
			en.akt == aktNeueAufnahme10; gilt != en.an { // ADR-0046 Punkte 6 und 7, ADR-0042 Nachtrag 2
			t.Errorf("Datei, %s: an %v", en.text, en.an)
		}
	}
	// Schlagworte sind an der offenen Aufnahme grau mit Erklärung; der Papierkorb geht über den Wechsel (Karte #347).
	for _, en := range l.e.menueEintraegeVon(menueBearbeiten) {
		if en.akt == aktBibSchlagworte && (en.an || en.erklaerung == "") {
			t.Errorf("%s an der offenen Aufnahme: an %v", en.text, en.an)
		}
		if en.akt == aktBibPapierkorb && !en.an {
			t.Errorf("%s an der offenen Aufnahme grau", en.text)
		}
	}
	// F10 öffnet das Menü, Esc schließt es; der Fokus kehrt in die Ansicht zurück.
	l.taste(key.NameF10, 0)
	if !l.e.menue.offen {
		t.Fatal("F10 öffnet das Menü nicht")
	}
	l.taste(key.NameEscape, 0)
	if l.e.menue.offen || !l.e.bibl.offen || !l.fok(&l.e.bibl) {
		t.Errorf("nach Esc im Menü: Menü %v, Ansicht %v, Fokus %v", l.e.menue.offen, l.e.bibl.offen, l.fok(&l.e.bibl))
	}
	// Strg+, öffnet die Einstellungen über der Ansicht; Esc dort führt zurück in die Ansicht.
	eigeneEinstellungen(t, "")
	l.taste(",", key.ModShortcut)
	if !l.e.einst.offen {
		t.Fatal("Strg+, wirkt in der Ansicht nicht")
	}
	l.taste(key.NameEscape, 0)
	if l.e.einst.offen || !l.e.bibl.offen || !l.fok(&l.e.bibl) {
		t.Errorf("nach der Tafel: Tafel %v, Ansicht %v, Fokus %v", l.e.einst.offen, l.e.bibl.offen, l.fok(&l.e.bibl))
	}
}

// Pfeile wandern, Enter öffnet eine andere Aufnahme im selben Fenster (über wechselAnfrage); danach ist die Ansicht zu
// und der Fokus auf der Leinwand. Enter auf der offenen schließt nur.
func TestBibliothekEnterOeffnet(t *testing.T) {
	l := bibLauf(t, 4)
	b := &l.e.bibl
	l.taste(key.NameReturn, 0) // Fokus auf der offenen (1)
	if b.offen || len(l.aufrufe()) != 0 {
		t.Fatalf("Enter auf der offenen: offen %v, geladen %v", b.offen, l.aufrufe())
	}
	l.taste("L", key.ModShortcut|key.ModShift)
	l.bibRuhig()
	l.taste(key.NameRightArrow, 0)
	l.taste(key.NameRightArrow, 0)
	l.taste(key.NameLeftArrow, 0)
	if b.r.Fokus() != 2 {
		t.Fatalf("Fokus %d", b.r.Fokus())
	}
	l.taste(key.NameReturn, 0)
	l.bibRuhig()
	if got := l.aufrufe(); len(got) != 1 || got[0] != l.pfade[2] {
		t.Fatalf("geladen %v", got)
	}
	if b.offen || !l.fok(l.e) || l.e.titel != filepath.Base(l.pfade[2]) {
		t.Errorf("nach dem Wechsel: offen %v, Fokus Leinwand %v, Titel %s", b.offen, l.fok(l.e), l.e.titel)
	}
}

// Ein Klick setzt nur den Fokus, ein Doppelklick öffnet.
func TestBibliothekDoppelklick(t *testing.T) {
	l := bibLauf(t, 4)
	b := &l.e.bibl
	p := l.bibKachel(3)
	l.bibDruck(p, time.Second)
	if b.r.Fokus() != 3 || len(l.aufrufe()) != 0 || !b.offen {
		t.Fatalf("ein Klick: Fokus %d, geladen %v", b.r.Fokus(), l.aufrufe())
	}
	l.bibDruck(p, 3*time.Second) // zu spät: wieder nur ein Klick
	if len(l.aufrufe()) != 0 {
		t.Fatal("zwei langsame Klicks öffnen")
	}
	l.bibDruck(p, 3*time.Second+200*time.Millisecond)
	l.bibRuhig()
	if got := l.aufrufe(); len(got) != 1 || got[0] != l.pfade[3] || b.offen {
		t.Errorf("Doppelklick: geladen %v, offen %v", got, b.offen)
	}
}

// ADR-0036 Punkt 4: Entf legt die Aufnahme mit Fokus in den Papierkorb (rückholbar), die Liste rückt nach; die offene
// ist ausgenommen.
func TestBibliothekPapierkorb(t *testing.T) {
	l := bibLauf(t, 4)
	b := &l.e.bibl
	l.taste(key.NameRightArrow, 0) // Stelle 2
	l.taste(key.NameDeleteForward, 0)
	l.bibRuhig()
	if _, err := os.Stat(l.pfade[2]); !os.IsNotExist(err) {
		t.Fatalf("Datei noch da: %v", err)
	}
	papier, _ := os.ReadDir(filepath.Join(l.b.Orte.Papierkorb, "files"))
	if len(papier) != 1 {
		t.Errorf("Papierkorb %v", papier)
	}
	if len(b.r.Eintraege()) != 3 || b.r.Fokus() != 2 || b.r.Eintraege()[2].Pfad != l.rel(3) {
		t.Errorf("nachgerückt: %d Einträge, Fokus %d", len(b.r.Eintraege()), b.r.Fokus())
	}
	if !strings.Contains(l.e.meldung.Text, "trash") || l.e.meldung.Fehler {
		t.Errorf("Meldung %q", l.e.meldung.Text)
	}
	if len(l.e.unten.l.Eintraege()) != 3 {
		t.Errorf("Leiste nicht neu gelesen: %d", len(l.e.unten.l.Eintraege()))
	}
	// Die offene Aufnahme (Stelle 1): erst Wechsel zur nächst älteren (jetzt Stelle 2), dann Papierkorb; die Ansicht
	// bleibt offen (Karte #347).
	b.loeschRuheBis = time.Time{}
	l.taste(key.NameLeftArrow, 0)
	l.taste(key.NameDeleteForward, 0)
	l.warte(func() bool { return !l.e.wechsel.laedt })
	l.bibRuhig()
	if _, err := os.Stat(l.pfade[1]); !os.IsNotExist(err) || len(b.r.Eintraege()) != 2 {
		t.Errorf("offene Aufnahme nicht im Papierkorb: %v, %d Einträge", err, len(b.r.Eintraege()))
	}
	if a := l.aufrufe(); len(a) != 1 || a[0] != l.pfade[3] {
		t.Errorf("geladen %v, Soll %s", a, l.pfade[3])
	}
	if !b.offen || l.e.pfad() != l.pfade[3] {
		t.Errorf("Ansicht offen %v, offen ist %s", b.offen, l.e.pfad())
	}
}

// ADR-0036 Punkt 4: F2 öffnet das Feld unter der Kachel mit den Schlagworten, Komma trennt, Enter übernimmt über
// SetzeSchlagworte, Esc bricht ab; an der offenen Aufnahme geht es nicht.
func TestBibliothekSchlagworte(t *testing.T) {
	l := bibLauf(t, 3)
	b := &l.e.bibl
	l.taste(key.NameF2, 0) // offene Aufnahme
	if b.bearbeite != "" || !strings.Contains(l.e.meldung.Text, "open capture") {
		t.Fatalf("F2 an der offenen: %q, Meldung %q", b.bearbeite, l.e.meldung.Text)
	}
	l.taste(key.NameLeftArrow, 0) // Stelle 0
	l.taste(key.NameF2, 0)
	if b.bearbeite != l.rel(0) || b.worte != "" {
		t.Fatalf("Feld %q %q", b.bearbeite, b.worte)
	}
	l.tippe("bug, Login ,bug")
	l.taste(key.NameDeleteBackward, 0)
	l.tippe("x")
	l.taste("R", 0) // Werkzeugtaste wirkt im Feld nicht
	if b.gezeigteWorte != "bug, Login ,bux" || b.feldRahmen.Empty() || l.e.s.Werkzeug != canvas.WerkzeugAuswahl {
		t.Errorf("Feld zeigt %q, Werkzeug %v", b.gezeigteWorte, l.e.s.Werkzeug)
	}
	l.taste(key.NameReturn, 0)
	l.bibRuhig()
	if got := b.r.Eintraege()[0].Schlagworte; !slices.Equal(got, []string{"bug", "Login", "bux"}) {
		t.Errorf("Schlagworte %q", got)
	}
	if b.bearbeite != "" || !b.offen {
		t.Error("Feld bleibt offen oder Ansicht zu")
	}
	// Erneut: Feld beginnt mit den Schlagworten; Esc bricht ab, die Ansicht bleibt offen.
	l.taste(key.NameF2, 0)
	if b.worte != "bug, Login, bux" {
		t.Errorf("Anfang des Felds %q", b.worte)
	}
	l.tippe(", neu")
	l.taste(key.NameEscape, 0)
	l.bibRuhig()
	if b.bearbeite != "" || !b.offen || len(b.r.Eintraege()[0].Schlagworte) != 3 {
		t.Errorf("Esc: Feld %q, offen %v, Schlagworte %q", b.bearbeite, b.offen, b.r.Eintraege()[0].Schlagworte)
	}
	// Suche nach dem Schlagwort (Strg+F) zeigt nur diese Aufnahme; Rücktaste löscht im Suchfeld.
	l.taste("F", key.ModShortcut)
	l.tippe("LOGIN")
	l.bibRuhig()
	if len(b.r.Eintraege()) != 1 || b.gezeigteSuche != "LOGIN" {
		t.Errorf("Suche: %d Treffer, Feld %q", len(b.r.Eintraege()), b.gezeigteSuche)
	}
	l.tippe("x")
	l.bibRuhig()
	if len(b.r.Eintraege()) != 0 || !strings.Contains(b.leerText, "match") {
		t.Errorf("keine Treffer: %d, Text %q", len(b.r.Eintraege()), b.leerText)
	}
	l.taste(key.NameDeleteBackward, 0)
	l.bibRuhig()
	if len(b.r.Eintraege()) != 1 {
		t.Errorf("nach Rücktaste: %d", len(b.r.Eintraege()))
	}
	// Teil eines Schlagworts genügt (F-671).
	for range len("LOGIN") {
		l.taste(key.NameDeleteBackward, 0)
	}
	l.tippe("OGI")
	l.bibRuhig()
	if len(b.r.Eintraege()) != 1 || b.gezeigteSuche != "OGI" {
		t.Errorf("Teilwort: %d Treffer, Feld %q", len(b.r.Eintraege()), b.gezeigteSuche)
	}
}

// Tab fährt durch Suchfeld, Zeitraum, Schalter, Zurück und Raster; der Zeitraum öffnet per Enter seine Liste, eine Wahl
// filtert. Der Schalter blendet Agenten-Aufnahmen ein und ist beim nächsten Öffnen wieder aus.
func TestBibliothekKopf(t *testing.T) {
	l := bibLauf(t, 3)
	b := &l.e.bibl
	var ziele []int
	for range bibZiele {
		l.taste(key.NameTab, 0)
		ziele = append(ziele, b.ziel)
	}
	if !slices.Equal(ziele, []int{bibZielSuche, bibZielZeitraum, bibZielAgenten, bibZielZurueck, bibZielRaster}) {
		t.Errorf("Tab-Fahrt %v", ziele)
	}
	l.taste(key.NameTab, key.ModShift)
	if b.ziel != bibZielZurueck || !l.fok(b) {
		t.Errorf("Umschalt+Tab: %d, Fokus %v", b.ziel, l.fok(b))
	}
	// Agenten-Schalter per Klick.
	l.klick(mitte(b.agentenRahmen))
	l.bibRuhig()
	if !b.agenten || len(b.r.Eintraege()) != 4 || !b.r.Eintraege()[0].Agent() {
		t.Errorf("mit Agenten: %v, %d", b.agenten, len(b.r.Eintraege()))
	}
	// Zeitraum per Tastatur: „Today“ (die Uhr steht auf dem 24.11. 18 Uhr; Aufnahmen ab 8 Uhr stündlich).
	b.ziel = bibZielZeitraum
	l.taste(key.NameReturn, 0)
	if !l.e.menue.offen || l.e.menue.welches != listeZeitraum || l.e.listenAnker(listeZeitraum) != b.zeitRahmen {
		t.Fatalf("Liste: offen %v, welche %d", l.e.menue.offen, l.e.menue.welches)
	}
	if got := l.e.menue.cursor; libview.Zeitraeume[got] != libview.ZeitraumAlle {
		t.Errorf("Cursor auf %d", got)
	}
	l.klick(mitte(l.e.menue.rahmen[0])) // Today
	l.bibRuhig()
	if b.zeitraum != libview.ZeitraumHeute || l.e.menue.offen || !l.fok(b) || len(b.r.Eintraege()) != 4 {
		t.Errorf("Heute: %d, Menü %v, Fokus %v, %d Einträge", b.zeitraum, l.e.menue.offen, l.fok(b), len(b.r.Eintraege()))
	}
	// Schließen und wieder öffnen: Schalter aus, Zeitraum alles.
	l.taste(key.NameEscape, 0)
	l.taste("L", key.ModShortcut|key.ModShift)
	l.bibRuhig()
	if b.agenten || b.zeitraum != libview.ZeitraumAlle || len(b.r.Eintraege()) != 3 {
		t.Errorf("wieder geöffnet: Agenten %v, Zeitraum %d, %d", b.agenten, b.zeitraum, len(b.r.Eintraege()))
	}
}

// Mausrad rollt das Raster; der Fokus mit Pfeil runter rollt ihn ins Bild; Bild↓ blättert.
func TestBibliothekRollen(t *testing.T) {
	l := bibLauf(t, 40) // 8 Zeilen à 5
	b := &l.e.bibl
	if rmax := b.r.RollMax(b.masse); rmax <= 0 {
		t.Fatalf("RollMax %d", rmax)
	}
	p := mitte(b.sicht)
	l.r.Queue(pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Position: f32.Pt(float32(p.X), float32(p.Y)),
		Scroll: f32.Pt(0, 600)})
	l.setle()
	if r := b.r.Rollen(b.masse); r != 600 {
		t.Errorf("Mausrad: %d", r)
	}
	if b.von == 0 {
		t.Error("gerollt: erste Zeile noch geladen")
	}
	l.taste(key.NameHome, 0)
	if b.r.Fokus() != 0 || b.r.Rollen(b.masse) != 0 {
		t.Errorf("Pos1: Fokus %d, Rollen %d", b.r.Fokus(), b.r.Rollen(b.masse))
	}
	l.taste(key.NamePageDown, 0)
	if f := b.r.Fokus(); f <= 4 || !b.kacheln[f-b.von].In(b.sicht) {
		t.Errorf("Bild runter: Fokus %d nicht ganz sichtbar", f)
	}
	l.taste(key.NameEnd, 0)
	if b.r.Fokus() != 39 || b.r.Rollen(b.masse) != b.r.RollMax(b.masse) {
		t.Errorf("Ende: Fokus %d, Rollen %d", b.r.Fokus(), b.r.Rollen(b.masse))
	}
	l.taste(key.NameUpArrow, 0)
	if b.r.Fokus() != 34 {
		t.Errorf("hoch: %d", b.r.Fokus())
	}
}

// Ein Wechsel, der nicht zustande kommt (Speichern scheitert), lässt die Ansicht offen; die Meldung steht im Fuß.
func TestBibliothekWechselScheitert(t *testing.T) {
	l := bibLauf(t, 3)
	b := &l.e.bibl
	l.e.speichern = func(*document.Dokument) (Gespeichert, error) { return Gespeichert{}, errors.New("Platte voll") }
	l.taste(key.NameLeftArrow, 0)
	l.taste(key.NameReturn, 0)
	l.bibRuhig()
	if !b.offen || len(l.aufrufe()) != 0 || !l.e.meldung.Fehler {
		t.Errorf("offen %v, geladen %v, Meldung %q", b.offen, l.aufrufe(), l.e.meldung.Text)
	}
	if !strings.Contains(l.e.hinweis(), "Capture of") {
		t.Errorf("Hinweis %q", l.e.hinweis())
	}
}

// F-671: Ändert sich die Suche, während der Index noch gelesen wird, gilt das Ergebnis der neuen Suche; das späte
// Ergebnis der alten wird verworfen.
func TestBibliothekUeberholteSuche(t *testing.T) {
	l := bibLauf(t, 3)
	if _, err := l.b.SetzeSchlagworte(l.rel(0), []string{"login"}); err != nil {
		t.Fatal(err)
	}
	b := &l.e.bibl
	b.suche = "zzz"
	l.e.bibliothekNeuLesen()
	b.suche = "log"
	l.e.bibliothekNeuLesen() // läuft noch: wird vorgemerkt
	if !b.indexNochmal {
		t.Fatal("zweite Suche nicht vorgemerkt")
	}
	l.bibRuhig()
	if es := b.r.Eintraege(); len(es) != 1 || es[0].Pfad != l.rel(0) {
		t.Errorf("Ergebnis der alten Suche gilt: %d Treffer", len(es))
	}
}

// F-672: Ist die Datei einer Aufnahme außerhalb von Schnapp verschwunden (im Dateimanager gelöscht), scheitert das
// Öffnen mit Meldung; danach gleicht das Lesen den Index ab, und der Eintrag verschwindet aus der Leiste.
func TestLeisteFehlendeDateiVerschwindet(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	l.e.s.Gespeichert()
	weg := l.rel(0) // vor dem Löschen: Relativ prüft die Datei
	if err := os.Remove(l.pfade[0]); err != nil {
		t.Fatal(err)
	}
	l.fehler = errors.New("Die Datei gibt es nicht.")
	l.klick(l.kachel(0))
	l.ruhig()
	if !l.e.meldung.Fehler || !strings.Contains(l.e.meldung.Text, "Datei gibt es nicht") {
		t.Errorf("Meldung %+v", l.e.meldung)
	}
	l.warte(func() bool { return !l.e.unten.indexLaeuft && len(l.e.unten.l.Eintraege()) == 2 })
	for _, e := range l.e.unten.l.Eintraege() {
		if e.Pfad == weg {
			t.Error("Eintrag der fehlenden Datei steht noch in der Leiste")
		}
	}
}

// F-672: Beim Öffnen der vollen Ansicht wird der Index vorher abgeglichen.
func TestBibliothekOeffnenGleichtAb(t *testing.T) {
	l := bibLauf(t, 3)
	l.taste(key.NameEscape, 0)
	l.bibRuhig()
	if err := os.Remove(l.pfade[2]); err != nil {
		t.Fatal(err)
	}
	l.taste("L", key.ModShortcut|key.ModShift)
	l.warte(func() bool { return l.e.bibl.offen && !l.e.bibl.indexLaeuft && len(l.e.bibl.r.Eintraege()) == 2 })
}

// F-672: Scheitert der Papierkorb, weil die Datei inzwischen fehlt, verschwindet der Eintrag nach dem Abgleich.
func TestBibliothekPapierkorbFehlendeDatei(t *testing.T) {
	l := bibLauf(t, 4)
	b := &l.e.bibl
	if err := os.Remove(l.pfade[2]); err != nil {
		t.Fatal(err)
	}
	l.taste(key.NameRightArrow, 0) // Stelle 2
	l.taste(key.NameDeleteForward, 0)
	l.bibRuhig()
	if !l.e.meldung.Fehler {
		t.Errorf("keine Fehlermeldung: %+v", l.e.meldung)
	}
	l.warte(func() bool { return !b.indexLaeuft && len(b.r.Eintraege()) == 3 && len(l.e.unten.l.Eintraege()) == 3 })
}

// F-672: Beim Start liest die Leiste den Index abgeglichen.
func TestLeisteStartGleichtAb(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	if err := os.Remove(l.pfade[0]); err != nil {
		t.Fatal(err)
	}
	l.e.leisteStart()
	l.warte(func() bool { return !l.e.unten.indexLaeuft && len(l.e.unten.l.Eintraege()) == 2 })
}

// F-674: Bei offener Ansicht öffnen Farb- und Zoomfeld der Fußzeile ihre Liste auch per Tastatur (Enter) nicht.
func TestBibliothekFusslistenGesperrt(t *testing.T) {
	l := bibLauf(t, 3)
	e := l.e
	for _, c := range []*widget.Clickable{&e.knoepfe.fussFarbe, &e.knoepfe.fussZoom} {
		e.knopfPerTaste(c)
		if e.menue.offen && (e.menue.welches == listeFarbe || e.menue.welches == listeZoom) {
			t.Errorf("Liste %d bei offener Ansicht geöffnet", e.menue.welches)
		}
	}
	// Auch ein Mausklick auf die Felder öffnet nichts (fussAuswerten; Mutant BV21).
	for _, art := range []canvas.FeldArt{canvas.FeldFarbe, canvas.FeldZoom} {
		l.setle()
		r, ok := e.fussFeldRahmen[art]
		if !ok || r.Empty() {
			t.Fatalf("Feld %v der Fußzeile nicht gezeichnet", art)
		}
		l.klick(mittenPunkt(r))
		l.setle()
		if e.menue.offen && (e.menue.welches == listeFarbe || e.menue.welches == listeZoom) {
			t.Errorf("Klick auf Feld %v öffnet Liste %d bei offener Ansicht", art, e.menue.welches)
		}
	}
	l.taste(key.NameEscape, 0)
	l.bibRuhig()
	e.knopfPerTaste(&e.knoepfe.fussZoom) // ohne Ansicht geht es wie immer
	if !e.menue.offen || e.menue.welches != listeZoom {
		t.Error("Zoomliste ohne Ansicht nicht geöffnet")
	}
}

// F-676: Solange die Ansicht Schlagworte schreibt oder in den Papierkorb legt, lädt kein Wechsel.
func TestWechselWartetAufArbeitDerAnsicht(t *testing.T) {
	l := bibLauf(t, 3)
	e := l.e
	e.s.Gespeichert() // ohne Änderung lädt ein Wechsel sofort (sonst wartete er schon aufs Speichern)
	tor := make(chan struct{})
	e.bibArbeiten(bibArbeit{rel: l.rel(0)}, func() error { <-tor; return nil })
	e.wechselAnfrage(l.pfade[0])
	time.Sleep(50 * time.Millisecond)
	if a := l.aufrufe(); len(a) != 0 || e.wechsel.laedt {
		t.Errorf("geladen während der Arbeit: %v", a)
	}
	if e.leisteAn() {
		t.Error("Leiste bedienbar während der Arbeit")
	}
	close(tor)
	l.bibRuhig()
}

// F-678 Punkt 1: Ist die Bibliothek beschäftigt, bleibt das Schlagwortfeld mit der Eingabe offen (statt still zu
// verwerfen), und die Statuszeile sagt es.
func TestBibliothekSchlagworteWartenStattVerwerfen(t *testing.T) {
	l := bibLauf(t, 3)
	b := &l.e.bibl
	b.bearbeite, b.worte = l.rel(0), "neu"
	b.arbeitLaeuft = true
	l.e.bibSchlagworteUebernehmen()
	b.arbeitLaeuft = false
	if b.bearbeite != l.rel(0) || b.worte != "neu" {
		t.Errorf("Feld geschlossen oder Eingabe verloren: %q %q", b.bearbeite, b.worte)
	}
	if !strings.Contains(l.e.meldung.Text, "wait") {
		t.Errorf("Meldung %q", l.e.meldung.Text)
	}
}

// F-678 Punkt 4: Während eine andere Aufnahme lädt, öffnet die Ansicht nicht (sie schlösse gleich wieder).
func TestBibliothekOeffnetNichtWaehrendWechsel(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	l.e.s.Gespeichert()
	l.tor = make(chan struct{})
	l.klick(l.kachel(0))
	l.warte(func() bool { return len(l.aufrufe()) == 1 })
	l.taste("L", key.ModShortcut|key.ModShift)
	if l.e.bibl.offen {
		t.Error("Ansicht während des Wechsels geöffnet")
	}
	close(l.tor)
	l.ruhig()
}

// F-678 Punkt 2: Liegt die Aufnahme im Papierkorb und ist nur der Index nicht nachgeführt (library.ErrIndex), meldet
// die Ansicht den Erfolg und gleicht beim nächsten Lesen ab.
func TestBibliothekPapierkorbIndexfehlerIstErfolg(t *testing.T) {
	l := bibLauf(t, 3)
	l.e.unten.abgleich, l.e.bibl.abgleich = false, false
	l.e.bibliothekArbeitFertig(bibArbeit{rel: l.rel(0), papierkorb: true, err: fmt.Errorf("library: %w", library.ErrIndex)})
	if l.e.meldung.Fehler || !strings.Contains(l.e.meldung.Text, "trash") {
		t.Errorf("Meldung %+v", l.e.meldung)
	}
	l.bibRuhig()
}
