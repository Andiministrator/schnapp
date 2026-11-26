// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"syscall"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/render"
	"github.com/Andiministrator/schnapp/internal/rescue"
	"github.com/Andiministrator/schnapp/internal/saveprogress"
	"github.com/Andiministrator/schnapp/internal/ui"
)

// editorStarten öffnet das Editorfenster. Eine Variable, damit Tests das Fenster durch eine Attrappe ersetzen und die
// Speicherfunktion prüfen können, die oeffneImEditor und bearbeiteNeu dem Editor übergeben (F-282).
var editorStarten = ui.EditorMit

// oeffneImEditor öffnet eine Datei im Editor (ADR-0020). Speichern schreibt eine Aufnahme der Bibliothek an ihren
// Platz zurück; alles andere (Datei von außen, fremdes oder beschädigtes Bild, neuere Fassung) kommt beim ersten
// Speichern als neue Aufnahme in die Bibliothek, danach wird diese fortgeschrieben. Eine Datei von außen speichert der
// Editor schon bei der ersten Änderung; wurde ihr Inhalt früher übernommen, öffnet sich gleich diese Aufnahme (ADR-0016
// Nachtrag 11). Die geöffnete Datei selbst bleibt unberührt.
//
// ohneGrenze ist --ignore-limit (ADR-0017 Nachtrag 6): Die Datei wird auch über dem Rechenbudget geöffnet. kante ist
// [capture] seam (ADR-0023 Nachtrag 6): Ein einfaches PNG/JPEG (auch eine beschädigte oder neuere Arbeitsdatei, die als
// Bild geöffnet wird) ist ein neues Dokument und bekommt diese Form; eine Arbeitsdatei behält ihre eigene.
func oeffneImEditor(pfad string, ohneGrenze bool, kante document.Kante) error {
	orte, err := library.OrteAus(os.Getenv)
	if err != nil {
		return err
	}
	b := library.Neu(orte)
	// Eine Datei von außen: früher übernommene Aufnahme wiederfinden (ADR-0016 Nachtrag 11).
	pfad, aussen := vorOeffnen(b, pfad)
	// Erst auf ein laufendes UmDatei warten, dann laden, dann die Rettung (F-769, F-770).
	abs, g, stand, r, err := ladeMitRettung(pfad, ohneGrenze, ladeFuerEditor)
	if err != nil {
		return err
	}
	if g.Fremd {
		if err := g.Dokument.VorgabeKante(kante); err != nil {
			r.ende()
			return err
		}
	}
	ui.SetzeExportOhneGrenze(ohneGrenze) // auch der Export aus dem Editor prüft das Budget dann nicht (ADR-0033 Nachtrag 4)
	return editorStarten(geoeffnetFuer(b, abs, g, stand, r, ohneGrenze, aussen), anbindungFuer(b, kante))
}

// geoeffnetFuer stellt zusammen, was der Editor für ein Dokument braucht: die Speicherfunktion, die Rettungskopie und
// den Pfad der Datei, die den Stand trägt (nach einem Speichern als neue Aufnahme die neue; die Leiste der letzten
// Aufnahmen umrandet sie, ADR-0036 Punkt 1). aussen: eine Datei von außen, die der Editor bei der ersten Änderung
// speichert (vorOeffnen, ADR-0016 Nachtrag 11).
func geoeffnetFuer(b *library.Bibliothek, abs string, g *fileformat.Gelesen, stand library.Stand, r *rettung, ohneGrenze, aussen bool) ui.Geoeffnet {
	var aktuell atomic.Pointer[string] // das Speichern läuft in einer Goroutine (ADR-0020 Nachtrag 6)
	aktuell.Store(&abs)
	var aktStand atomic.Pointer[library.Stand] // Stand der Datei nach Laden bzw. letztem eigenen Schreiben (F-834)
	aktStand.Store(&stand)
	rett := r.nachSpeichern()
	nachher := func(p string, st library.Stand) {
		aktuell.Store(&p)
		aktStand.Store(&st)
		if rett != nil {
			rett(p)
		}
	}
	return ui.Geoeffnet{Titel: filepath.Base(abs), Dokument: g.Dokument, Speichern: speichernMitNachher(b, abs, g, stand, nachher),
		Rettung: r.ui(), Pfad: func() string { return *aktuell.Load() }, OhneGrenze: ohneGrenze, Aussen: aussen,
		Stand: func() library.Stand { return *aktStand.Load() }}
}

// anbindungFuer verbindet den Editor mit der Bibliothek b (ADR-0036): Die Leiste liest Index und Vorschauen, ein Klick
// auf eine Vorschau lädt die Aufnahme über oeffnerFuer.
func anbindungFuer(b *library.Bibliothek, kante document.Kante) *ui.Anbindung {
	return &ui.Anbindung{Bibliothek: b, Oeffnen: oeffnerFuer(b, kante), UmDatei: rettungUmDatei, Zuordnen: zuordnerFuer(b),
		Fortsetzen: fortsetzerFuer(b, kante)}
}

