// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"fmt"
	"image"
	"image/color"
	"slices"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/libview"
)

// menueEintrag ist eine Zeile des Menüs (ADR-0026 Punkt 2).
type menueEintrag struct {
	akt    aktion
	text   string
	an     bool
	gruppe bool // beginnt eine neue Gruppe (Trennlinie davor)
	// erklaerung sagt bei einem grauen Eintrag, warum er nicht geht (etwa „Install tesseract …“, ADR-0026 Nachtrag 8):
	// Sie steht in der Statuszeile, solange der Eintrag überfahren oder per Tastatur angesteuert ist. Ein grauer Eintrag
	// mit Erklärung ist deshalb per Tastatur erreichbar; auslösen lässt er sich nicht.
	erklaerung string
	// Menü „Werkzeuge“ (ADR-0031 Punkt 1): wählt ein Werkzeug statt einer Handlung; aktiv markiert das gewählte.
	werkzeug    canvas.Werkzeug
	hatWerkzeug bool
	aktiv       bool
	// Aufklapplisten der Fußzeile (ADR-0031 Nachtrag 3): eine Farbe (hatFarbe) bzw. eine Zoomstufe in Prozent
	// (hatZoom; 0 heißt Einpassen) statt einer Handlung.
	farbe    color.NRGBA
	hatFarbe bool
	zoom     int
	hatZoom  bool
	// Auswahlliste der Einstellungstafel (ADR-0034): Stelle der Option in tafel.optionen.
	wahl    int
	hatWahl bool
	// Auswahl des Zeitraums in der vollen Ansicht der Bibliothek (ADR-0036 Punkt 3).
	zeitraum    libview.Zeitraum
	hatZeitraum bool
	// Spitzenart eines Pfeils (ADR-0021 Nachtrag 20): für das Ende bei A (kopfAnfang) bzw. bei B; die Zeile zeigt davor
	// eine Vorschau.
	kopf       document.Kopfart
	hatKopf    bool
	kopfAnfang bool
	// Strichart (ADR-0021 Nachtrag 25): die Zeile zeigt davor eine Vorschau.
	strichart    document.Strichart
	hatStrichart bool
	// kuerzel zeigt ein Kürzel, das nicht in der Tastentabelle steht, weil ein eigener Bereich es abholt (Menü
	// Bearbeiten der vollen Ansicht, ADR-0026 Nachtrag 10).
	kuerzel string
}

// menueEintraege liefert die Einträge des offenen Menüs (ADR-0031 Punkt 1) mit dem Zustand der Sitzung.
func (e *editor) menueEintraege() []menueEintrag { return e.menueEintraegeVon(e.menue.welches) }

// kantenMeldung ist die Statusmeldung nach einem Wechsel der Kantenform (Menüeintrag „Seam style“). Jede Form hat ihren
// eigenen Text als Literal, damit die Übersetzung ihn findet (ADR-0014 Nachtrag 1).
func kantenMeldung(k document.Kante) string {
	switch k {
	case document.KanteWelle:
		return i18n.T("Seam style: wave")
	case document.KanteZacken:
		return i18n.T("Seam style: zigzag")
	}
	return i18n.T("Seam style: torn edge")
}

// schliessenEintrag ist der Knopf des Menüeintrags „Schließen“ (nil, wenn es ihn nicht gibt).
func (e *editor) schliessenEintrag() *widget.Clickable {
	for i, en := range e.menueEintraege() {
		if en.akt == aktSchliessen {
			return &e.menue.eintraege[i]
		}
	}
	return nil
}

// menueKuerzel ist das angezeigte Kürzel eines Eintrags; „Text bearbeiten“ geht mit Enter, ein Werkzeug mit seinem
// Buchstaben.
func menueKuerzel(en menueEintrag) string {
	if en.kuerzel != "" {
		return en.kuerzel
	}
	if en.hatWerkzeug {
		return werkzeugKuerzel(en.werkzeug)
	}
	if en.hatFarbe {
		return fmt.Sprintf("#%02X%02X%02X", en.farbe.R, en.farbe.G, en.farbe.B)
	}
	if en.hatZoom {
		switch en.zoom {
		case 0:
			return kuerzelText(aktEinpassen)
		case 100:
			return kuerzelText(aktEins)
		}
		return ""
	}
	if en.akt == aktTextBearbeiten {
		return kuerzelText(aktEnter)
	}
	return kuerzelText(en.akt)
}

