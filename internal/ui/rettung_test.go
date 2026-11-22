// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

type rettungAttrappe struct{ gesichert, verworfen, ende int }

func (r *rettungAttrappe) Wiederherstellen(*document.Dokument) (string, bool) { return "", false }
func (r *rettungAttrappe) Sichere(*document.Dokument)                         { r.gesichert++ }
func (r *rettungAttrappe) Verwirf()                                           { r.verworfen++ }
func (r *rettungAttrappe) Ende()                                              { r.ende++ }

// ADR-0029 Punkt 4: frühestens 2 s nach der letzten Änderung, bei laufenden Änderungen spätestens 10 s nach der ersten;
// zurück auf dem gespeicherten Stand wird verworfen; nach gelungenem Speichern ebenso; beim Zerstören nur ohne Änderung.
func TestRettungBuendelung(t *testing.T) {
	jetzt := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	alt := rettungUhr
	rettungUhr = func() time.Time { return jetzt }
	t.Cleanup(func() { rettungUhr = alt })

	e := testEditor(t)
	r := &rettungAttrappe{}
	e.rettung = r
	e.rettungStart()
	e.rettungPruefen()
	if r.gesichert != 0 || r.verworfen != 0 {
		t.Fatalf("ohne Änderung: %+v", r)
	}

	haltAendere(t, e)
	e.rettungPruefen()
	jetzt = jetzt.Add(rettungRuhe - time.Millisecond)
	e.rettungPruefen()
	if r.gesichert != 0 {
		t.Fatal("vor der Ruhezeit gesichert")
	}
	jetzt = jetzt.Add(time.Millisecond)
	e.rettungPruefen()
	if r.gesichert != 1 {
		t.Fatalf("nach der Ruhezeit nicht gesichert: %+v", r)
	}
	e.rettungPruefen()
	if r.gesichert != 1 {
		t.Fatal("ohne neue Änderung erneut gesichert")
	}

	// Änderungen im Sekundentakt: spätestens nach rettungTakt.
	for i := 0; i < 12; i++ {
		haltAendere(t, e)
		e.rettungPruefen()
		jetzt = jetzt.Add(time.Second)
	}
	if r.gesichert != 2 {
		t.Errorf("bei laufenden Änderungen %d Sicherungen, erwartet 2 (nach %v)", r.gesichert, rettungTakt)
	}

	// Rückgängig bis zum gespeicherten Stand: verwerfen.
	for e.s.D.KannRueckgaengig() {
		if err := e.s.D.Rueckgaengig(); err != nil {
			t.Fatal(err)
		}
	}
	e.rettungPruefen()
	if r.verworfen != 1 {
		t.Errorf("auf dem gespeicherten Stand nicht verworfen: %+v", r)
	}

	// Gelungenes Speichern verwirft; Zerstören ohne Änderung verwirft und wartet.
	haltAendere(t, e)
	e.rettungGespeichert()
	if r.verworfen != 2 {
		t.Errorf("nach dem Speichern nicht verworfen: %+v", r)
	}
	e.s.Gespeichert()
	e.rettungEnde()
	if r.verworfen != 3 || r.ende != 1 {
		t.Errorf("Zerstören ohne Änderung: %+v", r)
	}
	// Mit Änderung (etwa Verwerfen nach gescheitertem Speichern bestätigt) bleibt die Kopie.
	haltAendere(t, e)
	e.rettungEnde()
	if r.verworfen != 3 || r.ende != 2 {
		t.Errorf("Zerstören mit Änderung hat verworfen: %+v", r)
	}
}

// Beim Beginn eines Speicherns wird eine ungesicherte Änderung sofort gesichert (V7-7), währenddessen ruht die Rettung
// (ADR-0029 Punkt 4).
func TestRettungRuhtWaehrendSpeichern(t *testing.T) {
	jetzt := time.Now()
	alt := rettungUhr
	rettungUhr = func() time.Time { return jetzt }
	t.Cleanup(func() { rettungUhr = alt })
	e, _ := haltStart(t)
	fehlerAbfangen(t)
	frei := make(chan struct{})
	e.speichern = func(*document.Dokument) (Gespeichert, error) { <-frei; return Gespeichert{}, nil }
	r := &rettungAttrappe{}
	e.rettung = r
	e.rettungStart()
	haltAendere(t, e)
	e.speichernJetzt()
	jetzt = jetzt.Add(time.Minute)
	e.rettungPruefen()
	if r.gesichert != 1 {
		t.Errorf("%d Sicherungen: erwartet genau eine beim Beginn des Speicherns (V7-7), keine weitere währenddessen", r.gesichert)
	}
	close(frei)
	warteSpeichern(t, e)
	if r.verworfen != 1 {
		t.Errorf("nach gelungenem Speichern nicht verworfen: %+v", r)
	}
}

// V7-2: Ein laufender Zug (etwa eine lange Texteingabe) hält die fällige Kopie nicht auf.
func TestRettungAuchWaehrendZug(t *testing.T) {
	jetzt := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	alt := rettungUhr
	rettungUhr = func() time.Time { return jetzt }
	t.Cleanup(func() { rettungUhr = alt })
	e := testEditor(t)
	r := &rettungAttrappe{}
	e.rettung = r
	e.rettungStart()
	haltAendere(t, e)
	e.rettungPruefen()
	if _, err := e.s.D.BeginneNeuenZug(document.Text{Ort: document.Pt(3, 20), Inhalt: "z", Schriftgrad: 10}); err != nil {
		t.Fatal(err)
	}
	jetzt = jetzt.Add(5 * time.Minute)
	e.rettungPruefen()
	if r.gesichert != 1 {
		t.Errorf("während des Zugs %d Sicherungen, erwartet 1", r.gesichert)
	}
}
