// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Karte #326 (ADR-0026 Nachtrag 11): Datei → Öffnen… und Strg+O zeigen den Dateidialog des Portals; die gewählte Datei
// öffnet sich im selben Fenster wie eine Aufnahme aus der Leiste. Abbruch tut nichts, ohne Portal sagt es die Fußzeile.

import (
	"context"
	"image"
	"image/color"
	"path/filepath"
	"strings"
	"testing"

	"gioui.org/io/key"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/portal"
)

// mitOeffnenDialog ersetzt den Dialog für einen Test: antwort liefert die Wahl, anfragen zählt die Aufrufe.
func mitOeffnenDialog(t *testing.T, erg portal.Ergebnis, err error) *[]portal.Anfrage {
	t.Helper()
	var anfragen []portal.Anfrage
	alt := oeffnenDialog
	oeffnenDialog = func(_ context.Context, a portal.Anfrage) (portal.Ergebnis, error) {
		anfragen = append(anfragen, a)
		return erg, err
	}
	t.Cleanup(func() { oeffnenDialog = alt })
	return &anfragen
}

func TestOeffnenWechseltZurGewaehltenDatei(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	ziel := filepath.Join(t.TempDir(), "fremd.png")
	anfragen := mitOeffnenDialog(t, portal.Ergebnis{URI: "file://" + ziel}, nil)
	l.taste("O", key.ModShortcut)
	l.warte(func() bool { return !l.e.oeffnenOffen && !l.e.wechsel.laedt && len(l.aufrufe()) > 0 })
	if len(*anfragen) != 1 {
		t.Fatalf("%d Dialoge", len(*anfragen))
	}
	a := (*anfragen)[0]
	if len(a.Filter) != 1 || !strings.Contains(strings.Join(a.Filter[0].MIMEs, " "), "image/png") {
		t.Errorf("Filter %+v", a.Filter)
	}
	if got := l.aufrufe(); got[len(got)-1] != ziel {
		t.Errorf("geladen %v, Soll %s", got, ziel)
	}
}

func TestOeffnenImMenueDatei(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 2)
	en := l.e.menueEintraegeRoh(menueDatei)
	if len(en) == 0 || en[0].akt != aktOeffnen || !en[0].an {
		t.Fatalf("erster Eintrag im Menü Datei: %+v", en[0])
	}
	if kuerzelText(aktOeffnen) == "" {
		t.Error("Öffnen ohne Kürzel")
	}
	// Ohne Anbindung (Fenster ohne Bibliothek) grau mit Erklärung.
	ohne := neuerMenueLauf(t, image.Pt(1200, 800))
	en = ohne.e.menueEintraegeRoh(menueDatei)
	if en[0].akt != aktOeffnen || en[0].an || en[0].erklaerung == "" {
		t.Errorf("ohne Bibliothek: %+v", en[0])
	}
}

func TestOeffnenAbbruchUndOhnePortal(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 2)
	mitOeffnenDialog(t, portal.Ergebnis{}, portal.ErrAbgebrochen)
	l.taste("O", key.ModShortcut)
	l.warte(func() bool { return !l.e.oeffnenOffen })
	if len(l.aufrufe()) != 0 || l.e.meldung.Fehler {
		t.Errorf("Abbruch: geladen %v, Meldung %q", l.aufrufe(), l.e.meldung.Text)
	}
	mitOeffnenDialog(t, portal.Ergebnis{}, portal.ErrKeinPortal)
	l.taste("O", key.ModShortcut)
	l.warte(func() bool { return !l.e.oeffnenOffen })
	if !l.e.meldung.Fehler || !strings.Contains(l.e.meldung.Text, "portal") {
		t.Errorf("ohne Portal: %q", l.e.meldung.Text)
	}
	mitOeffnenDialog(t, portal.Ergebnis{URI: "https://example.org/x.png"}, nil)
	l.e.melde("", false) // die Meldung ohne Portal steht noch
	l.taste("O", key.ModShortcut)
	l.warte(func() bool { return !l.e.oeffnenOffen })
	if len(l.aufrufe()) != 0 || !l.e.meldung.Fehler || !strings.Contains(l.e.meldung.Text, "local") {
		t.Errorf("entfernte Adresse: geladen %v, Meldung %q", l.aufrufe(), l.e.meldung.Text)
	}
}

// Kritik K-4: Hält ein Speicher-Hinweis den Wechsel auf, sagt die Meldung, dass die gewählte Datei nicht geöffnet ist.
func TestOeffnenNachSpeicherHinweis(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 2)
	l.e.speichern = func(*document.Dokument) (Gespeichert, error) {
		return Gespeichert{Hinweis: "Saved as a new capture."}, nil
	}
	if _, err := l.e.s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(1, 1), document.Pt(9, 9)),
		Strich: document.Strich{Farbe: color.NRGBA{R: 200, A: 255}, Breite: 2}}); err != nil {
		t.Fatal(err)
	}
	ziel := filepath.Join(t.TempDir(), "fremd.png")
	mitOeffnenDialog(t, portal.Ergebnis{URI: "file://" + ziel}, nil)
	l.taste("O", key.ModShortcut)
	l.warte(func() bool { return !l.e.oeffnenOffen && !l.e.s.SpeichernLaeuft() && !l.e.wechsel.laedt })
	if len(l.aufrufe()) != 0 {
		t.Errorf("trotz Hinweis geladen: %v", l.aufrufe())
	}
	if !strings.Contains(l.e.meldung.Text, "Saved as a new capture.") || !strings.Contains(l.e.meldung.Text, "Ctrl+O") {
		t.Errorf("Meldung %q", l.e.meldung.Text)
	}
}