// rettungUmDatei hält die Rettungskopie einer Aufnahme anwendbar, während ihre Metadaten neu geschrieben werden
// (Schlagworte der Bibliotheksansicht, F-703). Ohne Zustandsordner läuft aendere unverändert.
func rettungUmDatei(pfad string, aendere func() error) error {
	ort, err := rescue.Ort(rettungUmgebung)
	if err != nil {
		rettungDiagnose(err)
		return aendere()
	}
	return rescue.UmDatei(ort, pfad, func(unveraendert func() bool) error {
		// Unmittelbar vor dem Lesen durch aendere: Nur ein unveränderter Stand zieht die Kopie nach (F-757).
		unveraendert()
		return aendere()
	})
}

// oeffnerFuer ist die Ladefunktion für den Wechsel der Aufnahme im selben Fenster (ADR-0036 Punkt 5): wie
// oeffneImEditor ohne --ignore-limit — ladeFuerEditor mit Budget-Prüfung, ein einfaches oder beschädigtes Bild bekommt
// die Kantenform aus [capture] seam, eine eigene Rettungskopie (wendet der Editor beim Übernehmen an). Der Fehler ist
// übersetzt wie beim Öffnen von der Kommandozeile.
func oeffnerFuer(b *library.Bibliothek, kante document.Kante) ui.Oeffnen {
	return func(pfad string) (ui.Geoeffnet, error) {
		// Datei → Öffnen…: früher übernommene Aufnahme wiederfinden (ADR-0016 Nachtrag 11).
		pfad, aussen := vorOeffnen(b, pfad)
		abs, g, stand, r, err := ladeMitRettung(pfad, false, ladeFuer) // wartet vor dem Laden (F-769)
		if err != nil {
			return ui.Geoeffnet{}, errors.New(nutzerFehler(err))
		}
		return geladenFuer(b, kante, abs, g, stand, r, aussen)
	}
}

// geladenFuer macht aus einer frisch geladenen Datei das Geoeffnet des Wechsels: ein einfaches oder beschädigtes Bild
// bekommt die Kantenform aus [capture] seam.
func geladenFuer(b *library.Bibliothek, kante document.Kante, abs string, g *fileformat.Gelesen, stand library.Stand,
	r *rettung, aussen bool) (ui.Geoeffnet, error) {
	if g.Fremd {
		if err := g.Dokument.VorgabeKante(kante); err != nil {
			r.ende()
			return ui.Geoeffnet{}, errors.New(nutzerFehler(err))
		}
	}
	return geoeffnetFuer(b, abs, g, stand, r, false, aussen), nil
}

// fortsetzerFuer setzt eine früher verlassene Aufnahme mit ihrem gemerkten Dokument d fort (ADR-0036 Zusatz Karte #356):
// Sie lädt die Datei wie oeffnerFuer (Rettungskopie, Budget-Prüfung) und nimmt d nur, wenn der geladene Stand gleich
// dem gemerkten ist und die Datei eine Schnapp-Datei ist; dann trägt die Datei genau den gespeicherten Stand von d,
// auch seine Schwärzungen (geschriebenBei in ablegenIn ist d.SchwaerzStand()). Sonst gilt das gelesene Dokument wie
// beim gewöhnlichen Wechsel.
func fortsetzerFuer(b *library.Bibliothek, kante document.Kante) ui.Fortsetzen {
	return func(pfad string, d *document.Dokument, stand library.Stand) (ui.Geoeffnet, error) {
		abs, g, st, r, err := ladeMitRettung(pfad, false, ladeFuer) // wartet vor dem Laden (F-769)
		if err != nil {
			return ui.Geoeffnet{}, errors.New(nutzerFehler(err))
		}
		if st != stand || g.Fremd || d == nil {
			return geladenFuer(b, kante, abs, g, st, r, false) // geändert: das gelesene gilt
		}
		fort := *g
		fort.Dokument = d
		geo := geoeffnetFuer(b, abs, &fort, st, r, false, false)
		geo.Fortgesetzt = true
		return geo, nil
	}
}

// ladeFuer ist ladeFuerEditor; Tests lassen das Laden beim Wechsel scheitern.
var ladeFuer = ladeFuerEditor

