// SPDX-License-Identifier: GPL-3.0-or-later

package export

import (
	"errors"
	"os"
	"syscall"
	"testing"
	"time"
)

// Tests zu F-338: Sichern, Schreiben und Wiederherstellen laufen über einen einzigen Deskriptor; eine Lease lässt das
// Öffnen nicht dauerhaft scheitern.

// TestRueckfallSichertWasErOeffnet: Ändert sich die Datei zwischen Lstat und Öffnen (gleiche Länge, gleiche Datei),
// stellt ein Schreibfehler genau den Inhalt wieder her, der beim Öffnen dastand — nicht einen älteren Stand.
func TestRueckfallSichertWasErOeffnet(t *testing.T) {
	ziel := zielMit(t, []byte("alt"))
	vorOeffnen(t, func() {
		f, err := os.OpenFile(ziel, os.O_WRONLY, 0)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.WriteAt([]byte("ALT"), 0); err != nil {
			t.Fatal(err)
		}
		f.Close()
	})
	kaputt := errors.New("Platte voll")
	orig := oeffneDirekt // von vorOeffnen schon umgebogen; hier zusätzlich der Schreibfehler
	t.Cleanup(func() { oeffneDirekt = orig })
	oeffneDirekt = func(pfad string, st os.FileInfo) (zielDatei, error) {
		f, err := orig(pfad, st)
		if err != nil {
			return nil, err
		}
		return haelfteDatei{f, kaputt, scheitert(1)}, nil
	}
	if err := schreibeDirekt(ziel, schreibeNeu, os.ErrPermission); !errors.Is(err, kaputt) {
		t.Fatalf("Fehler %v, erwartet der Schreibfehler", err)
	}
	if got, _ := os.ReadFile(ziel); string(got) != "ALT" {
		t.Errorf("wiederhergestellt %q, erwartet der Stand beim Öffnen %q", got, "ALT")
	}
}

// fcntlLease setzt oder löst eine Lease auf fd (F_SETLEASE; Kernel-Konstanten, kein Fremdpaket).
func fcntlLease(fd uintptr, art int) error {
	const fSetlease = 1024
	if _, _, e := syscall.Syscall(syscall.SYS_FCNTL, fd, fSetlease, uintptr(art)); e != 0 {
		return e
	}
	return nil
}

// TestRueckfallWartetKurzAufLease: Hält ein anderer Deskriptor eine Lese-Lease, scheitert das Öffnen mit O_NONBLOCK
// zunächst mit EAGAIN; wird die Lease kurz darauf freigegeben, gelingt der Rückfall doch.
func TestRueckfallWartetKurzAufLease(t *testing.T) {
	const fRdlck, fUnlck = 0, 2
	ziel := zielMit(t, []byte("alt"))
	halter, err := os.Open(ziel)
	if err != nil {
		t.Fatal(err)
	}
	defer halter.Close()
	if err := fcntlLease(halter.Fd(), fRdlck); err != nil {
		t.Fatalf("Lease nicht gesetzt: %v", err)
	}
	altPause := leasePause
	t.Cleanup(func() { leasePause = altPause })
	leasePause = 20 * time.Millisecond
	geloest := make(chan error, 1)
	time.AfterFunc(100*time.Millisecond, func() { geloest <- fcntlLease(halter.Fd(), fUnlck) })
	if err := schreibeDirekt(ziel, schreibeNeu, os.ErrPermission); err != nil {
		t.Fatalf("Fehler %v, erwartet Erfolg nach Freigabe der Lease", err)
	}
	if err := <-geloest; err != nil {
		t.Fatalf("Lease nicht gelöst: %v", err)
	}
	if got, _ := os.ReadFile(ziel); string(got) != "neuer Inhalt, kürzer" {
		t.Errorf("Ziel %q", got)
	}
}
