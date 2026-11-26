// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
)

// uebernahmeEditor ist ein Editor ohne Fenster für eine Datei von außen (ADR-0016 Nachtrag 11); n zählt die Aufrufe der
// Speicherfunktion.
func uebernahmeEditor(t *testing.T) (*editor, *atomic.Int32) {
	t.Helper()
	i18n.Setze(nil)
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 200, 150)))
	if err != nil {
		t.Fatal(err)
	}
	n := new(atomic.Int32)
	e := neuerEditor("bild.png", d, func(*document.Dokument) (Gespeichert, error) {
		n.Add(1)
		return Gespeichert{}, nil
	}, nil)
	e.s.SetzeFenster(image.Pt(400, 300))
	e.aussenSetzen(true)
	return e, n
}

// Karte #348, F-825: Eine Datei von außen wird erst mit der ersten Änderung gespeichert (und damit eine Aufnahme), nicht
// während eines Zugs, und nur einmal angestoßen. Beim Öffnen sagt die Fußzeile, dass sie nicht in der Bibliothek ist.
func TestUebernahmeBeiErsterAenderung(t *testing.T) {
	e, n := uebernahmeEditor(t)
	if !strings.Contains(e.meldung.Text, "not in the library") {
		t.Errorf("Hinweis beim Öffnen fehlt: %q", e.meldung.Text)
	}
	if e.uebernahmePruefen() {
		t.Fatal("ohne Änderung angestoßen")
	}
	e.s.SetzeWerkzeug(canvas.WerkzeugKasten)
	e.s.Druecken(e.s.Abb.ZuFenster(document.Pt(50, 50)), canvas.TastePrimaer)
	e.s.Ziehen(e.s.Abb.ZuFenster(document.Pt(120, 110)))
	if e.uebernahmePruefen() {
		t.Fatal("während des Zugs angestoßen")
	}
	e.s.Loslassen(e.s.Abb.ZuFenster(document.Pt(120, 110)))
	if !e.s.Geaendert() {
		t.Fatal("Kasten hat das Dokument nicht geändert")
	}
	if !e.uebernahmePruefen() || e.aussen {
		t.Fatalf("nach der ersten Änderung nicht angestoßen (aussen %v)", e.aussen)
	}
	warteSpeichern(t, e)
	if n.Load() != 1 {
		t.Errorf("Speicherfunktion %d mal gerufen, Soll 1", n.Load())
	}
	// Eine weitere Änderung speichert nicht von selbst.
	e.s.Druecken(e.s.Abb.ZuFenster(document.Pt(10, 10)), canvas.TastePrimaer)
	e.s.Loslassen(e.s.Abb.ZuFenster(document.Pt(40, 40)))
	if e.uebernahmePruefen() {
		t.Error("zweites Mal angestoßen")
	}
}

// Eine laufende Texteingabe wird nicht durch das Speichern bestätigt: angestoßen wird erst danach.
func TestUebernahmeWartetAufTexteingabe(t *testing.T) {
	e, _ := uebernahmeEditor(t)
	e.s.SetzeWerkzeug(canvas.WerkzeugText)
	e.s.Druecken(e.s.Abb.ZuFenster(document.Pt(60, 60)), canvas.TastePrimaer)
	e.s.Loslassen(e.s.Abb.ZuFenster(document.Pt(60, 60)))
	if !e.s.TextLaeuft() {
		t.Fatal("keine Texteingabe")
	}
	e.s.Tippe("Hallo")
	if e.uebernahmePruefen() || !e.s.TextLaeuft() {
		t.Fatalf("während der Texteingabe angestoßen (Text läuft %v)", e.s.TextLaeuft())
	}
	e.s.SetzeWerkzeug(canvas.WerkzeugAuswahl) // bestätigt den Text
	if !e.s.Geaendert() || !e.uebernahmePruefen() {
		t.Fatalf("nach der Texteingabe nicht angestoßen (geändert %v)", e.s.Geaendert())
	}
	warteSpeichern(t, e)
}

// Ein Wechsel auf eine Aufnahme der Bibliothek hebt den Merker auf; auf eine Datei von außen setzt er ihn.
func TestUebernahmeNachWechsel(t *testing.T) {
	e, _ := uebernahmeEditor(t)
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 50, 50)))
	if err != nil {
		t.Fatal(err)
	}
	e.uebernimm(Geoeffnet{Titel: "aufnahme.png", Dokument: d})
	if e.aussen {
		t.Error("nach dem Wechsel auf eine Aufnahme noch aussen")
	}
	d2, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 50, 50)))
	if err != nil {
		t.Fatal(err)
	}
	e.uebernimm(Geoeffnet{Titel: "aussen.png", Dokument: d2, Aussen: true})
	if !e.aussen || !strings.Contains(e.meldung.Text, "not in the library") {
		t.Errorf("nach dem Wechsel auf eine Datei von außen: aussen %v, Meldung %q", e.aussen, e.meldung.Text)
	}
}

