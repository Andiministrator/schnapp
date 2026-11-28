// SPDX-License-Identifier: GPL-3.0-or-later

package wlclip

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// Der Halter läuft als eigener, abgelöster Prozess (dasselbe Programm mit HelferArgument): Der Editor kann sich
// schließen und Aufrufer, die auf sein Ende warten (Druck-Taste, Skripte, `$(schnapp)`), warten nicht auf den Halter
// (ADR-0024). Der Editor schreibt das Bild auf die Standardeingabe des Halters; der Halter meldet über den
// Dateideskriptor 3 „ok“ oder einen Fehlertext und schließt ihn.

// HelferArgument ist das interne, nicht dokumentierte Argument, mit dem Schnapp sich als Halter startet. Es steht nur
// allein; main prüft es vor allem anderen (kein Gio, keine Einstellungen).
const HelferArgument = "--internal-keep-clipboard"

// HelferArgumentText startet den Halter im Textmodus (ArtText, ADR-0024 Nachtrag 9); sonst wie HelferArgument.
const HelferArgumentText = "--internal-keep-clipboard-text"

// HelferArgumentObjekte startet den Halter für kopierte Objekte (ArtObjekte, ADR-0045 Punkt 2); sonst wie HelferArgument.
const HelferArgumentObjekte = "--internal-keep-clipboard-objects"

// Argument ist das Helfer-Argument der Art.
func (a Art) Argument() string {
	switch a {
	case ArtText:
		return HelferArgumentText
	case ArtObjekte:
		return HelferArgumentObjekte
	}
	return HelferArgument
}

// ArtVon liefert die Art zu einem Helfer-Argument; ok ist false bei jedem anderen Text.
func ArtVon(argument string) (a Art, ok bool) {
	switch argument {
	case HelferArgument:
		return ArtBild, true
	case HelferArgumentText:
		return ArtText, true
	case HelferArgumentObjekte:
		return ArtObjekte, true
	}
	return ArtBild, false
}

// Haltezeit ist die harte Zeitgrenze des Halters; danach gibt er die Zwischenablage frei und endet (ADR-0024,
// Wächter-Regel „harte Zeitgrenze“).
const Haltezeit = 12 * time.Hour

// Frist ist, wie lange der Editor höchstens auf die Meldung des Halters wartet.
const Frist = 3 * time.Second

// MIME ist der angebotene Typ eines Bilds (ArtBild).
const MIME = "image/png"

// maxDaten begrenzt, was der Halter von der Standardeingabe annimmt.
const maxDaten = 1 << 30

const meldungOK = "ok"

// Helfer ist der Lauf des Halter-Prozesses: Bild oder Text (Art a) von ein lesen, als Zwischenablage setzen, auf meldung „ok“ oder den
// Fehler schreiben und meldung schließen, dann bis zum Ende ausliefern (Halter.Diene). Rückgabe ist der Exit-Code.
func Helfer(ein io.Reader, meldung io.WriteCloser, grenze time.Duration, a Art) int {
	melde := func(text string) {
		if meldung != nil {
			io.WriteString(meldung, text+"\n")
			meldung.Close()
			meldung = nil
		}
	}
	defer melde("fehler: ohne Meldung beendet")
	daten, err := io.ReadAll(io.LimitReader(ein, maxDaten+1))
	if err != nil {
		melde("fehler: " + err.Error())
		return 1
	}
	if len(daten) == 0 || len(daten) > maxDaten {
		melde(fmt.Sprintf("fehler: %d Byte auf der Standardeingabe", len(daten)))
		return 1
	}
	h, err := Biete(daten, a)
	if err != nil {
		melde("fehler: " + err.Error())
		return 1
	}
	melde(meldungOK + " " + h.Manager)
	switch err := h.Diene(grenze); {
	case err == nil, errors.Is(err, ErrZeitgrenze), errors.Is(err, ErrVerbindungZu):
		return 0
	default:
		return 1
	}
}