// menue ist der Zustand des Menüs.
type menue struct {
	// titel sind die Menüs der Zeile (ADR-0031 Punkt 1), titelRahmen ihre Lage beim letzten Zeichnen; welches ist
	// das offene bzw. zuletzt offene Menü.
	titel       [menueAnzahl]widget.Clickable
	titelRahmen [menueAnzahl]image.Rectangle
	welches     int
	offen       bool
	eintraege   [24]widget.Clickable // je Eintrag einer; mehr als genug für menueEintraege (Ebene: 20)
	aussen      int                  // Ziel für Klicks neben die offene Liste (schließt sie)
	zeile       int                  // Ziel für Klicks in die Menüzeile abseits der Titel (schließt die Liste, F-515)

	// Tastatur (F-255): fokusNach ist der Eintrag, der beim Zeichnen den Fokus bekommt (die Klickfläche gibt es erst
	// dort); cursor der zuletzt per Taste angesteuerte Eintrag (-1: keiner).
	fokusNach *widget.Clickable
	cursor    int
	// perTasteOffen: Die Liste wurde per Tastatur geöffnet (Esc gibt den Fokus dann an den Menütitel zurück, sonst an
	// die Leinwand).
	perTasteOffen bool
	// gewechselt: In diesem Bild hat Pfeil links/rechts das Menü gewechselt. Der Fokus liegt dann noch auf der Klickfläche
	// des alten Menüs, die gleiche Nummer im neuen Menü wäre ein anderer Eintrag; Enter im selben Bild löst deshalb nichts
	// aus (F-512). `tasten` setzt es zu Beginn jedes Bildes zurück.
	gewechselt bool

	// rahmen: Lage der Zeilen im Fenster beim letzten Zeichnen (Tests und Drehbuch-Koordinaten, F-265).
	rahmen [24]image.Rectangle
}

// menueNaechster liefert den Index des nächsten bedienbaren Eintrags von von aus in Richtung dir (+1 abwärts, -1
// aufwärts), die Liste läuft an den Enden um (wie die Menüs in KDE/Qt); graue Einträge werden übersprungen, außer sie
// tragen eine Erklärung (ADR-0026 Nachtrag 8). von = -1:
// keiner angesteuert, dann der erste (abwärts) bzw. letzte (aufwärts). -1, wenn nichts bedienbar ist.
func menueNaechster(en []menueEintrag, von, dir int) int {
	n := len(en)
	if n == 0 {
		return -1
	}
	start := von
	if von < 0 {
		start = -1
		if dir < 0 {
			start = n
		}
	}
	for k := 1; k <= n; k++ {
		i := ((start+dir*k)%n + n) % n
		if en[i].an || en[i].erklaerung != "" {
			return i
		}
	}
	return -1
}

// menueErster und menueLetzter: der erste bzw. letzte bedienbare Eintrag (-1: keiner).
func menueErster(en []menueEintrag) int  { return menueNaechster(en, -1, 1) }
func menueLetzter(en []menueEintrag) int { return menueNaechster(en, -1, -1) }

// menueFokusIndex ist der Eintrag mit Tastaturfokus (-1: keiner).
func (e *editor) menueFokusIndex(gtx layout.Context) int {
	for i := range e.menueEintraege() {
		if gtx.Focused(&e.menue.eintraege[i]) {
			return i
		}
	}
	return -1
}

// menueTaste verarbeitet Pfeile, Pos1 und Ende bei offenem Menü (F-255, ADR-0026 Nachtrag 2): Der Fokus wandert zum
// nächsten bedienbaren Eintrag; die Wirkung folgt beim Zeichnen (menueFokusSetzen).
func (e *editor) menueTaste(gtx layout.Context, name key.Name) {
	m := &e.menue
	en := e.menueEintraege()
	cur := m.cursor
	if m.fokusNach == nil {
		if i := e.menueFokusIndex(gtx); i >= 0 {
			cur = i
		}
	}
	neu := -1
	switch name {
	case key.NameDownArrow:
		neu = menueNaechster(en, cur, 1)
	case key.NameUpArrow:
		neu = menueNaechster(en, cur, -1)
	case key.NameHome:
		neu = menueErster(en)
	case key.NameEnd:
		neu = menueLetzter(en)
	}
	if neu >= 0 {
		m.cursor, m.fokusNach = neu, &m.eintraege[neu]
	}
}

// menueKnopf meldet, ob c zur offenen Liste gehört: ein Eintrag, ein Menütitel oder der Knopf, der die Liste öffnet
// (F-873).
func (e *editor) menueKnopf(c *widget.Clickable) bool {
	m := &e.menue
	if c == nil {
		return false
	}
	for i := range m.eintraege {
		if c == &m.eintraege[i] {
			return true
		}
	}
	for i := range m.titel {
		if c == &m.titel[i] {
			return true
		}
	}
	return m.welches != listeKontext && c == e.listenKnopf(m.welches)
}

// menueSchliesse schließt die Liste und gibt den Fokus zurück: per Esc an den Menütitel, nach einer Handlung oder
// einem Klick daneben an die Leinwand (F-255; sonst fiele der Fokus auf nil, getippter Text ginge verloren).
func (e *editor) menueSchliesse(gtx layout.Context, zuKnopf bool) {
	m := &e.menue
	m.offen, m.fokusNach = false, nil
	if e.einst.offen { // Auswahlliste der Einstellungstafel: Fokus zurück an die Tafel (ADR-0034)
		gtx.Execute(key.FocusCmd{Tag: &e.einst})
		return
	}
	if e.bibl.offen { // volle Ansicht der Bibliothek: Fokus zurück an die Ansicht (ADR-0036 Punkt 4)
		gtx.Execute(key.FocusCmd{Tag: &e.bibl})
		return
	}
	if m.welches == listeKontext && e.kontext.inLeiste {
		// Kontextmenü einer Vorschau der Leiste: Fokus zurück an die Vorschau (ADR-0038 Punkt 3), sobald sie gezeichnet
		// ist; ist sie weg, an die Leinwand.
		if i := e.vorschauStelle(); i >= 0 {
			e.unten.fokusSoll = i
			gtx.Execute(op.InvalidateCmd{})
			return
		}
	}
	if zuKnopf && m.welches != listeKontext { // das Kontextmenü hat keinen Knopf: Fokus an die Leinwand (ADR-0038)
		gtx.Execute(key.FocusCmd{Tag: e.listenKnopf(m.welches)})
	} else {
		gtx.Execute(key.FocusCmd{Tag: e})
	}
}

