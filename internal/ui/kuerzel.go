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
	aktKantenform    // „Seam style“: Form der Stauch-Kante reihum, ohne eigenes Kürzel (ADR-0023 Nachtrag 6)
	aktTastenkuerzel // Menü Hilfe: Übersicht der Tastenkürzel (F1, ADR-0031 Punkt 1)
	aktUeber         // Menü Hilfe: Über Schnapp
	aktEinpassen
	aktEins
	aktSeitenleiste // Menü Ansicht: rechte Leiste ein- und ausklappen (ADR-0031 Punkt 3)
	aktSchliessen
	aktEsc   // Esc in Stufen (canvas.Sitzung.Esc), Ausnahme nach ADR-0026 Punkt 5
	aktEnter // Enter: Knopf mit Fokus, Text bestätigen bzw. Zeilenumbruch, sonst Text bearbeiten
	// Menü „Ebene“ (ADR-0032 Punkt 8)
	aktDoppeln
	aktUmbenennen
	aktAusblenden
	aktSperren
	aktEbeneVor
	aktEbeneZurueck
	aktEbeneDarueber
	aktEbeneDarunter
	aktFarbenTauschen // Menü Bearbeiten: Vordergrund und Füllung tauschen (ADR-0021 Nachtrag 11 Punkt 5)
	aktEinrasten      // Menü Ansicht: Einrasten an Objekten ein/aus, ohne Kürzel (ADR-0021 Nachtrag 12)
	aktNavigator      // Menü Ansicht: Navigator (Mini-Übersicht) ein/aus, ohne Kürzel (ADR-0031 Nachtrag 9)
	aktEinstellungen  // Menü Datei: Einstellungstafel (Strg+,, ADR-0034)
	aktTextKopieren   // Menü Bearbeiten: Text aus dem Bild kopieren (Strg+Umschalt+T, ADR-0035)
	aktLeiste         // Menü Ansicht: Leiste der letzten Aufnahmen ein/aus (Strg+Umschalt+R, ADR-0036 Punkt 2)
	aktBibliothek     // Menü Datei: volle Ansicht der Bibliothek auf/zu (Strg+Umschalt+L, ADR-0036 Punkt 3)
	// Menü Bearbeiten bei offener Ansicht der Bibliothek (ADR-0036 Punkt 4). Ihre Tasten (Enter, F2, Entf, Strg+F) holt
	// die Ansicht selbst ab (bibliothekFilter); sie stehen deshalb nicht in der Tabelle, der Menüeintrag zeigt sie über
	// menueEintrag.kuerzel (ADR-0026 Nachtrag 10).
	aktBibOeffnen
	aktBibSchlagworte
	aktBibPapierkorb
	aktBibSuchen
	aktOeffnen            // Menü Datei: Öffnen… (Strg+O, ADR-0026 Nachtrag 11)
	aktSchnellstilSichern // Kontextmenü: Stil des Objekts als Schnellstil sichern (ADR-0038 Punkt 2)
	// Kontextmenü einer Vorschau der Bibliothek (ADR-0038 Punkt 3): Sie gelten der Vorschau, an der es geöffnet wurde
	// (kontextMenue.vorschau), nicht der mit Fokus; deshalb eigene Handlungen neben aktBib….
	aktVorschauOeffnen
	aktVorschauZeigen
	aktVorschauSchlagworte
	aktVorschauPapierkorb
	// Mehrfachauswahl (ADR-0039): Menü Bearbeiten → „Select all“ (Strg+A), Menü Ebene → Ausrichten und Verteilen.
	aktAllesWaehlen
	aktAusrichtenLinks
	aktAusrichtenMitteWaagerecht
	aktAusrichtenRechts
	aktAusrichtenOben
	aktAusrichtenMitteSenkrecht
	aktAusrichtenUnten
	aktVerteilenWaagerecht
	aktVerteilenSenkrecht
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
// Pos1, Ende, Leertaste am Menütitel; ADR-0026 Nachträge 2 und 3): Ihre Filter setzt tasten selbst, und sie wirken nur
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
		// Alt allein, gedrückt und losgelassen (ADR-0021 Nachtrag 12 Punkt 4): Die Umschalter sind erlaubt, weil der
		// Tastendruck selbst je nach Reihenfolge mit oder ohne ModAlt ankommt.
		{filter: key.Filter{Name: key.NameAlt, Optional: key.ModAlt | key.ModShift | key.ModCtrl | key.ModSuper},
			ausnahme: "Zustand während des Ziehens, kein Kürzel: Einrasten aus, solange gehalten (Menü Ansicht → „Snap to objects“)"},
		// Leertaste gehalten (ADR-0039 Nachtrag 1): nur mit Fokus auf der Leinwand und außerhalb der Texteingabe angemeldet
		// (alleTastenFilter), dort bleibt sie ein Zeichen.
		{filter: key.Filter{Name: key.NameSpace, Optional: key.ModShift},
			ausnahme: "Zustand während des Ziehens, kein Kürzel: Ziehen verschiebt die Ansicht, solange gehalten (wie die mittlere Taste)"},
		{filter: key.Filter{Name: key.NameDeleteForward}, akt: aktLoeschen},
		{filter: key.Filter{Name: key.NameDeleteBackward}, akt: aktLoeschen},
		strg("Z", aktRueckgaengig),
		{filter: key.Filter{Name: "Z", Required: key.ModShortcut | key.ModShift}, akt: aktWiederholen},
		strg("Y", aktWiederholen),
		strg("S", aktSpeichern),
		strg("O", aktOeffnen),
		strg("C", aktKopieren),
		{filter: key.Filter{Name: "E", Required: key.ModShortcut | key.ModShift}, akt: aktExport},
		strg("0", aktEinpassen),
		strg("1", aktEins),
		{filter: key.Filter{Name: "B", Required: key.ModShortcut | key.ModShift}, akt: aktSeitenleiste},
		strg("W", aktSchliessen),
		strg(",", aktEinstellungen), // Komma: in Bildprogrammen und Editoren üblich (ADR-0034 Punkt 1)
		{filter: key.Filter{Name: key.NameF1}, akt: aktTastenkuerzel},
		{filter: key.Filter{Name: key.NameF10, Required: key.ModShift},
			ausnahme: "öffnet das Kontextmenü der Auswahl (ADR-0038 Punkt 6); dasselbe Menü öffnet der Rechtsklick"},
		strg("D", aktDoppeln),
		strg("A", aktAllesWaehlen), // A allein ist das Werkzeug Pfeil; Strg+A war frei (ADR-0039 Punkt 2)
		{filter: key.Filter{Name: key.NameF2}, akt: aktUmbenennen},
		strg("H", aktAusblenden),
		strg("L", aktSperren),
		strg(key.NamePageUp, aktEbeneVor),
		strg(key.NamePageDown, aktEbeneZurueck),
		{filter: key.Filter{Name: key.NamePageUp, Required: key.ModAlt}, akt: aktEbeneDarueber},
		{filter: key.Filter{Name: key.NamePageDown, Required: key.ModAlt}, akt: aktEbeneDarunter},
		// X allein ist Stauchen; Umschalt+X allein träfe beim Tippen eines großen X (ADR-0021 Nachtrag 11 Punkt 5).
		{filter: key.Filter{Name: "X", Required: key.ModShortcut | key.ModShift}, akt: aktFarbenTauschen},
		// T allein ist das Werkzeug Text; Strg+Umschalt+T war frei (ADR-0035 Punkt 5).
		{filter: key.Filter{Name: "T", Required: key.ModShortcut | key.ModShift}, akt: aktTextKopieren},
		// R allein ist das Werkzeug Kasten; Strg+Umschalt+R war frei (ADR-0036 Punkt 2).
		{filter: key.Filter{Name: "R", Required: key.ModShortcut | key.ModShift}, akt: aktLeiste},
		// L allein ist frei, Strg+L sperrt die Ebene; Strg+Umschalt+L war frei (ADR-0036 Punkt 3).
		{filter: key.Filter{Name: "L", Required: key.ModShortcut | key.ModShift}, akt: aktBibliothek},
		wz("V", canvas.WerkzeugAuswahl),
		wz("A", canvas.WerkzeugPfeil),
		wz("R", canvas.WerkzeugKasten),
		wz("T", canvas.WerkzeugText),
		wz("K", canvas.WerkzeugSprechblase), // K wie Kommentar-Blase; frei (ADR-0021 Nachtrag 13 Punkt 5)
		wz("N", canvas.WerkzeugNummer),
		wz("M", canvas.WerkzeugLupe), // M wie magnifier (ADR-0033 Punkt 8)
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
// bleibt er ohne Wirkung; Gio fährt den Fokus nur per Tab, Pfeile gehören sonst niemandem. Die Leertaste am Menütitel
// wirkt wie Enter und gilt als Tastatur (F-255 V-2; der Klick des Knopfes trägt keine Quelle); sie ist auf den
// Menütitel begrenzt, damit sie im Text weiter als Zeichen ankommt.
func menueNavigationsFilter(menueTitel ...event.Tag) []key.Filter {
	var f []key.Filter
	for _, n := range []key.Name{key.NameUpArrow, key.NameDownArrow, key.NameHome, key.NameEnd, key.NameLeftArrow,
		key.NameRightArrow, key.NameF10} {
		f = append(f, key.Filter{Name: n})
	}
	for _, t := range menueTitel {
		f = append(f, key.Filter{Focus: t, Name: key.NameSpace})
	}
	return f
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

// alleKuerzel sind alle Tasten einer Handlung in der Reihenfolge der Tabelle (Übersicht im Menü Hilfe).
func alleKuerzel(a aktion) []string {
	var k []string
	for _, t := range tastenTabelle() {
		if t.akt == a && !t.hatWerkzeug && a != aktKeine {
			k = append(k, tastenName(t.filter))
		}
	}
	return k
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
	if f.Required.Contain(key.ModAlt) {
		teile = append(teile, "Alt")
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
	case key.NamePageUp:
		teile = append(teile, i18n.T("PgUp"))
	case key.NamePageDown:
		teile = append(teile, i18n.T("PgDn"))
	default:
		teile = append(teile, string(f.Name))
	}
	return strings.Join(teile, "+")
}
