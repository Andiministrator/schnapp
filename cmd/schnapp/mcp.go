// SPDX-License-Identifier: GPL-3.0-or-later

package main

// MCP-Server `schnapp mcp` (ADR-0030 Fassung 2, Nachtrag 2): Werkzeuge auf denselben Kernschichten wie die
// Kommandozeile. Das Protokoll liegt in internal/mcp; hier stehen Einwilligung, Bestätigung, Hinweis und Ablage.

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/Andiministrator/schnapp/internal/annotate"
	"github.com/Andiministrator/schnapp/internal/capture"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/export"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/fonts"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/mcp"
	"github.com/Andiministrator/schnapp/internal/notify"
	"github.com/Andiministrator/schnapp/internal/po"
	"github.com/Andiministrator/schnapp/internal/render"
	"github.com/Andiministrator/schnapp/internal/selection"
	"github.com/Andiministrator/schnapp/internal/settings"
	"github.com/Andiministrator/schnapp/internal/ui"
)

// mcpArgument startet den Server: genau `schnapp mcp`, sonst nichts (eine Datei namens „mcp“ öffnet `./mcp`).
const mcpArgument = "mcp"

// bestaetigenArgument startet den Bestätigungsdialog als Kindprozess (ADR-0030 Nachtrag 2 Punkt 4). Die Angaben
// kommen als eine JSON-Zeile über stdin, nicht über die Kommandozeile.
const bestaetigenArgument = "--internal-confirm-capture"

// Variablen statt Konstanten, damit Tests die Grenzen klein setzen können.
var (
	// bestaetigenFrist: so lange wartet der Server auf eine Antwort im Dialog, dann lehnt er ab.
	bestaetigenFrist = 60 * time.Second
	// maxBildBytes: größtes PNG, das als Bildinhalt an den Agenten geht.
	maxBildBytes = 8 << 20
	// kindProgramm liefert das Programm des Dialog-Kindes (Naht für Tests: das Testprogramm selbst, F-471).
	kindProgramm = os.Executable
	// zeigeDialog ist der Dialog im Kind (Naht für Tests, F-471).
	zeigeDialog = ui.Bestaetige
	// Das Heimverzeichnis aus der Benutzerdatenbank liefert settings.BenutzerHeim (Naht für Tests, F-461): Die
	// Einstellungstafel des Editors bestimmt damit dieselbe Datei wie der Server (ADR-0034 Nachtrag 1, F-625).
	// kindZugabe: Das Kind schließt den Dialog nach bestaetigenFrist + kindZugabe selbst und lehnt ab; kindHart
	// danach beendet es sich hart (ADR-0030 Nachtrag 4 Punkt 5, F-469).
	kindZugabe = 5 * time.Second
	kindHart   = 2 * time.Second
)

const (
	// fensterWeg: Abstand zwischen dem Ende des Dialogs und der Aufnahme, damit das Fenster nicht im Bild steht.
	fensterWeg = 300 * time.Millisecond
	// hinweisWeg: Abstand zwischen dem Schließen des vorigen Hinweises und der nächsten Aufnahme; Plasma blendet ihn
	// aus, statt ihn sofort zu entfernen (F-490).
	hinweisWeg = 600 * time.Millisecond
	// maxKundenname: so viele Zeichen des vom Agenten gemeldeten Namens zeigt der Dialog.
	maxKundenname = 60
	// maxMonitorname: so viele Zeichen eines Monitornamens erscheinen in Dialog und Hinweis (F-463).
	maxMonitorname = 40
	// maxZieltext: so viele Zeichen des Zieltexts zeigt das Kind höchstens (zweite Maske, F-463).
	maxZieltext = 120
	// Grenzen für library_search.
	suchVoreinstellung = 20
	suchHoechstens     = 50
)

// mcpAnschluesse sind die Außenanschlüsse des Servers; Tests setzen Attrappen ein.
type mcpAnschluesse struct {
	lade     Ladefunktion
	quelle   func() (capture.Quelle, error)
	monitore func() ([]ui.Monitor, error)
	// monitoreListe liefert die Monitore mit Skala für list_screens (ADR-0017 Nachtrag 9); nil: monitore. Die
	// Namensprüfung von capture_screen nimmt monitore ohne zweite Wayland-Verbindung (F-681).
	monitoreListe func() ([]ui.Monitor, error)
	bestaetige    func(ctx context.Context, ziel, kunde string) (bool, error)
	melde         func(ctx context.Context, titel, text string) (uint32, error)
	schliesse     func(ctx context.Context, id uint32) error
	// hinweis hält den letzten Hinweis dieses Servers (Kennung 0: keiner offen). Ein Zeiger, weil die Werkzeuge auf
	// einer Kopie der Anschlüsse laufen; nil schaltet das Schließen ab (F-490).
	hinweis     *letzterHinweis
	orte        func() (library.Orte, error)
	letzterPfad string
	warte       func(ctx context.Context, d time.Duration) error
	// einstellungsOrt nennt die Datei in den Ablehnungstexten.
	einstellungsOrt string
	fehler          io.Writer
	// startWarnung schreibt laufeMcp beim Start auf stderr (leer = keine): abweichende Umgebung, unbekanntes Heim.
	startWarnung string
}

