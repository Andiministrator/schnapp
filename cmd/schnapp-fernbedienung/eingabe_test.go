// SPDX-License-Identifier: GPL-3.0-or-later

//go:build probe

package main

import (
	"encoding/xml"
	"net"
	"os"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/wlwire"
)

// fake-input.xml ist eine unveränderte Kopie des Protokolls org_kde_kwin_fake_input aus KDE KWin
// (src/wayland/protocols/fake-input.xml, SPDX-FileCopyrightText 2015 Martin Gräßlin, SPDX-License-Identifier:
// LGPL-2.1-or-later — verträglich mit GPL-3.0-or-later). Sie dient nur diesem Test: Er liest die Opcodes aus der
// Reihenfolge der Anfragen und prüft die Konstanten und Argumente der Fernbedienung dagegen.

type protokoll struct {
	Schnittstelle struct {
		Anfragen []struct {
			Name string `xml:"name,attr"`
			Args []struct {
				Name string `xml:"name,attr"`
				Typ  string `xml:"type,attr"`
			} `xml:"arg"`
		} `xml:"request"`
	} `xml:"interface"`
}

// opcodes liefert Anfragename -> (Opcode, Argumenttypen) aus fake-input.xml.
func opcodes(t *testing.T) map[string]struct {
	op   uint16
	args []string
} {
	t.Helper()
	b, err := os.ReadFile("fake-input.xml")
	if err != nil {
		t.Fatal(err)
	}
	var p protokoll
	if err := xml.Unmarshal(b, &p); err != nil {
		t.Fatal(err)
	}
	out := map[string]struct {
		op   uint16
		args []string
	}{}
	for i, a := range p.Schnittstelle.Anfragen {
		var typen []string
		for _, arg := range a.Args {
			typen = append(typen, arg.Typ)
		}
		out[a.Name] = struct {
			op   uint16
			args []string
		}{uint16(i), typen}
	}
	return out
}

// TestOpcodesGegenProtokoll: die Konstanten der Fernbedienung stimmen mit den Anfragen im Protokoll überein.
func TestOpcodesGegenProtokoll(t *testing.T) {
	p := opcodes(t)
	for name, op := range map[string]uint16{
		"authenticate": opAuthenticate, "pointer_motion": opPointerMotion, "button": opButton, "axis": opAxis,
		"pointer_motion_absolute": opPointerAbsolute, "keyboard_key": opKeyboardKey, "keyboard_keysym": opKeysym,
	} {
		if got, ok := p[name]; !ok || got.op != op {
			t.Errorf("%s: Konstante %d, Protokoll %v (vorhanden %v)", name, op, got.op, ok)
		}
	}
}

// attrappeKWin lauscht auf einem Socket im Testordner und spielt den Anfang (Registry, Bind) eines KWin mit
// fake_input in der Version version; die Verbindung der Fernbedienung kommt über den Kanal zurück.
func attrappeKWin(t *testing.T, version uint32) <-chan *wlwire.Verbindung {
	t.Helper()
	// Ein Unix-Socket-Pfad darf höchstens 107 Byte lang sein, ein langer TMPDIR (Sub-Agents, Mutantenlauf) sprengt das
	// (F-605). Statt in ein anderes Verzeichnis auszuweichen (nie ins echte XDG_RUNTIME_DIR, F-614), wechselt der Test in
	// den eigenen Testordner und benutzt den relativen Namen „s“: SocketPfad macht aus XDG_RUNTIME_DIR=„.“ und
	// WAYLAND_DISPLAY=„s“ den Pfad „s“, der gegen das Arbeitsverzeichnis aufgelöst wird.
	t.Chdir(t.TempDir())
	const pfad = "s"
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: pfad, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	t.Setenv("XDG_RUNTIME_DIR", ".")
	t.Setenv("WAYLAND_DISPLAY", pfad)
	ch := make(chan *wlwire.Verbindung, 1)
	go func() {
		c, err := l.AcceptUnix()
		if err != nil {
			return
		}
		s := wlwire.Neue(c)
		erwarte := func(obj uint32, op uint16, was string) *wlwire.Leser {
			s.SetReadDeadline(time.Now().Add(5 * time.Second))
			e, err := s.Lies()
			if err != nil {
				t.Errorf("%s: %v", was, err)
				return &wlwire.Leser{Err: err}
			}
			if (obj != 0 && e.Obj != obj) || e.Op != op {
				t.Errorf("%s: Objekt %d Opcode %d, erwartet Objekt %d Opcode %d", was, e.Obj, e.Op, obj, op)
			}
			return &wlwire.Leser{B: e.Args}
		}
		reg := erwarte(wlwire.IDDisplay, wlwire.OpDisplayGetRegistry, "get_registry").U32()
		cb := erwarte(wlwire.IDDisplay, wlwire.OpDisplaySync, "sync 1").U32()
		s.Sende(reg, wlwire.EvRegistryGlobal, new(wlwire.Nachricht).U32(9).Str(schnittstelle).U32(version))
		s.Sende(cb, wlwire.EvCallbackDone, new(wlwire.Nachricht).U32(1))
		l2 := erwarte(reg, wlwire.OpRegistryBind, "bind")
		if n, i, v, id := l2.U32(), l2.Str(), l2.U32(), l2.U32(); n != 9 || i != schnittstelle || v != min(version, hoechsteVers) || id < 3 {
			t.Errorf("bind: %d %q v%d id %d", n, i, v, id)
		}
		ch <- s
	}()
	return ch
}

