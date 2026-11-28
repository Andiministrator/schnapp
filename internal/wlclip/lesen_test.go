// SPDX-License-Identifier: GPL-3.0-or-later

package wlclip

import (
	"bytes"
	"context"
	"errors"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/wlwire"
)

// leseAnmeldung spielt im Test den Compositor bis zum Rundlauf nach get_data_device: bind(Sitz), bind(Manager),
// get_data_device, sync; vor der Antwort ruft es angebot mit der ID des Geräts. Liefert die ID des Geräts.
func (c *compositor) leseAnmeldung(reg uint32, mgr string, angebot func(geraet uint32)) uint32 {
	l := c.erwarte(reg, opRegistryBind, "bind wl_seat")
	if n, s := l.U32(), l.Str(); n != 3 || s != "wl_seat" {
		c.t.Errorf("bind Sitz: %d %q", n, s)
	}
	l.U32()
	sitz := l.U32()
	l = c.erwarte(reg, opRegistryBind, "bind Manager")
	if _, s := l.U32(), l.Str(); s != mgr {
		c.t.Errorf("bind Manager %q, erwartet %q", s, mgr)
	}
	l.U32()
	mgrID := l.U32()
	l = c.erwarte(mgrID, opManagerGetDevice, "get_data_device")
	geraet := l.U32()
	if s := l.U32(); s != sitz {
		c.t.Errorf("get_data_device mit Sitz %d, erwartet %d", s, sitz)
	}
	cb := c.erwarte(idDisplay, opDisplaySync, "sync").U32()
	if angebot != nil {
		angebot(geraet)
	}
	c.Sende(cb, evCallbackDone, new(wlwire.Nachricht).U32(2))
	return geraet
}

// biete schickt im Test ein Angebot mit den Typen und setzt es (bei auswahl) als Auswahl.
func (c *compositor) biete(geraet, id uint32, typen []string, auswahl bool) {
	c.Sende(geraet, evDeviceDataOffer, new(wlwire.Nachricht).U32(id))
	for _, t := range typen {
		c.Sende(id, evOfferOffer, new(wlwire.Nachricht).Str(t))
	}
	if auswahl {
		c.Sende(geraet, evDeviceSelection, new(wlwire.Nachricht).U32(id))
	}
}

// liefere erwartet receive mit typ und schreibt daten in den Deskriptor.
func (c *compositor) liefere(id uint32, typ string, daten []byte) {
	l := c.erwarte(id, opOfferReceive, "receive")
	if m := l.Str(); m != typ {
		c.t.Errorf("receive %q, erwartet %q", m, typ)
	}
	fd, err := c.v.NimmFD()
	if err != nil {
		c.t.Errorf("receive ohne Deskriptor: %v", err)
		return
	}
	f := os.NewFile(uintptr(fd), "liefern")
	f.Write(daten)
	f.Close()
}

