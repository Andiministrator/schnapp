// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Einstellungstafel im Editor (ADR-0034): Zustand und Bedienung ohne Zeichnen. Die Tafel liest beim Öffnen
// settings.Lade (den Reiter Agents aus der Datei des MCP-Servers, ADR-0034 Nachtrag 1), hält die bearbeiteten Werte
// und übergibt beim Speichern nur die geänderten Schlüssel an den Schreiber. Lage, Klicks und Zeichnen stehen in
// einstellungen.go.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"gioui.org/io/key"
	"gioui.org/widget"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/po"
	"github.com/Andiministrator/schnapp/internal/settings"
)

// tafelReiter ist ein Reiter der Tafel (ADR-0034 Punkt 2).
type tafelReiter int

const (
	reiterAllgemein tafelReiter = iota
	reiterAufnahme
	reiterBibliothek
	reiterAgenten
	reiterAnzahl
)

// tafelArt ist die Art eines Felds: Auswahl (Liste), Schalter (Kästchen) oder Zahl (− Wert +).
type tafelArt int

const (
	artAuswahl tafelArt = iota
	artSchalter
	artZahl
)

// tafelZeile ist eine Zeile der Tafel; Abschnitt und Schlüssel wie in settings.ini.
type tafelZeile struct {
	reiter                tafelReiter
	abschnitt, schluessel string
	art                   tafelArt
}

// tafelZeilen sind die Zeilen der Tafel in der Reihenfolge von settings.Werte (Test TestTafelDecktAlleSchluessel):
// Jeder Schlüssel des Lesers hat genau eine Zeile (ADR-0034 Punkt 2).
var tafelZeilen = [...]tafelZeile{
	{reiterAllgemein, "general", "language", artAuswahl},
	{reiterAufnahme, "capture", "screen", artAuswahl},
	{reiterAufnahme, "capture", "open-editor", artSchalter},
	{reiterAufnahme, "capture", "seam", artAuswahl},
	{reiterAufnahme, "capture", "delay", artAuswahl},
	{reiterBibliothek, "library", "bar", artSchalter},
	{reiterBibliothek, "library", "cleanup", artSchalter},
	{reiterBibliothek, "library", "cleanup-days", artZahl},
	{reiterAgenten, "agents", "capture", artSchalter},
	{reiterAgenten, "agents", "confirm", artSchalter},
	{reiterAgenten, "agents", "image", artSchalter},
	{reiterAgenten, "agents", "library", artSchalter},
	{reiterAgenten, "agents", "edit", artSchalter},
}

// Ziele der Tastatur hinter den Feldern eines Reiters (tafel.ziele).
const (
	zielAbbrechen = -1
	zielSpeichern = -2
)

// tafelFolge sagt dem Editor, was nach einer Taste zu tun ist.
type tafelFolge int

const (
	folgeNichts     tafelFolge = iota
	folgeSchliessen            // verwerfen und schließen (Esc, Abbrechen, Kreuz)
	folgeSpeichern             // speichern, bei Erfolg schließen
	folgeListe                 // Auswahlliste des fokussierten Felds öffnen
)

// tafelOption ist ein Eintrag einer Auswahl: Wert in settings.ini und angezeigter Text.
type tafelOption struct{ wert, text string }