// fuehreAus führt eine Handlung aus, gleich ob per Taste, Knopf oder Menü. vonMaus: Ein laufender Text wird vorher
// bestätigt (wie die Knöpfe); per Taste wirken Kopieren und Export in der Texteingabe nicht (wie bisher).
func (e *editor) fuehreAus(gtx layout.Context, a aktion, vonMaus bool) {
	if vonMaus {
		e.s.TextBestaetigen()
	}
	if w := e.wirkung(a); w != nil {
		w(gtx)
	}
}

// aktionAusfuehrbar meldet, ob fuehreAus für a etwas tut (Test: kein Menüeintrag ohne Wirkung, F-253).
func aktionAusfuehrbar(a aktion) bool { return (&editor{}).wirkung(a) != nil }

// wirkung ist die Handlung zu a; nil für aktKeine und die Tasten mit eigener Behandlung (aktEsc, aktEnter).
func (e *editor) wirkung(a aktion) func(gtx layout.Context) {
	switch a {
	case aktKopieren:
		return func(gtx layout.Context) {
			if e.s.KopierenErlaubt() {
				e.kopierenOderObjekte(gtx) // mit Auswahl die Objekte (ADR-0045 Punkt 1)
			}
		}
	case aktBildKopieren:
		return func(gtx layout.Context) {
			if e.s.KopierenErlaubt() {
				e.kopieren(gtx)
			}
		}
	case aktAusschneiden:
		return func(layout.Context) { e.objekteKopieren(true) }
	case aktEinfuegen:
		return func(layout.Context) { e.einfuegenStarten() }
	case aktNeuAusZwischenablage:
		return func(layout.Context) { e.neuAusZwischenablage() }
	case aktExport:
		return func(layout.Context) {
			if e.s.KopierenErlaubt() {
				e.exportStarten()
			}
		}
	case aktSpeichern:
		return func(layout.Context) { e.speichernJetzt() }
	case aktOeffnen:
		return func(layout.Context) { e.oeffnenStarten() }
	case aktSpeichernUnter:
		return func(layout.Context) { e.speichernUnterStarten() }
	case aktNeueAufnahme:
		return func(layout.Context) { e.neueAufnahme() }
	case aktRueckgaengig:
		return func(layout.Context) { e.rueckgaengig() }
	case aktWiederholen:
		return func(layout.Context) { e.wiederholen() }
	case aktTextBearbeiten:
		return func(layout.Context) { e.s.TextBearbeiten() }
	case aktLoeschen:
		return func(layout.Context) { e.s.Loeschen() }
	case aktFarbenTauschen:
		return func(layout.Context) { e.s.TauscheFarben() }
	case aktGanzesBild:
		// Zuschnitt aufheben und Randstreifen zurückholen, wie Doppelklick mit Zuschneiden (E-21, F-222).
		return func(layout.Context) {
			if e.s.HebeZuschnittAuf() {
				e.bildNeu()
				e.meldeAnsicht()
			}
		}
	case aktZurueckholen:
		return func(layout.Context) {
			if e.s.HoleGestauchtesZurueck() {
				e.bildNeu()
				e.meldeAnsicht()
			}
		}
	case aktKantenform:
		// Kantenform reihum, ein Rückgängig-Schritt (ADR-0023 Nachtrag 6); die Fußzeile nennt die neue Form. Die
		// Kanten zeichnet zeichneNaehte neu, ihr Zwischenspeicher ist nach der Form geschlüsselt.
		return func(layout.Context) {
			if k, ok := e.s.WechsleKante(); ok {
				e.melde(kantenMeldung(k), false)
			}
		}
	case aktTastenkuerzel:
		return func(layout.Context) { e.hilfeOffen = !e.hilfeOffen }
	case aktEinstellungen:
		return func(gtx layout.Context) { e.einstellungenOeffnen(gtx) }
	case aktTextKopieren:
		return func(layout.Context) { e.ocrBeginnen() }
	case aktUeber:
		return func(layout.Context) {
			e.melde(fmt.Sprintf(i18n.T("Schnapp %s — screenshots and annotations for Linux. Free software under the GPL 3.0 or later."),
				Version), false)
		}
	case aktEinpassen:
		return func(layout.Context) { e.s.Einpassen() }
	case aktEins:
		return func(layout.Context) { e.s.EinsZuEins() }
	case aktSeitenleiste:
		// Beim Einklappen geht der Fokus an die Leinwand: Ein Knopf der Leiste, der ihn hielt, ist dann weg, und Enter
		// träfe sonst die Texteingabe (Sitzungs-QA A-5).
		return func(gtx layout.Context) {
			e.seiteZu = !e.seiteZu
			if e.seiteZu {
				gtx.Execute(key.FocusCmd{Tag: e})
			}
		}
	case aktEinrasten:
		return func(layout.Context) { e.s.SchalteEinrasten() }
	case aktNavigator:
		return func(layout.Context) { e.s.SchalteNavigator() }
	case aktLeiste:
		return func(gtx layout.Context) { e.schalteLeiste(gtx) }
	case aktBibliothek:
		return func(gtx layout.Context) {
			if e.bibl.offen {
				e.bibliothekSchliessen(&gtx)
			} else {
				e.bibliothekOeffnen(gtx)
			}
		}
	case aktBibOeffnen:
		return func(gtx layout.Context) {
			if e.bibl.offen {
				e.bibOeffne(gtx, e.bibl.r.Fokus())
			}
		}
	case aktBibSchlagworte:
		return func(layout.Context) {
			if e.bibl.offen {
				e.bibSchlagworteBeginnen()
			}
		}
	case aktBibPapierkorb:
		return func(layout.Context) {
			if e.bibl.offen {
				e.bibPapierkorb()
			}
		}
	case aktBibSuchen:
		return func(layout.Context) {
			if e.bibl.offen {
				e.bibl.ziel, e.bibl.bearbeite = bibZielSuche, ""
			}
		}
	case aktDoppeln:
		return func(layout.Context) { e.s.Doppeln() }
	case aktAllesWaehlen:
		return func(layout.Context) { e.s.WaehleAlle() }
	case aktGruppieren:
		return func(layout.Context) { e.s.Gruppieren() }
	case aktGruppierungAufheben:
		return func(layout.Context) { e.s.GruppierungAufheben() }
	case aktAusrichtenLinks, aktAusrichtenMitteWaagerecht, aktAusrichtenRechts, aktAusrichtenOben,
		aktAusrichtenMitteSenkrecht, aktAusrichtenUnten:
		ausr := canvas.Ausrichtungen[a-aktAusrichtenLinks]
		return func(layout.Context) {
			if !e.s.Ausrichten(ausr) {
				e.meldeAbgelehnt()
			}
		}
	case aktVerteilenWaagerecht, aktVerteilenSenkrecht:
		waagerecht := a == aktVerteilenWaagerecht
		return func(layout.Context) {
			if !e.s.Verteilen(waagerecht) {
				e.meldeAbgelehnt()
			}
		}
	case aktSchnellstilSichern:
		return e.schnellstilSichernAktion
	case aktVorschauOeffnen, aktVorschauZeigen, aktVorschauSchlagworte, aktVorschauPapierkorb:
		return e.vorschauWirkung(a)
	case aktUmbenennen:
		return func(gtx layout.Context) { e.umbenennenBeginnen(gtx) }
	case aktAusblenden:
		return func(layout.Context) { e.schalteAusgeblendetGruppe(e.s.Auswahl) }
	case aktSperren:
		return func(layout.Context) { e.s.SchalteGesperrtGruppe(e.s.Auswahl) } // mit der gewählten Gruppe (ADR-0044)
	case aktEbeneVor:
		return func(layout.Context) { e.s.EbeneSchritt(1) }
	case aktEbeneZurueck:
		return func(layout.Context) { e.s.EbeneSchritt(-1) }
	case aktEbeneDarueber:
		return func(gtx layout.Context) { e.s.WaehleNachbar(1); e.ebenen.zeigeGewaehlte = true }
	case aktEbeneDarunter:
		return func(gtx layout.Context) { e.s.WaehleNachbar(-1); e.ebenen.zeigeGewaehlte = true }
	case aktSchliessen:
		return func(layout.Context) {
			diagnoseEditor("Schließen über Taste oder Menü")
			// In der eigenen Schleife, wie im Overlay (F-77): Die Anfrage kommt als ClosingEvent zurück und geht durch
			// schliessen; läuft ein Speichern, merkt SchliessenAnfrage sie vor (F-299).
			if e.perform != nil {
				e.perform(system.ActionClose)
				return
			}
			e.w.Perform(system.ActionClose)
		}
	}
	return nil
}