// TestEingabeOpcodesUndArgumente: Anmeldung und jede Anfrage der Fernbedienung kommen mit dem Opcode aus dem
// Protokoll und den erwarteten Argumenten (Festkomma 24.8) beim Compositor an.
func TestEingabeOpcodesUndArgumente(t *testing.T) {
	p := opcodes(t)
	ch := attrappeKWin(t, 6)
	fertig := make(chan struct{})
	e := make(chan *Eingabe, 1)
	go func() {
		eingabe, err := VerbindeEingabe()
		if err != nil {
			t.Error(err)
			close(e)
			return
		}
		e <- eingabe
	}()
	var s *wlwire.Verbindung
	select {
	case s = <-ch:
	case <-time.After(10 * time.Second):
		t.Fatal("Attrappe bekam keine Verbindung")
	}
	lies := func(was string) (wlwire.Ereignis, *wlwire.Leser) {
		s.SetReadDeadline(time.Now().Add(5 * time.Second))
		ev, err := s.Lies()
		if err != nil {
			t.Fatalf("%s: %v", was, err)
		}
		return ev, &wlwire.Leser{B: ev.Args}
	}
	// authenticate, dann der Abgleich (sync)
	ev, l := lies("authenticate")
	if ev.Op != p["authenticate"].op || l.Str() != "schnapp-probe" || l.Str() != "Probelauf" || l.Err != nil {
		t.Errorf("authenticate falsch: %+v", ev)
	}
	obj := ev.Obj
	ev, l = lies("sync")
	if ev.Obj != wlwire.IDDisplay || ev.Op != wlwire.OpDisplaySync {
		t.Fatalf("erwartet sync, bekam %+v", ev)
	}
	s.Sende(l.U32(), wlwire.EvCallbackDone, new(wlwire.Nachricht).U32(1))
	var eingabe *Eingabe
	select {
	case eingabe = <-e:
	case <-time.After(10 * time.Second):
		t.Fatal("VerbindeEingabe kehrte nicht zurück")
	}
	if eingabe == nil {
		t.FailNow()
	}
	defer close(fertig)
	defer eingabe.Close()
	if eingabe.Version != 6 {
		t.Errorf("Version %d", eingabe.Version)
	}

	fix := func(f float64) uint32 { return uint32(int32(f * 256)) }
	for _, f := range []struct {
		name string
		tu   func() error
		args []uint32
	}{
		{"pointer_motion_absolute", func() error { return eingabe.Zeiger(500, -3) }, []uint32{fix(500), fix(-3)}},
		{"button", func() error { return eingabe.Maus(btnRechts, gedrueckt) }, []uint32{btnRechts, gedrueckt}},
		{"axis", func() error { return eingabe.Rad(-15) }, []uint32{0, fix(-15)}},
		{"keyboard_key", func() error { return eingabe.Tastatur(29, los) }, []uint32{29, los}},
		{"keyboard_keysym", func() error { return eingabe.Keysym(0x20ac, gedrueckt) }, []uint32{0x20ac, gedrueckt}},
	} {
		if err := f.tu(); err != nil {
			t.Fatalf("%s: %v", f.name, err)
		}
		ev, l := lies(f.name)
		soll := p[f.name]
		if ev.Obj != obj || ev.Op != soll.op {
			t.Errorf("%s: Objekt %d Opcode %d, erwartet Objekt %d Opcode %d", f.name, ev.Obj, ev.Op, obj, soll.op)
		}
		if len(soll.args) != len(f.args) {
			t.Errorf("%s: Test kennt %d Argumente, Protokoll %d", f.name, len(f.args), len(soll.args))
		}
		for i, w := range f.args {
			if got := l.U32(); got != w {
				t.Errorf("%s: Argument %d = %#x, erwartet %#x", f.name, i, got, w)
			}
		}
		if l.Err != nil || len(l.B) != 0 {
			t.Errorf("%s: Argumente unvollständig gelesen (%v, Rest %d Byte)", f.name, l.Err, len(l.B))
		}
	}
}

// TestEingabeVersionen: ältere Compositor-Versionen lehnen Anfragen ab, die sie nicht kennen.
func TestEingabeVersionen(t *testing.T) {
	e := &Eingabe{Version: 2}
	if err := e.Zeiger(1, 1); err == nil {
		t.Error("pointer_motion_absolute bei Version 2 ohne Fehler")
	}
	if err := e.Tastatur(1, 1); err == nil {
		t.Error("keyboard_key bei Version 2 ohne Fehler")
	}
}