// HelferLauf ist Helfer für main: Art aus dem Helfer-Argument, Standardeingabe, Dateideskriptor 3, Haltezeit. Fd 3 gilt nur als Meldeweg, wenn er
// beim Start eine Pipe ist; sonst (Aufruf von Hand ohne fd 3) bekäme der erste Socket die Nummer 3, und die Meldung
// landete in der Wayland-Verbindung (Kritik Fable, Befund 1).
func HelferLauf(argument string) int {
	a, _ := ArtVon(argument) // main ruft nur mit einem gültigen Argument (ArtVon)
	zu := schliesseGeerbte()
	diagnose("Halter: %d geerbte Deskriptoren ohne CLOEXEC geschlossen (F-189): %v", len(zu), zu)
	var meldung io.WriteCloser
	if istPipe(3) {
		meldung = os.NewFile(3, "wlclip-meldung")
	}
	return Helfer(os.Stdin, meldung, Haltezeit, a)
}

// schliesseGeerbte schließt alle Dateideskriptoren ab 4, die ohne FD_CLOEXEC geerbt wurden (F-189): Was der Editor
// ohne CLOEXEC offen hatte (`schnapp 5>datei`, cgo-Bibliotheken), bliebe sonst bis zu 12 Stunden im Halter offen. Die
// Go-Laufzeit öffnet ihre eigenen Deskriptoren (epoll, Eventfd, Pipes) immer mit CLOEXEC; geerbte ohne Flag sind also
// nie die der Laufzeit. Deskriptoren 0 bis 3 (Eingabe, Ausgaben, Meldeweg) bleiben. Gibt die Nummern der geschlossenen
// zurück.
func schliesseGeerbte() []int {
	return schliesseOhneCloexec(offeneDeskriptoren())
}

// schliesseOhneCloexec schließt aus fds alle Deskriptoren ab 4 ohne FD_CLOEXEC und gibt deren Nummern zurück.
// Getrennt von schliesseGeerbte, damit die Zählung ohne Vorgeschichte im Prozess testbar ist.
func schliesseOhneCloexec(fds []int) []int { return schliesseOhneCloexecAb(fds, 4) }

// SchliesseGeerbteAb schließt alle ohne FD_CLOEXEC geerbten Deskriptoren ab ab (mindestens 3) und gibt ihre Nummern
// zurück: für einen Prozess, den der Editor abgelöst startet und der lange lebt (Neue Aufnahme, ADR-0046 Punkt 7);
// Gründe wie beim Halter (F-189).
func SchliesseGeerbteAb(ab int) []int {
	return schliesseOhneCloexecAb(offeneDeskriptoren(), max(ab, 3))
}

// IstPipe meldet, ob der Deskriptor fd eine Pipe ist: Ein lange lebender Kindprozess bekommt die Fehlerausgabe nie als
// Pipe, sonst endete ein `schnapp 2>&1 | tee` erst mit ihm (F-325).
func IstPipe(fd int) bool { return istPipe(fd) }

// schliesseOhneCloexecAb ist schliesseOhneCloexec ab dem Deskriptor ab.
func schliesseOhneCloexecAb(fds []int, ab int) []int {
	var zu []int
	for _, fd := range fds {
		if fd < ab {
			continue
		}
		flags, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_GETFD, 0)
		if errno != 0 || flags&syscall.FD_CLOEXEC != 0 {
			continue
		}
		if syscall.Close(fd) == nil {
			zu = append(zu, fd)
		}
	}
	return zu
}

// diagnose schreibt nur mit gesetztem SCHNAPP_DEBUG auf stderr (Fehlersuche bei echten Läufen, wie internal/ui).
func diagnose(format string, args ...any) {
	if os.Getenv("SCHNAPP_DEBUG") != "" {
		log.Printf("[wlclip] "+format, args...)
	}
}

// offeneDeskriptoren probiert alle Nummern bis zur weichen Grenze (höchstens 65536) per fcntl durch. Bewusst ohne
// das proc-Dateisystem: Schnapp liest keine Prozessverzeichnisse, auch nicht das eigene (Projektregel, F-189). Ein
// Durchlauf kostet einmal beim Start des Halters einige Millisekunden.
func offeneDeskriptoren() []int {
	var nr []int
	grenze := 65536
	var rl syscall.Rlimit
	if syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rl) == nil && rl.Cur < uint64(grenze) {
		grenze = int(rl.Cur)
	}
	for fd := 0; fd < grenze; fd++ {
		if _, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_GETFD, 0); errno == 0 {
			nr = append(nr, fd)
		}
	}
	return nr
}

