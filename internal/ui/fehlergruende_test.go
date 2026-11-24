// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/po"
	"github.com/Andiministrator/schnapp/internal/settings"
)

// deutsch stellt für die Dauer des Tests die deutsche Oberfläche ein.
func deutsch(t *testing.T) {
	t.Helper()
	k, err := i18n.Lade(po.Dateien, "de")
	if err != nil {
		t.Fatal(err)
	}
	i18n.Setze(k)
	t.Cleanup(func() { i18n.Setze(nil) })
}

// F-623: Die eigenen Fehlerwerte erscheinen übersetzt, Fehler des Betriebssystems bleiben englisch ohne Paketpräfix.
func TestFehlerGrundUebersetzt(t *testing.T) {
	deutsch(t)
	fehler := []error{settings.ErrZuGross, settings.ErrKeineDatei, settings.ErrKeinUTF8, settings.ErrUngueltig,
		settings.ErrVerweisLeer, settings.ErrKeinHeim, errKeinOrt, settings.ErrStileZuGross, settings.ErrStileKeineDatei,
		settings.ErrStileVerweis, settings.ErrStileUnlesbar, settings.ErrStileSicherung, settings.ErrStileGesperrt}
	for _, e := range fehler {
		g := fehlerGrund(fmt.Errorf("settings: %w: Zusatz", e)) // eingewickelt wie im Betrieb
		if g == "" || g == e.Error() {
			t.Errorf("%q nicht übersetzt: %q", e, g)
		}
	}
	if g := fehlerGrund(settings.ErrKeineDatei); g != "die Einstellungsdatei ist keine gewöhnliche Datei" {
		t.Errorf("Grund %q", g)
	}
	if g := fehlerGrund(errors.New("settings: permission denied")); g != "permission denied" {
		t.Errorf("Betriebssystem-Fehler: %q", g)
	}
}

// F-624: Ein Text mit Spitze heißt in der Ebenenliste „Sprechblase“, nicht „Text“.
func TestEbenenNameSprechblase(t *testing.T) {
	l, _ := ebenenLauf(t)
	sp := document.Pt(5, 40)
	id, err := l.e.s.D.Hinzufuegen(document.Text{Ort: document.Pt(10, 10), Inhalt: "Hi", Schriftgrad: 12, Farbe: canvas.Schwaerzfarbe, Spitze: &sp})
	if err != nil {
		t.Fatal(err)
	}
	var zeile canvas.EbenenZeile
	for _, z := range l.e.s.Ebenen() {
		if z.ID == id {
			zeile = z
		}
	}
	if got := l.e.ebenenName(zeile); got != "Speech bubble “Hi”" {
		t.Errorf("Name %q", got)
	}
	deutsch(t)
	if got := l.e.ebenenName(zeile); got != "Sprechblase „Hi“" {
		t.Errorf("deutsch %q", got)
	}
}
