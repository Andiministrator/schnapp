// SPDX-License-Identifier: GPL-3.0-or-later

// Package wlwire ist ein kleiner Wayland-Client in reinem Go (kein cgo, keine libwayland): Leitungsformat, Dateideskriptoren
// über SCM_RIGHTS, Leser für Argumente und der Rundlauf mit wl_display.sync. wlclip (Zwischenablage-Halter) und die
// Fernbedienung der Probeläufe benutzen ihn.
package wlwire

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// Das Wayland-Leitungsformat (wire protocol): je Nachricht ein Kopf aus Objekt-ID und (Größe << 16 | Opcode), beide
// 32 Bit in der Bytereihenfolge des Rechners, danach die Argumente in 32-Bit-Wörtern; Zeichenketten mit Länge samt
// Nullbyte, auf 4 Byte aufgefüllt; Dateideskriptoren laufen außerhalb über SCM_RIGHTS. Die Nachrichten selbst kennen die
// Aufrufer.

// maxNachricht ist die größte Nachricht, die libwayland annimmt (Kopf eingeschlossen).
const maxNachricht = 4096

var ordnung = binary.NativeEndian

// ErrVerbindungZu meldet, dass der Compositor die Verbindung geschlossen hat.
var ErrVerbindungZu = errors.New("wlwire: Verbindung zum Compositor geschlossen")

// Verbindung ist eine Client- oder (im Test) Server-Seite einer Wayland-Verbindung.
type Verbindung struct {
	c        *net.UnixConn
	ein      []byte // gelesene, noch nicht verarbeitete Bytes
	fds      []int  // empfangene Dateideskriptoren in Ankunftsreihenfolge
	naechste uint32 // nächste freie Objekt-ID des Clients (1 ist wl_display)
	// sendeMsg ersetzt WriteMsgUnix im Test (Teilschreiben, F-207); nil ist der Normalfall.
	sendeMsg func(b, oob []byte) (int, error)
}

// SocketPfad bestimmt den Socket des Compositors wie libwayland: WAYLAND_DISPLAY (Voreinstellung wayland-0), relativ
// zu XDG_RUNTIME_DIR, wenn nicht absolut.
func SocketPfad(getenv func(string) string) (string, error) {
	name := getenv("WAYLAND_DISPLAY")
	if name == "" {
		name = "wayland-0"
	}
	if filepath.IsAbs(name) {
		return name, nil
	}
	dir := getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		return "", errors.New("wlwire: XDG_RUNTIME_DIR nicht gesetzt")
	}
	return filepath.Join(dir, name), nil
}

// Verbinde öffnet die Verbindung zum Compositor der Sitzung.
func Verbinde() (*Verbindung, error) {
	pfad, err := SocketPfad(os.Getenv)
	if err != nil {
		return nil, err
	}
	c, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: pfad, Net: "unix"})
	if err != nil {
		return nil, fmt.Errorf("wlwire: Compositor nicht erreichbar: %w", err)
	}
	return Neue(c), nil
}

func Neue(c *net.UnixConn) *Verbindung {
	return &Verbindung{c: c, naechste: 2}
}

// NeueID vergibt die nächste Objekt-ID des Clients.
func (v *Verbindung) NeueID() uint32 {
	id := v.naechste
	v.naechste++
	return id
}

// Close schließt die Verbindung und alle nicht abgeholten Dateideskriptoren.
func (v *Verbindung) Close() error {
	for _, fd := range v.fds {
		syscall.Close(fd)
	}
	v.fds = nil
	return v.c.Close()
}

// Nachricht baut die Argumente einer Nachricht.
type Nachricht struct {
	b   []byte
	fds []int
}

func (n *Nachricht) U32(w uint32) *Nachricht {
	n.b = ordnung.AppendUint32(n.b, w)
	return n
}

func (n *Nachricht) Str(s string) *Nachricht {
	n.U32(uint32(len(s) + 1))
	n.b = append(n.b, s...)
	n.b = append(n.b, 0)
	for len(n.b)%4 != 0 {
		n.b = append(n.b, 0)
	}
	return n
}

func (n *Nachricht) FD(fd int) *Nachricht {
	n.fds = append(n.fds, fd)
	return n
}

