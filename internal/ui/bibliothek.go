// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Volle Ansicht der Bibliothek im Editor-Fenster (ADR-0036 Punkte 3 und 4): Menü Datei → „Library“, Strg+Umschalt+L
// oder der Knopf links in der Leiste der letzten Aufnahmen. Sie ersetzt Werkzeugleiste, Leinwand, rechte Leiste und die
// Leiste unten; Menüzeile und Fußzeile bleiben. Kopf: Suchfeld (Schlagwort), Zeitraum, Schalter für Agenten-Aufnahmen,
// „Back to image“. Darunter das Raster nach Monaten, neueste zuerst.
//
// Wie die Einstellungstafel hält die Ansicht den Tastaturfokus selbst (Gio-Fokus auf &e.bibl, die Stelle im Kopf oder
// im Raster in bibAnsicht.ziel): Tab bleibt in der Ansicht, getippter Text gehört dem Such- bzw. Schlagwortfeld.
// bibliothekAuswerten läuft im Bild vor feldTasten und tasten; tasten lässt von den übrigen Tasten nur F10 und die
// Kürzel der Menüeinträge durch, die in der Ansicht bedienbar sind (bibliothekErlaubt). Werkzeugtasten, Leinwand-Kürzel,
// Leinwand und Leisten wirken nicht, solange sie offen ist. Raster, Monate, Fokus, Blättern, Zeitraum und das Zerlegen
// der Schlagworte stehen ohne Gio in internal/libview.

import (
	"fmt"
	"image"
	"image/color"
	"path/filepath"
	"time"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/libview"
)

// Maße der Ansicht in dp (Muster 174 b).
const (
	bibKopfDp       = 62  // Kopf in einer Zeile
	bibKopfZweiDp   = 104 // Kopf in zwei Zeilen (schmales Fenster)
	bibKopfEinzeilg = 920 // ab dieser Fensterbreite passt der Kopf in eine Zeile
	bibRandDp       = 24
	bibObenDp       = 12
	bibKachelB      = 206
	bibBildH        = 128 // Vorschau; darunter zwei Textzeilen
	bibTextH        = 44
	bibLueckeX      = 16
	bibLueckeY      = 24
	bibMonatDp      = 34 // Monatsüberschrift samt Abstand
	bibSucheDp      = 350
	bibZeitraumDp   = 200
)

// Stellen im Kopf und das Raster für die Tastatur (Tab-Reihenfolge).
const (
	bibZielSuche = iota
	bibZielZeitraum
	bibZielAgenten
	bibZielZurueck
	bibZielRaster
	bibZiele
)

// bibAnsicht ist der Zustand der vollen Ansicht.
type bibAnsicht struct {
	abgleich bool // das nächste Lesen gleicht den Index vorher ab (F-672), wie untenLeiste.abgleich
	offen    bool
	r        *libview.Raster
	suche    string
	zeitraum libview.Zeitraum
	agenten  bool // Agenten-Aufnahmen zeigen; bei jedem Öffnen aus, nicht gemerkt (ADR-0036 Punkt 3)
	ziel     int  // Stelle mit Tastaturfokus (bibZiel…)
	// fokusAnfang: Nach dem Öffnen bekommt die offene Aufnahme (sonst die neueste) den Fokus, sobald der Index da ist.
	fokusAnfang bool
	// fokusLeinwand: Die Ansicht wurde ohne Bild-Kontext geschlossen (Wechsel); das nächste Bild gibt den Fokus an die
	// Leinwand.
	fokusLeinwand bool

	// Schlagworte bearbeiten (F2): Pfad der Aufnahme relativ zur Ablage (leer: keine Bearbeitung) und die Eingabe.
	bearbeite string
	worte     string

	// Index im Hintergrund: höchstens ein Lesen zugleich; nochmal heißt, das laufende ist veraltet (Suche, Zeitraum oder
	// Schalter haben sich geändert) und wird verworfen, ein neues folgt.
	index        chan indexErgebnis
	indexLaeuft  bool
	indexNochmal bool
	indexFehler  bool

	// Papierkorb und Schlagworte schreiben die Bibliothek im Hintergrund; währenddessen ruht die Bedienung des Rasters.
	arbeit       chan bibArbeit
	arbeitLaeuft bool

	sucheKnopf, zeitraumKnopf, agentenKnopf, zurueckKnopf widget.Clickable
	flaeche                                               int // Zeigerziel des Rasters (Klick, Mausrad, Zeiger)

	// Stand beim letzten Zeichnen (Fensterpixel): Klicks und Treffer beziehen sich auf das, was zu sehen war.
	masse                            libview.RasterMasse
	sicht                            image.Rectangle // Raster
	suchRahmen, zeitRahmen           image.Rectangle
	agentenRahmen, zurueckRahmen     image.Rectangle
	kacheln                          []image.Rectangle // Vorschauen der gezeichneten Stellen von bis bis (Tests)
	von, bis                         int
	ueber                            int // Kachel unter dem Zeiger; -1: keine
	ops                              map[string]leisteBild
	letzterKlick                     time.Duration // Zeit des letzten Drucks (Doppelklick)
	letzteKachel                     int
	hinweis                          string // Text für die Statuszeile
	leerText                         string // was das leere Raster sagt (Tests)
	gezeigteWorte, gezeigteSuche     string // Text der Felder beim letzten Zeichnen (Tests)
	feldRahmen                       image.Rectangle
	zeitraumListeAnker, kopfZweiZeil bool
}

// bibArbeit ist das Ergebnis einer Arbeit im Hintergrund.
type bibArbeit struct {
	papierkorb bool // sonst Schlagworte
	rel        string
	err        error
}

// bibliothekStart legt den Zustand an (mit der Leiste, nur mit Bibliothek).
func (e *editor) bibliothekStart() {
	e.bibl.r = libview.NeuesRaster(leisteUhr().Location())
	e.bibl.ueber, e.bibl.letzteKachel = -1, -1
}

// bibliothekGrau ist die Erklärung am grauen Menüeintrag ohne Bibliothek (wie die Leiste, ADR-0026 Nachtrag 8).
func (e *editor) bibliothekGrau() string { return e.leisteGrau() }

// offenRel ist der Pfad der offenen Aufnahme relativ zur Ablage; leer bei einer Datei von außen.
func (e *editor) offenRel() string {
	if e.pfad == nil || e.bib == nil || e.bib.Bibliothek == nil {
		return ""
	}
	if r, ok := e.bib.Bibliothek.Relativ(e.pfad()); ok {
		return r
	}
	return ""
}