// bearbeiteNeu öffnet eine eben abgelegte Aufnahme im Editor (F-112): mit dem Dokument aus dem Speicher, ohne die
// Datei neu zu lesen. Was das Lesen sonst liefert, ist hier bekannt: eine Schnapp-Datei der Bibliothek (nicht fremd),
// ohne Beschriftungen (keine Prüfung des Rechenbudgets nötig) und mit leeren Metadaten, wie sie abgelegt wurde (die
// Erstellungszeit setzt ein Ausweichen beim Speichern ohnehin neu). Der Stand der Datei stammt vom Ablegen, nicht vom
// Start des Editors (F-110, Verify W-5): Ändert sie sich dazwischen oder während der Bearbeitung, merkt es das Speichern.
func bearbeiteNeu(a abgelegt) error {
	abs, err := filepath.Abs(a.pfad)
	if err != nil {
		return err
	}
	orte, err := library.OrteAus(os.Getenv)
	if err != nil {
		return err
	}
	g := &fileformat.Gelesen{Dokument: a.d}
	b := library.Neu(orte)
	r := rettungFuer(abs, false)
	ui.SetzeExportOhneGrenze(false)
	return editorStarten(geoeffnetFuer(b, abs, g, a.stand, r, false, false), anbindungFuer(b, a.kante))
}

// ladeFuerEditor liest eine Datei für den Editor (von außen oder aus der Bibliothek) und lehnt ein Dokument ab, dessen
// Rastern das Rechenbudget überschritte (render.Pruefe, F-125, F-132): Gegen präparierte Dateien wird beim Öffnen
// geprüft, damit Speichern nie abbrechen muss (F-131, ADR-0020 Nachtrag 1). stand ist der Zustand der Datei, gemerkt
// vor dem Lesen (F-110): Ändert sie sich danach, merkt es das Speichern. Lässt er sich nicht lesen, bleibt der
// Nullwert, und das Zurückschreiben weicht auf eine neue Aufnahme aus.
//
// ohneGrenze (--ignore-limit, F-146) überspringt nur die Ablehnung durch render.Pruefe, etwa für eigene Arbeit, die
// über das Budget gewachsen ist (F-138). Die Grenzen beim Lesen (Objekte, Streifen, Kacheln, Bildgröße) und die
// Formatprüfung gelten weiter.
func ladeFuerEditor(pfad string, ohneGrenze bool) (abs string, g *fileformat.Gelesen, stand library.Stand, err error) {
	abs, err = filepath.Abs(pfad)
	if err != nil {
		return "", nil, library.Stand{}, err
	}
	stand, _ = library.StandVon(abs)
	g, err = fileformat.Oeffne(abs)
	if err != nil {
		return "", nil, library.Stand{}, err
	}
	if !ohneGrenze {
		if err := render.Pruefe(g.Dokument); err != nil {
			return "", nil, library.Stand{}, err
		}
	}
	return abs, g, stand, nil
}

// speichernIn liefert die Speicherfunktion für den Editor. Gescheiterte Nebenschritte (Index, Vorschau) sind kein
// Fehler: Die Datei liegt dann in der Ablage. Bleibt dabei die Vorschau des vorigen Stands liegen, sagt es ein
// Hinweis (F-115). Weicht Speichern auf eine neue Aufnahme aus, nennt das Ergebnis die neue Datei; ist die alte
// stehen geblieben, weil das Zurückschreiben scheiterte, als Warnung (F-116). Wurde die Datei seit dem Öffnen von
// anderer Seite geändert (stand, etwa ein zweiter Editor), wird sie nicht überschrieben, sondern ebenfalls eine neue
// Aufnahme angelegt und als Warnung gemeldet (F-110). Fehler kommen übersetzt zurück (F-117).
//
// Überschreitet das gespeicherte Dokument das Rechenbudget (render.Pruefe), warnt das Ergebnis zusätzlich (F-138):
// Gespeichert ist es trotzdem (F-131), aber ladeFuerEditor lehnte es beim nächsten Öffnen ab. Die Warnung hält das
// Fenster beim Schließen einmal offen (ADR-0020 Nachtrag 2), sodass sich Beschriftungen noch entfernen lassen.
func speichernIn(b *library.Bibliothek, abs string, g *fileformat.Gelesen, stand library.Stand) ui.Speichern {
	return speichernMitNachher(b, abs, g, stand, nil)
}

// speichernMitNachher ist speichernIn; nachher bekommt nach jedem gelungenen Speichern den absoluten Pfad der Datei,
// die jetzt den Stand trägt (Ausgangsdatei der Rettungskopie, ADR-0029). nil: niemand.
func speichernMitNachher(b *library.Bibliothek, abs string, g *fileformat.Gelesen, stand library.Stand, nachher func(string, library.Stand)) ui.Speichern {
	speichern := ablegenIn(b, abs, g, stand, nachher, false)
	return func(d *document.Dokument) (ui.Gespeichert, error) {
		erg, err := speichern(d)
		if err != nil {
			return erg, err
		}
		budget := render.Pruefe(d)
		saveprogress.Marke("budget-geprueft")
		if budget == nil {
			return erg, nil
		}
		text := i18n.T("Saved, but the file has too many or too large annotations to be opened again. Remove some before closing.")
		if erg.Hinweis != "" {
			text = erg.Hinweis + " " + text
		}
		erg.Hinweis, erg.Warnung = text, true
		return erg, nil
	}
}

