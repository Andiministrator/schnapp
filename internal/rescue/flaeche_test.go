// SPDX-License-Identifier: GPL-3.0-or-later

package rescue

import (
	"bytes"
	"errors"
	"image"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

// leeresDok legt ein Dokument ohne Inhalt an (schneller als bild für große Maße).
func leeresDok(t *testing.T, w, h int) *document.Dokument {
	t.Helper()
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, w, h)))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// F-412 (a): Eine präparierte Kopie mit Schwärzungen über dem Flächenbudget wird in unter einer Sekunde abgelehnt,
// bevor ein Pixel angefasst wird; das Dokument bleibt unverändert.
func TestAnwendenFlaechenbudget(t *testing.T) {
	const seite = 4096 // 64 MiB je vollflächiger Schwärzung; 9 davon überschreiten 512 MiB
	for _, n := range []int{9, 100, maxSchwaerzungen} {
		d := leeresDok(t, seite, seite)
		kp, err := Erzeuge(d, Kennung{Pfad: "/x"}, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < n; i++ {
			kp.Schwaerzungen = append(kp.Schwaerzungen, Schwaerzung{Bereich: [4]int{0, 0, seite, seite}, Farbe: "#000000ff"})
		}
		stand, platz := d.Stand(), d.VerlaufPlatz()
		start := time.Now()
		err = Anwenden(d, &kp, nil)
		dauer := time.Since(start)
		if !errors.Is(err, ErrZuGross) {
			t.Fatalf("%d Schwärzungen: %v, erwartet ErrZuGross", n, err)
		}
		if dauer > time.Second {
			t.Errorf("%d Schwärzungen: Ablehnung nach %v, erwartet unter 1 s", n, dauer)
		}
		if d.Stand() != stand || d.Geschwaerzt() || d.VerlaufPlatz() != platz {
			t.Errorf("%d Schwärzungen: Dokument verändert", n)
		}
	}
}

// F-412: Präparierte Bereiche weit außerhalb des Bilds zählen nur mit dem Teil im Bild (kein Überlauf beim Summieren),
// eine ungültige Schwärzung wird vorab abgelehnt, auch hinter vielen gültigen.
func TestAnwendenFlaecheImBild(t *testing.T) {
	d := leeresDok(t, 64, 32)
	kp, err := Erzeuge(d, Kennung{Pfad: "/x"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	riesig := 1 << 30
	kp.Schwaerzungen = append(kp.Schwaerzungen, Schwaerzung{Bereich: [4]int{-riesig, -riesig, riesig, riesig}, Farbe: "#000000ff"})
	if err := Anwenden(d, &kp, nil); err != nil {
		t.Fatalf("Bereich über das Bild hinaus: %v", err)
	}
	d2 := leeresDok(t, 64, 32)
	kp.Schwaerzungen = append(kp.Schwaerzungen, Schwaerzung{Bereich: [4]int{0, 0, 4, 4}, Block: 8, Farbe: "#000000ff"})
	if err := Anwenden(d2, &kp, nil); err == nil || d2.Geschwaerzt() {
		t.Errorf("ungültige Verpixelung hinter einer gültigen: Fehler %v, geschwärzt %v", err, d2.Geschwaerzt())
	}
}

// F-412: Der Probelauf am Abzug trägt die Angaben der Kopie (die Schwärzungen laufen nur noch am Dokument); die
// Prüfung (render.Pruefe) sieht also die Objekte der Kopie, nicht den Stand davor.
func TestAnwendenPruefungSiehtAngaben(t *testing.T) {
	_, k := ausgang(t, "datei")
	d, _ := document.Neu(bild(64, 64))
	bearbeite(t, d)
	kp, _ := Erzeuge(d, k, time.Now())
	ziel, _ := document.Neu(bild(64, 64))
	vorher := string(ziel.OriginalBild().Pix)
	nein := errors.New("Objekte zu teuer")
	err := Anwenden(ziel, &kp, func(p *document.Dokument) error {
		if len(p.Objekte()) > 0 {
			return nein
		}
		return nil
	})
	if !errors.Is(err, nein) {
		t.Fatalf("Prüfung sah die Objekte der Kopie nicht: %v", err)
	}
	if ziel.KannRueckgaengig() || string(ziel.OriginalBild().Pix) != vorher {
		t.Error("Dokument trotz abgelehnter Prüfung geändert")
	}
}

// F-412 (b): Innerhalb des Budgets bleibt jede wiederhergestellte Schwärzung im Verlauf: Rückgängig führt auf den
// gespeicherten Stand (Pixel wie vorher), und Erzeuge schreibt danach weiter Kopien (kein ErrUnfertig).
func TestVerlaufNachWiederherstellen(t *testing.T) {
	const seite = 1024
	vorher := bild(seite, seite)
	d, err := document.Neu(vorher)
	if err != nil {
		t.Fatal(err)
	}
	original := append([]byte(nil), d.OriginalBild().Pix...)
	kp, err := Erzeuge(d, Kennung{Pfad: "/x"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		kp.Schwaerzungen = append(kp.Schwaerzungen, Schwaerzung{Bereich: [4]int{0, 0, seite, seite}, Farbe: "#000000ff"})
	}
	kp.Schwaerzungen = append(kp.Schwaerzungen, Schwaerzung{Bereich: [4]int{8, 8, 200, 100}, Block: 16, Farbe: "#000000ff"})
	stand := d.Stand()
	if err := Anwenden(d, &kp, nil); err != nil {
		t.Fatal(err)
	}
	if liste, voll := d.SchwaerzungenImVerlauf(); !voll || len(liste) != 5 {
		t.Fatalf("nach dem Wiederherstellen %d Schwärzungen im Verlauf, vollständig %v", len(liste), voll)
	}
	neu, err := Erzeuge(d, Kennung{Pfad: "/x"}, time.Now())
	if err != nil || len(neu.Schwaerzungen) != 5 {
		t.Fatalf("Erzeuge nach dem Wiederherstellen: %v, %d Schwärzungen", err, len(neu.Schwaerzungen))
	}
	for d.KannRueckgaengig() {
		if err := d.Rueckgaengig(); err != nil {
			t.Fatal(err)
		}
	}
	if d.Stand() != stand || !bytes.Equal(d.OriginalBild().Pix, original) {
		t.Error("Rückgängig führt nicht auf den gespeicherten Stand zurück")
	}
}