// TestLies: ganzer Ablauf — Typen der Auswahl in Reihenfolge, das fremde Angebot (primäre Auswahl) wird freigegeben,
// gewählter Typ geliefert (größer als ein Stück), danach die Auswahl freigegeben.
func TestLies(t *testing.T) {
	client, server := paar(t)
	c := &compositor{t: t, v: server}
	daten := zufall(200 << 10)
	typen := []string{MIMEObjekte, MIME, MarkeKlipper}
	fertig := make(chan struct{})
	go func() {
		defer close(fertig)
		reg := c.anmeldung(normaleGlobals)
		c.leseAnmeldung(reg, ManagerExt, func(g uint32) {
			c.biete(g, 0xff000002, []string{"text/plain"}, false)
			c.biete(g, 0xff000001, typen, true)
		})
		c.erwarte(0xff000002, opOfferDestroy, "fremdes Angebot freigeben")
		c.liefere(0xff000001, MIMEObjekte, daten)
		c.erwarte(0xff000001, opOfferDestroy, "Auswahl freigeben")
	}()
	var gesehen []string
	g, err := lies(context.Background(), client, func(t []string) string { gesehen = t; return MIMEObjekte }, 1<<20, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(gesehen, typen) || !slices.Equal(g.Typen, typen) {
		t.Errorf("Typen %v / %v, erwartet %v", gesehen, g.Typen, typen)
	}
	if g.Typ != MIMEObjekte || !bytes.Equal(g.Daten, daten) {
		t.Errorf("gelesen %q mit %d Byte, erwartet %q mit %d gleichen", g.Typ, len(g.Daten), MIMEObjekte, len(daten))
	}
	<-fertig
}

// TestLiesNichtsGewaehlt: wahl liefert leer — kein receive, nur die Typen.
func TestLiesNichtsGewaehlt(t *testing.T) {
	client, server := paar(t)
	c := &compositor{t: t, v: server}
	fertig := make(chan struct{})
	go func() {
		defer close(fertig)
		reg := c.anmeldung(normaleGlobals)
		c.leseAnmeldung(reg, ManagerExt, func(g uint32) { c.biete(g, 0xff000001, []string{"text/plain"}, true) })
		c.erwarte(0xff000001, opOfferDestroy, "Auswahl freigeben")
	}()
	g, err := lies(context.Background(), client, func([]string) string { return "" }, 1<<20, 5*time.Second)
	if err != nil || g.Typ != "" || g.Daten != nil || !slices.Equal(g.Typen, []string{"text/plain"}) {
		t.Errorf("Ergebnis %+v, %v", g, err)
	}
	<-fertig
}

// TestLiesLeer: ohne Auswahl (selection mit 0) ErrLeer; ohne data-control ErrKeinDataControl; nur wlr wird benutzt.
func TestLiesLeer(t *testing.T) {
	client, server := paar(t)
	c := &compositor{t: t, v: server}
	fertig := make(chan struct{})
	defer func() { <-fertig }()
	go func() {
		defer close(fertig)
		reg := c.anmeldung([]globalImTest{{3, "wl_seat", 9}, {4, ManagerWlr, 2}})
		c.leseAnmeldung(reg, ManagerWlr, func(g uint32) {
			c.Sende(g, evDeviceSelection, new(wlwire.Nachricht).U32(0))
		})
	}()
	if _, err := lies(context.Background(), client, func([]string) string { return MIME }, 1<<20, 5*time.Second); !errors.Is(err, ErrLeer) {
		t.Errorf("Fehler %v, erwartet ErrLeer", err)
	}

	client2, server2 := paar(t)
	c2 := &compositor{t: t, v: server2} // eigene Variablen: die Goroutine oben liest c noch (Race, CI 6dbfaea)
	fertig2 := make(chan struct{})
	defer func() { <-fertig2 }()
	go func() { defer close(fertig2); c2.anmeldung([]globalImTest{{3, "wl_seat", 9}}) }()
	if _, err := lies(context.Background(), client2, func([]string) string { return MIME }, 1<<20, 5*time.Second); !errors.Is(err, ErrKeinDataControl) {
		t.Errorf("Fehler %v, erwartet ErrKeinDataControl", err)
	}
}

// TestLiesZuGross: mehr als max Byte sind ErrZuGross.
func TestLiesZuGross(t *testing.T) {
	client, server := paar(t)
	c := &compositor{t: t, v: server}
	fertig := make(chan struct{})
	defer func() { <-fertig }()
	go func() {
		defer close(fertig)
		reg := c.anmeldung(normaleGlobals)
		c.leseAnmeldung(reg, ManagerExt, func(g uint32) { c.biete(g, 0xff000001, []string{MIME}, true) })
		c.liefere(0xff000001, MIME, zufall(1000))
	}()
	if _, err := lies(context.Background(), client, func([]string) string { return MIME }, 999, 5*time.Second); !errors.Is(err, ErrZuGross) {
		t.Errorf("Fehler %v, erwartet ErrZuGross", err)
	}
}

// TestLiesStockt: Ein Lieferant, der nichts schreibt und nicht schließt, endet nach der Frist mit Fehler.
func TestLiesStockt(t *testing.T) {
	client, server := paar(t)
	c := &compositor{t: t, v: server}
	halt, fertig := make(chan struct{}), make(chan struct{})
	defer func() { close(halt); <-fertig }() // vor dem Schließen der Verbindung (t.Cleanup) enden
	go func() {
		defer close(fertig)
		reg := c.anmeldung(normaleGlobals)
		c.leseAnmeldung(reg, ManagerExt, func(g uint32) { c.biete(g, 0xff000001, []string{MIME}, true) })
		c.erwarte(0xff000001, opOfferReceive, "receive")
		fd, _ := c.v.NimmFD()
		<-halt // Deskriptor offen halten
		os.NewFile(uintptr(fd), "stockt").Close()
	}()
	_, err := lies(context.Background(), client, func([]string) string { return MIME }, 1<<20, 200*time.Millisecond)
	if err == nil {
		t.Error("kein Fehler bei stockendem Lieferanten")
	}
}

// TestPackeEntpacke: Rundlauf und kaputte Rahmen.
func TestPackeEntpacke(t *testing.T) {
	svg, png := []byte("<svg/>"), []byte{0x89, 'P', 'N', 'G'}
	s, p, err := Entpacke(Packe(svg, png))
	if err != nil || !bytes.Equal(s, svg) || !bytes.Equal(p, png) {
		t.Errorf("Rundlauf: %q %q %v", s, p, err)
	}
	for _, kaputt := range [][]byte{nil, {1, 2, 3}, Packe(nil, png), Packe(svg, nil), append(Packe(svg, png)[:4], 0xff, 0, 0, 0)} {
		if _, _, err := Entpacke(kaputt); !errors.Is(err, ErrRahmen) {
			t.Errorf("Entpacke(%v): %v, erwartet ErrRahmen", kaputt, err)
		}
	}
}

// TestBieteObjekte: ArtObjekte bietet Objekte, PNG und Marke an; jeder Typ bekommt seinen Teil, die Marke nichts.
func TestBieteObjekte(t *testing.T) {
	client, server := paar(t)
	svg, png := []byte("<svg>objekte</svg>"), zufall(70<<10)
	c := &compositor{t: t, v: server}
	fertig := make(chan struct{})
	go func() {
		defer close(fertig)
		reg := c.anmeldung(normaleGlobals)
		i := c.setzen(reg, 3, 5, ManagerExt, []string{MIMEObjekte, MIME, MarkeKlipper}, nil)
		for typ, want := range map[string][]byte{MIMEObjekte: svg, MIME: png, MarkeKlipper: nil, "text/plain": nil} {
			r, w, _ := os.Pipe()
			c.Sende(i.quelle, evSourceSend, new(wlwire.Nachricht).Str(typ).FD(int(w.Fd())))
			w.Close()
			b := make([]byte, 0)
			buf := make([]byte, 4096)
			for {
				n, err := r.Read(buf)
				b = append(b, buf[:n]...)
				if err != nil {
					break
				}
			}
			r.Close()
			if !bytes.Equal(b, want) {
				t.Errorf("%s bekam %d Byte, erwartet %d", typ, len(b), len(want))
			}
		}
		c.Sende(i.quelle, evSourceCancel, nil)
	}()
	h, err := biete(client, Packe(svg, png), ArtObjekte)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.Diene(5 * time.Second); err != nil {
		t.Errorf("Diene: %v", err)
	}
	<-fertig
	if _, err := biete(client, []byte("kein Rahmen"), ArtObjekte); !errors.Is(err, ErrRahmen) {
		t.Errorf("kaputter Rahmen: %v", err)
	}
}

// TestArgumentObjekte: Argument und ArtVon passen zueinander.
func TestArgumentObjekte(t *testing.T) {
	for _, a := range []Art{ArtBild, ArtText, ArtObjekte} {
		if b, ok := ArtVon(a.Argument()); !ok || b != a {
			t.Errorf("ArtVon(%q) = %v, %v", a.Argument(), b, ok)
		}
	}
}

// F-987: Ein tröpfelnder Lieferant endet mit dem Kontext (Gesamtfrist oder Abbruch), nicht erst mit der Frist je Stück.
func TestLiesTroepfeltAbbruch(t *testing.T) {
	client, server := paar(t)
	c := &compositor{t: t, v: server}
	halt, fertig := make(chan struct{}), make(chan struct{})
	defer func() { close(halt); <-fertig }()
	go func() {
		defer close(fertig)
		reg := c.anmeldung(normaleGlobals)
		c.leseAnmeldung(reg, ManagerExt, func(g uint32) { c.biete(g, 0xff000001, []string{MIME}, true) })
		c.erwarte(0xff000001, opOfferReceive, "receive")
		fd, _ := c.v.NimmFD()
		f := os.NewFile(uintptr(fd), "troepfelt")
		defer f.Close()
		for {
			select {
			case <-halt:
				return
			case <-time.After(20 * time.Millisecond):
				f.Write([]byte{1})
			}
		}
	}()
	ctx, ende := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer ende()
	start := time.Now()
	_, err := lies(ctx, client, func([]string) string { return MIME }, 1<<20, 5*time.Second)
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > 3*time.Second {
		t.Errorf("Fehler %v nach %v, erwartet Abbruch durch den Kontext", err, time.Since(start))
	}
}

// F-987: Ein schon abgebrochener Kontext beendet liesBis, auch wenn der Lieferant ohne Pause liefert.
func TestLiesBisAbgebrochen(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	go func() {
		defer w.Close()
		for range 64 {
			if _, err := w.Write(make([]byte, 4096)); err != nil {
				return
			}
		}
	}()
	ctx, ende := context.WithCancel(context.Background())
	ende()
	if _, err := liesBis(ctx, r, 1<<20, 5*time.Second); !errors.Is(err, context.Canceled) {
		t.Errorf("Fehler %v, erwartet Abbruch", err)
	}
}
