// SPDX-License-Identifier: GPL-3.0-or-later

package main

// Tests zu den Befunden der Kritik an Karte #202 (ADR-0030 Nachtrag 11): Abbruch nach dem Speichern (B-1), Signale
// während des Speicherns (B-2), Einwilligung und Grenzen (B-4), leere Liste (B-5), strenge Schlüssel (B-6), Lesefehler
// und Budget-Hinweis (B-7).

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"testing/iotest"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/mcp"
	"github.com/Andiministrator/schnapp/internal/render"
	"github.com/Andiministrator/schnapp/internal/saveprogress"
)

// TestMcpAnnotateAbbruchWaehrendSpeichern (B-1): Endet ctx, während schon gespeichert wird, ist die Datei geschwärzt
// gespeichert — die Antwort ist dann das Ergebnis, nicht „Cancelled.“.
func TestMcpAnnotateAbbruchWaehrendSpeichern(t *testing.T) {
	l := neuerMcpLauf(t)
	l.agenten.Bearbeiten = true
	drin := l.legeAb(t, false, "2026-11-01T10:00:00Z")
	vorher, _ := os.ReadFile(drin)
	ctx, abbrechen := context.WithCancel(context.Background())
	defer abbrechen()
	alt := saveprogress.SetzeHaken(func(s string) {
		if s == "start" {
			abbrechen()
		}
	})
	t.Cleanup(func() { saveprogress.SetzeHaken(alt) })
	e := l.rufeMit(t, ctx, "annotate", `{"file":"`+drin+`","annotations":[{"type":"redact","from":[0,0],"to":[4,4]}]}`, mcp.Kunde{Name: "x"})
	nachher, _ := os.ReadFile(drin)
	if bytes.Equal(vorher, nachher) {
		t.Fatal("nicht gespeichert — der Test prüft den Abbruch nach dem Speichern")
	}
	if e.Fehler || text(e) == "Cancelled." || !strings.Contains(text(e), `"redactions":1`) {
		t.Errorf("Antwort %q, aber die Datei ist geschwärzt gespeichert", text(e))
	}
}

// signalAttrappe ersetzt meldeAn/meldeAb und beendeProzess; kanal ist der Kanal der Wache, beendet zählt die
// Beendigungen.
type signalAttrappe struct {
	kanal   atomic.Pointer[chan<- os.Signal]
	beendet atomic.Int32
}

func neueSignalAttrappe(t *testing.T, beimAnmelden func(chan<- os.Signal)) *signalAttrappe {
	t.Helper()
	s := new(signalAttrappe)
	an, ab, ende := meldeAn, meldeAb, beendeProzess
	t.Cleanup(func() { meldeAn, meldeAb, beendeProzess = an, ab, ende })
	meldeAn = func(c chan<- os.Signal, _ ...os.Signal) {
		s.kanal.Store(&c)
		if beimAnmelden != nil {
			beimAnmelden(c)
		}
	}
	meldeAb = func(chan<- os.Signal) {}
	beendeProzess = func(int) { s.beendet.Add(1) }
	return s
}

func (s *signalAttrappe) sende(t *testing.T, n int) {
	t.Helper()
	c := s.kanal.Load()
	if c == nil {
		t.Fatal("Wache nicht angemeldet")
	}
	for i := 0; i < n; i++ {
		*c <- syscall.SIGTERM
	}
}

// vorschauen liest alle Vorschaubilder der Bibliothek.
func vorschauen(b *library.Bibliothek) map[string][]byte {
	m := map[string][]byte{}
	filepath.WalkDir(b.Orte.Vorschau, func(p string, de fs.DirEntry, err error) error {
		if err == nil && !de.IsDir() {
			m[p], _ = os.ReadFile(p)
		}
		return nil
	})
	return m
}

// helleAufnahme legt eine helle 400×300-Aufnahme in die Bibliothek und liefert ihren Pfad.
func helleAufnahme(t *testing.T, b *library.Bibliothek) string {
	t.Helper()
	bild := image.NewNRGBA(image.Rect(0, 0, 400, 300))
	for i := range bild.Pix {
		bild.Pix[i] = 0xF0
	}
	d, err := document.Neu(bild)
	if err != nil {
		t.Fatal(err)
	}
	sichtbar, err := render.Bild(d)
	if err != nil {
		t.Fatal(err)
	}
	e, err := b.Ablegen(d, sichtbar, fileformat.Metadaten{})
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
}