// menueOeffne öffnet das Menü i oder schließt es, wenn es schon offen ist. Per Tastatur geöffnet: Fokus auf den
// ersten bedienbaren Eintrag (F-255). Per Maus bleibt der Fokus unberührt, damit kein Ring ohne Tastatur erscheint. Die
// Art kommt vom Aufrufer, nicht aus dem Klick (F-255 V-2); Enter, Leertaste und F10 öffnen sofort statt erst im
// nächsten Bild (F-255 V-3).
func (e *editor) menueOeffne(i int, perTaste bool) {
	m := &e.menue
	if e.bibl.offen && (i == listeFarbe || i == listeZoom) {
		// Die volle Ansicht der Bibliothek sperrt Farbe und Zoom der Fußzeile auf jedem Weg, auch per F10, Tab und
		// Enter (ADR-0036 Teil 2 Punkt 5, ADR-0026 Nachtrag 10, F-674).
		return
	}
	if m.offen && m.welches == i {
		m.offen = false
		return
	}
	m.welches, m.offen, m.cursor = i, true, -1
	m.perTasteOffen = perTaste
	if perTaste {
		if j := menueErster(e.menueEintraege()); j >= 0 {
			m.cursor, m.fokusNach = j, &m.eintraege[j]
		}
	}
}