// bibliothekOeffnen öffnet die Ansicht: Filter zurückgesetzt, der Schalter für Agenten-Aufnahmen aus, Index neu gelesen
// (ADR-0036 Punkte 3 und 7). Eine laufende Texteingabe wird bestätigt, ein Eingabefeld der Leiste verworfen; während
// eines Zugs öffnet sie nicht.
func (e *editor) bibliothekOeffnen(gtx layout.Context) {
	b := &e.bibl
	if e.unten.l == nil || b.r == nil || b.offen || e.s.Zieht() {
		return
	}
	if e.wechsel.laedt {
		// Die übernommene Aufnahme schlösse die Ansicht gleich wieder (bibliothekNachWechsel); erst nach dem Laden
		// öffnen (F-678 Punkt 4).
		e.melde(i18n.T("Opening another capture. Open the library afterwards."), false)
		return
	}
	e.s.TextBestaetigen()
	if f := e.fokusFeld(gtx); f != nil {
		f.neu, f.text, f.fehler = true, "", false
		e.umbenennenEnde(f)
	}
	e.seite.fw.hatteFokus = nil
	e.seite.sw.offen = false
	e.hilfeOffen = false
	e.zeigerFenster = nil
	b.offen = true
	b.suche, b.zeitraum, b.agenten = "", libview.ZeitraumAlle, false
	b.ziel, b.bearbeite, b.worte, b.hinweis = bibZielRaster, "", "", ""
	b.ueber, b.letzteKachel = -1, -1
	b.r.SetzeFokus(-1)
	b.r.SetzeRollen(0)
	b.r.SetzeAktuell(e.offenRel())
	b.fokusAnfang = true
	gtx.Execute(key.FocusCmd{Tag: b})
	b.abgleich = true
	e.bibliothekNeuLesen()
	if e.unten.v != nil {
		e.unten.v.Pruefe() // fremd geänderte Vorschauen nicht weiter zeigen (F-677)
	}
}

// bibliothekSchliessen schließt die Ansicht; der Fokus geht an die Leinwand (ohne gtx im nächsten Bild).
func (e *editor) bibliothekSchliessen(gtx *layout.Context) {
	b := &e.bibl
	if !b.offen {
		return
	}
	b.offen, b.bearbeite, b.hinweis, b.ueber = false, "", "", -1
	b.kacheln, b.von, b.bis = nil, 0, 0
	if e.menue.offen && e.menue.welches == listeZeitraum {
		e.menue.offen, e.menue.fokusNach = false, nil
	}
	if gtx != nil {
		gtx.Execute(key.FocusCmd{Tag: e})
		return
	}
	b.fokusLeinwand = true
}

// bibliothekNachWechsel: Eine Aufnahme aus der Ansicht ist geöffnet; die Ansicht schließt (ADR-0036 Punkt 4).
func (e *editor) bibliothekNachWechsel() {
	if e.bibl.r != nil {
		e.bibl.r.SetzeAktuell(e.offenRel())
	}
	e.bibliothekSchliessen(nil)
}

// bibliothekNachSpeichern: Nach einem eigenen Speichern kann die offene Aufnahme eine neue sein (grüner Rand).
func (e *editor) bibliothekNachSpeichern() {
	if e.bibl.r == nil {
		return
	}
	e.bibl.r.SetzeAktuell(e.offenRel())
	if e.bibl.offen {
		e.bibliothekNeuLesen()
	}
}

// bibliothekNeuLesen liest den Index mit der Anfrage der Ansicht im Hintergrund; läuft schon ein Lesen, wird es
// verworfen und danach neu gelesen.
func (e *editor) bibliothekNeuLesen() {
	b := &e.bibl
	if b.r == nil {
		return
	}
	if b.indexLaeuft {
		b.indexNochmal = true
		return
	}
	b.indexLaeuft = true
	if b.index == nil {
		b.index = make(chan indexErgebnis, 1)
	}
	a := libview.Anfrage(b.suche, b.zeitraum, b.agenten, leisteUhr())
	bib, ch, wecker, abgleich := e.bib.Bibliothek, b.index, e.wecker, b.abgleich
	b.abgleich = false
	go func() {
		if abgleich {
			gleicheAb(bib)
		}
		es, err := bib.Suchen(a)
		ch <- indexErgebnis{es, err}
		if wecker != nil {
			wecker.Anfordern()
		}
	}()
}

// bibliothekAbholen nimmt einen gelesenen Index und das Ende einer Arbeit an (Ereignisschleife); liefert, ob etwas da
// war.
func (e *editor) bibliothekAbholen() bool {
	b := &e.bibl
	da := false
	if b.indexLaeuft {
		select {
		case r := <-b.index:
			da = true
			b.indexLaeuft = false
			switch {
			case b.indexNochmal: // veraltet: die Anfrage hat sich inzwischen geändert
				b.indexNochmal = false
				e.bibliothekNeuLesen()
			case r.err != nil:
				b.indexFehler = true
				diagnoseEditor("Index der Ansicht: %v", r.err)
			default:
				b.indexFehler = false
				b.r.Setze(r.es)
				if b.bearbeite != "" && b.r.Stelle(b.bearbeite) < 0 {
					b.bearbeite = "" // die bearbeitete Aufnahme ist nicht mehr da
				}
				if b.fokusAnfang {
					b.fokusAnfang = false
					i := b.r.Stelle(b.r.Aktuell())
					if i < 0 && len(r.es) > 0 {
						i = 0
					}
					b.r.SetzeFokus(i)
					if b.masse.Hoehe > 0 {
						b.r.Zeige(i, b.masse)
					}
				}
			}
		default:
		}
	}
	if b.arbeitLaeuft {
		select {
		case r := <-b.arbeit:
			da = true
			b.arbeitLaeuft = false
			e.bibliothekArbeitFertig(r)
		default:
		}
	}
	return da
}

// bibliothekArbeitFertig meldet das Ende von Papierkorb bzw. Schlagworten und liest Ansicht und Leiste neu.
func (e *editor) bibliothekArbeitFertig(r bibArbeit) {
	switch {
	case r.papierkorb && r.err != nil:
		e.melde(fmt.Sprintf(i18n.T("Could not move the capture to the trash: %v"), fehlerGrund(r.err)), true)
		e.abgleichAnfordern() // die Datei fehlt womöglich (F-672)
	case r.papierkorb:
		e.melde(i18n.T("Moved to the trash. It can be restored from there."), false)
	case r.err != nil:
		e.melde(fmt.Sprintf(i18n.T("Could not save the keywords: %v"), fehlerGrund(r.err)), true)
	default:
		e.melde(i18n.T("Keywords saved."), false)
	}
	if e.unten.v != nil {
		e.unten.v.Vergiss(r.rel)
	}
	e.bibliothekNeuLesen()
	e.leisteNeuLesen()
}

