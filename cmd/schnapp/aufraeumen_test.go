// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"errors"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/bibliothek"
	"github.com/Andiministrator/schnapp/oberflaeche"
	"github.com/Andiministrator/schnapp/sprache"
)

// aufraeumAttrappe merkt sich die Aufrufe des Aufräumens; keine echte Bibliothek, kein Papierkorb.
type aufraeumAttrappe struct {
	aufrufe []bibliothek.Einstellung
	erg     bibliothek.AufraeumErgebnis
	err     error
}

func (a *aufraeumAttrappe) anschliessen(l *lauf, marke string) {
	l.an.aufraeumen = func(e bibliothek.Einstellung, _ time.Time) (bibliothek.AufraeumErgebnis, error) {
		a.aufrufe = append(a.aufrufe, e)
		return a.erg, a.err
	}
	l.an.tagesmarke = marke
}

// Nach dem Speichern läuft das Aufräumen mit der Einstellung aus dem Auftrag (F-88: cleanup-days = 7 → 7 Tage),
// am selben Tag kein zweites Mal; stdout trägt weiter nur den Pfad.
func TestAufraeumenNachDemSpeichernEinmalAmTag(t *testing.T) {
	sprache.Setze(nil)
	l := neuerLauf(t)
	a := &aufraeumAttrappe{}
	a.anschliessen(l, filepath.Join(t.TempDir(), "state", "schnapp", "last-cleanup"))
	l.overlayErg = oberflaeche.OverlayErgebnis{Monitor: 0, Bereich: image.Rect(0, 0, 2, 2)}
	auftrag := Auftrag{Modus: ModusBereich, Aufraeumen: bibliothek.Einstellung{An: true, Tage: 7}}

	for i := 0; i < 2; i++ {
		code, aus, fehler := l.fuehre(auftrag)
		if code != exitOK || aus != "/ablage/2026-11/datei.png\n" || fehler != "" {
			t.Fatalf("Lauf %d: Code %d, aus %q, fehler %q", i+1, code, aus, fehler)
		}
	}
	if len(a.aufrufe) != 1 || a.aufrufe[0] != (bibliothek.Einstellung{An: true, Tage: 7}) {
		t.Fatalf("Aufräumen %+v, Soll genau einmal mit 7 Tagen", a.aufrufe)
	}
}

// Abgeschaltet ([library] cleanup = off): kein Aufruf, keine Marke.
func TestAufraeumenAbgeschaltet(t *testing.T) {
	sprache.Setze(nil)
	l := neuerLauf(t)
	a := &aufraeumAttrappe{}
	marke := filepath.Join(t.TempDir(), "last-cleanup")
	a.anschliessen(l, marke)
	l.overlayErg = oberflaeche.OverlayErgebnis{Monitor: 0, Bereich: image.Rect(0, 0, 2, 2)}
	code, _, fehler := l.fuehre(Auftrag{Modus: ModusBereich, Aufraeumen: bibliothek.Einstellung{An: false, Tage: 7}})
	if code != exitOK || fehler != "" || len(a.aufrufe) != 0 {
		t.Fatalf("Code %d, fehler %q, Aufrufe %d", code, fehler, len(a.aufrufe))
	}
	if _, err := os.Stat(marke); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Marke angelegt: %v", err)
	}
}

// Scheitert das Speichern, wird nicht aufgeräumt.
func TestAufraeumenNichtNachFehlschlag(t *testing.T) {
	sprache.Setze(nil)
	l := neuerLauf(t)
	a := &aufraeumAttrappe{}
	a.anschliessen(l, filepath.Join(t.TempDir(), "last-cleanup"))
	l.ablegenErr = errors.New("voll")
	l.overlayErg = oberflaeche.OverlayErgebnis{Monitor: 0, Bereich: image.Rect(0, 0, 2, 2)}
	code, _, _ := l.fuehre(Auftrag{Modus: ModusBereich, Aufraeumen: bibliothek.Voreinstellung})
	if code != exitFehler || len(a.aufrufe) != 0 {
		t.Fatalf("Code %d, Aufrufe %d", code, len(a.aufrufe))
	}
}

// Fehler und übersprungene Aufnahmen sind Warnungen: Exit-Code OK, stdout nur der Pfad.
func TestAufraeumenWarnungen(t *testing.T) {
	sprache.Setze(nil)
	l := neuerLauf(t)
	a := &aufraeumAttrappe{
		erg: bibliothek.AufraeumErgebnis{Uebersprungen: []bibliothek.Uebersprungen{{Pfad: "a.png"}, {Pfad: "b.png"}}},
		err: errors.New("Index kaputt"),
	}
	a.anschliessen(l, filepath.Join(t.TempDir(), "last-cleanup"))
	l.overlayErg = oberflaeche.OverlayErgebnis{Monitor: 0, Bereich: image.Rect(0, 0, 2, 2)}
	code, aus, fehler := l.fuehre(Auftrag{Modus: ModusBereich, Aufraeumen: bibliothek.Voreinstellung})
	if code != exitOK || aus != "/ablage/2026-11/datei.png\n" {
		t.Fatalf("Code %d, aus %q", code, aus)
	}
	for _, soll := range []string{"Automatic cleanup of the library failed: Index kaputt\n",
		"Automatic cleanup could not move 2 screenshots to the trash.\n"} {
		if !strings.Contains(fehler, soll) {
			t.Errorf("fehler %q ohne %q", fehler, soll)
		}
	}
}

// Ohne Zustandsordner (leerer Markenpfad) und bei nicht schreibbarer Marke: Warnung, kein Aufräumen.
func TestAufraeumenOhneMarke(t *testing.T) {
	sprache.Setze(nil)
	gesperrt := filepath.Join(t.TempDir(), "datei")
	if err := os.WriteFile(gesperrt, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		marke, soll string
	}{
		{"", "Automatic cleanup of the library skipped: no state directory (HOME is not set).\n"},
		// Elternordner ist eine Datei: MkdirAll scheitert, ohne fremde Rechte zu brauchen.
		{filepath.Join(gesperrt, "schnapp", "last-cleanup"), "Automatic cleanup of the library skipped: "},
	} {
		l := neuerLauf(t)
		a := &aufraeumAttrappe{}
		a.anschliessen(l, tc.marke)
		l.overlayErg = oberflaeche.OverlayErgebnis{Monitor: 0, Bereich: image.Rect(0, 0, 2, 2)}
		var aus, fehler bytes.Buffer
		code := l.an.Ausfuehren(Auftrag{Modus: ModusBereich, Aufraeumen: bibliothek.Voreinstellung}, &aus, &fehler)
		if code != exitOK || !strings.HasPrefix(fehler.String(), tc.soll) || len(a.aufrufe) != 0 {
			t.Errorf("Marke %q: Code %d, fehler %q, Aufrufe %d", tc.marke, code, fehler.String(), len(a.aufrufe))
		}
	}
}