// entfernen löscht eine Bibliotheksaufnahme endgültig; Tests lassen es scheitern.
var entfernen = (*library.Bibliothek).Entfernen

// ablegenIn ist speichernIn ohne die Prüfung des Budgets. agent: jede neue Aufnahme, die dabei entsteht, kommt unter
// den Agenten-Ordner der Ablage (MCP-Werkzeug annotate, F-699): eine fremde Datei, die ein Agent beschriftet, steht
// dann als Agenten-Aufnahme in der Bibliothek, nicht zwischen den eigenen Aufnahmen des Menschen.
func ablegenIn(b *library.Bibliothek, abs string, g *fileformat.Gelesen, stand library.Stand, nachher func(string, library.Stand), agent bool) ui.Speichern {
	rel, inBibliothek := b.Relativ(abs)
	inBibliothek = inBibliothek && !g.Fremd
	meta := g.Meta
	// Schwärzungsstand beim letzten gelungenen Schreiben: Nur was danach geschwärzt wurde, steht in der bisherigen
	// Datei noch im Klartext (F-169).
	geschriebenBei := g.Dokument.SchwaerzStand()
	// Fassungen aus früheren Speicherungen, die neben der aktuellen Datei liegen bleiben (F-221): die Datei von außen,
	// die andere Fassung eines Konflikts und eine Aufnahme, deren Zurückschreiben scheiterte. Wird später geschwärzt,
	// zeigen sie das Geschwärzte noch.
	var reste []rest
	if ort, ok := herkunftVon(abs); ok {
		// Aus einer Datei von außen übernommen (ADR-0016 Nachtrag 10 Punkt 6): Sie bleibt, wie sie ist, und zeigt
		// weiter, was hier geschwärzt wird; das Speichern warnt wie bisher mit ihrem Namen (ADR-0022 Punkt 4, F-221).
		reste = append(reste, rest{ort: ort, aussen: true, bei: geschriebenBei})
	}
	// schreibe ist ein Speichern ohne Blick auf frühere Fassungen; neue sind die Fassungen, die dabei liegen bleiben.
	schreibe := func(d *document.Dokument) (_ ui.Gespeichert, neue []rest, _ error) {
		vorher := geschriebenBei // Schwärzungsstand der Datei, die dieses Speichern ersetzen soll
		neuGeschwaerzt := d.Geschwaerzt() && d.SchwaerzStand() != vorher
		// gezeigt ist der Stand, den eine jetzt liegen bleibende Fassung zeigt oder bis zu dem gerade gewarnt wird.
		gezeigt := vorher
		if neuGeschwaerzt {
			gezeigt = d.SchwaerzStand()
		}
		saveprogress.Marke("start")
		sichtbar, err := render.Bild(d)
		if err != nil {
			return ui.Gespeichert{}, nil, speicherFehler(err)
		}
		saveprogress.Marke("gerendert")
		var zurueck error // Fehler beim Zurückschreiben, falls auch das Ausweichen scheitert
		versucht := inBibliothek
		altRel, altStand := rel, stand // bisherige Datei, falls nach dem Schwärzen ausgewichen wird (F-126)
		konflikt := false
		if inBibliothek {
			_, neu, err := b.Speichern(rel, stand, d, sichtbar)
			if neu != (library.Stand{}) {
				stand = neu // geschrieben (auch wenn Index oder Vorschau scheiterten)
			}
			switch {
			case errors.Is(err, library.ErrVorschauAlt) || errors.Is(err, library.ErrTempRest):
				// Gespeichert, aber die alte Vorschau blieb (F-127) und/oder eine liegen gebliebene temporäre Kopie dieser
				// Aufnahme ließ sich nicht entfernen: Beides kann den alten Stand zeigen (F-274, Kernprinzip 2;
				// ADR-0022 Nachtrag 2). Treffen beide zusammen, nennt der Hinweis beide (Z-4).
				geschriebenBei = d.SchwaerzStand()
				vorschau, kopie := errors.Is(err, library.ErrVorschauAlt), errors.Is(err, library.ErrTempRest)
				var hinweis string
				switch {
				case vorschau && neuGeschwaerzt:
					hinweis = i18n.T("Saved, but the old preview of this capture could not be removed and still shows the unredacted content.")
				case vorschau:
					hinweis = i18n.T("Saved, but the old preview of this capture could not be removed.")
				case neuGeschwaerzt:
					hinweis = i18n.T("Saved, but a leftover temporary copy of this capture could not be removed and may still show the unredacted content.")
				default:
					hinweis = i18n.T("Saved, but a leftover temporary copy of this capture could not be removed.")
				}
				if vorschau && kopie {
					if neuGeschwaerzt {
						hinweis += " " + i18n.T("Also, a leftover temporary copy of this capture could not be removed and may still show the unredacted content.")
					} else {
						hinweis += " " + i18n.T("Also, a leftover temporary copy of this capture could not be removed.")
					}
				}
				return ui.Gespeichert{Hinweis: hinweis, Warnung: true}, nil, nil
			case err == nil || errors.Is(err, library.ErrIndex) || errors.Is(err, library.ErrVorschau):
				geschriebenBei = d.SchwaerzStand()
				return ui.Gespeichert{}, nil, nil
			case errors.Is(err, library.ErrGeaendert):
				// Ein anderer Editor oder ein anderes Programm hat die Datei inzwischen geschrieben: dessen Stand nicht
				// still überschreiben (F-110), sondern ausweichen.
				konflikt = true
			case errors.Is(err, fs.ErrNotExist) || errors.Is(err, library.ErrFremd) ||
				errors.Is(err, library.ErrPfad) || errors.Is(err, fileformat.ErrFormatNeuer):
				// Die Datei ist weg, ersetzt oder nicht mehr unsere (gelöscht, aufgeräumt, neuere Fassung): nicht
				// verlieren, sondern als neue Aufnahme ablegen (Kritik A-1).
			default:
				// Anderer Fehler (z. B. keine Schreibberechtigung): trotzdem eine neue Aufnahme versuchen, statt die
				// Bearbeitung zu verlieren (Verify V-1).
				zurueck = err
			}
		}
		m := meta
		m.Bearbeitet = true
		m.Erstellt = b.Jetzt() // neue Aufnahme der Bibliothek, Zeitpunkt des Speicherns
		ablegen := b.Ablegen
		if agent {
			ablegen = b.AblegenAgent
		}
		e, err := ablegen(d, sichtbar, m)
		if e.Pfad == "" {
			return ui.Gespeichert{}, nil, speicherFehler(errors.Join(zurueck, err))
		}
		rel, inBibliothek = e.Pfad, true
		geschriebenBei = d.SchwaerzStand()
		// Stand der neuen Aufnahme für das nächste Speichern; unlesbar bleibt der Nullwert (dann wiederum Ausweichen).
		stand, _ = library.StandVon(filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad)))
		name := filepath.Base(filepath.FromSlash(e.Pfad))
		// Pfad relativ zum Bilderordner (schnapp/JJJJ-MM/…), ohne Benutzernamen (F-155).
		ort := filepath.Base(b.Orte.Ablage) + "/" + e.Pfad
		erg := ui.Gespeichert{Datei: name, Hinweis: fmt.Sprintf(i18n.T("Saved as a new capture in the library: %s (the original file is unchanged)"), ort)}
		altOrt := filepath.Base(b.Orte.Ablage) + "/" + altRel
		switch {
		case konflikt:
			// Die andere Fassung gehört nicht Schnapp allein: nie entfernen, bei späterer Schwärzung warnen (F-221).
			neue = append(neue, rest{ort: altOrt, bei: gezeigt})
			erg.Hinweis = fmt.Sprintf(i18n.T("The file was changed elsewhere since it was opened; saved as a new capture: %s"), name)
			if neuGeschwaerzt {
				// Die andere Fassung gehört nicht Schnapp allein: nicht entfernen, aber sagen (ADR-0022 Punkt 4).
				erg.Hinweis += " " + i18n.T("The other version may still show the unredacted content.")
			}
			erg.Warnung = true
		case versucht && neuGeschwaerzt:
			// Die bisherige Datei zeigt noch, was seit ihrem Schreiben geschwärzt wurde. Die neue Aufnahme enthält alles
			// außer den geschwärzten Pixeln: die alte endgültig entfernen (ADR-0022 Punkt 4, F-126), sonst laut warnen.
			var geloescht bool
			var fe error
			if altRel != rel {
				geloescht, fe = entfernen(b, altRel, altStand)
			} // sonst trägt die neue Aufnahme den Namen der alten: Die alte war schon weg (nie die neue entfernen)
			switch {
			case geloescht && errors.Is(fe, library.ErrVorschauAlt):
				// Datei weg, Vorschaubild nicht (F-164): das Bild nennen, nicht die Datei.
				erg.Hinweis = fmt.Sprintf(i18n.T("The previous file could not be updated and was removed, but its preview could not be removed and still shows the unredacted content. Saved as a new capture: %s"), ort)
			case geloescht && nurNebenwirkung(fe):
				erg.Hinweis = fmt.Sprintf(i18n.T("The previous file could not be updated and still showed the unredacted content, so it was removed. Saved as a new capture: %s"), ort)
			case errors.Is(fe, library.ErrVorschauAlt):
				// Die Datei fehlte schon, ihr Vorschaubild ließ sich nicht entfernen (F-186): beides sagen, nicht „Delete
				// it yourself“ für eine Datei, die es nicht gibt.
				erg.Hinweis = fmt.Sprintf(i18n.T("The previous file %s was no longer there; wherever it is now, it still shows the unredacted content. Its preview could not be removed and still shows it too. Saved as a new capture: %s"), altOrt, ort)
			case nurNebenwirkung(fe):
				// Die Datei fehlte schon (verschoben, Papierkorb): nicht „entfernt“ behaupten (F-167). Scheiterte nur der
				// Index, ist das für den Nutzer dasselbe; der Abgleich holt ihn nach (F-186).
				erg.Hinweis = fmt.Sprintf(i18n.T("The previous file %s was no longer there; wherever it is now, it still shows the unredacted content. Saved as a new capture: %s"), altOrt, ort)
			default:
				erg.Hinweis = fmt.Sprintf(i18n.T("The previous file could not be updated and still shows the unredacted content: %s. Delete it yourself. Saved as a new capture: %s"), altOrt, ort)
				// Bleibt liegen: eine weitere Schwärzung versucht das Entfernen erneut (F-221).
				neue = append(neue, rest{rel: altRel, stand: altStand, ort: altOrt, bei: gezeigt})
			}
			if errors.Is(fe, library.ErrTempRest) {
				// Z-2: Die alte Datei ist weg, eine temporäre Kopie von ihr nicht.
				erg.Hinweis += " " + i18n.T("A temporary copy of the previous file could not be removed and may still show the unredacted content.")
			}
			erg.Warnung = true
		case versucht:
			// Die bisherige Datei wurde nicht aktualisiert; liegt sie noch, enthält sie den alten Stand. Bewusst so
			// (ADR-0020 Nachtrag 1: Ausweichen statt Verlust); ohne Schwärzung ist darin nichts Vertrauliches — bis
			// später geschwärzt wird (F-221): dann entfernt der Rückblick sie.
			erg.Hinweis = fmt.Sprintf(i18n.T("The previous file could not be updated; saved as a new capture: %s"), name)
			erg.Warnung = true
			if altRel != rel {
				neue = append(neue, rest{rel: altRel, stand: altStand, ort: altOrt, bei: gezeigt})
			}
		case neuGeschwaerzt:
			// Datei von außen: Sie bleibt, wie sie ist (ADR-0020), zeigt aber noch, was geschwärzt wurde.
			erg.Hinweis = fmt.Sprintf(i18n.T("Saved as a new capture in the library: %s. The original file is unchanged and still shows the unredacted content."), ort)
			erg.Warnung = true
		}
		if !versucht {
			// Datei von außen (oder fremd): bleibt unberührt; bei späterer Schwärzung warnen (F-221). Die Bibliothek merkt
			// sich ihren Inhalt, damit erneutes Öffnen diese Aufnahme findet (ADR-0016 Nachtrag 11).
			neue = append(neue, rest{ort: filepath.Base(abs), aussen: true, bei: gezeigt})
			if !agent {
				merkeUebernahme(b, abs, e.Pfad, filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad)))
			}
		}
		if errors.Is(err, library.ErrVorschauAlt) {
			erg.Hinweis += " " + i18n.T("The old preview could not be removed.")
			if neuGeschwaerzt {
				erg.Hinweis += " " + i18n.T("It still shows the unredacted content.")
			}
			erg.Warnung = true
		}
		if errors.Is(err, library.ErrTempRest) {
			// Z-4: Auch beim Ablegen der neuen Aufnahme kann eine temporäre Kopie bleiben, neben der Vorschau-Meldung.
			erg.Hinweis += " " + i18n.T("A temporary copy of the new capture could not be removed.")
			erg.Warnung = true
		}
		return erg, neue, nil
	}
	return func(d *document.Dokument) (ui.Gespeichert, error) {
		erg, neue, err := schreibe(d)
		if err != nil {
			return erg, err
		}
		erg, reste = rueckblick(b, d, reste, rel, erg)
		reste = append(reste, neue...)
		if nachher != nil {
			// Datei, die jetzt den Stand trägt (ADR-0029), und ihr Stand nach diesem Schreiben (F-834).
			nachher(filepath.Join(b.Orte.Ablage, filepath.FromSlash(rel)), stand)
		}
		return erg, nil
	}
}