// tafel ist der Zustand der Einstellungstafel. Die Knöpfe (Gio) stehen mit darin, ihre Auswertung in einstellungen.go.
type tafel struct {
	offen   bool
	reiter  tafelReiter
	geladen settings.Einstellungen // Stand beim Öffnen; Save schreibt nur, was davon abweicht
	wert    settings.Einstellungen // bearbeiteter Stand
	// Warnungen des Lesers (ADR-0034 Punkt 7): ungültige Zeilen (Zahl, mehr: es gab mehr als der Leser nennt) und eine
	// Warnung zur ganzen Datei (unlesbar, zu groß …), die der Fuß wörtlich zeigt.
	warnZeilen int
	warnMehr   bool
	warnDatei  string
	fehler     string // Fehler beim letzten Speichern (rot im Fuß)
	// Orte (ADR-0034 Nachtrag 1, F-625): pfad ist die Datei aus settings.Pfad (XDG), pfadFehler der Grund, wenn es
	// keinen gibt. agentenDatei ist die Datei, die `schnapp mcp` liest, nur wenn sie eine andere ist: Dann liest und
	// schreibt der Reiter Agents sie. agentenOhneOrt: Der Server kennt kein Heim und nimmt die Voreinstellungen.
	pfad           string
	pfadFehler     error
	agentenDatei   string
	agentenOhneOrt bool
	fokus          int // Stelle in ziele()
	// Zahlfeld: getippter Text, solange zahlNeu falsch ist; zahlFehler: der Text ist keine gültige Zahl.
	// zahlVorher: der Wert vor dem Tippen; solange der Text ungültig ist, gilt er (F-626).
	zahlText   string
	zahlNeu    bool
	zahlFehler bool
	zahlVorher int
	listeZeile int         // Zeile, deren Auswahlliste offen ist bzw. zuletzt offen war
	sprachen   []string    // wählbare Sprachen (ohne „auto“)
	roll       rollZustand // Rollen des Inhalts, wenn er nicht in die Tafel passt (F-709, tafelrollen.go)

	// Knöpfe und Lage beim letzten Zeichnen (Fensterpixel; Tests und Drehbücher).
	reiterKnopf    [reiterAnzahl]widget.Clickable
	feldKnopf      [len(tafelZeilen)]widget.Clickable
	minus, plus    widget.Clickable
	abbrechen, ok  widget.Clickable
	kreuz          widget.Clickable
	rahmen         tafelRahmen
	gezeigteFehler string // Text im Fuß beim letzten Zeichnen (Tests)
}

// oeffne setzt die Tafel auf die gelesenen Werte; pfad ist der Ort der Datei (zum Zählen der Zeilenwarnungen).
func (t *tafel) oeffne(werte settings.Einstellungen, warn []string, pfad string, sprachen []string) {
	t.offen, t.reiter, t.fehler = true, reiterAllgemein, ""
	t.geladen, t.wert = werte, werte
	t.pfad, t.pfadFehler, t.agentenDatei, t.agentenOhneOrt = pfad, nil, "", false
	t.warnZeilen, t.warnMehr, t.warnDatei = warnungenZaehlen(pfad, warn)
	t.sprachen = sprachen
	if s := werte.Sprache; s != "" && !slices.Contains(t.sprachen, s) {
		t.sprachen = append(slices.Clone(t.sprachen), s) // von Hand eingetragen: bleibt wählbar und sichtbar
	}
	t.listeZeile = -1
	t.roll.rollen, t.roll.rest = 0, 0
	t.setzeFokus(0)
}

// agentenOrt setzt nach oeffne den Ort der Datei, die `schnapp mcp` liest (ADR-0034 Nachtrag 1, F-625): ort aus
// settings.AgentenPfad ("" = der Server kennt kein Heim und nimmt die Voreinstellungen), lade liest ihn. Ist es eine
// andere Datei als t.pfad, kommen die Werte des Reiters Agents aus ihr, und Save schreibt sie dorthin.
func (t *tafel) agentenOrt(ort string, lade func(string) (settings.Einstellungen, []string)) {
	t.agentenDatei, t.agentenOhneOrt = "", ort == ""
	if ort == "" || gleicheDatei(ort, t.pfad) {
		return
	}
	werte, warn := lade(ort)
	t.agentenDatei = ort
	t.geladen.Agenten, t.wert.Agenten = werte.Agenten, werte.Agenten
	z, m, d := warnungenZaehlen(ort, warn)
	t.warnZeilen += z
	t.warnMehr = t.warnMehr || m
	if t.warnDatei == "" {
		t.warnDatei = d
	}
}

// gleicheDatei meldet, ob a und b dieselbe Einstellungsdatei meinen: gleicher Pfad, oder gleicher Name in demselben
// Ordner (auch über einen Verweis, etwa XDG_CONFIG_HOME als Link auf ~/.config). Die Datei selbst darf fehlen.
func gleicheDatei(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if filepath.Clean(a) == filepath.Clean(b) {
		return true
	}
	if filepath.Base(a) != filepath.Base(b) {
		return false
	}
	da, ea := os.Stat(filepath.Dir(a))
	db, eb := os.Stat(filepath.Dir(b))
	return ea == nil && eb == nil && os.SameFile(da, db)
}

