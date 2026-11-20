// SPDX-License-Identifier: GPL-3.0-or-later

// Package wlclip hält ein Bild in der Wayland-Zwischenablage, ohne Fenster: über das data-control-Protokoll
// (ext-data-control-v1, sonst wlr-data-control-unstable-v1), das Zwischenablage-Verwalter wie Klipper benutzen. So
// bleibt ein im Editor kopiertes Bild erhalten, nachdem der Editor geschlossen ist (ADR-0024, E-19).
//
// Der Halter spricht das Wayland-Leitungsformat selbst (wire.go), ohne libwayland und ohne cgo. Er bietet genau einen
// MIME-Typ an, liefert die Bytes an jeden, der einfügt, und endet, sobald eine andere Anwendung die Zwischenablage
// übernimmt, der Compositor die Verbindung schließt oder die Zeitgrenze erreicht ist. Er greift in nichts ein.
package wlclip

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"
)

// Objekt-IDs und Opcodes der benutzten Schnittstellen. ext_data_control_v1 und zwlr_data_control_v1 haben dieselben
// Nachrichten in derselben Reihenfolge.
const (
	idDisplay = 1

	opDisplaySync        = 0 // wl_display.sync(new_id callback)
	opDisplayGetRegistry = 1 // wl_display.get_registry(new_id registry)
	evDisplayError       = 0 // wl_display.error(object, code, message)
	evDisplayDeleteID    = 1

	opRegistryBind   = 0 // wl_registry.bind(name, interface, version, new_id)
	evRegistryGlobal = 0 // wl_registry.global(name, interface, version)

	evCallbackDone = 0

	opManagerCreateSource = 0 // create_data_source(new_id)
	opManagerGetDevice    = 1 // get_data_device(new_id, seat)

	opDeviceSetSelection = 0 // set_selection(source)
	evDeviceDataOffer    = 0 // data_offer(new_id offer)
	evDeviceFinished     = 2

	opSourceOffer  = 0 // offer(mime)
	evSourceSend   = 0 // send(mime, fd)
	evSourceCancel = 1 // cancelled
	opOfferDestroy = 1
)

// Namen der Manager-Schnittstellen, bevorzugt in dieser Reihenfolge.
const (
	ManagerExt = "ext_data_control_manager_v1"
	ManagerWlr = "zwlr_data_control_manager_v1"
)

// Fehler, die der Aufrufer unterscheiden kann.
var (
	// ErrKeinDataControl: Der Compositor bietet keines der beiden data-control-Protokolle an.
	ErrKeinDataControl = errors.New("wlclip: Compositor bietet kein data-control an")
	// ErrKeinSitz: Der Compositor nennt keinen wl_seat.
	ErrKeinSitz = errors.New("wlclip: kein wl_seat")
	// ErrAbgeloest: Die Zwischenablage wurde schon beim Setzen von etwas anderem ersetzt.
	ErrAbgeloest = errors.New("wlclip: Zwischenablage sofort ersetzt")
	// ErrZeitgrenze: Die Haltezeit ist abgelaufen; der Halter gibt die Zwischenablage frei.
	ErrZeitgrenze = errors.New("wlclip: Haltezeit abgelaufen")
)

// Stau ist, wie lange das Senden an einen Leser ohne Fortschritt wartet, bevor es aufgibt (wie writeAllFD im
// Gio-Patch, F-176).
const Stau = 10 * time.Second

// Halter bietet einen Inhalt in der Zwischenablage an.
type Halter struct {
	v       *verbindung
	daten   []byte
	mime    string
	quelle  uint32
	geraet  uint32
	Manager string // benutzte Manager-Schnittstelle (ManagerExt oder ManagerWlr)
	senden  sync.WaitGroup
}

// Biete verbindet sich mit dem Compositor der Sitzung und setzt daten mit dem MIME-Typ mime als Zwischenablage. Es
// kehrt zurück, wenn der Compositor das Setzen verarbeitet hat; danach liefert Diene die Bytes aus.
func Biete(daten []byte, mime string) (*Halter, error) {
	v, err := verbinde()
	if err != nil {
		return nil, err
	}
	h, err := biete(v, daten, mime)
	if err != nil {
		v.Close()
		return nil, err
	}
	return h, nil
}

