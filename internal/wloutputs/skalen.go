// SPDX-License-Identifier: GPL-3.0-or-later

// Package wloutputs liest die Skala jedes Monitors (auch gebrochen, etwa 1,25) aus dem Compositor: wl_output meldet
// den Modus in physischen Pixeln, zxdg_output_v1 die logische Größe; die Skala ist der Quotient (ADR-0017
// Nachtrag 9, F-668). Die ganzzahlige Pufferskala von wl_output führt bei gebrochener Skalierung in die Irre (F-57).
// Reines Go über internal/wlwire, eigene kurze Verbindung je Aufruf.
package wloutputs

import (
	"errors"
	"math"
	"time"

	"github.com/Andiministrator/schnapp/internal/wlwire"
)

// Opcodes und Ereignisse von wl_output, zxdg_output_manager_v1 und zxdg_output_v1.
const (
	evOutputMode = 1 // wl_output.mode(flags, width, height, refresh)
	evOutputName = 4 // wl_output.name(name), ab Version 4

	opManagerGetXdgOutput = 1 // zxdg_output_manager_v1.get_xdg_output(new_id, output)

	evXdgLogicalSize = 1 // zxdg_output_v1.logical_size(width, height)
	evXdgName        = 3 // zxdg_output_v1.name(name), ab Version 2

	modusAktuell = 1 // wl_output.mode flags: current
)

// Frist ist die längste Wartezeit je Rundlauf mit dem Compositor.
const Frist = 2 * time.Second

// Grenzen gegen einen fehlerhaften Compositor (Kritik C-2, 24.11.2026): höchstens so viele Globale und Ausgaben; darüber
// bricht die Abfrage ab (die Skala fehlt dann, die Monitorliste bleibt).
const (
	maxGlobale  = 4096
	maxAusgaben = 64
)

// ErrZuViele meldet mehr Globale oder Ausgaben, als die Abfrage annimmt.
var ErrZuViele = errors.New("wloutputs: zu viele Globale oder Ausgaben")

// ErrOhneXdgOutput meldet einen Compositor ohne zxdg_output_manager_v1: die logische Größe ist unbekannt.
var ErrOhneXdgOutput = errors.New("wloutputs: zxdg_output_manager_v1 fehlt")

// Skalen verbindet sich mit dem Compositor der Sitzung und liefert je Monitorname die Skala.
func Skalen() (map[string]float64, error) {
	v, err := wlwire.Verbinde()
	if err != nil {
		return nil, err
	}
	defer v.Close()
	return lies(v, Frist)
}

type ausgabe struct {
	id, xdg             uint32
	name, xdgName       string
	breite, hoehe       int32 // aktueller Modus, physische Pixel
	logBreite, logHoehe int32
}

type global struct {
	name, version uint32
	schnitt       string
}