// agentenHinweis ist der Hinweis im Reiter Agents, wenn der Server eine andere Datei liest als die übrigen Reiter
// oder keine (ADR-0034 Nachtrag 1); sonst leer.
func (t *tafel) agentenHinweis() string {
	switch {
	case t.agentenDatei != "":
		return fmt.Sprintf(i18n.T("Agents (schnapp mcp) read %s, not %s from XDG_CONFIG_HOME. This tab edits the file of the agents; the other tabs edit the other file."),
			t.agentenDatei, t.pfad)
	case t.agentenOhneOrt:
		return i18n.T("The home directory of the user is unknown: agents (schnapp mcp) find no settings file and keep the defaults, whatever this tab saves.")
	}
	return ""
}

// warnungenZaehlen teilt die Warnungen des Lesers: Zeilenwarnungen beginnen mit „<pfad>:<Zeile>:“; die letzte
// Warnung „… und N weitere“ steht ohne Pfad und heißt, dass es mehr gibt; eine Warnung mit „<pfad>: “ gilt der ganzen
// Datei; der Fuß zeigt sie ohne den Pfad (der lange Pfad schöbe den Grund aus der Zeile).
func warnungenZaehlen(pfad string, warn []string) (zeilen int, mehr bool, datei string) {
	p := strconv.Quote(pfad) // wie der Leser den Pfad in Warnungen setzt (Steuerzeichen maskiert)
	p = p[1:len(p)-1] + ":"
	for _, w := range warn {
		rest, mitPfad := strings.CutPrefix(w, p)
		switch {
		case mitPfad && rest != "" && rest[0] >= '0' && rest[0] <= '9':
			zeilen++
		case strings.HasPrefix(w, "…"):
			mehr = true
		case datei == "" && mitPfad:
			datei = "settings.ini:" + rest
		case datei == "":
			datei = w
		}
	}
	return zeilen, mehr, datei
}

// zeilenDes liefert die Zeilen (Stellen in tafelZeilen) eines Reiters.
func zeilenDes(r tafelReiter) []int {
	var z []int
	for i, zl := range tafelZeilen {
		if zl.reiter == r {
			z = append(z, i)
		}
	}
	return z
}

// ziele sind die Tastaturziele des Reiters in Tab-Reihenfolge: seine Felder, dann Abbrechen und Speichern.
func (t *tafel) ziele() []int {
	return append(zeilenDes(t.reiter), zielAbbrechen, zielSpeichern)
}

// ziel ist das Ziel mit Tastaturfokus.
func (t *tafel) ziel() int {
	z := t.ziele()
	return z[min(max(t.fokus, 0), len(z)-1)]
}

// setzeFokus setzt den Fokus auf die Stelle i (läuft an den Enden um); ein Zahlfeld beginnt mit leerer Eingabe.
func (t *tafel) setzeFokus(i int) {
	n := len(t.ziele())
	t.fokus = ((i % n) + n) % n
	t.zahlNeu, t.zahlText, t.zahlFehler = true, "", false
}

// fokusAuf setzt den Fokus auf das Ziel z (Klick); ohne dieses Ziel im Reiter bleibt er stehen.
func (t *tafel) fokusAuf(z int) {
	if i := slices.Index(t.ziele(), z); i >= 0 && i != t.fokus {
		t.setzeFokus(i)
	}
}

// waehleReiter wechselt den Reiter; der Fokus geht auf das erste Feld.
func (t *tafel) waehleReiter(r tafelReiter) {
	t.reiter = (r%reiterAnzahl + reiterAnzahl) % reiterAnzahl
	t.roll.rollen, t.roll.rest = 0, 0 // jeder Reiter beginnt oben (F-709)
	t.setzeFokus(0)
}

// werteText ist der Wert der Zeile z in der Normalform von settings.Werte.
func werteText(e settings.Einstellungen, z int) string { return settings.Werte(e)[z].Wert }

// schalter liefert den Wert eines Schalters der Zeile z.
func (t *tafel) schalter(z int) bool {
	return werteText(t.wert, z) == "true" || werteText(t.wert, z) == "yes"
}

