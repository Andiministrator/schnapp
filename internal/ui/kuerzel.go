// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"strings"

	"gioui.org/io/event"
	"gioui.org/io/key"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/i18n"
)

// aktion ist eine Handlung des Editors, die eine Taste, ein Knopf oder das Menü auslöst (ADR-0026). Taste und Menü
// führen sie über dieselbe Funktion aus (editor.fuehreAus), damit beide Wege gleich wirken.
type aktion int

const (
	aktKeine aktion = iota
	aktKopieren
	aktExport
	aktSpeichern
	aktRueckgaengig
	aktWiederholen
	aktTextBearbeiten
	aktLoeschen
	aktGanzesBild
	aktZurueckholen
	aktKantenform // „Seam style“: Form der Stauch-Kante reihum, ohne eigenes Kürzel (ADR-0023 Nachtrag 6)
	aktEinpassen
	aktEins
	aktSchliessen
	aktEsc   // Esc in Stufen (canvas.Sitzung.Esc), Ausnahme nach ADR-0026 Punkt 5
	aktEnter // Enter: Knopf mit Fokus, Text bestätigen bzw. Zeilenumbruch, sonst Text bearbeiten
)

// taste ist ein Tastenkürzel des Editors. Genau eines gilt: akt (Handlung), werkzeug (wählt ein Werkzeug, hatWerkzeug)
// oder ausnahme (Begründung, warum Knopf und Menü sie nicht brauchen, ADR-0026 Punkt 5).
type taste struct {
	filter      key.Filter
	akt         aktion
	werkzeug    canvas.Werkzeug
	hatWerkzeug bool
	nurText     bool   // Filter nur während der Texteingabe
	ausnahme    string // nur für den Test und Leser; nie in der Oberfläche
}

func wz(name key.Name, w canvas.Werkzeug) taste {
	return taste{filter: key.Filter{Name: name}, werkzeug: w, hatWerkzeug: true}
}

func strg(name key.Name, a aktion) taste {
	return taste{filter: key.Filter{Name: name, Required: key.ModShortcut}, akt: a}
}

// tastenTabelle sind die Tasten des Editors mit eigener Handlung (Kürzel, Werkzeuge, Esc, Enter). Die Tastenfilter
// entstehen daraus (tastenFilter); der Test verlangt für jede einen Knopf, einen Menüeintrag oder eine begründete
// Ausnahme (E-21). Nicht hier stehen die Tasten, mit denen sich das offene Menü bedienen lässt (Pfeile hoch und runter,
// Pos1, Ende, Leertaste am Menüknopf; ADR-0026 Nachträge 2 und 3): Ihre Filter setzt tasten selbst, und sie wirken nur
// im Menü.
func tastenTabelle() []taste {
	return []taste{
		{filter: key.Filter{Name: key.NameEscape}, akt: aktEsc,
			ausnahme: "Stufen: Text verwerfen (Rückgängig), Ziehen abbrechen, abwählen (Klick daneben), Auswählen (Knopf)"},
		{filter: key.Filter{Name: key.NameReturn, Optional: key.ModShift}, akt: aktEnter,
			ausnahme: "Texteingabe: bestätigen (Klick daneben), Zeilenumbruch nur per Tastatur; sonst wie Menü „Text bearbeiten“"},
		{filter: key.Filter{Name: key.NameEnter, Optional: key.ModShift}, akt: aktEnter,
			ausnahme: "wie Return (Ziffernblock)"},
		{filter: key.Filter{Name: key.NameTab, Optional: key.ModShift}, nurText: true,
			ausnahme: "während der Texteingabe abgefangen und ohne Wirkung (F-153)"},
		{filter: key.Filter{Name: key.NameDeleteForward}, akt: aktLoeschen},
		{filter: key.Filter{Name: key.NameDeleteBackward}, akt: aktLoeschen},
		strg("Z", aktRueckgaengig),
		{filter: key.Filter{Name: "Z", Required: key.ModShortcut | key.ModShift}, akt: aktWiederholen},
		strg("Y", aktWiederholen),
		strg("S", aktSpeichern),
		strg("C", aktKopieren),
		{filter: key.Filter{Name: "E", Required: key.ModShortcut | key.ModShift}, akt: aktExport},
		strg("0", aktEinpassen),
		strg("1", aktEins),
		strg("W", aktSchliessen),
		wz("V", canvas.WerkzeugAuswahl),
		wz("A", canvas.WerkzeugPfeil),
		wz("R", canvas.WerkzeugKasten),
		wz("T", canvas.WerkzeugText),
		wz("N", canvas.WerkzeugNummer),
		wz("B", canvas.WerkzeugSchwaerzen),
		wz("P", canvas.WerkzeugVerpixeln),
		wz("C", canvas.WerkzeugZuschneiden), // ohne Strg; Strg+C bleibt Kopieren
		wz("X", canvas.WerkzeugStauchen),
	}
}

