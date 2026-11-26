// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/pointer"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// langeListe fügt n Kästen hinzu, damit die Ebenenliste länger ist als ihre acht sichtbaren Zeilen (Verify V23-5).
func langeListe(t *testing.T, l *menueLauf, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		if _, err := l.e.s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(2, 2), document.Pt(20, 20)),
			Strich: document.Strich{Farbe: canvas.Schwaerzfarbe, Breite: 2}}); err != nil {
			t.Fatal(err)
		}
	}
	l.setle()
	if eb := &l.e.ebenen; eb.inhaltH <= eb.liste.Dy() {
		t.Fatalf("Liste nicht länger als ihr Fenster: Inhalt %d, Liste %d", eb.inhaltH, eb.liste.Dy())
	}
}

// ziehZeit läuft je Druck weiter, damit zwei Drücke auf dieselbe Zeile kein Doppelklick sind.
var ziehZeit time.Duration

// ziehenBis drückt in der Mitte der Liste, zieht zu p und lässt noch nicht los; danach ist der Zug „zieht“.
func (l *menueLauf) ziehenBis(t *testing.T, p image.Point) {
	t.Helper()
	eb := &l.e.ebenen
	a := mitte(eb.liste)
	ziehZeit += time.Minute
	pa, pb := f32.Pt(float32(a.X), float32(a.Y)), f32.Pt(float32(p.X), float32(p.Y))
	l.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: pa})
	l.setle()
	l.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: pa, Time: ziehZeit})
	l.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: pb}) // Drag
	l.setle()
	if !eb.druck.zieht {
		t.Fatalf("kein Ziehen: %+v", eb.druck)
	}
}

func (l *menueLauf) loslassenBei(p image.Point) {
	l.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(float32(p.X), float32(p.Y))})
	l.setle()
}

// Verify V23-5: Wer eine Zeile am oberen oder unteren Rand einer langen Liste hält, blättert sie um je ein Drittel
// Zeile pro Zeigerereignis; in der Mitte der Liste bleibt sie stehen.
func TestEbenenListeBlaettertBeimZiehenAmRand(t *testing.T) {
	l, k := ebenenLauf(t)
	langeListe(t, l, 12)
	eb := &l.e.ebenen
	// Ein Kasten ist schon gewählt: Der Druck auf eine Zeile wählt einen Kasten, und dessen Eigenschaften (seit #363 auch
	// der Abschnitt „Shadow“) verschöben sonst die Liste nach dem Merken ihrer Lage.
	l.e.s.WaehleEbene(k)
	l.setle()
	liste := eb.liste
	drittel := eb.zeileH / 3

	// Mitte: keine Bewegung der Liste.
	eb.versatz = 4 * eb.zeileH
	l.setle()
	l.ziehenBis(t, image.Pt(mitte(liste).X, mitte(liste).Y+12))
	if eb.versatz != 4*eb.zeileH {
		t.Errorf("Ziehen in der Mitte: Versatz %d, Soll %d", eb.versatz, 4*eb.zeileH)
	}
	l.loslassenBei(image.Pt(mitte(liste).X, mitte(liste).Y+12))

	// Oben: die Liste blättert zurück.
	eb.versatz = 4 * eb.zeileH
	l.setle()
	vor := eb.versatz
	l.ziehenBis(t, image.Pt(mitte(liste).X, liste.Min.Y+2))
	if eb.versatz != vor-drittel {
		t.Errorf("Ziehen am oberen Rand: Versatz %d, Soll %d", eb.versatz, vor-drittel)
	}
	l.loslassenBei(image.Pt(mitte(liste).X, liste.Min.Y+2))

	// Unten: die Liste blättert vor, aber nie über ihr Ende.
	eb.versatz = 2 * eb.zeileH
	l.setle()
	vor = eb.versatz
	l.ziehenBis(t, image.Pt(mitte(liste).X, liste.Max.Y-2))
	if eb.versatz != vor+drittel {
		t.Errorf("Ziehen am unteren Rand: Versatz %d, Soll %d", eb.versatz, vor+drittel)
	}
	l.loslassenBei(image.Pt(mitte(liste).X, liste.Max.Y-2))
}