// bibliothekFrei meldet, ob Öffnen, Papierkorb und Schlagworte gerade gehen: nicht während des Speicherns, eines
// Wechsels oder einer laufenden Arbeit der Ansicht.
func (e *editor) bibliothekFrei() bool {
	return !e.s.SpeichernLaeuft() && !e.wechsel.laedt && !e.bibl.arbeitLaeuft
}

// bibAbs ist der absolute Pfad der Aufnahme an der Stelle i.
func (e *editor) bibAbs(i int) string {
	return filepath.Join(e.bib.Bibliothek.Orte.Ablage, filepath.FromSlash(e.bibl.r.Eintraege()[i].Pfad))
}

// bibIstOffen meldet, ob die Stelle i die offene Aufnahme ist (grüner Rand oder derselbe Ort über einen Verweis).
func (e *editor) bibIstOffen(i int) bool { return e.bibl.r.IstAktuell(i) || e.istOffen(e.bibAbs(i)) }

// bibOeffne öffnet die Aufnahme an der Stelle i im selben Fenster (Enter, Doppelklick; über wechselAnfrage wie die
// Leiste). Die offene Aufnahme schließt nur die Ansicht. Die Ansicht schließt, sobald die neue Aufnahme übernommen ist
// (uebernimm); scheitert das Speichern oder Laden, bleibt sie offen, und die Statuszeile sagt warum.
func (e *editor) bibOeffne(gtx layout.Context, i int) {
	b := &e.bibl
	if i < 0 || i >= len(b.r.Eintraege()) || !e.bibliothekFrei() {
		return
	}
	if e.bibIstOffen(i) {
		e.bibliothekSchliessen(&gtx)
		return
	}
	e.wechselAnfrage(e.bibAbs(i))
}

// bibPapierkorb legt die Aufnahme mit Fokus in den Papierkorb (Entf); die offene ist ausgenommen.
func (e *editor) bibPapierkorb() {
	b := &e.bibl
	i := b.r.Fokus()
	if i < 0 || i >= len(b.r.Eintraege()) || !e.bibliothekFrei() {
		return
	}
	if e.bibIstOffen(i) {
		e.melde(i18n.T("The open capture cannot be moved to the trash. Open another one first."), false)
		return
	}
	rel := b.r.Eintraege()[i].Pfad
	e.bibArbeiten(bibArbeit{papierkorb: true, rel: rel}, func() error {
		_, err := e.bib.Bibliothek.InPapierkorb(rel, leisteUhr())
		return err
	})
}

// bibSchlagworteBeginnen öffnet das Feld unter der Kachel mit Fokus (F2); die offene Aufnahme ist ausgenommen: Ihre
// Datei schreibt der Editor beim Speichern selbst, ein zweiter Schreiber dazwischen machte daraus eine neue Aufnahme.
func (e *editor) bibSchlagworteBeginnen() {
	b := &e.bibl
	i := b.r.Fokus()
	if i < 0 || i >= len(b.r.Eintraege()) || !e.bibliothekFrei() {
		return
	}
	if e.bibIstOffen(i) {
		e.melde(i18n.T("The keywords of the open capture cannot be changed here. Open another one first."), false)
		return
	}
	es := b.r.Eintraege()[i]
	b.ziel, b.bearbeite, b.worte = bibZielRaster, es.Pfad, libview.SchlagwortText(es.Schlagworte)
}

// bibSchlagworteUebernehmen schreibt die Schlagworte des offenen Felds (Enter) im Hintergrund.
func (e *editor) bibSchlagworteUebernehmen() {
	b := &e.bibl
	rel, worte := b.bearbeite, libview.ZerlegeSchlagworte(b.worte)
	if rel == "" {
		return
	}
	if !e.bibliothekFrei() {
		// Speichern, Wechsel oder eine andere Arbeit läuft: das Feld bleibt offen mit der Eingabe, statt sie still zu
		// verwerfen (F-678 Punkt 1); Enter danach übernimmt.
		e.melde(i18n.T("Please wait a moment, then press Enter again."), false)
		return
	}
	b.bearbeite = ""
	e.bibArbeiten(bibArbeit{rel: rel}, func() error {
		_, err := e.bib.Bibliothek.SetzeSchlagworte(rel, worte)
		return err
	})
}

// bibArbeiten führt f in einer Goroutine aus; das Ergebnis holt bibliothekAbholen.
func (e *editor) bibArbeiten(a bibArbeit, f func() error) {
	b := &e.bibl
	if b.arbeit == nil {
		b.arbeit = make(chan bibArbeit, 1)
	}
	b.arbeitLaeuft = true
	ch, wecker := b.arbeit, e.wecker
	go func() {
		a.err = f()
		ch <- a
		if wecker != nil {
			wecker.Anfordern()
		}
	}()
}

// bibZeitraumListe öffnet die Auswahl des Zeitraums über die Liste des Menüs (wie die Felder der Fußzeile); per
// Tastatur steht der Fokus auf dem gewählten Zeitraum.
func (e *editor) bibZeitraumListe(perTaste bool) {
	e.menueOeffne(listeZeitraum, perTaste)
	if !perTaste || !e.menue.offen {
		return
	}
	for j, z := range libview.Zeitraeume {
		if z == e.bibl.zeitraum {
			e.menue.cursor, e.menue.fokusNach = j, &e.menue.eintraege[j]
		}
	}
}

// zeitraumListe sind die Einträge der Auswahl des Zeitraums (menueEintraegeVon).
func (e *editor) zeitraumListe() []menueEintrag {
	var en []menueEintrag
	for _, z := range libview.Zeitraeume {
		en = append(en, menueEintrag{hatZeitraum: true, zeitraum: z, text: z.Text(), an: true, aktiv: z == e.bibl.zeitraum})
	}
	return en
}

// bibSetzeZeitraum übernimmt einen Zeitraum aus der Liste.
func (e *editor) bibSetzeZeitraum(z libview.Zeitraum) {
	if e.bibl.zeitraum != z {
		e.bibl.zeitraum = z
		e.bibliothekNeuLesen()
	}
}

