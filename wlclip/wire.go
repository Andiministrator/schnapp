// SPDX-License-Identifier: GPL-3.0-or-later

package wlclip

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"syscall"
)

// Das Wayland-Leitungsformat (wire protocol): je Nachricht ein Kopf aus Objekt-ID und (Größe << 16 | Opcode), beide
// 32 Bit in der Bytereihenfolge des Rechners, danach die Argumente in 32-Bit-Wörtern; Zeichenketten mit Länge samt
// Nullbyte, auf 4 Byte aufgefüllt; Dateideskriptoren laufen außerhalb über SCM_RIGHTS. Dieses Paket spricht nur die
// wenigen Nachrichten, die es braucht, ohne libwayland (kein cgo).

// maxNachricht ist die größte Nachricht, die libwayland annimmt (Kopf eingeschlossen).
const maxNachricht = 4096

var ordnung = binary.NativeEndian

// ErrVerbindungZu meldet, dass der Compositor die Verbindung geschlossen hat.
var ErrVerbindungZu = errors.New("wlclip: Verbindung zum Compositor geschlossen")

// verbindung ist eine Client- oder (im Test) Server-Seite einer Wayland-Verbindung.
type verbindung struct {
	c        *net.UnixConn
	ein      []byte // gelesene, noch nicht verarbeitete Bytes
	fds      []int  // empfangene Dateideskriptoren in Ankunftsreihenfolge
	naechste uint32 // nächste freie Objekt-ID des Clients (1 ist wl_display)
}

// socketPfad bestimmt den Socket des Compositors wie libwayland: WAYLAND_DISPLAY (Voreinstellung wayland-0), relativ
// zu XDG_RUNTIME_DIR, wenn nicht absolut.
func socketPfad(getenv func(string) string) (string, error) {
	name := getenv("WAYLAND_DISPLAY")
	if name == "" {
		name = "wayland-0"
	}
	if filepath.IsAbs(name) {
		return name, nil
	}
	dir := getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		return "", errors.New("wlclip: XDG_RUNTIME_DIR nicht gesetzt")
	}
	return filepath.Join(dir, name), nil
}

// verbinde öffnet die Verbindung zum Compositor der Sitzung.
func verbinde() (*verbindung, error) {
	pfad, err := socketPfad(os.Getenv)
	if err != nil {
		return nil, err
	}
	c, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: pfad, Net: "unix"})
	if err != nil {
		return nil, fmt.Errorf("wlclip: Compositor nicht erreichbar: %w", err)
	}
	return neueVerbindung(c), nil
}

func neueVerbindung(c *net.UnixConn) *verbindung {
	return &verbindung{c: c, naechste: 2}
}

// neueID vergibt die nächste Objekt-ID des Clients.
func (v *verbindung) neueID() uint32 {
	id := v.naechste
	v.naechste++
	return id
}

// Close schließt die Verbindung und alle nicht abgeholten Dateideskriptoren.
func (v *verbindung) Close() error {
	for _, fd := range v.fds {
		syscall.Close(fd)
	}
	v.fds = nil
	return v.c.Close()
}

// nachricht baut die Argumente einer Nachricht.
type nachricht struct {
	b   []byte
	fds []int
}

func (n *nachricht) u32(w uint32) *nachricht {
	n.b = ordnung.AppendUint32(n.b, w)
	return n
}

func (n *nachricht) str(s string) *nachricht {
	n.u32(uint32(len(s) + 1))
	n.b = append(n.b, s...)
	n.b = append(n.b, 0)
	for len(n.b)%4 != 0 {
		n.b = append(n.b, 0)
	}
	return n
}

func (n *nachricht) fd(fd int) *nachricht {
	n.fds = append(n.fds, fd)
	return n
}

