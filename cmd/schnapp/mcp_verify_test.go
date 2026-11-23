// SPDX-License-Identifier: GPL-3.0-or-later

package main

// Tests zu den Verify-Resten des MCP-Servers (Karte #205): F-480, F-482 bis F-485, F-492, F-493.

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/Andiministrator/schnapp/internal/capture"
	"github.com/Andiministrator/schnapp/internal/export"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/mcp"
	"github.com/Andiministrator/schnapp/internal/po"
)

// TestMcpExportZielNieInDieAblage: target in Ablage, Vorschau-Cache oder über einen Link auf einen Ordner der Ablage
// wird abgelehnt; nichts entsteht (F-480, Gegenstück zu TestMcpOutputNieInDieAblage).
func TestMcpExportZielNieInDieAblage(t *testing.T) {
	l := neuerMcpLauf(t)
	monat := filepath.Join(l.orte.Ablage, "2026-11")
	for _, o := range []string{monat, l.orte.Vorschau} {
		if err := os.MkdirAll(o, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(l.heim, "verweis")
	if err := os.Symlink(monat, link); err != nil {
		t.Fatal(err)
	}
	aussen := filepath.Join(l.heim, "aussen.png")
	if err := os.WriteFile(aussen, mustPNG(t), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, ziel := range []string{
		filepath.Join(monat, "2026-11-22_10-00-00.png"), filepath.Join(l.orte.Ablage, "a.png"),
		filepath.Join(l.orte.Vorschau, "v.png"), filepath.Join(link, "b.png"),
	} {
		e := l.rufe(t, "export", `{"file":"`+aussen+`","target":"`+ziel+`"}`)
		if !e.Fehler || !strings.Contains(text(e), "library") {
			t.Errorf("%s: %+v", ziel, e)
		}
		if _, err := os.Lstat(ziel); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s entstand", ziel)
		}
	}
}

// signalNachDemSchreiben: Die Schreibnaht schreibt alles und schickt dann SIGHUP an den eigenen Prozess (die Wache
// fängt es), wie ein Signal während des Schreibens.
func signalNachDemSchreiben(t *testing.T) {
	t.Helper()
	fangbar(t, syscall.SIGHUP)
	setzeNaht(t, func(w io.Writer, inhalt []byte, abgebrochen func() bool) error {
		if _, err := w.Write(inhalt); err != nil {
			return err
		}
		if err := syscall.Kill(os.Getpid(), syscall.SIGHUP); err != nil {
			t.Error(err)
		}
		wartePunkt(abgebrochen)
		return nil
	})
}

// TestMcpSignalNachDemSchreiben: Bei capture_screen mit output wird die fertige Datei nach einem Signal verworfen
// (Nachtrag 4 Punkt 6); export sagt ehrlich, dass die Datei vollständig ist, statt „could not be written“ (F-482).
func TestMcpSignalNachDemSchreiben(t *testing.T) {
	l := neuerMcpLauf(t)
	signalNachDemSchreiben(t)
	ziel := filepath.Join(l.heim, "aufnahme.png")
	e := l.rufe(t, "capture_screen", `{"output":"`+ziel+`"}`)
	if !e.Fehler || !strings.Contains(text(e), "discarded") {
		t.Errorf("output nach Signal: %+v", e)
	}
	if _, err := os.Lstat(ziel); !errors.Is(err, os.ErrNotExist) {
		t.Error("Aufnahme blieb nach dem Signal liegen")
	}

	aussen := filepath.Join(l.heim, "aussen.png")
	if err := os.WriteFile(aussen, mustPNG(t), 0o600); err != nil {
		t.Fatal(err)
	}
	exp := filepath.Join(l.heim, "export.png")
	e = l.rufe(t, "export", `{"file":"`+aussen+`","target":"`+exp+`"}`)
	if !e.Fehler || !strings.Contains(text(e), "written completely") {
		t.Errorf("export nach Signal: %+v", e)
	}
	if st, err := os.Stat(exp); err != nil || st.Size() == 0 {
		t.Errorf("Export fehlt: %v, %v", st, err)
	}
}

// TestMcpAbbruchWaehrendDesAblegens: Kommt der Abbruch nach der letzten Prüfung, während die Aufnahme geschrieben oder
// abgelegt wird, bleibt nichts zurück (F-483): weder an output noch in der Bibliothek (Datei, Index, Vorschau).
func TestMcpAbbruchWaehrendDesAblegens(t *testing.T) {
	t.Run("output", func(t *testing.T) {
		l := neuerMcpLauf(t)
		ctx, abbrechen := context.WithCancel(context.Background())
		defer abbrechen()
		setzeNaht(t, func(w io.Writer, inhalt []byte, _ func() bool) error {
			abbrechen() // notifications/cancelled während des Schreibens
			_, err := w.Write(inhalt)
			return err
		})
		ziel := filepath.Join(l.heim, "aufnahme.png")
		e := l.rufeMit(t, ctx, "capture_screen", `{"output":"`+ziel+`"}`, mcp.Kunde{})
		if !e.Fehler || !strings.Contains(text(e), "discarded") {
			t.Errorf("%+v", e)
		}
		if _, err := os.Lstat(ziel); !errors.Is(err, os.ErrNotExist) {
			t.Error("Aufnahme blieb an output liegen")
		}
	})
	t.Run("Bibliothek", func(t *testing.T) {
		l := neuerMcpLauf(t)
		ctx, abbrechen := context.WithCancel(context.Background())
		defer abbrechen()
		orte := l.an.orte
		l.an.orte = func() (library.Orte, error) { // legeAb fragt die Orte, nachdem alles geprüft ist
			abbrechen()
			return orte()
		}
		e := l.rufeMit(t, ctx, "capture_screen", `{}`, mcp.Kunde{})
		if !e.Fehler || !strings.Contains(text(e), "discarded") {
			t.Errorf("%+v", e)
		}
		if p := pngsIn(t, l.orte.Ablage); len(p) != 0 {
			t.Errorf("in der Ablage blieb: %v", p)
		}
		if p := pngsIn(t, l.orte.Vorschau); len(p) != 0 {
			t.Errorf("Vorschau blieb: %v", p)
		}
		treffer, err := library.Neu(l.orte).Suchen(library.Anfrage{})
		if err != nil || len(treffer) != 0 {
			t.Errorf("Index: %v, %v", treffer, err)
		}
	})
}

// TestMcpZweitesSignalRaeumtTemp: Hängt das Schreiben, räumt das zweite Signal an den Server die temporäre Datei ab,
// bevor er den Prozess beendet — nicht erst die Wache im 10-ms-Takt (F-484).
func TestMcpZweitesSignalRaeumtTemp(t *testing.T) {
	dir := t.TempDir()
	frei := make(chan struct{})
	setzeNaht(t, func(w io.Writer, inhalt []byte, _ func() bool) error {
		if _, err := w.Write(inhalt[:1]); err != nil {
			return err
		}
		<-frei // hängt
		return errors.New("test: hängendes Schreiben")
	})
	fertig := make(chan error, 1)
	go func() {
		fertig <- schreibeInhaltMit(filepath.Join(dir, "z.png"), []byte("inhalt"), export.SchreibeOhneErsetzen, nil)
	}()
	var temp string
	if !wartePunkt(func() bool {
		laufendeTemps.Range(func(_, n any) bool { temp = n.(string); return false })
		return temp != ""
	}) {
		close(frei)
		t.Fatal("keine laufende temporäre Datei")
	}
	kanal := make(chan os.Signal, 2)
	kanal <- syscall.SIGTERM
	kanal <- syscall.SIGTERM
	var lagNoch bool
	beendet := make(chan struct{})
	mcpSignale(kanal, func() {}, func(int) {
		_, err := os.Lstat(temp)
		lagNoch = err == nil
		close(beendet)
	}, io.Discard, make(chan struct{}))
	<-beendet
	close(frei)
	<-fertig
	if lagNoch {
		t.Error("temporäre Datei lag beim Beenden noch")
	}
}

// TestKindSpracheVomServer: Das Kind nimmt die Sprache des Servers aus der Anfrage, nicht aus seiner Umgebung (F-485).
func TestKindSpracheVomServer(t *testing.T) {
	ohneProbe(t)
	t.Cleanup(func() { i18n.Setze(nil) })
	i18n.Setze(nil)
	var gesehen string
	dialogAttrappe(t, func(context.Context, string, string) (bool, error) {
		gesehen = i18n.T("Allow")
		return false, nil
	})
	laufeBestaetigung(strings.NewReader(`{"target":"z","lang":"de"}`+"\n"), io.Discard)
	if gesehen != "Erlauben" {
		t.Errorf("Sprache des Servers nicht übernommen: %q", gesehen)
	}
	i18n.Setze(nil)
	laufeBestaetigung(strings.NewReader(`{"target":"z","lang":"xx"}`+"\n"), io.Discard)
	if gesehen != "Allow" {
		t.Errorf("unbekannte Sprache: %q", gesehen)
	}
}

// TestMcpFehlertexteEnglisch: Auch bei deutscher Oberfläche gehen Fehlertexte englisch an den Agenten, ohne Paketfehler
// (F-492).
func TestMcpFehlertexteEnglisch(t *testing.T) {
	l := neuerMcpLauf(t)
	k, err := i18n.Lade(po.Dateien, "de")
	if err != nil {
		t.Fatal(err)
	}
	i18n.Setze(k)
	t.Cleanup(func() { i18n.Setze(nil) })
	l.an.quelle = func() (capture.Quelle, error) { return nil, errors.New("capture: Verbindung zum Kompositor verloren") }
	e := l.rufe(t, "capture_screen", `{}`)
	if !e.Fehler || strings.Contains(text(e), "Verbindung") || !strings.Contains(text(e), "unexpected error") {
		t.Errorf("Aufnahmefehler: %q", text(e))
	}
	l.an.quelle = func() (capture.Quelle, error) { return l.q, nil }
	kaputt := filepath.Join(l.orte.Ablage, "2026-11", "kaputt.png")
	if err := os.MkdirAll(filepath.Dir(kaputt), 0o700); err != nil {
		t.Fatal(err)
	}
	schreibeText(t, kaputt, "\x89PNG\r\n\x1a\nkaputt")
	e = l.rufe(t, "library_read", `{"path":"`+kaputt+`"}`)
	if !e.Fehler || !strings.Contains(text(e), "damaged") {
		t.Errorf("Lesefehler: %q", text(e))
	}
	for _, d := range []string{"Datei", "Bild", "beschädigt"} {
		if strings.Contains(text(e), d) {
			t.Errorf("deutscher Text an den Agenten: %q", text(e))
		}
	}
}

// TestMcpExportVerraetQuelleNicht: Fehlende, unlesbare, fremde und Ordner-Quellen ergeben denselben Text (F-493); in der
// Ablage ohne Einwilligung gilt „library = yes“, ob die Datei es gibt oder nicht.
func TestMcpExportVerraetQuelleNicht(t *testing.T) {
	l := neuerMcpLauf(t)
	unlesbar := filepath.Join(l.heim, "geheim.png")
	if err := os.WriteFile(unlesbar, mustPNG(t), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(unlesbar, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(unlesbar, 0o600) })
	fremd := filepath.Join(l.heim, "text.txt")
	schreibeText(t, fremd, "kein Bild")
	var texte []string
	for i, quelle := range []string{filepath.Join(l.heim, "fehlt.png"), unlesbar, fremd, l.heim} {
		ziel := filepath.Join(l.heim, "ziel"+string(rune('a'+i))+".png")
		e := l.rufe(t, "export", `{"file":"`+quelle+`","target":"`+ziel+`"}`)
		if !e.Fehler {
			t.Fatalf("%s: kein Fehler", quelle)
		}
		texte = append(texte, text(e))
	}
	for _, s := range texte[1:] {
		if s != texte[0] {
			t.Errorf("unterschiedliche Texte verraten die Quelle: %q", texte)
			break
		}
	}
	l.agenten.Bibliothek = false
	drin := l.legeAb(t, false, "2026-11-01T10:00:00Z")
	fehlt := filepath.Join(filepath.Dir(drin), "gibt-es-nicht.png")
	a := l.rufe(t, "export", `{"file":"`+drin+`","target":"`+filepath.Join(l.heim, "x.png")+`"}`)
	b := l.rufe(t, "export", `{"file":"`+fehlt+`","target":"`+filepath.Join(l.heim, "y.png")+`"}`)
	if text(a) != text(b) || !strings.Contains(text(a), "library = yes") {
		t.Errorf("Ablage ohne Einwilligung: %q gegen %q", text(a), text(b))
	}
}
