// SPDX-License-Identifier: GPL-3.0-or-later

package portal

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

// attrappe ersetzt D-Bus: kein echter Bus, kein Portal.
type attrappe struct {
	mu          sync.Mutex // schützt abos (ein Test liest aus einer zweiten Goroutine)
	name        string
	abos        map[dbus.ObjectPath]chan Antwort
	ablauf      []string // Reihenfolge der Aufrufe
	handle      dbus.ObjectPath
	fehler      error
	antwort     *Antwort // wird noch während SaveFile an die Abonnenten geschickt (Portal antwortet schnell)
	letzteOpt   map[string]dbus.Variant
	letzteElt   string
	geschlossen []dbus.ObjectPath
}

func (a *attrappe) Name() string { return a.name }

func (a *attrappe) Abonniere(pfad dbus.ObjectPath) (<-chan Antwort, func(), error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ablauf = append(a.ablauf, "abonniere "+string(pfad))
	if a.abos == nil {
		a.abos = map[dbus.ObjectPath]chan Antwort{}
	}
	c := make(chan Antwort, 1)
	a.abos[pfad] = c
	return c, func() { a.ablauf = append(a.ablauf, "abgemeldet") }, nil
}

func (a *attrappe) SaveFile(_ context.Context, elt, _ string, opt map[string]dbus.Variant) (dbus.ObjectPath, error) {
	a.ablauf = append(a.ablauf, "SaveFile")
	a.letzteOpt, a.letzteElt = opt, elt
	if a.fehler != nil {
		return "", a.fehler
	}
	token := opt["handle_token"].Value().(string)
	erwartet := RequestPfad(a.name, token)
	if a.antwort != nil {
		// Antwort kommt, bevor der Aufruf zurückkehrt: Ohne vorherige Anmeldung ginge sie verloren.
		a.mu.Lock()
		c, ok := a.abos[erwartet]
		a.mu.Unlock()
		if ok {
			c <- *a.antwort
		}
	}
	if a.handle != "" {
		return a.handle, nil
	}
	return erwartet, nil
}

func (a *attrappe) Schliesse(p dbus.ObjectPath) { a.geschlossen = append(a.geschlossen, p) }
func (a *attrappe) Close() error                { return nil }

func pngFilter() Filter {
	return Filter{Name: "PNG", Globs: []string{"*.png"}, MIMEs: []string{"image/png"}}
}
func jpgFilter() Filter {
	return Filter{Name: "JPEG", Globs: []string{"*.jpg"}, MIMEs: []string{"image/jpeg"}}
}

func anfrage() Anfrage {
	return Anfrage{Titel: "Export", Dateiname: "bild.png", Filter: []Filter{pngFilter(), jpgFilter()}}
}

func neuerDialog(a *attrappe) *Dialog {
	return &Dialog{t: a, token: func() string { return "tok_1" }}
}

func okAntwort(uri, filter string) *Antwort {
	erg := map[string]dbus.Variant{"uris": dbus.MakeVariant([]string{uri})}
	if filter != "" {
		erg["current_filter"] = dbus.MakeVariant(filterD{Name: filter, Muster: []musterD{{0, "*.x"}}})
	}
	return &Antwort{Code: 0, Ergebnisse: erg}
}