// Sende schreibt eine Nachricht an obj mit dem Opcode op.
func (v *Verbindung) Sende(obj uint32, op uint16, n *Nachricht) error {
	if n == nil {
		n = &Nachricht{}
	}
	groesse := 8 + len(n.b)
	if groesse > maxNachricht {
		return fmt.Errorf("wlwire: Nachricht zu groß (%d Byte)", groesse)
	}
	buf := make([]byte, 0, groesse)
	buf = ordnung.AppendUint32(buf, obj)
	buf = ordnung.AppendUint32(buf, uint32(groesse)<<16|uint32(op))
	buf = append(buf, n.b...)
	var oob []byte
	if len(n.fds) > 0 {
		oob = syscall.UnixRights(n.fds...)
	}
	var geschrieben int
	var err error
	if v.sendeMsg != nil {
		geschrieben, err = v.sendeMsg(buf, oob)
	} else {
		geschrieben, _, err = v.c.WriteMsgUnix(buf, oob, nil)
	}
	if err != nil {
		return err
	}
	// Teilschreiben (Puffer voll, Frist): die Dateideskriptoren sind mit den ersten Bytes schon beim Compositor,
	// den Rest schickt ein gewöhnliches Write nach (F-207). Scheitert das, ist der Strom nicht mehr
	// Nachrichten-genau, der Fehler nennt die Lage.
	for geschrieben < len(buf) {
		n, err := v.c.Write(buf[geschrieben:])
		geschrieben += n
		if err != nil {
			return fmt.Errorf("wlwire: Nachricht nur teilweise gesendet (%d von %d Byte): %w", geschrieben, len(buf), err)
		}
	}
	return nil
}

// Ereignis ist eine empfangene Nachricht.
type Ereignis struct {
	Obj  uint32
	Op   uint16
	Args []byte
}

// Lies liefert die nächste vollständige Nachricht. Eine Lesefrist der Verbindung (SetReadDeadline) gilt.
func (v *Verbindung) Lies() (Ereignis, error) {
	for {
		if len(v.ein) >= 8 {
			kopf := ordnung.Uint32(v.ein[4:8])
			groesse := int(kopf >> 16)
			if groesse < 8 || groesse%4 != 0 {
				return Ereignis{}, fmt.Errorf("wlwire: ungültige Nachrichtengröße %d", groesse)
			}
			if len(v.ein) >= groesse {
				e := Ereignis{Obj: ordnung.Uint32(v.ein[0:4]), Op: uint16(kopf & 0xffff),
					Args: append([]byte(nil), v.ein[8:groesse]...)}
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
				return Ereignis{}, ErrVerbindungZu
			}
			return Ereignis{}, err
		}
		if n == 0 && oobn == 0 {
			return Ereignis{}, ErrVerbindungZu
		}
	}
}

// NimmFD holt den nächsten empfangenen Dateideskriptor (für ein Argument vom Typ fd).
func (v *Verbindung) NimmFD() (int, error) {
	if len(v.fds) == 0 {
		return -1, errors.New("wlwire: Dateideskriptor erwartet, aber keiner empfangen")
	}
	fd := v.fds[0]
	v.fds = v.fds[1:]
	return fd, nil
}

// Leser liest die Argumente eines Ereignisses.
type Leser struct {
	B   []byte
	Err error
}

func (l *Leser) U32() uint32 {
	if l.Err != nil {
		return 0
	}
	if len(l.B) < 4 {
		l.Err = errors.New("wlwire: Argumente zu kurz")
		return 0
	}
	w := ordnung.Uint32(l.B)
	l.B = l.B[4:]
	return w
}

func (l *Leser) Str() string {
	n := int(l.U32())
	if l.Err != nil {
		return ""
	}
	if n == 0 { // null-String
		return ""
	}
	gepolstert := (n + 3) &^ 3
	if n < 1 || len(l.B) < gepolstert {
		l.Err = errors.New("wlwire: Zeichenkette zu kurz")
		return ""
	}
	s := string(l.B[:n-1])
	l.B = l.B[gepolstert:]
	return s
}