// Wahl per Menü „Ebene“ (Alt+Bild↑/↓) blättert die Liste zur gewählten Ebene, auch wenn sie außerhalb des sichtbaren
// Ausschnitts liegt (oben wie unten).
func TestEbenenListeBlaettertZurGewaehlten(t *testing.T) {
	l, _ := ebenenLauf(t)
	langeListe(t, l, 12)
	eb := &l.e.ebenen
	gewaehlt := func() (zeile, oben, unten int) {
		for i, z := range l.e.s.Ebenen() {
			if z.Gewaehlt {
				zeile = i
			}
		}
		return zeile, zeile * eb.zeileH, (zeile + 1) * eb.zeileH
	}
	// Ganz unten gewählt (der Text), die Liste steht oben: ein Schritt nach unten in der Liste blättert hinab.
	l.e.s.WaehleEbene(l.id)
	l.setle()
	eb.versatz = 0
	l.e.ebenen.zeigeGewaehlte = true
	l.setle()
	_, _, unten := gewaehlt()
	if eb.versatz != unten-eb.liste.Dy() {
		t.Errorf("gewählte Ebene unten: Versatz %d, Soll %d", eb.versatz, unten-eb.liste.Dy())
	}
	// Oberste gewählt, die Liste steht unten: zurück nach oben.
	l.e.s.WaehleEbene(eb.zeilen[0].id)
	eb.versatz = eb.inhaltH - eb.liste.Dy()
	eb.zeigeGewaehlte = true
	l.setle()
	_, oben, _ := gewaehlt()
	if eb.versatz != oben {
		t.Errorf("gewählte Ebene oben: Versatz %d, Soll %d", eb.versatz, oben)
	}
	// Ohne die Anforderung (Wahl per Klick) bleibt die Liste stehen.
	eb.versatz = 3 * eb.zeileH
	l.setle()
	if eb.versatz != 3*eb.zeileH {
		t.Errorf("ohne Anforderung: Versatz %d, Soll %d", eb.versatz, 3*eb.zeileH)
	}
}

// Doppelklick auf eine Zeile öffnet das Namensfeld, ein einzelner Klick nicht.
func TestEbenenDoppelklickBenennt(t *testing.T) {
	l, k := ebenenLauf(t)
	eb := &l.e.ebenen
	p := mitte(eb.zeilen[0].rahmen)
	l.klick(p)
	if eb.umbenennen != 0 || l.e.s.Auswahl != k {
		t.Fatalf("ein Klick: umbenennen %d, Auswahl %d", eb.umbenennen, l.e.s.Auswahl)
	}
	l.klick(p)
	if eb.umbenennen != k || !l.fokus(&eb.name) {
		t.Errorf("Doppelklick: umbenennen %d, Soll %d, Fokus im Feld %v", eb.umbenennen, k, l.fokus(&eb.name))
	}
}

// Verify V23-5: F2 bei einer leeren, neuen Texteingabe: Sie wird beim Bestätigen verworfen, die Auswahl ist weg, das
// Namensfeld bleibt zu (kein Feld für Ebene 0).
func TestEbenenUmbenennenNachLeererTexteingabe(t *testing.T) {
	l, _ := ebenenLauf(t)
	l.e.s.Auswahl = 0
	l.e.s.SetzeWerkzeug(canvas.WerkzeugText)
	l.setle()
	f := l.e.leinwandFlaeche(l.gtx())
	p := l.e.s.Abb.ZuFenster(document.Pt(35, 3))
	l.klick(f.Min.Add(image.Pt(int(p.X), int(p.Y))))
	if !l.e.s.TextLaeuft() || !l.e.s.KannEbene() {
		t.Fatalf("keine leere Texteingabe: läuft %v, Ebene %v", l.e.s.TextLaeuft(), l.e.s.KannEbene())
	}
	n := len(l.e.s.D.Objekte())
	l.e.umbenennenBeginnen(l.gtx())
	l.setle()
	eb := &l.e.ebenen
	if eb.umbenennen != 0 || eb.umbenennenFokus || eb.name.neu || l.e.s.Auswahl != 0 || l.e.s.TextLaeuft() {
		t.Errorf("nach F2: umbenennen %d, Fokus-Anforderung %v, neu %v, Auswahl %d, Text läuft %v", eb.umbenennen,
			eb.umbenennenFokus, eb.name.neu, l.e.s.Auswahl, l.e.s.TextLaeuft())
	}
	if m := len(l.e.s.D.Objekte()); m != n-1 {
		t.Errorf("Objekte %d, Soll %d (leerer Text verworfen)", m, n-1)
	}
}