// bibliothekErlaubt meldet, ob die Handlung a bei offener Ansicht wirkt (Menü und Kürzel). Alles, was das Bild oder die
// Leisten darunter ändert, wirkt nicht (ADR-0036 Punkt 4).
func bibliothekErlaubt(a aktion) bool {
	switch a {
	case aktSpeichern, aktEinstellungen, aktSchliessen, aktBibliothek, aktTastenkuerzel, aktUeber,
		aktBibOeffnen, aktBibSchlagworte, aktBibPapierkorb, aktBibSuchen:
		return true
	}
	return false
}

// bibliothekMenue passt die Einträge des Menüs i an die offene Ansicht an: Bearbeiten bedient das Raster, alles
// übrige, was das Bild ändert, ist grau mit Erklärung (ADR-0026 Nachtrag 8).
func (e *editor) bibliothekMenue(i int, en []menueEintrag) []menueEintrag {
	if i == menueBearbeiten {
		return e.bibliothekBearbeiten()
	}
	for j := range en {
		if en[j].hatWerkzeug || !bibliothekErlaubt(en[j].akt) {
			en[j].an = false
			en[j].erklaerung = i18n.T("Not available in the library. Esc returns to the image.")
		}
	}
	return en
}

// bibliothekBearbeiten ist das Menü Bearbeiten bei offener Ansicht: Öffnen, Schlagworte, Papierkorb, Suchen.
func (e *editor) bibliothekBearbeiten() []menueEintrag {
	b := &e.bibl
	i := b.r.Fokus()
	da := i >= 0 && i < len(b.r.Eintraege()) && e.bibliothekFrei()
	offen := da && e.bibIstOffen(i)
	grau := ""
	if offen {
		grau = i18n.T("This is the open capture. Open another one first.")
	}
	return []menueEintrag{
		{akt: aktBibOeffnen, text: i18n.T("Open"), an: da, kuerzel: tastenName(key.Filter{Name: key.NameReturn})},
		{akt: aktBibSchlagworte, text: i18n.T("Edit keywords"), an: da && !offen, erklaerung: grau,
			kuerzel: tastenName(key.Filter{Name: key.NameF2}), gruppe: true},
		{akt: aktBibPapierkorb, text: i18n.T("Move to trash"), an: da && !offen, erklaerung: grau,
			kuerzel: tastenName(key.Filter{Name: key.NameDeleteForward})},
		{akt: aktBibSuchen, text: i18n.T("Search by keyword"), an: true,
			kuerzel: tastenName(key.Filter{Name: "F", Required: key.ModShortcut}), gruppe: true},
	}
}

// bibliothekFilter sind die Tasten der Ansicht. Sie gelten ohne Fokusbindung wie die der Einstellungstafel, damit keine
// in die Leinwand gerät; was hier fehlt, filtert tasten (nur F10 und erlaubte Kürzel).
func (e *editor) bibliothekFilter() []event.Filter {
	f := []event.Filter{key.FocusFilter{Target: &e.bibl}}
	for _, n := range []key.Name{key.NameEscape, key.NameReturn, key.NameEnter, key.NameTab} {
		f = append(f, key.Filter{Name: n, Optional: key.ModShift})
	}
	for _, n := range []key.Name{key.NameUpArrow, key.NameDownArrow, key.NameLeftArrow, key.NameRightArrow,
		key.NameHome, key.NameEnd, key.NamePageUp, key.NamePageDown, key.NameDeleteForward, key.NameDeleteBackward,
		key.NameF2} {
		f = append(f, key.Filter{Name: n})
	}
	return append(f, key.Filter{Name: "F", Required: key.ModShortcut})
}

// bibliothekAuswerten verarbeitet Klicks, Mausrad und Tasten der offenen Ansicht; läuft im Bild vor feldTasten und
// tasten.
func (e *editor) bibliothekAuswerten(gtx layout.Context) {
	b := &e.bibl
	if b.fokusLeinwand {
		b.fokusLeinwand = false
		if !b.offen {
			gtx.Execute(key.FocusCmd{Tag: e})
		}
	}
	if !b.offen {
		return
	}
	e.bibliothekZeiger(gtx)
	if b.sucheKnopf.Clicked(gtx) {
		b.ziel, b.bearbeite = bibZielSuche, ""
	}
	if b.zeitraumKnopf.Clicked(gtx) {
		b.ziel, b.bearbeite = bibZielZeitraum, ""
		e.bibZeitraumListe(false)
	}
	if b.agentenKnopf.Clicked(gtx) {
		b.ziel, b.bearbeite = bibZielAgenten, ""
		b.agenten = !b.agenten
		e.bibliothekNeuLesen()
	}
	if b.zurueckKnopf.Clicked(gtx) {
		e.bibliothekSchliessen(&gtx)
		return
	}
	if e.einst.offen || e.hilfeOffen || e.menue.offen {
		// Tafel, Hilfe oder Menü darüber bedienen ihre Tasten selbst (einstellungenAuswerten, tasten). Das Fokusziel der
		// Ansicht bleibt angemeldet, damit der Fokus nach dem Schließen hierher zurückkann.
		for {
			if _, ok := gtx.Event(key.FocusFilter{Target: b}); !ok {
				return
			}
		}
	}
	if !gtx.Focused(b) {
		gtx.Execute(key.FocusCmd{Tag: b})
	}
	filter := e.bibliothekFilter()
	for {
		ev, ok := gtx.Event(filter...)
		if !ok {
			return
		}
		switch ev := ev.(type) {
		case key.EditEvent:
			e.bibliothekTippe(ev.Text)
		case key.Event:
			if ev.State == key.Press {
				e.bibliothekTaste(gtx, ev)
				if !b.offen || e.menue.offen {
					return // geschlossen bzw. die Liste des Zeitraums bedient tasten
				}
			}
		}
		gtx.Execute(op.InvalidateCmd{})
	}
}

