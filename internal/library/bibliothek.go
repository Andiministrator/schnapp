// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/errprefix"
	"github.com/Andiministrator/schnapp/internal/fileformat"
)

// Fehler der Bibliothek. Nach ErrIndex und ErrVorschau liegt die Aufnahme trotzdem in der Ablage; der Index
// heilt sich beim nächsten Neuaufbau, das Vorschaubild beim nächsten Ablegen oder Neuaufbau. Nach ErrVorschauAlt
// liegt zusätzlich noch das Vorschaubild des vorigen Stands im Cache (F-115) — das muss der Aufrufer melden.
var (
	ErrIndex       = errors.New("library: Index nicht aktualisiert")
	ErrVorschau    = errors.New("library: Vorschaubild nicht geschrieben")
	ErrVorschauAlt = errors.New("library: altes Vorschaubild nicht entfernt")
	ErrFremd       = errors.New("library: keine Arbeitsdatei von Schnapp")
	ErrPfad        = errors.New("library: Pfad liegt nicht in der Ablage")
	// ErrAblageLeer: Die Ablage enthält keine einzige Aufnahme, der Vorschau-Cache aber Vorschaubilder (leerer
	// Einhängepunkt, verschobener Ordner). VerwaisteVorschauen liefert dann keine Liste (F-270).
	ErrAblageLeer = errors.New("library: Ablage ohne Aufnahmen, Cache nicht leer")
)

// Eintrag ist eine Aufnahme im Index. Alle Angaben stehen auch im snAP-Block der Datei (ADR-0008).
type Eintrag struct {
	Pfad        string    `json:"pfad"` // relativ zur Ablage, mit Schrägstrichen
	Erstellt    time.Time `json:"erstellt"`
	Breite      int       `json:"breite"`
	Hoehe       int       `json:"hoehe"`
	Schlagworte []string  `json:"schlagworte,omitempty"`
	Quelle      string    `json:"quelle,omitempty"`
	Bearbeitet  bool      `json:"bearbeitet,omitempty"`
}

// Bibliothek ist die Ablage samt Index. Zugriffe innerhalb eines Prozesses sind serialisiert.
type Bibliothek struct {
	Orte Orte
	// Zone bestimmt Ordner- und Dateinamen sowie das Löschdatum im Papierkorb (Voreinstellung: lokale Zeit).
	Zone *time.Location
	// Jetzt liefert die Zeit, wenn Metadaten.Erstellt fehlt (Voreinstellung time.Now).
	Jetzt func() time.Time

	mu sync.Mutex
}

// Neu legt eine Bibliothek an den gegebenen Orten an. Es wird nichts auf der Platte angelegt.
func Neu(o Orte) *Bibliothek {
	return &Bibliothek{Orte: o, Zone: time.Local, Jetzt: time.Now}
}

// umbenenne ist os.Rename; Tests ersetzen es, um einen Fehlschlag im letzten Schritt des atomaren Schreibens
// nachzubauen.
var umbenenne = os.Rename

const tempPraefix = ".schnapp-tmp-"

// schreibeAtomar schreibt ziel über eine temporäre Datei (0600) im selben Ordner: schreiben, fsync, umbenennen.
// Der Ordner muss existieren. Bei einem Fehler wird genau die eigene temporäre Datei entfernt.
func schreibeAtomar(ziel string, schreibe func(io.Writer) error) error {
	return schreibeAtomarGeprueft(ziel, schreibe, nil)
}