// rest ist eine Fassung aus einem früheren Speichern, die neben der aktuellen Datei liegen blieb (F-221).
type rest struct {
	rel    string        // Aufnahme der Bibliothek, die Schnapp entfernen darf; leer: nur warnen
	stand  library.Stand // Stand dieser Aufnahme beim Liegenbleiben (Entfernen nur, wenn unverändert)
	ort    string        // Name für die Meldung, ohne Benutzerpfad (F-155)
	aussen bool          // nur warnen: Datei von außen; sonst (ohne rel) die andere Fassung eines Konflikts
	bei    uint64        // Schwärzungsstand, den die Fassung zeigt oder bis zu dem schon gewarnt wurde
}

// rueckblick prüft nach einem gelungenen Speichern die früher liegen gebliebenen Fassungen (F-221, ADR-0022 Punkt 4):
// Wurde seit ihrem Stand geschwärzt, entfernt es eigene Aufnahmen der Bibliothek wie F-126 und warnt sonst. Die
// gerade geschriebene Aufnahme aktuell wird nie entfernt. Zurück kommen das ergänzte Ergebnis und die Fassungen, die
// weiter liegen.
func rueckblick(b *library.Bibliothek, d *document.Dokument, reste []rest, aktuell string, erg ui.Gespeichert) (ui.Gespeichert, []rest) {
	jetzt := d.SchwaerzStand()
	sage := func(text string) {
		if erg.Hinweis != "" {
			erg.Hinweis += " "
		}
		erg.Hinweis += text
		erg.Warnung = true
	}
	var bleiben []rest
	for _, r := range reste {
		switch {
		case r.rel != "" && r.rel == aktuell:
			continue // trägt jetzt die aktuelle Aufnahme (Name wiederverwendet): nichts mehr zu tun
		case !d.Geschwaerzt() || jetzt == r.bei:
			bleiben = append(bleiben, r)
			continue
		case r.rel == "":
			if r.aussen {
				sage(fmt.Sprintf(i18n.T("The original file %s is unchanged and still shows the unredacted content."), r.ort))
			} else {
				sage(fmt.Sprintf(i18n.T("The other version %s may still show the unredacted content."), r.ort))
			}
			r.bei = jetzt
			bleiben = append(bleiben, r)
			continue
		}
		geloescht, fe := entfernen(b, r.rel, r.stand)
		vorschau := errors.Is(fe, library.ErrVorschauAlt)
		switch {
		case geloescht && vorschau:
			sage(fmt.Sprintf(i18n.T("The earlier file %s, left behind when saving failed, was removed, but its preview could not be removed and still shows the unredacted content."), r.ort))
		case geloescht && nurNebenwirkung(fe):
			sage(fmt.Sprintf(i18n.T("The earlier file %s, left behind when saving failed, still showed the unredacted content and was removed."), r.ort))
		case vorschau:
			sage(fmt.Sprintf(i18n.T("The earlier file %s was no longer there; wherever it is now, it still shows the unredacted content. Its preview could not be removed and still shows it too."), r.ort))
		case nurNebenwirkung(fe):
			sage(fmt.Sprintf(i18n.T("The earlier file %s was no longer there; wherever it is now, it still shows the unredacted content."), r.ort))
		default:
			// Geändert oder nicht entfernbar: bleibt vorgemerkt, eine weitere Schwärzung versucht es erneut.
			sage(fmt.Sprintf(i18n.T("The earlier file %s still shows the unredacted content. Delete it yourself."), r.ort))
			r.bei = jetzt
			bleiben = append(bleiben, r)
		}
		if errors.Is(fe, library.ErrTempRest) {
			sage(fmt.Sprintf(i18n.T("A temporary copy of the earlier file %s could not be removed and may still show the unredacted content."), r.ort))
		}
	}
	return erg, bleiben
}

