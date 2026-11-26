// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Kontextmenü der Leinwand (ADR-0038 Teil 1): Rechtsklick wählt das Objekt unter dem Zeiger (canvas.KontextBei) und
// öffnet die Liste listeKontext am Klickpunkt; Umschalt+F10 öffnet sie für die Auswahl. Die Liste läuft über die
// Menü-Mechanik (menue.go): Tastatur, Klick daneben und Sperre beim Speichern wie bei jedem Menü.

import (
	"fmt"
	"image"
	"math"

	"gioui.org/layout"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/settings"
)

// kontextMenue ist der Zustand des Kontextmenüs.
type kontextMenue struct {
	anker  image.Rectangle // Fensterpunkt des Aufrufs als Rechteck 1×1 (listenAnker)
	objekt bool            // die Liste gilt dem gewählten Objekt, sonst dem Bild
}

// kontextAmPunkt öffnet das Kontextmenü am Fensterpunkt p (Leinwandkoordinaten plus Ursprung der Leinwand).
func (e *editor) kontextAmPunkt(p image.Point, objekt, perTaste bool) {
	e.kontext = kontextMenue{anker: image.Rectangle{Min: p, Max: p.Add(image.Pt(1, 1))}, objekt: objekt}
	e.menue.offen = false // ein offenes Menü schließt; menueOeffne öffnet sonst nicht neu
	e.menueOeffne(listeKontext, perTaste)
}

// kontextRechtsklick verarbeitet einen Rechtsklick auf die Leinwand am Leinwandpunkt p.
func (e *editor) kontextRechtsklick(p image.Point) {
	offen, objekt := e.s.KontextBei(document.Pt(float64(p.X), float64(p.Y)))
	if offen {
		e.kontextAmPunkt(p.Add(e.leinwandUrsprung), objekt, false)
	}
}

// kontextPerTaste öffnet das Kontextmenü per Umschalt+F10 (ADR-0038 Punkt 6): an der Mitte der Auswahl, sonst der
// Leinwand. Während eines Zugs oder des Speicherns nicht.
func (e *editor) kontextPerTaste() {
	if e.s.Zieht() || e.s.SpeichernLaeuft() || e.s.Ruht() {
		return
	}
	e.s.TextBestaetigen()
	q, objekt := e.s.KontextAnker()
	p := image.Pt(int(math.Round(q.X)), int(math.Round(q.Y)))
	e.kontextAmPunkt(p.Add(e.leinwandUrsprung), objekt, true)
}

// kontextEintraege sind die Einträge der Liste (ADR-0038 Punkt 2), je nachdem, ob sie einem Objekt gilt.
func (e *editor) kontextEintraege() []menueEintrag {
	s := e.s
	if !e.kontext.objekt || s.Auswahl == 0 {
		en := []menueEintrag{
			{akt: aktKopieren, text: i18n.T("Copy"), an: true},
			{akt: aktExport, text: i18n.T("Export…"), an: !e.exportOffen},
			{akt: aktTextKopieren, text: i18n.T("Copy text from image"), an: e.ocrGrau() == "", erklaerung: e.ocrGrau()},
			{akt: aktEinpassen, text: i18n.T("Fit"), an: true, gruppe: true},
			{akt: aktEins, text: "1:1", an: true},
		}
		if s.KannGanzesBild() {
			en = append(en, menueEintrag{akt: aktGanzesBild, text: i18n.T("Whole image"), an: true, gruppe: true})
		}
		return en
	}
	eb, _ := s.GewaehlteEbene()
	an := s.KannEbene()
	aus, sperr := i18n.T("Hide layer"), i18n.T("Lock layer")
	if eb.Ausgeblendet {
		aus = i18n.T("Show layer")
	}
	if eb.Gesperrt {
		sperr = i18n.T("Unlock layer")
	}
	var en []menueEintrag
	if s.KannTextBearbeiten() {
		en = append(en, menueEintrag{akt: aktTextBearbeiten, text: i18n.T("Edit text"), an: true})
	}
	en = append(en,
		menueEintrag{akt: aktDoppeln, text: i18n.T("Duplicate"), an: an},
		menueEintrag{akt: aktLoeschen, text: i18n.T("Delete"), an: s.KannLoeschen()},
		menueEintrag{akt: aktEbeneVor, text: i18n.T("Bring forward"), an: s.KannEbeneSchritt(1), gruppe: true},
		menueEintrag{akt: aktEbeneZurueck, text: i18n.T("Send backward"), an: s.KannEbeneSchritt(-1)},
		menueEintrag{akt: aktAusblenden, text: aus, an: an, gruppe: true},
		menueEintrag{akt: aktSperren, text: sperr, an: an},
		menueEintrag{akt: aktUmbenennen, text: i18n.T("Rename…"), an: an},
	)
	grau := e.schnellstilSichernGrau()
	en = append(en, menueEintrag{akt: aktSchnellstilSichern, text: i18n.T("Save style as quick style"), an: grau == "",
		erklaerung: grau, gruppe: true})
	if s.KannTauschen() {
		en = append(en, menueEintrag{akt: aktFarbenTauschen, text: i18n.T("Swap colors"), an: true})
	}
	return en
}

// schnellstilSichernGrau sagt, warum „Als Schnellstil sichern“ nicht geht (leer: es geht), wie der Hinweis am „+“ der
// Schnellstile (ADR-0037).
func (e *editor) schnellstilSichernGrau() string {
	art, ok := e.s.SchnellArt()
	if !ok {
		return i18n.T("This object has no quick styles.")
	}
	switch e.s.PruefeSchnellstilHinzu(art) {
	case canvas.HinzuOK:
		return ""
	case canvas.HinzuUnbrauchbar:
		return i18n.T("The current settings cannot be saved as a style (stroke width, font name).")
	case canvas.HinzuVoll:
		return fmt.Sprintf(i18n.T("At most %d custom styles per kind."), canvas.MaxEigeneSchnellstile)
	case canvas.HinzuDoppelt:
		return i18n.T("The current settings are already a style.")
	}
	return i18n.T("This object has no quick styles.")
}

// schnellstilSichern legt den Stand der Art des gewählten Objekts bzw. Werkzeugs als eigenen Schnellstil an und
// schreibt styles.ini (wie das „+“ der Schnellstile, ADR-0037 Punkt 5).
func (e *editor) schnellstilSichern(art canvas.SchnellArt) {
	if st, r := e.s.FuegeSchnellstilHinzu(art); r == canvas.HinzuOK {
		neu := st.Einstellung()
		e.stileSpeichern(i18n.T("Style saved"), func(alt []settings.Stil) []settings.Stil { return settings.StilHinzu(alt, neu) })
	}
}

// schnellstilSichernAktion ist die Wirkung des Menüeintrags.
func (e *editor) schnellstilSichernAktion(layout.Context) {
	if art, ok := e.s.SchnellArt(); ok && !e.s.SpeichernLaeuft() {
		e.schnellstilSichern(art)
	}
}
