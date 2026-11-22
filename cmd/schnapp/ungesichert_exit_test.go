// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"errors"
	"fmt"
	"image"
	"io/fs"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/ui"
)

// F-301: ui.Editor liefert errors.Join(fensterErr, schlussErr, err). Mit ui.ErrSpeichernHaengt als Teil nennt
// nutzerFehler beide Teile, nicht nur den ersten passenden Fall der Weiche.
func TestNutzerFehlerJoinMitHaengt(t *testing.T) {
	i18n.Setze(nil)
	haengt := i18n.T("Saving did not respond; the changes may not have been saved.")

	// Fensterfehler unbekannter Art plus hängendes Speichern: beide Teile, Fenster zuerst.
	fenster := errors.New("wayland: Verbindung verloren")
	got := nutzerFehler(errors.Join(fenster, ui.ErrSpeichernHaengt))
	if got != fenster.Error()+"; "+haengt {
		t.Errorf("Fall 1: %q", got)
	}

	// Fensterfehler mit fs.ErrNotExist: beide Texte, der Hinweis auf ungesicherte Änderungen fehlt nicht.
	fenster2 := fmt.Errorf("wayland: connect: %w", fs.ErrNotExist)
	got = nutzerFehler(errors.Join(fenster2, ui.ErrSpeichernHaengt))
	if !strings.Contains(got, haengt) || !strings.Contains(got, i18n.T("The file does not exist.")) {
		t.Errorf("Fall 2: %q", got)
	}

	// Mit übersetztem Speicherfehler dazu: alles einmal, nie die Ursache mit ihrem Pfad.
	ursache := fmt.Errorf("open /geheim/ordner/x.png: %w", fs.ErrPermission)
	got = nutzerFehler(errors.Join(fenster, ui.ErrSpeichernHaengt, speicherFehler(ursache)))
	if strings.Count(got, haengt) != 1 || !strings.Contains(got, fenster.Error()) ||
		!strings.Contains(got, "No permission to write the file.") || strings.Contains(got, "/geheim/") {
		t.Errorf("Fall 3: %q", got)
	}

	// Ein einzelner Fehler bleibt wie bisher.
	if got := nutzerFehler(ui.ErrSpeichernHaengt); got != haengt {
		t.Errorf("einzeln: %q", got)
	}
}

// F-302: Ein Editorfehler, der ungesicherte Änderungen bedeutet, ist nach einer Aufnahme ein Fehler-Exit; ein Fenster-
// fehler allein bleibt eine Warnung (die Aufnahme liegt in der Bibliothek).
func TestEditorUngesichertNachDerAufnahmeIstFehler(t *testing.T) {
	i18n.Setze(nil)
	ursache := fmt.Errorf("open /geheim/ordner/x.png: %w", fs.ErrPermission)
	faelle := map[string]struct {
		err  error
		code int
	}{
		"hängt":                 {errors.Join(errors.New("Fenster"), ui.ErrSpeichernHaengt), exitFehler},
		"hängt allein":          {ui.ErrSpeichernHaengt, exitFehler},
		"Speichern gescheitert": {errors.Join(nil, speicherFehler(ursache)), exitFehler},
		"nur Fenster":           {errors.New("kein Wayland"), exitOK},
	}
	for name, f := range faelle {
		l := neuerLauf(t)
		l.an.bearbeiten = func(abgelegt) error { return f.err }
		l.overlayErg = ui.OverlayErgebnis{Monitor: 0, Bereich: image.Rect(0, 0, 2, 2)}
		code, aus, fehler := l.fuehre(Auftrag{Modus: ModusBereich, Editor: true})
		if code != f.code || aus == "" || !strings.Contains(fehler, "The editor could not be opened or saved: ") {
			t.Errorf("%s: Code %d (Soll %d), aus %q, fehler %q", name, code, f.code, aus, fehler)
		}
		if strings.Contains(fehler, "/geheim/") {
			t.Errorf("%s: Pfad der Ursache in %q", name, fehler)
		}
	}
}

// F-302, DATEI-Weg: Ein Speicherfehler beim Schließen ist ein Fehler-Exit, aber nicht „Cannot open“; ein echter
// Öffnen-Fehler bleibt „Cannot open“.
func TestOeffnenUngesichertMeldung(t *testing.T) {
	i18n.Setze(nil)
	l := neuerLauf(t)
	l.an.editor = func(string, bool, document.Kante) error { return errors.Join(nil, ui.ErrSpeichernHaengt) }
	code, _, fehler := l.fuehre(Auftrag{Modus: ModusOeffnen, Datei: "x.png"})
	if code != exitFehler || strings.Contains(fehler, "Cannot open") ||
		!strings.Contains(fehler, "The editor could not be opened or saved: ") ||
		!strings.Contains(fehler, i18n.T("Saving did not respond; the changes may not have been saved.")) {
		t.Errorf("Code %d, fehler %q", code, fehler)
	}
	l.an.editor = func(string, bool, document.Kante) error { return fs.ErrNotExist }
	code, _, fehler = l.fuehre(Auftrag{Modus: ModusOeffnen, Datei: "x.png"})
	if code != exitFehler || !strings.Contains(fehler, "Cannot open x.png: ") {
		t.Errorf("Öffnen-Fehler: Code %d, fehler %q", code, fehler)
	}
}

// F-303: Die Warnung zur liegen gebliebenen temporären Kopie trägt weder Pfad noch Paketpräfix noch rohe Ursache; die
// Ursache steht nur mit SCHNAPP_DEBUG im Debug-Log.
func TestAufnahmeWarntBeiTempRestOhnePfad(t *testing.T) {
	i18n.Setze(nil)
	ordner := t.TempDir()
	roh := fmt.Errorf("remove %s/.schnapp-tmp-1: %w", ordner, fs.ErrPermission)
	ablegen := func(l *lauf) {
		l.an.ablegen = func(d *document.Dokument) (string, library.Stand, error) {
			l.abgelegt = append(l.abgelegt, d)
			return ordner + "/datei.png", l.stand, fmt.Errorf("%w: %v", library.ErrTempRest, roh)
		}
	}

	l := neuerLauf(t)
	ablegen(l)
	t.Setenv("SCHNAPP_DEBUG", "")
	code, _, fehler := l.fuehre(Auftrag{Modus: ModusFenster})
	if code != exitOK {
		t.Errorf("Code %d, Soll 0", code)
	}
	if !strings.Contains(fehler, "Warning: a temporary copy of the screenshot could not be removed") {
		t.Errorf("Warnung fehlt: %q", fehler)
	}
	for _, v := range []string{ordner, "bibliothek:", ".schnapp-tmp", "permission denied"} {
		if strings.Contains(fehler, v) {
			t.Errorf("stderr enthält %q: %q", v, fehler)
		}
	}

	l = neuerLauf(t)
	ablegen(l)
	t.Setenv("SCHNAPP_DEBUG", "1")
	_, _, fehler = l.fuehre(Auftrag{Modus: ModusFenster})
	if !strings.Contains(fehler, "[schnapp] Ursache der temporären Kopie: ") || !strings.Contains(fehler, ".schnapp-tmp-1") {
		t.Errorf("Debug-Log ohne Ursache: %q", fehler)
	}
}
