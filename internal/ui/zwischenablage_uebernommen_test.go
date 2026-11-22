// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"os"
	"testing"

	"gioui.org/app"
	"gioui.org/layout"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// nachEreignisBereit rüstet den Editor für nachEreignis (Vorlage: TestNachEreignisSichert).
func nachEreignisBereit(t *testing.T, e *editor) {
	t.Helper()
	e.weitereSignale = make(chan struct{}, 1)
	e.wache = canvas.NeueSignalwache(make(chan os.Signal), func() {}, func(os.Signal, bool) {})
	t.Cleanup(e.wache.Ende)
	if e.w == nil {
		e.w = new(app.Window) // ein leeres Fenster: Invalidate tut nichts
	}
}

// F-328, QA-4: Die Verdrahtung trägt: Rückruf eingehängt, nachEreignis verbraucht den Merker, das Schließen warnt nicht.
func TestRueckrufVerdrahtet(t *testing.T) {
	e, _ := editorMitFensterKopie(t)
	nachEreignisBereit(t, e)
	alt := app.ClipboardCancelled
	t.Cleanup(func() { app.ClipboardCancelled = alt; zwischenablageUebernommen.Store(false) })
	geweckt := 0
	zwischenablageRueckrufSetzen(func() { geweckt++ })
	if app.ClipboardCancelled == nil {
		t.Fatal("Rückruf nicht eingehängt")
	}
	app.ClipboardCancelled()
	if geweckt != 1 {
		t.Errorf("%d Weckrufe, erwartet 1", geweckt)
	}
	e.nachEreignis()
	if e.fensterHaeltKopie || e.fensterWarnen() {
		t.Error("Schließen warnt trotz Übernahme")
	}
}

// F-328, QA-4: Ein Merker ohne Fensterkopie wird verbraucht und räumt eine spätere Kopie im Rückfall nicht ab.
func TestMerkerOhneKopieRaeumtSpaeteKopieNichtAb(t *testing.T) {
	t.Setenv("WAYLAND_SOCKET", "1")
	e := testEditor(t)
	nachEreignisBereit(t, e)
	e.speichern = func(*document.Dokument) (Gespeichert, error) { return Gespeichert{}, nil }
	zwischenablageUebernommen.Store(true)
	t.Cleanup(func() { zwischenablageUebernommen.Store(false) })
	e.nachEreignis()
	if zwischenablageUebernommen.Load() {
		t.Error("Merker ohne Fensterkopie nicht verbraucht")
	}
	gtx := layout.Context{}
	e.kopieren(gtx)
	warteKopieren(t, e, gtx)
	e.nachEreignis()
	if !e.fensterHaeltKopie {
		t.Error("späte Kopie wurde abgeräumt")
	}
}

// F-328, QA-4: Ein Merker aus einem früheren Lauf wird beim Start zurückgesetzt.
func TestMerkerAltWirdBeimStartZurueckgesetzt(t *testing.T) {
	alt := app.ClipboardCancelled
	t.Cleanup(func() { app.ClipboardCancelled = alt; zwischenablageUebernommen.Store(false) })
	zwischenablageUebernommen.Store(true)
	zwischenablageRueckrufSetzen(func() {})
	if zwischenablageUebernommen.Load() {
		t.Error("alter Merker blieb gesetzt")
	}
}

// F-328, QA-5: Ein Abbruch der alten Quelle, der vor dem Abholen der neuen Kopie eintraf, räumt sie nicht ab.
func TestAbbruchAlterQuelleRaeumtNeueKopieNichtAb(t *testing.T) {
	e, _ := editorMitFensterKopie(t)
	gtx := layout.Context{}
	e.kopieren(gtx)
	zwischenablageUebernommen.Store(true)
	t.Cleanup(func() { zwischenablageUebernommen.Store(false) })
	warteKopieren(t, e, gtx)
	e.zwischenablageUebernommenMelden()
	if !e.fensterHaeltKopie {
		t.Error("neue Kopie abgeräumt")
	}
}

// F-328: Übernimmt ein anderes Programm die Zwischenablage (Gio-Patch: app.ClipboardCancelled), hält das Fenster die
// Kopie nicht mehr: Das Schließen warnt nicht.
func TestUebernommenOhneWarnungBeimSchliessen(t *testing.T) {
	e, _ := editorMitFensterKopie(t)
	zwischenablageUebernommen.Store(true) // das täte der Gio-Patch
	t.Cleanup(func() { zwischenablageUebernommen.Store(false) })
	if !e.zwischenablageUebernommenMelden() {
		t.Fatal("meldete die Übernahme nicht")
	}
	if e.fensterHaeltKopie {
		t.Error("fensterHaeltKopie blieb gesetzt")
	}
	if e.fensterWarnen() {
		t.Error("Schließen warnt trotz Übernahme")
	}
	if e.meldung.Text == warnungKopie() {
		t.Error("Warnung gezeigt")
	}
}

// F-328: Eine bereits gezeigte Fenster-Warnung verschwindet, wenn die Übernahme danach eintrifft.
func TestUebernommenLoestGezeigteWarnungAuf(t *testing.T) {
	e, _ := editorMitFensterKopie(t)
	if !e.fensterWarnen() || e.meldung.Text != warnungKopie() || e.meldungHaelt != haltFenster {
		t.Fatal("Vorbedingung: Warnung nicht gezeigt")
	}
	zwischenablageUebernommen.Store(true)
	t.Cleanup(func() { zwischenablageUebernommen.Store(false) })
	if !e.zwischenablageUebernommenMelden() {
		t.Fatal("meldete die Übernahme nicht")
	}
	if e.haltendeMeldung() || e.meldungHaelt != keinHalt || e.meldung.Text != "" {
		t.Errorf("Warnung blieb: %q (Halt %v)", e.meldung.Text, e.meldungHaelt)
	}
	if e.zwischenablageUebernommenMelden() {
		t.Error("meldete zweimal")
	}
}

// F-328 Gegenprobe: Ohne Übernahme bleibt die Warnung.
func TestOhneUebernahmeBleibtWarnung(t *testing.T) {
	e, _ := editorMitFensterKopie(t)
	zwischenablageUebernommen.Store(false)
	if e.zwischenablageUebernommenMelden() {
		t.Fatal("meldete ohne Übernahme")
	}
	if !e.fensterWarnen() || e.meldung.Text != warnungKopie() {
		t.Errorf("Warnung fehlt: %q", e.meldung.Text)
	}
}