func mitFrist(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestRequestPfad(t *testing.T) {
	got := RequestPfad(":1.42", "tok")
	if want := dbus.ObjectPath("/org/freedesktop/portal/desktop/request/1_42/tok"); got != want {
		t.Errorf("Pfad %q, soll %q", got, want)
	}
}

func TestOptionen(t *testing.T) {
	opt := baueOptionen("tok", anfrage())
	if opt["handle_token"].Value() != "tok" || opt["modal"].Value() != true || opt["current_name"].Value() != "bild.png" {
		t.Errorf("Optionen: %v", opt)
	}
	if s := opt["filters"].Signature().String(); s != "a(sa(us))" {
		t.Errorf("filters hat Signatur %q", s)
	}
	if s := opt["current_filter"].Signature().String(); s != "(sa(us))" {
		t.Errorf("current_filter hat Signatur %q", s)
	}
	fl := opt["filters"].Value().([]filterD)
	if len(fl) != 2 || fl[0].Name != "PNG" || fl[1].Name != "JPEG" {
		t.Errorf("Filter: %v", fl)
	}
	want := []musterD{{0, "*.png"}, {1, "image/png"}}
	if !reflect.DeepEqual(fl[0].Muster, want) {
		t.Errorf("Muster %v, soll %v", fl[0].Muster, want)
	}
	if opt["current_filter"].Value().(filterD).Name != "PNG" {
		t.Error("current_filter ist nicht der erste Filter")
	}
}

// Die Anmeldung für Request.Response steht vor SaveFile, und die Antwort, die schon während des Aufrufs kommt, geht nicht
// verloren.
func TestAbonnementVorSaveFile(t *testing.T) {
	a := &attrappe{name: ":1.7", antwort: okAntwort("file:///x/a.png", "PNG")}
	erg, err := neuerDialog(a).Speichern(mitFrist(t), anfrage())
	if err != nil {
		t.Fatal(err)
	}
	if len(a.ablauf) < 2 || a.ablauf[0] != "abonniere /org/freedesktop/portal/desktop/request/1_7/tok_1" || a.ablauf[1] != "SaveFile" {
		t.Errorf("Reihenfolge: %v", a.ablauf)
	}
	if erg.URI != "file:///x/a.png" || erg.Filter != 0 {
		t.Errorf("Ergebnis %+v", erg)
	}
	if a.letzteElt != "" {
		t.Errorf("Elternfenster %q, soll leer", a.letzteElt)
	}
}

func TestFilterAusAntwort(t *testing.T) {
	a := &attrappe{name: ":1.7", antwort: okAntwort("file:///x/a", "JPEG")}
	erg, err := neuerDialog(a).Speichern(mitFrist(t), anfrage())
	if err != nil || erg.Filter != 1 {
		t.Errorf("Filter %d, Fehler %v", erg.Filter, err)
	}
	// Unbekannter Filtername: -1.
	a = &attrappe{name: ":1.7", antwort: okAntwort("file:///x/a", "Unbekannt")}
	if erg, err = neuerDialog(a).Speichern(mitFrist(t), anfrage()); err != nil || erg.Filter != -1 {
		t.Errorf("Filter %d, Fehler %v", erg.Filter, err)
	}
	// Ohne current_filter: -1.
	a = &attrappe{name: ":1.7", antwort: okAntwort("file:///x/a", "")}
	if erg, err = neuerDialog(a).Speichern(mitFrist(t), anfrage()); err != nil || erg.Filter != -1 {
		t.Errorf("Filter %d, Fehler %v", erg.Filter, err)
	}
}

// Der Bus liefert Strukturen als []interface{} (nicht als filterD): auch das muss ausgewertet werden.
func TestFilterAlsBusStruktur(t *testing.T) {
	roh := []interface{}{"JPEG", [][]interface{}{{uint32(0), "*.jpg"}}}
	r := Antwort{Ergebnisse: map[string]dbus.Variant{
		"uris":           dbus.MakeVariant([]string{"file:///x/a"}),
		"current_filter": dbus.MakeVariant(roh),
	}}
	erg, err := auswerten(r, anfrage())
	if err != nil || erg.Filter != 1 {
		t.Errorf("Filter %d, Fehler %v", erg.Filter, err)
	}
}

func TestAbbruchUndFehlerantwort(t *testing.T) {
	a := &attrappe{name: ":1.7", antwort: &Antwort{Code: 1}}
	if _, err := neuerDialog(a).Speichern(mitFrist(t), anfrage()); !errors.Is(err, ErrAbgebrochen) {
		t.Errorf("Antwort 1: Fehler %v, soll ErrAbgebrochen", err)
	}
	a = &attrappe{name: ":1.7", antwort: &Antwort{Code: 2}}
	if _, err := neuerDialog(a).Speichern(mitFrist(t), anfrage()); err == nil || errors.Is(err, ErrAbgebrochen) {
		t.Errorf("Antwort 2: Fehler %v, soll ein echter Fehler sein", err)
	}
	// Antwort 0 ohne Adresse ist ein Fehler.
	a = &attrappe{name: ":1.7", antwort: &Antwort{Code: 0, Ergebnisse: map[string]dbus.Variant{}}}
	if _, err := neuerDialog(a).Speichern(mitFrist(t), anfrage()); err == nil {
		t.Error("Antwort 0 ohne uris soll scheitern")
	}
}

func TestSaveFileFehler(t *testing.T) {
	a := &attrappe{name: ":1.7", fehler: ErrKeinPortal}
	if _, err := neuerDialog(a).Speichern(mitFrist(t), anfrage()); !errors.Is(err, ErrKeinPortal) {
		t.Errorf("Fehler %v", err)
	}
	if a.ablauf[len(a.ablauf)-1] != "abgemeldet" {
		t.Errorf("Abmelden fehlt: %v", a.ablauf)
	}
}

// Wird der Kontext beendet, bevor eine Antwort kommt, schließt der Dialog die Anfrage und kehrt zurück.
func TestKontextEnde(t *testing.T) {
	a := &attrappe{name: ":1.7"}
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(20 * time.Millisecond); cancel() }()
	_, err := neuerDialog(a).Speichern(ctx, anfrage())
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Fehler %v", err)
	}
	if len(a.geschlossen) != 1 {
		t.Errorf("Request.Close nicht gerufen: %v", a.geschlossen)
	}
}

