// SPDX-License-Identifier: GPL-3.0-or-later

package rescue

// Tests zur Sperre und zur Kette der Sicherung aus Karte 198 (F-436, F-440, F-446, F-451). Die Mutationsprobe
// scripts/mutanten/rescue.json prüft, dass sie die Fixes tragen. Die Tests ändern Nähte (flockFn, sperrHaken,
// budgetPruefer) und laufen deshalb nicht parallel.

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

// diagnosen sammelt die Fehler der Diagnose einer Sicherung.
type diagnosen struct {
	mu   sync.Mutex
	list []error
}

func (g *diagnosen) fuer(s *Sicherung) {
	s.Diagnose = func(err error) {
		g.mu.Lock()
		g.list = append(g.list, err)
		g.mu.Unlock()
	}
}

func (g *diagnosen) hat(ziel error) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, e := range g.list {
		if errors.Is(e, ziel) {
			return true
		}
	}
	return false
}

// fremdeKopie legt eine Kopie des "anderen Fensters" zur Ausgangsdatei k an und liefert ihren Inhalt.
func fremdeKopie(t *testing.T, ort string, k Kennung) []byte {
	t.Helper()
	d, _ := document.Neu(bild(64, 64))
	bearbeite(t, d)
	kp, err := Erzeuge(d, k, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := Schreibe(ort, kp); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(ort, Name(k.Pfad)))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// pruefeStumm prüft, dass die Sitzung s eine vorhandene Kopie weder anwendet, beiseitelegt, überschreibt noch löscht.
func pruefeStumm(t *testing.T, s *Sicherung, ort string, k Kennung, vorher []byte) {
	t.Helper()
	neu, _ := document.Neu(bild(64, 64))
	if w := s.Wiederherstellen(neu, nil); !w.Gestoert || w.Angewendet || w.Liegt || len(neu.Objekte()) != 0 {
		t.Errorf("Wiederherstellen: %+v", w)
	}
	mitText(t, neu)
	s.Sichere(neu)
	s.Verwirf()
	s.NeueBasis(k) // scheitert die Sperre weiter, bleibt die Sitzung stumm
	s.Sichere(neu)
	s.Ende()
	if nachher, err := os.ReadFile(filepath.Join(ort, Name(k.Pfad))); err != nil || !bytes.Equal(nachher, vorher) {
		t.Errorf("Kopie berührt oder gelöscht: %v", err)
	}
	reste, _ := os.ReadDir(ort)
	for _, e := range reste {
		if strings.HasSuffix(e.Name(), ".alt.json") {
			t.Errorf("Kopie beiseitegelegt: %s", e.Name())
		}
	}
}

// F-440: Ein Symlink am Sperrnamen (O_NOFOLLOW, ELOOP) macht die Sitzung stumm: keine fremde Kopie anwenden,
// beiseitelegen, überschreiben oder löschen; der Symlink wird nicht verfolgt.
func TestSperreSymlinkMachtStumm(t *testing.T) {
	_, k := ausgang(t, "datei")
	ort := t.TempDir()
	vorher := fremdeKopie(t, ort, k)
	ziel := filepath.Join(t.TempDir(), "woanders")
	if err := os.WriteFile(ziel, []byte("unberuehrt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(ziel, filepath.Join(ort, sperrName(k.Pfad))); err != nil {
		t.Fatal(err)
	}
	pruefeStumm(t, NeueSicherung(ort, k), ort, k, vorher)
	if b, err := os.ReadFile(ziel); err != nil || string(b) != "unberuehrt" {
		t.Errorf("Ziel des Symlinks verändert: %q %v", b, err)
	}
	if fi, err := os.Lstat(filepath.Join(ort, sperrName(k.Pfad))); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("Symlink am Sperrnamen ersetzt oder entfernt: %v", err)
	}
}

// F-440: Ein hängender Symlink am Sperrnamen legt sein Ziel nicht an (O_NOFOLLOW zusammen mit O_CREATE), die Sitzung
// ist stumm.
func TestSperreHaengenderSymlink(t *testing.T) {
	_, k := ausgang(t, "datei")
	ort := t.TempDir()
	vorher := fremdeKopie(t, ort, k)
	ziel := filepath.Join(t.TempDir(), "nicht-da")
	if err := os.Symlink(ziel, filepath.Join(ort, sperrName(k.Pfad))); err != nil {
		t.Fatal(err)
	}
	pruefeStumm(t, NeueSicherung(ort, k), ort, k, vorher)
	if _, err := os.Lstat(ziel); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("das Ziel des Symlinks wurde angelegt: %v", err)
	}
}

