// SPDX-License-Identifier: GPL-3.0-or-later

package i18n

import (
	"strings"
	"testing"
	"testing/fstest"
)

func probeKatalog(t *testing.T) *Katalog {
	t.Helper()
	e, err := LesePO(strings.NewReader(probePO))
	if err != nil {
		t.Fatal(err)
	}
	k, err := NeuerKatalog("de", e)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestKatalog(t *testing.T) {
	k := probeKatalog(t)
	for _, f := range []struct{ got, want string }{
		{k.T("Save"), "Speichern"},
		{k.T("Missing"), "Missing"},      // fehlt → Ausgangstext
		{k.T("Close"), "Close"},          // unscharf → nicht verwendet
		{k.T("Empty"), "Empty"},          // leer → Ausgangstext
		{k.T("Old"), "Old"},              // veraltet → nicht geladen
		{k.T("Open"), "Open"},            // nur mit Kontext übersetzt
		{k.TK("menu", "Open"), "Öffnen"}, // mit Kontext
		{k.TK("other", "Open"), "Open"},  // anderer Kontext
		{k.N("%d image", "%d images", 1), "%d Bild"},
		{k.N("%d image", "%d images", 0), "%d Bilder"},
		{k.N("%d image", "%d images", 5), "%d Bilder"},
		{k.N("%d file", "%d files", 1), "%d file"}, // fehlt → englische Regel
		{k.N("%d file", "%d files", 2), "%d files"},
	} {
		if f.got != f.want {
			t.Errorf("got %q, want %q", f.got, f.want)
		}
	}
}

func TestKatalogNil(t *testing.T) {
	var k *Katalog
	if k.T("a") != "a" || k.TK("c", "a") != "a" || k.N("a", "b", 1) != "a" || k.N("a", "b", 2) != "b" {
		t.Error("nil-Katalog übersetzt")
	}
}

func TestKatalogMehrzahlAusKopf(t *testing.T) {
	// Französische Regel: 0 ist Einzahl — beweist, dass die Regel aus dem Kopf gilt, nicht die englische.
	po := "msgid \"\"\nmsgstr \"Plural-Forms: nplurals=2; plural=(n > 1);\\n\"\n\n" +
		"msgid \"%d image\"\nmsgid_plural \"%d images\"\nmsgstr[0] \"%d image FR\"\nmsgstr[1] \"%d images FR\"\n"
	e, err := LesePO(strings.NewReader(po))
	if err != nil {
		t.Fatal(err)
	}
	k, err := NeuerKatalog("fr", e)
	if err != nil {
		t.Fatal(err)
	}
	if got := k.N("%d image", "%d images", 0); got != "%d image FR" {
		t.Errorf("N(0) = %q", got)
	}
	if _, err := NeuerKatalog("x", []Eintrag{{Texte: []string{"Plural-Forms: kaputt\n"}}}); err == nil {
		t.Error("kaputte Plural-Forms ohne Fehler")
	}
}

func TestLadeUndStarte(t *testing.T) {
	dateien := fstest.MapFS{
		"de.po":      {Data: []byte(probePO)},
		"kaputt.po":  {Data: []byte("msgid \"a\"\n")},
		"readme.txt": {Data: []byte("x")},
	}
	sprachen, err := Sprachen(dateien)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(sprachen, ",") != "de,kaputt" {
		t.Errorf("Sprachen = %v", sprachen)
	}
	if k, err := Lade(dateien, "en"); err != nil || k.T("Save") != "Save" {
		t.Errorf("Lade(en) = %v, %v", k, err)
	}
	if _, err := Lade(dateien, "kaputt"); err == nil {
		t.Error("kaputte Datei ohne Fehler")
	}
	if _, err := Lade(dateien, "fr"); err == nil {
		t.Error("fehlende Datei ohne Fehler")
	}

	// Starte liest die echte Umgebung; der Test setzt sie selbst (t.Setenv stellt sie danach wieder her).
	for _, v := range []string{"LANGUAGE", "LC_ALL", "LC_MESSAGES"} {
		t.Setenv(v, "")
	}
	t.Setenv("LANG", "de_DE.UTF-8")
	defer Setze(nil)
	if s, err := Starte(dateien, ""); err != nil || s != "de" || T("Save") != "Speichern" || TK("menu", "Open") != "Öffnen" ||
		N("%d image", "%d images", 2) != "%d Bilder" {
		t.Errorf("Starte = %q, %v; T = %q", s, err, T("Save"))
	}
	// Ladefehler: Englisch bleibt aktiv, Fehler kommt zurück.
	if s, err := Starte(dateien, "kaputt"); err == nil || s != "en" || T("Save") != "Save" {
		t.Errorf("Starte(kaputt) = %q, %v; T = %q", s, err, T("Save"))
	}
}
