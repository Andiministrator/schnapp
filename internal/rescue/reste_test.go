// SPDX-License-Identifier: GPL-3.0-or-later

package rescue

// Tests zu den Rettungskopie-Resten aus Sprint 11 (F-410, F-413, F-417). Die Mutationsprobe
// scripts/mutanten/rescue.json prüft, dass sie die Fixes tragen.

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/Andiministrator/schnapp/internal/document"
)

// warteKette wartet, bis alle eingereihten Aufträge von s fertig sind, ohne die Sperre freizugeben (anders als Ende).
func warteKette(t *testing.T, s *Sicherung) {
	t.Helper()
	s.mu.Lock()
	k := s.kette
	s.mu.Unlock()
	if k == nil {
		return
	}
	select {
	case <-k:
	case <-time.After(10 * time.Second):
		t.Fatal("Kette nicht fertig")
	}
}

// mitText fügt d einen Text hinzu (eine Änderung, ein neuer Stand).
func mitText(t *testing.T, d *document.Dokument) {
	t.Helper()
	if _, err := d.Hinzufuegen(document.Text{Ort: document.Pt(5, 20), Inhalt: "t", Schriftgrad: 10}); err != nil {
		t.Fatal(err)
	}
}

// F-410: Ein Pfad ohne gültiges UTF-8 (alter Latin-1-Dateiname) übersteht Schreiben und Lesen verlustfrei; die Kopie
// wird beim nächsten Öffnen angewendet.
func TestPfadOhneUTF8(t *testing.T) {
	ort := t.TempDir()
	pfad := "/bilder/gr\xfc\xdfe.png"
	d, _ := document.Neu(bild(64, 64))
	bearbeite(t, d)
	k := Kennung{Pfad: pfad, Groesse: 1, SHA256: strings.Repeat("ab", 32)}
	s := NeueSicherung(ort, k)
	s.Sichere(d)
	s.Ende()
	kp, err := Lies(ort, pfad)
	if err != nil {
		t.Fatalf("eigene Kopie nicht wieder lesbar: %v", err)
	}
	if kp.Datei.Pfad != pfad {
		t.Errorf("Pfad %q, erwartet %q", kp.Datei.Pfad, pfad)
	}
	daten, _ := os.ReadFile(filepath.Join(ort, Name(pfad)))
	if !utf8.Valid(daten) || !bytes.Contains(daten, []byte(`"pfad_roh"`)) {
		t.Errorf("JSON ohne rohen Pfad oder ungültig: %s", daten)
	}
	neu, _ := document.Neu(bild(64, 64))
	s2 := NeueSicherung(ort, k)
	w := s2.Wiederherstellen(neu, nil)
	s2.Ende()
	if !w.Angewendet || len(neu.Objekte()) != 2 {
		t.Errorf("nicht angewendet: %+v", w)
	}
}

// F-410: Ein gültiger Pfad steht wie bisher nur in pfad (ältere Kopien und ältere Fassungen lesen sich gleich); eine
// Kopie ohne pfad_roh liest sich unverändert, unbekannte Felder in der Kennung bleiben abgelehnt.
func TestKennungJSONAbwaertsvertraeglich(t *testing.T) {
	k := Kennung{Pfad: "/bilder/grüße.png", Groesse: 7, Zeit: time.Unix(1700000000, 5).UTC(), SHA256: "00"}
	daten, err := json.Marshal(k)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(daten, []byte("pfad_roh")) {
		t.Errorf("gültiger Pfad mit pfad_roh: %s", daten)
	}
	alt := `{"pfad":"/bilder/grüße.png","groesse":7,"zeit":"2023-11-14T22:13:20.000000005Z","sha256":"00"}`
	var o Kennung
	if err := json.Unmarshal([]byte(alt), &o); err != nil || !o.Gleich(k) {
		t.Errorf("alte Kennung: %v %+v", err, o)
	}
	if err := json.Unmarshal([]byte(`{"pfad":"/a","fremd":1}`), &o); err == nil {
		t.Error("unbekanntes Feld in der Kennung angenommen")
	}
}

