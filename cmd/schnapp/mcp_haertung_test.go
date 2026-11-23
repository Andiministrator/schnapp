// SPDX-License-Identifier: GPL-3.0-or-later

package main

// Tests zur Härtung des MCP-Servers (ADR-0030 Nachtrag 4, F-461 bis F-471), zum Teil aus den Proben der Sitzungs-QA
// Sprint 12 (Kritik A) übernommen.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/capture"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/mcp"
	"github.com/Andiministrator/schnapp/internal/settings"
	"github.com/Andiministrator/schnapp/internal/ui"
)

func (l *mcpLauf) rufeMit(t *testing.T, ctx context.Context, werkzeug, args string, k mcp.Kunde) mcp.Ergebnis {
	t.Helper()
	for _, w := range l.an.werkzeuge() {
		if w.Name == werkzeug {
			return w.Aufruf(ctx, json.RawMessage(args), k)
		}
	}
	t.Fatalf("Werkzeug %s fehlt", werkzeug)
	return mcp.Ergebnis{}
}

func schreibeText(t *testing.T, pfad, inhalt string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(pfad), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pfad, []byte(inhalt), 0o600); err != nil {
		t.Fatal(err)
	}
}

// --- F-461: Einwilligung und Orte aus dem echten Heim ---

// heimLauf baut einen Lauf, dessen Einstellungen und Orte über mitHeim aus heim kommen; die Umgebung zeigt auf fremd,
// wo eine Einstellungsdatei alles erlaubt.
func heimLauf(t *testing.T, heimIni string) (l *mcpLauf, heim, fremd string) {
	t.Helper()
	l = neuerMcpLauf(t)
	heim, fremd = t.TempDir(), t.TempDir()
	alles := "[agents]\ncapture = yes\nlibrary = yes\nconfirm = no\n"
	schreibeText(t, filepath.Join(fremd, ".config", "schnapp", "settings.ini"), alles)
	schreibeText(t, filepath.Join(fremd, ".config", "user-dirs.dirs"), `XDG_PICTURES_DIR="`+fremd+`/Bilder"`+"\n")
	if heimIni != "" {
		schreibeText(t, filepath.Join(heim, ".config", "schnapp", "settings.ini"), heimIni)
	}
	t.Setenv("HOME", fremd)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(fremd, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(fremd, "daten"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(fremd, "cache"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(fremd, "zustand"))
	l.an.mitHeim(heim, os.Getenv)
	return l, heim, fremd
}

// TestMcpEinwilligungNichtAusDerUmgebung: Die Umgebung bringt capture = yes mit, das echte Heim nicht: abgelehnt.
func TestMcpEinwilligungNichtAusDerUmgebung(t *testing.T) {
	l, heim, _ := heimLauf(t, "")
	for _, w := range []string{"capture_screen", "capture_repeat", "library_search"} {
		if e := l.rufe(t, w, `{}`); !e.Fehler || !strings.Contains(text(e), "yes") {
			t.Errorf("%s mit Einwilligung nur in der Umgebung: %+v", w, e)
		}
	}
	if len(l.schritte) != 0 {
		t.Errorf("Schritte: %v", l.schritte)
	}
	// Die Ablehnung nennt den Ort ohne Heimpfad (Benutzername, Sitzungs-QA A-4).
	if want := "~/.config/schnapp/settings.ini"; l.an.einstellungsOrt != want || strings.Contains(l.an.einstellungsOrt, heim) {
		t.Errorf("Ablehnung nennt %s, erwartet %s", l.an.einstellungsOrt, want)
	}
	for _, v := range mcpUmgebung {
		if !strings.Contains(l.an.startWarnung, v) {
			t.Errorf("Warnung nennt %s nicht: %q", v, l.an.startWarnung)
		}
	}
}

// TestMcpAblageImEchtenHeim: Mit Einwilligung im echten Heim liegt die Aufnahme dort, nicht im Bilderordner der
// Umgebung; der letzte Bereich kommt ebenfalls von dort.
func TestMcpAblageImEchtenHeim(t *testing.T) {
	l, heim, fremd := heimLauf(t, "[agents]\ncapture = yes\nconfirm = no\n")
	e := l.rufe(t, "capture_screen", `{}`)
	if e.Fehler {
		t.Fatal(text(e))
	}
	p := e.Strukturiert.(ergebnis).Path
	if !strings.HasPrefix(p, filepath.Join(heim, "Pictures", "schnapp", "agents")+string(filepath.Separator)) {
		t.Errorf("Aufnahme nicht im echten Heim: %s", p)
	}
	if n := pngsIn(t, fremd); len(n) != 0 {
		t.Errorf("Dateien im Ort der Umgebung: %v", n)
	}
	if !strings.HasPrefix(l.an.letzterPfad, heim+string(filepath.Separator)) {
		t.Errorf("letzter Bereich aus der Umgebung: %s", l.an.letzterPfad)
	}
}

// TestMcpKeineWarnungBeiPassenderUmgebung: Zeigt die Umgebung auf dieselben Orte, gibt es keine Warnung.
func TestMcpKeineWarnungBeiPassenderUmgebung(t *testing.T) {
	heim := t.TempDir()
	umg := map[string]string{"HOME": heim + "/", "XDG_CONFIG_HOME": filepath.Join(heim, ".config")}
	var an mcpAnschluesse
	an.mitHeim(heim, func(k string) string { return umg[k] })
	if an.startWarnung != "" {
		t.Errorf("Warnung: %q", an.startWarnung)
	}
}

// TestEchteMcpAnschluesseNutzenBenutzerHeim: echteMcpAnschluesse nimmt das Heim aus der Benutzerdatenbank (Naht);
// ist es unbekannt, gelten die Voreinstellungen, und kein Ort ist bekannt.
func TestEchteMcpAnschluesseNutzenBenutzerHeim(t *testing.T) {
	heim := t.TempDir()
	alt := benutzerHeim
	t.Cleanup(func() { benutzerHeim = alt })
	t.Setenv("HOME", t.TempDir())
	benutzerHeim = func() (string, error) { return heim, nil }
	var fehler bytes.Buffer
	an := echteMcpAnschluesse(&fehler)
	o, err := an.orte()
	if err != nil || !strings.HasPrefix(o.Ablage, heim) || !strings.HasPrefix(o.Vorschau, heim) {
		t.Errorf("Orte: %+v, %v", o, err)
	}
	if !strings.Contains(an.startWarnung, "HOME") {
		t.Errorf("keine Warnung über HOME: %q", an.startWarnung)
	}

	schreibeText(t, filepath.Join(os.Getenv("HOME"), ".config", "schnapp", "settings.ini"), "[agents]\ncapture = yes\n")
	benutzerHeim = func() (string, error) { return "", errors.New("unbekannt") }
	an = echteMcpAnschluesse(&fehler)
	if e, _ := an.lade(); e.Agenten.Aufnehmen || e.Agenten.Bibliothek {
		t.Errorf("ohne Heim trotzdem erlaubt: %+v", e.Agenten)
	}
	if _, err := an.orte(); err == nil {
		t.Error("ohne Heim ein Ort")
	}
	if an.startWarnung == "" {
		t.Error("keine Warnung ohne Heim")
	}
}

// --- F-462, F-465: Ziele ---

// TestMcpOutputNieInDieAblage: output in Ablage oder Vorschau-Cache, auch über einen Link auf den Ordner, wird ohne
// Dialog abgelehnt; nichts entsteht (Probe A4 der Kritik).
func TestMcpOutputNieInDieAblage(t *testing.T) {
	l := neuerMcpLauf(t)
	monat := filepath.Join(l.orte.Ablage, "2026-11")
	if err := os.MkdirAll(monat, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(l.orte.Vorschau, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(l.heim, "verweis")
	if err := os.Symlink(monat, link); err != nil {
		t.Fatal(err)
	}
	for _, ziel := range []string{
		filepath.Join(monat, "2026-11-22_10-00-00.png"), filepath.Join(l.orte.Ablage, "a.png"),
		filepath.Join(l.orte.Vorschau, "v.png"), filepath.Join(link, "b.png"),
	} {
		e := l.rufe(t, "capture_screen", `{"output":"`+ziel+`"}`)
		if !e.Fehler || !strings.Contains(text(e), "library") {
			t.Errorf("%s: %+v", ziel, e)
		}
		if _, err := os.Lstat(ziel); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s entstand", ziel)
		}
	}
	if len(l.schritte) != 0 {
		t.Errorf("Dialog oder Aufnahme trotz verbotenem Ziel: %v", l.schritte)
	}
}

// TestMcpVorhandenesZielBleibt: Weder output noch target ersetzen eine vorhandene Datei (Probe A1 der Kritik).
func TestMcpVorhandenesZielBleibt(t *testing.T) {
	l := neuerMcpLauf(t)
	foto := filepath.Join(l.heim, "urlaub.jpg")
	schreibeText(t, foto, "unersetzliches Foto")
	aussen := filepath.Join(l.heim, "aussen.png")
	if err := os.WriteFile(aussen, mustPNG(t), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(l.heim, "link.png")
	if err := os.Symlink(foto, link); err != nil {
		t.Fatal(err)
	}
	for _, ziel := range []string{foto, link} {
		if e := l.rufe(t, "capture_screen", `{"output":"`+ziel+`"}`); !e.Fehler || !strings.Contains(text(e), "already exists") {
			t.Errorf("output %s: %+v", ziel, e)
		}
		if e := l.rufe(t, "export", `{"file":"`+aussen+`","target":"`+ziel+`"}`); !e.Fehler || !strings.Contains(text(e), "already exists") {
			t.Errorf("target %s: %+v", ziel, e)
		}
	}
	if b, _ := os.ReadFile(foto); string(b) != "unersetzliches Foto" {
		t.Errorf("Ziel verändert: %q", b)
	}
	if len(l.schritte) != 0 {
		t.Errorf("Dialog oder Aufnahme trotz vorhandenem Ziel: %v", l.schritte)
	}
}

// TestMcpSchreibenErsetztNie: Entsteht das Ziel erst nach der Prüfung (zwischen Dialog und Schreiben), ersetzt das
// Schreiben es trotzdem nicht.
func TestMcpSchreibenErsetztNie(t *testing.T) {
	l := neuerMcpLauf(t)
	ziel := filepath.Join(l.heim, "spaet.png")
	l.an.bestaetige = func(context.Context, string, string) (bool, error) {
		schreibeText(t, ziel, "kam dazwischen")
		return true, nil
	}
	e := l.rufe(t, "capture_screen", `{"output":"`+ziel+`"}`)
	if !e.Fehler || !strings.Contains(text(e), "already exists") {
		t.Errorf("%+v", e)
	}
	if b, _ := os.ReadFile(ziel); string(b) != "kam dazwischen" {
		t.Errorf("Ziel ersetzt: %q", b)
	}
	aussen := filepath.Join(l.heim, "aussen.png")
	if err := os.WriteFile(aussen, mustPNG(t), 0o600); err != nil {
		t.Fatal(err)
	}
	neu := filepath.Join(l.heim, "export.png")
	if e := l.rufe(t, "export", `{"file":"`+aussen+`","target":"`+neu+`"}`); e.Fehler {
		t.Fatalf("Export in eine neue Datei: %s", text(e))
	}
	if fi, err := os.Stat(neu); err != nil || fi.Size() == 0 {
		t.Errorf("Export: %v, %v", fi, err)
	}
}

// TestMcpExportAusVorschauBrauchtBibliothek: Vorschauen sind verkleinerte Bibliotheksbilder (Probe A5, F-465).
func TestMcpExportAusVorschauBrauchtBibliothek(t *testing.T) {
	l := neuerMcpLauf(t)
	vorschau := filepath.Join(l.orte.Vorschau, "ab.png")
	if err := os.MkdirAll(l.orte.Vorschau, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(vorschau, mustPNG(t), 0o600); err != nil {
		t.Fatal(err)
	}
	l.agenten = settings.Agenten{}
	ziel := filepath.Join(l.heim, "v.png")
	if e := l.rufe(t, "export", `{"file":"`+vorschau+`","target":"`+ziel+`"}`); !e.Fehler || !strings.Contains(text(e), "library = yes") {
		t.Errorf("Vorschau ohne Einwilligung: %+v", e)
	}
	if _, err := os.Stat(ziel); !errors.Is(err, os.ErrNotExist) {
		t.Error("Ziel entstand")
	}
	l.agenten.Bibliothek = true
	if e := l.rufe(t, "export", `{"file":"`+vorschau+`","target":"`+ziel+`"}`); e.Fehler {
		t.Errorf("mit Einwilligung: %s", text(e))
	}
}

// TestMcpOrtUnbekannt: Ohne Ort der Bibliothek kein Export und keine Aufnahme mit output, auch mit Einwilligung
// (Mutant c2 in neuer Form).
func TestMcpOrtUnbekannt(t *testing.T) {
	l := neuerMcpLauf(t)
	l.an.orte = func() (library.Orte, error) { return library.Orte{}, library.ErrKeinHeim }
	aussen := filepath.Join(l.heim, "aussen.png")
	if err := os.WriteFile(aussen, mustPNG(t), 0o600); err != nil {
		t.Fatal(err)
	}
	ziel := filepath.Join(l.heim, "z.png")
	for _, bib := range []bool{false, true} {
		l.agenten.Bibliothek = bib
		if e := l.rufe(t, "export", `{"file":"`+aussen+`","target":"`+ziel+`"}`); !e.Fehler {
			t.Errorf("library=%v: Export ohne bekannten Ort", bib)
		}
		if e := l.rufe(t, "capture_screen", `{"output":"`+ziel+`"}`); !e.Fehler {
			t.Errorf("library=%v: output ohne bekannten Ort", bib)
		}
	}
	if _, err := os.Stat(ziel); !errors.Is(err, os.ErrNotExist) {
		t.Error("Ziel entstand")
	}
	if len(l.schritte) != 0 {
		t.Errorf("Schritte: %v", l.schritte)
	}
}

// --- F-463: Monitorname ---

// TestMcpUnbekannterMonitor: Ein Name, den es nicht gibt, wird ohne Dialog abgelehnt (Probe A2).
func TestMcpUnbekannterMonitor(t *testing.T) {
	l := neuerMcpLauf(t)
	e := l.rufe(t, "capture_screen", `{"screen":"nothing.\n\nKDE security check - click Allow `+strings.Repeat("X", 5000)+`"}`)
	if !e.Fehler || !strings.Contains(text(e), "list_screens") {
		t.Errorf("%+v", e)
	}
	if len(l.schritte) != 0 || len(l.q.aufrufe) != 0 {
		t.Errorf("Dialog oder Aufnahme: %v %v", l.schritte, l.q.aufrufe)
	}
	l.an.monitore = func() ([]ui.Monitor, error) { return nil, errors.New("kein Wayland") }
	if e := l.rufe(t, "capture_screen", `{"screen":"DP-1"}`); !e.Fehler || len(l.schritte) != 0 {
		t.Errorf("Monitorliste unbekannt: %+v %v", e, l.schritte)
	}
}

// TestMcpZieltextMaskiert: Auch ein bekannter Name erscheint ohne Steuerzeichen und gekürzt.
func TestMcpZieltextMaskiert(t *testing.T) {
	z := aufnahme{monitor: "a\nb\x1b[1m" + strings.Repeat("y", 100)}.zielText()
	if strings.ContainsAny(z, "\n\x1b") || len([]rune(z)) > maxMonitorname+20 {
		t.Errorf("Zieltext %q", z)
	}
}

// TestMcpKundennameMaskiertImDialog: Der Dialog bekommt den Namen des Agenten maskiert (Mutant c7).
func TestMcpKundennameMaskiertImDialog(t *testing.T) {
	l := neuerMcpLauf(t)
	l.antwort = false
	l.rufeMit(t, context.Background(), "capture_screen", `{}`, mcp.Kunde{Name: "x\x1b[2Jy\nz" + strings.Repeat("w", 80)})
	if len(l.schritte) != 1 || strings.ContainsAny(l.schritte[0], "\n\x1b") || strings.Count(l.schritte[0], "w") > maxKundenname {
		t.Errorf("Dialog: %q", l.schritte)
	}
}

// --- F-467, F-469: Abbruch ---

// TestMcpAbbruchVerwirftFertigeAufnahme: Kommt der Abbruch während oder nach der Aufnahme, wird nichts abgelegt und
// nichts geschrieben (Probe A6).
func TestMcpAbbruchVerwirftFertigeAufnahme(t *testing.T) {
	for _, wann := range []string{"aufnahme", "hinweis"} {
		for _, args := range []string{`{}`, `{"output":"OUT"}`} {
			l := neuerMcpLauf(t)
			l.agenten.Bestaetigen = false
			ziel := filepath.Join(l.heim, "aus.png")
			args = strings.ReplaceAll(args, "OUT", ziel)
			ctx, abbrechen := context.WithCancel(context.Background())
			// Nach dem Abbruch wird nicht einmal versucht zu schreiben oder abzulegen (F-469; das Entfernen nach
			// dem Schreiben, F-483, ist nur das Netz dahinter): weder die Orte der Ablage gefragt noch Bytes geschrieben.
			var versucht bool
			orte := l.an.orte
			l.an.orte = func() (library.Orte, error) {
				if ctx.Err() != nil {
					versucht = true
				}
				return orte()
			}
			setzeNaht(t, func(w io.Writer, inhalt []byte, _ func() bool) error {
				versucht = true
				_, err := w.Write(inhalt)
				return err
			})
			if wann == "aufnahme" {
				quelle := l.an.quelle
				l.an.quelle = func() (capture.Quelle, error) { abbrechen(); return quelle() }
			} else {
				melde := l.an.melde
				l.an.melde = func(c context.Context, ti, te string) (uint32, error) { abbrechen(); return melde(c, ti, te) }
			}
			e := l.rufeMit(t, ctx, "capture_screen", args, mcp.Kunde{})
			if !e.Fehler || !strings.Contains(text(e), "discarded") {
				t.Errorf("%s %s: %+v", wann, args, e)
			}
			if n := pngsIn(t, l.orte.Ablage); len(n) != 0 {
				t.Errorf("%s %s: abgelegt: %v", wann, args, n)
			}
			if _, err := os.Stat(ziel); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("%s %s: output geschrieben", wann, args)
			}
			if versucht {
				t.Errorf("%s %s: nach dem Abbruch noch geschrieben oder abgelegt", wann, args)
			}
			// Abbruch während der Aufnahme: nicht einmal ein Hinweis auf ein Bild, das verworfen wird.
			if wann == "aufnahme" && strings.Contains(strings.Join(l.schritte, ";"), "hinweis:") {
				t.Errorf("%s %s: Hinweis nach dem Abbruch: %v", wann, args, l.schritte)
			}
			abbrechen()
		}
	}
}

// TestMcpLesenUndExportNachAbbruch: library_read und export prüfen den Abbruch zwischen den Schritten.
func TestMcpLesenUndExportNachAbbruch(t *testing.T) {
	l := neuerMcpLauf(t)
	pfad := l.legeAb(t, false, "2026-11-01T10:00:00Z")
	ctx, abbrechen := context.WithCancel(context.Background())
	abbrechen()
	if e := l.rufeMit(t, ctx, "library_read", `{"path":"`+pfad+`"}`, mcp.Kunde{}); !e.Fehler || bild(t, e) != nil {
		t.Errorf("library_read nach Abbruch: %+v", e)
	}
	ziel := filepath.Join(l.heim, "x.png")
	if e := l.rufeMit(t, ctx, "export", `{"file":"`+pfad+`","target":"`+ziel+`"}`, mcp.Kunde{}); !e.Fehler {
		t.Errorf("export nach Abbruch: %+v", e)
	}
	if _, err := os.Stat(ziel); !errors.Is(err, os.ErrNotExist) {
		t.Error("Ziel trotz Abbruch geschrieben")
	}
}

// TestMcpZweitesSignal: Das erste Signal bricht sanft ab, das zweite beendet sofort mit Exit 1 (F-467).
func TestMcpZweitesSignal(t *testing.T) {
	kanal := make(chan os.Signal, 2)
	abgebrochen := make(chan struct{}, 2)
	beendet := make(chan int, 1)
	fertig := make(chan struct{})
	var fehler bytes.Buffer
	go mcpSignale(kanal, func() { abgebrochen <- struct{}{} }, func(c int) { beendet <- c }, &fehler, fertig)
	kanal <- syscall.SIGTERM
	select {
	case <-abgebrochen:
	case <-time.After(5 * time.Second):
		t.Fatal("erstes Signal bricht nicht ab")
	}
	select {
	case c := <-beendet:
		t.Fatalf("schon nach dem ersten Signal beendet (%d)", c)
	case <-time.After(50 * time.Millisecond):
	}
	kanal <- syscall.SIGINT
	select {
	case c := <-beendet:
		if c != exitFehler {
			t.Errorf("Exit %d, erwartet %d", c, exitFehler)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("zweites Signal beendet nicht")
	}
	close(fertig)
}

// --- F-471: bisher ungetestete Zweige ---

// TestMcpLesenNurRegulaerePNG: library_read nimmt nur reguläre .png-Dateien der Ablage (Mutant c1).
func TestMcpLesenNurRegulaerePNG(t *testing.T) {
	l := neuerMcpLauf(t)
	monat := filepath.Dir(l.legeAb(t, false, "2026-11-01T10:00:00Z"))
	dat := filepath.Join(l.orte.Ablage, "bild.dat")
	if err := os.WriteFile(dat, mustPNG(t), 0o600); err != nil {
		t.Fatal(err)
	}
	var j bytes.Buffer
	if err := jpeg.Encode(&j, testbild(4, 4, 1), nil); err != nil {
		t.Fatal(err)
	}
	jpg := filepath.Join(l.orte.Ablage, "bild.jpg")
	if err := os.WriteFile(jpg, j.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{dat, jpg, monat} {
		if e := l.rufe(t, "library_read", `{"path":"`+p+`"}`); !e.Fehler {
			t.Errorf("%s gelesen", filepath.Base(p))
		}
	}
}

// TestMcpBildGrenze: Ein PNG über maxBildBytes geht nicht als Bild an den Agenten (Mutant c6).
func TestMcpBildGrenze(t *testing.T) {
	alt := maxBildBytes
	maxBildBytes = 16
	t.Cleanup(func() { maxBildBytes = alt })
	l := neuerMcpLauf(t)
	l.agenten.Bestaetigen = false
	e := l.rufe(t, "capture_screen", `{}`)
	if e.Fehler || bild(t, e) != nil || !strings.Contains(text(e), "larger than") {
		t.Errorf("über der Grenze: %+v", e)
	}
}