// schalte kehrt den Schalter der Zeile z um.
func (t *tafel) schalte(z int) {
	w := &t.wert
	switch tafelZeilen[z].abschnitt + "." + tafelZeilen[z].schluessel {
	case "capture.open-editor":
		w.Editor = !w.Editor
	case "library.bar":
		w.Leiste = !w.Leiste
	case "library.cleanup":
		w.Bibliothek.An = !w.Bibliothek.An
	case "agents.capture":
		w.Agenten.Aufnehmen = !w.Agenten.Aufnehmen
	case "agents.confirm":
		w.Agenten.Bestaetigen = !w.Agenten.Bestaetigen
	case "agents.image":
		w.Agenten.Bild = !w.Agenten.Bild
	case "agents.library":
		w.Agenten.Bibliothek = !w.Agenten.Bibliothek
	case "agents.edit":
		w.Agenten.Bearbeiten = !w.Agenten.Bearbeiten
	}
}

// optionen sind die Einträge der Auswahl in Zeile z.
func (t *tafel) optionen(z int) []tafelOption {
	switch tafelZeilen[z].schluessel {
	case "language":
		o := []tafelOption{{"auto", i18n.T("Like the system")}}
		for _, s := range t.sprachen {
			o = append(o, tafelOption{s, sprachName(s)})
		}
		return o
	case "screen":
		return []tafelOption{{string(settings.WahlKlick), i18n.T("Pick by clicking")},
			{string(settings.WahlZeiger), i18n.T("Monitor under the pointer")},
			{string(settings.WahlAlle), i18n.T("All monitors")}}
	case "seam":
		var o []tafelOption
		for _, k := range kantenFormen {
			o = append(o, tafelOption{k.Name(), kantenName(k)})
		}
		return o
	case "delay": // ADR-0042: die Stufen der Karte; ein anderer Wert aus der Datei steht als Zahl (optionText)
		o := []tafelOption{{"0", i18n.TK("delay", "None")}}
		for _, n := range verzoegerungStufen {
			o = append(o, tafelOption{strconv.Itoa(n), fmt.Sprintf(i18n.N("%d second", "%d seconds", n), n)})
		}
		return o
	}
	return nil
}

// optionText ist der angezeigte Text des aktuellen Werts der Auswahl in Zeile z.
func (t *tafel) optionText(z int) string {
	w := werteText(t.wert, z)
	for _, o := range t.optionen(z) {
		if o.wert == w {
			return o.text
		}
	}
	return w
}

// waehle setzt die Auswahl in Zeile z auf die Option i.
func (t *tafel) waehle(z, i int) {
	o := t.optionen(z)
	if i < 0 || i >= len(o) {
		return
	}
	w := o[i].wert
	switch tafelZeilen[z].schluessel {
	case "language":
		if w == "auto" {
			w = ""
		}
		t.wert.Sprache = w
	case "screen":
		t.wert.Bildschirm = settings.Bildschirmwahl(w)
	case "seam":
		if k, ok := document.KanteAus(w); ok {
			t.wert.Kante = k
		}
	case "delay":
		if n, err := strconv.Atoi(w); err == nil {
			t.wert.Verzoegerung = n
		}
	}
}

// verzoegerungStufen sind die Wartezeiten der Auswahl in Sekunden (Karte #338).
var verzoegerungStufen = []int{3, 5, 10}

// gewaehlt ist die Stelle des aktuellen Werts in den Optionen der Zeile z (-1: keine).
func (t *tafel) gewaehlt(z int) int {
	w := werteText(t.wert, z)
	return slices.IndexFunc(t.optionen(z), func(o tafelOption) bool { return o.wert == w })
}

// zahlZeile ist die Zeile des Zahlfelds (Tage).
func zahlZeile() int {
	return slices.IndexFunc(tafelZeilen[:], func(z tafelZeile) bool { return z.art == artZahl })
}

// zahlSchritt ändert die Tage um d, begrenzt auf 1 … library.MaxTage; eine angefangene Eingabe endet.
func (t *tafel) zahlSchritt(d int) {
	t.wert.Bibliothek.Tage = min(library.MaxTage, max(1, t.wert.Bibliothek.Tage+d))
	t.zahlNeu, t.zahlText, t.zahlFehler = true, "", false
}