// Ältere Portale nennen einen anderen Pfad als erwartet: dort wird zusätzlich gelauscht.
func TestAndererHandle(t *testing.T) {
	a := &attrappe{name: ":1.7", handle: "/org/freedesktop/portal/desktop/request/anders"}
	go func() {
		for i := 0; i < 100; i++ {
			time.Sleep(10 * time.Millisecond)
			if c, ok := a.abosGeprueft("/org/freedesktop/portal/desktop/request/anders"); ok {
				c <- *okAntwort("file:///x/a.png", "PNG")
				return
			}
		}
	}()
	erg, err := neuerDialog(a).Speichern(mitFrist(t), anfrage())
	if err != nil || erg.URI != "file:///x/a.png" {
		t.Errorf("Ergebnis %+v, Fehler %v", erg, err)
	}
}

func (a *attrappe) abosGeprueft(p dbus.ObjectPath) (chan Antwort, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	c, ok := a.abos[p]
	return c, ok
}

func TestURIZuPfad(t *testing.T) {
	ok := map[string]string{
		"file:///home/x/a.png":           "/home/x/a.png",
		"file:///home/x/a%20b%C3%A4.png": "/home/x/a bä.png",
		"file://localhost/tmp/a.svg":     "/tmp/a.svg",
		"FILE:///tmp/a.jpg":              "/tmp/a.jpg",
		"file:///tmp/a%23b%3F.png":       "/tmp/a#b?.png",
	}
	for uri, want := range ok {
		if got, err := URIZuPfad(uri); err != nil || got != want {
			t.Errorf("%q: %q, %v; soll %q", uri, got, err, want)
		}
	}
	for _, uri := range []string{"http://example.org/a.png", "smb://srv/frei/a.png", "file://fremd/tmp/a.png", "file:///", "file://", "a.png", "",
		"file:///tmp/ordner/", "file:///tmp/a%00b.png", "/tmp/a.png"} {
		if got, err := URIZuPfad(uri); err == nil {
			t.Errorf("%q wurde angenommen: %q", uri, got)
		}
	}
}