// bibliothekZeiger: Mausrad rollt, ein Klick setzt den Fokus auf die Kachel, ein Doppelklick öffnet sie; der Zeiger
// über einer Kachel zeigt ihren Hinweis.
func (e *editor) bibliothekZeiger(gtx layout.Context) {
	b := &e.bibl
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &b.flaeche, Kinds: pointer.Press | pointer.Scroll | pointer.Move |
			pointer.Enter | pointer.Leave, ScrollY: pointer.ScrollRange{Min: -1 << 20, Max: 1 << 20}})
		if !ok {
			return
		}
		pe, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		stelle := b.r.Treffer(pe.Position.Round().Sub(b.sicht.Min), b.masse)
		switch pe.Kind {
		case pointer.Scroll:
			b.r.Rad(pe.Scroll.Y, b.masse)
		case pointer.Move, pointer.Enter:
			b.ueber = stelle
		case pointer.Leave:
			b.ueber = -1
		case pointer.Press:
			if pe.Buttons != pointer.ButtonPrimary {
				continue
			}
			if b.bearbeite != "" && (stelle < 0 || b.r.Eintraege()[stelle].Pfad != b.bearbeite) {
				b.bearbeite = "" // Klick woanders bricht die Bearbeitung ab
			}
			b.ziel = bibZielRaster
			gtx.Execute(key.FocusCmd{Tag: b})
			if stelle < 0 {
				continue
			}
			b.r.SetzeFokus(stelle)
			// Doppelklick wie auf der Leinwand: zweiter Druck auf dieselbe Kachel innerhalb der Doppelklick-Zeit.
			doppel := stelle == b.letzteKachel && pe.Time-b.letzterKlick <= doppelklickZeit
			b.letzteKachel, b.letzterKlick = stelle, pe.Time
			if doppel {
				b.letzteKachel = -1
				e.bibOeffne(gtx, stelle)
			}
		}
	}
}

// bibliothekTippe nimmt getippten Text an: ins Schlagwortfeld, sonst ins Suchfeld, wenn es den Fokus hat.
func (e *editor) bibliothekTippe(text string) {
	b := &e.bibl
	switch {
	case b.bearbeite != "":
		b.worte = libview.Tippe(b.worte, text)
	case b.ziel == bibZielSuche:
		if neu := libview.Tippe(b.suche, text); neu != b.suche {
			b.suche = neu
			e.bibliothekNeuLesen()
		}
	}
}

// bibliothekTaste behandelt eine gedrückte Taste der Ansicht (ADR-0036 Punkt 4).
func (e *editor) bibliothekTaste(gtx layout.Context, k key.Event) {
	b := &e.bibl
	enter := k.Name == key.NameReturn || k.Name == key.NameEnter
	if b.bearbeite != "" { // Schlagwortfeld offen: Enter übernimmt, Esc bricht ab, sonst wirkt nichts
		switch {
		case enter:
			e.bibSchlagworteUebernehmen()
		case k.Name == key.NameEscape:
			b.bearbeite = ""
		case k.Name == key.NameDeleteBackward:
			b.worte = libview.Zurueck(b.worte)
		}
		return
	}
	switch {
	case k.Name == "F" && k.Modifiers.Contain(key.ModShortcut):
		b.ziel = bibZielSuche
		return
	case k.Name == key.NameTab:
		schritt := 1
		if k.Modifiers.Contain(key.ModShift) {
			schritt = bibZiele - 1
		}
		b.ziel = (b.ziel + schritt) % bibZiele
		if b.ziel == bibZielRaster && b.r.Fokus() < 0 {
			b.r.SetzeFokus(b.r.Bewege(-1, libview.SchrittAnfang, b.masse))
		}
		return
	case k.Name == key.NameEscape:
		e.bibliothekSchliessen(&gtx)
		return
	}
	switch b.ziel {
	case bibZielSuche:
		switch {
		case k.Name == key.NameDeleteBackward:
			if b.suche != "" {
				b.suche = libview.Zurueck(b.suche)
				e.bibliothekNeuLesen()
			}
		case enter, k.Name == key.NameDownArrow:
			b.ziel = bibZielRaster
			if b.r.Fokus() < 0 {
				b.r.SetzeFokus(b.r.Bewege(-1, libview.SchrittAnfang, b.masse))
			}
		}
	case bibZielZeitraum:
		if enter || k.Name == key.NameDownArrow {
			e.bibZeitraumListe(true)
		}
	case bibZielAgenten:
		if enter {
			b.agenten = !b.agenten
			e.bibliothekNeuLesen()
		}
	case bibZielZurueck:
		if enter {
			e.bibliothekSchliessen(&gtx)
		}
	case bibZielRaster:
		e.bibliothekRasterTaste(gtx, k)
	}
}

// bibliothekRasterTaste: Pfeile, Pos1, Ende, Bild↑/↓ wandern; Enter öffnet, Entf legt in den Papierkorb, F2 bearbeitet
// die Schlagworte.
func (e *editor) bibliothekRasterTaste(gtx layout.Context, k key.Event) {
	b := &e.bibl
	var s libview.Schritt
	switch k.Name {
	case key.NameReturn, key.NameEnter:
		e.bibOeffne(gtx, b.r.Fokus())
		return
	case key.NameDeleteForward:
		e.bibPapierkorb()
		return
	case key.NameF2:
		e.bibSchlagworteBeginnen()
		return
	case key.NameLeftArrow:
		s = libview.SchrittLinks
	case key.NameRightArrow:
		s = libview.SchrittRechts
	case key.NameUpArrow:
		s = libview.SchrittHoch
	case key.NameDownArrow:
		s = libview.SchrittRunter
	case key.NameHome:
		s = libview.SchrittAnfang
	case key.NameEnd:
		s = libview.SchrittEnde
	case key.NamePageUp:
		s = libview.SchrittSeiteHoch
	case key.NamePageDown:
		s = libview.SchrittSeiteRunter
	default:
		return
	}
	if ziel := b.r.Bewege(b.r.Fokus(), s, b.masse); ziel >= 0 {
		b.r.SetzeFokus(ziel)
		b.r.Zeige(ziel, b.masse)
	}
}

// --- Zeichnen ---

// bibText misst s (eine Zeile, höchstens breite Pixel) und liefert den gemessenen Text.
func (e *editor) bibText(gtx layout.Context, s string, sp unit.Sp, c color.NRGBA, breite int, fett bool) schreiber {
	return e.tafelText(gtx, s, sp, c, max(1, breite), 1, fett)
}

// ring zeichnet den Tastaturfokus: 2 dp orange außen (corporate-design.md §6).
func bibRing(gtx layout.Context, r image.Rectangle) {
	paint.FillShape(gtx.Ops, farbeAkzent, clip.Stroke{Path: clip.Rect(r.Inset(-gtx.Dp(3))).Path(), Width: float32(gtx.Dp(2))}.Op())
}

// bibKopfHoehe ist die Höhe des Kopfs für die Fensterbreite.
func bibKopfHoehe(gtx layout.Context) int {
	if gtx.Constraints.Max.X >= gtx.Dp(bibKopfEinzeilg) {
		return gtx.Dp(bibKopfDp)
	}
	return gtx.Dp(bibKopfZweiDp)
}