// TestAnnotateSignaleWaehrendSpeichern (B-2): Kommen zwei Signale nach dem Ersetzen der Datei, aber vor der neuen
// Vorschau, speichert --annotate zu Ende: Die ungeschwärzte Vorschau ist weg, das Ergebnis steht auf stdout, Exit 1.
func TestAnnotateSignaleWaehrendSpeichern(t *testing.T) {
	i18n.Setze(nil)
	b := eigenesHeim(t)
	pfad := helleAufnahme(t, b)
	alt := vorschauen(b)
	if len(alt) == 0 {
		t.Fatal("keine Vorschau")
	}
	s := neueSignalAttrappe(t, nil)
	h := saveprogress.SetzeHaken(func(schritt string) {
		if schritt == "ersetzt" {
			s.sende(t, 2)
			time.Sleep(100 * time.Millisecond) // die Wache nimmt beide an, während das Speichern noch läuft
		}
	})
	t.Cleanup(func() { saveprogress.SetzeHaken(h) })
	liste := schreibeListe(t, t.TempDir(), `{"annotations":[{"type":"redact","from":[0,0],"to":[400,300]}]}`)
	code, aus, fehler := fuehre(t, keineMonitore, "--annotate="+liste, pfad)
	if n := s.beendet.Load(); n != 0 {
		t.Fatalf("zweites Signal hat das laufende Speichern beendet (%d)", n)
	}
	if code != exitFehler || strings.TrimSpace(aus) != pfad || !strings.Contains(fehler, "written completely") {
		t.Errorf("Code %d, aus %q, fehler %q", code, aus, fehler)
	}
	for p, inhalt := range vorschauen(b) {
		if bytes.Equal(inhalt, alt[p]) {
			t.Errorf("die ungeschwärzte Vorschau %s ist geblieben", filepath.Base(p))
		}
	}
}

// TestAnnotateSignalVorDemSpeichern (B-2): Ein Signal vor dem Speichern speichert nichts, Exit 1.
func TestAnnotateSignalVorDemSpeichern(t *testing.T) {
	i18n.Setze(nil)
	b := eigenesHeim(t)
	pfad := helleAufnahme(t, b)
	vorher, _ := os.ReadFile(pfad)
	neueSignalAttrappe(t, func(c chan<- os.Signal) {
		c <- syscall.SIGINT
		for len(c) > 0 { // die Wache hat es angenommen
			time.Sleep(time.Millisecond)
		}
		time.Sleep(20 * time.Millisecond)
	})
	liste := schreibeListe(t, t.TempDir(), `{"annotations":[{"type":"redact","from":[0,0],"to":[400,300]}]}`)
	code, aus, fehler := fuehre(t, keineMonitore, "--annotate="+liste, pfad)
	if code != exitFehler || aus != "" || !strings.Contains(fehler, "nothing was saved") {
		t.Errorf("Code %d, aus %q, fehler %q", code, aus, fehler)
	}
	if nachher, _ := os.ReadFile(pfad); !bytes.Equal(vorher, nachher) {
		t.Error("trotz Signal gespeichert")
	}
}

// TestAnnotateHaengendesSpeichern (B-2): Meldet das Speichern länger als die Frist keinen Fortschritt, beendet ein
// weiteres Signal sofort (wie im Editor, F-283).
func TestAnnotateHaengendesSpeichern(t *testing.T) {
	i18n.Setze(nil)
	b := eigenesHeim(t)
	pfad := helleAufnahme(t, b)
	altFrist := beschriftenFrist
	beschriftenFrist = 20 * time.Millisecond
	t.Cleanup(func() { beschriftenFrist = altFrist })
	s := neueSignalAttrappe(t, nil)
	h := saveprogress.SetzeHaken(func(schritt string) {
		if schritt == "ersetzt" {
			time.Sleep(60 * time.Millisecond) // „hängt“: kein Fortschritt über die Frist
			s.sende(t, 2)
			for i := 0; i < 200 && s.beendet.Load() == 0; i++ {
				time.Sleep(5 * time.Millisecond)
			}
		}
	})
	t.Cleanup(func() { saveprogress.SetzeHaken(h) })
	liste := schreibeListe(t, t.TempDir(), `{"annotations":[{"type":"step","at":[10,10]}]}`)
	_, _, fehler := fuehre(t, keineMonitore, "--annotate="+liste, pfad)
	if s.beendet.Load() != 1 || !strings.Contains(fehler, "made no progress") {
		t.Errorf("hängendes Speichern: beendet %d, fehler %q", s.beendet.Load(), fehler)
	}
}

