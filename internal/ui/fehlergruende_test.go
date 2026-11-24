// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
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

// F-623, F-660: Jeder eigene Fehlerwert erscheint mit seinem deutschen Text (Sollwert je Wert, nicht nur „anders als
// englisch“), und der Zusatz hinter dem Wert bleibt stehen. Fehler des Betriebssystems bleiben englisch, ohne
// Paketpräfix und ohne Pfad (F-661).
func TestFehlerGrundUebersetzt(t *testing.T) {
	deutsch(t)
	for _, f := range []struct {
		wert error
		soll string
	}{
		{settings.ErrZuGross, "die Einstellungsdatei ist größer als 64 KiB"},
		{settings.ErrKeineDatei, "die Einstellungsdatei ist keine gewöhnliche Datei"},
		{settings.ErrKeinUTF8, "die Einstellungsdatei ist kein gültiges UTF-8"},
		{settings.ErrUngueltig, "eine Einstellung hat einen ungültigen Wert"},
		{settings.ErrVerweisLeer, "die Einstellungsdatei ist ein Verweis ins Leere"},
		{settings.ErrKeinHeim, "das Heimverzeichnis des Benutzers ist unbekannt"},
		{errKeinOrt, "die Einstellungsdatei ist nicht auffindbar"},
		{settings.ErrStileZuGross, "die Stildatei wäre größer als 64 KiB"},
		{settings.ErrStileKeineDatei, "die Stildatei ist keine gewöhnliche Datei"},
		{settings.ErrStileVerweis, "die Stildatei ist ein Verweis ins Leere"},
		{settings.ErrStileUnlesbar, "die vorhandene Stildatei ist nicht lesbar und bleibt deshalb unverändert"},
		{settings.ErrStileSicherung, "die vorhandene Stildatei hat unlesbare Teile und styles.ini.bak existiert schon, deshalb bleibt sie unverändert"},
		{settings.ErrStileGesperrt, "die Stildatei ist von einem anderen Fenster gesperrt"},
	} {
		if g := fehlerGrund(f.wert); g != f.soll {
			t.Errorf("%q: %q, Soll %q", f.wert, g, f.soll)
		}
		// eingewickelt wie im Betrieb: Präfix weg, Zusatz bleibt
		if g := fehlerGrund(fmt.Errorf("settings: %w: Zusatz", f.wert)); g != f.soll+": Zusatz" {
			t.Errorf("%q mit Zusatz: %q", f.wert, g)
		}
	}
	// Der Schlüssel hinter dem Fehlerwert bleibt stehen, sonst weiß niemand, welcher Wert ungültig ist.
	if g := fehlerGrund(fmt.Errorf("settings: %w", fmt.Errorf("%w: capture.seam = %q", settings.ErrUngueltig, "x"))); g != `eine Einstellung hat einen ungültigen Wert: capture.seam = "x"` {
		t.Errorf("Zusatz verloren: %q", g)
	}
	if g := fehlerGrund(errors.New("settings: something else")); g != "something else" {
		t.Errorf("unbekannter Fehler: %q", g)
	}
}

// F-661: Ein echter Schreibfehler (Ordner ohne Schreibrecht) zeigt nur die Ursache, nie Pfad, Heim oder Temporärnamen
// — wie exportFehlerText, über dieselbe Funktion.
func TestFehlerGrundOhnePfad(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("als root wirken Dateirechte nicht")
	}
	ordner := filepath.Join(t.TempDir(), "zu")
	if err := os.Mkdir(ordner, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(ordner, 0o700) })
	err := settings.SchreibeDatei(filepath.Join(ordner, "settings.ini"), []settings.Aenderung{{Abschnitt: "library", Schluessel: "bar", Wert: "no"}})
	if err == nil {
		t.Fatal("Vorbedingung: Schreiben gelang")
	}
	g := fehlerGrund(err)
	if strings.Contains(g, "/") || strings.Contains(g, "schnapp-tmp") || g != "permission denied" {
		t.Errorf("Grund %q (Fehler %v)", g, err)
	}
	if x := exportFehlerText(err, "bild.png"); x != g {
		t.Errorf("Export %q, Tafel %q: nicht einheitlich", x, g)
	}
	// os.LinkError ebenso
	le := &os.LinkError{Op: "rename", Old: "/home/jemand/a", New: "/home/jemand/b", Err: syscall.EXDEV}
	if g := fehlerGrund(fmt.Errorf("settings: %w", le)); g != syscall.EXDEV.Error() {
		t.Errorf("LinkError: %q", g)
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