// zahlTippe nimmt Ziffern ins Zahlfeld; der erste Tastendruck ersetzt den Wert. Eine gültige Zahl gilt sofort, eine
// ungültige (leer, 0, über MaxTage) zeigt den Fehler, und es gilt wieder der Wert vor dem Tippen (F-626): Kein
// Zwischenstand des Tippens („9“ auf dem Weg von 90 zu leer) bleibt stehen.
func (t *tafel) zahlTippe(s string) {
	if t.zahlNeu {
		t.zahlNeu, t.zahlText, t.zahlVorher = false, "", t.wert.Bibliothek.Tage
	}
	for _, r := range s {
		if r >= '0' && r <= '9' && utf8.RuneCountInString(t.zahlText) < len(strconv.Itoa(library.MaxTage)) {
			t.zahlText += string(r)
		}
	}
	t.zahlUebernimm()
}

// zahlZurueck löscht die letzte Ziffer der Eingabe (vom angezeigten Wert aus).
func (t *tafel) zahlZurueck() {
	if t.zahlNeu {
		t.zahlNeu, t.zahlText, t.zahlVorher = false, strconv.Itoa(t.wert.Bibliothek.Tage), t.wert.Bibliothek.Tage
	}
	if n := len(t.zahlText); n > 0 {
		t.zahlText = t.zahlText[:n-1]
	}
	t.zahlUebernimm()
}

func (t *tafel) zahlUebernimm() {
	n, err := strconv.Atoi(t.zahlText)
	t.zahlFehler = err != nil || n < 1 || n > library.MaxTage
	if t.zahlFehler {
		t.wert.Bibliothek.Tage = t.zahlVorher
	} else {
		t.wert.Bibliothek.Tage = n
	}
}

// zahlAnzeige ist der Text im Zahlfeld.
func (t *tafel) zahlAnzeige() string {
	if t.zahlNeu {
		return strconv.Itoa(t.wert.Bibliothek.Tage)
	}
	return t.zahlText
}

// taste bedient die Tafel per Tastatur (ADR-0034 Punkt 4), solange keine Auswahlliste offen ist.
func (t *tafel) taste(n key.Name, m key.Modifiers) tafelFolge {
	z := t.ziel()
	strg := m.Contain(key.ModShortcut)
	switch {
	case n == key.NameEscape:
		return folgeSchliessen
	case n == key.NameTab:
		if m.Contain(key.ModShift) {
			t.setzeFokus(t.fokus - 1)
		} else {
			t.setzeFokus(t.fokus + 1)
		}
	case strg && n == key.NamePageDown:
		t.waehleReiter(t.reiter + 1)
	case strg && n == key.NamePageUp:
		t.waehleReiter(t.reiter - 1)
	case n == key.NameReturn || n == key.NameEnter:
		if z == zielAbbrechen {
			return folgeSchliessen
		}
		return folgeSpeichern // auch aus einem Feld
	case n == key.NameSpace:
		switch {
		case z == zielAbbrechen:
			return folgeSchliessen
		case z == zielSpeichern:
			return folgeSpeichern
		case tafelZeilen[z].art == artSchalter:
			t.schalte(z)
		case tafelZeilen[z].art == artAuswahl:
			t.listeZeile = z
			return folgeListe
		}
	case z >= 0 && tafelZeilen[z].art == artZahl:
		switch n {
		case key.NameUpArrow, "+":
			t.zahlSchritt(1)
		case key.NameDownArrow, "-":
			t.zahlSchritt(-1)
		case key.NameDeleteBackward:
			t.zahlZurueck()
		}
	}
	return folgeNichts
}

// tippe nimmt getippten Text: Ziffern im Zahlfeld, sonst ohne Wirkung.
func (t *tafel) tippe(s string) {
	if z := t.ziel(); z >= 0 && tafelZeilen[z].art == artZahl {
		t.zahlTippe(s)
	}
}

// errKeinOrt: Es gibt keinen Ort für die Einstellungsdatei (settings.Pfad scheiterte ohne Grund).
var errKeinOrt = errors.New("cannot locate the settings file")

