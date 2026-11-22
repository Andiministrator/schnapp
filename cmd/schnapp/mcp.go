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
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/Andiministrator/schnapp/internal/capture"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/export"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/mcp"
	"github.com/Andiministrator/schnapp/internal/notify"
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

const (
	// bestaetigenFrist: so lange wartet der Server auf eine Antwort im Dialog, dann lehnt er ab.
	bestaetigenFrist = 60 * time.Second
	// fensterWeg: Abstand zwischen dem Ende des Dialogs und der Aufnahme, damit das Fenster nicht im Bild steht.
	fensterWeg = 300 * time.Millisecond
	// maxBildBytes: größtes PNG, das als Bildinhalt an den Agenten geht.
	maxBildBytes = 8 << 20
	// maxKundenname: so viele Zeichen des vom Agenten gemeldeten Namens zeigt der Dialog.
	maxKundenname = 60
	// Grenzen für library_search.
	suchVoreinstellung = 20
	suchHoechstens     = 50
)

// mcpAnschluesse sind die Außenanschlüsse des Servers; Tests setzen Attrappen ein.
type mcpAnschluesse struct {
	lade        Ladefunktion
	quelle      func() (capture.Quelle, error)
	monitore    func() ([]ui.Monitor, error)
	bestaetige  func(ctx context.Context, ziel, kunde string) (bool, error)
	melde       func(ctx context.Context, titel, text string) error
	orte        func() (library.Orte, error)
	letzterPfad string
	warte       func(ctx context.Context, d time.Duration) error
	// einstellungsOrt nennt die Datei in den Ablehnungstexten.
	einstellungsOrt string
	fehler          io.Writer
}

