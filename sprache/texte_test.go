// SPDX-License-Identifier: GPL-3.0-or-later

package sprache

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// text ist ein Oberflächentext, wie er im Code an sprache.T, sprache.TK oder sprache.N übergeben wird.
type text struct{ kontext, id, mehrzahl string }

// sammleTexte durchsucht alle .go-Dateien unter wurzel nach Aufrufen von sprache.T, sprache.TK und sprache.N.
// Verzeichnisse mit eigenem go.mod (andere Module), testdata und versteckte Verzeichnisse werden übersprungen.
// Ein Aufruf mit einem Argument, das kein Zeichenketten-Literal ist, ist ein Problem: dann kann niemand prüfen, ob
// die Übersetzung existiert.
func sammleTexte(wurzel string) ([]text, []string, error) {
	var texte []text
	var probleme []string
	err := filepath.WalkDir(wurzel, func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if pfad != wurzel && (name == "testdata" || strings.HasPrefix(name, ".") || name == "tmp") {
				return filepath.SkipDir
			}
			if pfad != wurzel {
				if _, err := os.Stat(filepath.Join(pfad, "go.mod")); err == nil {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") {
			return nil
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, pfad, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			aufruf, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := aufruf.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			paket, ok := sel.X.(*ast.Ident)
			if !ok || paket.Name != "sprache" {
				return true
			}
			var anzahl int
			switch sel.Sel.Name {
			case "T":
				anzahl = 1
			case "TK", "N":
				anzahl = 2
			default:
				return true
			}
			var werte []string
			for _, arg := range aufruf.Args[:min(anzahl, len(aufruf.Args))] {
				lit, ok := arg.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					probleme = append(probleme, fmt.Sprintf("%s: sprache.%s ohne Zeichenketten-Literal",
						fset.Position(arg.Pos()), sel.Sel.Name))
					return true
				}
				w, err := strconv.Unquote(lit.Value)
				if err != nil {
					probleme = append(probleme, fmt.Sprintf("%s: %v", fset.Position(arg.Pos()), err))
					return true
				}
				werte = append(werte, w)
			}
			if len(werte) != anzahl {
				return true
			}
			switch sel.Sel.Name {
			case "T":
				texte = append(texte, text{id: werte[0]})
			case "TK":
				texte = append(texte, text{kontext: werte[0], id: werte[1]})
			case "N":
				texte = append(texte, text{id: werte[0], mehrzahl: werte[1]})
			}
			return true
		})
		return nil
	})
	return texte, probleme, err
}

// pruefeUebersetzung vergleicht die Texte im Code mit einer PO-Datei: jeder Text braucht einen fertigen Eintrag
// (nicht unscharf, nicht leer, bei Mehrzahl mit derselben msgid_plural und so vielen Formen wie die Regel kennt),
// und jeder Eintrag braucht einen Text im Code (sonst ist er veraltet).
func pruefeUebersetzung(texte []text, eintraege []Eintrag) []string {
	var probleme []string
	formen := englischeRegel.Formen
	nachSchluessel := map[schluessel]Eintrag{}
	for _, e := range eintraege {
		if e.ID == "" && e.Kontext == "" {
			if r, ok := kopffeld(e, "Plural-Forms"); ok {
				if regel, err := LeseMehrzahlregel(r); err == nil {
					formen = regel.Formen
				}
			}
			continue
		}
		nachSchluessel[schluessel{e.Kontext, e.ID}] = e
	}
	imCode := map[schluessel]bool{}
	for _, t := range texte {
		s := schluessel{t.kontext, t.id}
		imCode[s] = true
		e, ok := nachSchluessel[s]
		switch {
		case !ok:
			probleme = append(probleme, fmt.Sprintf("fehlt: %q (Kontext %q)", t.id, t.kontext))
			continue
		case e.Unscharf:
			probleme = append(probleme, fmt.Sprintf("unscharf: %q", t.id))
		case e.Mehrzahl != t.mehrzahl:
			probleme = append(probleme, fmt.Sprintf("Mehrzahl passt nicht: %q: Code %q, PO %q", t.id, t.mehrzahl, e.Mehrzahl))
		case t.mehrzahl != "" && len(e.Texte) != formen:
			probleme = append(probleme, fmt.Sprintf("Mehrzahlformen: %q hat %d, Regel verlangt %d", t.id, len(e.Texte), formen))
		}
		for _, w := range e.Texte {
			if w == "" {
				probleme = append(probleme, fmt.Sprintf("leer: %q", t.id))
				break
			}
		}
	}
	for s := range nachSchluessel {
		if !imCode[s] {
			probleme = append(probleme, fmt.Sprintf("veraltet (nicht im Code): %q (Kontext %q)", s.id, s.kontext))
		}
	}
	sort.Strings(probleme)
	return probleme
}

