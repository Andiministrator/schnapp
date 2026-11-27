// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Kontextmenü an den Vorschauen der Bibliothek (ADR-0038 Teil 2, Punkte 3 bis 6): Ein Rechtsklick auf eine Vorschau der
// Leiste unten oder der vollen Ansicht setzt den Fokus auf sie und öffnet die Liste listeKontext am Klickpunkt;
// Umschalt+F10 öffnet sie für die Vorschau mit Fokus (Anker: Mitte der Vorschau). Die Einträge gelten der Vorschau, an
// der die Liste aufging (kontextMenue.vorschau, Pfad relativ zur Ablage): Liest der Index im Hintergrund neu, sucht
// vorschauStelle sie beim Auslösen wieder; ist sie weg, tut der Eintrag nichts.

import (
	"context"
	"errors"
	"image"
	"path/filepath"
	"time"

	"gioui.org/io/key"
	"gioui.org/layout"

	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/portal"
)

// dateimanagerZeigen zeigt eine Datei im Dateimanager; Naht für Tests.
var dateimanagerZeigen = portal.ZeigeImDateimanager

// dateimanagerFrist begrenzt den Busaufruf: Startet der Dateimanager per D-Bus-Aktivierung, antwortet er erst danach.
const dateimanagerFrist = 10 * time.Second

// dateimanagerZustand: höchstens ein Aufruf zugleich; die Antwort holt nachEreignis ab (dateimanagerAbholen).
type dateimanagerZustand struct {
	laeuft  bool
	antwort chan error
}

// vorschauKontext öffnet das Kontextmenü für die Vorschau rel am Fensterpunkt p. inLeiste: Vorschau der Leiste unten,
// sonst der vollen Ansicht. Eine Texteingabe wird bestätigt, Pipette und Bereichsmodus der Texterkennung enden wie beim
// Rechtsklick auf die Leinwand (F-882, F-878, F-862).
func (e *editor) vorschauKontext(rel string, inLeiste bool, p image.Point, perTaste bool) {
	if rel == "" {
		return
	}
	e.s.TextBestaetigen() // wie der Rechtsklick auf die Leinwand: sonst liefe die Eingabe ohne Tastatur weiter (F-882)
	if e.s.PipetteLaeuft() {
		e.s.SchaltePipette()
	}
	e.s.BeendeTextBereich()
	e.kontextOeffnen(kontextMenue{anker: image.Rectangle{Min: p, Max: p.Add(image.Pt(1, 1))}, vorschau: rel,
		inLeiste: inLeiste}, perTaste)
}

// leisteKontext öffnet das Kontextmenü für die Stelle i der Leiste am Fensterpunkt p und gibt ihr den Fokus.
func (e *editor) leisteKontext(i int, p image.Point, perTaste bool) {
	k := &e.unten
	if k.l == nil || i < 0 || i >= len(k.gezeigt) || !e.leisteAn() {
		return
	}
	k.l.SetzeFokus(i)
	k.fokusSoll, k.langDruck = i, -1
	e.vorschauKontext(k.gezeigt[i], true, p, perTaste)
}

// bibKontext öffnet das Kontextmenü für die Stelle i der vollen Ansicht am Fensterpunkt p und gibt ihr den Fokus.
func (e *editor) bibKontext(gtx layout.Context, i int, p image.Point, perTaste bool) {
	b := &e.bibl
	if !b.offen || i < 0 || i >= len(b.r.Eintraege()) || !e.bibliothekFrei() {
		return
	}
	if b.bearbeite != "" && b.r.Eintraege()[i].Pfad == b.bearbeite {
		return // an der Kachel im Schlagwortfeld kein Menü, wie bei Umschalt+F10 (F-876)
	}
	if b.bearbeite != "" {
		b.bearbeite = "" // wie ein Linksklick woanders: bricht die Bearbeitung der Schlagworte ab
	}
	b.r.SetzeFokus(i)
	b.ziel, b.langDruck, b.letzteKachel = bibZielRaster, -1, -1
	gtx.Execute(key.FocusCmd{Tag: b})
	e.vorschauKontext(b.r.Eintraege()[i].Pfad, false, p, perTaste)
}

// leisteStelleBei ist die Stelle der Leiste, deren Vorschau beim letzten Zeichnen am Fensterpunkt p lag (-1: keine).
func (e *editor) leisteStelleBei(p image.Point) int {
	k := &e.unten
	if k.l == nil || !p.In(k.rahmen) {
		return -1
	}
	for j, r := range k.kacheln {
		if p.In(r) {
			return k.von + j
		}
	}
	return -1
}

// bibStelleBei ist die Stelle der vollen Ansicht unter dem Fensterpunkt p (-1: keine).
func (e *editor) bibStelleBei(p image.Point) int {
	b := &e.bibl
	if !b.offen || b.r == nil || !p.In(b.sicht) {
		return -1
	}
	return b.r.Treffer(p.Sub(b.sicht.Min), b.masse)
}