// Fixed hängt ein Argument vom Typ fixed an (vorzeichenbehaftet, 24.8 Festkomma).
func (n *Nachricht) Fixed(v float64) *Nachricht {
	return n.U32(uint32(int32(v * 256)))
}

// SetReadDeadline setzt die Lesefrist der Verbindung.
func (v *Verbindung) SetReadDeadline(t time.Time) error { return v.c.SetReadDeadline(t) }

// SetWriteDeadline setzt die Schreibfrist der Verbindung.
func (v *Verbindung) SetWriteDeadline(t time.Time) error { return v.c.SetWriteDeadline(t) }

// SetDeadline setzt Lese- und Schreibfrist der Verbindung.
func (v *Verbindung) SetDeadline(t time.Time) error { return v.c.SetDeadline(t) }

// Objekt-IDs, Opcodes und Ereignisse von wl_display, wl_registry und wl_callback.
const (
	IDDisplay = 1

	OpDisplaySync        = 0 // wl_display.sync(new_id callback)
	OpDisplayGetRegistry = 1 // wl_display.get_registry(new_id registry)
	EvDisplayError       = 0 // wl_display.error(object, code, message)

	OpRegistryBind   = 0 // wl_registry.bind(name, interface, version, new_id)
	EvRegistryGlobal = 0 // wl_registry.global(name, interface, version)

	EvCallbackDone = 0
)

// Rundlauf schickt wl_display.sync und verarbeitet Ereignisse mit je, bis die Antwort da ist, höchstens frist lang.
// Ein wl_display.error beendet mit Fehler.
func (v *Verbindung) Rundlauf(frist time.Duration, je func(Ereignis) error) error {
	cb := v.NeueID()
	if err := v.Sende(IDDisplay, OpDisplaySync, new(Nachricht).U32(cb)); err != nil {
		return err
	}
	if err := v.c.SetReadDeadline(time.Now().Add(frist)); err != nil {
		return err
	}
	defer v.c.SetReadDeadline(time.Time{})
	for {
		e, err := v.Lies()
		if err != nil {
			return err
		}
		if e.Obj == cb && e.Op == EvCallbackDone {
			return nil
		}
		if err := PruefeDisplay(e); err != nil {
			return err
		}
		if err := je(e); err != nil {
			return err
		}
	}
}

// PruefeDisplay wandelt wl_display.error in einen Fehler.
func PruefeDisplay(e Ereignis) error {
	if e.Obj != IDDisplay || e.Op != EvDisplayError {
		return nil
	}
	l := Leser{B: e.Args}
	obj, code, text := l.U32(), l.U32(), l.Str()
	return fmt.Errorf("wlwire: Protokollfehler an Objekt %d (Code %d): %s", obj, code, text)
}

// Global ist ein Eintrag aus wl_registry.global.
type Global struct {
	Name, Version uint32
}

// Globale sendet wl_display.get_registry und sammelt im Rundlauf alle Globale (Schnittstelle -> erste Meldung; bei
// mehreren Sitzen o. Ä. gilt die erste). Liefert die Registry-ID für Bind.
func (v *Verbindung) Globale(frist time.Duration) (registry uint32, globale map[string]Global, err error) {
	registry = v.NeueID()
	if err = v.Sende(IDDisplay, OpDisplayGetRegistry, new(Nachricht).U32(registry)); err != nil {
		return 0, nil, err
	}
	globale = map[string]Global{}
	err = v.Rundlauf(frist, func(e Ereignis) error {
		if e.Obj == registry && e.Op == EvRegistryGlobal {
			l := Leser{B: e.Args}
			name, schnitt, version := l.U32(), l.Str(), l.U32()
			if l.Err != nil {
				return l.Err
			}
			if _, da := globale[schnitt]; !da {
				globale[schnitt] = Global{name, version}
			}
		}
		return nil
	})
	return registry, globale, err
}

// Bind bindet das Global g als Schnittstelle schnitt in der Version version und liefert die neue Objekt-ID.
func (v *Verbindung) Bind(registry uint32, g Global, schnitt string, version uint32) (uint32, error) {
	id := v.NeueID()
	err := v.Sende(registry, OpRegistryBind, new(Nachricht).U32(g.Name).Str(schnitt).U32(version).U32(id))
	return id, err
}