// speichere schreibt die Änderungen gegenüber dem Stand beim Öffnen (ADR-0034 Punkte 3 und 5): die des Reiters Agents
// in die Datei des MCP-Servers, wenn sie eine andere ist (Nachtrag 1, F-625), die übrigen nach t.pfad. Ein ungültiges
// Zahlfeld speichert nichts (F-626). ok ist falsch bei einem Fehler (dann steht er in t.fehler, die Tafel bleibt
// offen); geschrieben, wenn etwas geschrieben wurde.
func (t *tafel) speichere(schreibe func(string, []settings.Aenderung) error) (geschrieben, ok bool) {
	if t.zahlFehler {
		t.fehler = fmt.Sprintf(i18n.T("Could not save the settings: “%s” must be a number from 1 to %d."),
			zeileTitel(zahlZeile()), library.MaxTage)
		return false, false
	}
	var agenten, uebrige []settings.Aenderung
	for _, a := range settings.Unterschiede(t.geladen, t.wert) {
		if t.agentenDatei != "" && a.Abschnitt == "agents" {
			agenten = append(agenten, a)
		} else {
			uebrige = append(uebrige, a)
		}
	}
	// Agenten zuerst: Scheitert danach die andere Datei, gilt das Abschalten für den Server schon.
	if len(agenten) > 0 {
		if err := schreibe(t.agentenDatei, agenten); err != nil {
			t.fehler = fmt.Sprintf(i18n.T("Could not save the settings: %v"), fehlerGrund(err))
			return false, false
		}
		t.geladen.Agenten = t.wert.Agenten
		geschrieben = true
	}
	if len(uebrige) > 0 {
		err := t.pfadFehler
		if t.pfad != "" {
			err = schreibe(t.pfad, uebrige)
		} else if err == nil {
			err = errKeinOrt
		}
		if err != nil {
			t.fehler = fmt.Sprintf(i18n.T("Could not save the settings: %v"), fehlerGrund(err))
			return geschrieben, false
		}
		t.geladen = t.wert
		geschrieben = true
	}
	return geschrieben, true
}

// sprachName ist der Name einer Sprache in ihr selbst (so findet man die eigene auch in einer fremden Oberfläche).
func sprachName(code string) string {
	switch code {
	case "en":
		return "English"
	case "de":
		return "Deutsch"
	}
	return code
}

// tafelSprachen sind die eingebauten Sprachen: die Ausgangssprache und je PO-Datei eine (ADR-0014).
func tafelSprachen() []string {
	s, _ := i18n.Sprachen(po.Dateien)
	if !slices.Contains(s, i18n.Ausgangssprache) {
		s = append([]string{i18n.Ausgangssprache}, s...)
	}
	return s
}

// reiterName ist der Text eines Reiters.
func reiterName(r tafelReiter) string {
	switch r {
	case reiterAufnahme:
		return i18n.T("Capture")
	case reiterBibliothek:
		return i18n.T("Library")
	case reiterAgenten:
		return i18n.T("Agents")
	}
	return i18n.T("General")
}

// reiterKopf ist die Überschrift über den Zeilen eines Reiters.
func reiterKopf(r tafelReiter) string {
	switch r {
	case reiterAufnahme:
		return i18n.T("CAPTURE")
	case reiterBibliothek:
		return i18n.T("LIBRARY")
	case reiterAgenten:
		return i18n.T("AGENTS (MCP SERVER “schnapp mcp”)")
	}
	return i18n.T("GENERAL")
}

// reiterWirkung ist der Hinweis im Fuß, wann die Einstellungen des Reiters wirken (ADR-0034 Punkt 3).
func reiterWirkung(r tafelReiter) string {
	switch r {
	case reiterAufnahme:
		return i18n.T("Applies from the next capture.")
	case reiterBibliothek:
		// Die Leiste der letzten Aufnahmen folgt sofort (ADR-0034 Nachtrag 2), das Aufräumen läuft beim Start.
		return i18n.T("Recent captures apply immediately, clean-up at the next start.")
	case reiterAgenten:
		return i18n.T("Applies immediately.")
	}
	return i18n.T("Applies after Schnapp restarts.")
}

// reiterEinleitung ist ein Text unter der Überschrift (nur Agenten).
func reiterEinleitung(r tafelReiter) string {
	if r == reiterAgenten {
		return i18n.T("What an AI agent may do through Schnapp. Only changeable here or in the file, never by the agent itself. Applies from the agent's next tool call.")
	}
	return ""
}