func echteMcpAnschluesse(fehler io.Writer) mcpAnschluesse {
	an := mcpAnschluesse{
		quelle:        func() (capture.Quelle, error) { return capture.NeuKWin() },
		monitore:      ui.Monitore,
		monitoreListe: ui.MonitoreMitSkala, // nur list_screens (F-668, F-681)
		bestaetige:    bestaetigeImKind,
		melde:         notify.Zeige,
		schliesse:     notify.Schliesse,
		hinweis:       new(letzterHinweis),
		warte: func(ctx context.Context, d time.Duration) error {
			select {
			case <-time.After(d):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
		fehler: fehler,
	}
	heim, err := settings.BenutzerHeim()
	if err != nil {
		diagnose(fehler, "Heimverzeichnis aus der Benutzerdatenbank: %v", err)
		heim = ""
	}
	an.mitHeim(heim, os.Getenv)
	return an
}

// mcpUmgebung sind die Variablen, denen der Server nicht folgt (ADR-0030 Nachtrag 4 Punkt 1, F-461).
var mcpUmgebung = []string{"HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME"}

// mitHeim setzt Einstellungen und Orte der Bibliothek aus heim (Heimverzeichnis der Benutzerdatenbank), ohne HOME und
// XDG_* der Umgebung: Die setzt der Agenten-Client, womöglich aus einer fremden `.mcp.json` (F-461). umgebung dient nur
// der Warnung. Ist heim nicht absolut, gelten die Voreinstellungen und kein Ort ist bekannt.
func (an *mcpAnschluesse) mitHeim(heim string, umgebung func(string) string) {
	// Schriftordner ebenso aus dem Heim, nicht aus XDG_DATA_HOME/XDG_DATA_DIRS: Ein Ordner des Agenten-Clients schöbe
	// sonst jedem Familiennamen eigene Glyphen unter (Sitzungs-QA B-3, ADR-0021 Nachtrag 10 Punkt 3).
	schriften := []string{"/usr/local/share/fonts", "/usr/share/fonts"}
	if filepath.IsAbs(heim) {
		h := filepath.Clean(heim)
		schriften = append([]string{filepath.Join(h, ".local", "share", "fonts"), filepath.Join(h, ".fonts")}, schriften...)
	}
	fonts.SetzeOrdner(schriften)
	if !filepath.IsAbs(heim) {
		an.lade = func() (settings.Einstellungen, []string) { return settings.Voreinstellungen(), nil }
		an.orte = func() (library.Orte, error) { return library.Orte{}, library.ErrKeinHeim }
		an.letzterPfad = ""
		an.einstellungsOrt = "~/.config/schnapp/settings.ini"
		an.startWarnung = i18n.T("schnapp mcp: the home directory of the user is unknown; agents can neither capture nor read the library.")
		return
	}
	heim = filepath.Clean(heim)
	nurHeim := func(k string) string {
		if k == "HOME" {
			return heim
		}
		return ""
	}
	ini, _ := settings.AgentenPfadIn(heim) // heim ist absolut, also kein Fehler; derselbe Ort wie der Reiter Agents (F-625)
	an.lade = func() (settings.Einstellungen, []string) { return settings.LadeDatei(ini) }
	an.orte = func() (library.Orte, error) { return library.OrteAus(nurHeim) }
	an.letzterPfad = selection.LetzterPfad(nurHeim)
	an.einstellungsOrt = "~/.config/schnapp/settings.ini" // ohne Heimpfad an den Agenten (Sitzungs-QA A-4)
	an.startWarnung = ""
	if abw := abweichendeUmgebung(heim, umgebung); len(abw) > 0 {
		an.startWarnung = fmt.Sprintf(i18n.T("schnapp mcp: ignoring %s from the environment; settings and library come from the home directory %s."),
			strings.Join(abw, ", "), heim)
	}
}

// abweichendeUmgebung nennt die Variablen aus mcpUmgebung, die gesetzt sind und nicht auf den Ort unter heim zeigen,
// den die XDG-Vorgabe ohnehin nähme.
func abweichendeUmgebung(heim string, umgebung func(string) string) []string {
	vorgabe := map[string]string{
		"HOME":            heim,
		"XDG_CONFIG_HOME": filepath.Join(heim, ".config"),
		"XDG_DATA_HOME":   filepath.Join(heim, ".local", "share"),
		"XDG_CACHE_HOME":  filepath.Join(heim, ".cache"),
		"XDG_STATE_HOME":  filepath.Join(heim, ".local", "state"),
	}
	var abw []string
	for _, k := range mcpUmgebung {
		if w := umgebung(k); w != "" && filepath.Clean(w) != vorgabe[k] {
			abw = append(abw, k)
		}
	}
	return abw
}

// mcpSignale beendet den Server beim ersten Signal sanft (abbrechen) und beim zweiten sofort mit Exit 1 (ADR-0030
// Nachtrag 4 Punkt 6, F-467): Hängt ein Werkzeug, hilft sonst nur SIGKILL. Endet, wenn fertig schließt. Seine Meldungen
// auf stderr warten höchstens meldungFrist (meldeKurz, QS-1).
func mcpSignale(kanal <-chan os.Signal, abbrechen func(), beende func(int), fehler io.Writer, fertig <-chan struct{}) {
	n := 0
	for {
		select {
		case <-fertig:
			return
		case <-kanal:
			n++
			if n == 1 {
				abbrechen()
				continue
			}
			if signalHalten() {
				// Ein laufendes annotate speichert zu Ende (Vorschau, Entfernen einer ausgewichenen Fassung), solange es
				// Fortschritt meldet (Kritik #202 B-2); weitere Signale ändern daran nichts.
				meldeKurz(fehler, i18n.T("schnapp mcp: second signal, stopping after the running save."))
				for signalHalten() {
					select {
					case <-fertig:
						return
					case <-kanal:
					case <-time.After(10 * time.Millisecond):
					}
				}
			}
			raeumeLaufendeTempsAuf() // vor dem Beenden, nicht der Signalwache überlassen (F-484); vor der Meldung (QS-1)
			// Ein blockiertes stderr (volle Pipe, die der Client nicht liest) hält das Beenden nicht auf (QS-1).
			meldeKurz(fehler, i18n.T("schnapp mcp: second signal, stopping now."))
			beende(exitFehler)
			return
		}
	}
}

// laufeMcp bedient einen Client über stdin/stdout, bis stdin endet oder ein Signal kommt.
func laufeMcp(ein io.Reader, aus, fehler io.Writer, an mcpAnschluesse) int {
	if err := probeGuard(); err != nil {
		fmt.Fprintln(fehler, "schnapp (Probe-Build):", err)
		return exitFehler
	}
	if an.startWarnung != "" {
		fmt.Fprintln(fehler, an.startWarnung)
	}
	ctx, abbrechen := context.WithCancel(context.Background())
	defer abbrechen()
	var fangen []os.Signal
	for _, s := range []os.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP} {
		if !signal.Ignored(s) { // wie die Signalwache (F-443): geerbtes Ignorieren bleibt
			fangen = append(fangen, s)
		}
	}
	kanal := make(chan os.Signal, 4)
	if len(fangen) > 0 {
		meldeAn(kanal, fangen...)
		defer meldeAb(kanal)
	}
	fertig := make(chan struct{})
	defer close(fertig)
	go mcpSignale(kanal, abbrechen, beendeProzess, fehler, fertig)
	s := &mcp.Server{Name: "schnapp", Version: versionsText(), Log: fehler, Werkzeuge: an.werkzeuge(),
		Anleitung: "Schnapp takes screenshots on a KDE Plasma (Wayland) desktop. Capturing, reading the library and " +
			"annotating files need the user's consent in the settings file; a denied call says which switch is missing."}
	if err := s.Bediene(ctx, ein, aus); err != nil {
		fmt.Fprintf(fehler, i18n.T("MCP server stopped: %v")+"\n", err)
		return exitFehler
	}
	return exitOK
}

func schema(s string) json.RawMessage { return json.RawMessage(s) }

func (an mcpAnschluesse) werkzeuge() []mcp.Werkzeug {
	return []mcp.Werkzeug{
		{Name: "list_screens", Titel: "List screens",
			Beschreibung: "Lists the monitors with name and size in pixels. The name is the value for capture_screen.",
			Eingabe:      schema(`{"type":"object","properties":{},"additionalProperties":false}`),
			Aufruf:       an.listeMonitore},
		{Name: "capture_screen", Titel: "Capture a screen",
			Beschreibung: "Captures one monitor (screen), all monitors (all: true) or, without both, the active monitor. " +
				"Needs consent ([agents] capture = yes); the user may have to confirm in a dialog. Without output the " +
				"screenshot is stored in the Schnapp library (agents folder); with output (absolute path ending in " +
				".png, .jpg or .svg) it is written there instead.",
			Eingabe: schema(`{"type":"object","properties":{"screen":{"type":"string","description":"Monitor name from list_screens"},` +
				`"all":{"type":"boolean","description":"Capture all monitors as one image"},` +
				`"output":{"type":"string","description":"Absolute path of the file to write instead of the library"}},"additionalProperties":false}`),
			Aufruf: an.nimmBildschirmAuf},
		{Name: "capture_repeat", Titel: "Capture the last area again",
			Beschreibung: "Captures the area the user selected last time again. Same consent, dialog and output as capture_screen.",
			Eingabe: schema(`{"type":"object","properties":{"output":{"type":"string","description":"Absolute path of the file to write instead of the library"}},` +
				`"additionalProperties":false}`),
			Aufruf: an.nimmLetztenAuf},
		{Name: "library_search", Titel: "Search the library",
			Beschreibung: "Lists screenshots in the Schnapp library, newest first. Needs consent ([agents] library = yes).",
			Eingabe: schema(`{"type":"object","properties":{"tag":{"type":"string"},` +
				`"from":{"type":"string","description":"RFC 3339 time or YYYY-MM-DD, inclusive"},` +
				`"to":{"type":"string","description":"RFC 3339 time or YYYY-MM-DD, exclusive"},` +
				`"limit":{"type":"integer","minimum":1,"maximum":50},` +
				`"agents":{"type":"boolean","description":"Include screenshots taken for agents (default true)"}},"additionalProperties":false}`),
			Aufruf: an.sucheInBibliothek},
		{Name: "library_read", Titel: "Read a screenshot from the library",
			Beschreibung: "Returns the visible image of a library screenshot (path from library_search). Needs consent ([agents] library = yes).",
			Eingabe:      schema(`{"type":"object","properties":{"path":{"type":"string"}},"required":["path"],"additionalProperties":false}`),
			Aufruf:       an.liesAusBibliothek},
		{Name: "export", Titel: "Export a file",
			Beschreibung: "Writes the visible image of a Schnapp file, PNG or JPEG (file) to target (both absolute paths; " +
				"target ends in .png, .jpg or .svg). A file from the library needs consent ([agents] library = yes).",
			Eingabe: schema(`{"type":"object","properties":{"file":{"type":"string"},"target":{"type":"string"}},` +
				`"required":["file","target"],"additionalProperties":false}`),
			Aufruf: an.exportiere},
		{Name: "annotate", Titel: "Annotate or redact a file",
			Beschreibung: "Adds arrows, rectangles, text and step numbers to a Schnapp file, PNG or JPEG (file, absolute " +
				"path) and redacts or pixelates areas for good, then saves like the Schnapp editor: a library screenshot " +
				"is updated, any other file stays unchanged and the result becomes a new library screenshot. Coordinates " +
				"are pixels of the visible image (width and height as export reports them). All or nothing: one invalid " +
				"entry saves nothing. Needs consent ([agents] edit = yes); a file from the library also needs " +
				"[agents] library = yes.",
			Eingabe: schema(`{"type":"object","properties":{"file":{"type":"string"},` +
				`"annotations":{"type":"array","items":{"type":"object","properties":{` +
				`"type":{"enum":["arrow","rect","text","step","redact","pixelate"]},` +
				`"from":{"type":"array","items":{"type":"number"},"minItems":2,"maxItems":2},` +
				`"to":{"type":"array","items":{"type":"number"},"minItems":2,"maxItems":2},` +
				`"at":{"type":"array","items":{"type":"number"},"minItems":2,"maxItems":2},` +
				`"tip":{"type":"array","items":{"type":"number"},"minItems":2,"maxItems":2,"description":"text only: makes a speech bubble pointing at this point"},` +
				`"text":{"type":"string"},"color":{"type":"string","description":"#RRGGBB or #RRGGBBAA"},` +
				`"fill":{"type":"string","description":"#RRGGBB, #RRGGBBAA or none"},` +
				`"width":{"type":"number"},"size":{"type":"number"},"radius":{"type":"number"},"font":{"type":"string"},` +
				`"bold":{"type":"boolean"},"italic":{"type":"boolean"},"underline":{"type":"boolean"},` +
				`"strikethrough":{"type":"boolean"},"align":{"enum":["left","center","right"]}},` +
				`"required":["type"],"additionalProperties":false}}},` +
				`"required":["file","annotations"],"additionalProperties":false}`),
			Aufruf: an.beschrifte},
	}
}

// --- Hilfen ---

// einstellungen lädt die Datei frisch (ein Widerruf wirkt sofort) und schreibt die Warnungen nach stderr.
func (an mcpAnschluesse) einstellungen() settings.Einstellungen {
	e, warn := an.lade()
	for _, w := range warn {
		fmt.Fprintln(an.fehler, w)
	}
	return e
}

func (an mcpAnschluesse) nichtErlaubt(was, schalter string) mcp.Ergebnis {
	return mcp.FehlerErgebnis(fmt.Sprintf("%s is not allowed. The user can allow it with `%s = yes` in section [agents] of %s.",
		was, schalter, an.einstellungsOrt))
}

// liesArgs dekodiert die Argumente streng: unbekannte Schlüssel sind ein Fehler, auch in anderer Schreibung
// ("FILE" statt "file"; encoding/json vergleicht ohne Rücksicht auf Groß- und Kleinschreibung), und kein Schlüssel darf
// doppelt stehen (Kritik #202 B-6). v ist ein Zeiger auf eine Struktur mit json-Namen.
func liesArgs(args json.RawMessage, v any) error {
	dec := json.NewDecoder(bytes.NewReader(args))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if err := annotate.KeineDoppelten(args); err != nil {
		return errors.New(strings.TrimPrefix(err.Error(), "invalid JSON: "))
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(args, &m) != nil {
		return nil // kein Objekt: Decode hat es schon beurteilt
	}
	namen := jsonNamen(v)
	for k := range m {
		if !namen[k] {
			return fmt.Errorf("json: unknown field %q", k)
		}
	}
	return nil
}

// jsonNamen liefert die JSON-Namen der Felder der Struktur, auf die v zeigt.
func jsonNamen(v any) map[string]bool {
	namen := map[string]bool{}
	t := reflect.TypeOf(v)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return namen
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		switch name {
		case "-":
			continue
		case "":
			name = f.Name
		}
		namen[name] = true
	}
	return namen
}

func falscheArgs(err error) mcp.Ergebnis {
	return mcp.FehlerErgebnis("Invalid arguments: " + err.Error())
}

// pruefeAusgabe prüft ein output/target: absolut, gültiges UTF-8, bekannte Endung.
func pruefeAusgabe(name, pfad string) error {
	switch {
	case !utf8.ValidString(pfad):
		return fmt.Errorf("%s must be valid UTF-8", name)
	case !filepath.IsAbs(pfad):
		return fmt.Errorf("%s must be an absolute path", name)
	}
	if _, ok := export.AusEndung(pfad); !ok {
		return fmt.Errorf("%s must end in .png, .jpg, .jpeg or .svg", name)
	}
	return nil
}

// aufgeloest löst die Symlinks des längsten vorhandenen Anfangs von p auf; der Rest bleibt, wie er ist.
func aufgeloest(p string) string {
	p = filepath.Clean(p)
	rest := ""
	for {
		if a, err := filepath.EvalSymlinks(p); err == nil {
			return filepath.Join(a, rest)
		}
		eltern := filepath.Dir(p)
		if eltern == p {
			return filepath.Join(p, rest)
		}
		rest = filepath.Join(filepath.Base(p), rest)
		p = eltern
	}
}

// liegtIn: p ist basis selbst oder liegt darunter (beide schon bereinigt).
func liegtIn(basis, p string) bool {
	rel, err := filepath.Rel(basis, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// pruefeZiel prüft ein output/target über pruefeAusgabe hinaus (ADR-0030 Nachtrag 4 Punkt 2, F-462): Es darf nicht in
// der Ablage und nicht im Vorschau-Cache liegen (der Ordner nach dem Auflösen seiner Symlinks) und es darf es noch
// nicht geben. Das Schreiben legt die Datei danach exklusiv an; diese Prüfung erspart nur Dialog und Aufnahme.
func (an mcpAnschluesse) pruefeZiel(name, pfad string) error {
	o, err := an.orte()
	if err != nil {
		return fmt.Errorf("%s cannot be checked because the location of the Schnapp library is unknown", name)
	}
	ordner, err := filepath.EvalSymlinks(filepath.Dir(pfad))
	if err != nil {
		return fmt.Errorf("the folder of %s does not exist", name)
	}
	echt := filepath.Join(ordner, filepath.Base(pfad))
	for _, b := range []string{o.Ablage, o.Vorschau} {
		if liegtIn(aufgeloest(b), echt) || liegtIn(filepath.Clean(b), filepath.Clean(pfad)) {
			return fmt.Errorf("%s must not be inside the Schnapp library or its preview cache", name)
		}
	}
	switch _, err := os.Lstat(pfad); {
	case err == nil:
		return fmt.Errorf("%s already exists; Schnapp never replaces a file for an agent", name)
	case !errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("%s cannot be checked", name)
	}
	return nil
}

// mcpSchreibFehler ist der englische Satz zu einem gescheiterten Schreiben von output/target.
func mcpSchreibFehler(err error) string {
	switch {
	case errors.Is(err, export.ErrZielVorhanden):
		return "the file already exists; Schnapp never replaces a file for an agent."
	case errors.Is(err, fileformat.ErrLupenZuGross):
		return "the magnifiers would embed too many pixels in an SVG; export as PNG or JPG instead."
	case errors.Is(err, fs.ErrPermission):
		return "no permission to write there."
	case errors.Is(err, fs.ErrNotExist):
		return "the folder does not exist."
	}
	return "the file could not be written."
}

// schreibeBildNeu schreibt das sichtbare Bild nach pfad wie schreibeBild, ersetzt aber nie ein vorhandenes Ziel
// (exklusiv angelegt; ADR-0030 Nachtrag 4 Punkt 2). privat: neue Datei mit 0600, sonst nach der umask.
func schreibeBildNeu(d *document.Dokument, pfad string, privat bool) (string, error) {
	f, ok := export.AusEndung(pfad)
	if !ok {
		return "", fmt.Errorf("schnapp: unbekannte Endung %q", filepath.Ext(pfad))
	}
	abs, err := filepath.Abs(pfad)
	if err != nil {
		return "", err
	}
	var inhalt bytes.Buffer
	if err := export.Schreibe(d, f, &inhalt); err != nil {
		return "", err
	}
	schreiber := export.SchreibeOhneErsetzen
	if privat {
		schreiber = export.SchreibeOhneErsetzenPrivat
	}
	return abs, schreibeInhaltMit(abs, inhalt.Bytes(), schreiber, nil)
}

// kundenname macht den vom Agenten gemeldeten Namen für den Dialog unschädlich: keine Steuerzeichen, gekürzt.
func kundenname(k mcp.Kunde) string { return maskiere(k.Name, maxKundenname) }

// maskiere macht einen Text des Agenten für Dialog und Hinweis unschädlich: gültiges UTF-8, keine Steuer- oder
// Formatzeichen (auch keine Zeilenumbrüche), höchstens max Zeichen (F-463).
func maskiere(s string, max int) string {
	n := strings.TrimSpace(s)
	if !utf8.ValidString(n) {
		n = strings.ToValidUTF8(n, "�")
	}
	var b strings.Builder
	for _, r := range n {
		if strconv.IsPrint(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune('�')
		}
	}
	n = b.String()
	if r := []rune(n); len(r) > max {
		n = string(r[:max]) + "…"
	}
	return n
}

// ergebnisMitBild baut die Antwort einer Aufnahme oder eines Lesens: structuredContent, dasselbe als Text und mit
// [agents] image = yes das sichtbare Bild als PNG (höchstens maxBildBytes).
func ergebnisMitBild(pfad string, sichtbar image.Image, mitBild bool) mcp.Ergebnis {
	g := sichtbar.Bounds().Size()
	e := ergebnis{Path: pfad, Width: g.X, Height: g.Y}
	text, _ := json.Marshal(e)
	res := mcp.Ergebnis{Inhalt: []mcp.Inhalt{mcp.TextInhalt(string(text))}, Strukturiert: e}
	if !mitBild {
		return res
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, sichtbar); err != nil {
		res.Inhalt = append(res.Inhalt, mcp.TextInhalt("The image could not be encoded; use the path."))
		return res
	}
	if buf.Len() > maxBildBytes {
		res.Inhalt = append(res.Inhalt, mcp.TextInhalt(fmt.Sprintf("The image is larger than %d MiB and is not included; use the path.", maxBildBytes>>20)))
		return res
	}
	res.Inhalt = append(res.Inhalt, mcp.BildInhalt(base64.StdEncoding.EncodeToString(buf.Bytes()), "image/png"))
	return res
}

// --- Werkzeuge ---

func (an mcpAnschluesse) listeMonitore(_ context.Context, args json.RawMessage, _ mcp.Kunde) mcp.Ergebnis {
	var leer struct{}
	if err := liesArgs(args, &leer); err != nil {
		return falscheArgs(err)
	}
	abfrage := an.monitoreListe
	if abfrage == nil {
		abfrage = an.monitore
	}
	ms, err := abfrage()
	if err != nil {
		return mcp.FehlerErgebnis("Could not query the monitors.")
	}
	liste := bildschirmeJSON(ms)
	text, _ := json.Marshal(liste)
	return mcp.Ergebnis{Inhalt: []mcp.Inhalt{mcp.TextInhalt(string(text))}, Strukturiert: liste}
}

// aufnahme beschreibt, was aufgenommen wird.
type aufnahme struct {
	monitor string // Name; leer mit alle=false und letzter=false: der aktive Monitor
	alle    bool
	letzter bool
}

// zielText ist die Beschreibung für Dialog und Hinweis (übersetzt).
func (a aufnahme) zielText() string {
	switch {
	case a.letzter:
		return i18n.T("the last selected area")
	case a.alle:
		return i18n.T("all monitors")
	case a.monitor != "":
		return fmt.Sprintf(i18n.T("monitor %s"), maskiere(a.monitor, maxMonitorname))
	}
	return i18n.T("the active monitor")
}

func (an mcpAnschluesse) nimmBildschirmAuf(ctx context.Context, args json.RawMessage, k mcp.Kunde) mcp.Ergebnis {
	var p struct {
		Screen string `json:"screen"`
		All    bool   `json:"all"`
		Output string `json:"output"`
	}
	if err := liesArgs(args, &p); err != nil {
		return falscheArgs(err)
	}
	if p.Screen != "" && p.All {
		return mcp.FehlerErgebnis("Give either screen or all, not both.")
	}
	a := aufnahme{monitor: p.Screen, alle: p.All}
	return an.nimmAuf(ctx, a, p.Output, k)
}

func (an mcpAnschluesse) nimmLetztenAuf(ctx context.Context, args json.RawMessage, k mcp.Kunde) mcp.Ergebnis {
	var p struct {
		Output string `json:"output"`
	}
	if err := liesArgs(args, &p); err != nil {
		return falscheArgs(err)
	}
	return an.nimmAuf(ctx, aufnahme{letzter: true}, p.Output, k)
}

// letzterHinweis ist der zuletzt gezeigte Hinweis des Servers: Kennung beim Benachrichtigungsdienst und Text (zum
// erneuten Zeigen).
type letzterHinweis struct {
	id   uint32
	text string
}

// vorigenHinweisSchliessen nimmt den Hinweis der letzten Aufnahme dieses Servers zurück und wartet, bis Plasma ihn
// ausgeblendet hat (F-490). Zurück kommt der geschlossene Hinweis (Kennung 0: keiner): Folgt kein neuer, zeigt
// hinweisZurueck ihn wieder (Sitzungs-QA B-1). Scheitert das Schließen, geht es weiter: Der alte Hinweis steht dann im
// Bild, verborgen wird nichts. Fehler nur bei Abbruch während des Wartens.
func (an mcpAnschluesse) vorigenHinweisSchliessen(ctx context.Context) (letzterHinweis, error) {
	if an.hinweis == nil || an.hinweis.id == 0 || an.schliesse == nil {
		return letzterHinweis{}, nil
	}
	alt := *an.hinweis
	if err := an.schliesse(ctx, alt.id); err != nil {
		diagnose(an.fehler, "voriger Hinweis nicht geschlossen: %v", err)
		// Abbruch oder Frist mitten im Aufruf: Ob der Dienst schon geschlossen hat, ist unbekannt. Dann gilt er als
		// geschlossen und kommt bei einem Fehlschlag wieder, doppelt ist die vorsichtige Richtung (Verify 2, Fund 2).
		// Nur eine Fehlerantwort des Dienstes heißt sicher „steht noch“: nichts wieder zeigen (N-3).
		if !(ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
			return letzterHinweis{}, nil
		}
	}
	*an.hinweis = letzterHinweis{}
	return alt, an.warte(ctx, hinweisWeg)
}

// hinweisZurueck zeigt einen geschlossenen Hinweis erneut, wenn die Aufnahme danach ohne eigenen Hinweis endete
// (abgebrochen, gescheitert): Ein Agent kann den Hinweis seiner vorigen Aufnahme so nicht durch einen Fehlversuch
// verschwinden lassen (Sitzungs-QA B-1, ADR-0030 Nachtrag 8). Eigener Kontext mit Frist, weil der Aufruf oft gerade
// abgebrochen wurde.
func (an mcpAnschluesse) hinweisZurueck(alt letzterHinweis) {
	if alt.id == 0 || an.hinweis == nil {
		return
	}
	ctx, ende := context.WithTimeout(context.Background(), 5*time.Second)
	defer ende()
	id, err := an.melde(ctx, i18n.T("Schnapp captured the screen for an agent"), alt.text)
	if err != nil {
		diagnose(an.fehler, "voriger Hinweis nicht wieder gezeigt: %v", err)
		return
	}
	*an.hinweis = letzterHinweis{id: id, text: alt.text}
}

// nimmAuf ist der gemeinsame Weg: Einwilligung, Bestätigung, Aufnahme, Hinweis, Ablage (ADR-0030 Nachtrag 2).
func (an mcpAnschluesse) nimmAuf(ctx context.Context, a aufnahme, ausgabe string, k mcp.Kunde) mcp.Ergebnis {
	e := an.einstellungen()
	if !e.Agenten.Aufnehmen {
		return an.nichtErlaubt("Capturing", "capture")
	}
	// Ohne Ziel gilt [capture] screen = all als „alle Monitore“, sonst der aktive Monitor (Nachtrag 2 Punkt 3).
	if !a.letzter && !a.alle && a.monitor == "" && e.Bildschirm == settings.WahlAlle {
		a.alle = true
	}
	// Nur ein Name aus der Monitorliste, vor dem Dialog (ADR-0030 Nachtrag 4 Punkt 3, F-463).
	if a.monitor != "" {
		ms, err := an.monitore()
		if err != nil {
			return mcp.FehlerErgebnis("Could not query the monitors; nothing was captured.")
		}
		if !slices.ContainsFunc(ms, func(m ui.Monitor) bool { return m.Name == a.monitor }) {
			return mcp.FehlerErgebnis("Unknown screen; use a name from list_screens.")
		}
	}
	if ausgabe != "" {
		if err := pruefeAusgabe("output", ausgabe); err != nil {
			return mcp.FehlerErgebnis(err.Error() + ".")
		}
		if err := an.pruefeZiel("output", ausgabe); err != nil {
			return mcp.FehlerErgebnis(err.Error() + ".")
		}
	}
	var letzter selection.Letzter
	if a.letzter {
		l, err := selection.LadeLetzten(an.letzterPfad)
		if err != nil {
			return mcp.FehlerErgebnis("There is no last area to repeat yet; the user has to select one first.")
		}
		letzter = l
	}
	if e.Agenten.Bestaetigen {
		ok, err := an.bestaetige(ctx, a.zielText(), kundenname(k))
		if err != nil {
			diagnose(an.fehler, "Bestätigung gescheitert: %v", err)
			if errors.Is(err, errFremdeSitzung) {
				return mcp.FehlerErgebnis("The confirmation dialog would not appear on the desktop that would be captured " +
					"(WAYLAND_DISPLAY and the session bus belong to different compositors); nothing was captured.")
			}
			return mcp.FehlerErgebnis("The confirmation dialog could not be shown; nothing was captured.")
		}
		if !ok {
			return mcp.FehlerErgebnis("The user did not allow the capture.")
		}
		if err := an.warte(ctx, fensterWeg); err != nil {
			return mcp.FehlerErgebnis("Cancelled; nothing was captured.")
		}
	}
	// Der Hinweis der vorigen Aufnahme stünde sonst in diesem Bild (F-490, Echtlauf 23.11.2026). Erst nach dem Dialog,
	// damit eine Ablehnung ihn nicht verschwinden lässt; endet die Aufnahme ohne neuen Hinweis, kommt er wieder (B-1).
	alt, err := an.vorigenHinweisSchliessen(ctx)
	neuerHinweis := false
	defer func() {
		if !neuerHinweis {
			an.hinweisZurueck(alt)
		}
	}()
	if err != nil {
		return mcp.FehlerErgebnis("Cancelled; nothing was captured.")
	}
	b, bereich, err := an.nimm(ctx, a, letzter)
	if err != nil {
		diagnose(an.fehler, "Aufnahme für einen Agenten: %v", err)
		if ctx.Err() != nil {
			return mcp.FehlerErgebnis("Cancelled; nothing was captured.")
		}
		return mcp.FehlerErgebnis("The screenshot could not be taken: " + agentGrund(err))
	}
	// Abbruch während der Aufnahme: das fertige Bild wird verworfen (ADR-0030 Nachtrag 4 Punkt 6, F-469).
	if ctx.Err() != nil {
		return mcp.FehlerErgebnis("Cancelled; the screenshot was discarded.")
	}
	d, err := document.Neu(b.SubImage(bereich))
	if err != nil {
		diagnose(an.fehler, "Dokument der Aufnahme: %v", err)
		return mcp.FehlerErgebnis("The screenshot could not be taken: " + agentGrund(err))
	}
	if err := d.VorgabeKante(e.Kante); err != nil {
		diagnose(an.fehler, "Kantenform %v nicht gesetzt: %v", e.Kante, err)
	}
	// Erst der Hinweis, dann die Ablage: Scheitert er, bleibt vom Bild nichts zurück (Punkt 8).
	id, err := an.melde(ctx, i18n.T("Schnapp captured the screen for an agent"), a.zielText())
	if err != nil {
		diagnose(an.fehler, "Hinweis gescheitert: %v", err)
		return mcp.FehlerErgebnis("The desktop notification could not be shown, so the screenshot was discarded.")
	}
	neuerHinweis = true
	if an.hinweis != nil {
		*an.hinweis = letzterHinweis{id: id, text: a.zielText()}
	}
	sichtbar, err := render.Bild(d)
	if err != nil {
		return mcp.FehlerErgebnis("The screenshot could not be rendered.")
	}
	// Letzte Gelegenheit vor dem Ablegen: Nach einem Abbruch bleibt vom Bild nichts zurück (F-469).
	if ctx.Err() != nil {
		return mcp.FehlerErgebnis("Cancelled; the screenshot was discarded.")
	}
	var pfad string
	if ausgabe != "" {
		pfad, err = schreibeBildNeu(d, ausgabe, true)
		switch {
		case errors.Is(err, errAbgebrochen):
			return mcp.FehlerErgebnis("Cancelled; the screenshot was discarded.")
		case errors.Is(err, errNachSchreiben):
			// Das Signal kam während des Schreibens: Die Datei ist vollständig, wird aber wie jede Aufnahme nach einem
			// Abbruch verworfen (Nachtrag 4 Punkt 6, F-482). Sie ist exklusiv neu angelegt, also unsere.
			return an.verwirfAusgabe(pfad)
		case err != nil:
			diagnose(an.fehler, "Schreiben nach output: %v", err)
			return mcp.FehlerErgebnis("Could not write " + ausgabe + ": " + mcpSchreibFehler(err))
		case ctx.Err() != nil: // Abbruch während des Schreibens ohne Signal (notifications/cancelled, EOF; F-483)
			return an.verwirfAusgabe(pfad)
		}
	} else {
		var ab agentAblage
		ab, err = an.legeAb(d, sichtbar)
		pfad = ab.abs
		if err != nil {
			diagnose(an.fehler, "Ablegen: %v", err)
			return mcp.FehlerErgebnis("Could not save the screenshot: " + agentGrund(err))
		}
		if ctx.Err() != nil { // Abbruch während des Ablegens (F-483)
			return an.verwirfAblage(ab)
		}
	}
	return ergebnisMitBild(pfad, sichtbar, e.Agenten.Bild)
}

// nimm führt die Aufnahme aus und liefert das Bild samt dem Bereich, der davon gilt.
func (an mcpAnschluesse) nimm(ctx context.Context, a aufnahme, l selection.Letzter) (*image.NRGBA, image.Rectangle, error) {
	q, err := an.quelle()
	if err != nil {
		return nil, image.Rectangle{}, err
	}
	defer q.Close()
	var b *capture.Bild
	switch {
	case a.letzter:
		b, err = q.Bildschirm(ctx, l.Monitor, capture.Optionen{})
		if err == nil && (b.Pixel.Bounds().Size() != l.Groesse || !l.SkalaPasst(b.Skala)) {
			return nil, image.Rectangle{}, errLetzterVeraltet
		}
		if err == nil {
			return b.Pixel, l.Bereich, nil
		}
	case a.alle:
		b, err = q.Alle(ctx, capture.Optionen{})
	case a.monitor != "":
		b, err = q.Bildschirm(ctx, a.monitor, capture.Optionen{})
	default:
		b, err = q.AktiverBildschirm(ctx, capture.Optionen{})
	}
	if err != nil {
		return nil, image.Rectangle{}, err
	}
	return b.Pixel, b.Pixel.Bounds(), nil
}

var errLetzterVeraltet = errors.New("schnapp: der Monitor des letzten Bereichs hat sich geändert")

// legeAb legt die Aufnahme unter agents/ ab und liefert den absoluten und den relativen Pfad.
// agentAblage ist eine eben abgelegte Agenten-Aufnahme: Orte, Pfade und der Stand der Datei direkt nach dem Ablegen
// (verwirfAblage löscht nur genau diese Datei, Sitzungs-QA A-5).
type agentAblage struct {
	orte     library.Orte
	abs, rel string
	stand    library.Stand
	standOK  bool
}

// legeAb legt die Aufnahme unter agents/ ab.
func (an mcpAnschluesse) legeAb(d *document.Dokument, sichtbar image.Image) (agentAblage, error) {
	o, err := an.orte()
	if err != nil {
		return agentAblage{}, err
	}
	e, err := library.Neu(o).AblegenAgent(d, sichtbar, fileformat.Metadaten{})
	if e.Pfad == "" {
		return agentAblage{}, err
	}
	if err != nil {
		diagnose(an.fehler, "Ablegen mit Folgefehler: %v", err) // Index/Vorschau/Temp-Rest: die Datei liegt vor
	}
	a := agentAblage{orte: o, abs: filepath.Join(o.Ablage, filepath.FromSlash(e.Pfad)), rel: e.Pfad}
	if st, err := library.StandVon(a.abs); err == nil {
		a.stand, a.standOK = st, true
	}
	return a, nil
}

// verwirfAusgabe entfernt eine nach dem Abbruch fertig gewordene Datei an output (F-482, F-483). Gelingt das nicht,
// sagt die Antwort, wo sie liegt.
func (an mcpAnschluesse) verwirfAusgabe(pfad string) mcp.Ergebnis {
	if err := entferneAusgabe(pfad); err != nil && !errors.Is(err, fs.ErrNotExist) {
		diagnose(an.fehler, "Verwerfen von output: %v", err)
		return mcp.FehlerErgebnis("Cancelled, but the screenshot could not be removed again: " + pfad)
	}
	return mcp.FehlerErgebnis("Cancelled; the screenshot was discarded.")
}

// entferneAusgabe ist os.Remove (Naht für Tests).
var entferneAusgabe = os.Remove

// verwirfAblage entfernt eine nach dem Abbruch abgelegte Aufnahme samt Vorschau und Index-Eintrag (F-483).
func (an mcpAnschluesse) verwirfAblage(ab agentAblage) mcp.Ergebnis {
	err := library.ErrGeaendert
	if ab.standOK { // nur mit dem Stand direkt nach dem Ablegen; eine inzwischen ersetzte Datei bleibt (A-5)
		_, err = library.Neu(ab.orte).Entfernen(ab.rel, ab.stand)
	}
	if err != nil {
		diagnose(an.fehler, "Verwerfen der Ablage: %v", err)
		return mcp.FehlerErgebnis("Cancelled, but the screenshot could not be removed from the library again: " + ab.abs)
	}
	return mcp.FehlerErgebnis("Cancelled; the screenshot was discarded.")
}

// agentGrund ist der englische Grund eines Fehlers für den Agenten (F-492): nie übersetzt, nie mit Pfad oder
// Paketpräfix; Unbekanntes steht nur auf stderr des Servers.
func agentGrund(err error) string {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "the file does not exist."
	case errors.Is(err, fs.ErrPermission):
		return "no permission."
	case errors.Is(err, syscall.ENOSPC):
		return "not enough disk space."
	case errors.Is(err, fileformat.ErrKeineDatei):
		return "this is not a regular file."
	case errors.Is(err, fileformat.ErrKeinPNG):
		return "this is neither a PNG nor a JPEG image."
	case errors.Is(err, fileformat.ErrZuGross):
		return "the image is too large."
	case errors.Is(err, fileformat.ErrBeschaedigt):
		return "the image is damaged."
	case errors.Is(err, render.ErrZuAufwendig):
		return "the file has too many or too large annotations."
	case errors.Is(err, library.ErrKeinHeim):
		return "the home directory is unknown."
	case errors.Is(err, errLetzterVeraltet):
		return "the monitor of the last area has changed; the user has to select the area again."
	}
	return "unexpected error (details on the standard error output of the server)."
}

// datum liest RFC 3339 oder JJJJ-MM-TT (lokale Mitternacht).
func datum(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.ParseInLocation("2006-01-02", s, time.Local)
}

func (an mcpAnschluesse) sucheInBibliothek(_ context.Context, args json.RawMessage, _ mcp.Kunde) mcp.Ergebnis {
	var p struct {
		Tag    string `json:"tag"`
		From   string `json:"from"`
		To     string `json:"to"`
		Limit  *int   `json:"limit"`
		Agents *bool  `json:"agents"`
	}
	if err := liesArgs(args, &p); err != nil {
		return falscheArgs(err)
	}
	if !an.einstellungen().Agenten.Bibliothek {
		return an.nichtErlaubt("Reading the library", "library")
	}
	anfrage := library.Anfrage{Schlagwort: p.Tag, OhneAgenten: p.Agents != nil && !*p.Agents}
	var err error
	if p.From != "" {
		if anfrage.Von, err = datum(p.From); err != nil {
			return mcp.FehlerErgebnis("from must be an RFC 3339 time or YYYY-MM-DD.")
		}
	}
	if p.To != "" {
		if anfrage.Bis, err = datum(p.To); err != nil {
			return mcp.FehlerErgebnis("to must be an RFC 3339 time or YYYY-MM-DD.")
		}
	}
	grenze := suchVoreinstellung
	if p.Limit != nil {
		if *p.Limit < 1 || *p.Limit > suchHoechstens {
			return mcp.FehlerErgebnis(fmt.Sprintf("limit must be between 1 and %d.", suchHoechstens))
		}
		grenze = *p.Limit
	}
	o, err := an.orte()
	if err != nil {
		return mcp.FehlerErgebnis("The library could not be found.")
	}
	treffer, err := library.Neu(o).Suchen(anfrage)
	if err != nil {
		diagnose(an.fehler, "Suche: %v", err)
		return mcp.FehlerErgebnis("The library could not be read.")
	}
	type eintrag struct {
		Path    string    `json:"path"`
		Created time.Time `json:"created"`
		Width   int       `json:"width"`
		Height  int       `json:"height"`
		Tags    []string  `json:"tags"`
		Agent   bool      `json:"agent"`
	}
	liste := struct {
		Entries []eintrag `json:"entries"`
		More    bool      `json:"more"`
	}{Entries: []eintrag{}, More: len(treffer) > grenze}
	for _, t := range treffer[:min(grenze, len(treffer))] {
		tags := t.Schlagworte
		if tags == nil {
			tags = []string{}
		}
		liste.Entries = append(liste.Entries, eintrag{Path: filepath.Join(o.Ablage, filepath.FromSlash(t.Pfad)),
			Created: t.Erstellt, Width: t.Breite, Height: t.Hoehe, Tags: tags, Agent: t.Agent()})
	}
	text, _ := json.Marshal(liste)
	return mcp.Ergebnis{Inhalt: []mcp.Inhalt{mcp.TextInhalt(string(text))}, Strukturiert: liste}
}

// inAblage löst pfad (absolut oder relativ zur Ablage) samt Symlinks auf und meldet, ob er in der Ablage liegt.
func inAblage(ablage, pfad string) (aufgeloest string, drin bool) {
	if !filepath.IsAbs(pfad) {
		pfad = filepath.Join(ablage, pfad)
	}
	a, err := filepath.EvalSymlinks(ablage)
	if err != nil {
		return "", false
	}
	p, err := filepath.EvalSymlinks(pfad)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(a, p)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return p, false
	}
	return p, true
}

// exportQuelleFehler ist die eine Antwort auf jede Quelle von export, die sich nicht öffnen lässt (F-493).
const exportQuelleFehler = "Cannot open the file: it does not exist, cannot be read or is not an image Schnapp can open."

// abgebrochenErgebnis ist die Antwort, wenn ctx zwischen zwei Schritten endete (F-467).
var abgebrochenErgebnis = mcp.FehlerErgebnis("Cancelled.")

func (an mcpAnschluesse) liesAusBibliothek(ctx context.Context, args json.RawMessage, _ mcp.Kunde) mcp.Ergebnis {
	var p struct {
		Path string `json:"path"`
	}
	if err := liesArgs(args, &p); err != nil {
		return falscheArgs(err)
	}
	e := an.einstellungen()
	if !e.Agenten.Bibliothek {
		return an.nichtErlaubt("Reading the library", "library")
	}
	if p.Path == "" {
		return mcp.FehlerErgebnis("path is missing.")
	}
	o, err := an.orte()
	if err != nil {
		return mcp.FehlerErgebnis("The library could not be found.")
	}
	pfad, drin := inAblage(o.Ablage, p.Path)
	if !drin {
		return mcp.FehlerErgebnis("path is not a screenshot in the Schnapp library.")
	}
	if fi, err := os.Stat(pfad); err != nil || !fi.Mode().IsRegular() || !strings.EqualFold(filepath.Ext(pfad), ".png") {
		return mcp.FehlerErgebnis("path is not a screenshot in the Schnapp library.")
	}
	_, g, _, err := ladeFuerEditor(pfad, false)
	if err != nil {
		diagnose(an.fehler, "Lesen aus der Bibliothek: %v", err)
		return mcp.FehlerErgebnis("Cannot open the screenshot: " + agentGrund(err))
	}
	if ctx.Err() != nil {
		return abgebrochenErgebnis
	}
	sichtbar, err := render.Bild(g.Dokument)
	if err != nil {
		return mcp.FehlerErgebnis("The screenshot could not be rendered.")
	}
	if ctx.Err() != nil {
		return abgebrochenErgebnis
	}
	return ergebnisMitBild(pfad, sichtbar, e.Agenten.Bild)
}

func (an mcpAnschluesse) exportiere(ctx context.Context, args json.RawMessage, _ mcp.Kunde) mcp.Ergebnis {
	var p struct {
		File   string `json:"file"`
		Target string `json:"target"`
	}
	if err := liesArgs(args, &p); err != nil {
		return falscheArgs(err)
	}
	if !filepath.IsAbs(p.File) {
		return mcp.FehlerErgebnis("file must be an absolute path.")
	}
	if err := pruefeAusgabe("target", p.Target); err != nil {
		return mcp.FehlerErgebnis(err.Error() + ".")
	}
	// Ohne bekannten Ort der Bibliothek lässt sich weder die Quelle noch das Ziel prüfen (Nachtrag 4 Punkt 2).
	o, err := an.orte()
	if err != nil {
		return mcp.FehlerErgebnis("The location of the Schnapp library is unknown, so export is not possible.")
	}
	// Ablage und Vorschau-Cache (verkleinerte Bibliotheksbilder) brauchen library = yes (F-465).
	for _, ort := range []string{o.Ablage, o.Vorschau} {
		// Auch dem Namen nach, ohne Auflösen: Sonst unterschiede die Antwort eine fehlende von einer vorhandenen Datei
		// in der Ablage (F-493).
		_, drin := inAblage(ort, p.File)
		if (drin || liegtIn(filepath.Clean(ort), filepath.Clean(p.File))) && !an.einstellungen().Agenten.Bibliothek {
			return an.nichtErlaubt("Reading the library", "library")
		}
	}
	if err := pruefeZielNichtQuelle(p.File, p.Target); err != nil {
		return mcp.FehlerErgebnis("Cannot export a file onto itself; choose a different target.")
	}
	if err := an.pruefeZiel("target", p.Target); err != nil {
		return mcp.FehlerErgebnis(err.Error() + ".")
	}
	_, g, _, err := ladeFuerEditor(p.File, false)
	if err != nil {
		// Ein Satz für jeden Grund: Die Antwort verrät nicht, ob es einen Pfad gibt und was er ist (F-493).
		diagnose(an.fehler, "Export, Quelle: %v", err)
		return mcp.FehlerErgebnis(exportQuelleFehler)
	}
	if ctx.Err() != nil {
		return abgebrochenErgebnis
	}
	abs, err := schreibeBildNeu(g.Dokument, p.Target, false)
	switch {
	case errors.Is(err, errAbgebrochen):
		return mcp.FehlerErgebnis("Cancelled before anything was written to " + p.Target + ".")
	case errors.Is(err, errNachSchreiben):
		// Ein Export ist keine Aufnahme: Die vollständige Datei bleibt, die Antwort sagt es (F-482).
		return mcp.FehlerErgebnis("Interrupted by a signal after " + p.Target + " was written completely.")
	}
	if err != nil {
		diagnose(an.fehler, "Export: %v", err)
		return mcp.FehlerErgebnis("Could not write " + p.Target + ": " + mcpSchreibFehler(err))
	}
	gr := g.Dokument.Groesse()
	e := ergebnis{Path: abs, Width: gr.X, Height: gr.Y}
	text, _ := json.Marshal(e)
	return mcp.Ergebnis{Inhalt: []mcp.Inhalt{mcp.TextInhalt(string(text))}, Strukturiert: e}
}

// beschriftenQuelleFehler ist wie bei export ein Satz für jeden Grund, aus dem die Datei nicht zu öffnen ist (F-493).
const beschriftenQuelleFehler = exportQuelleFehler

// beschrifte ist das Werkzeug annotate (ADR-0030 Nachtrag 10 Punkt 5): dieselbe Liste und Wirkung wie
// --annotate, mit der Einwilligung edit (und library für eine Datei aus der Ablage oder dem Vorschau-Cache).
func (an mcpAnschluesse) beschrifte(ctx context.Context, args json.RawMessage, _ mcp.Kunde) mcp.Ergebnis {
	var p struct {
		File        string          `json:"file"`
		Annotations json.RawMessage `json:"annotations"`
	}
	if err := liesArgs(args, &p); err != nil {
		return falscheArgs(err)
	}
	einst := an.einstellungen()
	if !einst.Agenten.Bearbeiten {
		return an.nichtErlaubt("Annotating and redacting files", "edit")
	}
	if !filepath.IsAbs(p.File) {
		return mcp.FehlerErgebnis("file must be an absolute path.")
	}
	o, err := an.orte()
	if err != nil {
		return mcp.FehlerErgebnis("The location of the Schnapp library is unknown, so annotate is not possible.")
	}
	// Wie export (F-465, F-493): Ablage und Vorschau-Cache brauchen library, auch dem Namen nach.
	for _, ort := range []string{o.Vorschau, o.Ablage} {
		_, drin := inAblage(ort, p.File)
		if (drin || liegtIn(filepath.Clean(ort), filepath.Clean(p.File))) && !einst.Agenten.Bibliothek {
			return an.nichtErlaubt("Reading the library", "library")
		}
	}
	if len(p.Annotations) == 0 {
		return mcp.FehlerErgebnis("annotations is missing.")
	}
	huelle, _ := json.Marshal(struct {
		Annotations json.RawMessage `json:"annotations"`
	}{p.Annotations})
	es, err := annotate.Lies(bytes.NewReader(huelle))
	if err != nil {
		return mcp.FehlerErgebnis(err.Error() + ". Nothing was saved.")
	}
	// Die Lage hält ein zweites Signal zurück, solange das Speichern läuft (mcpSignale, Kritik #202 B-2).
	lage := new(beschriftenLage)
	defer lage.ende()
	// Neue Aufnahmen eines Agenten kommen unter agents/ (F-699): Wer edit und library erlaubt, lässt den Agenten jede
	// Bilddatei lesen, die er beschriftet (docs/agents.md); so steht sie wenigstens sichtbar als Agenten-Aufnahme da.
	b, err := beschrifteDatei(ctx, p.File, es, o, einst.Kante, false, true, lage)
	// Gespeichert oder gescheitert: Ab hier hält nichts mehr ein Signal zurück, auch nicht ein Hinweis auf einem
	// blockierten stderr (F-603).
	lage.ende()
	var f *annotate.Fehler
	switch {
	case errors.As(err, &f):
		return mcp.FehlerErgebnis(err.Error() + ". Nothing was saved.")
	case errors.Is(err, errBeschriftenLesen):
		diagnose(an.fehler, "Beschriften, Quelle: %v", err)
		return mcp.FehlerErgebnis(beschriftenQuelleFehler)
	case err != nil && ctx.Err() != nil:
		// Nur ein gescheitertes Beschriften heißt „abgebrochen“: Gelang das Speichern, obwohl ctx inzwischen endete,
		// trägt die Datei den Stand, und die Antwort sagt es (Kritik #202 B-1).
		return abgebrochenErgebnis
	case err != nil:
		diagnose(an.fehler, "Beschriften: %v", err)
		return mcp.FehlerErgebnis("Nothing was saved: " + agentGrund(err))
	}
	if b.hinweis != "" {
		fmt.Fprintln(an.fehler, b.hinweis) // übersetzt, womöglich mit Ortsangabe: nur auf stderr des Servers
	}
	text, _ := json.Marshal(b.erg)
	return mcp.Ergebnis{Inhalt: []mcp.Inhalt{mcp.TextInhalt(string(text))}, Strukturiert: b.erg}
}

// --- Bestätigung im Kindprozess ---

// bestaetigung ist die Zeile, die der Server dem Dialog-Kind über stdin schickt.
type bestaetigung struct {
	Ziel  string `json:"target"`
	Kunde string `json:"client"`
	// Sprache des Servers (aus dem echten Heim, F-485); das Kind liest keine Einstellungen aus seiner Umgebung.
	Sprache string `json:"lang,omitempty"`
}

// mcpSprache ist die Sprache, die der Server beim Start gewählt hat (main.go); das Kind übernimmt sie (F-485).
var mcpSprache string

// setzeKindSprache lädt die Sprache des Servers im Kind; eine unbekannte bleibt ohne Wirkung (die Sprache aus der
// Umgebung gilt dann weiter).
func setzeKindSprache(sprache string, fehler io.Writer) {
	if sprache == "" {
		return
	}
	verfuegbar, err := i18n.Sprachen(po.Dateien)
	if err != nil || !slices.Contains(verfuegbar, sprache) {
		diagnose(fehler, "Sprache %q des Servers unbekannt", sprache)
		return
	}
	if k, err := i18n.Lade(po.Dateien, sprache); err == nil {
		i18n.Setze(k)
	}
}

// bestaetigeImKind startet `schnapp --internal-confirm-capture` und wartet höchstens bestaetigenFrist. Erlaubt ist nur
// Exit 0. Das Kind schreibt nie auf die stdout des Servers (dort steht das Protokoll).
func bestaetigeImKind(ctx context.Context, ziel, kunde string) (bool, error) {
	exe, err := kindProgramm()
	if err != nil {
		return false, err
	}
	ctx, ende := context.WithTimeout(ctx, bestaetigenFrist)
	defer ende()
	// Der Dialog erscheint nur auf dem Compositor, der auch aufnimmt (ADR-0030 Nachträge 5 und 6, F-491): Das Kind
	// bekommt die geprüfte Verbindung als Deskriptor 3 (WAYLAND_SOCKET), libwayland nimmt sie statt des Pfads.
	wl, err := pruefeSitzung(ctx, os.Getenv)
	if err != nil {
		return false, err
	}
	defer wl.Close()
	daten, _ := json.Marshal(bestaetigung{Ziel: ziel, Kunde: kunde, Sprache: mcpSprache})
	cmd := exec.CommandContext(ctx, exe, bestaetigenArgument)
	cmd.Stdin = bytes.NewReader(append(daten, '\n'))
	cmd.ExtraFiles = []*os.File{wl}
	cmd.Env = append(os.Environ(), "WAYLAND_SOCKET=3")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) } // eigenes Kind, über exec.Cmd
	cmd.WaitDelay = 2 * time.Second                                          // danach SIGKILL
	err = cmd.Run()
	if ctx.Err() != nil {
		return false, nil // Frist abgelaufen oder abgebrochen: abgelehnt
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return false, nil // Ablehnen, Esc, Schließen
	}
	return err == nil, err
}