// menueMit ist das Menü, das die Handlung a führt (-1: keines). Tests und Drehbücher wählen so das richtige Menü.
func (e *editor) menueMit(a aktion) int {
	for i := 0; i < menueAnzahl; i++ {
		for _, en := range e.menueEintraegeVon(i) {
			if !en.hatWerkzeug && en.akt == a {
				return i
			}
		}
	}
	return -1
}

// menueWechsel öffnet das Nachbarmenü (Pfeil links/rechts, ADR-0031 Punkt 1); läuft an den Enden um. Eine
// Aufklappliste der Fußzeile hat keine Nachbarn: Dort wirken die Pfeile nicht.
func (e *editor) menueWechsel(dir int) {
	m := &e.menue
	if istListe(m.welches) {
		return
	}
	perTaste := m.perTasteOffen
	m.gewechselt = true
	m.offen = false
	e.menueOeffne(((m.welches+dir)%menueAnzahl+menueAnzahl)%menueAnzahl, perTaste)
}

// menueTitelFokus ist der Index des Menütitels mit Tastaturfokus (-1: keiner).
func (e *editor) menueTitelFokus(gtx layout.Context) int {
	for i := range e.menue.titel {
		if gtx.Focused(&e.menue.titel[i]) {
			return i
		}
	}
	return -1
}

// knopfPerTaste löst einen per Tastatur fokussierten Knopf aus (Enter): einen Menütitel und ein Feld der Fußzeile mit
// Liste über menueOeffne (Fokus in die Liste), die übrigen als Klick.
func (e *editor) knopfPerTaste(c *widget.Clickable) {
	for i := range e.menue.titel {
		if c == &e.menue.titel[i] {
			e.menueOeffne(i, true)
			return
		}
	}
	switch c {
	case &e.knoepfe.fussFarbe:
		e.listeAmFarbKnopf = false
		e.menueOeffne(listeFarbe, true)
		return
	case &e.knoepfe.fussZoom:
		e.menueOeffne(listeZoom, true)
		return
	case &e.pfeilform.feld[0]:
		e.kopfListeOeffnen(0, true)
		return
	case &e.pfeilform.feld[1]:
		e.kopfListeOeffnen(1, true)
		return
	case &e.strichart.feld:
		e.strichartListeOeffnen(true)
		return
	case &e.knoepfe.farbe, &e.knoepfe.fuellung:
		// Per Tastatur öffnet ein Farbfeld der Werkzeugleiste immer die Liste für sein Feld; die rechte Leiste klappt
		// Strg+Umschalt+B auf. Das graue Feld Füllung tut nichts (ADR-0021 Nachtrag 11).
		f := canvas.FeldVordergrund
		if c == &e.knoepfe.fuellung {
			f = canvas.FeldFuellung
		}
		if !e.s.SetzeFeld(f) {
			return
		}
		e.listeAmFarbKnopf = true
		e.menueOeffne(listeFarbe, true)
		return
	}
	c.Click()
}

// listenKnopf ist der Knopf, der das Menü bzw. die Liste i öffnet: der Menütitel oder das Feld der Fußzeile.
func (e *editor) listenKnopf(i int) *widget.Clickable {
	switch i {
	case listeFarbe:
		if e.listeAmFarbKnopf {
			if e.s.AktivesFeld() == canvas.FeldFuellung {
				return &e.knoepfe.fuellung
			}
			return &e.knoepfe.farbe
		}
		return &e.knoepfe.fussFarbe
	case listeZoom:
		return &e.knoepfe.fussZoom
	case listeEinstellung:
		return &e.einst.feldKnopf[max(0, e.einst.listeZeile)]
	case listeZeitraum:
		return &e.bibl.zeitraumKnopf
	case listeKopfAnfang, listeKopfEnde:
		return &e.pfeilform.feld[i-listeKopfAnfang]
	case listeStrichart:
		return &e.strichart.feld
	}
	return &e.menue.titel[i]
}