// vorschauKontextPerTaste öffnet per Umschalt+F10 das Kontextmenü der Vorschau mit Fokus, in der Mitte ihrer Kachel;
// liefert, ob eine Vorschau den Fokus hatte.
func (e *editor) vorschauKontextPerTaste(gtx layout.Context) bool {
	if e.bibl.offen {
		b := &e.bibl
		i := b.r.Fokus()
		if b.ziel != bibZielRaster || b.bearbeite != "" || i < b.von || i >= b.bis || i-b.von >= len(b.kacheln) {
			return false
		}
		e.bibKontext(gtx, i, mitteVon(b.kacheln[i-b.von]), true)
		return true
	}
	k := &e.unten
	for _, c := range k.alle() {
		if i := k.stelle(c); i >= 0 && gtx.Focused(c) {
			if j := i - k.von; j >= 0 && j < len(k.kacheln) {
				e.leisteKontext(i, mitteVon(k.kacheln[j]), true)
				return true
			}
		}
	}
	return false
}

// mitteVon ist die Mitte von r.
func mitteVon(r image.Rectangle) image.Point { return r.Min.Add(r.Size().Div(2)) }

// vorschauStelle ist die Stelle der Vorschau, der das Kontextmenü gilt, in ihrer Liste, wie sie jetzt steht (-1: nicht
// mehr da).
func (e *editor) vorschauStelle() int {
	rel := e.kontext.vorschau
	if rel == "" {
		return -1
	}
	if e.kontext.inLeiste {
		for i, p := range e.unten.gezeigt {
			if p == rel {
				return i
			}
		}
		return -1
	}
	if !e.bibl.offen || e.bibl.r == nil {
		return -1
	}
	for i, es := range e.bibl.r.Eintraege() {
		if es.Pfad == rel {
			return i
		}
	}
	return -1
}

// vorschauAbs ist der absolute Pfad der Vorschau des Kontextmenüs.
func (e *editor) vorschauAbs() string {
	if e.bib == nil || e.bib.Bibliothek == nil {
		return ""
	}
	return filepath.Join(e.bib.Bibliothek.Orte.Ablage, filepath.FromSlash(e.kontext.vorschau))
}

// vorschauFrei meldet, ob Leiste bzw. Ansicht gerade bedienbar sind (nicht während Speichern, Wechsel, Arbeit).
func (e *editor) vorschauFrei() bool {
	if e.kontext.inLeiste {
		return e.leisteAn()
	}
	return e.bibliothekFrei()
}

// vorschauOffen meldet, ob die Stelle i die offene Aufnahme ist.
func (e *editor) vorschauOffen(i int) bool {
	if e.kontext.inLeiste {
		return e.unten.l.IstAktuell(i) || e.istOffen(e.vorschauAbs())
	}
	return e.bibIstOffen(i)
}

// vorschauNachbarn sind die Aufnahmen, zu denen der Editor wechseln kann, bevor die offene an der Stelle i in den
// Papierkorb geht (wie ✕ und Entf).
func (e *editor) vorschauNachbarn(i int) []string {
	if e.kontext.inLeiste {
		return e.nachbarnIn(e.unten.gezeigt, i)
	}
	return e.bibNachbarn(i)
}

// vorschauGrund sagt, warum Leiste bzw. Ansicht gerade nicht bedienbar sind (leer: sie sind es), mit den Texten von
// papierkorbAnfrage (F-877).
func (e *editor) vorschauGrund() string {
	switch {
	case e.s.SpeichernLaeuft():
		return i18n.T("Saving is still running. Please wait a moment, then try again.")
	case e.wechsel.laedt:
		return i18n.T("Another capture is still loading. Please wait a moment, then try again.")
	case e.bibl.arbeitLaeuft:
		return i18n.T("The library is still working. Please wait a moment, then try again.")
	case !e.vorschauFrei():
		return i18n.T("Not available right now. Please wait a moment, then try again.")
	}
	return ""
}

// vorschauEintraege sind die Einträge des Kontextmenüs einer Vorschau (ADR-0038 Punkt 3): Öffnen · Im Dateimanager
// zeigen │ Schlagworte… (nur volle Ansicht) │ In den Papierkorb. Sind Leiste oder Ansicht gesperrt, sind alle grau mit
// dem Grund (F-877).
func (e *editor) vorschauEintraege() []menueEintrag {
	en := e.vorschauEintraegeRoh()
	if grund := e.vorschauGrund(); grund != "" && e.vorschauStelle() >= 0 {
		for j := range en {
			en[j].an, en[j].erklaerung = false, grund
		}
	}
	return en
}