// tastenFilter liefert die Filter für gtx.Event; die Tab-Taste nur während der Texteingabe.
func tastenFilter(textLaeuft bool) []key.Filter {
	var f []key.Filter
	for _, t := range tastenTabelle() {
		if !t.nurText || textLaeuft {
			f = append(f, t.filter)
		}
	}
	return f
}

// menueNavigationsFilter sind die Filter für die Navigation im Menü; sie stehen bewusst nicht in tastenTabelle
// (ADR-0026 Nachtrag 4, F-286): Es sind keine Kürzel mit eigener Handlung, sondern die Bedienung der offenen Liste, und
// sie haben keinen Knopf und keinen Menüeintrag zu spiegeln.
//
// Pfeile, Pos1 und Ende wählen im offenen Menü den Eintrag (F-255). Der Filter steht immer, nicht erst nach dem
// Öffnen: Kommen Enter und Runter ohne Bild dazwischen, ginge Runter sonst verloren (F-255 V-3). Bei geschlossenem Menü
// bleibt er ohne Wirkung; Gio fährt den Fokus nur per Tab, Pfeile gehören sonst niemandem. Die Leertaste am Menüknopf
// wirkt wie Enter und gilt als Tastatur (F-255 V-2; der Klick des Knopfes trägt keine Quelle); sie ist auf den
// Menüknopf begrenzt, damit sie im Text weiter als Zeichen ankommt.
func menueNavigationsFilter(menueKnopf event.Tag) []key.Filter {
	var f []key.Filter
	for _, n := range []key.Name{key.NameUpArrow, key.NameDownArrow, key.NameHome, key.NameEnd} {
		f = append(f, key.Filter{Name: n})
	}
	return append(f, key.Filter{Focus: menueKnopf, Name: key.NameSpace})
}

// passt meldet, ob das Ereignis zum Filter gehört: Name gleich, alle verlangten Umschalter gedrückt und keine
// außer verlangten und erlaubten (wie Gio die Filter auswertet).
func passt(f key.Filter, name key.Name, m key.Modifiers) bool {
	return f.Name == name && m.Contain(f.Required) && m&^(f.Required|f.Optional) == 0
}

// findeTaste liefert den Eintrag der Tabelle zu einer gedrückten Taste.
func findeTaste(name key.Name, m key.Modifiers, textLaeuft bool) (taste, bool) {
	for _, t := range tastenTabelle() {
		if (!t.nurText || textLaeuft) && passt(t.filter, name, m) {
			return t, true
		}
	}
	return taste{}, false
}

// kuerzelText ist das erste Kürzel einer Handlung zum Anzeigen („Ctrl+C“, deutsch „Strg+C“); leer ohne Kürzel.
func kuerzelText(a aktion) string {
	for _, t := range tastenTabelle() {
		if t.akt == a && !t.hatWerkzeug && a != aktKeine {
			return tastenName(t.filter)
		}
	}
	return ""
}

// werkzeugKuerzel ist der Buchstabe eines Werkzeugs (Hinweis über dem Knopf, ADR-0026 Punkt 3).
func werkzeugKuerzel(w canvas.Werkzeug) string {
	for _, t := range tastenTabelle() {
		if t.hatWerkzeug && t.werkzeug == w {
			return tastenName(t.filter)
		}
	}
	return ""
}

func tastenName(f key.Filter) string {
	var teile []string
	if f.Required.Contain(key.ModShortcut) {
		teile = append(teile, i18n.T("Ctrl"))
	}
	if f.Required.Contain(key.ModShift) {
		teile = append(teile, i18n.T("Shift"))
	}
	switch f.Name {
	case key.NameDeleteForward:
		teile = append(teile, i18n.T("Del"))
	case key.NameDeleteBackward:
		teile = append(teile, i18n.T("Backspace"))
	case key.NameReturn, key.NameEnter:
		teile = append(teile, i18n.T("Enter"))
	case key.NameEscape:
		teile = append(teile, "Esc")
	default:
		teile = append(teile, string(f.Name))
	}
	return strings.Join(teile, "+")
}