func biete(v *verbindung, daten []byte, mime string) (*Halter, error) {
	registry := v.neueID()
	if err := v.sende(idDisplay, opDisplayGetRegistry, new(nachricht).u32(registry)); err != nil {
		return nil, err
	}
	type global struct {
		name, version uint32
	}
	globals := map[string]global{}
	sitzGefunden := false
	var sitz global
	err := v.rundlauf(func(e ereignis) error {
		if e.obj == registry && e.op == evRegistryGlobal {
			l := leser{b: e.args}
			name, schnitt, version := l.u32(), l.str(), l.u32()
			if l.err != nil {
				return l.err
			}
			if schnitt == "wl_seat" {
				if !sitzGefunden { // erster Sitz; mehrere Sitze kennt KDE nicht
					sitz, sitzGefunden = global{name, version}, true
				}
				return nil
			}
			globals[schnitt] = global{name, version}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	h := &Halter{v: v, daten: daten, mime: mime}
	var mgr global
	for _, kandidat := range []string{ManagerExt, ManagerWlr} {
		if g, ok := globals[kandidat]; ok {
			h.Manager, mgr = kandidat, g
			break
		}
	}
	if h.Manager == "" {
		return nil, ErrKeinDataControl
	}
	if !sitzGefunden {
		return nil, ErrKeinSitz
	}
	sitzID, mgrID := v.neueID(), v.neueID()
	if err := v.sende(registry, opRegistryBind, new(nachricht).u32(sitz.name).str("wl_seat").u32(1).u32(sitzID)); err != nil {
		return nil, err
	}
	if err := v.sende(registry, opRegistryBind, new(nachricht).u32(mgr.name).str(h.Manager).u32(1).u32(mgrID)); err != nil {
		return nil, err
	}
	h.quelle = v.neueID()
	if err := v.sende(mgrID, opManagerCreateSource, new(nachricht).u32(h.quelle)); err != nil {
		return nil, err
	}
	if err := v.sende(h.quelle, opSourceOffer, new(nachricht).str(mime)); err != nil {
		return nil, err
	}
	h.geraet = v.neueID()
	if err := v.sende(mgrID, opManagerGetDevice, new(nachricht).u32(h.geraet).u32(sitzID)); err != nil {
		return nil, err
	}
	if err := v.sende(h.geraet, opDeviceSetSelection, new(nachricht).u32(h.quelle)); err != nil {
		return nil, err
	}
	abgeloest := false
	err = v.rundlauf(func(e ereignis) error {
		ende, err := h.verarbeite(e)
		if ende && err == nil {
			abgeloest = true
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	if abgeloest {
		return nil, ErrAbgeloest
	}
	return h, nil
}

// rundlauf schickt wl_display.sync und verarbeitet Ereignisse mit je, bis die Antwort da ist. Ein wl_display.error
// beendet mit Fehler.
func (v *verbindung) rundlauf(je func(ereignis) error) error {
	cb := v.neueID()
	if err := v.sende(idDisplay, opDisplaySync, new(nachricht).u32(cb)); err != nil {
		return err
	}
	if err := v.c.SetReadDeadline(time.Now().Add(Stau)); err != nil {
		return err
	}
	defer v.c.SetReadDeadline(time.Time{})
	for {
		e, err := v.lies()
		if err != nil {
			return err
		}
		if e.obj == cb && e.op == evCallbackDone {
			return nil
		}
		if err := pruefeDisplay(e); err != nil {
			return err
		}
		if err := je(e); err != nil {
			return err
		}
	}
}

// pruefeDisplay wandelt wl_display.error in einen Fehler.
func pruefeDisplay(e ereignis) error {
	if e.obj != idDisplay || e.op != evDisplayError {
		return nil
	}
	l := leser{b: e.args}
	obj, code, text := l.u32(), l.u32(), l.str()
	return fmt.Errorf("wlclip: Protokollfehler an Objekt %d (Code %d): %s", obj, code, text)
}

// verarbeite behandelt ein Ereignis der eigenen Objekte. ende meldet, dass der Halter fertig ist (abgelöst oder
// Gerät beendet).
func (h *Halter) verarbeite(e ereignis) (ende bool, err error) {
	switch {
	case e.obj == h.quelle && e.op == evSourceSend:
		l := leser{b: e.args}
		mime := l.str()
		if l.err != nil {
			return false, l.err
		}
		fd, err := h.v.nimmFD()
		if err != nil {
			return false, err
		}
		if mime != h.mime {
			syscall.Close(fd)
			return false, nil
		}
		h.senden.Add(1)
		go func() {
			defer h.senden.Done()
			_ = schreibe(fd, h.daten, Stau)
		}()
	case e.obj == h.quelle && e.op == evSourceCancel:
		return true, nil
	case e.obj == h.geraet && e.op == evDeviceDataOffer:
		// Angebote (auch das eigene, als Echo) liest der Halter nicht: gleich wieder freigeben.
		l := leser{b: e.args}
		angebot := l.u32()
		if l.err != nil {
			return false, l.err
		}
		return false, h.v.sende(angebot, opOfferDestroy, nil)
	case e.obj == h.geraet && e.op == evDeviceFinished:
		return true, nil
	}
	return false, nil
}

// Diene liefert die Bytes an jeden aus, der einfügt, bis eine andere Anwendung die Zwischenablage übernimmt (nil),
// der Compositor die Verbindung schließt (ErrVerbindungZu) oder grenze verstrichen ist (ErrZeitgrenze). Danach ist die
// Verbindung geschlossen und jedes begonnene Senden beendet (ein Einfügen, das gerade läuft, bricht nicht ab).
func (h *Halter) Diene(grenze time.Duration) error {
	defer h.senden.Wait()
	defer h.v.Close()
	if err := h.v.c.SetReadDeadline(time.Now().Add(grenze)); err != nil {
		return err
	}
	for {
		e, err := h.v.lies()
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				return ErrZeitgrenze
			}
			return err
		}
		if err := pruefeDisplay(e); err != nil {
			return err
		}
		ende, err := h.verarbeite(e)
		if err != nil {
			return err
		}
		if ende {
			return nil
		}
	}
}

// schreibe schreibt daten ganz in fd und schließt ihn. Stockt der Leser länger als stau ohne Fortschritt, gibt es auf;
// ein geschlossener Leser (EPIPE) beendet still.
func schreibe(fd int, daten []byte, stau time.Duration) error {
	if err := syscall.SetNonblock(fd, true); err != nil {
		syscall.Close(fd)
		return err
	}
	f := os.NewFile(uintptr(fd), "wlclip-senden")
	defer f.Close()
	for rest := daten; len(rest) > 0; {
		stueck := rest[:min(len(rest), 64<<10)]
		if err := f.SetWriteDeadline(time.Now().Add(stau)); err != nil {
			return err
		}
		n, err := f.Write(stueck)
		rest = rest[n:]
		if err != nil {
			return err
		}
	}
	return nil
}
