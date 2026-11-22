// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/render"
	"github.com/Andiministrator/schnapp/internal/rescue"
	"github.com/Andiministrator/schnapp/internal/ui"
)

// rettung verbindet die Rettungskopie (internal/rescue, ADR-0029) mit dem Editor: Wiederherstellen beim Öffnen mit
// Statusmeldung, Schreiben und Verwerfen auf Zuruf der Oberfläche, neue Ausgangsdatei nach jedem gelungenen Speichern.
type rettung struct {
	s          *rescue.Sicherung
	ohneGrenze bool // --ignore-limit: beim Wiederherstellen das Rechenbudget nicht prüfen
}

// rettungUmgebung liefert die Umgebung für den Ort der Rettungskopien; Naht für Tests.
var rettungUmgebung = os.Getenv

// rettungFuer legt die Rettung für die Ausgangsdatei abs an und räumt alte Kopien auf. Ohne Zustandsordner oder wenn
// die Datei unlesbar ist, gibt es keine (nil); der Editor läuft dann wie bisher.
func rettungFuer(abs string, ohneGrenze bool) *rettung {
	ort, err := rescue.Ort(rettungUmgebung)
	if err != nil {
		rettungDiagnose(err)
		return nil
	}
	if n, err := rescue.Aufraeumen(ort, time.Now()); err != nil || n > 0 {
		rettungDiagnose(fmt.Errorf("aufgeräumt: %d, %v", n, err))
	}
	k, err := rescue.KennungVon(abs)
	if err != nil {
		rettungDiagnose(err)
		return nil
	}
	s := rescue.NeueSicherung(ort, k)
	s.Diagnose = rettungDiagnose
	return &rettung{s: s, ohneGrenze: ohneGrenze}
}

func rettungDiagnose(err error) {
	if os.Getenv("SCHNAPP_DEBUG") != "" {
		log.Printf("[rescue] %v", err)
	}
}

// ui liefert die Rettung für den Editor; ohne Rettung ein echtes nil (kein Interface mit nil-Zeiger).
func (r *rettung) ui() ui.Rettung {
	if r == nil {
		return nil
	}
	return r
}

// nachSpeichern liefert den Rückruf für speichernMitNachher: Die geschriebene Datei ist die neue Ausgangsdatei der
// Rettungskopie (sie kann eine neue Aufnahme der Bibliothek sein). Ohne Rettung nil.
func (r *rettung) nachSpeichern() func(string) {
	if r == nil {
		return nil
	}
	return func(pfad string) {
		k, err := rescue.KennungVon(pfad)
		if err != nil {
			rettungDiagnose(err)
			k = rescue.Kennung{Pfad: pfad} // passt nie: eine spätere Kopie wird nicht angewendet, nur aufgeräumt
		}
		r.s.NeueBasis(k)
	}
}

// Wiederherstellen wendet eine passende Rettungskopie an und liefert die Meldung für die Statusleiste.
func (r *rettung) Wiederherstellen(d *document.Dokument) (string, bool) {
	pruefe := render.Pruefe
	if r.ohneGrenze {
		pruefe = nil
	}
	w := r.s.Wiederherstellen(d, pruefe)
	if w.Grund != nil {
		rettungDiagnose(w.Grund)
	}
	// Datumsformat je Sprache: ein Go-Zeitmuster im Katalog (ADR-0014 Nachtrag 2).
	zeit := w.Gesichert.Local().Format(i18n.TK("date format", "2006-01-02 15:04"))
	// Ort der liegenden Kopie relativ zum Zustandsordner, ohne Benutzerpfad (ADR-0029 Punkt 6, F-155, Verify 7, V7-6).
	name := w.Beiseite
	if name == "" {
		name = rescue.Name(r.s.Basis().Pfad)
	}
	ort := "schnapp/rescue/" + name
	switch {
	case w.Gestoert:
		// Die Sperre auf die Ausgangsdatei ließ sich nicht prüfen (F-440): Diese Sitzung sichert nicht.
		return i18n.T("The rescue copy for this file is unavailable. Changes made here are not protected by a rescue copy."), true
	case w.Fremd:
		// Ein anderes Fenster führt die Rettungskopie dieser Datei (F-413): Diese Sitzung lässt sie in Ruhe.
		return i18n.T("This file is open in another Schnapp window. Changes made here are not protected by a rescue copy."), false
	case w.Angewendet:
		return fmt.Sprintf(i18n.T("Restored unsaved changes from %s. Undo returns to the saved state."), zeit), false
	case !w.Liegt:
		return "", false
	case w.Gesichert.IsZero():
		// Der Ort steht vorn (F-475): Die einzeilige Statusleiste kürzt das Ende, „Öffnen mit“ hat keine Fehlerausgabe.
		return fmt.Sprintf(i18n.T("Rescue copy %s could not be read. It is kept for 30 days in the state folder."), ort), true
	case errors.Is(w.Grund, rescue.ErrAnders):
		return fmt.Sprintf(i18n.T("Rescue copy %s (unsaved changes from %s) was not restored: the file has changed since. It is kept for 30 days in the state folder."), ort, zeit), true
	default:
		return fmt.Sprintf(i18n.T("Rescue copy %s (unsaved changes from %s) could not be restored. It is kept for 30 days in the state folder."), ort, zeit), true
	}
}

// Lage meldet, wenn sich der Stand der Sperre seit dem Start geändert hat (F-454): Die Sitzung wurde bei NeueBasis
// fremd oder gestört, oder sie hat die Sperre eines anderen Fensters später übernommen. nr wächst mit jeder Änderung.
func (r *rettung) Lage() (string, bool, uint64) {
	l := r.s.Lage()
	switch {
	case l.Gestoert:
		return i18n.T("The rescue copy for this file is unavailable. Changes made here are not protected by a rescue copy."), true, l.Nr
	case l.Fremd:
		return i18n.T("This file is open in another Schnapp window. Changes made here are not protected by a rescue copy."), false, l.Nr
	case l.Nr == 0:
		return "", false, 0
	case l.Aus && l.Grund == rescue.LageAus:
		// Die Übernahme der Sperre ist gelungen, die Kopie des anderen Fensters aber nicht beiseite zu legen (F-472).
		return i18n.T("The rescue copy left by the other Schnapp window could not be set aside. Changes made here are not protected by a rescue copy."), true, l.Nr
	case l.Grund == rescue.LageBasiswechsel:
		// Eine andere Ausgangsdatei: Kein anderes Fenster hat etwas losgelassen (F-472).
		return i18n.T("Saved as a different file. Changes made here are protected by a rescue copy again."), false, l.Nr
	case l.Grund == rescue.LageUebernahme:
		return i18n.T("The other Schnapp window has let go of this file. Changes made here are protected by a rescue copy again."), false, l.Nr
	}
	return "", false, l.Nr
}

// Sichere schreibt die Rettungskopie im Hintergrund.
func (r *rettung) Sichere(d *document.Dokument) { r.s.Sichere(d) }

// Verwirf löscht die Rettungskopie.
func (r *rettung) Verwirf() { r.s.Verwirf() }

// Ende wartet auf begonnene Aufträge, höchstens die Frist der Sicherung (F-417), und gibt die Sperre frei.
func (r *rettung) Ende() { r.s.Ende() }