// istPipe meldet, ob fd offen und eine Pipe ist.
func istPipe(fd int) bool {
	var st syscall.Stat_t
	return syscall.Fstat(fd, &st) == nil && st.Mode&syscall.S_IFMT == syscall.S_IFIFO
}

// ErrWaylandSocket: Die Sitzung hat ihre Wayland-Verbindung über WAYLAND_SOCKET geerbt; der Halter als eigener Prozess
// hätte keine zweite.
var ErrWaylandSocket = errors.New("wlclip: WAYLAND_SOCKET gesetzt")

// HalterMoeglich meldet, ob ein Halter-Prozess sich verbinden könnte: Bei WAYLAND_SOCKET (geerbter Socket) nicht.
func HalterMoeglich(getenv func(string) string) error {
	if getenv("WAYLAND_SOCKET") != "" {
		return ErrWaylandSocket
	}
	return nil
}

// Uebergib startet programm mit dem Helfer-Argument der Art a als abgelösten Prozess (eigene Sitzung, Arbeitsordner /, ohne
// Terminal-Ausgaben), schreibt daten auf seine Standardeingabe und wartet bis zu frist auf seine Meldung. Bei Erfolg
// hält der Halter die Zwischenablage, und Uebergib liefert seine PID und die benutzte Manager-Schnittstelle. Sonst beendet Uebergib den eigenen Kindprozess
// (Ziel steht beim Start fest, keine Suche; die Goroutine unten holt ihn direkt nach dem Start ab, Kill trifft also nur
// einen noch laufenden) und liefert den Fehler; der Aufrufer bietet dann
// selbst an.
func Uebergib(programm string, a Art, daten []byte, frist time.Duration) (pid int, manager string, err error) {
	r, w, err := os.Pipe()
	if err != nil {
		return 0, "", err
	}
	defer r.Close()
	cmd := exec.Command(programm, a.Argument())
	cmd.Stdin = bytes.NewReader(daten)
	cmd.ExtraFiles = []*os.File{w} // Dateideskriptor 3 im Kind
	cmd.Dir = "/"
	// Die Diagnose des Halters soll im Terminal oder in der Logdatei erscheinen, sonst /dev/null. Nie in eine Pipe: Der
	// Halter lebt bis zu 12 Stunden und hielte sie offen, ein `schnapp 2>&1 | tee` endete erst mit ihm.
	if os.Getenv("SCHNAPP_DEBUG") != "" && !istPipe(2) {
		cmd.Stderr = os.Stderr
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	err = cmd.Start()
	w.Close()
	if err != nil {
		return 0, "", err
	}
	fertig := make(chan struct{})
	go func() { // holt das Kind ab, sobald es endet (kein Zombie, solange der Editor läuft)
		_ = cmd.Wait()
		close(fertig)
	}()
	text, err := liesMeldung(r, frist)
	wort, rest, _ := strings.Cut(text, " ")
	if err == nil && wort != meldungOK {
		err = errors.New(strings.TrimPrefix(text, "fehler: "))
		if text == "" {
			err = errors.New("wlclip: Halter ohne Meldung beendet")
		}
		if strings.Contains(text, ErrKeinDataControl.Error()) {
			err = ErrKeinDataControl // der Text des Halters, auf den Sentinel abgebildet
		}
	}
	if err != nil {
		_ = cmd.Process.Kill() // eigenes Kind; die Goroutine oben holt es ab, sobald es endet, dann schlägt Kill fehl (Fehler ignoriert)
		<-fertig
		return 0, "", err
	}
	return cmd.Process.Pid, rest, nil
}

// liesMeldung liest die Meldungszeile bis EOF, höchstens frist lang.
func liesMeldung(r *os.File, frist time.Duration) (string, error) {
	if err := r.SetReadDeadline(time.Now().Add(frist)); err != nil {
		return "", err
	}
	b, err := io.ReadAll(io.LimitReader(r, 4096))
	if err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			return "", fmt.Errorf("wlclip: keine Meldung des Halters nach %v", frist)
		}
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}
