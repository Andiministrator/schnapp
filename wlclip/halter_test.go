// SPDX-License-Identifier: GPL-3.0-or-later

package wlclip

import (
	"bytes"
	"errors"
	"github.com/Andiministrator/schnapp/internal/wlwire"
	"io"
	"math/rand/v2"
	"net"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

// paar liefert zwei verbundene Enden: Client (wie zum Compositor) und Server (der nachgebaute Compositor).
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

// compositor ist der nachgebaute Compositor im Test. Fehler meldet er über t.Error (läuft in einer Goroutine).
type compositor struct {
	t *testing.T
	v *wlwire.Verbindung
}

// erwarte liest die nächste Anfrage und prüft Objekt und Opcode.
func (c *compositor) erwarte(obj uint32, op uint16, was string) *wlwire.Leser {
	c.v.SetReadDeadline(time.Now().Add(5 * time.Second))
	e, err := c.v.Lies()
	if err != nil {
		c.t.Errorf("%s: %v", was, err)
		return &wlwire.Leser{Err: err}
	}
	if (obj != 0 && e.Obj != obj) || e.Op != op {
		c.t.Errorf("%s: Anfrage an Objekt %d Opcode %d, erwartet Objekt %d Opcode %d", was, e.Obj, e.Op, obj, op)
	}
	return &wlwire.Leser{B: e.Args}
}

func (c *compositor) Sende(obj uint32, op uint16, n *wlwire.Nachricht) {
	if err := c.v.Sende(obj, op, n); err != nil {
		c.t.Errorf("senden an %d/%d: %v", obj, op, err)
	}
}

type globalImTest struct {
	name    uint32
	schnitt string
	version uint32
}

// anmeldung spielt get_registry und den ersten Rundlauf mit den Globals; liefert die Registry-ID.
func (c *compositor) anmeldung(globals []globalImTest) uint32 {
	reg := c.erwarte(idDisplay, opDisplayGetRegistry, "get_registry").U32()
	cb := c.erwarte(idDisplay, opDisplaySync, "sync 1").U32()
	for _, g := range globals {
		c.Sende(reg, evRegistryGlobal, new(wlwire.Nachricht).U32(g.name).Str(g.schnitt).U32(g.version))
	}
	c.Sende(cb, evCallbackDone, new(wlwire.Nachricht).U32(1))
	return reg
}

// ids sind die Objekte, die der Client nach der Anmeldung anlegt.
type ids struct{ sitz, mgr, quelle, geraet uint32 }

// setzen erwartet bind(Sitz), bind(Manager), create_data_source, offer, get_data_device, set_selection und den
// zweiten sync; vor dessen Antwort ruft es vorAntwort.
func (c *compositor) setzen(reg, sitzName, mgrName uint32, mgr, mime string, vorAntwort func(ids, uint32)) ids {
	var i ids
	l := c.erwarte(reg, opRegistryBind, "bind wl_seat")
	if n, s, v := l.U32(), l.Str(), l.U32(); n != sitzName || s != "wl_seat" || v != 1 {
		c.t.Errorf("bind Sitz: %d %q v%d", n, s, v)
	}
	i.sitz = l.U32()
	l = c.erwarte(reg, opRegistryBind, "bind Manager")
	if n, s, v := l.U32(), l.Str(), l.U32(); n != mgrName || s != mgr || v != 1 {
		c.t.Errorf("bind Manager: %d %q v%d, erwartet %d %q v1", n, s, v, mgrName, mgr)
	}
	i.mgr = l.U32()
	i.quelle = c.erwarte(i.mgr, opManagerCreateSource, "create_data_source").U32()
	for _, want := range []string{mime, MarkeKlipper} {
		if m := c.erwarte(i.quelle, opSourceOffer, "offer").Str(); m != want {
			c.t.Errorf("offer %q, erwartet %q", m, want)
		}
	}
	l = c.erwarte(i.mgr, opManagerGetDevice, "get_data_device")
	i.geraet = l.U32()
	if s := l.U32(); s != i.sitz {
		c.t.Errorf("get_data_device mit Sitz %d, erwartet %d", s, i.sitz)
	}
	if q := c.erwarte(i.geraet, opDeviceSetSelection, "set_selection").U32(); q != i.quelle {
		c.t.Errorf("set_selection mit Quelle %d, erwartet %d", q, i.quelle)
	}
	cb := c.erwarte(idDisplay, opDisplaySync, "sync 2").U32()
	if vorAntwort != nil {
		vorAntwort(i, cb)
	}
	c.Sende(cb, evCallbackDone, new(wlwire.Nachricht).U32(2))
	return i
}

var normaleGlobals = []globalImTest{
	{7, "wl_compositor", 6}, {3, "wl_seat", 9}, {4, ManagerWlr, 2}, {5, ManagerExt, 1}, {8, "wl_seat", 9},
}

func zufall(n int) []byte {
	r := rand.New(rand.NewPCG(1, 2))
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(r.UintN(256))
	}
	return b
}