// F-413: Ein zweites Fenster auf derselben Datei wendet die junge Kopie des ersten nicht an, legt sie nicht beiseite,
// überschreibt und löscht sie nicht. Nach dem Ende (oder Absturz) des ersten wendet ein neues Fenster sie an.
func TestZweitesFensterLaesstKopieInRuhe(t *testing.T) {
	pfad, k := ausgang(t, "datei")
	ort := t.TempDir()
	d, _ := document.Neu(bild(64, 64))
	bearbeite(t, d)
	a := NeueSicherung(ort, k)
	a.Sichere(d)
	warteKette(t, a)
	kopie := filepath.Join(ort, Name(pfad))
	vorher, err := os.ReadFile(kopie)
	if err != nil {
		t.Fatal(err)
	}

	b := NeueSicherung(ort, k)
	neu, _ := document.Neu(bild(64, 64))
	w := b.Wiederherstellen(neu, nil)
	if !w.Fremd || w.Angewendet || w.Liegt || len(neu.Objekte()) != 0 || neu.KannRueckgaengig() {
		t.Errorf("zweites Fenster: %+v, %d Objekte", w, len(neu.Objekte()))
	}
	mitText(t, neu)
	b.Sichere(neu)
	b.Verwirf()
	b.NeueBasis(k) // Speichern auf dieselbe Datei: die Sperre hält weiter das erste Fenster
	b.Ende()
	if nachher, err := os.ReadFile(kopie); err != nil || !bytes.Equal(nachher, vorher) {
		t.Errorf("Kopie des ersten Fensters berührt: %v", err)
	}
	if reste, _ := os.ReadDir(ort); len(reste) != 2 { // Kopie und Sperre des ersten Fensters
		t.Errorf("Ordner: %v", reste)
	}

	// Absturz des ersten Fensters: Der Kern gibt die Sperre frei, die Sperrdatei bleibt liegen.
	_ = a.sperre.f.Close()
	c := NeueSicherung(ort, k)
	neu2, _ := document.Neu(bild(64, 64))
	if w := c.Wiederherstellen(neu2, nil); !w.Angewendet || w.Fremd || len(neu2.Objekte()) != 2 {
		t.Errorf("nach dem Ende des ersten Fensters nicht angewendet: %+v", w)
	}
	c.Verwirf()
	c.Ende()
	if reste, _ := os.ReadDir(ort); len(reste) != 0 {
		t.Errorf("Reste nach dem Ende: %v", reste)
	}
}

// F-413: Ende gibt die Sperre frei; NeueBasis auf eine andere Datei nimmt deren Sperre und gibt die alte frei.
func TestSperreFreiNachEndeUndNeueBasis(t *testing.T) {
	_, k := ausgang(t, "datei")
	_, k2 := ausgang(t, "neu")
	ort := t.TempDir()
	a := NeueSicherung(ort, k)
	a.NeueBasis(k2)
	warteKette(t, a)
	b := NeueSicherung(ort, k) // die alte Ausgangsdatei ist frei
	if b.Wiederherstellen(nil, nil).Fremd {
		t.Error("alte Ausgangsdatei nach NeueBasis noch gesperrt")
	}
	b.Ende()
	c := NeueSicherung(ort, k2)
	if !c.Wiederherstellen(nil, nil).Fremd {
		t.Error("neue Ausgangsdatei nicht gesperrt")
	}
	c.Ende()
	a.Ende()
	d := NeueSicherung(ort, k2)
	if d.Wiederherstellen(nil, nil).Fremd {
		t.Error("nach Ende noch gesperrt")
	}
	d.Ende()
}

