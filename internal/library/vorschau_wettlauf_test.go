// SPDX-License-Identifier: GPL-3.0-or-later

package library

// F-651 (Sitzungs-QA 24.11., Kritiker B, KB-5): ErzeugeVorschau läuft unter b.mu und schreibt nur, solange die Datei
// den beim Prüfen gelesenen Stand behält. Sonst legte es nach einem Speichern mit Schwärzung den ungeschwärzten Stand
// in den Vorschau-Cache oder ließe die Vorschau einer entfernten Aufnahme als Waise neu entstehen (Kernprinzip 2).

import (
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
)

// weisseAufnahme legt eine weiße Aufnahme ab und entfernt ihre Vorschau (Cache geleert).
func weisseAufnahme(t *testing.T, b *Bibliothek) (*document.Dokument, Eintrag, Stand) {
	t.Helper()
	weiss := image.NewNRGBA(image.Rect(0, 0, 400, 200))
	for i := range weiss.Pix {
		weiss.Pix[i] = 255
	}
	d, err := document.Neu(weiss)
	if err != nil {
		t.Fatal(err)
	}
	e, err := b.Ablegen(d, d.Original(), fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")})
	if err != nil {
		t.Fatal(err)
	}
	stand, err := StandVon(filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad)))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(b.VorschauPfad(e.Pfad)); err != nil {
		t.Fatal(err)
	}
	return d, e, stand
}

// halteKodierer lässt den ersten Aufruf des Vorschau-Kodierers anhalten: erreicht schließt beim Eintritt, weiter gibt
// ihn frei.
func halteKodierer(t *testing.T) (erreicht, weiter chan struct{}) {
	t.Helper()
	var mu sync.Mutex
	n := 0
	erreicht, weiter = make(chan struct{}), make(chan struct{})
	alt := kodiereVorschau
	kodiereVorschau = func(w io.Writer, m image.Image) error {
		mu.Lock()
		n++
		erster := n == 1
		mu.Unlock()
		if erster {
			close(erreicht)
			<-weiter
		}
		return png.Encode(w, m)
	}
	t.Cleanup(func() { kodiereVorschau = alt })
	return erreicht, weiter
}

func warteAuf(t *testing.T, ch chan struct{}, was string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(10 * time.Second):
		t.Fatalf("%s: nicht erreicht", was)
	}
}

func mitteHell(t *testing.T, pfad string) bool {
	t.Helper()
	f, err := os.Open(pfad)
	if err != nil {
		t.Fatalf("keine Vorschau: %v", err)
	}
	defer f.Close()
	bild, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	r, g, bl, _ := bild.At(bild.Bounds().Dx()/2, bild.Bounds().Dy()/2).RGBA()
	return r > 0x8000 && g > 0x8000 && bl > 0x8000
}