// TestBieteUndDiene: ganzer Ablauf — ext bevorzugt, erster Sitz, eigenes Angebot freigegeben, Einfügen liefert alle
// Bytes (größer als der Pipe-Puffer), fremder MIME bekommt nichts, cancelled beendet Diene ohne Fehler.
func TestBieteUndDiene(t *testing.T) {
	client, server := paar(t)
	daten := zufall(300 << 10)
	c := &compositor{t: t, v: server}
	fertig := make(chan struct{})
	go func() {
		defer close(fertig)
		reg := c.anmeldung(normaleGlobals)
		i := c.setzen(reg, 3, 5, ManagerExt, "image/png", func(i ids, _ uint32) {
			c.Sende(i.geraet, evDeviceDataOffer, new(wlwire.Nachricht).U32(0xff000001))
		})
		c.erwarte(0xff000001, opOfferDestroy, "Angebot freigeben")
		// fremder MIME-Typ und die Klipper-Marke: Pipe wird ohne Daten geschlossen
		for _, fremd := range []string{"text/plain", MarkeKlipper} {
			r, w, _ := os.Pipe()
			c.Sende(i.quelle, evSourceSend, new(wlwire.Nachricht).Str(fremd).FD(int(w.Fd())))
			w.Close()
			if b, _ := io.ReadAll(r); len(b) != 0 {
				t.Errorf("%s bekam %d Byte", fremd, len(b))
			}
			r.Close()
		}
		// Einfügen
		r, w, _ := os.Pipe()
		c.Sende(i.quelle, evSourceSend, new(wlwire.Nachricht).Str("image/png").FD(int(w.Fd())))
		w.Close()
		b, _ := io.ReadAll(r)
		r.Close()
		if !bytes.Equal(b, daten) {
			t.Errorf("eingefügt %d Byte, erwartet %d gleiche", len(b), len(daten))
		}
		c.Sende(i.quelle, evSourceCancel, nil)
	}()
	h, err := biete(client, daten, "image/png")
	if err != nil {
		t.Fatal(err)
	}
	if h.Manager != ManagerExt {
		t.Errorf("Manager %q, erwartet %q", h.Manager, ManagerExt)
	}
	if err := h.Diene(5 * time.Second); err != nil {
		t.Errorf("Diene: %v, erwartet nil (abgelöst)", err)
	}
	<-fertig
}

// TestNurWlr: ohne ext-data-control nimmt der Halter zwlr_data_control_manager_v1.
func TestNurWlr(t *testing.T) {
	client, server := paar(t)
	c := &compositor{t: t, v: server}
	go func() {
		reg := c.anmeldung([]globalImTest{{3, "wl_seat", 9}, {4, ManagerWlr, 2}})
		c.setzen(reg, 3, 4, ManagerWlr, "image/png", nil)
	}()
	h, err := biete(client, []byte{1}, "image/png")
	if err != nil {
		t.Fatal(err)
	}
	if h.Manager != ManagerWlr {
		t.Errorf("Manager %q, erwartet %q", h.Manager, ManagerWlr)
	}
}

// TestOhneDataControlOderSitz: fehlt das Protokoll oder der Sitz, setzt der Halter nichts.
func TestOhneDataControlOderSitz(t *testing.T) {
	for _, fall := range []struct {
		name    string
		globals []globalImTest
		want    error
	}{
		{"ohne data-control", []globalImTest{{3, "wl_seat", 9}, {7, "wl_data_device_manager", 3}}, ErrKeinDataControl},
		{"ohne Sitz", []globalImTest{{5, ManagerExt, 1}}, ErrKeinSitz},
	} {
		t.Run(fall.name, func(t *testing.T) {
			client, server := paar(t)
			c := &compositor{t: t, v: server}
			go c.anmeldung(fall.globals)
			if _, err := biete(client, []byte{1}, "image/png"); !errors.Is(err, fall.want) {
				t.Errorf("Fehler %v, erwartet %v", err, fall.want)
			}
		})
	}
}