// F-440: Eine Pipe am Sperrnamen hängt das Öffnen nicht auf und macht die Sitzung stumm.
func TestSperrePipeMachtStumm(t *testing.T) {
	_, k := ausgang(t, "datei")
	ort := t.TempDir()
	vorher := fremdeKopie(t, ort, k)
	if err := syscall.Mkfifo(filepath.Join(ort, sperrName(k.Pfad)), 0o600); err != nil {
		t.Fatal(err)
	}
	pruefeStumm(t, NeueSicherung(ort, k), ort, k, vorher)
}

// F-440: Ein Verzeichnis am Sperrnamen macht die Sitzung stumm.
func TestSperreVerzeichnisMachtStumm(t *testing.T) {
	_, k := ausgang(t, "datei")
	ort := t.TempDir()
	vorher := fremdeKopie(t, ort, k)
	if err := os.Mkdir(filepath.Join(ort, sperrName(k.Pfad)), 0o700); err != nil {
		t.Fatal(err)
	}
	pruefeStumm(t, NeueSicherung(ort, k), ort, k, vorher)
}

// F-440: Nur "kann kein flock" (ENOLCK, ENOTSUP, EOPNOTSUPP) lässt die Sitzung ohne Sperre weiterlaufen; jeder andere
// Fehler beim Sperren macht sie stumm.
func TestSperreFlockFehler(t *testing.T) {
	alt := flockFn
	t.Cleanup(func() { flockFn = alt })
	for _, f := range []struct {
		err   error
		stumm bool
	}{
		{syscall.ENOLCK, false}, {syscall.ENOTSUP, false}, {syscall.EOPNOTSUPP, false},
		{syscall.EACCES, true}, {syscall.EIO, true}, {syscall.EBADF, true},
	} {
		t.Run(f.err.Error(), func(t *testing.T) {
			flockFn = func(int, int) error { return f.err }
			pfad, k := ausgang(t, "datei")
			ort := t.TempDir()
			s := NeueSicherung(ort, k)
			d, _ := document.Neu(bild(64, 64))
			bearbeite(t, d)
			s.Sichere(d)
			s.Ende()
			_, err := os.Stat(filepath.Join(ort, Name(pfad)))
			if f.stumm && err == nil {
				t.Errorf("%v: stumme Sitzung hat geschrieben", f.err)
			}
			if !f.stumm && err != nil {
				t.Errorf("%v: Sitzung ohne flock hat nicht geschrieben: %v", f.err, err)
			}
		})
	}
}

// F-440: Ordner 0700, Sperrdatei 0600, auch nach dem Anlegen unter einer lockeren umask-Vorgabe des Aufrufers.
func TestSperreRechte(t *testing.T) {
	_, k := ausgang(t, "datei")
	ort := filepath.Join(t.TempDir(), "neu", "rescue")
	s := NeueSicherung(ort, k)
	defer s.Ende()
	if fi, err := os.Stat(ort); err != nil || fi.Mode().Perm() != 0o700 {
		t.Errorf("Ordner: %v %v", fi, err)
	}
	if fi, err := os.Stat(filepath.Join(ort, sperrName(k.Pfad))); err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("Sperrdatei: %v %v", fi, err)
	}
}

// F-440: Wird die Sperrdatei zwischen Sperren und Prüfen entfernt (Aufräumen eines anderen Fensters), sperrt der
// Deskriptor eine verwaiste Datei; sperreNehmen versucht es erneut und hält am Ende die Datei, die unter dem Namen liegt.
func TestSperreVerwaisteDatei(t *testing.T) {
	alt := sperrHaken
	t.Cleanup(func() { sperrHaken = alt })
	ort := t.TempDir()
	einmal := true
	sperrHaken = func(name string) {
		if einmal {
			einmal = false
			if err := os.Remove(name); err != nil {
				t.Error(err)
			}
		}
	}
	sp, err := sperreNehmen(ort, "/x/bild.png")
	if err != nil {
		t.Fatal(err)
	}
	defer sp.f.Close()
	if !gleicheDatei(sp.f, sp.name) {
		t.Fatal("gehaltene Sperre ist nicht die Datei unter dem Namen")
	}
	// Ein zweites Fenster sieht die Sperre: sie hängt an der Datei unter dem Namen.
	if _, err := sperreNehmen(ort, "/x/bild.png"); !errors.Is(err, ErrBelegt) {
		t.Errorf("zweite Sperre: %v, erwartet ErrBelegt", err)
	}
}

