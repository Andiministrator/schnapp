// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Öffnen einer anderen Aufnahme im selben Fenster (ADR-0036 Punkt 5): Ein Klick oder Enter auf eine Vorschau der Leiste
// speichert zuerst die offene Aufnahme wie Strg+S im Hintergrund. Scheitert das, bleibt sie offen und die Statuszeile
// sagt es; bringt es einen Hinweis (etwa „als neue Aufnahme gespeichert“), bleibt sie für diesen einen Hinweis offen wie
// beim Schließen, ein weiterer Klick wechselt. Danach lädt die Ladefunktion von cmd/schnapp die Datei in einer
// Goroutine wie `schnapp DATEI` (Budget, Rettungskopie, Kantenvorgabe), und der Editor übernimmt sie mit neuer Sitzung
// und neuem Verlauf; Fenster, Leisten und die Schalter der Ansicht bleiben.

import (
	"errors"
	"fmt"
	"time"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
)

// Geoeffnet ist ein Dokument samt allem, was der Editor zum Speichern braucht: beim Start und nach jedem Wechsel.
type Geoeffnet struct {
	Titel     string             // Dateiname für die Titelleiste
	Dokument  *document.Dokument // das geladene Dokument
	Speichern Speichern          // Speicherfunktion (führt Zustand: je Dokument eine eigene)
	Rettung   Rettung            // Rettungskopie (ADR-0029); nil: keine
	// Pfad liefert den absoluten Pfad der Datei, die gerade den Stand trägt (nach einem Speichern als neue Aufnahme die
	// neue); nil: unbekannt. Darf aus jeder Goroutine gerufen werden.
	Pfad func() string
	// OhneGrenze: mit --ignore-limit geöffnet; der Export prüft das Budget dann nicht (ADR-0033 Nachtrag 4). Ein Wechsel
	// öffnet immer mit Budget.
	OhneGrenze bool
}

// Oeffnen lädt die Datei pfad für den Wechsel (cmd/schnapp: ladeFuerEditor, Rettungskopie, Kantenvorgabe). Läuft in
// einer Goroutine; der Fehlertext ist übersetzt und erscheint in der Statuszeile.
type Oeffnen func(pfad string) (Geoeffnet, error)

// Anbindung verbindet den Editor mit der Bibliothek: die Leiste der letzten Aufnahmen (ADR-0036) liest ihren Index und
// ihre Vorschaubilder, Oeffnen lädt eine gewählte Aufnahme. nil: keine Leiste (Tests, die das Fenster prüfen).
type Anbindung struct {
	Bibliothek *library.Bibliothek
	Oeffnen    Oeffnen
}

// wechselZustand ist ein laufender Wechsel.
type wechselZustand struct {
	ziel    string // absoluter Pfad der gewählten Aufnahme; leer: kein Wechsel angefragt
	laedt   bool   // die Ladefunktion läuft
	antwort chan wechselErgebnis
}

type wechselErgebnis struct {
	g   Geoeffnet
	err error
}

// wechselAnfrage nimmt einen Klick oder Enter auf die Vorschau mit dem absoluten Pfad abs an.
func (e *editor) wechselAnfrage(abs string) {
	switch {
	case e.bib == nil || e.bib.Oeffnen == nil || abs == "":
		return
	case e.wechsel.laedt || e.s.SpeichernLaeuft() || e.erzwungen || e.s.SignalVorgemerkt():
		return // ein Wechsel oder ein Speichern läuft schon; das Fenster schließt
	case e.exportOffen:
		e.melde(i18n.T("Finish the export first."), false)
		return
	case e.ocr.laeuft:
		e.melde(i18n.T("Text recognition is running. Press Esc to cancel it first."), false)
		return
	}
	if e.istOffen(abs) {
		return // die offene Aufnahme
	}
	e.wechsel.ziel = abs
	abzug, err := e.s.SpeichernJetzt()
	switch {
	case errors.Is(err, canvas.ErrNichtsZuSpeichern):
		e.wechselLaden()
	case err != nil:
		e.wechsel.ziel = "" // Ziehen läuft: der Klick gilt nicht
	default:
		e.speichernStarten(abzug)
	}
}