// sende schreibt eine Nachricht an obj mit dem Opcode op.
func (v *verbindung) sende(obj uint32, op uint16, n *nachricht) error {
	if n == nil {
		n = &nachricht{}
	}
	groesse := 8 + len(n.b)
	if groesse > maxNachricht {
		return fmt.Errorf("wlclip: Nachricht zu groß (%d Byte)", groesse)
	}
	buf := make([]byte, 0, groesse)
	buf = ordnung.AppendUint32(buf, obj)
	buf = ordnung.AppendUint32(buf, uint32(groesse)<<16|uint32(op))
	buf = append(buf, n.b...)
	var oob []byte
	if len(n.fds) > 0 {
		oob = syscall.UnixRights(n.fds...)
	}
	geschrieben, _, err := v.c.WriteMsgUnix(buf, oob, nil)
	if err == nil && geschrieben != len(buf) {
		err = fmt.Errorf("wlclip: Nachricht nur teilweise gesendet (%d von %d Byte)", geschrieben, len(buf))
	}
	return err
}

// ereignis ist eine empfangene Nachricht.
type ereignis struct {
	obj  uint32
	op   uint16
	args []byte
}

// lies liefert die nächste vollständige Nachricht. Eine Lesefrist der Verbindung (SetReadDeadline) gilt.
func (v *verbindung) lies() (ereignis, error) {
	for {
		if len(v.ein) >= 8 {
			kopf := ordnung.Uint32(v.ein[4:8])
			groesse := int(kopf >> 16)
			if groesse < 8 || groesse%4 != 0 {
				return ereignis{}, fmt.Errorf("wlclip: ungültige Nachrichtengröße %d", groesse)
			}
			if len(v.ein) >= groesse {
				e := ereignis{obj: ordnung.Uint32(v.ein[0:4]), op: uint16(kopf & 0xffff),
					args: append([]byte(nil), v.ein[8:groesse]...)}
				v.ein = v.ein[groesse:]
				return e, nil
			}
		}
		buf := make([]byte, maxNachricht)
		oob := make([]byte, syscall.CmsgSpace(28*4))
		n, oobn, _, _, err := v.c.ReadMsgUnix(buf, oob)
		n, oobn = max(n, 0), max(oobn, 0) // bei Fehler (etwa Lesefrist) liefert ReadMsgUnix -1
		if oobn > 0 {
			msgs, perr := syscall.ParseSocketControlMessage(oob[:oobn])
			if perr == nil {
				for _, m := range msgs {
					if fds, ferr := syscall.ParseUnixRights(&m); ferr == nil {
						v.fds = append(v.fds, fds...)
					}
				}
			}
		}
		v.ein = append(v.ein, buf[:n]...)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return ereignis{}, ErrVerbindungZu
			}
			return ereignis{}, err
		}
		if n == 0 && oobn == 0 {
			return ereignis{}, ErrVerbindungZu
		}
	}
}

// nimmFD holt den nächsten empfangenen Dateideskriptor (für ein Argument vom Typ fd).
func (v *verbindung) nimmFD() (int, error) {
	if len(v.fds) == 0 {
		return -1, errors.New("wlclip: Dateideskriptor erwartet, aber keiner empfangen")
	}
	fd := v.fds[0]
	v.fds = v.fds[1:]
	return fd, nil
}

// leser liest die Argumente eines Ereignisses.
type leser struct {
	b   []byte
	err error
}

func (l *leser) u32() uint32 {
	if l.err != nil {
		return 0
	}
	if len(l.b) < 4 {
		l.err = errors.New("wlclip: Argumente zu kurz")
		return 0
	}
	w := ordnung.Uint32(l.b)
	l.b = l.b[4:]
	return w
}

func (l *leser) str() string {
	n := int(l.u32())
	if l.err != nil {
		return ""
	}
	if n == 0 { // null-String
		return ""
	}
	gepolstert := (n + 3) &^ 3
	if n < 1 || len(l.b) < gepolstert {
		l.err = errors.New("wlclip: Zeichenkette zu kurz")
		return ""
	}
	s := string(l.b[:n-1])
	l.b = l.b[gepolstert:]
	return s
}