// Speichern mit Schwärzung während ErzeugeVorschau liest: Speichern wartet, danach zeigt der Cache den geschwärzten
// Stand.
func TestErzeugeVorschauWettlaufSchwaerzen(t *testing.T) {
	b, _ := neueBibliothek(t)
	d, e, stand := weisseAufnahme(t, b)
	erreicht, weiter := halteKodierer(t)
	fertig := make(chan struct{})
	go func() {
		defer close(fertig)
		b.ErzeugeVorschau(e.Pfad) // wie libview.LaderFuer in der Goroutine des Vorschauspeichers
	}()
	warteAuf(t, erreicht, "Kodierer")
	if err := d.Schwaerzen(image.Rect(0, 0, 400, 200), document.Schwaerzung{Farbe: color.NRGBA{A: 255}}); err != nil {
		t.Fatal(err)
	}
	speicherFertig := make(chan error, 1)
	go func() {
		_, _, err := b.Speichern(e.Pfad, stand, d, d.Original())
		speicherFertig <- err
	}()
	select {
	case err := <-speicherFertig:
		t.Errorf("Speichern kehrte zurück (%v), während ErzeugeVorschau noch schrieb: keine gemeinsame Sperre", err)
	case <-time.After(time.Second):
	}
	close(weiter)
	<-fertig
	select {
	case err := <-speicherFertig:
		if err != nil {
			t.Fatalf("Speichern: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Speichern kehrt nicht zurück")
	}
	if mitteHell(t, b.VorschauPfad(e.Pfad)) {
		t.Error("Vorschau im Cache zeigt nach dem Speichern der Schwärzung den ungeschwärzten Stand")
	}
}

// Entfernen während ErzeugeVorschau liest: Danach gibt es weder Datei noch Vorschau (keine Waise).
func TestErzeugeVorschauWettlaufEntfernen(t *testing.T) {
	b, _ := neueBibliothek(t)
	_, e, stand := weisseAufnahme(t, b)
	erreicht, weiter := halteKodierer(t)
	fertig := make(chan struct{})
	go func() {
		defer close(fertig)
		b.ErzeugeVorschau(e.Pfad)
	}()
	warteAuf(t, erreicht, "Kodierer")
	entfernt := make(chan error, 1)
	go func() {
		_, err := b.Entfernen(e.Pfad, stand)
		entfernt <- err
	}()
	time.Sleep(100 * time.Millisecond)
	close(weiter)
	<-fertig
	if err := <-entfernt; err != nil {
		t.Fatalf("Entfernen: %v", err)
	}
	if _, err := os.Lstat(b.VorschauPfad(e.Pfad)); err == nil {
		t.Error("Vorschau einer entfernten Aufnahme liegt als Waise im Cache")
	}
}

// Ein anderer Prozess ändert die Datei, während ErzeugeVorschau kodiert: Die Vorschau wird nicht umbenannt.
func TestErzeugeVorschauStandVorDemUmbenennen(t *testing.T) {
	b, _ := neueBibliothek(t)
	_, e, _ := weisseAufnahme(t, b)
	voll := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
	altKodierer := kodiereVorschau
	t.Cleanup(func() { kodiereVorschau = altKodierer })
	kodiereVorschau = func(w io.Writer, m image.Image) error {
		// „zweiter Prozess“ schreibt an Ort und Stelle (andere Zeit)
		spaeter := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
		if err := os.Chtimes(voll, spaeter, spaeter); err != nil {
			t.Error(err)
		}
		return png.Encode(w, m)
	}
	umbenannt := false
	ersetzeUmbenenne(t, func(alt, neu string) error {
		if neu == b.VorschauPfad(e.Pfad) {
			umbenannt = true
		}
		return os.Rename(alt, neu)
	})
	b.ErzeugeVorschau(e.Pfad)
	if umbenannt {
		t.Error("Vorschau trotz geändertem Stand umbenannt")
	}
	if _, err := os.Lstat(b.VorschauPfad(e.Pfad)); err == nil {
		t.Error("Vorschau eines veralteten Stands geschrieben")
	}
	if reste, _ := filepath.Glob(filepath.Join(b.Orte.Vorschau, tempPraefix+"*")); len(reste) != 0 {
		t.Errorf("temporäre Datei liegt: %v", reste)
	}
}

// Ein anderer Prozess ersetzt die Datei genau zwischen Prüfen und Umbenennen: Die eben geschriebene Vorschau wird
// wieder entfernt.
func TestErzeugeVorschauStandNachDemUmbenennen(t *testing.T) {
	b, _ := neueBibliothek(t)
	_, e, _ := weisseAufnahme(t, b)
	voll := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
	ersetzeUmbenenne(t, func(alt, neu string) error {
		err := os.Rename(alt, neu)
		if neu == b.VorschauPfad(e.Pfad) {
			spaeter := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
			if err := os.Chtimes(voll, spaeter, spaeter); err != nil {
				t.Error(err)
			}
		}
		return err
	})
	b.ErzeugeVorschau(e.Pfad)
	if _, err := os.Lstat(b.VorschauPfad(e.Pfad)); err == nil {
		t.Error("Vorschau eines inzwischen ersetzten Stands bleibt im Cache")
	}
}

// Verweise (F-651, KM26): Eine Aufnahme als Symlink und ein Monatsordner als Symlink aus der Ablage hinaus bekommen
// keine Vorschau.
func TestErzeugeVorschauVerweise(t *testing.T) {
	b, _ := neueBibliothek(t)
	_, e, _ := weisseAufnahme(t, b)
	voll := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
	// Datei als Verweis auf eine echte Aufnahme in der Ablage.
	verweis := filepath.Join(filepath.Dir(voll), "verweis.png")
	if err := os.Symlink(voll, verweis); err != nil {
		t.Fatal(err)
	}
	rel := filepath.ToSlash(filepath.Join(filepath.Dir(e.Pfad), "verweis.png"))
	b.ErzeugeVorschau(rel)
	if _, err := os.Lstat(b.VorschauPfad(rel)); err == nil {
		t.Error("Vorschau für einen Verweis geschrieben")
	}
	// Monatsordner als Verweis auf einen Ordner außerhalb der Ablage mit einer gültigen Arbeitsdatei.
	aussen := filepath.Join(filepath.Dir(b.Orte.Ablage), "aussen")
	if err := os.Mkdir(aussen, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(voll, filepath.Join(aussen, "fremd.png")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(aussen, filepath.Join(b.Orte.Ablage, "2026-10")); err != nil {
		t.Fatal(err)
	}
	b.ErzeugeVorschau("2026-10/fremd.png")
	if _, err := os.Lstat(b.VorschauPfad("2026-10/fremd.png")); err == nil {
		t.Error("Vorschau für eine Datei hinter einem Ordner-Verweis aus der Ablage hinaus geschrieben")
	}
	// Gegenprobe: die echte Aufnahme bekommt ihre Vorschau.
	b.ErzeugeVorschau(e.Pfad)
	if _, err := os.Lstat(b.VorschauPfad(e.Pfad)); err != nil {
		t.Errorf("echte Aufnahme ohne Vorschau: %v", err)
	}
}
