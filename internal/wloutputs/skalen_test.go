// SPDX-License-Identifier: GPL-3.0-or-later

package wloutputs

import (
	"errors"
	"net"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/wlwire"
)

func paar(t *testing.T) (client, server *wlwire.Verbindung) {
	t.Helper()
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM|syscall.SOCK_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	ende := func(fd int) *net.UnixConn {
		f := os.NewFile(uintptr(fd), "paar")
		defer f.Close()
		c, err := net.FileConn(f)
		if err != nil {
			t.Fatal(err)
		}
		return c.(*net.UnixConn)
	}
	client, server = wlwire.Neue(ende(fds[0])), wlwire.Neue(ende(fds[1]))
	t.Cleanup(func() { client.Close(); server.Close() })
	return client, server
}

// monitor ist ein Ausgang des nachgebauten Compositors.
type monitor struct {
	version             uint32 // wl_output-Version; unter 4 ohne wl_output.name
	name                string
	breite, hoehe       int32
	logBreite, logHoehe int32
}

// compositor spielt die Gegenseite: Globale, Bind, get_xdg_output und sync. Ereignisse eines gebundenen Ausgangs
// schickt er beim nächsten sync, wie ein echter Compositor sie vor der Antwort auf sync schickt.
func compositor(t *testing.T, s *wlwire.Verbindung, monitore []monitor, mitManager bool) {
	t.Helper()
	var registry uint32
	gebunden := map[uint32]int{} // Objekt-ID -> Index in monitore
	xdg := map[uint32]int{}      // xdg-ID -> Index
	var offen []func()
	u := func(x int32) uint32 { return uint32(x) }
	for {
		e, err := s.Lies()
		if err != nil {
			return
		}
		l := wlwire.Leser{B: e.Args}
		switch {
		case e.Obj == wlwire.IDDisplay && e.Op == wlwire.OpDisplayGetRegistry:
			registry = l.U32()
			// Ein fremdes Global, die Ausgänge, der Manager (Namen 1, 10…, 99).
			s.Sende(registry, wlwire.EvRegistryGlobal, new(wlwire.Nachricht).U32(1).Str("wl_seat").U32(7))
			for i, m := range monitore {
				s.Sende(registry, wlwire.EvRegistryGlobal, new(wlwire.Nachricht).U32(uint32(10+i)).Str("wl_output").U32(m.version))
			}
			if mitManager {
				s.Sende(registry, wlwire.EvRegistryGlobal, new(wlwire.Nachricht).U32(99).Str("zxdg_output_manager_v1").U32(3))
			}
		case e.Obj == registry && e.Op == wlwire.OpRegistryBind:
			name, schnitt, _, id := l.U32(), l.Str(), l.U32(), l.U32()
			if schnitt == "wl_output" {
				i := int(name - 10)
				gebunden[id] = i
				m := monitore[i]
				offen = append(offen, func() {
					// Nicht aktuelle Modi vor und nach dem aktuellen; sie dürfen nicht zählen.
					s.Sende(id, evOutputMode, new(wlwire.Nachricht).U32(0).U32(640).U32(480).U32(60000))
					s.Sende(id, evOutputMode, new(wlwire.Nachricht).U32(modusAktuell).U32(u(m.breite)).U32(u(m.hoehe)).U32(60000))
					s.Sende(id, evOutputMode, new(wlwire.Nachricht).U32(0).U32(800).U32(600).U32(60000))
					if m.version >= 4 {
						s.Sende(id, evOutputName, new(wlwire.Nachricht).Str(m.name))
					}
				})
			}
		case e.Op == opManagerGetXdgOutput:
			id, out := l.U32(), l.U32()
			i := gebunden[out]
			xdg[id] = i
			m := monitore[i]
			offen = append(offen, func() {
				s.Sende(id, evXdgLogicalSize, new(wlwire.Nachricht).U32(u(m.logBreite)).U32(u(m.logHoehe)))
				s.Sende(id, evXdgName, new(wlwire.Nachricht).Str(m.name))
			})
		case e.Obj == wlwire.IDDisplay && e.Op == wlwire.OpDisplaySync:
			cb := l.U32()
			for _, f := range offen {
				f()
			}
			offen = nil
			s.Sende(cb, wlwire.EvCallbackDone, new(wlwire.Nachricht).U32(1))
		}
	}
}

func TestSkalen(t *testing.T) {
	c, s := paar(t)
	monitore := []monitor{
		{4, "DP-1", 3840, 2160, 2560, 1440},    // 1,5
		{4, "HDMI-A-1", 1920, 1080, 864, 1536}, // gedreht, 1,25
		{3, "eDP-1", 2880, 1800, 1646, 1029},   // ohne wl_output.name, 1,75 (gerundet)
		{4, "DP-2", 1920, 1080, 0, 0},          // ohne logische Größe: fehlt
	}
	go compositor(t, s, monitore, true)
	got, err := lies(c, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]float64{"DP-1": 1.5, "HDMI-A-1": 1.25, "eDP-1": 1.75}
	if len(got) != len(want) {
		t.Fatalf("Skalen %v, erwartet %v", got, want)
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("%s: %v, erwartet %v", k, got[k], w)
		}
	}
}

func TestOhneManager(t *testing.T) {
	c, s := paar(t)
	go compositor(t, s, []monitor{{4, "DP-1", 1920, 1080, 1920, 1080}}, false)
	if _, err := lies(c, time.Second); !errors.Is(err, ErrOhneXdgOutput) {
		t.Errorf("Fehler %v, erwartet ErrOhneXdgOutput", err)
	}
}

func TestSkalaUndProzent(t *testing.T) {
	for _, f := range []struct {
		b, h, lb, lh int32
		want         float64
		ok           bool
	}{
		{1920, 1080, 1920, 1080, 1, true},
		{2560, 1440, 2048, 1152, 1.25, true},
		{1080, 1920, 1536, 864, 1.25, true},
		{0, 0, 10, 10, 0, false},
		{10, 10, 0, 0, 0, false},
		{10, 10, -5, -5, 0, false},
	} {
		got, ok := Skala(f.b, f.h, f.lb, f.lh)
		if got != f.want || ok != f.ok {
			t.Errorf("Skala(%d,%d,%d,%d) = %v, %v; erwartet %v, %v", f.b, f.h, f.lb, f.lh, got, ok, f.want, f.ok)
		}
	}
	if p := Prozent(1.75); p != 175 {
		t.Errorf("Prozent(1,75) = %d", p)
	}
}