// laufeBestaetigung ist die Seite des Kindes: liest die Angaben von stdin, zeigt den Dialog, Exit 0 nur bei „Allow“.
// Das Kind hat eine eigene Frist (bestaetigenFrist + kindZugabe): Stirbt der Server, bleibt kein Fenster stehen; kehrt
// der Dialog danach nicht zurück, beendet es sich nach kindHart hart (ADR-0030 Nachtrag 4 Punkt 5, F-469).
func laufeBestaetigung(ein io.Reader, fehler io.Writer) int {
	if err := probeGuard(); err != nil {
		fmt.Fprintln(fehler, "schnapp (Probe-Build):", err)
		return exitFehler
	}
	var b bestaetigung
	if err := json.NewDecoder(io.LimitReader(ein, 64<<10)).Decode(&b); err != nil || b.Ziel == "" {
		fmt.Fprintln(fehler, "schnapp: invalid confirmation request")
		return exitAufruf
	}
	setzeKindSprache(b.Sprache, fehler)
	ctx, ende := context.WithTimeout(context.Background(), bestaetigenFrist+kindZugabe)
	defer ende()
	hart := time.AfterFunc(bestaetigenFrist+kindZugabe+kindHart, func() {
		fmt.Fprintln(fehler, "schnapp: confirmation dialog did not close in time")
		beendeProzess(exitFehler)
	})
	defer hart.Stop()
	ok, err := zeigeDialog(ctx, maskiere(b.Ziel, maxZieltext), kundenname(mcp.Kunde{Name: b.Kunde}))
	if ctx.Err() != nil {
		return exitFehler // eigene Frist abgelaufen: abgelehnt, auch wenn der Dialog noch etwas meldete
	}
	if err != nil {
		diagnose(fehler, "Dialog: %v", err)
		return exitFehler
	}
	if !ok {
		return exitFehler
	}
	return exitOK
}
