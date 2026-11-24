// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const (
	modulPfad    = "github.com/Andiministrator/schnapp"
	settingsPfad = modulPfad + "/internal/settings"
	uiPfad       = modulPfad + "/internal/ui"
)

// ADR-0034 Punkt 6: `schnapp mcp` bekommt keinen Weg, die Einstellungen (und damit die Einwilligungen der Agenten) zu
// schreiben. Das Programm ist ein Binary: Der Schreiber steckt über die Einstellungstafel des Editors (Paket ui) mit
// darin, ein Verbot über die Abhängigkeiten der Pakete wäre also wirkungslos. Geprüft wird deshalb der Quelltext:
//
//  1. internal/mcp (Protokoll) bindet weder settings noch ui ein.
//  2. Von den Funktionen in mcp.go aus ist im Paket main (Aufrufgraph über die Namen, überschätzt: jeder Bezeichner,
//     der eine Funktion oder Methode des Pakets nennt, zählt als Aufruf) kein settings.Schreibe… erreichbar, und auch
//     nicht ui.Editor — nur der Editor öffnet die Tafel, die schreibt.
//  3. Gegenprobe: Von main aus ist ui.Editor erreichbar (sonst prüfte 2 nichts).
//  4. Im Paket ui ruft nur die Tafel (einstellungen.go) den Schreiber (und schnellstile.go SchreibeStile für
//     styles.ini), und ui.Bestaetige (der Dialog des MCP-Servers) steht in einer Datei ohne Bezug auf die Tafel.
//  5. Über alle Pakete des Moduls (F-630): Ein Selektor auf das Paket …/internal/settings, dessen Name mit „Schreibe“
//     beginnt, steht nur in internal/ui/einstellungen.go und (nur SchreibeStile) internal/ui/schnellstile.go; einen
//     Punkt-Import des Pakets gibt es nicht.
//
// Die Pakete settings und ui erkennt der Test an den Importen jeder Datei (importNamen), nicht am Bezeichner im
// Quelltext: Ein Import unter anderem Namen fällt genauso auf.
func TestMCPErreichtDenSchreiberNicht(t *testing.T) {
	// 1.
	for _, d := range goDateien(t, filepath.Join("..", "..", "internal", "mcp")) {
		f, err := parser.ParseFile(token.NewFileSet(), d, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, im := range f.Imports {
			p, _ := strconv.Unquote(im.Path.Value)
			if strings.HasSuffix(p, "/internal/settings") || strings.HasSuffix(p, "/internal/ui") {
				t.Errorf("%s bindet %s ein", filepath.Base(d), p)
			}
		}
	}

	// 2. und 3.: Aufrufgraph im Paket main.
	knoten := map[string][]ast.Node{} // Name einer Funktion, Methode oder Paketvariable → Deklarationen
	inMCP := map[string]bool{}
	paketVon := map[ast.Node]map[string]string{} // Deklaration → Importnamen ihrer Datei (Name → "settings"/"ui")
	for _, d := range goDateien(t, ".") {
		f, err := parser.ParseFile(token.NewFileSet(), d, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		namen := map[string]string{}
		for n, p := range importNamen(f) {
			switch p {
			case settingsPfad:
				namen[n] = "settings"
			case uiPfad:
				namen[n] = "ui"
			}
		}
		for _, dekl := range f.Decls {
			switch dk := dekl.(type) {
			case *ast.FuncDecl:
				knoten[dk.Name.Name] = append(knoten[dk.Name.Name], dk)
				paketVon[dk] = namen
				if filepath.Base(d) == "mcp.go" {
					inMCP[dk.Name.Name] = true
				}
			case *ast.GenDecl:
				for _, s := range dk.Specs {
					if vs, ok := s.(*ast.ValueSpec); ok {
						paketVon[vs] = namen
						for _, n := range vs.Names {
							knoten[n.Name] = append(knoten[n.Name], vs)
						}
					}
				}
			}
		}
	}
	erreichbar := func(start []string) (map[string]bool, map[string]bool) {
		gesehen, fremd := map[string]bool{}, map[string]bool{}
		offen := slices.Clone(start)
		for len(offen) > 0 {
			n := offen[len(offen)-1]
			offen = offen[:len(offen)-1]
			if gesehen[n] {
				continue
			}
			gesehen[n] = true
			for _, k := range knoten[n] {
				namen := paketVon[k]
				ast.Inspect(k, func(x ast.Node) bool {
					switch x := x.(type) {
					case *ast.SelectorExpr:
						if p, ok := x.X.(*ast.Ident); ok && namen[p.Name] != "" {
							fremd[namen[p.Name]+"."+x.Sel.Name] = true
						}
						if _, ok := knoten[x.Sel.Name]; ok { // Methode über einen Empfänger
							offen = append(offen, x.Sel.Name)
						}
					case *ast.Ident:
						if _, ok := knoten[x.Name]; ok {
							offen = append(offen, x.Name)
						}
					}
					return true
				})
			}
		}
		return gesehen, fremd
	}
	var start []string
	for n := range inMCP {
		start = append(start, n)
	}
	if len(start) < 5 {
		t.Fatalf("mcp.go: nur %d Funktionen gefunden", len(start))
	}
	_, fremd := erreichbar(start)
	for f := range fremd {
		if strings.HasPrefix(f, "settings.Schreibe") || f == "ui.Editor" {
			t.Errorf("von mcp.go aus erreichbar: %s", f)
		}
	}
	if !fremd["settings.LadeDatei"] && !fremd["settings.Lade"] {
		t.Error("Gegenprobe: mcp.go liest die Einstellungen nicht (Graph unvollständig?)")
	}
	if _, f := erreichbar([]string{"main"}); !f["ui.Editor"] {
		t.Error("Gegenprobe: ui.Editor ist auch von main aus nicht erreichbar (Graph unvollständig?)")
	}

	// 4.
	ui := filepath.Join("..", "..", "internal", "ui")
	schreiber := 0
	for _, d := range goDateien(t, ui) {
		f, err := parser.ParseFile(token.NewFileSet(), d, nil, 0) // ohne Kommentare
		if err != nil {
			t.Fatal(err)
		}
		bestaetige, tafel := false, false
		namen := importNamen(f)
		ast.Inspect(f, func(x ast.Node) bool {
			switch x := x.(type) {
			case *ast.SelectorExpr:
				if p, ok := x.X.(*ast.Ident); ok && namen[p.Name] == settingsPfad && strings.HasPrefix(x.Sel.Name, "Schreibe") {
					schreiber++
					// Ausnahme: styles.ini, die eigenen Schnellstile des Editors (ADR-0037 Punkt 6): keine Einstellung und
					// keine Einwilligung eines Agenten, nur Farben und Maße.
					if filepath.Base(d) != "einstellungen.go" && !(filepath.Base(d) == "schnellstile.go" && x.Sel.Name == "SchreibeStile") {
						t.Errorf("%s ruft settings.%s", filepath.Base(d), x.Sel.Name)
					}
				}
				if x.Sel.Name == "einst" {
					tafel = true
				}
			case *ast.FuncDecl:
				bestaetige = bestaetige || (x.Recv == nil && x.Name.Name == "Bestaetige")
			case *ast.Ident:
				tafel = tafel || x.Name == "tafel"
			}
			return true
		})
		if bestaetige && tafel {
			t.Errorf("%s: Bestaetige steht neben der Einstellungstafel", filepath.Base(d))
		}
	}
	if schreiber == 0 {
		t.Error("Gegenprobe: die Tafel in internal/ui ruft den Schreiber gar nicht")
	}

	// 5.
	erlaubt := 0
	for _, a := range schreiberAufrufe(t, filepath.Join("..", "..")) {
		switch {
		case a.datei == "internal/ui/einstellungen.go",
			a.datei == "internal/ui/schnellstile.go" && a.name == "SchreibeStile":
			erlaubt++
		default:
			t.Errorf("%s ruft settings.%s (nur die Einstellungstafel schreibt, ADR-0034 Punkt 6)", a.datei, a.name)
		}
	}
	if erlaubt == 0 {
		t.Error("Gegenprobe: kein erlaubter Aufruf des Schreibers gefunden (Suche über die Pakete unvollständig?)")
	}
}

// goDateien sind die Go-Dateien eines Ordners ohne Tests.
func goDateien(t *testing.T, ordner string) []string {
	t.Helper()
	m, err := filepath.Glob(filepath.Join(ordner, "*.go"))
	if err != nil || len(m) == 0 {
		t.Fatalf("%s: %v", ordner, err)
	}
	return slices.DeleteFunc(m, func(s string) bool { return strings.HasSuffix(s, "_test.go") })
}

// importNamen ordnet jedem Namen, unter dem eine Datei ein Paket einbindet, den Importpfad zu: ohne Umbenennung der
// letzte Pfadteil (die Pakete des Moduls heißen wie ihr Ordner), sonst der gewählte Name; „.“ und „_“ stehen als
// solche darin.
func importNamen(f *ast.File) map[string]string {
	m := map[string]string{}
	for _, im := range f.Imports {
		p, _ := strconv.Unquote(im.Path.Value)
		n := p[strings.LastIndex(p, "/")+1:]
		if im.Name != nil {
			n = im.Name.Name
		}
		m[n] = p
	}
	return m
}

// schreiberAufruf ist ein Selektor settings.Schreibe… in einer Datei (Pfad relativ zur Wurzel, mit /).
type schreiberAufruf struct{ datei, name string }

// schreiberAufrufe sucht in allen Go-Dateien des Moduls unter wurzel (ohne Tests, ohne Ordner mit eigenem go.mod wie
// draw/ und tmp/, ohne testdata und versteckte Ordner wie .claude/) die Selektoren auf das Paket settings, deren Name
// mit „Schreibe“ beginnt (F-630). Das Paket wird über den Import aufgelöst, nicht über den Bezeichner; ein
// Punkt-Import von settings zählt als Aufruf „.“. Das Paket settings selbst ruft sich ohne Selektor und zählt nicht.
func schreiberAufrufe(t *testing.T, wurzel string) []schreiberAufruf {
	t.Helper()
	var r []schreiberAufruf
	dateien := 0
	err := filepath.WalkDir(wurzel, func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if pfad == wurzel {
				return nil
			}
			if n := d.Name(); strings.HasPrefix(n, ".") || n == "testdata" {
				return filepath.SkipDir
			}
			if _, e := os.Stat(filepath.Join(pfad, "go.mod")); e == nil {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(wurzel, pfad)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		f, err := parser.ParseFile(token.NewFileSet(), pfad, nil, 0)
		if err != nil {
			return err
		}
		dateien++
		namen := importNamen(f)
		if namen["."] == settingsPfad {
			r = append(r, schreiberAufruf{rel, "."})
		}
		ast.Inspect(f, func(x ast.Node) bool {
			if s, ok := x.(*ast.SelectorExpr); ok {
				if id, ok := s.X.(*ast.Ident); ok && namen[id.Name] == settingsPfad && strings.HasPrefix(s.Sel.Name, "Schreibe") {
					r = append(r, schreiberAufruf{rel, s.Sel.Name})
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if dateien < 50 {
		t.Fatalf("%s: nur %d Go-Dateien gefunden", wurzel, dateien)
	}
	return r
}

// TestSchreiberAufrufeFindet: Die Suche aus Punkt 5 findet einen Aufruf in einem beliebigen Paket, unter anderem
// Importnamen, einen neuen Namen mit Präfix und einen Punkt-Import; ein fremdes Paket namens settings, Tests, Ordner
// mit eigenem go.mod und versteckte Ordner zählen nicht.
func TestSchreiberAufrufeFindet(t *testing.T) {
	w := t.TempDir()
	schreib := func(rel, inhalt string) {
		t.Helper()
		p := filepath.Join(w, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(inhalt), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for i := range 50 { // Mindestzahl der Dateien
		schreib("fuell/f"+strconv.Itoa(i)+".go", "package fuell\n")
	}
	imp := func(name string) string { return "import " + name + " " + strconv.Quote(settingsPfad) + "\n\n" }
	schreib("internal/canvas/a.go", "package canvas\n\n"+imp("")+"var _ = settings.Schreibe\n")
	schreib("internal/annotate/b.go", "package annotate\n\n"+imp("einst")+"var _ = einst.SchreibeNeu\n")
	schreib("internal/x/c.go", "package x\n\n"+imp(".")+"var _ = Lade\n")
	schreib("internal/y/d.go", "package y\n\nimport settings \"example.org/anders/settings\"\n\nvar _ = settings.Schreibe\n")
	schreib("internal/y/d_test.go", "package y\n\n"+imp("")+"var _ = settings.Schreibe\n")
	schreib("draw/go.mod", "module beispiel\n")
	schreib("draw/e.go", "package draw\n\n"+imp("")+"var _ = settings.Schreibe\n")
	schreib(".claude/f.go", "package z\n\n"+imp("")+"var _ = settings.Schreibe\n")
	schreib("internal/z/testdata/g.go", "package z\n\n"+imp("")+"var _ = settings.Schreibe\n")
	got := map[string]string{}
	for _, a := range schreiberAufrufe(t, w) {
		got[a.datei] = a.name
	}
	want := map[string]string{"internal/canvas/a.go": "Schreibe", "internal/annotate/b.go": "SchreibeNeu", "internal/x/c.go": "."}
	if len(got) != len(want) {
		t.Errorf("gefunden %v, erwartet %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: %q, erwartet %q", k, got[k], v)
		}
	}
}