// nurNebenwirkung: Entfernen meldet nichts, nur den Index (der Abgleich holt ihn nach, F-186) und/oder eine liegen
// gebliebene temporäre Kopie (Z-2, wird gesondert genannt). Die Datei ist dann entfernt oder war schon weg.
func nurNebenwirkung(fe error) bool {
	return fe == nil || errors.Is(fe, library.ErrIndex) || errors.Is(fe, library.ErrTempRest)
}

// uebersetzt trägt einen fertig übersetzten Text für die Ausgabe und behält den ursprünglichen Fehler (errors.Is).
type uebersetzt struct {
	text string
	err  error
}

func (u uebersetzt) Error() string { return u.text }
func (u uebersetzt) Unwrap() error { return u.err }

// speicherFehler übersetzt einen Fehler beim Speichern (F-117): ohne Pfad und ohne Paketpräfix, Schreiben statt Lesen.
func speicherFehler(err error) error {
	var text string
	switch {
	case errors.Is(err, fs.ErrPermission):
		text = i18n.T("No permission to write the file.")
	case errors.Is(err, syscall.ENOSPC):
		text = i18n.T("Not enough disk space.")
	case errors.Is(err, library.ErrKeinHeim):
		text = i18n.T("No home directory (HOME is not set).")
	default:
		text = i18n.T("The file could not be written.")
	}
	return uebersetzt{text: text, err: err}
}

