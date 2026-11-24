// SPDX-License-Identifier: GPL-3.0-or-later

// Befehl testsprung liest die Ausgabe von `go test -json` und macht übersprungene Tests sichtbar (F-714): Er gibt die
// Ausgabe gescheiterter Tests und Pakete aus, listet am Ende jeden übersprungenen Test mit Grund und endet mit Fehler,
// wenn ein Test scheiterte oder ein Test übersprungen wurde, der nicht in der Liste der erlaubten Sprünge steht. So
// bleibt die CI nicht still grün, wenn etwa tesseract oder rsvg-convert fehlt.
//
// Aufruf (bash mit pipefail, in der CI über defaults.run.shell: bash):
//
//	go test -json ./... | go run ./scripts/testsprung -erlaubt scripts/testsprung/erlaubt.txt
//
// Die Liste hat je Zeile `<Importpfad> <Testname>` und einen Grund nach `#`; leere Zeilen und reine Kommentare zählen
// nicht. Ein erlaubter Test deckt seine Untertests mit ab. Ein Eintrag, der in einem Lauf nicht übersprungen wird,
// wird gemeldet (die Liste soll nicht veralten), aber nicht als Fehler.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// ereignis ist eine Zeile von `go test -json` (test2json).
type ereignis struct {
	Action     string
	Package    string
	ImportPath string // Bau-Ereignisse (build-output, build-fail) tragen ImportPath statt Package
	Test       string
	Output     string
}

// sprung ist ein übersprungener Test.
type sprung struct {
	paket, test, grund string
	erlaubt            bool
}

// ergebnis fasst einen Lauf zusammen.
type ergebnis struct {
	bestanden, gescheitert int
	sprünge                []sprung
	fehlerAusgabe          []string // Ausgabe gescheiterter Tests und Pakete, in Reihenfolge
	ungenutzt              []string // erlaubte Sprünge, die nicht vorkamen
	ohneEnde               []string // gestartete Pakete ohne pass/fail/skip: der Lauf wurde abgeschnitten
	ereignisse             int
}

// schluessel eines Tests: Paket und Test.
type schluessel struct{ paket, test string }

// leseErlaubt liest die Liste der erlaubten Sprünge.
func leseErlaubt(r io.Reader) (map[schluessel]bool, error) {
	m := map[schluessel]bool{}
	s := bufio.NewScanner(r)
	for nr := 1; s.Scan(); nr++ {
		z := s.Text()
		if i := strings.IndexByte(z, '#'); i >= 0 {
			z = z[:i]
		}
		f := strings.Fields(z)
		if len(f) == 0 {
			continue
		}
		if len(f) != 2 {
			return nil, fmt.Errorf("testsprung: Zeile %d: erwartet „<Importpfad> <Testname>“", nr)
		}
		m[schluessel{f[0], f[1]}] = true
	}
	return m, s.Err()
}

// oberster liefert den Namen des obersten Tests (ohne Untertests).
func oberster(test string) string {
	if i := strings.IndexByte(test, '/'); i >= 0 {
		return test[:i]
	}
	return test
}

