// SPDX-License-Identifier: GPL-3.0-or-later

// Package wlclip hält ein Bild in der Wayland-Zwischenablage, ohne Fenster: über das data-control-Protokoll
// (ext-data-control-v1, sonst wlr-data-control-unstable-v1), das Zwischenablage-Verwalter wie Klipper benutzen. So
// bleibt ein im Editor kopiertes Bild erhalten, nachdem der Editor geschlossen ist (ADR-0024, E-19).
//
// Der Halter spricht das Wayland-Leitungsformat selbst (Paket internal/wlwire), ohne libwayland und ohne cgo. Er bietet die Typen
// seiner Art an (ArtBild: image/png und zusätzlich den leeren MarkeKlipper; ArtText: die üblichen Text-Typen, ADR-0024
// Nachtrag 9), liefert die Bytes an jeden, der einfügt, und endet, sobald eine andere Anwendung die Zwischenablage
// übernimmt, der Compositor die Verbindung schließt oder die Zeitgrenze erreicht ist. Er greift in nichts ein.
package wlclip

import (
	"errors"
	"os"
	"slices"
	"sync"
	"syscall"
	"time"

	"github.com/Andiministrator/schnapp/internal/wlwire"
)

// Objekt-IDs und Opcodes der benutzten Schnittstellen. ext_data_control_v1 und zwlr_data_control_v1 haben dieselben
// Nachrichten in derselben Reihenfolge.
const (
	idDisplay            = wlwire.IDDisplay
	opDisplaySync        = wlwire.OpDisplaySync
	opDisplayGetRegistry = wlwire.OpDisplayGetRegistry
	evDisplayError       = wlwire.EvDisplayError

	opRegistryBind   = wlwire.OpRegistryBind
	evRegistryGlobal = wlwire.EvRegistryGlobal

	evCallbackDone = wlwire.EvCallbackDone

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
	// ErrVerbindungZu: Der Compositor hat die Verbindung geschlossen.
	ErrVerbindungZu = wlwire.ErrVerbindungZu
	// ErrKeinDataControl: Der Compositor bietet keines der beiden data-control-Protokolle an.
	ErrKeinDataControl = errors.New("wlclip: Compositor bietet kein data-control an")
	// ErrKeinSitz: Der Compositor nennt keinen wl_seat.
	ErrKeinSitz = errors.New("wlclip: kein wl_seat")
	// ErrAbgeloest: Die Zwischenablage wurde schon beim Setzen von etwas anderem ersetzt.
	ErrAbgeloest = errors.New("wlclip: Zwischenablage sofort ersetzt")
	// ErrZeitgrenze: Die Haltezeit ist abgelaufen; der Halter gibt die Zwischenablage frei.
	ErrZeitgrenze = errors.New("wlclip: Haltezeit abgelaufen")
)

// MarkeKlipper ist der leere MIME-Typ, mit dem Klipper ein Bild auch dann in den Verlauf (Meta+V) übernimmt, wenn
// „Image data copied explicitly“ (SaveImages, Voreinstellung aus) nicht eingeschaltet ist — wie Spectacle (ADR-0024
// Nachtrag 2). Der Halter bietet ihn zusätzlich an und liefert dafür null Byte.
const MarkeKlipper = "x-kde-force-image-copy"

// Stau ist, wie lange das Senden an einen Leser ohne Fortschritt wartet, bevor es aufgibt (wie writeAllFD im
// Gio-Patch, F-176).
const Stau = 10 * time.Second

// scheibe ist, wie oft Diene die Wanduhr gegen die Grenze prüft (im Test kürzer).
var scheibe = time.Minute

// jetzt ist die Uhr, nach der Diene die Grenze misst (im Test einspeisbar, um einen Standby-Sprung nachzustellen).
var jetzt = time.Now

// monoSeit ist die auf der monotonen Uhr vergangene Zeit seit einem mit jetzt genommenen Zeitpunkt (F-232). Die
// monotone Uhr ist von einer zurückgestellten Wanduhr unberührt; im Test einspeisbar.
var monoSeit = time.Since

// nachLies wird im Test zwischen Lesen und Verarbeiten eines Ereignisses gerufen (in Produktion nil).
var nachLies func()

// Art ist, was der Halter anbietet: ein Bild (Strg+C im Editor) oder Text (Text aus dem Bild, ADR-0035, ADR-0024
// Nachtrag 9).
type Art int

const (
	ArtBild Art = iota
	ArtText
)