// zeichneBibliothek zeichnet die Ansicht in r (zwischen Menüzeile und Fußzeile, Fensterpixel).
func (e *editor) zeichneBibliothek(gtx layout.Context, r image.Rectangle) {
	b := &e.bibl
	paint.FillShape(gtx.Ops, farbeFenster, clip.Rect(r).Op())
	// Zuunterst das Ziel der Ansicht selbst: Gio gibt den Tastaturfokus nur einem Ziel, das im Bild steht; darüber
	// liegende Knöpfe und das Raster bekommen ihre Zeigerereignisse weiter.
	ganz := clip.Rect(r).Push(gtx.Ops)
	event.Op(gtx.Ops, b)
	ganz.Pop()
	kopf := bibKopfHoehe(gtx)
	e.zeichneBibliothekKopf(gtx, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+kopf))
	paint.FillShape(gtx.Ops, farbeLinie, clip.Rect(image.Rect(r.Min.X, r.Min.Y+kopf-gtx.Dp(1), r.Max.X, r.Min.Y+kopf)).Op())
	sicht := image.Rect(r.Min.X, r.Min.Y+kopf, r.Max.X, max(r.Min.Y+kopf, r.Max.Y))
	b.sicht = sicht
	b.masse = libview.RasterMasse{Rand: gtx.Dp(bibRandDp), Oben: gtx.Dp(bibObenDp), KachelB: gtx.Dp(bibKachelB),
		KachelH: gtx.Dp(bibBildH + bibTextH), LueckeX: gtx.Dp(bibLueckeX), LueckeY: gtx.Dp(bibLueckeY),
		Kopf: gtx.Dp(bibMonatDp), Breite: sicht.Dx(), Hoehe: sicht.Dy()}
	e.zeichneBibliothekRaster(gtx)
	b.hinweis = e.bibliothekHinweis()
}

// zeichneBibliothekKopf zeichnet Titel, Suchfeld, Zeitraum, Schalter und „Back to image“; in schmalen Fenstern in zwei
// Zeilen ohne Titel.
func (e *editor) zeichneBibliothekKopf(gtx layout.Context, r image.Rectangle) {
	b := &e.bibl
	dp := gtx.Dp
	fh := dp(30)
	zweiZeilen := r.Dx() < dp(bibKopfEinzeilg)
	b.kopfZweiZeil = zweiZeilen
	y1 := r.Min.Y + (dp(bibKopfDp)-fh)/2
	y2 := y1
	if zweiZeilen {
		y1 = r.Min.Y + dp(10)
		y2 = y1 + fh + dp(12)
	}
	// Zurück rechts in der ersten Zeile.
	zt := e.bibText(gtx, i18n.T("Back to image"), unit.Sp(14), farbeText, r.Dx()/3, false)
	zb := zt.groesse.X + dp(32)
	zr := image.Rect(r.Max.X-dp(bibRandDp)-zb, y1-dp(2), r.Max.X-dp(bibRandDp), y1+fh+dp(2))
	g := farbeLinie
	if b.zurueckKnopf.Hovered() {
		g = farbeGrund
	}
	paint.FillShape(gtx.Ops, g, clip.Rect(zr).Op())
	zt.setze(gtx, zr.Min.Add(zr.Size().Sub(zt.groesse).Div(2)))
	if b.ziel == bibZielZurueck {
		bibRing(gtx, zr)
	}
	tafelKnopf(gtx, &b.zurueckKnopf, zr)
	b.zurueckRahmen = zr

	x := r.Min.X + dp(bibRandDp)
	if !zweiZeilen {
		ti := e.bibText(gtx, i18n.T("Library"), unit.Sp(17), farbeText, dp(130), true)
		ti.setze(gtx, image.Pt(x, r.Min.Y+(dp(bibKopfDp)-ti.groesse.Y)/2))
		x += dp(146)
	}
	// Suchfeld
	sb := dp(bibSucheDp)
	if zweiZeilen {
		sb = max(dp(80), zr.Min.X-dp(16)-x)
	}
	sr := image.Rect(x, y1, x+sb, y1+fh)
	paint.FillShape(gtx.Ops, farbeGrund, clip.Rect(sr).Op())
	rand := farbeLinie
	if b.ziel == bibZielSuche {
		rand = farbeAkzent
	}
	paint.FillShape(gtx.Ops, rand, clip.Stroke{Path: clip.Rect(sr).Path(), Width: float32(dp(1))}.Op())
	b.gezeigteSuche = b.suche
	if b.suche == "" {
		ph := e.bibText(gtx, i18n.T("Search by keyword …"), unit.Sp(14), farbeNebentext, sr.Dx()-dp(20), false)
		ph.setze(gtx, image.Pt(sr.Min.X+dp(10), sr.Min.Y+(fh-ph.groesse.Y)/2))
		if b.ziel == bibZielSuche {
			e.bibMarke(gtx, image.Pt(sr.Min.X+dp(10), sr.Min.Y+dp(7)), fh-dp(14))
		}
	} else {
		st := e.bibText(gtx, b.suche, unit.Sp(14), farbeText, sr.Dx()-dp(24), false)
		st.setze(gtx, image.Pt(sr.Min.X+dp(10), sr.Min.Y+(fh-st.groesse.Y)/2))
		if b.ziel == bibZielSuche {
			e.bibMarke(gtx, image.Pt(sr.Min.X+dp(11)+st.groesse.X, sr.Min.Y+dp(7)), fh-dp(14))
		}
	}
	tafelKnopf(gtx, &b.sucheKnopf, sr)
	b.suchRahmen = sr
	// Zeitraum
	x2 := sr.Max.X + dp(16)
	if zweiZeilen {
		x2 = r.Min.X + dp(bibRandDp)
	}
	tr := image.Rect(x2, y2, x2+dp(bibZeitraumDp), y2+fh)
	paint.FillShape(gtx.Ops, farbeGrund, clip.Rect(tr).Op())
	paint.FillShape(gtx.Ops, farbeLinie, clip.Stroke{Path: clip.Rect(tr).Path(), Width: float32(dp(1))}.Op())
	zz := e.bibText(gtx, b.zeitraum.Text(), unit.Sp(14), farbeText, tr.Dx()-dp(40), false)
	zz.setze(gtx, image.Pt(tr.Min.X+dp(10), tr.Min.Y+(fh-zz.groesse.Y)/2))
	zeichneAufklappPfeil(gtx, image.Pt(tr.Max.X-dp(15), tr.Min.Y+fh/2), farbeText)
	if b.ziel == bibZielZeitraum {
		bibRing(gtx, tr)
	}
	tafelKnopf(gtx, &b.zeitraumKnopf, tr)
	b.zeitRahmen = tr
	// Schalter
	q := dp(20)
	kx := tr.Max.X + dp(20)
	kr := image.Rect(kx, y2+(fh-q)/2, kx+q, y2+(fh-q)/2+q)
	at := e.bibText(gtx, i18n.T("Show agent captures"), unit.Sp(14), farbeText, max(dp(40), r.Max.X-kr.Max.X-dp(24)), false)
	if !zweiZeilen {
		at = e.bibText(gtx, i18n.T("Show agent captures"), unit.Sp(14), farbeText, max(dp(40), zr.Min.X-kr.Max.X-dp(24)), false)
	}
	zeichneKaestchen(gtx, kr, b.agenten)
	at.setze(gtx, image.Pt(kr.Max.X+dp(10), y2+(fh-at.groesse.Y)/2))
	ar := image.Rect(kr.Min.X, y2, kr.Max.X+dp(10)+at.groesse.X+dp(4), y2+fh)
	if b.ziel == bibZielAgenten {
		bibRing(gtx, kr)
	}
	tafelKnopf(gtx, &b.agentenKnopf, ar)
	b.agentenRahmen = ar
}