// werte liest die Ereignisse und wertet sie gegen die Liste der erlaubten Sprünge aus.
func werte(r io.Reader, erlaubt map[schluessel]bool) (ergebnis, error) {
	var erg ergebnis
	ausgabe := map[schluessel][]string{}
	genutzt := map[schluessel]bool{}
	offen := map[string]bool{} // gestartete Pakete ohne Ende-Ereignis
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 1<<20), 16<<20)
	for s.Scan() {
		zeile := s.Bytes()
		var e ereignis
		if json.Unmarshal(zeile, &e) != nil || e.Action == "" {
			// Keine JSON-Zeile (etwa eine Bau-Meldung auf stdout): sichtbar weitergeben.
			erg.fehlerAusgabe = append(erg.fehlerAusgabe, string(zeile)+"\n")
			continue
		}
		erg.ereignisse++
		if e.Package == "" {
			e.Package = e.ImportPath
		}
		k := schluessel{e.Package, e.Test}
		if e.Test == "" {
			switch e.Action {
			case "start":
				offen[e.Package] = true
			case "pass", "fail", "skip":
				delete(offen, e.Package)
			}
		}
		switch e.Action {
		case "output", "build-output":
			ausgabe[k] = append(ausgabe[k], e.Output)
		case "pass":
			if e.Test != "" {
				erg.bestanden++
			}
			delete(ausgabe, k)
		case "skip":
			if e.Test != "" {
				ok := erlaubt[schluessel{e.Package, oberster(e.Test)}]
				if ok {
					genutzt[schluessel{e.Package, oberster(e.Test)}] = true
				}
				erg.sprünge = append(erg.sprünge, sprung{paket: e.Package, test: e.Test, grund: grund(ausgabe[k]), erlaubt: ok})
			}
			delete(ausgabe, k)
		case "build-fail":
			erg.fehlerAusgabe = append(erg.fehlerAusgabe, ausgabe[k]...)
			erg.fehlerAusgabe = append(erg.fehlerAusgabe, "testsprung: Bau gescheitert: "+e.Package+"\n")
			delete(ausgabe, k)
		case "fail":
			if e.Test != "" {
				erg.gescheitert++
			}
			erg.fehlerAusgabe = append(erg.fehlerAusgabe, ausgabe[k]...)
			delete(ausgabe, k)
		}
	}
	if err := s.Err(); err != nil {
		return erg, err
	}
	for p := range offen {
		erg.ohneEnde = append(erg.ohneEnde, p)
	}
	sort.Strings(erg.ohneEnde)
	for k := range erlaubt {
		if !genutzt[k] {
			erg.ungenutzt = append(erg.ungenutzt, k.paket+" "+k.test)
		}
	}
	sort.Strings(erg.ungenutzt)
	return erg, nil
}

// grund sucht in der Ausgabe eines übersprungenen Tests die Begründung (die Zeilen nach „--- SKIP“ bzw. die letzte
// Zeile mit Inhalt).
func grund(zeilen []string) string {
	var teile []string
	for _, z := range zeilen {
		t := strings.TrimSpace(z)
		if t == "" || strings.HasPrefix(t, "=== ") || strings.HasPrefix(t, "--- SKIP") {
			continue
		}
		teile = append(teile, t)
	}
	if len(teile) == 0 {
		return "(ohne Grund)"
	}
	return strings.Join(teile, " ")
}

// bericht schreibt die Zusammenfassung und meldet, ob der Lauf als bestanden gilt.
func bericht(w io.Writer, erg ergebnis) bool {
	for _, z := range erg.fehlerAusgabe {
		fmt.Fprint(w, z)
	}
	unerlaubt := 0
	fmt.Fprintf(w, "\ntestsprung: %d bestanden, %d gescheitert, %d übersprungen\n", erg.bestanden, erg.gescheitert, len(erg.sprünge))
	for _, sp := range erg.sprünge {
		marke := "erlaubt  "
		if !sp.erlaubt {
			marke = "UNERLAUBT"
			unerlaubt++
		}
		fmt.Fprintf(w, "  SKIP %s %s %s: %s\n", marke, sp.paket, sp.test, sp.grund)
	}
	for _, u := range erg.ungenutzt {
		fmt.Fprintf(w, "  Hinweis: erlaubter Sprung kam nicht vor: %s\n", u)
	}
	for _, p := range erg.ohneEnde {
		fmt.Fprintf(w, "testsprung: Paket %s ohne Ende (Lauf abgeschnitten?)\n", p)
	}
	ok := erg.gescheitert == 0 && unerlaubt == 0 && erg.ereignisse > 0 && len(erg.fehlerAusgabe) == 0 && len(erg.ohneEnde) == 0
	switch {
	case erg.ereignisse == 0:
		fmt.Fprintln(w, "testsprung: keine Ereignisse gelesen (lief go test mit -json?)")
	case unerlaubt > 0:
		fmt.Fprintf(w, "testsprung: %d unerlaubte Sprünge — Werkzeug in der CI installieren oder mit Grund in die Liste aufnehmen\n", unerlaubt)
	}
	return ok
}

func main() {
	pfad := flag.String("erlaubt", "", "Datei mit den erlaubten Sprüngen")
	flag.Parse()
	erlaubt := map[schluessel]bool{}
	if *pfad != "" {
		f, err := os.Open(*pfad)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		erlaubt, err = leseErlaubt(f)
		f.Close()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	erg, err := werte(os.Stdin, erlaubt)
	if err != nil {
		fmt.Fprintln(os.Stderr, "testsprung:", err)
		os.Exit(2)
	}
	if !bericht(os.Stdout, erg) {
		os.Exit(1)
	}
}