// wechselNachSpeichern läuft nach der Antwort des Speicherns (speichernFertig): Wartet ein Wechsel, lädt er nach einem
// gelungenen Speichern ohne Hinweis die gewählte Aufnahme; sonst bleibt die offene, und die Statuszeile sagt warum.
// Schließt das Fenster (Signal, Schließen-Anfrage), entfällt der Wechsel.
func (e *editor) wechselNachSpeichern(r speicherErgebnis, ende canvas.Ende) {
	if e.wechsel.ziel == "" || e.wechsel.laedt {
		return
	}
	switch {
	case ende.Schliessen || ende.Erneut != nil || ende.Schluss != canvas.SchlussZu || e.s.SignalVorgemerkt():
		e.wechsel.ziel = ""
	case r.err != nil:
		e.wechsel.ziel = ""
		e.melde(fmt.Sprintf(i18n.T("Could not save: %v. The other capture was not opened."), ohnePaketpraefix(r.err.Error())), true)
	case r.g.HaeltOffen():
		e.wechsel.ziel = "" // der Hinweis bleibt stehen; ein weiterer Klick wechselt (gespeichert ist nichts mehr offen)
	default:
		e.wechselLaden()
	}
}

// wechselLaden startet die Ladefunktion in einer Goroutine.
//
// Solange ruht das Bearbeiten wie beim Speichern (F-652, F-653): Ein Export-Dialog, eine Texterkennung, eine Texteingabe
// oder ein Zug, die jetzt begännen, gehörten zur alten Aufnahme und gingen mit dem Wechsel verloren oder träfen die neue.
// Den Export und die Texterkennung hält exportStarten bzw. die Sitzung (BeginneTextBereich) an.
func (e *editor) wechselLaden() {
	e.wechsel.laedt = true
	e.s.SetzeRuhe(true)
	e.melde(i18n.T("Opening…"), false)
	ch := make(chan wechselErgebnis, 1)
	e.wechsel.antwort = ch
	oeffnen, ziel, wecker := e.bib.Oeffnen, e.wechsel.ziel, e.wecker
	go func() {
		g, err := oeffnen(ziel)
		ch <- wechselErgebnis{g, err}
		if wecker != nil {
			wecker.Anfordern()
		}
	}()
}

// wechselAbholen nimmt das Ergebnis der Ladefunktion an (Ereignisschleife) und liefert, ob eins da war.
func (e *editor) wechselAbholen() bool {
	if !e.wechsel.laedt {
		return false
	}
	var r wechselErgebnis
	select {
	case r = <-e.wechsel.antwort:
	default:
		return false
	}
	e.wechsel = wechselZustand{}
	e.s.SetzeRuhe(false)
	switch {
	case r.err != nil:
		// Die Ladefunktion übersetzt bekannte Fehler; was sie nicht kennt, verliert hier Paketpräfix und Pfad (F-663).
		e.melde(fmt.Sprintf(i18n.T("Could not open the capture: %v"), fehlerGrund(r.err)), true)
	case e.s.Geaendert() || e.s.SpeichernLaeuft() || e.erzwungen || e.s.SignalVorgemerkt() ||
		e.exportOffen || e.s.TextLaeuft() || e.s.Zieht():
		// Während des Ladens bearbeitet, gespeichert oder zum Schließen freigegeben, oder ein Export-Dialog, eine
		// Texteingabe oder ein Zug ist offen (an der Ruhe vorbei, F-652, F-653): die offene Aufnahme nicht verlassen.
		// Die geladene gibt ihre Rettungskopie frei.
		if r.g.Rettung != nil {
			r.g.Rettung.Ende()
		}
		switch {
		case e.erzwungen || e.s.SignalVorgemerkt():
		case e.exportOffen:
			e.melde(i18n.T("The other capture was not opened: the export is still open."), false)
		default:
			e.melde(i18n.T("The other capture was not opened: this one was changed meanwhile."), false)
		}
	default:
		e.uebernimm(r.g)
	}
	return true
}