// bibMarke zeichnet die Einfügemarke eines Eingabefelds (2 dp breit) ab p, h hoch.
func (e *editor) bibMarke(gtx layout.Context, p image.Point, h int) {
	paint.FillShape(gtx.Ops, farbeText, clip.Rect(image.Rect(p.X, p.Y, p.X+gtx.Dp(2), p.Y+h)).Op())
}

// zeichneBibliothekRaster zeichnet Monatsüberschriften, Kacheln und den Rollbalken im Raster b.sicht.
func (e *editor) zeichneBibliothekRaster(gtx layout.Context) {
	b := &e.bibl
	dp := gtx.Dp
	sicht, m := b.sicht, b.masse
	b.kacheln, b.von, b.bis, b.feldRahmen, b.gezeigteWorte = nil, 0, 0, image.Rectangle{}, ""
	// Das Zeigerziel steht im Ausschnitt des Rasters vor den Kacheln (wie die Leiste, Lehre aus F-598).
	c := clip.Rect(sicht).Push(gtx.Ops)
	event.Op(gtx.Ops, &b.flaeche)
	pointer.CursorDefault.Add(gtx.Ops)
	es := b.r.Eintraege()
	b.leerText = ""
	if len(es) == 0 {
		switch {
		case b.indexLaeuft:
		case b.indexFehler:
			b.leerText = i18n.T("The library index could not be read.")
		case b.suche != "" || b.zeitraum != libview.ZeitraumAlle:
			b.leerText = i18n.T("No captures match the search.")
		default:
			b.leerText = i18n.T("No captures in the library yet.")
		}
		if b.leerText != "" {
			t := e.bibText(gtx, b.leerText, unit.Sp(14), farbeNebentext, sicht.Dx()-2*m.Rand, false)
			t.setze(gtx, image.Pt(sicht.Min.X+m.Rand, sicht.Min.Y+m.Oben+dp(8)))
		}
		c.Pop()
		b.ops = nil
		return
	}
	gv, gb := b.r.SichtbareGruppen(m)
	for g := gv; g < gb; g++ {
		gr := b.r.Gruppen()[g]
		t := e.bibText(gtx, libview.MonatTitel(gr), unit.Sp(12), farbeNebentext, sicht.Dx()-2*m.Rand, true)
		t.setze(gtx, image.Pt(sicht.Min.X+m.Rand, sicht.Min.Y+b.r.KopfLage(g, m)+dp(6)))
	}
	b.von, b.bis = b.r.Sichtbar(m)
	jetzt := leisteUhr()
	neueOps := map[string]leisteBild{}
	if e.unten.v != nil {
		e.unten.v.Runde() // Aufträge weggerollter Vorschauen verfallen (F-669)
	}
	fokus := b.r.Fokus()
	for i := b.von; i < b.bis; i++ {
		kr := b.r.Lage(i, m).Add(sicht.Min)
		br := image.Rect(kr.Min.X, kr.Min.Y, kr.Max.X, kr.Min.Y+dp(bibBildH))
		b.kacheln = append(b.kacheln, br)
		eintrag := es[i]
		zeichneVorschau(gtx, e.unten.v, b.ops, eintrag.Pfad, br, neueOps)
		aktuell := b.r.IstAktuell(i)
		switch {
		case aktuell:
			paint.FillShape(gtx.Ops, farbeAuswahl, clip.Stroke{Path: clip.Rect(br.Inset(dp(1))).Path(),
				Width: float32(dp(2))}.Op())
		case i == b.ueber:
			paint.FillShape(gtx.Ops, farbeNebentext, clip.Stroke{Path: clip.Rect(br).Path(), Width: float32(dp(1))}.Op())
		}
		if i == fokus && b.ziel == bibZielRaster {
			bibRing(gtx, br)
		}
		z := e.bibText(gtx, libview.KachelZeit(eintrag.Erstellt, jetzt), unit.Sp(12), farbeText, kr.Dx(), false)
		z.setze(gtx, image.Pt(kr.Min.X, br.Max.Y+dp(6)))
		zeile2 := image.Pt(kr.Min.X, br.Max.Y+dp(24))
		if b.bearbeite == eintrag.Pfad {
			fr := image.Rect(kr.Min.X, br.Max.Y+dp(21), kr.Max.X, br.Max.Y+dp(43))
			paint.FillShape(gtx.Ops, farbeGrund, clip.Rect(fr).Op())
			paint.FillShape(gtx.Ops, farbeAkzent, clip.Stroke{Path: clip.Rect(fr).Path(), Width: float32(dp(1))}.Op())
			w := e.bibText(gtx, b.worte, unit.Sp(12), farbeText, fr.Dx()-dp(12), false)
			w.setze(gtx, image.Pt(fr.Min.X+dp(5), fr.Min.Y+(fr.Dy()-w.groesse.Y)/2))
			e.bibMarke(gtx, image.Pt(min(fr.Max.X-dp(4), fr.Min.X+dp(6)+w.groesse.X), fr.Min.Y+dp(4)), fr.Dy()-dp(8))
			b.feldRahmen, b.gezeigteWorte = fr, b.worte
		} else {
			a := e.bibText(gtx, libview.KachelAngaben(eintrag), unit.Sp(12), farbeNebentext, kr.Dx(), false)
			a.setze(gtx, zeile2)
		}
	}
	c.Pop()
	b.ops = neueOps
	// Rollbalken rechts, eckig wie die rechte Leiste (seitenrollen.go).
	if rmax := b.r.RollMax(m); rmax > 0 {
		oben, laenge := rollDaumen(sicht.Dy(), sicht.Dy(), sicht.Dy()+rmax, b.r.Rollen(m))
		rr := image.Rect(sicht.Max.X-dp(6), sicht.Min.Y+oben, sicht.Max.X-dp(2), sicht.Min.Y+oben+laenge)
		paint.FillShape(gtx.Ops, farbeNebentext, clip.Rect(rr).Op())
	}
}

