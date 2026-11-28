// SPDX-License-Identifier: GPL-3.0-or-later

package notify

import (
	"bufio"
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

// eigenerBus startet einen leeren Sitzungsbus nur für diesen Test (nie den der Sitzung) und richtet
// DBUS_SESSION_BUS_ADDRESS darauf. Ohne dbus-daemon wird der Test übersprungen.
func eigenerBus(t *testing.T) {
	t.Helper()
	prog, err := exec.LookPath("dbus-daemon")
	if err != nil {
		t.Skip("dbus-daemon fehlt")
	}
	c := exec.Command(prog, "--session", "--nofork", "--print-address=1")
	aus, err := c.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Process.Kill(); _ = c.Wait() })
	adresse, err := bufio.NewReader(aus).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", strings.TrimSpace(adresse))
}

// sende schickt ActionInvoked von einer eigenen Verbindung, mit dem Namen des Dienstes oder ohne.
func sende(t *testing.T, alsDienst bool, id uint32) {
	t.Helper()
	c, err := dbus.ConnectSessionBus()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if alsDienst {
		if r, err := c.RequestName(dienst, dbus.NameFlagDoNotQueue); err != nil || r != dbus.RequestNameReplyPrimaryOwner {
			t.Fatalf("Name: %v %v", r, err)
		}
	}
	if err := c.Emit(dbus.ObjectPath(pfad), dienst+".ActionInvoked", id, "cancel"); err != nil {
		t.Fatal(err)
	}
	var s string
	_ = c.BusObject().Call("org.freedesktop.DBus.GetId", 0).Store(&s) // Rundlauf: das Signal ist beim Bus
}

// F-1052, F-1054 (Kritik der Sitzung 28.11.): Das echte Abonnement lebt über die Aufbaufrist hinaus, bis sein Kontext
// endet, und nimmt nur Signale des Benachrichtigungsdienstes an; ein anderes Programm bricht nichts ab.
func TestAboAufEigenemBus(t *testing.T) {
	eigenerBus(t)
	ctx, ende := context.WithCancel(context.Background())
	defer ende()
	k, err := Aktionen(ctx, 800*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(900 * time.Millisecond) // länger als die Aufbaufrist
	sende(t, false, 3)
	select {
	case a := <-k:
		t.Fatalf("Signal eines fremden Absenders angenommen: %+v", a)
	case <-time.After(300 * time.Millisecond):
	}
	sende(t, true, 7)
	select {
	case a := <-k:
		if a != (Aktion{ID: 7, Schluessel: "cancel"}) {
			t.Errorf("Aktion %+v", a)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Klick des Dienstes kam nicht an")
	}
}