// sammleOhneUebersetzt geht einen Join-Baum durch und legt die Ausgabe aller Blätter ab, die keinen übersetzten
// Speicherfehler tragen (F-129); texte sammelt die übersetzten Texte aller Speicherfehler in Baumreihenfolge, auch
// Dubletten (die entfernt nutzerFehler bei der Ausgabe).
func sammleOhneUebersetzt(err error, teile, texte *[]string) {
	if err == nil {
		return
	}
	if j, ok := err.(interface{ Unwrap() []error }); ok {
		for _, e := range j.Unwrap() {
			sammleOhneUebersetzt(e, teile, texte)
		}
		return
	}
	var u uebersetzt
	if errors.As(err, &u) {
		*texte = append(*texte, u.text)
		return
	}
	*teile = append(*teile, nutzerFehlerBlatt(err))
}

// ursacheSpeicherfehler liefert die ursprüngliche Ursache hinter einem übersetzten Fehler (nur für das Debug-Log:
// sie kann Pfade enthalten, F-129); ohne übersetzten Fehler ist es leer.
func ursacheSpeicherfehler(err error) string {
	var u uebersetzt
	if errors.As(err, &u) && u.err != nil {
		return u.err.Error()
	}
	return ""
}

// nutzerFehler übersetzt bekannte Fehler beim Öffnen für die Ausgabe; Fehler des Speicherns sind schon übersetzt; Unbekanntes bleibt, wie es ist
// (Paketfehler sind nicht übersetzt, Kritik A-9).
func nutzerFehler(err error) string {
	// Jeder Join wird durchlaufen, nicht nur einer mit übersetztem Speicherfehler (F-301): Sonst nähme die Weiche des
	// Blatts den ersten passenden Fall, und ein Teil — etwa der Hinweis auf ungesicherte Änderungen neben einem
	// Fensterfehler — fehlte. Schon übersetzte Speicherfehler (F-117) stehen hinter den übrigen Teilen (Fenster, F-129);
	// jeder verschiedene Text erscheint einmal (Reihenfolge wie im Baum), die Ursache mit ihren Pfaden nie.
	var teile, texte []string
	sammleOhneUebersetzt(err, &teile, &texte)
	for _, t := range texte {
		if !slices.Contains(teile, t) {
			teile = append(teile, t)
		}
	}
	return strings.Join(teile, "; ")
}