// Ist das Dokument schon geändert (etwa durch einen Druck, dem gleich ein Zug folgt), wartet das Anstoßen auf das Ende
// eines laufenden Zugs und einer laufenden Texteingabe (Mutanten IM12, IM13).
func TestUebernahmeWartetBeiSchonGeaendertemDokument(t *testing.T) {
	for _, art := range []string{"Zug", "Text"} {
		e, _ := uebernahmeEditor(t)
		if _, err := e.s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(2, 2), document.Pt(20, 20)),
			Strich: document.Strich{Farbe: canvas.Schwaerzfarbe, Breite: 2}}); err != nil {
			t.Fatal(err)
		}
		switch art {
		case "Zug":
			e.s.SetzeWerkzeug(canvas.WerkzeugKasten)
			e.s.Druecken(e.s.Abb.ZuFenster(document.Pt(50, 50)), canvas.TastePrimaer)
			e.s.Ziehen(e.s.Abb.ZuFenster(document.Pt(120, 110)))
		case "Text":
			e.s.SetzeWerkzeug(canvas.WerkzeugText)
			e.s.Druecken(e.s.Abb.ZuFenster(document.Pt(60, 60)), canvas.TastePrimaer)
			e.s.Loslassen(e.s.Abb.ZuFenster(document.Pt(60, 60)))
			e.s.Tippe("Hallo")
		}
		if !e.s.Geaendert() || e.uebernahmePruefen() {
			t.Errorf("%s: während der Eingabe angestoßen (geändert %v)", art, e.s.Geaendert())
		}
		if art == "Zug" {
			e.s.Loslassen(e.s.Abb.ZuFenster(document.Pt(120, 110)))
		} else {
			e.s.SetzeWerkzeug(canvas.WerkzeugAuswahl)
		}
		if !e.uebernahmePruefen() {
			t.Errorf("%s: nach dem Ende nicht angestoßen", art)
		}
		warteSpeichern(t, e)
	}
}

// Sitzungs-Kritik 26.11. K-1, K-2: Ist die erste Änderung eine Schwärzung, wird nicht von selbst gespeichert — das
// Speichern leerte den Verlauf, die Schwärzung wäre sofort endgültig. Rückgängig bleibt möglich; danach (ohne
// Schwärzung im Verlauf) stößt die nächste Änderung das Speichern an.
func TestUebernahmeNichtNachSchwaerzung(t *testing.T) {
	e, n := uebernahmeEditor(t)
	e.s.SetzeWerkzeug(canvas.WerkzeugSchwaerzen)
	e.s.Druecken(e.s.Abb.ZuFenster(document.Pt(20, 20)), canvas.TastePrimaer)
	e.s.Ziehen(e.s.Abb.ZuFenster(document.Pt(60, 50)))
	e.s.Loslassen(e.s.Abb.ZuFenster(document.Pt(60, 50)))
	if !e.s.Geaendert() || !e.s.D.VerlaufMitSchwaerzung() {
		t.Fatal("keine Schwärzung")
	}
	if e.uebernahmePruefen() || !e.aussen {
		t.Fatalf("nach einer Schwärzung angestoßen (aussen %v)", e.aussen)
	}
	if !e.s.Rueckgaengig() {
		t.Fatal("Schwärzung lässt sich nicht zurücknehmen")
	}
	e.s.SetzeWerkzeug(canvas.WerkzeugKasten)
	e.s.Druecken(e.s.Abb.ZuFenster(document.Pt(80, 80)), canvas.TastePrimaer)
	e.s.Ziehen(e.s.Abb.ZuFenster(document.Pt(120, 110)))
	e.s.Loslassen(e.s.Abb.ZuFenster(document.Pt(120, 110)))
	if !e.uebernahmePruefen() {
		t.Fatal("nach der Rücknahme nicht angestoßen")
	}
	warteSpeichern(t, e)
	if n.Load() != 1 {
		t.Errorf("Speicherfunktion %d mal", n.Load())
	}
}

// K-4: Hat ein anderes Speichern (Strg+S) die Datei schon übernommen, stößt die nächste Änderung nichts mehr an.
func TestUebernahmeNachStrgS(t *testing.T) {
	e, n := uebernahmeEditor(t)
	if _, err := e.s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(2, 2), document.Pt(20, 20)),
		Strich: document.Strich{Farbe: canvas.Schwaerzfarbe, Breite: 2}}); err != nil {
		t.Fatal(err)
	}
	e.speichernJetzt()
	warteSpeichern(t, e)
	if e.aussen || n.Load() != 1 {
		t.Fatalf("nach Strg+S: aussen %v, %d Speicherungen", e.aussen, n.Load())
	}
	if _, err := e.s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(30, 30), document.Pt(50, 50)),
		Strich: document.Strich{Farbe: canvas.Schwaerzfarbe, Breite: 2}}); err != nil {
		t.Fatal(err)
	}
	if e.uebernahmePruefen() {
		t.Error("nach Strg+S noch einmal angestoßen")
	}
}

// K-3: Eine Meldung der Rettungskopie beim Start bleibt stehen; der Hinweis auf die Datei von außen verdrängt sie nicht.
func TestUebernahmeHinweisVerdraengtRettungNicht(t *testing.T) {
	e, _ := uebernahmeEditor(t)
	e.melde("Restored unsaved changes", false)
	e.aussenSetzen(true)
	if e.meldung.Text != "Restored unsaved changes" || !e.aussen {
		t.Errorf("Meldung %q, aussen %v", e.meldung.Text, e.aussen)
	}
}
