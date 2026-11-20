// SPDX-License-Identifier: GPL-3.0-or-later

package wlclip

import (
	"bytes"
	"errors"
	"fmt"
	"io"
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

// Haltezeit ist die harte Zeitgrenze des Halters; danach gibt er die Zwischenablage frei und endet (ADR-0024,
// Wächter-Regel „harte Zeitgrenze“).
const Haltezeit = 12 * time.Hour

// Frist ist, wie lange der Editor höchstens auf die Meldung des Halters wartet.
const Frist = 3 * time.Second

// MIME ist der angebotene Typ.
const MIME = "image/png"

// maxDaten begrenzt, was der Halter von der Standardeingabe annimmt.
const maxDaten = 1 << 30

const meldungOK = "ok"

// Helfer ist der Lauf des Halter-Prozesses: Bild von ein lesen, als Zwischenablage setzen, auf meldung „ok“ oder den
// Fehler schreiben und meldung schließen, dann bis zum Ende ausliefern (Halter.Diene). Rückgabe ist der Exit-Code.
func Helfer(ein io.Reader, meldung io.WriteCloser, grenze time.Duration) int {
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
	h, err := Biete(daten, MIME)
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

// HelferLauf ist Helfer für main: Standardeingabe, Dateideskriptor 3, Haltezeit. Fd 3 gilt nur als Meldeweg, wenn er
// beim Start eine Pipe ist; sonst (Aufruf von Hand ohne fd 3) bekäme der erste Socket die Nummer 3, und die Meldung
// landete in der Wayland-Verbindung (Kritik Fable, Befund 1).
func HelferLauf() int {
	var meldung io.WriteCloser
	if istPipe(3) {
		meldung = os.NewFile(3, "wlclip-meldung")
	}
	return Helfer(os.Stdin, meldung, Haltezeit)
}

// istPipe meldet, ob fd offen und eine Pipe ist.
func istPipe(fd int) bool {
	var st syscall.Stat_t
	return syscall.Fstat(fd, &st) == nil && st.Mode&syscall.S_IFMT == syscall.S_IFIFO
}

// Uebergib startet programm mit HelferArgument als abgelösten Prozess (eigene Sitzung, Arbeitsordner /, ohne
// Terminal-Ausgaben), schreibt daten auf seine Standardeingabe und wartet bis zu frist auf seine Meldung. Bei Erfolg
// hält der Halter die Zwischenablage, und Uebergib liefert seine PID und die benutzte Manager-Schnittstelle. Sonst beendet Uebergib den eigenen, noch nicht
// abgeholten Kindprozess (Ziel steht beim Start fest, keine Suche) und liefert den Fehler; der Aufrufer bietet dann
// selbst an.
func Uebergib(programm string, daten []byte, frist time.Duration) (pid int, manager string, err error) {
	r, w, err := os.Pipe()
	if err != nil {
		return 0, "", err
	}
	defer r.Close()
	cmd := exec.Command(programm, HelferArgument)
	cmd.Stdin = bytes.NewReader(daten)
	cmd.ExtraFiles = []*os.File{w} // Dateideskriptor 3 im Kind
	cmd.Dir = "/"
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
	}
	if err != nil {
		_ = cmd.Process.Kill() // eigenes Kind, noch nicht abgeholt: die PID ist nicht wiederverwendet
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