// F-440: raeumeSperre entfernt nur, was es geprüft hat: Ersetzt jemand den Namen nach dem Sperren durch eine andere
// Datei, bleibt sie liegen. Eine fehlende Sperrdatei legt es nicht an.
func TestRaeumeSperreNurGepruefte(t *testing.T) {
	alt := sperrHaken
	t.Cleanup(func() { sperrHaken = alt })
	ort := t.TempDir()
	name := filepath.Join(ort, sperrName("/x/bild.png"))

	if ok, err := raeumeSperre(name); ok || err != nil {
		t.Errorf("fehlende Sperrdatei: %v %v", ok, err)
	}
	if _, err := os.Lstat(name); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("raeumeSperre hat eine Sperrdatei angelegt: %v", err)
	}

	if err := os.WriteFile(name, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	sperrHaken = func(string) {
		andere := filepath.Join(ort, "andere")
		if err := os.WriteFile(andere, []byte("neu"), 0o600); err != nil {
			t.Error(err)
		}
		if err := os.Rename(andere, name); err != nil {
			t.Error(err)
		}
	}
	if ok, err := raeumeSperre(name); ok || err != nil {
		t.Errorf("ersetzte Sperrdatei: %v %v", ok, err)
	}
	if b, err := os.ReadFile(name); err != nil || string(b) != "neu" {
		t.Errorf("die untergeschobene Datei wurde entfernt: %q %v", b, err)
	}
	sperrHaken = alt
	if ok, err := raeumeSperre(name); !ok || err != nil {
		t.Errorf("freie Sperrdatei: %v %v", ok, err)
	}
	if _, err := os.Lstat(name); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("freie Sperrdatei nicht entfernt: %v", err)
	}
}