// uebernimm zeigt das Dokument g im Fenster statt des bisherigen (gespeichert, unverändert): neue Sitzung, neuer
// Verlauf, Rettungskopie des neuen; die Rettungskopie des alten endet. Fenster, Leisten, Zoom-Voreinstellung (eingepasst)
// und die Schalter der Ansicht (Einrasten, Navigator) bleiben.
func (e *editor) uebernimm(g Geoeffnet) {
	e.ocrAbbrechen()
	e.rettungEnde() // die alte Aufnahme ist gespeichert: Kopie verwerfen, Sperre freigeben
	einrasten, navigator := e.s.EinrastenAn(), e.s.NavigatorAn()
	e.s = canvas.NeueSitzung(g.Dokument)
	if einrasten != e.s.EinrastenAn() {
		e.s.SchalteEinrasten()
	}
	if navigator != e.s.NavigatorAn() {
		e.s.SchalteNavigator()
	}
	e.titel, e.speichern, e.rettung, e.pfad = g.Titel, g.Speichern, g.Rettung, g.Pfad
	e.ohneGrenze = g.OhneGrenze
	exportOhneGrenze.Store(g.OhneGrenze)
	// Zustand, der am Dokument hängt.
	e.meldung, e.meldungHaelt = canvas.Meldung{}, keinHalt
	e.schliessenNachSpeichern = false
	e.lupe.bilder = nil
	e.naht.gueltig = false
	e.ebenen.umbenennen, e.ebenen.umbenennenFokus, e.ebenen.nameHatteFokus = 0, false, false
	e.hatDruck, e.letzterDruck = false, 0
	e.nav = navigatorZustand{}
	e.rett = rettungZustand{}
	e.stileLaden() // eigene Schnellstile gehören der Sitzung (ADR-0037)
	e.rettungStart()
	e.bildStand = e.s.BildAenderung()
	e.bildNeu()
	e.leisteNachWechsel()
	e.bibliothekNachWechsel() // aus der vollen Ansicht geöffnet: sie schließt (ADR-0036 Punkt 4)
	diagnoseEditor("Wechsel zu %s", g.Titel)
	if e.w != nil {
		e.w.Invalidate()
	}
}

// wechselFrist: So lange wartet das Zerstören des Fensters höchstens auf eine laufende Ladefunktion, damit ihre
// Rettungskopie die Sperre freigibt (sonst bliebe sie bis zum Ende des Prozesses).
var wechselFrist = 5 * time.Second

// wechselAbbrechen räumt beim Zerstören einen laufenden Wechsel ab.
func (e *editor) wechselAbbrechen() {
	if !e.wechsel.laedt {
		return
	}
	select {
	case r := <-e.wechsel.antwort:
		if r.g.Rettung != nil {
			r.g.Rettung.Ende()
		}
	case <-time.After(wechselFrist):
	}
	e.wechsel = wechselZustand{}
	e.s.SetzeRuhe(false)
}

// istOffen meldet, ob abs die offene Aufnahme ist, auch wenn einer der beiden Pfade über einen Verweis führt
// (Bilderordner als Symlink, „Öffnen mit“; F-657): wie der grüne Rand (leisteAktuell) über library.Relativ.
func (e *editor) istOffen(abs string) bool {
	if e.pfad == nil {
		return false
	}
	offen := e.pfad()
	switch {
	case offen == "":
		return false
	case abs == offen:
		return true
	case e.bib == nil || e.bib.Bibliothek == nil:
		return false
	}
	a, okA := e.bib.Bibliothek.Relativ(abs)
	o, okO := e.bib.Bibliothek.Relativ(offen)
	return okA && okO && a == o
}