// menueAuswerten verarbeitet Klicks auf den Menütitel, die Einträge und daneben.
func (e *editor) menueAuswerten(gtx layout.Context) {
	m := &e.menue
	if m.offen && !m.perTasteOffen {
		// Per Maus geöffnet: Hält ein Knopf hinter der Liste den Fokus, den Gio selbst per Leertaste auslöst (nicht in
		// leertastenZiele, etwa ein Werkzeug), bekommt die Leinwand ihn (F-883, F-257).
		if c := e.fokussierterKnopf(gtx); c != nil && !e.menueKnopf(c) && !slices.Contains(e.leertastenZiele(), event.Tag(c)) {
			gtx.Execute(key.FocusCmd{Tag: e})
		}
	}
	// Während des Speicherns ruht das Bearbeiten (ADR-0020 Nachtrag 6); je Klick neu gefragt, denn ein Eintrag im selben
	// Bild kann das Speichern starten (F-280).
	an := func() bool { return !e.s.SpeichernLaeuft() }
	for i := range m.titel {
		if m.titel[i].Clicked(gtx) && an() {
			if m.offen && m.welches == i { // zweiter Klick auf den Titel: schließen, Fokus an die Leinwand
				e.menueSchliesse(gtx, false)
				continue
			}
			e.menueOeffne(i, false) // ein Klick ist die Maus; Enter, Leertaste und F10 laufen in tasten
		}
	}
	// Ist ein Menü offen, wechselt das Überfahren eines anderen Titels die Liste (wie in Qt, ADR-0031 Punkt 1).
	if m.offen && !m.perTasteOffen && !istListe(m.welches) {
		for i := range m.titel {
			if i != m.welches && m.titel[i].Hovered() {
				m.welches, m.cursor = i, -1
			}
		}
	}
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &m.aussen, Kinds: pointer.Press})
		if !ok {
			break
		}
		if p, ok := ev.(pointer.Event); ok && p.Kind == pointer.Press && m.offen {
			e.menueSchliesse(gtx, false)
			// Ein zweiter Rechtsklick auf die Leinwand öffnet das Kontextmenü am neuen Punkt für das neue Ziel, wie in
			// Qt/KDE (Kritik K-6 #313a); sonst bräuchte es drei Klicks.
			q := image.Pt(int(p.Position.X), int(p.Position.Y))
			if m.welches == listeKontext && p.Buttons == pointer.ButtonSecondary {
				e.kontextErneut(gtx, q)
			}
		}
	}
	for { // Klick in die Menüzeile abseits der Titel („s“, leere Fläche, Dateiname): wie ein Klick daneben
		ev, ok := gtx.Event(pointer.Filter{Target: &m.zeile, Kinds: pointer.Press})
		if !ok {
			break
		}
		if p, ok := ev.(pointer.Event); ok && p.Kind == pointer.Press && m.offen {
			e.menueSchliesse(gtx, false)
		}
	}
	for { // Klicks auf die Fläche zwischen den Einträgen: abholen und nichts tun
		if _, ok := gtx.Event(pointer.Filter{Target: m, Kinds: pointer.Press}); !ok {
			break
		}
	}
	for i, en := range e.menueEintraege() {
		// Schließen wirkt auch während des Speicherns wie der Fensterknopf: Die Anfrage wird vorgemerkt (F-299).
		if m.eintraege[i].Clicked(gtx) && m.offen && en.an && (an() || en.akt == aktSchliessen) {
			e.menueSchliesse(gtx, false) // Fokus an die Leinwand: getippter Text nach „Text bearbeiten“ kommt an
			switch {
			case en.hatWerkzeug:
				e.s.SetzeWerkzeug(en.werkzeug)
				continue
			case en.hatFarbe:
				e.s.SetzeFarbe(en.farbe)
				continue
			case en.hatZoom && en.zoom == 0:
				e.s.Einpassen()
				continue
			case en.hatZoom:
				e.s.SetzeZoom(en.zoom)
				continue
			case en.hatWahl:
				e.einst.waehle(e.einst.listeZeile, en.wahl)
				continue
			case en.hatZeitraum:
				e.bibSetzeZeitraum(en.zeitraum)
				continue
			case en.hatKopf:
				e.s.SetzeKopf(en.kopfAnfang, en.kopf)
				continue
			case en.hatStrichart:
				e.s.SetzeStrichart(en.strichart)
				continue
			}
			e.fuehreAus(gtx, en.akt, true)
		}
	}
}