// F-413: Aufräumen entfernt eine alte Sperrdatei nur, wenn niemand sie hält.
func TestAufraeumenSperren(t *testing.T) {
	ort := t.TempDir()
	a := NeueSicherung(ort, Kennung{Pfad: "/gehalten"})
	verwaist := filepath.Join(ort, sperrName("/verwaist"))
	if err := os.WriteFile(verwaist, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	gehalten := filepath.Join(ort, sperrName("/gehalten"))
	z := time.Now().Add(-Haltezeit - time.Hour)
	for _, p := range []string{verwaist, gehalten} {
		if err := os.Chtimes(p, z, z); err != nil {
			t.Fatal(err)
		}
	}
	if n, err := Aufraeumen(ort, time.Now()); n != 1 || err != nil {
		t.Errorf("%d entfernt, %v; erwartet 1", n, err)
	}
	if _, err := os.Stat(verwaist); !os.IsNotExist(err) {
		t.Errorf("verwaiste Sperre liegt noch: %v", err)
	}
	if _, err := os.Stat(gehalten); err != nil {
		t.Errorf("gehaltene Sperre entfernt: %v", err)
	}
	a.Ende()
}

// haltenderSchreiber ersetzt das Schreiben von s durch eines, das bis zum Freigeben hängt (wie ein Sync auf einem
// hängenden Zustandsordner, F-417). Zurück kommen der Zähler der begonnenen Schreibvorgänge und das Freigeben.
func haltenderSchreiber(t *testing.T, s *Sicherung) (*atomic.Int32, func()) {
	t.Helper()
	halt := make(chan struct{})
	var einmal sync.Once
	frei := func() { einmal.Do(func() { close(halt) }) }
	t.Cleanup(frei)
	var n atomic.Int32
	s.schreibe = func(ort string, kp Kopie) error {
		n.Add(1)
		<-halt
		return Schreibe(ort, kp)
	}
	return &n, frei
}

// wartetAuf wartet, bis bed gilt (höchstens 5 s).
func wartetAuf(t *testing.T, bed func() bool) {
	t.Helper()
	for ende := time.Now().Add(5 * time.Second); !bed(); time.Sleep(time.Millisecond) {
		if time.Now().After(ende) {
			t.Fatal("Bedingung nicht erreicht")
		}
	}
}

// F-417: Ende kehrt nach der Frist zurück, auch wenn ein Schreiben hängt, und meldet ErrFrist.
func TestEndeMitFrist(t *testing.T) {
	alt := endeFrist
	endeFrist = 50 * time.Millisecond
	t.Cleanup(func() { endeFrist = alt })
	_, k := ausgang(t, "datei")
	s := NeueSicherung(t.TempDir(), k)
	var mu sync.Mutex
	var diagnose []error
	s.Diagnose = func(err error) { mu.Lock(); diagnose = append(diagnose, err); mu.Unlock() }
	_, frei := haltenderSchreiber(t, s)
	d, _ := document.Neu(bild(32, 32))
	mitText(t, d)
	s.Sichere(d)
	erg := make(chan bool, 1)
	go func() { erg <- s.Ende() }()
	select {
	case ok := <-erg:
		if ok {
			t.Error("Ende meldet fertig, obwohl das Schreiben hängt")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Ende wartet ohne Frist auf ein hängendes Schreiben")
	}
	mu.Lock()
	if len(diagnose) != 1 || !errors.Is(diagnose[0], ErrFrist) {
		t.Errorf("Diagnose %v, erwartet ErrFrist", diagnose)
	}
	mu.Unlock()
	frei()
	endeFrist = alt
	if !s.Ende() {
		t.Error("nach dem Freigeben nicht fertig")
	}
}

// F-417: Verwirf kehrt sofort zurück, auch wenn ein Schreiben hängt; das Löschen läuft danach in der Reihenfolge der
// Aufrufe: Die hängende Kopie wird gelöscht, eine danach gewollte bleibt, von mehreren eingereihten nur die jüngste.
// Eine vor Verwirf eingereihte, noch nicht begonnene Kopie verfällt (gen), statt geschrieben und gleich gelöscht zu
// werden.
func TestVerwirfWartetNicht(t *testing.T) {
	for _, danach := range []int{0, 2} {
		pfad, k := ausgang(t, "datei")
		ort := t.TempDir()
		s := NeueSicherung(ort, k)
		n, frei := haltenderSchreiber(t, s)
		d, _ := document.Neu(bild(32, 32))
		mitText(t, d)
		s.Sichere(d)
		wartetAuf(t, func() bool { return n.Load() == 1 }) // das Schreiben hängt
		mitText(t, d)
		s.Sichere(d) // eingereiht, noch nicht begonnen: verfällt mit Verwirf
		fertig := make(chan struct{})
		go func() { s.Verwirf(); close(fertig) }()
		select {
		case <-fertig:
		case <-time.After(2 * time.Second):
			t.Fatal("Verwirf blockiert die Ereignisschleife, solange ein Schreiben hängt")
		}
		for i := 0; i < danach; i++ {
			mitText(t, d)
			s.Sichere(d)
		}
		frei()
		warteKette(t, s)
		kp, err := Lies(ort, pfad)
		switch {
		case danach == 0 && !errors.Is(err, ErrKeine):
			t.Errorf("hängende Kopie nach Verwirf nicht gelöscht: %v", err)
		case danach > 0 && err != nil:
			t.Errorf("Kopie nach Verwirf fehlt: %v", err)
		case danach > 0:
			neu, _ := document.Neu(bild(32, 32))
			if err := Anwenden(neu, kp, nil); err != nil || len(neu.Objekte()) != 2+danach {
				t.Errorf("nicht die jüngste Kopie: %d Objekte, %v", len(neu.Objekte()), err)
			}
		}
		if soll := int32(min(danach, 1) + 1); n.Load() != soll {
			t.Errorf("danach %d: %d Schreibvorgänge, erwartet %d (eingereihte ältere Kopien übersprungen)", danach, n.Load(), soll)
		}
		s.Ende()
	}
}
