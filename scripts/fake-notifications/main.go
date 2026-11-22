// SPDX-License-Identifier: GPL-3.0-or-later

// Command fake-notifications ist eine Attrappe des Dienstes org.freedesktop.Notifications für Probeläufe im
// virtuellen KWin (scripts/probe.sh, privater D-Bus): Sie nimmt Notify an, schreibt Titel und Text als eine Zeile in
// die Datei aus dem ersten Argument und antwortet mit einer fortlaufenden Kennung. Nach der Zeitgrenze aus dem zweiten
// Argument (Sekunden, höchstens 600) endet sie von selbst. Nie im ausgelieferten Programm.
//
//	fake-notifications PROTOKOLL SEKUNDEN
package main

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

type dienst struct {
	mu    sync.Mutex
	n     uint32
	datei *os.File
}

// Notify hat die Signatur aus der Spezifikation: susssasa{sv}i → u.
func (d *dienst) Notify(app string, ersetzt uint32, symbol, titel, text string, aktionen []string,
	hinweise map[string]dbus.Variant, dauer int32) (uint32, *dbus.Error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.n++
	fmt.Fprintf(d.datei, "notify app=%q titel=%q text=%q\n", app, titel, text)
	_ = d.datei.Sync()
	return d.n, nil
}

func (d *dienst) GetCapabilities() ([]string, *dbus.Error) { return []string{"body"}, nil }

func (d *dienst) GetServerInformation() (string, string, string, string, *dbus.Error) {
	return "fake-notifications", "schnapp", "1", "1.2", nil
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "Aufruf: fake-notifications PROTOKOLL SEKUNDEN")
		os.Exit(2)
	}
	sek, err := strconv.Atoi(os.Args[2])
	if err != nil || sek < 1 || sek > 600 {
		fmt.Fprintln(os.Stderr, "SEKUNDEN: 1 bis 600")
		os.Exit(2)
	}
	f, err := os.OpenFile(os.Args[1], os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	c, err := dbus.ConnectSessionBus()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	d := &dienst{datei: f}
	if err := c.Export(d, "/org/freedesktop/Notifications", "org.freedesktop.Notifications"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	antwort, err := c.RequestName("org.freedesktop.Notifications", dbus.NameFlagDoNotQueue)
	if err != nil || antwort != dbus.RequestNameReplyPrimaryOwner {
		fmt.Fprintln(os.Stderr, "Name nicht bekommen:", antwort, err)
		os.Exit(1)
	}
	fmt.Fprintln(f, "bereit")
	_ = f.Sync()
	time.Sleep(time.Duration(sek) * time.Second)
}