// TestMcpZweitesSignalWartetAufSpeichern (B-2): Läuft ein annotate-Speichern mit Fortschritt, beendet das zweite
// Signal den Server erst danach; hängt es über die Frist, sofort.
func TestMcpZweitesSignalWartetAufSpeichern(t *testing.T) {
	for _, haengt := range []bool{false, true} {
		altFrist := beschriftenFrist
		if haengt {
			beschriftenFrist = 30 * time.Millisecond
		}
		lage := new(beschriftenLage)
		lage.setze(1)
		kanal := make(chan os.Signal, 2)
		beendet := make(chan int, 1)
		fertig := make(chan struct{})
		var fehler bytes.Buffer
		go mcpSignale(kanal, func() {}, func(c int) { beendet <- c }, &fehler, fertig)
		kanal <- syscall.SIGTERM
		kanal <- syscall.SIGINT
		select {
		case <-beendet:
			if !haengt {
				t.Error("beendet, während das Speichern läuft")
			}
		case <-time.After(150 * time.Millisecond):
			if haengt {
				t.Error("hängendes Speichern hält das zweite Signal auf")
			}
		}
		lage.ende()
		if !haengt {
			select {
			case c := <-beendet:
				if c != exitFehler {
					t.Errorf("Exit %d", c)
				}
			case <-time.After(5 * time.Second):
				t.Error("nach dem Speichern nicht beendet")
			}
		}
		close(fertig)
		beschriftenFrist = altFrist
	}
}

// TestMcpAnnotateEinwilligungHaertung (B-4): Symlink, fehlende Datei in der Ablage, „..“ und der Vorschau-Cache
// brauchen library; jeder Lesefehler außerhalb ergibt denselben Satz; ein relativer Pfad wird abgelehnt; der Hinweis
// des Speicherns geht nicht an den Agenten, warning steht in der Antwort.
func TestMcpAnnotateEinwilligungHaertung(t *testing.T) {
	l := neuerMcpLauf(t)
	l.agenten.Bearbeiten = true
	drin := l.legeAb(t, false, "2026-11-01T10:00:00Z")
	vorher, _ := os.ReadFile(drin)
	l.agenten.Bibliothek = false
	liste := `,"annotations":[{"type":"redact","from":[0,0],"to":[4,4]}]}`
	braucht := func(name, datei string) {
		t.Helper()
		if e := l.rufe(t, "annotate", `{"file":"`+datei+`"`+liste); !e.Fehler || !strings.Contains(text(e), "library = yes") {
			t.Errorf("%s: %q", name, text(e))
		}
	}
	link := filepath.Join(l.heim, "link.png")
	if err := os.Symlink(drin, link); err != nil {
		t.Fatal(err)
	}
	braucht("Symlink auf Bibliotheksdatei", link)
	braucht("fehlende Datei in der Ablage", filepath.Join(l.orte.Ablage, "2026-11", "x.png"))
	rel, _ := filepath.Rel(l.orte.Ablage, drin)
	braucht("Punkt-Punkt", l.orte.Ablage+"/../"+filepath.Base(l.orte.Ablage)+"/"+rel)
	var vorschau string
	filepath.WalkDir(l.orte.Vorschau, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			vorschau = p
		}
		return nil
	})
	if vorschau == "" {
		t.Fatal("keine Vorschau")
	}
	braucht("Vorschau-Cache", vorschau)
	if n, _ := os.ReadFile(drin); !bytes.Equal(n, vorher) {
		t.Error("ohne library verändert")
	}
	txt := filepath.Join(l.heim, "a.txt")
	if err := os.WriteFile(txt, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	var saetze []string
	for _, f := range []string{filepath.Join(l.heim, "fehlt.png"), l.heim, txt} {
		saetze = append(saetze, text(l.rufe(t, "annotate", `{"file":"`+f+`"`+liste)))
	}
	if saetze[0] != beschriftenQuelleFehler || saetze[1] != saetze[0] || saetze[2] != saetze[0] {
		t.Errorf("Lesefehler unterscheidbar: %q", saetze)
	}
	t.Chdir(l.heim)
	if err := os.WriteFile(filepath.Join(l.heim, "rel.png"), mustPNG(t), 0o600); err != nil {
		t.Fatal(err)
	}
	if e := l.rufe(t, "annotate", `{"file":"rel.png"`+liste); !e.Fehler || !strings.Contains(text(e), "absolute") {
		t.Errorf("relativer Pfad: %q", text(e))
	}
	aussen := filepath.Join(l.heim, "aussen.png")
	if err := os.WriteFile(aussen, mustPNG(t), 0o600); err != nil {
		t.Fatal(err)
	}
	l.fehler.Reset()
	e := l.rufe(t, "annotate", `{"file":"`+aussen+`"`+liste)
	if e.Fehler || !strings.Contains(text(e), `"warning":true`) || strings.Contains(text(e), "unredacted") || strings.Contains(text(e), "Saved") {
		t.Errorf("Antwort mit Hinweis oder ohne warning: %q", text(e))
	}
	if !strings.Contains(l.fehler.String(), "unredacted") {
		t.Errorf("Hinweis fehlt auf stderr des Servers: %q", l.fehler.String())
	}
}

