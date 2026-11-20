// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"testing"
	"time"

	"gioui.org/app"

	"github.com/Andiministrator/schnapp/canvas"
)

// F-239: Ein Ereignis ohne Bild (Fokuswechsel als ConfigEvent) quittiert den Wecker nicht; erst ein FrameEvent tut es.
// Sonst würde die Dialogantwort angenommen, aber nie gezeigt, und „Dateidialog geöffnet“ bliebe stehen.
func TestWeckerQuittiertNurMitBild(t *testing.T) {
	w := canvas.NeuerWecker(func() {}, time.Hour, time.Hour)
	defer w.Ende()
	w.Anfordern()
	quittiere(w, app.ConfigEvent{})
	if !w.Offen() {
		t.Error("ein ConfigEvent ohne Bild hat den Wecker quittiert (F-239)")
	}
	quittiere(w, app.FrameEvent{})
	if w.Offen() {
		t.Error("ein FrameEvent hat den Wecker nicht quittiert")
	}
}
