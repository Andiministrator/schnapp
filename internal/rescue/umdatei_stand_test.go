// SPDX-License-Identifier: GPL-3.0-or-later

package rescue

// F-757: UmDatei zieht die Kopie nur bei bestätigtem Stand nach, und ein Fenster, das die Aufnahme während UmDatei
// öffnet, wartet auf die Marke, statt fremd zu werden. Mutanten: scripts/mutanten/rescue.json (N6 bis N8, N12; N9 und
// N10 in cmd/schnapp). Marke als flock-Sperre und Aufräumen: umdatei_marke_test.go (F-770, F-771).

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Ersetzt ein anderer Schreiber die Datei, bevor aendere sie liest (aendere bestätigt den Stand nicht), zieht UmDatei die
// Kopie nicht nach: Sie würde sonst auf einen fremden Inhalt angewendet.
func TestUmDateiZiehtBeiFremdemInhaltNichtNach(t *testing.T) {
	pfad, ort, _ := aufnahmeMitKopie(t)
	err := UmDatei(ort, pfad, func(func() bool) error {
		ueberschreibe(t, pfad, "GANZ ANDERES BILD, andere Objekte, andere Größe der Datei")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	k, _ := KennungVon(pfad)
	kp, err := Lies(ort, pfad)
	if err != nil || kp.Datei.Gleich(k) {
		t.Fatalf("Kopie trägt die Kennung des fremden Stands: %v", err)
	}
}

// Wird die Datei zwischen der Kennung vor aendere und der Bestätigung ersetzt, meldet unveraendert false, und die Kopie
// wird nicht nachgezogen.
func TestUmDateiBestaetigungMeldetErsetzteDatei(t *testing.T) {
	pfad, ort, _ := aufnahmeMitKopie(t)
	var gemeldet bool
	err := UmDatei(ort, pfad, func(unv func() bool) error {
		ueberschreibe(t, pfad, "ersetzt vor dem Lesen")
		gemeldet = unv()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	k, _ := KennungVon(pfad)
	if kp, err := Lies(ort, pfad); gemeldet || err != nil || kp.Datei.Gleich(k) {
		t.Errorf("unveraendert=%v, Kopie nachgezogen: %v", gemeldet, err)
	}
}

// Ein Fenster, das die Aufnahme während UmDatei öffnet, wartet in WarteUmDatei, bekommt danach die Sperre und wendet die
// nachgezogene Kopie an, statt fremd zu werden.
func TestFensterWaehrendUmDateiWartetUndWendetAn(t *testing.T) {
	pfad, ort, _ := aufnahmeMitKopie(t)
	drin, weiter := make(chan struct{}), make(chan struct{})
	fertig := make(chan error, 1)
	go func() {
		fertig <- UmDatei(ort, pfad, func(unv func() bool) error {
			unv()
			close(drin)
			<-weiter
			ueberschreibe(t, pfad, "bild mit schlagwort b, länger")
			return nil
		})
	}()
	<-drin
	ergebnis := make(chan Wiederherstellung, 1)
	go func() {
		WarteUmDatei(ort, pfad, 10*time.Second)
		k, err := KennungVon(pfad)
		if err != nil {
			t.Error(err)
		}
		s := NeueSicherung(ort, k)
		defer s.Ende()
		neu, _ := document.Neu(bild(64, 64))
		ergebnis <- s.Wiederherstellen(neu, nil)
	}()
	select {
	case w := <-ergebnis:
		t.Fatalf("das Fenster hat nicht gewartet: %+v", w)
	case <-time.After(300 * time.Millisecond):
	}
	close(weiter)
	if err := <-fertig; err != nil {
		t.Fatal(err)
	}
	w := <-ergebnis
	if w.Fremd || !w.Angewendet {
		t.Errorf("Fenster nach UmDatei: fremd=%v angewendet=%v grund=%v", w.Fremd, w.Angewendet, w.Grund)
	}
}

// Ohne Marke oder mit einer liegen gebliebenen (nicht gehaltenen) Marke kehrt WarteUmDatei sofort zurück.
func TestWarteUmDateiOhneMarkeKehrtSofortZurueck(t *testing.T) {
	ort := t.TempDir()
	t0 := time.Now()
	WarteUmDatei(ort, "/x", 5*time.Second)
	marke := filepath.Join(ort, umMarkeName("/x"))
	if err := os.WriteFile(marke, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	alt := time.Now().Add(-time.Hour)
	if err := os.Chtimes(marke, alt, alt); err != nil {
		t.Fatal(err)
	}
	WarteUmDatei(ort, "/x", 5*time.Second)
	if d := time.Since(t0); d > time.Second {
		t.Errorf("WarteUmDatei wartete %v", d)
	}
}