// vorschauEintraegeRoh sind die Einträge ohne die Sperre von Leiste und Ansicht.
func (e *editor) vorschauEintraegeRoh() []menueEintrag {
	i := e.vorschauStelle()
	da := i >= 0 && e.vorschauFrei()
	offen := da && e.vorschauOffen(i)
	oeffnenGrau := ""
	if offen && e.kontext.inLeiste { // in der vollen Ansicht führt Öffnen der offenen zurück zum Bild (wie Enter)
		oeffnenGrau = i18n.T("This capture is already open.")
	}
	en := []menueEintrag{
		{akt: aktVorschauOeffnen, text: i18n.T("Open"), an: da && oeffnenGrau == "", erklaerung: oeffnenGrau,
			kuerzel: tastenName(key.Filter{Name: key.NameReturn})},
		{akt: aktVorschauZeigen, text: i18n.T("Show in file manager"), an: da && !e.dateimanager.laeuft},
	}
	if !e.kontext.inLeiste {
		grau := ""
		if offen {
			grau = i18n.T("This is the open capture. Open another one first.")
		}
		en = append(en, menueEintrag{akt: aktVorschauSchlagworte, text: i18n.T("Edit keywords"), an: da && !offen,
			erklaerung: grau, kuerzel: tastenName(key.Filter{Name: key.NameF2}), gruppe: true})
	}
	einzigGrau := ""
	if offen && len(e.vorschauNachbarn(i)) == 0 {
		einzigGrau = e.einzigText()
	}
	// Kurz nach einem Löschen gilt kein Papierkorb (loeschRuht, Kritik K-2 #312): grau statt still (F-877).
	return append(en, menueEintrag{akt: aktVorschauPapierkorb, text: i18n.T("Move to trash"),
		an: da && (!offen || einzigGrau == "") && !e.loeschRuht(), erklaerung: einzigGrau,
		kuerzel: tastenName(key.Filter{Name: key.NameDeleteForward}), gruppe: true})
}

// vorschauWirkung ist die Handlung eines Eintrags des Kontextmenüs einer Vorschau (wirkung).
func (e *editor) vorschauWirkung(a aktion) func(gtx layout.Context) {
	return func(gtx layout.Context) {
		i := e.vorschauStelle()
		if i < 0 || !e.vorschauFrei() {
			return
		}
		switch a {
		case aktVorschauOeffnen:
			if e.kontext.inLeiste {
				if !e.vorschauOffen(i) {
					e.leisteOeffne(i)
				}
				return
			}
			e.bibOeffne(gtx, i)
		case aktVorschauZeigen:
			e.dateimanagerStarten(e.vorschauAbs())
		case aktVorschauSchlagworte:
			if !e.kontext.inLeiste {
				e.bibl.r.SetzeFokus(i)
				e.bibSchlagworteBeginnen()
			}
		case aktVorschauPapierkorb:
			if e.loeschRuht() { // für Leiste und Ansicht gleich (F-877)
				return
			}
			if e.kontext.inLeiste {
				e.leistePapierkorb(i)
				return
			}
			e.bibl.r.SetzeFokus(i)
			e.bibPapierkorb()
		}
	}
}

// dateimanagerStarten zeigt die Datei abs im Dateimanager, in einer Goroutine (der Busaufruf wartet, bis der
// Dateimanager antwortet). Ein zweiter Aufruf während eines laufenden tut nichts.
func (e *editor) dateimanagerStarten(abs string) {
	d := &e.dateimanager
	if abs == "" || d.laeuft {
		return
	}
	if d.antwort == nil {
		d.antwort = make(chan error, 1)
	}
	d.laeuft = true
	ch, wecker := d.antwort, e.wecker
	go func() {
		ctx, ende := context.WithTimeout(context.Background(), dateimanagerFrist)
		defer ende()
		ch <- dateimanagerZeigen(ctx, abs)
		if wecker != nil {
			wecker.Anfordern() // erst nach dem Ablegen der Antwort (F-231)
		}
	}()
}

// dateimanagerAbholen nimmt die Antwort an; ein Fehler steht in der Fußzeile. Rückgabe: ob eine kam.
func (e *editor) dateimanagerAbholen() bool {
	select {
	case err := <-e.dateimanager.antwort: // nil-Kanal: default
		e.dateimanager.laeuft = false
		switch {
		case err == nil:
		case errors.Is(err, portal.ErrKeinDateimanager):
			diagnoseEditor("Dateimanager: %v", err)
			e.meldeNebenbei(i18n.T("No file manager available to show the file (org.freedesktop.FileManager1 missing)."), true)
		default:
			diagnoseEditor("Dateimanager: %v", err)
			e.meldeNebenbei(i18n.T("Could not show the file in the file manager."), true)
		}
		return true
	default:
		return false
	}
}

// kontextErneut: Ein zweiter Rechtsklick bei offenem Kontextmenü öffnet es am neuen Punkt q (Fensterpixel) für das neue
// Ziel, wie in Qt/KDE (Kritik K-6 #313a) — auf der Leinwand, an einer Vorschau der Leiste oder der vollen Ansicht.
func (e *editor) kontextErneut(gtx layout.Context, q image.Point) {
	switch {
	case e.bibl.offen: // die Leinwand ist dann nicht zu sehen, leinwandRahmen stammt von früher
		if i := e.bibStelleBei(q); i >= 0 {
			e.bibKontext(gtx, i, q, false)
		}
	case q.In(e.leinwandRahmen):
		e.kontextRechtsklick(q.Sub(e.leinwandUrsprung))
	default:
		if i := e.leisteStelleBei(q); i >= 0 {
			e.leisteKontext(i, q, false)
		}
	}
}
