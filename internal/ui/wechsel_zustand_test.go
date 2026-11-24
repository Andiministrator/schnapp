// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Oberflächenzustand beim Wechsel der Aufnahme und beim Schließen (Abschluss-Kritik C, Sprint 12): F-706, F-708, F-710.
// Mutanten: scripts/mutanten/ui-wechsel-zustand.json.

import (
	"image"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

func wzKasten() document.Kasten {
	return document.Kasten{Bereich: document.R(document.Pt(2, 2), document.Pt(12, 12)),
		Strich: document.Strich{Farbe: canvas.Schwaerzfarbe, Breite: 2}}
}

// F-706: Ein Zug in der Ebenenliste, der vor dem Wechsel der Aufnahme begann, ordnet nach dem Wechsel nichts um.
func TestWechselBeendetZugDerEbenenliste(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 1600), 3)
	if _, err := l.e.s.D.Hinzufuegen(wzKasten()); err != nil {
		t.Fatal(err)
	}
	l.e.s.Gespeichert()
	l.setle()
	l.e.bib.Oeffnen = func(pfad string) (Geoeffnet, error) {
		g, err := l.oeffnen(pfad)
		if err == nil {
			for range 2 {
				if _, err := g.Dokument.Hinzufuegen(wzKasten()); err != nil {
					return g, err
				}
			}
		}
		return g, err
	}
	l.fokusiere(&l.e.unten.knoepfe[0])
	eb := &l.e.ebenen
	if len(eb.zeilen) < 2 {
		t.Fatalf("Vorbedingung: Zeilen %+v", eb.zeilen)
	}
	a := mitte(eb.zeilen[0].rahmen)
	b := image.Pt(a.X, eb.zeilen[1].rahmen.Max.Y-2)
	pa, pb := f32.Pt(float32(a.X), float32(a.Y)), f32.Pt(float32(b.X), float32(b.Y))
	l.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: pa})
	l.setle()
	l.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: pa, Time: time.Hour})
	l.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: pb})
	l.setle()
	if !eb.druck.zieht {
		t.Fatalf("Vorbedingung: kein Ziehen %+v", eb.druck)
	}
	alt := l.e.s.D
	l.taste(key.NameReturn, 0)
	l.warte(func() bool { return l.e.s.D != alt && !l.e.wechsel.laedt })
	if eb.druck.aktiv || eb.druck.zieht {
		t.Errorf("Zug nach dem Wechsel noch offen: %+v", eb.druck)
	}
	vor := l.e.s.D.Objekte()
	l.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: pb})
	l.setle()
	nach := l.e.s.D.Objekte()
	if vor[0].Kennung() != nach[0].Kennung() || l.e.s.Geaendert() {
		t.Errorf("die neue Aufnahme wurde durch den alten Zug umgeordnet (geändert %v)", l.e.s.Geaendert())
	}
}

// F-708: Das Zerstören wartet auf eine laufende Arbeit der Bibliotheksansicht (Schlagworte, Papierkorb).
func TestZerstoerenWartetAufArbeitDerAnsicht(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	e := l.e
	e.s.Gespeichert()
	tor := make(chan struct{})
	notbremse := time.AfterFunc(5*time.Second, func() { close(tor) }) // hängt nie, auch wenn der Fix fehlt
	defer notbremse.Stop()
	e.bibArbeiten(bibArbeit{rel: l.rel(0)}, func() error { <-tor; return nil })
	fertig := make(chan struct{})
	go func() { e.zerstoert(nil); close(fertig) }()
	select {
	case <-fertig:
		t.Error("zerstoert wartete nicht auf die laufende Arbeit der Ansicht")
	case <-time.After(300 * time.Millisecond):
	}
	if notbremse.Stop() { // Notbremse noch nicht ausgelöst: Tor selbst öffnen
		close(tor)
	}
	select {
	case <-fertig:
	case <-time.After(8 * time.Second):
		t.Fatal("zerstoert kehrt nach der Arbeit nicht zurück")
	}
	if e.bibl.arbeitLaeuft {
		t.Error("arbeitLaeuft nach dem Zerstören noch gesetzt")
	}
}

// F-708: Hängt die Arbeit, endet das Zerstören nach der Frist.
func TestZerstoerenWartetNurBisZurFrist(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	e := l.e
	e.s.Gespeichert()
	alt := wechselFrist
	wechselFrist = 100 * time.Millisecond
	defer func() { wechselFrist = alt }()
	tor := make(chan struct{})
	defer close(tor)
	e.bibArbeiten(bibArbeit{rel: l.rel(0)}, func() error { <-tor; return nil })
	fertig := make(chan struct{})
	go func() { e.zerstoert(nil); close(fertig) }()
	select {
	case <-fertig:
	case <-time.After(3 * time.Second):
		t.Fatal("zerstoert hängt trotz Frist")
	}
}

// F-710: Solange ein Kopieren läuft, wechselt ein Klick auf die Vorschau nicht; danach geht es.
func TestWechselSperrtWaehrendKopieren(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	e := l.e
	e.s.Gespeichert()
	e.kopierLaeuft, e.kopierStart, e.kopierStand = true, time.Now(), e.s.D.Stand()
	e.wechselAnfrage(l.pfade[0])
	time.Sleep(50 * time.Millisecond)
	if a := l.aufrufe(); len(a) != 0 || e.wechsel.laedt || e.wechsel.ziel != "" {
		t.Errorf("gewechselt während des Kopierens: %v", a)
	}
	e.kopierLaeuft = false
	e.wechselAnfrage(l.pfade[0])
	l.warte(func() bool { return len(l.aufrufe()) == 1 })
	l.ruhig()
}