// F-440: frei entfernt den Namen nur, wenn er noch dieselbe Datei ist.
func TestSperreFreiNurGepruefte(t *testing.T) {
	ort := t.TempDir()
	sp, err := sperreNehmen(ort, "/x/bild.png")
	if err != nil {
		t.Fatal(err)
	}
	andere := filepath.Join(ort, "andere")
	if err := os.WriteFile(andere, []byte("neu"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(andere, sp.name); err != nil {
		t.Fatal(err)
	}
	if err := sp.frei(); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(sp.name); err != nil || string(b) != "neu" {
		t.Errorf("frei hat eine fremde Datei entfernt: %q %v", b, err)
	}
}

// F-436: Endet das erste Fenster, holt das zweite die Sperre beim nächsten Sichern nach und sichert dann selbst. Eine
// Kopie, die das erste Fenster hinterlassen hat, legt es vorher beiseite, statt sie zu überschreiben.
func TestFremdesFensterUebernimmtSperre(t *testing.T) {
	pfad, k := ausgang(t, "datei")
	ort := t.TempDir()
	d, _ := document.Neu(bild(64, 64))
	bearbeite(t, d)
	a := NeueSicherung(ort, k)
	a.Sichere(d)
	warteKette(t, a)
	kopieA, err := os.ReadFile(filepath.Join(ort, Name(pfad)))
	if err != nil {
		t.Fatal(err)
	}

	b := NeueSicherung(ort, k)
	neu, _ := document.Neu(bild(64, 64))
	if w := b.Wiederherstellen(neu, nil); !w.Fremd {
		t.Fatalf("zweites Fenster nicht fremd: %+v", w)
	}
	mitText(t, neu)
	b.Sichere(neu) // A hält die Sperre noch: nichts geschrieben
	warteKette(t, b)
	if nachher, _ := os.ReadFile(filepath.Join(ort, Name(pfad))); !bytes.Equal(nachher, kopieA) {
		t.Fatal("solange A die Sperre hält, bleibt seine Kopie unberührt")
	}

	a.Ende() // A endet mit ungesicherten Änderungen: seine Kopie bleibt liegen, die Sperre ist frei
	mitText(t, neu)
	b.Sichere(neu)
	b.Ende()
	kp, err := Lies(ort, pfad)
	if err != nil {
		t.Fatal(err)
	}
	leer, _ := document.Neu(bild(64, 64))
	if err := Anwenden(leer, kp, nil); err != nil || len(leer.Objekte()) != 2 {
		t.Errorf("Kopie von B nicht geschrieben: %d Objekte, %v", len(leer.Objekte()), err)
	}
	beiseite := 0
	reste, _ := os.ReadDir(ort)
	for _, e := range reste {
		if strings.HasSuffix(e.Name(), ".alt.json") {
			beiseite++
			if b, err := os.ReadFile(filepath.Join(ort, e.Name())); err != nil || !bytes.Equal(b, kopieA) {
				t.Errorf("beiseitegelegte Kopie ist nicht die von A: %v", err)
			}
		}
	}
	if beiseite != 1 {
		t.Errorf("%d beiseitegelegte Kopien, erwartet 1 (die von A)", beiseite)
	}
}

// F-446: Ein Sichere direkt nach NeueBasis fällt nicht weg, auch wenn die Sitzung vorher stumm (aus) war: Die Kette
// setzt aus zurück und entscheidet, nicht der Aufruf davor.
func TestSichereDirektNachNeueBasis(t *testing.T) {
	_, k := ausgang(t, "datei")
	ort := t.TempDir()
	s := NeueSicherung(ort, k)
	s.mu.Lock()
	s.aus = true // eine Kopie ließ sich nicht beiseitelegen
	s.mu.Unlock()
	pfad2, k2 := ausgang(t, "neu")
	d, _ := document.Neu(bild(64, 64))
	bearbeite(t, d)
	s.NeueBasis(k2)
	s.Sichere(d) // ohne auf die Kette zu warten
	s.Ende()
	if _, err := Lies(ort, pfad2); err != nil {
		t.Errorf("Kopie zur neuen Basis fehlt: %v", err)
	}
}

// F-446: War die Sitzung fremd und ist die neue Ausgangsdatei frei, sichert ein Sichere direkt nach NeueBasis.
func TestSichereDirektNachNeueBasisAusFremd(t *testing.T) {
	_, k := ausgang(t, "datei")
	ort := t.TempDir()
	a := NeueSicherung(ort, k)
	defer a.Ende()
	b := NeueSicherung(ort, k) // fremd
	pfad2, k2 := ausgang(t, "neu")
	d, _ := document.Neu(bild(64, 64))
	bearbeite(t, d)
	b.NeueBasis(k2)
	b.Sichere(d)
	b.Ende()
	if _, err := Lies(ort, pfad2); err != nil {
		t.Errorf("Kopie zur neuen Basis fehlt: %v", err)
	}
}

// F-446: Wird die Sitzung bei NeueBasis fremd (ein anderes Fenster hält die neue Ausgangsdatei), meldet das die
// Diagnose, und die Kopie des anderen Fensters bleibt unberührt.
func TestNeueBasisFremdMeldet(t *testing.T) {
	_, k := ausgang(t, "datei")
	pfad2, k2 := ausgang(t, "neu")
	ort := t.TempDir()
	d, _ := document.Neu(bild(64, 64))
	bearbeite(t, d)
	a := NeueSicherung(ort, k2)
	a.Sichere(d)
	warteKette(t, a)
	vorher, _ := os.ReadFile(filepath.Join(ort, Name(pfad2)))
	defer a.Ende()

	b := NeueSicherung(ort, k)
	var g diagnosen
	g.fuer(b)
	b.NeueBasis(k2)
	mitText(t, d)
	b.Sichere(d)
	b.Ende()
	if !g.hat(ErrBelegt) {
		t.Errorf("keine Meldung, dass die neue Ausgangsdatei in einem anderen Fenster liegt: %v", g.list)
	}
	if nachher, _ := os.ReadFile(filepath.Join(ort, Name(pfad2))); !bytes.Equal(nachher, vorher) {
		t.Error("Kopie des anderen Fensters berührt")
	}
}

// F-446: Bei hängender Platte hält die Sitzung höchstens eine wartende Kopie und reiht höchstens einen weiteren
// Auftrag ein; geschrieben wird danach die jüngste.
func TestHaengendePlatteEineWartendeKopie(t *testing.T) {
	pfad, k := ausgang(t, "datei")
	ort := t.TempDir()
	s := NeueSicherung(ort, k)
	var (
		mu       sync.Mutex
		geschr   []Kopie
		erster   = make(chan struct{})
		freigabe = make(chan struct{})
	)
	s.schreibe = func(ort string, kp Kopie) error {
		mu.Lock()
		geschr = append(geschr, kp)
		n := len(geschr)
		mu.Unlock()
		if n == 1 {
			close(erster)
			<-freigabe
		}
		return Schreibe(ort, kp)
	}
	d, _ := document.Neu(bild(64, 64))
	mitText(t, d)
	s.Sichere(d)
	<-erster // der erste Auftrag hängt in der Platte
	vorher := runtime.NumGoroutine()
	for range 50 {
		mitText(t, d)
		s.Sichere(d)
	}
	if mehr := runtime.NumGoroutine() - vorher; mehr > 2 {
		t.Errorf("%d zusätzliche Goroutinen bei hängender Platte, erwartet höchstens einen wartenden Auftrag", mehr)
	}
	close(freigabe)
	s.Ende()
	mu.Lock()
	n := len(geschr)
	mu.Unlock()
	if n != 2 {
		t.Errorf("%d Schreibaufrufe, erwartet 2 (der hängende und die jüngste)", n)
	}
	kp, err := Lies(ort, pfad)
	if err != nil {
		t.Fatal(err)
	}
	leer, _ := document.Neu(bild(64, 64))
	if err := Anwenden(leer, kp, nil); err != nil || len(leer.Objekte()) != 51 {
		t.Errorf("nicht die jüngste Kopie: %d Objekte, %v", len(leer.Objekte()), err)
	}
}

// F-446: Eine vor Verwirf eingereihte Kopie schreibt die Kopie nach Verwirf nicht schon vor dem Löschen: Verwirf löscht
// in der Kette, die danach gewollte Kopie folgt erst dahinter.
func TestKopieNachVerwirfBleibtHinterDemLoeschen(t *testing.T) {
	pfad, k := ausgang(t, "datei")
	ort := t.TempDir()
	s := NeueSicherung(ort, k)
	frei := make(chan struct{})
	erstes := true
	var mu sync.Mutex
	alt := s.loesche
	s.loesche = func(ort, p string) error {
		mu.Lock()
		blockiere := erstes
		erstes = false
		mu.Unlock()
		if blockiere {
			<-frei
		}
		return alt(ort, p)
	}
	d, _ := document.Neu(bild(64, 64))
	bearbeite(t, d)
	s.Verwirf() // hängt im Löschen
	s.Sichere(d)
	s.Verwirf()
	mitText(t, d)
	s.Sichere(d)
	close(frei)
	s.Ende()
	kp, err := Lies(ort, pfad)
	if err != nil {
		t.Fatalf("die nach Verwirf gewollte Kopie fehlt: %v", err)
	}
	leer, _ := document.Neu(bild(64, 64))
	if err := Anwenden(leer, kp, nil); err != nil || len(leer.Objekte()) != 3 {
		t.Errorf("%d Objekte, %v", len(leer.Objekte()), err)
	}
}

// F-451: Fehler aus Erzeuge tragen das Präfix "rescue:", auch der aus document.PruefeSchwaerzung.
func TestErzeugeFehlerMitPraefix(t *testing.T) {
	alt := budgetPruefer
	t.Cleanup(func() { budgetPruefer = alt })
	ursache := errors.New("document: Mosaik 99 außerhalb 0..32")
	budgetPruefer = func(*document.Dokument, []document.SchwaerzAngabe, int64) error { return ursache }
	d, _ := document.Neu(bild(64, 64))
	_, err := Erzeuge(d, Kennung{Pfad: "/x"}, time.Now())
	if err == nil || !strings.HasPrefix(err.Error(), "rescue: ") || !errors.Is(err, ursache) {
		t.Errorf("Fehler aus Erzeuge: %v", err)
	}
}