// lies fragt über v ab (im Test ein nachgebauter Compositor).
func lies(v *wlwire.Verbindung, frist time.Duration) (map[string]float64, error) {
	// Schreiben hat sonst keine Frist: Liest der Compositor nicht mehr, blockierte Sende bei vollem Puffer für immer.
	// Zwei Rundläufe plus Schreiben bleiben so in drei Fristen (C-2).
	if err := v.SetWriteDeadline(time.Now().Add(3 * frist)); err != nil {
		return nil, err
	}
	defer v.SetWriteDeadline(time.Time{})
	registry := v.NeueID()
	if err := v.Sende(wlwire.IDDisplay, wlwire.OpDisplayGetRegistry, new(wlwire.Nachricht).U32(registry)); err != nil {
		return nil, err
	}
	var globale []global
	err := v.Rundlauf(frist, func(e wlwire.Ereignis) error {
		if e.Obj != registry || e.Op != wlwire.EvRegistryGlobal {
			return nil
		}
		l := wlwire.Leser{B: e.Args}
		g := global{name: l.U32(), schnitt: l.Str(), version: l.U32()}
		if l.Err != nil {
			return l.Err
		}
		if len(globale) >= maxGlobale || (g.schnitt == "wl_output" && zaehle(globale, "wl_output") >= maxAusgaben) {
			return ErrZuViele
		}
		globale = append(globale, g)
		return nil
	})
	if err != nil {
		return nil, err
	}
	var manager *global
	var ausgaben []*ausgabe
	for i := range globale {
		switch globale[i].schnitt {
		case "zxdg_output_manager_v1":
			if manager == nil {
				manager = &globale[i]
			}
		case "wl_output":
			ausgaben = append(ausgaben, &ausgabe{})
		}
	}
	if manager == nil {
		return nil, ErrOhneXdgOutput
	}
	mid, err := v.Bind(registry, wlwire.Global{Name: manager.name, Version: manager.version}, manager.schnitt, min(manager.version, 3))
	if err != nil {
		return nil, err
	}
	nr := 0
	for _, g := range globale {
		if g.schnitt != "wl_output" {
			continue
		}
		a := ausgaben[nr]
		nr++
		if a.id, err = v.Bind(registry, wlwire.Global{Name: g.name, Version: g.version}, g.schnitt, min(g.version, 4)); err != nil {
			return nil, err
		}
		a.xdg = v.NeueID()
		if err := v.Sende(mid, opManagerGetXdgOutput, new(wlwire.Nachricht).U32(a.xdg).U32(a.id)); err != nil {
			return nil, err
		}
	}
	err = v.Rundlauf(frist, func(e wlwire.Ereignis) error {
		for _, a := range ausgaben {
			l := wlwire.Leser{B: e.Args}
			switch {
			case e.Obj == a.id && e.Op == evOutputMode:
				flags, b, h := l.U32(), int32(l.U32()), int32(l.U32())
				if l.Err == nil && flags&modusAktuell != 0 {
					a.breite, a.hoehe = b, h
				}
			case e.Obj == a.id && e.Op == evOutputName:
				if s := l.Str(); l.Err == nil {
					a.name = s
				}
			case e.Obj == a.xdg && e.Op == evXdgLogicalSize:
				b, h := int32(l.U32()), int32(l.U32())
				if l.Err == nil {
					a.logBreite, a.logHoehe = b, h
				}
			case e.Obj == a.xdg && e.Op == evXdgName:
				if s := l.Str(); l.Err == nil {
					a.xdgName = s
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	name := func(a *ausgabe) string {
		if a.name != "" {
			return a.name
		}
		return a.xdgName
	}
	anzahl := map[string]int{}
	for _, a := range ausgaben {
		anzahl[name(a)]++
	}
	res := map[string]float64{}
	for _, a := range ausgaben {
		n := name(a)
		// Ohne Namen nicht zuzuordnen; zwei Ausgaben gleichen Namens: unklar, welche gemeint ist — keine Skala.
		if n == "" || anzahl[n] > 1 {
			continue
		}
		if s, ok := Skala(a.breite, a.hoehe, a.logBreite, a.logHoehe); ok {
			res[n] = s
		}
	}
	return res, nil
}

// Skala ist die längere physische Seite durch die längere logische Seite, auf Hundertstel gerundet. Der Vergleich der
// längeren Seiten macht die Drehung gleichgültig: wl_output meldet den Modus ungedreht, xdg_output die logische Größe
// gedreht. Unbekannte oder leere Größen liefern ok = false.
func Skala(breite, hoehe, logBreite, logHoehe int32) (float64, bool) {
	phys, logi := max(breite, hoehe), max(logBreite, logHoehe)
	if phys <= 0 || logi <= 0 {
		return 0, false
	}
	return math.Round(float64(phys)/float64(logi)*100) / 100, true
}

// Prozent ist die Skala als ganze Prozent (1,25 → 125), wie die Systemeinstellungen sie zeigen.
func Prozent(s float64) int { return int(math.Round(s * 100)) }

// zaehle ist die Zahl der Globale mit der Schnittstelle schnitt.
func zaehle(gs []global, schnitt string) int {
	n := 0
	for _, g := range gs {
		if g.schnitt == schnitt {
			n++
		}
	}
	return n
}