// ungesichert meldet, ob ein Fehler des Editors ungesicherte Änderungen bedeutet: das Speichern hat nicht geantwortet
// (ui.ErrSpeichernHaengt) oder ist gescheitert, ohne dass der Nutzer es verworfen hat (übersetzter Speicherfehler,
// auch beim Zerstören des Fensters, F-300). Nur dann ist der Exit-Code ein Fehler (ADR-0020 Nachtrag 7, F-302); ein
// Fensterfehler allein (kein Wayland, Overlay offen) lässt eine schon abgelegte Aufnahme unberührt.
func ungesichert(err error) bool {
	var u uebersetzt
	return errors.Is(err, ui.ErrSpeichernHaengt) || errors.As(err, &u)
}

// nutzerFehlerBlatt übersetzt einen einzelnen Fehler (kein Join, kein übersetzter Speicherfehler).
func nutzerFehlerBlatt(err error) string {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return i18n.T("The file does not exist.")
	case errors.Is(err, fs.ErrPermission):
		return i18n.T("No permission to read the file.")
	case errors.Is(err, fileformat.ErrKeineDatei):
		return i18n.T("This is not a regular file.")
	case errors.Is(err, fileformat.ErrKeinPNG):
		return i18n.T("This is neither a PNG nor a JPEG image.")
	case errors.Is(err, fileformat.ErrZuGross):
		return i18n.T("The image is too large.")
	case errors.Is(err, fileformat.ErrBeschaedigt):
		return i18n.T("The image is damaged.")
	case errors.Is(err, fileformat.ErrFormatNeuer): // F-663; Oeffne öffnet sie als Bild, andere Wege nicht
		return i18n.T("The file comes from a newer version of Schnapp.")
	case errors.Is(err, ui.ErrSpeichernHaengt):
		return i18n.T("Saving did not respond; the changes may not have been saved.")
	case errors.Is(err, ui.ErrOverlayOffen):
		return i18n.T("The selection overlay did not close in time.")
	case errors.Is(err, library.ErrKeinHeim):
		return i18n.T("No home directory (HOME is not set).")
	case errors.Is(err, render.ErrZuAufwendig):
		return i18n.T("This file has too many or too large annotations to open. To open it anyway, use --ignore-limit.")
	}
	return err.Error()
}