// zeichneMenue legt die offene Liste unter den Titel des offenen Menüs (ADR-0031 Punkt 1). Darunter liegt eine Fläche
// über dem ganzen Fenster außer der Menüzeile, die einen Klick daneben abfängt; die Titel der Zeile bleiben bedienbar.
func (e *editor) zeichneMenue(gtx layout.Context) {
	m := &e.menue
	if !m.offen {
		return
	}
	mx := gtx.Constraints.Max
	y := gtx.Dp(menuezeileDp)
	anker, liste := image.Rectangle{}, istListe(m.welches)
	if liste {
		// Aufklappliste der Fußzeile: Die Fläche für Klicks daneben deckt das ganze Fenster, auch das Feld selbst (ein
		// zweiter Klick darauf schließt die Liste).
		anker = e.listenAnker(m.welches)
		y = 0
	}
	ha := clip.Rect(image.Rect(0, y, mx.X, mx.Y)).Push(gtx.Ops)
	event.Op(gtx.Ops, &m.aussen)
	ha.Pop()

	eintraege := e.menueEintraege()
	zeile, innen, trenn := gtx.Dp(32), gtx.Dp(14), gtx.Dp(9)
	// Breite: längster Text plus längstes Kürzel, gemessen ohne Zeichnen.
	mess := func(s string) int {
		if s == "" {
			return 0
		}
		r := op.Record(gtx.Ops)
		g := gtx
		g.Constraints.Min = image.Point{}
		d := material.Label(e.thema, unit.Sp(14), s).Layout(g)
		r.Stop()
		return d.Size.X
	}
	textB, kuerzelB, hoehe := 0, 0, gtx.Dp(6)*2
	for _, en := range eintraege {
		textB = max(textB, mess(en.text))
		kuerzelB = max(kuerzelB, mess(menueKuerzel(en)))
		hoehe += zeile
		if en.gruppe {
			hoehe += trenn
		}
	}
	breite := innen + textB + gtx.Dp(32) + kuerzelB + innen
	farbFeld := 0
	if liste && m.welches == listeFarbe {
		farbFeld = gtx.Dp(14 + 10) // Farbquadrat vor dem Namen
		breite += farbFeld
	}
	if liste && (m.welches == listeKopfAnfang || m.welches == listeKopfEnde || m.welches == listeStrichart) {
		farbFeld = gtx.Dp(44 + 12) // Vorschau der Spitze bzw. Strichart vor dem Namen
		breite += farbFeld
	}
	if liste {
		// Unter dem Anker, wenn es passt (Farbfeld der Werkzeugleiste), sonst darüber (Felder der Fußzeile).
		y = anker.Max.Y
		if y+hoehe > mx.Y {
			y = anker.Min.Y - hoehe
		}
		y = max(0, y)
	}
	// Passt die Liste nicht unter den Knopf, rückt sie nach oben (über die Leiste); ist das Fenster niedriger als die
	// Liste, fallen die Abstände der Gruppen weg und die Zeilen werden niedriger, bis zur Höhe einer Textzeile
	// (F-252: bei der Mindesthöhe 320 dp wären sonst die unteren Einträge nicht erreichbar).
	if y+hoehe > mx.Y {
		y = max(0, mx.Y-hoehe)
	}
	if hoehe > mx.Y {
		trenn = 0
		zeile = menueZeilenhoehe(mx.Y, gtx.Dp(6), gtx.Dp(20), gtx.Dp(32), len(eintraege))
		hoehe = gtx.Dp(6)*2 + zeile*len(eintraege)
		y = 0
	}
	var links int
	if liste {
		links = anker.Min.X
	} else {
		links = m.titelRahmen[m.welches].Min.X
	}
	ecke := image.Pt(max(0, min(links, mx.X-breite)), y)
	t := op.Offset(ecke).Push(gtx.Ops)
	flaeche := image.Rectangle{Max: image.Pt(breite, hoehe)}
	paint.FillShape(gtx.Ops, farbeLeisteVoll, clip.Rect(flaeche).Op()) // eckig (Karte #186)
	paint.FillShape(gtx.Ops, farbeLinie, clip.Stroke{Path: clip.Rect(flaeche).Path(), Width: float32(gtx.Dp(1))}.Op())
	// Die Fläche selbst fängt Klicks zwischen den Einträgen ab, damit sie nicht als „daneben“ gelten.
	hf := clip.Rect(flaeche).Push(gtx.Ops)
	event.Op(gtx.Ops, m)
	hf.Pop()
	oy := gtx.Dp(6)
	for i, en := range eintraege {
		if en.gruppe && i > 0 {
			ly := oy + trenn/2
			paint.FillShape(gtx.Ops, farbeLinie, clip.Rect(image.Rect(innen/2, ly, breite-innen/2, ly+gtx.Dp(1))).Op())
		}
		if en.gruppe {
			oy += trenn
		}
		m.rahmen[i] = image.Rectangle{Min: ecke.Add(image.Pt(0, oy)), Max: ecke.Add(image.Pt(breite, oy+zeile))}
		e.menueZeile(gtx, &m.eintraege[i], en, image.Pt(0, oy), image.Pt(breite, zeile), innen, farbFeld)
		oy += zeile
	}
	t.Pop()
	if m.fokusNach != nil { // die Klickflächen sind jetzt angemeldet: Fokus in die Liste (F-255)
		gtx.Execute(key.FocusCmd{Tag: m.fokusNach})
		gtx.Execute(op.InvalidateCmd{})
		m.fokusNach = nil
	}
}

// menueZeilenhoehe ist die Zeilenhöhe, wenn das Fenster niedriger ist als die Liste: der Platz nach Abzug des
// Randes, geteilt durch die Zeilen, nie unter kleinst und nie über groesst (F-262: sonst würden die Zeilen bei
// Fenstern knapp unter der Listenhöhe höher statt niedriger). Alles in Pixeln.
func menueZeilenhoehe(fensterHoehe, rand, kleinst, groesst, zeilen int) int {
	return min(groesst, max(kleinst, (fensterHoehe-rand*2)/zeilen))
}

