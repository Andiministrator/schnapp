// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Tests zu F-411 (Ort der beiseitegelegten Kopie auf der Fehlerausgabe) und F-454 (Statusleiste meldet Fremdwerden und
// spätere Übernahme der Rettungs-Sperre), im Editor ohne Fenster.

import (
	"image"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
)

// lageRettung ist eine Attrappe mit einstellbarer Lage.
type lageRettung struct {
	meldendeRettung
	lText   string
	lFehl   bool
	lNr     uint64
	gefragt int
}

func (r *lageRettung) Lage() (string, bool, uint64) {
	r.gefragt++
	return r.lText, r.lFehl, r.lNr
}

// F-411: Eine Wiederherstellungs-Meldung mit Fehler steht auch auf der Fehlerausgabe, mit dem Ort der Kopie.
func TestWiederherstellungFehlerAufFehlerausgabe(t *testing.T) {
	i18n.Setze(nil)
	aus := fehlerAbfangen(t)
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 40, 30)))
	if err != nil {
		t.Fatal(err)
	}
	r := &meldendeRettung{text: "Unsaved changes were not restored. The rescue copy is kept: schnapp/rescue/abc.alt.json", fehler: true}
	neuerEditor("datei.png", d, nil, r)
	if !strings.Contains(aus.String(), "abc.alt.json") {
		t.Errorf("Fehlerausgabe %q enthält den Dateinamen der Kopie nicht", aus.String())
	}
}

// F-411: Eine Meldung ohne Fehler (angewendet) bleibt der Statusleiste vorbehalten.
func TestWiederherstellungOhneFehlerNichtAufFehlerausgabe(t *testing.T) {
	i18n.Setze(nil)
	aus := fehlerAbfangen(t)
	d, _ := document.Neu(image.NewNRGBA(image.Rect(0, 0, 40, 30)))
	neuerEditor("datei.png", d, nil, &meldendeRettung{text: "Restored unsaved changes", fehler: false})
	if aus.Len() != 0 {
		t.Errorf("Fehlerausgabe %q, erwartet leer", aus.String())
	}
}

// F-454: Wird die Sitzung später fremd, zeigt die Statusleiste das; übernimmt sie die Sperre wieder, ändert sich die
// Meldung. Ohne Änderung des Zählers bleibt die Meldung stehen.
func TestStatusleisteMeldetFremdUndUebernahme(t *testing.T) {
	e := testEditor(t)
	r := &lageRettung{}
	e.rettung = r
	e.rettungStart()
	e.rettungPruefen()
	if e.meldung.Text != "" {
		t.Fatalf("Meldung %q ohne Änderung der Lage", e.meldung.Text)
	}

	r.lText, r.lNr = "fremd: another window", 1 // wurde bei NeueBasis fremd
	e.rettungPruefen()
	if e.meldung.Text != "fremd: another window" {
		t.Fatalf("Meldung %q, erwartet die Fremd-Meldung", e.meldung.Text)
	}

	e.melde("etwas anderes", false)
	e.rettungPruefen() // Zähler gleich: nicht erneut melden
	if e.meldung.Text != "etwas anderes" {
		t.Errorf("Meldung %q, erwartet unverändert", e.meldung.Text)
	}

	r.lText, r.lNr = "wieder geschützt", 2 // Übernahme der Sperre
	e.rettungPruefen()
	if e.meldung.Text != "wieder geschützt" {
		t.Errorf("Meldung %q, erwartet die Übernahme-Meldung", e.meldung.Text)
	}
}

// F-454: Eine Lage, die schon beim Öffnen galt (Meldung von Wiederherstellen), meldet der Editor nicht ein zweites Mal.
func TestStatusleisteLageVomStartNichtDoppelt(t *testing.T) {
	e := testEditor(t)
	r := &lageRettung{lText: "fremd: another window", lNr: 1}
	r.text = "fremd: another window"
	e.rettung = r
	e.rettungStart()
	e.melde("danach", false)
	e.rettungPruefen()
	if e.meldung.Text != "danach" {
		t.Errorf("Meldung %q, erwartet unverändert", e.meldung.Text)
	}
}

// F-454: Eine haltende Meldung (Verwerfen) verdrängt die Lage-Meldung nicht.
func TestStatusleisteLageVerdraengtHaltendeMeldungNicht(t *testing.T) {
	e := testEditor(t)
	aus := fehlerAbfangen(t)
	r := &lageRettung{}
	e.rettung = r
	e.rettungStart()
	e.meldeHaltend(haltVerwerfen, "Saving failed", true)
	r.lText, r.lFehl, r.lNr = "unavailable", true, 1
	e.rettungPruefen()
	if e.meldung.Text != "Saving failed" {
		t.Errorf("Meldung %q, die haltende Meldung muss stehen bleiben", e.meldung.Text)
	}
	if !strings.Contains(aus.String(), "unavailable") {
		t.Errorf("Fehlerausgabe %q, erwartet die Lage-Meldung", aus.String())
	}
}