// bibliothekHinweis ist der Text der Statuszeile: zur Kachel unter dem Zeiger, sonst zur Kachel mit Fokus, sonst die
// Bedienung des Felds.
func (e *editor) bibliothekHinweis() string {
	b := &e.bibl
	es := b.r.Eintraege()
	switch {
	case b.bearbeite != "":
		return i18n.T("Keywords, separated by commas. Enter saves, Esc cancels.")
	case b.ziel == bibZielSuche:
		return i18n.T("Type a keyword to show only captures with it. Down or Enter goes to the captures.")
	}
	i := b.ueber
	if i < 0 && b.ziel == bibZielRaster {
		i = b.r.Fokus()
	}
	if i < 0 || i >= len(es) {
		return ""
	}
	zeit := es[i].Erstellt.Local().Format(i18n.TK("date format", "2006-01-02 15:04"))
	if b.r.IstAktuell(i) {
		return fmt.Sprintf(i18n.T("Capture of %s, %d × %d px — open now. Enter or Esc returns to it."), zeit, es[i].Breite, es[i].Hoehe)
	}
	return fmt.Sprintf(i18n.T("Capture of %s, %d × %d px — Enter or double-click opens it, Del moves it to the trash, F2 edits keywords"),
		zeit, es[i].Breite, es[i].Hoehe)
}

// --- Knopf „Library“ in der Leiste der letzten Aufnahmen ---

// leisteKnopfB ist die Breite des Knopfs „Library“ links in der Leiste (Muster 174 a); die Vorschauen beginnen dahinter.
const (
	leisteKnopfB    = 100
	leisteKnopfRand = 10
)

// zeichneLeisteKnopf zeichnet den Knopf „Library“ in r (Fensterpixel): ein Stapel Rahmen und der Name darunter.
func (e *editor) zeichneLeisteKnopf(gtx layout.Context, r image.Rectangle) {
	k := &e.unten
	g := farbeLeisteVoll
	if k.bibKnopf.Hovered() {
		g = farbeGrund
	}
	paint.FillShape(gtx.Ops, farbeLinie, clip.Rect(r).Op())
	paint.FillShape(gtx.Ops, g, clip.Rect(r.Inset(gtx.Dp(1))).Op())
	// Symbol: drei versetzte Rahmen, eckig aus Flächen (Gios Strich kennt nur runde Ecken).
	s := gtx.Dp(2)
	w, h := gtx.Dp(28), gtx.Dp(18)
	m := image.Pt((r.Min.X+r.Max.X)/2, r.Min.Y+gtx.Dp(30))
	for j := 2; j >= 0; j-- {
		o := image.Pt(m.X-w/2+j*gtx.Dp(4)-gtx.Dp(4), m.Y-h/2-j*gtx.Dp(4)+gtx.Dp(4))
		f := image.Rectangle{Min: o, Max: o.Add(image.Pt(w, h))}
		paint.FillShape(gtx.Ops, g, clip.Rect(f).Op())
		for _, kante := range []image.Rectangle{
			image.Rect(f.Min.X, f.Min.Y, f.Max.X, f.Min.Y+s), image.Rect(f.Min.X, f.Max.Y-s, f.Max.X, f.Max.Y),
			image.Rect(f.Min.X, f.Min.Y, f.Min.X+s, f.Max.Y), image.Rect(f.Max.X-s, f.Min.Y, f.Max.X, f.Max.Y),
		} {
			paint.FillShape(gtx.Ops, farbeText, clip.Rect(kante).Op())
		}
	}
	t := e.bibText(gtx, i18n.T("Library"), unit.Sp(12), farbeText, r.Dx()-gtx.Dp(8), false)
	t.setze(gtx, image.Pt((r.Min.X+r.Max.X-t.groesse.X)/2, r.Max.Y-gtx.Dp(10)-t.groesse.Y))
	seiteKnopf(gtx, &k.bibKnopf, r)
	if k.bibKnopf.Hovered() || gtx.Focused(&k.bibKnopf) {
		k.hinweis = fmt.Sprintf(i18n.T("Open the library (%s)"), kuerzelText(aktBibliothek))
	}
}

// zeichneVorschau zeichnet das Vorschaubild von rel aus v eingepasst in r, bis es geladen ist ein graues Feld
// (ADR-0036 Punkt 6). alt sind die Bild-Ops vom letzten Bild (Gio lädt die Textur sonst je Bild neu), neu sammelt die
// dieses Bilds.
func zeichneVorschau(gtx layout.Context, v *libview.Vorschauen, alt map[string]leisteBild, rel string, r image.Rectangle,
	neu map[string]leisteBild) {
	paint.FillShape(gtx.Ops, farbeGrund, clip.Rect(r).Op())
	bild, z := v.Bild(rel)
	if z != libview.Da || bild == nil || bild.Rect.Dx() == 0 || bild.Rect.Dy() == 0 {
		paint.FillShape(gtx.Ops, farbeLinie, clip.Rect(r).Op()) // Platzhalter (ADR-0036 Punkt 6)
		return
	}
	lb, ok := alt[rel]
	if !ok || lb.bild != bild {
		lb = leisteBild{bild: bild, op: paint.NewImageOp(bild)}
	}
	neu[rel] = lb
	bw, bh := float32(bild.Rect.Dx()), float32(bild.Rect.Dy())
	s := min(float32(r.Dx())/bw, float32(r.Dy())/bh)
	w, h := bw*s, bh*s
	ecke := f32.Pt(float32(r.Min.X)+(float32(r.Dx())-w)/2, float32(r.Min.Y)+(float32(r.Dy())-h)/2)
	cl := clip.Rect(r).Push(gtx.Ops)
	t := op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(s, s)).Offset(ecke)).Push(gtx.Ops)
	lb.op.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	t.Pop()
	cl.Pop()
}