// menueZeile zeichnet einen Eintrag: Text links, Kürzel rechts, grau wenn nicht bedienbar.
//
// farbFeld > 0: Vor dem Text steht ein Farbquadrat der Breite farbFeld (Farbliste der Fußzeile).
func (e *editor) menueZeile(gtx layout.Context, c *widget.Clickable, en menueEintrag, ecke, groesse image.Point, innen, farbFeld int) {
	t := op.Offset(ecke).Push(gtx.Ops)
	fokus := (en.an || en.erklaerung != "") && gtx.Focused(c) // graue mit Erklärung: Ring ohne Hintergrund
	if en.an && (c.Hovered() || fokus) {
		paint.FillShape(gtx.Ops, farbeGrund, clip.Rect(image.Rectangle{Max: groesse}).Op())
	}
	if fokus { // Tastaturfokus immer sichtbar: 2 px orangener Ring (corporate-design.md §6). Die Zeilen stoßen aneinander,
		// außen ist kein Platz: Der Ring liegt innen mit 2 px Abstand zum Zeilenrand (ADR-0026 Nachtrag 3).
		r := image.Rectangle{Max: groesse}.Inset(gtx.Dp(3))
		paint.FillShape(gtx.Ops, farbeAkzent, clip.Stroke{Path: clip.Rect(r).Path(), Width: float32(gtx.Dp(2))}.Op())
	}
	farbe, neben := farbeText, farbeNebentext
	switch {
	case !en.an:
		farbe = farbeNebentext
	case en.aktiv:
		farbe = farbeGruenText
	}
	g := gtx
	g.Constraints.Min = image.Point{}
	textX := innen // Anfang des Texts; hinter dem Farbquadrat weiter rechts
	schreibe := func(s string, c color.NRGBA, rechtsbuendig bool) {
		r := op.Record(gtx.Ops)
		lbl := material.Label(e.thema, unit.Sp(14), s)
		lbl.Color = c
		d := lbl.Layout(g)
		call := r.Stop()
		x := textX
		if rechtsbuendig {
			x = groesse.X - innen - d.Size.X
		}
		tt := op.Offset(image.Pt(x, (groesse.Y-d.Size.Y)/2)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		tt.Pop()
	}
	if farbFeld > 0 && en.hatFarbe {
		q := gtx.Dp(14)
		fr := image.Rect(innen, (groesse.Y-q)/2, innen+q, (groesse.Y-q)/2+q)
		paint.FillShape(gtx.Ops, en.farbe, clip.Rect(fr).Op())
		paint.FillShape(gtx.Ops, farbeText, clip.Stroke{Path: clip.Rect(fr).Path(), Width: float32(gtx.Dp(1))}.Op())
		textX += farbFeld
	}
	if farbFeld > 0 && en.hatKopf {
		h := gtx.Dp(16)
		vr := image.Rect(innen, (groesse.Y-h)/2, innen+gtx.Dp(44), (groesse.Y-h)/2+h)
		zeichneKopfMuster(gtx, en.kopf, en.kopfAnfang, vr, farbe)
		textX += farbFeld
	}
	if farbFeld > 0 && en.hatStrichart {
		h := gtx.Dp(16)
		zeichneStrichMuster(gtx, en.strichart, image.Rect(innen, (groesse.Y-h)/2, innen+gtx.Dp(44), (groesse.Y-h)/2+h), farbe)
		textX += farbFeld
	}
	schreibe(en.text, farbe, false)
	if k := menueKuerzel(en); k != "" {
		schreibe(k, neben, true)
	}
	if en.an || en.erklaerung != "" {
		// Ein grauer Eintrag mit Erklärung meldet sich an, damit Überfahren und Fokus die Erklärung zeigen
		// (menueErklaerung); auslösen kann er nichts (menueAuswerten verlangt en.an), der Zeiger bleibt der Pfeil.
		g.Constraints = layout.Exact(groesse)
		c.Layout(g, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: groesse} })
		if en.an {
			handZeiger(gtx, groesse)
		}
	}
	t.Pop()
}

// menueErklaerung ist die Erklärung des grauen Eintrags im offenen Menü, der gerade überfahren oder per Tastatur
// angesteuert ist (leer: keiner). Die Fußzeile zeigt sie statt der Meldung (ADR-0026 Nachtrag 8).
func (e *editor) menueErklaerung(gtx layout.Context) string {
	if !e.menue.offen {
		return ""
	}
	for i, en := range e.menueEintraege() {
		if en.an || en.erklaerung == "" || i >= len(e.menue.eintraege) {
			continue
		}
		if c := &e.menue.eintraege[i]; c.Hovered() || gtx.Focused(c) {
			return en.erklaerung
		}
	}
	return ""
}

// hinweisKnoepfe sind die Werkzeugknöpfe, über denen ein Hinweis steht: bei Zeiger und bei Tastaturfokus; ist ein
// Knopf fokussiert, nur dieser, damit nicht zwei Hinweise zugleich stehen (F-255).
func (e *editor) hinweisKnoepfe(gtx layout.Context) []werkzeugKnopf {
	var fokus, hover []werkzeugKnopf
	for _, w := range e.werkzeugKnoepfe() {
		if gtx.Focused(w.c) {
			fokus = append(fokus, w)
		} else if w.c.Hovered() {
			hover = append(hover, w)
		}
	}
	if len(fokus) > 0 {
		return fokus
	}
	return hover
}
