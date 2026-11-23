// SPDX-License-Identifier: GPL-3.0-or-later

//go:build probe

package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/Andiministrator/schnapp/internal/wlwire"
)

// Schnittstelle org_kde_kwin_fake_input: Opcodes in der Reihenfolge der Anfragen im Protokoll.
const (
	schnittstelle = "org_kde_kwin_fake_input"
	hoechsteVers  = 6

	opAuthenticate    = 0
	opPointerMotion   = 1
	opButton          = 2
	opAxis            = 3
	opPointerAbsolute = 9  // seit 3
	opKeyboardKey     = 10 // seit 4
	opKeysym          = 12 // seit 6
)

// Tasten der Maus (linux/input-event-codes.h) und Zustände.
const (
	btnLinks  = 0x110
	btnRechts = 0x111
	btnMitte  = 0x112
	los       = 0
	gedrueckt = 1
)

// frist ist die Zeitgrenze für Rundläufe zum Compositor.
const frist = 5 * time.Second

// Eingabe speist über org_kde_kwin_fake_input Zeiger- und Tastenereignisse in den (virtuellen) KWin ein.
type Eingabe struct {
	v       *wlwire.Verbindung
	obj     uint32
	Version uint32
}

// VerbindeEingabe verbindet sich mit dem Compositor der Sitzung, bindet die Schnittstelle (höchstens Version 6) und
// meldet sich an. KWin winkt authenticate ohne Prüfung durch — deshalb nur im virtuellen KWin benutzen.
func VerbindeEingabe() (*Eingabe, error) {
	v, err := wlwire.Verbinde()
	if err != nil {
		return nil, err
	}
	reg, globale, err := v.Globale(frist)
	if err != nil {
		v.Close()
		return nil, err
	}
	g, ok := globale[schnittstelle]
	if !ok {
		v.Close()
		return nil, errors.New("der Compositor bietet " + schnittstelle + " nicht an (kein virtueller KWin?)")
	}
	version := min(g.Version, hoechsteVers)
	id, err := v.Bind(reg, g, schnittstelle, version)
	if err != nil {
		v.Close()
		return nil, err
	}
	e := &Eingabe{v: v, obj: id, Version: version}
	if err := v.Sende(id, opAuthenticate, new(wlwire.Nachricht).Str("schnapp-probe").Str("Probelauf")); err != nil {
		v.Close()
		return nil, err
	}
	if err := e.Abgleich(); err != nil {
		v.Close()
		return nil, err
	}
	return e, nil
}

// Close beendet die Verbindung.
func (e *Eingabe) Close() error { return e.v.Close() }

// Abgleich wartet, bis der Compositor alles bisher Gesendete verarbeitet hat (wl_display.sync); so zeigt sich ein
// Protokollfehler am Befehl, der ihn auslöst.
func (e *Eingabe) Abgleich() error {
	return e.v.Rundlauf(frist, func(wlwire.Ereignis) error { return nil })
}

// Zeiger setzt den Zeiger absolut (logische Pixel des virtuellen Bildschirms).
func (e *Eingabe) Zeiger(x, y int) error {
	if e.Version < 3 {
		return fmt.Errorf("pointer_motion_absolute braucht Version 3, der Compositor bietet %d", e.Version)
	}
	return e.v.Sende(e.obj, opPointerAbsolute, new(wlwire.Nachricht).Fixed(float64(x)).Fixed(float64(y)))
}

// Taste drückt (gedrueckt) oder löst (los) eine Maustaste.
func (e *Eingabe) Maus(knopf, zustand uint32) error {
	return e.v.Sende(e.obj, opButton, new(wlwire.Nachricht).U32(knopf).U32(zustand))
}

// Rad dreht das Mausrad um wert (positiv = nach unten, wie libinput: 15 je Raste).
func (e *Eingabe) Rad(wert float64) error {
	return e.v.Sende(e.obj, opAxis, new(wlwire.Nachricht).U32(0).Fixed(wert))
}

// Tastatur drückt oder löst eine Taste mit evdev-Code.
func (e *Eingabe) Tastatur(code, zustand uint32) error {
	if e.Version < 4 {
		return fmt.Errorf("keyboard_key braucht Version 4, der Compositor bietet %d", e.Version)
	}
	return e.v.Sende(e.obj, opKeyboardKey, new(wlwire.Nachricht).U32(code).U32(zustand))
}

// Keysym drückt oder löst ein X11-Keysym.
func (e *Eingabe) Keysym(sym, zustand uint32) error {
	return e.v.Sende(e.obj, opKeysym, new(wlwire.Nachricht).U32(sym).U32(zustand))
}

// Fassung liefert die gebundene Version der Schnittstelle.
func (e *Eingabe) Fassung() uint32 { return e.Version }
