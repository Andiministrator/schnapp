// SPDX-License-Identifier: GPL-3.0-or-later

package main

// Sitzungsbindung des Bestätigungsdialogs (ADR-0030 Nachtrag 5, F-491).

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/godbus/dbus/v5"
	"golang.org/x/sys/unix"
)

// errFremdeSitzung: Der Compositor hinter WAYLAND_DISPLAY ist nicht das KWin, das auf dem Sitzungsbus aufnimmt.
var errFremdeSitzung = errors.New("schnapp: der Dialog erschiene auf einem anderen Compositor als dem, der aufnimmt")

// Nähte für Tests: geprüfte Verbindung zum Wayland-Socket samt Prozess dahinter, Besitzer von org.kde.KWin auf dem
// Sitzungsbus und Elternprozess eines Prozesses.
var (
	waylandVerbinde = waylandVerbindung
	kwinPid         = kwinBusPid
	elternVon       = elternProzess
)

// waylandSocketPfad nennt den Socket, mit dem sich ein Wayland-Client verbände: WAYLAND_DISPLAY (Voreinstellung
// wayland-0), relativ zu XDG_RUNTIME_DIR. WAYLAND_SOCKET (ein geerbter Deskriptor) lässt sich nicht prüfen und wird
// abgelehnt.
func waylandSocketPfad(umgebung func(string) string) (string, error) {
	if umgebung("WAYLAND_SOCKET") != "" {
		return "", errors.New("schnapp: WAYLAND_SOCKET ist gesetzt")
	}
	name := umgebung("WAYLAND_DISPLAY")
	if name == "" {
		name = "wayland-0"
	}
	if filepath.IsAbs(name) {
		return filepath.Clean(name), nil
	}
	lauf := umgebung("XDG_RUNTIME_DIR")
	if !filepath.IsAbs(lauf) {
		return "", errors.New("schnapp: XDG_RUNTIME_DIR fehlt")
	}
	return filepath.Join(lauf, name), nil
}

// pruefeSitzung bindet den Dialog an die Aufnahme (F-491, ADR-0030 Nachträge 5 und 6): Der Server verbindet sich
// selbst mit dem Wayland-Socket und fragt den Kernel nach dem Prozess dahinter (SO_PEERCRED). Er muss der Besitzer von
// org.kde.KWin auf dem Sitzungsbus sein, über den die Aufnahme läuft, oder dessen Elternprozess: In Plasma legt
// kwin_wayland_wrapper den Socket an und startet kwin_wayland als Kind (Sitzungs-QA A-1). Die geprüfte Verbindung geht
// als Deskriptor an das Kind (WAYLAND_SOCKET); es löst den Pfad nicht noch einmal auf, ein umgehängter Symlink oder ein
// Pfad mit „..“ führt es also nicht woandershin (A-2). Zeigt die Umgebung beides (Wayland und Bus) auf einen fremden
// Compositor, nimmt auch die Aufnahme von dort auf — der echte Bildschirm bleibt unberührt.
func pruefeSitzung(ctx context.Context, umgebung func(string) string) (*os.File, error) {
	socket, err := waylandSocketPfad(umgebung)
	if err != nil {
		return nil, err
	}
	f, w, err := waylandVerbinde(socket)
	if err != nil {
		return nil, fmt.Errorf("schnapp: Wayland-Socket: %w", err)
	}
	k, err := kwinPid(ctx)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("schnapp: KWin auf dem Sitzungsbus: %w", err)
	}
	if w > 0 && (w == k || elternVon(k) == w) {
		return f, nil
	}
	f.Close()
	return nil, errFremdeSitzung
}

// elternProzess liest den Elternprozess aus /proc/<pid>/stat (Feld 4, nach dem Programmnamen in Klammern); 0, wenn er
// sich nicht lesen lässt. Gelesen wird nur diese Statuszeile, nie Kommandozeile oder Umgebung.
func elternProzess(pid int32) int32 {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0
	}
	s := string(b)
	i := strings.LastIndexByte(s, ')')
	if i < 0 {
		return 0
	}
	felder := strings.Fields(s[i+1:])
	if len(felder) < 2 {
		return 0
	}
	p, err := strconv.Atoi(felder[1])
	if err != nil {
		return 0
	}
	return int32(p)
}

// waylandVerbindung verbindet sich mit dem Socket, fragt den Kernel nach dem Prozess auf der anderen Seite
// (SO_PEERCRED) und liefert die Verbindung als Datei zum Vererben. Es wird kein Byte gesendet.
func waylandVerbindung(socket string) (*os.File, int32, error) {
	c, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: socket, Net: "unix"})
	if err != nil {
		return nil, 0, err
	}
	defer c.Close()
	roh, err := c.SyscallConn()
	if err != nil {
		return nil, 0, err
	}
	var cred *unix.Ucred
	var cerr error
	if err := roh.Control(func(fd uintptr) {
		cred, cerr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	}); err != nil {
		return nil, 0, err
	}
	if cerr != nil {
		return nil, 0, cerr
	}
	f, err := c.File() // eigene Kopie des Deskriptors, bleibt nach c.Close offen
	if err != nil {
		return nil, 0, err
	}
	return f, cred.Pid, nil
}

// kwinBusPid fragt den Sitzungsbus nach dem Prozess, dem der Name org.kde.KWin gehört.
func kwinBusPid(ctx context.Context) (int32, error) {
	c, err := dbus.ConnectSessionBus() // eigene Verbindung, danach geschlossen
	if err != nil {
		return 0, err
	}
	defer c.Close()
	var pid uint32
	if err := c.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.GetConnectionUnixProcessID", 0,
		"org.kde.KWin").Store(&pid); err != nil {
		return 0, err
	}
	return int32(pid), nil
}
