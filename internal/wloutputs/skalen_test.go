// SPDX-License-Identifier: GPL-3.0-or-later

package wloutputs

import (
	"errors"
	"fmt"
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
func compositor(t *testing.T, s *wlwire.Verbindung, monitore []monitor, mitManager bool, versionen chan<- string) {
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
				s.Sende(registry, wlwire.EvRegistryGlobal, new(wlwire.Nachricht).U32(99).Str("zxdg_output_manager_v1").U32(9))
			}
		case e.Obj == registry && e.Op == wlwire.OpRegistryBind:
			name, schnitt, version, id := l.U32(), l.Str(), l.U32(), l.U32()
			if versionen != nil {
				versionen <- fmt.Sprintf("%s %d", schnitt, version)
			}
			if schnitt == "wl_output" {
				i := int(name - 10)
				gebunden[id] = i
				m := monitore[i]
				offen = append(offen, func() {
					// Nicht aktuelle Modi vor und nach dem aktuellen; sie dürfen nicht zählen.
					s.Sende(id, 1 /* wl_output.mode */, new(wlwire.Nachricht).U32(0).U32(640).U32(480).U32(60000))
					s.Sende(id, 1 /* wl_output.mode */, new(wlwire.Nachricht).U32(modusAktuell).U32(u(m.breite)).U32(u(m.hoehe)).U32(60000))
					s.Sende(id, 1 /* wl_output.mode */, new(wlwire.Nachricht).U32(0).U32(800).U32(600).U32(60000))
					if m.version >= 4 {
						s.Sende(id, 4 /* wl_output.name */, new(wlwire.Nachricht).Str(m.name))
					}
				})
			}
		case e.Op == 1 /* zxdg_output_manager_v1.get_xdg_output */ && e.Obj != wlwire.IDDisplay && e.Obj != registry:
			id, out := l.U32(), l.U32()
			i := gebunden[out]
			xdg[id] = i
			m := monitore[i]
			offen = append(offen, func() {
				s.Sende(id, 1 /* zxdg_output_v1.logical_size */, new(wlwire.Nachricht).U32(u(m.logBreite)).U32(u(m.logHoehe)))
				s.Sende(id, 3 /* zxdg_output_v1.name */, new(wlwire.Nachricht).Str(m.name))
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
		{7, "DP-3", 1920, 1080, 1536, 864},     // neuere Version angekündigt: gebunden wird höchstens 4; 1,25
	}
	versionen := make(chan string, 64)
	go compositor(t, s, monitore, true, versionen)
	got, err := lies(c, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]float64{"DP-1": 1.5, "HDMI-A-1": 1.25, "eDP-1": 1.75, "DP-3": 1.25}
	if len(got) != len(want) {
		t.Fatalf("Skalen %v, erwartet %v", got, want)
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("%s: %v, erwartet %v", k, got[k], w)
		}
	}
	// Gebunden wird wl_output höchstens in Version 4 (name), zxdg_output_manager_v1 höchstens in 3.
	soll := []string{"zxdg_output_manager_v1 3", "wl_output 4", "wl_output 4", "wl_output 3", "wl_output 4", "wl_output 4"}
	var gebunden []string
	for range soll { // die Binds kamen vor der Antwort auf sync, liegen also schon im Kanal
		select {
		case v := <-versionen:
			gebunden = append(gebunden, v)
		case <-time.After(time.Second):
		}
	}
	if fmt.Sprint(gebunden) != fmt.Sprint(soll) {
		t.Errorf("gebundene Versionen %v, erwartet %v", gebunden, soll)
	}
}

// Kritik C-2: Mehr Ausgaben als maxAusgaben (oder Globale als maxGlobale) brechen die Abfrage ab, statt alle zu binden.
func TestZuVieleAusgaben(t *testing.T) {
	c, s := paar(t)
	viele := make([]monitor, maxAusgaben+1)
	for i := range viele {
		viele[i] = monitor{4, fmt.Sprintf("V-%d", i), 1920, 1080, 1920, 1080}
	}
	go compositor(t, s, viele, true, nil)
	if _, err := lies(c, time.Second); !errors.Is(err, ErrZuViele) {
		t.Errorf("Fehler %v, erwartet ErrZuViele", err)
	}
}

// Kritik C-2: Zwei Ausgaben gleichen Namens bekommen keine Skala (unklar, welche gemeint ist); die übrigen schon.
func TestDoppelteNamen(t *testing.T) {
	c, s := paar(t)
	go compositor(t, s, []monitor{
		{4, "DP-1", 3840, 2160, 2560, 1440},
		{4, "DP-1", 1920, 1080, 1920, 1080},
		{4, "DP-2", 2560, 1440, 2048, 1152},
	}, true, nil)
	got, err := lies(c, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got["DP-2"] != 1.25 {
		t.Errorf("Skalen %v, erwartet nur DP-2: 1.25", got)
	}
}

func TestOhneManager(t *testing.T) {
	c, s := paar(t)
	go compositor(t, s, []monitor{{4, "DP-1", 1920, 1080, 1920, 1080}}, false, nil)
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