func echteMcpAnschluesse(fehler io.Writer) mcpAnschluesse {
	ort, err := settings.Pfad()
	if err != nil {
		ort = "~/.config/schnapp/settings.ini"
	}
	return mcpAnschluesse{
		lade:        settings.Lade,
		quelle:      func() (capture.Quelle, error) { return capture.NeuKWin() },
		monitore:    ui.Monitore,
		bestaetige:  bestaetigeImKind,
		melde:       notify.Zeige,
		orte:        func() (library.Orte, error) { return library.OrteAus(os.Getenv) },
		letzterPfad: selection.LetzterPfad(os.Getenv),
		warte: func(ctx context.Context, d time.Duration) error {
			select {
			case <-time.After(d):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
		einstellungsOrt: ort,
		fehler:          fehler,
	}
}

// laufeMcp bedient einen Client über stdin/stdout, bis stdin endet oder ein Signal kommt.
func laufeMcp(ein io.Reader, aus, fehler io.Writer, an mcpAnschluesse) int {
	if err := probeGuard(); err != nil {
		fmt.Fprintln(fehler, "schnapp (Probe-Build):", err)
		return exitFehler
	}
	ctx, ende := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	defer ende()
	s := &mcp.Server{Name: "schnapp", Version: versionsText(), Log: fehler, Werkzeuge: an.werkzeuge(),
		Anleitung: "Schnapp takes screenshots on a KDE Plasma (Wayland) desktop. Capturing and reading the library " +
			"need the user's consent in the settings file; a denied call says which switch is missing."}
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

// liesArgs dekodiert die Argumente streng: unbekannte Schlüssel sind ein Fehler.
func liesArgs(args json.RawMessage, v any) error {
	dec := json.NewDecoder(bytes.NewReader(args))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
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

// kundenname macht den vom Agenten gemeldeten Namen für den Dialog unschädlich: keine Steuerzeichen, gekürzt.
func kundenname(k mcp.Kunde) string {
	n := strings.TrimSpace(k.Name)
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
	if r := []rune(n); len(r) > maxKundenname {
		n = string(r[:maxKundenname]) + "…"
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
	ms, err := an.monitore()
	if err != nil {
		return mcp.FehlerErgebnis("Could not query the monitors.")
	}
	type bildschirm struct {
		Name   string `json:"name"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	}
	liste := struct {
		Screens []bildschirm `json:"screens"`
	}{Screens: []bildschirm{}}
	for _, m := range ms {
		liste.Screens = append(liste.Screens, bildschirm{m.Name, m.Breite, m.Hoehe})
	}
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
		return fmt.Sprintf(i18n.T("monitor %s"), a.monitor)
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
	if ausgabe != "" {
		if err := pruefeAusgabe("output", ausgabe); err != nil {
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
			return mcp.FehlerErgebnis("The confirmation dialog could not be shown; nothing was captured.")
		}
		if !ok {
			return mcp.FehlerErgebnis("The user did not allow the capture.")
		}
		if err := an.warte(ctx, fensterWeg); err != nil {
			return mcp.FehlerErgebnis("Cancelled; nothing was captured.")
		}
	}
	b, bereich, err := an.nimm(ctx, a, letzter)
	if err != nil {
		diagnose(an.fehler, "Aufnahme für einen Agenten: %v", err)
		if ctx.Err() != nil {
			return mcp.FehlerErgebnis("Cancelled; nothing was captured.")
		}
		return mcp.FehlerErgebnis("The screenshot could not be taken: " + nutzerFehler(err))
	}
	d, err := document.Neu(b.SubImage(bereich))
	if err != nil {
		return mcp.FehlerErgebnis("The screenshot could not be taken: " + nutzerFehler(err))
	}
	if err := d.VorgabeKante(e.Kante); err != nil {
		diagnose(an.fehler, "Kantenform %v nicht gesetzt: %v", e.Kante, err)
	}
	// Erst der Hinweis, dann die Ablage: Scheitert er, bleibt vom Bild nichts zurück (Punkt 8).
	if err := an.melde(ctx, i18n.T("Schnapp captured the screen for an agent"), a.zielText()); err != nil {
		diagnose(an.fehler, "Hinweis gescheitert: %v", err)
		return mcp.FehlerErgebnis("The desktop notification could not be shown, so the screenshot was discarded.")
	}
	sichtbar, err := render.Bild(d)
	if err != nil {
		return mcp.FehlerErgebnis("The screenshot could not be rendered.")
	}
	var pfad string
	if ausgabe != "" {
		pfad, err = schreibeBild(d, ausgabe, true)
		if err != nil {
			diagnose(an.fehler, "Schreiben nach output: %v", err)
			return mcp.FehlerErgebnis("Could not write " + ausgabe + ": " + schreibFehler(err))
		}
	} else {
		pfad, err = an.legeAb(d, sichtbar)
		if err != nil {
			diagnose(an.fehler, "Ablegen: %v", err)
			return mcp.FehlerErgebnis("Could not save the screenshot: " + speicherFehler(err).Error())
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

func (an mcpAnschluesse) legeAb(d *document.Dokument, sichtbar image.Image) (string, error) {
	o, err := an.orte()
	if err != nil {
		return "", err
	}
	e, err := library.Neu(o).AblegenAgent(d, sichtbar, fileformat.Metadaten{})
	if e.Pfad == "" {
		return "", err
	}
	if err != nil {
		diagnose(an.fehler, "Ablegen mit Folgefehler: %v", err) // Index/Vorschau/Temp-Rest: die Datei liegt vor
	}
	return filepath.Join(o.Ablage, filepath.FromSlash(e.Pfad)), nil
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

func (an mcpAnschluesse) liesAusBibliothek(_ context.Context, args json.RawMessage, _ mcp.Kunde) mcp.Ergebnis {
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
		return mcp.FehlerErgebnis("Cannot open the screenshot: " + nutzerFehler(err))
	}
	sichtbar, err := render.Bild(g.Dokument)
	if err != nil {
		return mcp.FehlerErgebnis("The screenshot could not be rendered.")
	}
	return ergebnisMitBild(pfad, sichtbar, e.Agenten.Bild)
}

func (an mcpAnschluesse) exportiere(_ context.Context, args json.RawMessage, _ mcp.Kunde) mcp.Ergebnis {
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
	if o, err := an.orte(); err == nil {
		if _, drin := inAblage(o.Ablage, p.File); drin && !an.einstellungen().Agenten.Bibliothek {
			return an.nichtErlaubt("Reading the library", "library")
		}
	} else if !an.einstellungen().Agenten.Bibliothek {
		// Ohne bekannten Ort der Ablage lässt sich nicht ausschließen, dass die Datei dort liegt.
		return an.nichtErlaubt("Reading the library", "library")
	}
	if err := pruefeZielNichtQuelle(p.File, p.Target); err != nil {
		return mcp.FehlerErgebnis("Cannot export a file onto itself; choose a different target.")
	}
	_, g, _, err := ladeFuerEditor(p.File, false)
	if err != nil {
		return mcp.FehlerErgebnis("Cannot open the file: " + nutzerFehler(err))
	}
	abs, err := schreibeBild(g.Dokument, p.Target, false)
	if err != nil {
		diagnose(an.fehler, "Export: %v", err)
		return mcp.FehlerErgebnis("Could not write " + p.Target + ": " + schreibFehler(err))
	}
	gr := g.Dokument.Groesse()
	e := ergebnis{Path: abs, Width: gr.X, Height: gr.Y}
	text, _ := json.Marshal(e)
	return mcp.Ergebnis{Inhalt: []mcp.Inhalt{mcp.TextInhalt(string(text))}, Strukturiert: e}
}

// --- Bestätigung im Kindprozess ---

// bestaetigung ist die Zeile, die der Server dem Dialog-Kind über stdin schickt.
type bestaetigung struct {
	Ziel  string `json:"target"`
	Kunde string `json:"client"`
}

// bestaetigeImKind startet `schnapp --internal-confirm-capture` und wartet höchstens bestaetigenFrist. Erlaubt ist nur
// Exit 0. Das Kind schreibt nie auf die stdout des Servers (dort steht das Protokoll).
func bestaetigeImKind(ctx context.Context, ziel, kunde string) (bool, error) {
	exe, err := os.Executable()
	if err != nil {
		return false, err
	}
	ctx, ende := context.WithTimeout(ctx, bestaetigenFrist)
	defer ende()
	daten, _ := json.Marshal(bestaetigung{Ziel: ziel, Kunde: kunde})
	cmd := exec.CommandContext(ctx, exe, bestaetigenArgument)
	cmd.Stdin = bytes.NewReader(append(daten, '\n'))
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
	ok, err := ui.Bestaetige(b.Ziel, kundenname(mcp.Kunde{Name: b.Kunde}))
	if err != nil {
		diagnose(fehler, "Dialog: %v", err)
		return exitFehler
	}
	if !ok {
		return exitFehler
	}
	return exitOK
}