// schreibeAtomarGeprueft wie schreibeAtomar; vorUmbenennen (nil: keine Prüfung) läuft nach dem Schreiben und vor dem
// Umbenennen. Liefert es einen Fehler, bleibt das Ziel unberührt und die temporäre Datei wird entfernt.
func schreibeAtomarGeprueft(ziel string, schreibe func(io.Writer) error, vorUmbenennen func() error) (fehler error) {
	ordner := filepath.Dir(ziel)
	tmp, err := os.CreateTemp(ordner, tempPraefix+"*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	geschlossen := false
	defer func() {
		if fehler == nil {
			return
		}
		if !geschlossen {
			tmp.Close()
		}
		// Entfernt wird nur die eigene temporäre Datei: im Zielordner und mit eigenem Präfix.
		if filepath.Dir(name) == filepath.Clean(ordner) && strings.HasPrefix(filepath.Base(name), tempPraefix) {
			os.Remove(name)
		}
	}()
	if err := tmp.Chmod(rechteDatei); err != nil {
		return err
	}
	// tmp geht unverpackt an schreibe: Die Arbeitsdatei synchronisiert je Stück und pulst danach (F-313); dieser
	// letzte Sync sichert nur noch den Rest und die Metadaten.
	if err := schreibe(tmp); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	geschlossen = true
	if err := tmp.Close(); err != nil {
		return err
	}
	if vorUmbenennen != nil {
		if err := vorUmbenennen(); err != nil {
			return err
		}
	}
	if err := umbenenne(name, ziel); err != nil {
		return err
	}
	syncOrdner(ordner) // Verzeichniseintrag sichern (F-274 c)
	return nil
}

// zeitname liefert Ordner und Dateiname (ohne Endung) zu einer Erstellzeit.
func (b *Bibliothek) zeitname(t time.Time) (monat, name string) {
	t = t.In(b.Zone)
	return t.Format("2006-01"), t.Format("2006-01-02_15-04-05")
}

// Ablegen schreibt die Arbeitsdatei in die Ablage, hängt eine Zeile an den Index und schreibt das Vorschaubild.
// sichtbar ist das fertige Bild (vom Aufrufer gerendert, siehe fileformat.SchreibeArbeitsdatei). Fehlt m.Erstellt,
// gilt die aktuelle Zeit. Scheitert nur Index oder Vorschau, liefert Ablegen den Eintrag samt ErrIndex bzw.
// ErrVorschau.
func (b *Bibliothek) Ablegen(d *document.Dokument, sichtbar image.Image, m fileformat.Metadaten) (_ Eintrag, fehler error) {
	defer errprefix.Defer("library", &fehler)
	return b.ablegen("", d, sichtbar, m)
}

// AgentenOrdner ist der Unterordner der Ablage für Aufnahmen über den MCP-Server (ADR-0030 Nachtrag 2): Ablage
// `agents/<JJJJ-MM>/…png`, Index-Pfad `agents/…`. Aufräumen und Neuaufbau behandeln sie wie jede Aufnahme.
const AgentenOrdner = "agents"

// AblegenAgent wie Ablegen, aber im Unterordner AgentenOrdner.
func (b *Bibliothek) AblegenAgent(d *document.Dokument, sichtbar image.Image, m fileformat.Metadaten) (_ Eintrag, fehler error) {
	defer errprefix.Defer("library", &fehler)
	return b.ablegen(AgentenOrdner, d, sichtbar, m)
}

// Agent meldet, ob der Eintrag eine Agenten-Aufnahme ist (liegt unter AgentenOrdner).
func (e Eintrag) Agent() bool {
	return strings.HasPrefix(e.Pfad, AgentenOrdner+"/")
}

// ablegen legt unter dem Unterordner unter der Ablage ab (leer = direkt in der Ablage).
func (b *Bibliothek) ablegen(unter string, d *document.Dokument, sichtbar image.Image, m fileformat.Metadaten) (Eintrag, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if m.Erstellt.IsZero() {
		m.Erstellt = b.Jetzt()
	}
	m.Erstellt = m.Erstellt.UTC().Truncate(time.Second)
	m.Schlagworte = bereinige(m.Schlagworte)

	monat, name := b.zeitname(m.Erstellt)
	ordner := filepath.Join(b.Orte.Ablage, unter, monat)
	ordnerKette := []string{b.Orte.Ablage}
	if unter != "" {
		ordnerKette = append(ordnerKette, filepath.Join(b.Orte.Ablage, unter))
		monat = unter + "/" + monat
	}
	for _, o := range append(ordnerKette, ordner) {
		if err := eigenerOrdner(o); err != nil {
			return Eintrag{}, err
		}
	}
	// Neuer Name: exklusiv anlegen, nie überschreiben — auch nicht gegen einen zweiten Schnapp-Prozess (F-48).
	dn, err := schreibeNeu(ordner, func(n int) string {
		if n == 1 {
			return name + ".png"
		}
		return fmt.Sprintf("%s-%d.png", name, n)
	}, func(w io.Writer) error {
		return fileformat.SchreibeArbeitsdatei(w, d, sichtbar, m)
	})
	var folge error
	if err != nil {
		if !errors.Is(err, ErrTempRest) || dn == "" {
			return Eintrag{}, err
		}
		folge = err // abgelegt, aber eine temporäre Kopie blieb liegen (F-274): gilt als Erfolg mit Warnung
	}
	rel := monat + "/" + dn
	g := d.Groesse()
	e := Eintrag{Pfad: rel, Erstellt: m.Erstellt, Breite: g.X, Hoehe: g.Y, Schlagworte: m.Schlagworte,
		Quelle: m.Quelle, Bearbeitet: m.Bearbeitet}

	if err := b.haengeAn(e); err != nil {
		folge = errors.Join(folge, fmt.Errorf("%w: %v", ErrIndex, err))
	}
	if err := b.erneuereVorschau(e.Pfad, sichtbar); err != nil {
		folge = errors.Join(folge, err)
	}
	return e, folge
}

// bereinige entfernt leere und doppelte Schlagworte (Groß-/Kleinschreibung egal, das erste gewinnt).
func bereinige(worte []string) []string {
	var aus []string
	for _, w := range worte {
		w = strings.TrimSpace(w)
		if w == "" {
			continue
		}
		doppelt := false
		for _, a := range aus {
			if strings.EqualFold(a, w) {
				doppelt = true
				break
			}
		}
		if !doppelt {
			aus = append(aus, w)
		}
	}
	return aus
}

// --- Index ---

// gueltig prüft einen Eintrag aus dem Index: Der Pfad muss lokal bleiben (kein „..“, nicht absolut) und auf .png
// enden — der Index ist nur ein Cache und darf nie auf Dateien außerhalb der Ablage zeigen.
func (e Eintrag) gueltig() bool {
	p := filepath.FromSlash(e.Pfad)
	return e.Pfad != "" && !strings.Contains(e.Pfad, `\`) && filepath.IsLocal(p) &&
		strings.EqualFold(filepath.Ext(p), ".png") && e.Breite > 0 && e.Hoehe > 0
}

// haengeAn hängt eine Zeile an den Index an (Ordner 0700, Datei 0600), unter der Sperre des Index (F-686).
func (b *Bibliothek) haengeAn(e Eintrag) error {
	zeile, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return b.unterSperre(func() error { return b.haengeZeileAn(zeile) })
}

// haengeZeileAn hängt die fertige Zeile an. Läuft unter der Sperre des Index.
func (b *Bibliothek) haengeZeileAn(zeile []byte) error {
	if err := eigenerOrdner(filepath.Dir(b.Orte.Index)); err != nil {
		return err
	}
	f, err := os.OpenFile(b.Orte.Index, os.O_RDWR|os.O_APPEND|os.O_CREATE, rechteDatei)
	if err != nil {
		return err
	}
	defer f.Close()
	// Ein Absturz mitten in einer Zeile ließ evtl. keinen Zeilenumbruch zurück: sonst klebte die neue Zeile an
	// der kaputten und beide gingen verloren.
	if st, err := f.Stat(); err == nil && st.Size() > 0 {
		var letztes [1]byte
		if _, err := f.ReadAt(letztes[:], st.Size()-1); err == nil && letztes[0] != '\n' {
			zeile = append([]byte{'\n'}, zeile...)
		}
	}
	if _, err := f.Write(append(zeile, '\n')); err != nil {
		return err
	}
	return f.Close()
}

// Geladen ist das Ergebnis von Laden.
type Geladen struct {
	Eintraege []Eintrag
	Defekt    int // übersprungene, unlesbare oder ungültige Zeilen
}

// Laden liest den Index. Kaputte Zeilen werden übersprungen und gezählt; ein fehlender Index ist leer.
func (b *Bibliothek) Laden() (_ Geladen, fehler error) {
	defer errprefix.Defer("library", &fehler)
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.lade()
}

func (b *Bibliothek) lade() (Geladen, error) {
	f, err := os.Open(b.Orte.Index)
	if errors.Is(err, fs.ErrNotExist) {
		return Geladen{}, nil
	}
	if err != nil {
		return Geladen{}, err
	}
	defer f.Close()
	var g Geladen
	r := bufio.NewReader(f)
	for {
		zeile, err := r.ReadBytes('\n')
		if z := bytes.TrimSpace(zeile); len(z) > 0 {
			var e Eintrag
			if json.Unmarshal(z, &e) != nil || !e.gueltig() {
				g.Defekt++
			} else {
				g.Eintraege = append(g.Eintraege, e)
			}
		}
		if err == io.EOF {
			return g, nil
		}
		if err != nil {
			return g, err
		}
	}
}

// schreibeIndex ersetzt den Index atomar.
func (b *Bibliothek) schreibeIndex(es []Eintrag) error {
	if err := eigenerOrdner(filepath.Dir(b.Orte.Index)); err != nil {
		return err
	}
	return schreibeAtomar(b.Orte.Index, func(w io.Writer) error {
		bw := bufio.NewWriter(w)
		for _, e := range es {
			zeile, err := json.Marshal(e)
			if err != nil {
				return err
			}
			bw.Write(zeile)
			bw.WriteByte('\n')
		}
		return bw.Flush()
	})
}

func sortiere(es []Eintrag) {
	sort.SliceStable(es, func(i, j int) bool {
		if !es[i].Erstellt.Equal(es[j].Erstellt) {
			return es[i].Erstellt.Before(es[j].Erstellt)
		}
		return es[i].Pfad < es[j].Pfad
	})
}

// NeuaufbauErgebnis ist das Ergebnis von Neuaufbau.
type NeuaufbauErgebnis struct {
	Aufgenommen   int      // Aufnahmen im neuen Index
	Uebersprungen []string // Pfade (relativ zur Ablage) fremder oder beschädigter PNG-Dateien; sie bleiben unberührt
	// TempReste: liegen gebliebene temporäre Dateien von Schnapp, die sich nicht entfernen ließen (F-274); jede ist
	// womöglich ein zweiter Name einer Aufnahme und zeigt nach dem Schwärzen noch den alten Stand.
	TempReste []string
}

// Neuaufbau liest alle PNG-Dateien unter der Ablage, übernimmt die Arbeitsdateien von Schnapp und schreibt den
// Index atomar neu. Fremde und beschädigte Dateien werden übersprungen, nie verändert oder gelöscht. Das einzige, was
// Neuaufbau entfernt, sind liegen gebliebene eigene temporäre Dateien (`.schnapp-tmp-*`, reguläre Dateien, älter als
// eine Minute, F-274); Zukunftsdatierte meldet er in TempReste, ohne sie zu löschen.
//
// Er hält b.mu und die Sperre des Index (F-686) über das Ablaufen der Ablage und das Schreiben des Index: Ein zweiter
// Prozess, der in dieser Zeit ablegt, wartet mit seiner Indexzeile, bis der neue Index steht, statt dass sie beim
// Ersetzen verloren geht. Fehlende Vorschaubilder (volles Dekodieren je Bild) heilt er erst danach, nur noch unter
// b.mu: Sonst hielt ein Neuaufbau über eine große Ablage die Sperre länger als sperrFrist, und die Indexzeile des
// zweiten Prozesses scheiterte (F-691).
func (b *Bibliothek) Neuaufbau() (erg NeuaufbauErgebnis, fehler error) {
	defer errprefix.Defer("library", &fehler)
	b.mu.Lock()
	defer b.mu.Unlock()
	var heilen []heilAuftrag
	fehler = b.unterSperre(func() error {
		var err error
		erg, heilen, err = b.neuaufbau()
		return err
	})
	b.heileAlle(heilen)
	return erg, fehler
}

// heilAuftrag ist eine Aufnahme, deren Vorschaubild der Neuaufbau nach der Freigabe der Index-Sperre nachzieht (F-52,
// F-691); stand ist ihr Stand beim Ablaufen.
type heilAuftrag struct {
	pfad, rel string
	stand     Stand
}

// heileNachNeuaufbau ist heileVorschau für die Aufträge des Neuaufbaus; Tests ersetzen es, um zu sehen, unter welchen
// Sperren es läuft.
var heileNachNeuaufbau = (*Bibliothek).heileVorschau

// heileAlle zieht die fehlenden Vorschaubilder nach dem Neuaufbau nach. Läuft unter b.mu, ohne die Sperre des Index;
// heileVorschau prüft den Stand jeder Datei selbst (F-651), eine inzwischen ersetzte bleibt ohne Vorschau.
func (b *Bibliothek) heileAlle(as []heilAuftrag) {
	for _, a := range as {
		heileNachNeuaufbau(b, a.pfad, a.rel, a.stand) // scheitert es, bleibt nur die Vorschau aus
	}
}

// neuaufbau ist Neuaufbau ohne Sperren. Läuft unter b.mu und der Sperre des Index. Die Aufnahmen, deren Vorschaubild
// fehlt, liefert er als Aufträge, die der Aufrufer nach der Freigabe der Sperre mit heileAlle abarbeitet (F-691).
func (b *Bibliothek) neuaufbau() (NeuaufbauErgebnis, []heilAuftrag, error) {
	var erg NeuaufbauErgebnis
	var es []Eintrag
	var heilen []heilAuftrag
	// Ablage vor dem Ablaufen auflösen: WalkDir steigt an einer Wurzel, die ein Symlink ist, nicht ab, der Index
	// bliebe leer (F-271). Fehlt die Ablage ganz, gibt es noch keine Aufnahme.
	wurzel, err := filepath.EvalSymlinks(b.Orte.Ablage)
	if errors.Is(err, fs.ErrNotExist) {
		return erg, nil, b.schreibeIndex(nil)
	}
	if err != nil {
		return erg, nil, err
	}
	jetzt := time.Now()
	err = filepath.WalkDir(wurzel, func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() && strings.HasPrefix(d.Name(), tempPraefix) {
			// Liegen gebliebene eigene temporäre Datei (F-274): nach dem Link-Weg ein zweiter Name einer Aufnahme.
			dir := filepath.Dir(pfad)
			if fi, err := d.Info(); err == nil && fi.Mode().IsRegular() {
				if jetzt.Sub(fi.ModTime()) >= tempAlter {
					if err := entferneEigeneTemp(dir, pfad); err != nil {
						rel, _ := filepath.Rel(wurzel, pfad)
						erg.TempReste = append(erg.TempReste, filepath.ToSlash(rel))
					}
				} else if fi.ModTime().After(jetzt) {
					// Zeit in der Zukunft (verstellte Uhr): nie alt genug, bliebe sonst für immer unbemerkt (Z-1).
					rel, _ := filepath.Rel(wurzel, pfad)
					erg.TempReste = append(erg.TempReste, filepath.ToSlash(rel))
				}
			}
			return nil
		}
		if !d.Type().IsRegular() || !strings.EqualFold(filepath.Ext(pfad), ".png") {
			return nil
		}
		rel, err := filepath.Rel(wurzel, pfad)
		if err != nil {
			return err
		}
		e, ok := b.liesEintrag(pfad, filepath.ToSlash(rel))
		if !ok {
			erg.Uebersprungen = append(erg.Uebersprungen, filepath.ToSlash(rel))
			return nil
		}
		es = append(es, e)
		if _, err := os.Stat(b.VorschauPfad(e.Pfad)); err != nil {
			if fi, err := os.Lstat(pfad); err == nil && fi.Mode().IsRegular() {
				heilen = append(heilen, heilAuftrag{pfad: pfad, rel: e.Pfad, stand: standAus(fi)})
			}
		}
		return nil
	})
	if err != nil {
		return erg, nil, err
	}
	sortiere(es)
	erg.Aufgenommen = len(es)
	return erg, heilen, b.schreibeIndex(es)
}

// liesEintrag liest die Angaben einer Arbeitsdatei; ok ist falsch bei fremden, beschädigten oder unlesbaren
// Dateien. Gelesen werden nur snAP und IHDR (fileformat.LeseMetadaten), die Pixel bleiben undekodiert: Der
// Neuaufbau über viele Aufnahmen bleibt schnell und braucht keinen Speicher für Bilder.
func (b *Bibliothek) liesEintrag(pfad, rel string) (Eintrag, bool) {
	f, err := os.Open(pfad)
	if err != nil {
		return Eintrag{}, false
	}
	defer f.Close()
	g, err := fileformat.LeseMetadaten(f)
	if err != nil || g.Fremd {
		return Eintrag{}, false
	}
	erstellt := g.Meta.Erstellt.UTC()
	if g.Meta.Erstellt.IsZero() {
		st, err := f.Stat()
		if err != nil {
			return Eintrag{}, false
		}
		erstellt = st.ModTime().UTC().Truncate(time.Second)
	}
	e := Eintrag{Pfad: rel, Erstellt: erstellt, Breite: g.Breite, Hoehe: g.Hoehe, Schlagworte: bereinige(g.Meta.Schlagworte),
		Quelle: g.Meta.Quelle, Bearbeitet: g.Meta.Bearbeitet}
	return e, e.gueltig()
}

// Abgleich lässt Index-Einträge fallen, deren Datei fehlt (etwa nach Verschieben im Dateimanager), und
// auch kaputte Zeilen; die Vorschauen der fehlenden fallen mit. Geschrieben wird nur bei einer Änderung. Rückgabe: Zahl
// der entfernten Einträge.
//
// Fehlt die Ablage oder enthält sie keine einzige Aufnahme (keine PNG-Datei), während Einträge des Index fehlen, ist
// das fast sicher ein nicht eingehängtes Laufwerk, ein leerer Einhängepunkt oder ein umbenannter Ordner, nicht „alles
// gelöscht“: Dann wird nichts geschrieben und nichts entfernt, es kommt ErrAblageLeer (F-685, wie VerwaisteVorschauen,
// F-270). Fehlt die Indexdatei ganz, während die Ablage Aufnahmen enthält, baut Abgleich den Index neu auf
// (Neuaufbau). Das ist zugleich der Weg, einen verlorenen oder veralteten Index wiederherzustellen: Indexdatei löschen
// oder beiseitelegen, dann gleicht der nächste Start des Editors ab (ADR-0016 Nachtrag 6).
//
// Die Dateiprüfung (ein Lstat je Eintrag) läuft ohne b.mu und ohne die Sperre des Index (F-690): Auf einer langsamen
// Ablage warteten Speichern und Vorschau sonst so lange. Unter beiden wird der Index danach frisch gelesen, und nur die
// als fehlend erkannten Pfade, die auch jetzt noch fehlen, werden gestrichen; was inzwischen dazukam, bleibt (F-686).
func (b *Bibliothek) Abgleich() (_ int, fehler error) {
	defer errprefix.Defer("library", &fehler)
	b.mu.Lock()
	g, err := b.lade()
	b.mu.Unlock()
	if err != nil {
		return 0, err
	}
	if len(g.Eintraege) == 0 && g.Defekt == 0 {
		return 0, b.neuaufbauOhneIndex()
	}
	fehlt := map[string]bool{}
	for _, e := range g.Eintraege {
		if b.fehltInAblage(e.Pfad) { // andere Fehler (Rechte): im Zweifel behalten
			fehlt[e.Pfad] = true
		}
	}
	if len(fehlt) == 0 && g.Defekt == 0 {
		return 0, nil
	}
	if len(fehlt) > 0 && b.ablageOhneAufnahme() {
		return 0, fmt.Errorf("%w: %d Einträge im Index ohne Datei", ErrAblageLeer, len(fehlt))
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	entfernt := 0
	err = b.unterSperre(func() error {
		g, err := b.lade()
		if err != nil {
			return err
		}
		var behalten []Eintrag
		var weg []string
		for _, e := range g.Eintraege {
			if fehlt[e.Pfad] && b.fehltInAblage(e.Pfad) {
				weg = append(weg, e.Pfad)
				continue
			}
			behalten = append(behalten, e)
		}
		if len(weg) == 0 && g.Defekt == 0 {
			return nil
		}
		for _, p := range weg {
			_ = b.entferneVorschau(p) // Vorschau der verschwundenen Aufnahme nicht liegen lassen (F-51)
		}
		entfernt = len(weg)
		return b.schreibeIndex(behalten)
	})
	if err != nil {
		return 0, err
	}
	return entfernt, nil
}

// lstatAblage ist os.Lstat für die Dateiprüfung des Abgleichs; Tests ersetzen es, um zu sehen, unter welchen Sperren
// sie läuft (F-690).
var lstatAblage = os.Lstat

// fehltInAblage meldet, ob die Aufnahme rel (relativ zur Ablage) sicher fehlt (ENOENT). Andere Fehler zählen nicht.
func (b *Bibliothek) fehltInAblage(rel string) bool {
	_, err := lstatAblage(filepath.Join(b.Orte.Ablage, filepath.FromSlash(rel)))
	return errors.Is(err, fs.ErrNotExist)
}

// ablageOhneAufnahme meldet, ob die Ablage fehlt, nicht lesbar ist oder keine einzige PNG-Datei enthält. Die Ablage
// wird vorher aufgelöst (Symlink-Wurzel, F-271); die Suche endet beim ersten Fund.
func (b *Bibliothek) ablageOhneAufnahme() bool {
	wurzel, err := filepath.EvalSymlinks(b.Orte.Ablage)
	if err != nil {
		return true
	}
	gefunden := false
	_ = filepath.WalkDir(wurzel, func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // unlesbarer Teil: weitersuchen
		}
		if d.Type().IsRegular() && strings.EqualFold(filepath.Ext(pfad), ".png") {
			gefunden = true
			return fs.SkipAll
		}
		return nil
	})
	return !gefunden
}

// neuaufbauOhneIndex baut den Index neu auf, wenn die Indexdatei fehlt und die Ablage Aufnahmen enthält (Abgleich,
// ADR-0016 Nachtrag 6). Fehlt beides, bleibt alles, wie es ist (erster Start: nichts anlegen).
func (b *Bibliothek) neuaufbauOhneIndex() error {
	if _, err := os.Lstat(b.Orte.Index); !errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if b.ablageOhneAufnahme() {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	var heilen []heilAuftrag
	err := b.unterSperre(func() error {
		// Ein anderer Prozess kann den Index inzwischen angelegt haben (Ablegen): dann nicht ersetzen.
		if _, err := os.Lstat(b.Orte.Index); !errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		var err error
		_, heilen, err = b.neuaufbau()
		return err
	})
	b.heileAlle(heilen) // nach der Freigabe der Sperre (F-691)
	return err
}

// --- Suchen ---

// Anfrage für Suchen. Leere Angaben schränken nicht ein.
type Anfrage struct {
	Schlagwort string // ein Schlagwort, Groß-/Kleinschreibung egal
	// Teilwort trifft jede Aufnahme, in deren Schlagworten es vorkommt, Groß-/Kleinschreibung egal (Suchfeld der
	// vollen Ansicht, F-671); Schlagwort bleibt der genaue Vergleich (MCP library_search).
	Teilwort string
	Von      time.Time // Erstellt >= Von
	Bis      time.Time // Erstellt < Bis
	// OhneAgenten blendet Aufnahmen über den MCP-Server aus (Eintrag.Agent).
	OhneAgenten bool
}

// Suchen liefert die passenden Aufnahmen, neueste zuerst.
func (b *Bibliothek) Suchen(a Anfrage) (_ []Eintrag, fehler error) {
	defer errprefix.Defer("library", &fehler)
	b.mu.Lock()
	defer b.mu.Unlock()
	g, err := b.lade()
	if err != nil {
		return nil, err
	}
	var treffer []Eintrag
	for _, e := range g.Eintraege {
		if !a.Von.IsZero() && e.Erstellt.Before(a.Von) {
			continue
		}
		if !a.Bis.IsZero() && !e.Erstellt.Before(a.Bis) {
			continue
		}
		if a.Schlagwort != "" && !hatSchlagwort(e, a.Schlagwort) {
			continue
		}
		if a.Teilwort != "" && !hatTeilwort(e, a.Teilwort) {
			continue
		}
		if a.OhneAgenten && e.Agent() {
			continue
		}
		treffer = append(treffer, e)
	}
	sortiere(treffer)
	for i, j := 0, len(treffer)-1; i < j; i, j = i+1, j-1 {
		treffer[i], treffer[j] = treffer[j], treffer[i]
	}
	return treffer, nil
}

// hatTeilwort meldet, ob w (ohne Leerraum an den Enden) in einem Schlagwort von e vorkommt, Groß-/Kleinschreibung
// egal. Verglichen wird mit derselben Faltung wie beim ganzen Schlagwort (strings.EqualFold, F-689): Schluss-Sigma,
// langes s und Kelvin-Zeichen treffen hier genau dann, wenn sie dort treffen. Ein Teilwort nur aus Leerraum gilt als
// keins und trifft alles.
func hatTeilwort(e Eintrag, w string) bool {
	w = falte(strings.TrimSpace(w))
	if w == "" {
		return true
	}
	for _, s := range e.Schlagworte {
		if strings.Contains(falte(s), w) {
			return true
		}
	}
	return false
}

// falte bildet jedes Zeichen auf die kleinste Form seiner einfachen Faltungsklasse ab (unicode.SimpleFold). Zwei
// Texte sind genau dann gleich nach strings.EqualFold, wenn ihre Faltungen gleich sind.
func falte(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))
	for _, r := range s {
		klein := r
		for f := unicode.SimpleFold(r); f != r; f = unicode.SimpleFold(f) {
			klein = min(klein, f)
		}
		sb.WriteRune(klein)
	}
	return sb.String()
}

func hatSchlagwort(e Eintrag, w string) bool {
	w = strings.TrimSpace(w)
	for _, s := range e.Schlagworte {
		if strings.EqualFold(s, w) {
			return true
		}
	}
	return false
}

// SetzeSchlagworte ersetzt die Schlagworte einer Aufnahme: Die Arbeitsdatei wird mit geänderten Metadaten atomar
// neu geschrieben (Original, Objekte und Erstellzeit bleiben), danach folgt der Index. pfad ist relativ zur
// Ablage, wie Eintrag.Pfad. Fremde oder beschädigte Dateien lehnt die Funktion ab (ErrFremd), ebenso einen Pfad, dessen
// Ordner als Verweis aus der Ablage führt (ErrPfad, wie Speichern, F-675).
//
// Hat ein anderer Prozess die Datei zwischen Lesen und Umbenennen ersetzt oder geändert (etwa ein zweites Fenster, das
// eine Schwärzung speichert), wird nicht überschrieben: ErrGeaendert, die Datei behält den neueren Stand (F-673,
// Kernprinzip 2). Verglichen wird der Stand wie bei Speichern (standAus) direkt vor dem Umbenennen.
func (b *Bibliothek) SetzeSchlagworte(pfad string, worte []string) (_ Eintrag, fehler error) {
	defer errprefix.Defer("library", &fehler)
	b.mu.Lock()
	defer b.mu.Unlock()
	e := Eintrag{Pfad: pfad, Breite: 1, Hoehe: 1}
	if !e.gueltig() {
		return Eintrag{}, fmt.Errorf("%w: %q", ErrPfad, pfad)
	}
	voll := filepath.Join(b.Orte.Ablage, filepath.FromSlash(pfad))
	st, err := os.Lstat(voll)
	if err != nil {
		return Eintrag{}, err
	}
	if !st.Mode().IsRegular() {
		return Eintrag{}, fmt.Errorf("%w: %q", ErrPfad, pfad)
	}
	// Auch die Ordner auf dem Weg dürfen nicht aus der Ablage hinausführen (Monatsordner als Verweis, F-675).
	if echt, ok := b.Relativ(voll); !ok || echt != pfad {
		return Eintrag{}, fmt.Errorf("%w: %q", ErrPfad, pfad)
	}
	stand := standAus(st)
	f, err := os.Open(voll)
	if err != nil {
		return Eintrag{}, err
	}
	g, err := fileformat.LeseArbeitsdatei(f)
	if err == nil && !g.Fremd {
		_, err = f.Seek(0, io.SeekStart)
	}
	var sichtbar image.Image
	if err == nil && !g.Fremd {
		sichtbar, err = fileformat.LeseBild(f)
	}
	f.Close()
	if err != nil {
		if fileformat.IstBeschaedigt(err) {
			return Eintrag{}, fmt.Errorf("%w: %v", ErrFremd, err)
		}
		return Eintrag{}, err
	}
	if g.Fremd {
		return Eintrag{}, ErrFremd
	}
	m := g.Meta
	m.Schlagworte = bereinige(worte)
	err = schreibeAtomarGeprueft(voll, func(w io.Writer) error {
		return fileformat.SchreibeArbeitsdatei(w, g.Dokument, sichtbar, m)
	}, func() error {
		// Gelesen wurde der Stand stand; ist die Datei inzwischen eine andere, trüge die neue Fassung den alten Inhalt.
		if fi, err := os.Lstat(voll); err != nil || standAus(fi) != stand {
			return fmt.Errorf("%w: %q", ErrGeaendert, pfad)
		}
		return nil
	})
	if err != nil {
		return Eintrag{}, err
	}
	gr := g.Dokument.Groesse()
	neu := Eintrag{Pfad: pfad, Erstellt: m.Erstellt.UTC(), Breite: gr.X, Hoehe: gr.Y, Schlagworte: m.Schlagworte,
		Quelle: m.Quelle, Bearbeitet: m.Bearbeitet}
	if m.Erstellt.IsZero() {
		neu.Erstellt = st.ModTime().UTC().Truncate(time.Second)
	}
	if err := b.ersetzeImIndex(neu); err != nil {
		return neu, fmt.Errorf("%w: %v", ErrIndex, err)
	}
	return neu, nil
}

// ErzeugeVorschau schreibt das Vorschaubild der Aufnahme rel (Pfad relativ zur Ablage, wie im Index) neu, wenn es
// fehlt, aus dem sichtbaren Bild der Datei, nie aus dem Original (Leiste der letzten Aufnahmen, ADR-0036 Punkt 6). Ein
// Pfad, der aus der Ablage hinausführt, und alles außer einer regulären Datei bleiben unberührt; ein Fehler bleibt still,
// die Leiste zeigt dann weiter das graue Feld.
//
// Es läuft unter b.mu wie Speichern und Entfernen (F-651, Kernprinzip 2): Sonst könnte es eine Datei vor einem
// Speichern mit Schwärzung lesen und sein Bild danach über die neue Vorschau legen, oder die Vorschau einer eben
// entfernten Aufnahme als Waise neu anlegen. Gegen einen zweiten Prozess vergleicht es den Stand der Datei vor dem
// Lesen, vor dem Umbenennen und danach; weicht er ab, wird nicht geschrieben bzw. die eben geschriebene Vorschau
// wieder entfernt.
func (b *Bibliothek) ErzeugeVorschau(rel string) {
	if !(Eintrag{Pfad: rel, Breite: 1, Hoehe: 1}).gueltig() {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	pfad := filepath.Join(b.Orte.Ablage, filepath.FromSlash(rel))
	fi, err := os.Lstat(pfad)
	if err != nil || !fi.Mode().IsRegular() {
		return
	}
	// Auch die Ordner auf dem Weg dürfen nicht aus der Ablage hinausführen (Monatsordner als Symlink).
	if echt, ok := b.Relativ(pfad); !ok || echt != rel {
		return
	}
	b.heileVorschau(pfad, rel, standAus(fi))
}

// heileVorschau schreibt das Vorschaubild einer Arbeitsdatei neu, wenn es fehlt. stand ist der Stand der Datei, wie
// ihn der Aufrufer geprüft hat; gelesen und geschrieben wird nur, solange die Datei diesen Stand behält (F-651).
// Läuft unter b.mu.
func (b *Bibliothek) heileVorschau(pfad, rel string, stand Stand) {
	if _, err := os.Stat(b.VorschauPfad(rel)); err == nil {
		return
	}
	unveraendert := func() bool {
		fi, err := os.Lstat(pfad)
		return err == nil && fi.Mode().IsRegular() && standAus(fi) == stand
	}
	f, err := os.OpenFile(pfad, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return
	}
	defer f.Close()
	if fi, err := f.Stat(); err != nil || !fi.Mode().IsRegular() || standAus(fi) != stand {
		return // zwischen Prüfen und Öffnen ersetzt
	}
	bild, err := fileformat.LeseBild(f)
	if err != nil || !unveraendert() {
		return
	}
	if err := eigenerOrdner(b.Orte.Vorschau); err != nil {
		return
	}
	klein := verkleinere(bild, VorschauSeite)
	err = schreibeAtomarGeprueft(b.VorschauPfad(rel), func(w io.Writer) error { return kodiereVorschau(w, klein) },
		func() error {
			if !unveraendert() {
				return ErrGeaendert
			}
			return nil
		})
	if err == nil && !unveraendert() {
		// Ein anderer Prozess hat die Datei zwischen Prüfen und Umbenennen ersetzt oder entfernt: Die Vorschau zeigte
		// einen alten Stand (etwa vor einer Schwärzung) oder wäre eine Waise.
		_ = b.entferneVorschau(rel)
	}
}
