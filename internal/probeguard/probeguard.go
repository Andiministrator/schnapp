// SPDX-License-Identifier: GPL-3.0-or-later

// Package probeguard sichert Probe-Builds (Tag probe) dagegen, in der echten Sitzung zu laufen (F-229). Probe-Programme
// (Schnapp mit Auto-Wahl des ganzen Monitors, Fernbedienung mit fake_input) bleiben nach einem Lauf in tmp/probe liegen;
// von Hand gestartet, würden sie den echten Monitor aufnehmen und in der echten Bibliothek ablegen bzw. Eingaben in die
// echte Sitzung speisen.
package probeguard

import (
	"fmt"
	"strings"
)

// NurVirtuell lässt ein Probe-Programm nur im virtuellen KWin von scripts/probe.sh zu. Verlangt werden vier Dinge
// zugleich, die probe.sh gemeinsam setzt und die die echte Sitzung nie alle erfüllt: die Marke PROBE_INNEN=1, ein
// privater D-Bus (DBUS_SESSION_BUS_ADDRESS gesetzt und verschieden von der Adresse des äußeren Aufrufs PROBE_AUSSENBUS;
// ohne Busadresse fiele godbus auf den Bus der echten Sitzung zurück), der Wayland-Socket „schnapp-probe“ und ein
// Laufzeitordner, der auf /tmp/probe/run endet. Eine einzelne, versehentlich gesetzte Umgebungsvariable genügt nicht.
// getenv kommt von außen, damit der Test ohne echte Umgebung auskommt.
func NurVirtuell(getenv func(string) string) error {
	if getenv("PROBE_INNEN") != "1" {
		return fmt.Errorf("PROBE_INNEN ist nicht 1 — nur im virtuellen KWin von scripts/probe.sh")
	}
	if b := getenv("DBUS_SESSION_BUS_ADDRESS"); b == "" || b == getenv("PROBE_AUSSENBUS") {
		return fmt.Errorf("kein privater D-Bus (DBUS_SESSION_BUS_ADDRESS leer oder gleich dem äußeren Bus) — nur im virtuellen KWin von scripts/probe.sh")
	}
	if d := getenv("WAYLAND_DISPLAY"); d != "schnapp-probe" {
		return fmt.Errorf("WAYLAND_DISPLAY ist %q, erwartet \"schnapp-probe\" — nur im virtuellen KWin von scripts/probe.sh", d)
	}
	if r := getenv("XDG_RUNTIME_DIR"); !strings.HasSuffix(r, "/tmp/probe/run") {
		return fmt.Errorf("XDG_RUNTIME_DIR endet nicht auf /tmp/probe/run — nur im virtuellen KWin von scripts/probe.sh")
	}
	return nil
}