// TestMcpAnnotateRechenbudget (B-4, M3): Über dem Rechenbudget speichert annotate nichts, auch nicht als neue
// Aufnahme; der Agent kann die Grenze nicht abschalten.
func TestMcpAnnotateRechenbudget(t *testing.T) {
	l := neuerMcpLauf(t)
	l.agenten.Bearbeiten = true
	aussen := filepath.Join(l.heim, "gross.png")
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, 4000, 3000))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(aussen, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	var a strings.Builder
	a.WriteString(`{"file":"` + aussen + `","annotations":[`)
	for i := 0; i < 400; i++ {
		if i > 0 {
			a.WriteString(",")
		}
		a.WriteString(`{"type":"rect","from":[0,0],"to":[4000,3000],"fill":"#11223380","width":500}`)
	}
	a.WriteString("]}")
	if e := l.rufe(t, "annotate", a.String()); !e.Fehler || !strings.Contains(text(e), "Nothing was saved") {
		t.Errorf("über dem Budget: %q", text(e))
	}
	if ps := pngsIn(t, l.orte.Ablage); len(ps) != 0 {
		t.Errorf("über dem Budget abgelegt: %v", ps)
	}
}

// TestAnnotateLeereListe (B-5): Eine leere Liste ist ein Fehler (Exit 2, MCP „Nothing was saved“), es entsteht keine
// Aufnahme.
func TestAnnotateLeereListe(t *testing.T) {
	i18n.Setze(nil)
	b := eigenesHeim(t)
	dir := t.TempDir()
	quelle := filepath.Join(dir, "bild.png")
	var buf bytes.Buffer
	if err := png.Encode(&buf, testbild(40, 30, 7)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(quelle, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	liste := schreibeListe(t, dir, `{"annotations":[]}`)
	code, _, fehler := fuehre(t, keineMonitore, "--annotate="+liste, quelle)
	if es, _ := b.Suchen(library.Anfrage{}); code != exitAufruf || len(es) != 0 || !strings.Contains(fehler, "empty") {
		t.Errorf("leere Liste: Code %d, %d Aufnahmen angelegt, %q", code, len(es), fehler)
	}
	l := neuerMcpLauf(t)
	l.agenten.Bearbeiten = true
	if e := l.rufe(t, "annotate", `{"file":"`+quelle+`","annotations":[]}`); !e.Fehler || !strings.Contains(text(e), "Nothing was saved") {
		t.Errorf("MCP leere Liste: %q", text(e))
	}
	if ps := pngsIn(t, l.orte.Ablage); len(ps) != 0 {
		t.Errorf("MCP leere Liste abgelegt: %v", ps)
	}
}

// TestMcpArgumenteExakt (B-6): Schlüssel nur in genau ihrer Schreibung, keine doppelten — für alle Werkzeuge.
func TestMcpArgumenteExakt(t *testing.T) {
	l := neuerMcpLauf(t)
	l.agenten.Bearbeiten = true
	aussen := filepath.Join(l.heim, "aussen.png")
	if err := os.WriteFile(aussen, mustPNG(t), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, fall := range []struct{ werkzeug, args string }{
		{"annotate", `{"FILE":"` + aussen + `","annotations":[{"type":"step","at":[1,1]}]}`},
		{"annotate", `{"file":"` + aussen + `","Annotations":[{"type":"step","at":[1,1]}]}`},
		{"annotate", `{"file":"/x.png","file":"` + aussen + `","annotations":[{"type":"step","at":[1,1]}]}`},
		{"library_read", `{"PATH":"x.png"}`},
		{"export", `{"file":"` + aussen + `","target":"/a.png","target":"` + filepath.Join(l.heim, "b.png") + `"}`},
	} {
		if e := l.rufe(t, fall.werkzeug, fall.args); !e.Fehler || !strings.Contains(text(e), "Invalid arguments") {
			t.Errorf("%s %s: %q", fall.werkzeug, fall.args, text(e))
		}
	}
	if ps := pngsIn(t, l.orte.Ablage); len(ps) != 0 {
		t.Errorf("trotz ungültiger Argumente abgelegt: %v", ps)
	}
	if _, err := os.Stat(filepath.Join(l.heim, "b.png")); err == nil {
		t.Error("export trotz doppeltem target geschrieben")
	}
}

// TestAnnotateListeNichtLesbar (B-7): Ein Ordner als Liste oder ein E/A-Fehler auf stdin ist Exit 1, nicht 2.
func TestAnnotateListeNichtLesbar(t *testing.T) {
	i18n.Setze(nil)
	eigenesHeim(t)
	dir := t.TempDir()
	quelle := filepath.Join(dir, "bild.png")
	var buf bytes.Buffer
	if err := png.Encode(&buf, testbild(40, 30, 7)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(quelle, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, _, fehler := fuehre(t, keineMonitore, "--annotate="+dir, quelle); code != exitFehler || !strings.Contains(fehler, "Cannot read") {
		t.Errorf("Ordner als Liste: Code %d, %q", code, fehler)
	}
	alt := eingabe
	t.Cleanup(func() { eingabe = alt })
	eingabe = iotest.ErrReader(syscall.EIO)
	if code, _, fehler := fuehre(t, keineMonitore, "--annotate=-", quelle); code != exitFehler || !strings.Contains(fehler, "Cannot read") {
		t.Errorf("E/A-Fehler auf stdin: Code %d, %q", code, fehler)
	}
}

// TestAnnotateIgnoreLimitHinweis (B-7): --ignore-limit über dem Budget speichert mit einem Hinweis der Kommandozeile,
// nicht dem Satz des Editors („before closing“).
func TestAnnotateIgnoreLimitHinweis(t *testing.T) {
	i18n.Setze(nil)
	eigenesHeim(t)
	dir := t.TempDir()
	quelle := filepath.Join(dir, "bild.png")
	var buf bytes.Buffer
	if err := png.Encode(&buf, testbild(40, 30, 7)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(quelle, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	// Viele Zeichen statt großer Flächen: über ZeichenBudget, aber schnell gezeichnet (40×30).
	var l strings.Builder
	l.WriteString(`{"annotations":[`)
	zeile := strings.Repeat("x", 9000)
	for i := 0; i < int(render.ZeichenBudget)/9000+1; i++ {
		if i > 0 {
			l.WriteString(",")
		}
		l.WriteString(`{"type":"text","at":[1,1],"text":"` + zeile + `","size":1}`)
	}
	l.WriteString("]}")
	liste := schreibeListe(t, dir, l.String())
	code, aus, fehler := fuehre(t, keineMonitore, "--annotate="+liste, "--ignore-limit", "--json", quelle)
	if code != exitOK || !strings.Contains(aus, `"warning":true`) || !strings.Contains(fehler, "without --ignore-limit") ||
		strings.Contains(fehler, "before closing") {
		t.Errorf("Code %d, aus %q, fehler %q", code, aus, fehler)
	}
}
