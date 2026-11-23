// SPDX-License-Identifier: GPL-3.0-or-later

package main

// Sitzungsbindung des Bestätigungsdialogs (ADR-0030 Nachtrag 5, F-491).

import (
	"context"
	"errors"
	"fmt"
	"net"
	"path/filepath"

	"github.com/godbus/dbus/v5"
	"golang.org/x/sys/unix"
)

// errFremdeSitzung: Der Compositor hinter WAYLAND_DISPLAY ist nicht das KWin, das auf dem Sitzungsbus aufnimmt.
var errFremdeSitzung = errors.New("schnapp: der Dialog erschiene auf einem anderen Compositor als dem, der aufnimmt")

// Nähte für Tests: Prozessnummer des Compositors hinter einem Wayland-Socket und des Besitzers von org.kde.KWin auf dem
// Sitzungsbus.
var (
	waylandPid = waylandCompositorPid
	kwinPid    = kwinBusPid
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

// pruefeSitzung bindet den Dialog an die Aufnahme (F-491): Der Prozess, der den Wayland-Socket des Dialogs bedient
// (SO_PEERCRED), muss derselbe sein wie der Besitzer von org.kde.KWin auf dem Sitzungsbus, über den die Aufnahme läuft.
// Zeigt die Umgebung des Agenten-Clients den Dialog auf einen eigenen Compositor, scheitert die Prüfung; zeigt sie
// beides (Wayland und Bus) auf einen fremden, nimmt auch die Aufnahme von dort auf — der echte Bildschirm bleibt
// unberührt. Das Kind erbt WAYLAND_DISPLAY und XDG_RUNTIME_DIR unverändert und löst so denselben Socket auf;
// WAYLAND_SOCKET (ein geerbter Deskriptor, nicht prüfbar) lehnt waylandSocketPfad ab.
func pruefeSitzung(ctx context.Context, umgebung func(string) string) error {
	socket, err := waylandSocketPfad(umgebung)
	if err != nil {
		return err
	}
	w, err := waylandPid(socket)
	if err != nil {
		return fmt.Errorf("schnapp: Wayland-Socket: %w", err)
	}
	k, err := kwinPid(ctx)
	if err != nil {
		return fmt.Errorf("schnapp: KWin auf dem Sitzungsbus: %w", err)
	}
	if w <= 0 || w != k {
		return errFremdeSitzung
	}
	return nil
}

// waylandCompositorPid verbindet sich mit dem Socket und fragt den Kernel nach dem Prozess auf der anderen Seite
// (SO_PEERCRED). Es wird kein Byte gesendet.
func waylandCompositorPid(socket string) (int32, error) {
	c, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: socket, Net: "unix"})
	if err != nil {
		return 0, err
	}
	defer c.Close()
	roh, err := c.SyscallConn()
	if err != nil {
		return 0, err
	}
	var cred *unix.Ucred
	var cerr error
	if err := roh.Control(func(fd uintptr) {
		cred, cerr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	}); err != nil {
		return 0, err
	}
	if cerr != nil {
		return 0, cerr
	}
	return cred.Pid, nil
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