// TestProtokollfehler: wl_display.error beim Setzen kommt als Fehler mit dem Text des Compositors zurück.
func TestProtokollfehler(t *testing.T) {
	client, server := paar(t)
	c := &compositor{t: t, v: server}
	fertig := make(chan struct{})
	defer func() { <-fertig }() // Compositor-Goroutine endet vor dem Test (kein t.Errorf nach Testende)
	go func() {
		defer close(fertig)
		reg := c.anmeldung(normaleGlobals)
		c.setzen(reg, 3, 5, ManagerExt, "image/png", func(i ids, _ uint32) {
			c.Sende(idDisplay, evDisplayError, new(wlwire.Nachricht).U32(i.geraet).U32(1).Str("used source"))
		})
	}()
	_, err := biete(client, []byte{1}, "image/png")
	if err == nil || !strings.Contains(err.Error(), "used source") {
		t.Errorf("Fehler %v, erwartet den Protokollfehler", err)
	}
}

// TestSofortAbgeloest: cancelled schon beim Setzen ist ein Fehler (der Editor bietet dann selbst an).
func TestSofortAbgeloest(t *testing.T) {
	client, server := paar(t)
	c := &compositor{t: t, v: server}
	fertig := make(chan struct{})
	defer func() { <-fertig }()
	go func() {
		defer close(fertig)
		reg := c.anmeldung(normaleGlobals)
		c.setzen(reg, 3, 5, ManagerExt, "image/png", func(i ids, _ uint32) {
			c.Sende(i.quelle, evSourceCancel, nil)
		})
	}()
	if _, err := biete(client, []byte{1}, "image/png"); !errors.Is(err, ErrAbgeloest) {
		t.Errorf("Fehler %v, erwartet ErrAbgeloest", err)
	}
}

// bereit liefert einen gesetzten Halter und den Compositor dazu.
func bereit(t *testing.T) (*Halter, *compositor, ids) {
	t.Helper()
	client, server := paar(t)
	c := &compositor{t: t, v: server}
	ch := make(chan ids, 1)
	go func() {
		reg := c.anmeldung(normaleGlobals)
		ch <- c.setzen(reg, 3, 5, ManagerExt, "image/png", nil)
	}()
	h, err := biete(client, []byte("bild"), "image/png")
	if err != nil {
		t.Fatal(err)
	}
	return h, c, <-ch
}

// TestZeitgrenze: ohne Ereignis endet Diene nach der Grenze mit ErrZeitgrenze.
func TestZeitgrenze(t *testing.T) {
	alt := scheibe
	scheibe = 20 * time.Millisecond // mehrere Scheiben bis zur Grenze
	defer func() { scheibe = alt }()
	h, _, _ := bereit(t)
	start := time.Now()
	if err := h.Diene(100 * time.Millisecond); !errors.Is(err, ErrZeitgrenze) {
		t.Errorf("Fehler %v, erwartet ErrZeitgrenze", err)
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("Diene endete nach %v", d)
	}
}

// TestCompositorWeg: schließt der Compositor die Verbindung, endet Diene mit ErrVerbindungZu.
func TestCompositorWeg(t *testing.T) {
	h, c, _ := bereit(t)
	c.v.Close()
	if err := h.Diene(5 * time.Second); !errors.Is(err, ErrVerbindungZu) {
		t.Errorf("Fehler %v, erwartet ErrVerbindungZu", err)
	}
}

// TestGeraetBeendet: finished am Gerät beendet Diene ohne Fehler.
func TestGeraetBeendet(t *testing.T) {
	h, c, i := bereit(t)
	c.Sende(i.geraet, evDeviceFinished, nil)
	if err := h.Diene(5 * time.Second); err != nil {
		t.Errorf("Fehler %v, erwartet nil", err)
	}
}

// TestSchreibeStau: ein Leser, der nicht liest, hält das Senden höchstens die Stau-Zeit auf; ein geschlossener Leser
// beendet es mit Fehler statt Absturz (SIGPIPE).
func TestSchreibeStau(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	fd, err := syscall.Dup(int(w.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	w.Close()
	start := time.Now()
	if err := schreibe(fd, zufall(4<<20), 100*time.Millisecond); err == nil {
		t.Error("kein Fehler bei stockendem Leser")
	}
	if d := time.Since(start); d > 3*time.Second {
		t.Errorf("Aufgabe nach %v", d)
	}

	r2, w2, _ := os.Pipe()
	fd2, _ := syscall.Dup(int(w2.Fd()))
	w2.Close()
	r2.Close()
	if err := schreibe(fd2, zufall(1<<20), time.Second); err == nil {
		t.Error("kein Fehler bei geschlossenem Leser")
	}
}
