// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"fmt"
	"image"
	"io"
	"os"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	kwinDienst   = "org.kde.KWin"
	kwinObjekt   = "/org/kde/KWin/ScreenShot2"
	kwinSchnitts = "org.kde.KWin.ScreenShot2"
)

// aufnahme ist ein Bild von KWin samt den Angaben aus der Antwort.
type aufnahme struct {
	bild   *image.RGBA
	skala  float64 // devicePixelRatio laut KWin (ab Version 4), sonst 0
	format uint32  // QImage::Format der Rohdaten
	dauer  time.Duration
}

// kwinVersion liest die Eigenschaft Version der Schnittstelle.
func kwinVersion() (uint32, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return 0, err
	}
	v, err := conn.Object(kwinDienst, kwinObjekt).GetProperty(kwinSchnitts + ".Version")
	if err != nil {
		return 0, err
	}
	n, ok := v.Value().(uint32)
	if !ok {
		return 0, fmt.Errorf("Version hat Typ %T", v.Value())
	}
	return n, nil
}

// aufnehmen ruft eine Capture-Methode von ScreenShot2 auf (z. B. "CaptureWorkspace") und liest die Pixel
// aus der Pipe. args sind die Parameter vor options und pipe.
func aufnehmen(methode string, args ...interface{}) (*aufnahme, error) {
	start := time.Now()
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, err
	}
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	type gelesen struct {
		daten []byte
		err   error
	}
	fertig := make(chan gelesen, 1)
	go func() { // KWin schreibt erst nach der Antwort; parallel lesen, sonst blockiert die Pipe
		d, err := io.ReadAll(r)
		r.Close()
		fertig <- gelesen{d, err}
	}()
	optionen := map[string]dbus.Variant{
		"native-resolution":   dbus.MakeVariant(true),
		"include-cursor":      dbus.MakeVariant(false),
		"hide-caller-windows": dbus.MakeVariant(true),
	}
	aufruf := append(append([]interface{}{}, args...), optionen, dbus.UnixFD(w.Fd()))
	var antwort map[string]dbus.Variant
	err = conn.Object(kwinDienst, kwinObjekt).Call(kwinSchnitts+"."+methode, 0, aufruf...).Store(&antwort)
	w.Close() // unsere Kopie schließen, damit das Lesen mit KWins Ende EOF bekommt
	g := <-fertig
	if err != nil {
		return nil, fmt.Errorf("%s: %w", methode, err)
	}
	if g.err != nil {
		return nil, fmt.Errorf("Pipe lesen: %w", g.err)
	}
	if typ, _ := antwort["type"].Value().(string); typ != "raw" {
		return nil, fmt.Errorf("unerwarteter Ergebnistyp %q", typ)
	}
	breite, e1 := ganzzahl(antwort, "width")
	hoehe, e2 := ganzzahl(antwort, "height")
	zeile, e3 := ganzzahl(antwort, "stride")
	format, e4 := ganzzahl(antwort, "format")
	for _, e := range []error{e1, e2, e3, e4} {
		if e != nil {
			return nil, e
		}
	}
	bild, err := rohZuRGBA(g.daten, int(breite), int(hoehe), int(zeile), uint32(format))
	if err != nil {
		return nil, err
	}
	a := &aufnahme{bild: bild, format: uint32(format), dauer: time.Since(start)}
	if s, ok := antwort["scale"].Value().(float64); ok {
		a.skala = s
	}
	return a, nil
}

func ganzzahl(m map[string]dbus.Variant, schluessel string) (uint64, error) {
	v, ok := m[schluessel]
	if !ok {
		return 0, fmt.Errorf("Antwort ohne %q", schluessel)
	}
	switch n := v.Value().(type) {
	case uint32:
		return uint64(n), nil
	case int32:
		return uint64(n), nil
	case uint64:
		return n, nil
	case int64:
		return uint64(n), nil
	}
	return 0, fmt.Errorf("%q hat Typ %T", schluessel, v.Value())
}