func signalAn(pfad dbus.ObjectPath, sender string) *dbus.Signal {
	return &dbus.Signal{
		Sender: sender, Path: pfad, Name: schnittstelleReq + "." + signalAntwort,
		Body: []interface{}{uint32(0), map[string]dbus.Variant{}},
	}
}

// Der Signalfilter nimmt nur passende Antworten des Portals an; ein geschlossener Kanal endet mit ErrVerbindung (A-2, A-7).
func TestWertSignal(t *testing.T) {
	pfad := dbus.ObjectPath("/org/freedesktop/portal/desktop/request/1_7/tok")
	pruefe := absenderPruefung(":1.99")

	if r, art := wertSignal(nil, false, pfad, pruefe); art != signalEnde || !errors.Is(r.Err, ErrVerbindung) {
		t.Errorf("geschlossener Kanal: %v, %v", art, r.Err)
	}
	// Ein geschlossener Kanal liefert nil mit ok=false; nil mit ok=true wird verworfen, nicht als Ende gewertet.
	if _, art := wertSignal(nil, true, pfad, pruefe); art != signalVerwerfen {
		t.Errorf("nil-Signal: %v", art)
	}
	if _, art := wertSignal(signalAn(pfad, dienst), true, pfad, pruefe); art != signalPassend {
		t.Errorf("Name des Dienstes: %v", art)
	}
	if _, art := wertSignal(signalAn(pfad, ":1.99"), true, pfad, pruefe); art != signalPassend {
		t.Errorf("Besitzer des Dienstes: %v", art)
	}
	if _, art := wertSignal(signalAn(pfad, ":1.5"), true, pfad, pruefe); art != signalVerwerfen {
		t.Errorf("fremder Absender: %v", art)
	}
	// Besitzer unbekannt: nur der Name zählt.
	if _, art := wertSignal(signalAn(pfad, ":1.99"), true, pfad, absenderPruefung("")); art != signalVerwerfen {
		t.Errorf("ohne Besitzer: %v", art)
	}
	if _, art := wertSignal(signalAn("/anderer/pfad", dienst), true, pfad, pruefe); art != signalVerwerfen {
		t.Errorf("anderer Pfad: %v", art)
	}
	falsch := signalAn(pfad, dienst)
	falsch.Body = []interface{}{"kein uint32", map[string]dbus.Variant{}}
	if _, art := wertSignal(falsch, true, pfad, pruefe); art != signalVerwerfen {
		t.Errorf("falscher Körper: %v", art)
	}
	falsch = signalAn(pfad, dienst)
	falsch.Name = "x.y.Z"
	if _, art := wertSignal(falsch, true, pfad, pruefe); art != signalVerwerfen {
		t.Errorf("anderer Name: %v", art)
	}
}

// Meldet der Transport einen Fehler statt einer Antwort (Verbindung getrennt), kehrt Speichern mit diesem Fehler zurück.
func TestVerbindungGetrennt(t *testing.T) {
	a := &attrappe{name: ":1.7", antwort: &Antwort{Err: ErrVerbindung}}
	_, err := neuerDialog(a).Speichern(mitFrist(t), anfrage())
	if !errors.Is(err, ErrVerbindung) {
		t.Errorf("Fehler %v, soll ErrVerbindung", err)
	}
}

// Scheitert SaveFile, weil der Kontext endete, wird die Anfrage unter dem erwarteten Pfad geschlossen (A-8).
func TestKontextEndeWaehrendSaveFile(t *testing.T) {
	a := &attrappe{name: ":1.7", fehler: context.Canceled}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := neuerDialog(a).Speichern(ctx, anfrage())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Fehler %v", err)
	}
	soll := RequestPfad(":1.7", "tok_1")
	if len(a.geschlossen) != 1 || a.geschlossen[0] != soll {
		t.Errorf("geschlossen: %v, soll [%s]", a.geschlossen, soll)
	}
}
