// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// ADR-0034 Punkt 6: `schnapp mcp` bekommt keinen Weg, die Einstellungen (und damit die Einwilligungen der Agenten) zu
// schreiben. Das Programm ist ein Binary: Der Schreiber steckt über die Einstellungstafel des Editors (Paket ui) mit
// darin, ein Verbot über die Abhängigkeiten der Pakete wäre also wirkungslos. Geprüft wird deshalb der Quelltext:
//
//  1. internal/mcp (Protokoll) bindet weder settings noch ui ein.
//  2. Von den Funktionen in mcp.go aus ist im Paket main (Aufrufgraph über die Namen, überschätzt: jeder Bezeichner,
//     der eine Funktion oder Methode des Pakets nennt, zählt als Aufruf) weder settings.Schreibe noch
//     settings.SchreibeDatei erreichbar, und auch nicht ui.Editor — nur der Editor öffnet die Tafel, die schreibt.
//  3. Gegenprobe: Von main aus ist ui.Editor erreichbar (sonst prüfte 2 nichts).
//  4. Im Paket ui ruft nur die Tafel (einstellungen.go) den Schreiber (und schnellstile.go SchreibeStile für styles.ini), und ui.Bestaetige (der Dialog des MCP-Servers)
//     steht in einer Datei ohne Bezug auf die Tafel.
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
	for _, d := range goDateien(t, ".") {
		f, err := parser.ParseFile(token.NewFileSet(), d, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, dekl := range f.Decls {
			switch dk := dekl.(type) {
			case *ast.FuncDecl:
				knoten[dk.Name.Name] = append(knoten[dk.Name.Name], dk)
				if filepath.Base(d) == "mcp.go" {
					inMCP[dk.Name.Name] = true
				}
			case *ast.GenDecl:
				for _, s := range dk.Specs {
					if vs, ok := s.(*ast.ValueSpec); ok {
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
				ast.Inspect(k, func(x ast.Node) bool {
					switch x := x.(type) {
					case *ast.SelectorExpr:
						if p, ok := x.X.(*ast.Ident); ok && (p.Name == "settings" || p.Name == "ui") {
							fremd[p.Name+"."+x.Sel.Name] = true
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
	for _, verboten := range []string{"settings.Schreibe", "settings.SchreibeDatei", "ui.Editor"} {
		if fremd[verboten] {
			t.Errorf("von mcp.go aus erreichbar: %s", verboten)
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
		ast.Inspect(f, func(x ast.Node) bool {
			switch x := x.(type) {
			case *ast.SelectorExpr:
				if p, ok := x.X.(*ast.Ident); ok && p.Name == "settings" && strings.HasPrefix(x.Sel.Name, "Schreibe") {
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