// zeileTitel ist die Beschriftung links, zeileSchalterText der Text neben einem Kästchen, zeileErklaerung der graue Text
// unter dem Feld, zeileWarnung der Warnkasten unter einem eingeschalteten Agenten-Schalter (ADR-0034 Punkt 6).
func zeileTitel(z int) string {
	switch tafelZeilen[z].abschnitt + "." + tafelZeilen[z].schluessel {
	case "general.language":
		return i18n.T("Language")
	case "capture.screen", "agents.capture":
		return i18n.T("Capture screen")
	case "capture.open-editor":
		return i18n.T("After capture")
	case "capture.seam":
		return i18n.T("Seam edge")
	case "capture.delay":
		return i18n.T("Delay")
	case "library.bar":
		return i18n.T("Recent captures")
	case "library.cleanup":
		return i18n.T("Clean up")
	case "library.cleanup-days":
		return i18n.T("After days")
	case "agents.confirm":
		return i18n.T("Before each capture")
	case "agents.image":
		return i18n.T("Image to the agent")
	case "agents.library":
		return i18n.T("Read the library")
	case "agents.edit":
		return i18n.T("Annotate and redact files")
	}
	return ""
}

func zeileSchalterText(z int) string {
	switch tafelZeilen[z].abschnitt + "." + tafelZeilen[z].schluessel {
	case "capture.open-editor":
		return i18n.T("Open the editor")
	case "library.bar":
		return i18n.T("Show below the image")
	case "library.cleanup":
		return i18n.T("Old captures to the trash")
	case "agents.confirm":
		return i18n.T("Ask")
	case "agents.image":
		return i18n.T("Send along")
	}
	return i18n.T("Allowed")
}

func zeileErklaerung(z int) string {
	switch tafelZeilen[z].abschnitt + "." + tafelZeilen[z].schluessel {
	case "general.language":
		return i18n.T("Applies after Schnapp restarts.")
	case "capture.screen":
		return i18n.T("What “Capture screen” does without a monitor name: pick by clicking, the monitor under the pointer or all monitors.")
	case "capture.seam":
		return i18n.T("Applies to new captures and opened images; a working file keeps its edge.")
	case "capture.delay":
		return i18n.T("Wait before each capture, for example to open a menu; a notification counts down. --delay=SECONDS overrides it.")
	case "library.bar":
		return i18n.T("The bar with the latest captures; also View → Recent captures.")
	case "library.cleanup-days":
		return i18n.T("Older than this number of days, checked at start.")
	}
	return ""
}

func zeileWarnung(z int) string {
	switch tafelZeilen[z].abschnitt + "." + tafelZeilen[z].schluessel {
	case "agents.capture":
		return i18n.T("An agent then sees what is on the screen — including mails, passwords in plain text, names. Every capture shows a notice first; with “Ask” below, every capture must be confirmed.")
	case "agents.library":
		return i18n.T("An agent can then search and read every capture in the library, old ones included.")
	case "agents.edit":
		return i18n.T("An agent can then annotate and black out image and working files; the files change on disk.")
	}
	return ""
}

// warnungSichtbar meldet, ob unter der Zeile z die Warnung steht: nur unter eingeschalteten Agenten-Schaltern mit
// Warnung.
func (t *tafel) warnungSichtbar(z int) bool {
	return tafelZeilen[z].art == artSchalter && zeileWarnung(z) != "" && t.schalter(z)
}

// fussWarnung ist die Zeile über ungültige Zeilen bzw. die Warnung zur Datei (ADR-0034 Punkt 7); leer ohne Warnung.
func (t *tafel) fussWarnung() string {
	switch {
	case t.warnDatei != "":
		return t.warnDatei
	case t.warnMehr:
		return fmt.Sprintf(i18n.T("More than %d lines in settings.ini are invalid and stay as they are."), t.warnZeilen)
	case t.warnZeilen > 0:
		return fmt.Sprintf(i18n.N("%d line in settings.ini is invalid and stays as it is.",
			"%d lines in settings.ini are invalid and stay as they are.", t.warnZeilen), t.warnZeilen)
	}
	return ""
}