// TextTypen sind die MIME-Typen, unter denen der Halter Text anbietet, in dieser Reihenfolge: UTF-8 ausdrücklich, dann
// die Namen, die ältere und X11-nahe Programme erfragen (Xwayland, Qt, GTK).
var TextTypen = []string{"text/plain;charset=utf-8", "text/plain", "UTF8_STRING"}

// typen sind die Typen, für die der Halter die Bytes liefert; marken die zusätzlich angebotenen leeren Typen.
func (a Art) typen() []string {
	if a == ArtText {
		return TextTypen
	}
	return []string{MIME}
}

func (a Art) marken() []string {
	if a == ArtText {
		return nil // die Klipper-Marke gilt nur Bildern (Nachtrag 2)
	}
	return []string{MarkeKlipper}
}

// angebot sind alle angebotenen Typen in der Reihenfolge der offer-Anfragen.
func (a Art) angebot() []string { return append(append([]string(nil), a.typen()...), a.marken()...) }

// Halter bietet einen Inhalt in der Zwischenablage an.
type Halter struct {
	v       *wlwire.Verbindung
	daten   []byte
	typen   []string // Typen, die die Bytes bekommen
	quelle  uint32
	geraet  uint32
	Manager string // benutzte Manager-Schnittstelle (ManagerExt oder ManagerWlr)
	senden  sync.WaitGroup
}

// Biete verbindet sich mit dem Compositor der Sitzung und setzt daten mit den Typen der Art a als Zwischenablage. Es
// kehrt zurück, wenn der Compositor das Setzen verarbeitet hat; danach liefert Diene die Bytes aus.
func Biete(daten []byte, a Art) (*Halter, error) {
	v, err := wlwire.Verbinde()
	if err != nil {
		return nil, err
	}
	h, err := biete(v, daten, a)
	if err != nil {
		v.Close()
		return nil, err
	}
	return h, nil
}