// Jeder Text im Code steht fertig in jeder PO-Datei, und keine PO-Datei trägt Veraltetes (ADR-0014).
func TestUebersetzungenVollstaendig(t *testing.T) {
	texte, probleme, err := sammleTexte("..")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range probleme {
		t.Error(p)
	}
	poDateien, err := filepath.Glob(filepath.Join("..", "po", "*.po"))
	if err != nil || len(poDateien) == 0 {
		t.Fatalf("keine PO-Dateien gefunden (%v)", err)
	}
	for _, datei := range poDateien {
		f, err := os.Open(datei)
		if err != nil {
			t.Fatal(err)
		}
		eintraege, err := LesePO(f)
		f.Close()
		if err != nil {
			t.Fatalf("%s: %v", datei, err)
		}
		for _, p := range pruefeUebersetzung(texte, eintraege) {
			t.Errorf("%s: %s", filepath.Base(datei), p)
		}
	}
}

// Die Prüfung selbst: ein Probemodul unter testdata mit bekannten Lücken.
func TestPruefungFindetLuecken(t *testing.T) {
	texte, probleme, err := sammleTexte(filepath.Join("testdata", "probemodul"))
	if err != nil {
		t.Fatal(err)
	}
	wantTexte := []text{
		{id: "Save"},
		{id: "Missing"},
		{kontext: "menu", id: "Open"},
		{id: "%d image", mehrzahl: "%d images"},
		{id: "%d file", mehrzahl: "%d files"},
		{id: "Close"},
	}
	if !reflect.DeepEqual(texte, wantTexte) {
		t.Errorf("Texte:\n got %v\nwant %v", texte, wantTexte)
	}
	if len(probleme) != 1 || !strings.Contains(probleme[0], "sprache.T ohne Zeichenketten-Literal") {
		t.Errorf("Probleme = %v", probleme)
	}

	po := "msgid \"\"\nmsgstr \"Plural-Forms: nplurals=2; plural=(n != 1);\\n\"\n\n" +
		"msgid \"Save\"\nmsgstr \"Speichern\"\n\n" +
		"msgctxt \"menu\"\nmsgid \"Open\"\nmsgstr \"Öffnen\"\n\n" +
		"msgid \"%d image\"\nmsgid_plural \"%d pictures\"\nmsgstr[0] \"%d Bild\"\nmsgstr[1] \"%d Bilder\"\n\n" +
		"msgid \"%d file\"\nmsgid_plural \"%d files\"\nmsgstr[0] \"%d Datei\"\n\n" +
		"#, fuzzy\nmsgid \"Close\"\nmsgstr \"Schließen\"\n\n" +
		"msgid \"Gone\"\nmsgstr \"Weg\"\n"
	eintraege, err := LesePO(strings.NewReader(po))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		`Mehrzahl passt nicht: "%d image": Code "%d images", PO "%d pictures"`,
		`Mehrzahlformen: "%d file" hat 1, Regel verlangt 2`,
		`fehlt: "Missing" (Kontext "")`,
		`unscharf: "Close"`,
		`veraltet (nicht im Code): "Gone" (Kontext "")`,
	}
	if got := pruefeUebersetzung(texte, eintraege); !reflect.DeepEqual(got, want) {
		t.Errorf("pruefeUebersetzung:\n got %q\nwant %q", got, want)
	}
}