func biete(v *wlwire.Verbindung, daten []byte, a Art) (*Halter, error) {
	registry := v.NeueID()
	if err := v.Sende(idDisplay, opDisplayGetRegistry, new(wlwire.Nachricht).U32(registry)); err != nil {
		return nil, err
	}
	type global struct {
		name, version uint32
	}
	globals := map[string]global{}
	sitzGefunden := false
	var sitz global
	err := v.Rundlauf(Stau, func(e wlwire.Ereignis) error {
		if e.Obj == registry && e.Op == evRegistryGlobal {
			l := wlwire.Leser{B: e.Args}
			name, schnitt, version := l.U32(), l.Str(), l.U32()
			if l.Err != nil {
				return l.Err
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
	h := &Halter{v: v, daten: daten, typen: a.typen()}
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
	sitzID, mgrID := v.NeueID(), v.NeueID()
	if err := v.Sende(registry, opRegistryBind, new(wlwire.Nachricht).U32(sitz.name).Str("wl_seat").U32(1).U32(sitzID)); err != nil {
		return nil, err
	}
	if err := v.Sende(registry, opRegistryBind, new(wlwire.Nachricht).U32(mgr.name).Str(h.Manager).U32(1).U32(mgrID)); err != nil {
		return nil, err
	}
	h.quelle = v.NeueID()
	if err := v.Sende(mgrID, opManagerCreateSource, new(wlwire.Nachricht).U32(h.quelle)); err != nil {
		return nil, err
	}
	for _, m := range a.angebot() {
		if err := v.Sende(h.quelle, opSourceOffer, new(wlwire.Nachricht).Str(m)); err != nil {
			return nil, err
		}
	}
	h.geraet = v.NeueID()
	if err := v.Sende(mgrID, opManagerGetDevice, new(wlwire.Nachricht).U32(h.geraet).U32(sitzID)); err != nil {
		return nil, err
	}
	if err := v.Sende(h.geraet, opDeviceSetSelection, new(wlwire.Nachricht).U32(h.quelle)); err != nil {
		return nil, err
	}
	abgeloest := false
	err = v.Rundlauf(Stau, func(e wlwire.Ereignis) error {
		fertig, err := h.verarbeite(e)
		if fertig && err == nil {
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

// verarbeite behandelt ein Ereignis der eigenen Objekte. ende meldet, dass der Halter fertig ist (abgelöst oder
// Gerät beendet).
func (h *Halter) verarbeite(e wlwire.Ereignis) (ende bool, err error) {
	switch {
	case e.Obj == h.quelle && e.Op == evSourceSend:
		l := wlwire.Leser{B: e.Args}
		mime := l.Str()
		if l.Err != nil {
			return false, l.Err
		}
		fd, err := h.v.NimmFD()
		if err != nil {
			return false, err
		}
		if !slices.Contains(h.typen, mime) { // auch MarkeKlipper: leer, der Leser sieht sofort das Ende
			syscall.Close(fd)
			return false, nil
		}
		h.senden.Add(1)
		go func() {
			defer h.senden.Done()
			_ = schreibe(fd, h.daten, Stau)
		}()
	case e.Obj == h.quelle && e.Op == evSourceCancel:
		return true, nil
	case e.Obj == h.geraet && e.Op == evDeviceDataOffer:
		// Angebote (auch das eigene, als Echo) liest der Halter nicht: gleich wieder freigeben.
		l := wlwire.Leser{B: e.Args}
		angebot := l.U32()
		if l.Err != nil {
			return false, l.Err
		}
		return false, h.v.Sende(angebot, opOfferDestroy, nil)
	case e.Obj == h.geraet && e.Op == evDeviceFinished:
		return true, nil
	}
	return false, nil
}

// Diene liefert die Bytes an jeden aus, der einfügt, bis eine andere Anwendung die Zwischenablage übernimmt (nil),
// der Compositor die Verbindung schließt (ErrVerbindungZu) oder grenze verstrichen ist (ErrZeitgrenze). Danach ist die
// Verbindung geschlossen und jedes begonnene Senden beendet (ein Einfügen, das gerade läuft, bricht nicht ab; Diene
// wartet darauf aber höchstens Stau).
//
// Die Grenze gilt nach der Wanduhr: Fristen in Go laufen auf der monotonen Uhr, die im Standby steht. Diene setzt die
// Lesefrist deshalb in Scheiben neu und vergleicht mit der Wanduhr (Kritik Fable, Befund 2). Eine zurückgestellte
// Wanduhr würde die Grenze dafür verlängern; deshalb gilt die frühere von Wanduhr- und monotoner Grenze (F-232). Die Scheibe ist nur eine
// Lesefrist; für das Schreiben an den Compositor gilt je Ereignis eine eigene Frist von Stau, damit ein hängender
// Compositor den Halter nicht festhält, eine abgelaufene Scheibe aber auch kein gesundes Antworten verhindert.
func (h *Halter) Diene(grenze time.Duration) error {
	defer warteSenden(&h.senden, Stau)
	defer h.v.Close()
	start := jetzt()
	grenzzeit := start.Round(0).Add(grenze) // Round(0): ohne monotonen Anteil, Vergleich nach der Wanduhr
	for {
		// die frühere Grenze zählt: Wanduhr (Standby springt vor) oder monotone Uhr (Rückstellen der Wanduhr, F-232)
		rest := min(grenzzeit.Sub(jetzt().Round(0)), grenze-monoSeit(start))
		if rest <= 0 {
			return ErrZeitgrenze
		}
		if err := h.v.SetReadDeadline(time.Now().Add(min(rest, scheibe))); err != nil {
			return err
		}
		e, err := h.v.Lies()
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				continue // Scheibe abgelaufen; Grenze oben prüfen
			}
			return err
		}
		if err := wlwire.PruefeDisplay(e); err != nil {
			return err
		}
		if nachLies != nil {
			nachLies()
		}
		if err := h.v.SetWriteDeadline(time.Now().Add(Stau)); err != nil {
			return err
		}
		fertig, err := h.verarbeite(e)
		if err != nil {
			return err
		}
		if fertig {
			return nil
		}
	}
}

// warteSenden wartet auf laufende Sendungen, höchstens max. Die Frist je 64-KiB-Stück in schreibe gilt nicht für die
// Gesamtdauer; ein Leser, der Stück für Stück weiterliest, hielte sonst den Halter nach seinem Ende fest. Der Prozess
// endet danach ohnehin.
func warteSenden(wg *sync.WaitGroup, max time.Duration) {
	fertig := make(chan struct{})
	go func() {
		wg.Wait()
		close(fertig)
	}()
	t := time.NewTimer(max)
	defer t.Stop()
	select {
	case <-fertig:
	case <-t.C:
	}
}

// schreibe schreibt daten ganz in fd und schließt ihn. Stockt der Leser bei einem 64-KiB-Stück länger als stau, gibt es
// auf (die Frist gilt je Stück, nicht für das Ganze); ein geschlossener Leser (EPIPE) beendet still.
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
