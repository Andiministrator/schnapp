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
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// text ist ein Oberflächentext, wie er im Code an sprache.T, sprache.TK oder sprache.N übergeben wird.
type text struct{ kontext, id, mehrzahl string }

// sprachePfad ist der Importpfad des Pakets, dessen Aufrufe gesammelt werden.
const sprachePfad = "github.com/Andiministrator/schnapp/sprache"

// sammleTexte durchsucht alle .go-Dateien unter wurzel nach Aufrufen von sprache.T, sprache.TK und sprache.N —
// auch unter einem Import-Alias; Dateien ohne diesen Import zählen nicht. Verzeichnisse mit eigenem go.mod
// (andere Module), testdata und versteckte Verzeichnisse werden übersprungen. Probleme sind: ein Argument, das kein
// Zeichenketten-Literal ist (dann kann niemand prüfen, ob die Übersetzung existiert), und ein Punkt-Import des
// Pakets (dann sind die Aufrufe nicht erkennbar). Aufrufe über einen *Katalog findet die Prüfung nicht; im
// Programmcode gelten deshalb nur die Paketfunktionen (CONTRIBUTING).
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
		paketname := ""
		for _, imp := range f.Imports {
			if pfad, _ := strconv.Unquote(imp.Path.Value); pfad == sprachePfad {
				paketname = "sprache"
				if imp.Name != nil {
					paketname = imp.Name.Name
				}
			}
		}
		if paketname == "." {
			probleme = append(probleme, fmt.Sprintf("%s: Punkt-Import von sprache — Aufrufe nicht prüfbar", pfad))
			return nil
		}
		if paketname == "" || paketname == "_" {
			return nil
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
			if !ok || paket.Name != paketname {
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
	kopfMitMehrzahl := false
	nachSchluessel := map[schluessel]Eintrag{}
	for _, e := range eintraege {
		if e.ID == "" && e.Kontext == "" {
			if r, ok := kopffeld(e, "Plural-Forms"); ok {
				regel, err := LeseMehrzahlregel(r)
				if err != nil {
					probleme = append(probleme, fmt.Sprintf("Plural-Forms: %v", err))
				} else {
					formen, kopfMitMehrzahl = regel.Formen, true
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
		for i, w := range e.Texte {
			if w == "" {
				probleme = append(probleme, fmt.Sprintf("leer: %q", t.id))
				break
			}
			// Platzhalter: ohne Mehrzahl genau dieselben wie im Original; bei Mehrzahl darf eine Form welche
			// weglassen ("ein Bild"), aber keine fremden haben.
			soll := platzhalter(t.id)
			if t.mehrzahl != "" {
				soll = platzhalter(t.mehrzahl)
			}
			ist := platzhalter(w)
			if (t.mehrzahl == "" && !reflect.DeepEqual(ist, soll)) || (t.mehrzahl != "" && !teilmenge(ist, soll)) {
				probleme = append(probleme, fmt.Sprintf("Platzhalter: %q Form %d hat %v, Original %v", t.id, i, ist, soll))
			}
		}
	}
	if !kopfMitMehrzahl {
		probleme = append(probleme, "Kopf ohne Plural-Forms")
	}
	for s := range nachSchluessel {
		if !imCode[s] {
			probleme = append(probleme, fmt.Sprintf("veraltet (nicht im Code): %q (Kontext %q)", s.id, s.kontext))
		}
	}
	sort.Strings(probleme)
	return probleme
}

// platzhalterMuster findet printf-Platzhalter wie %d, %s, %[1]d, %.2f (ohne %%). Das Leerzeichen-Flag ("% d") ist
// bewusst nicht dabei: sonst gälte "100% fit" als Platzhalter "% f" (F-46). Oberflächentexte nutzen es nicht.
var platzhalterMuster = regexp.MustCompile(`%(\[\d+\])?[-+#0]*(\d+|\*)?(\.(\d+|\*))?[a-zA-Z%]`)

// platzhalter liefert die Platzhalter eines Texts, sortiert.
func platzhalter(s string) []string {
	var res []string
	for _, p := range platzhalterMuster.FindAllString(s, -1) {
		if p != "%%" {
			res = append(res, p)
		}
	}
	sort.Strings(res)
	return res
}

// teilmenge meldet, ob jeder Platzhalter aus a (mit Vielfachheit) auch in b steht.
func teilmenge(a, b []string) bool {
	rest := map[string]int{}
	for _, p := range b {
		rest[p]++
	}
	for _, p := range a {
		if rest[p] == 0 {
			return false
		}
		rest[p]--
	}
	return true
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
		{id: "%s saved"}, // b.go, unter Import-Alias
	}
	if !reflect.DeepEqual(texte, wantTexte) {
		t.Errorf("Texte:\n got %v\nwant %v", texte, wantTexte)
	}
	if len(probleme) != 2 || !strings.Contains(probleme[0], "a.go") || !strings.Contains(probleme[0], "sprache.T ohne Zeichenketten-Literal") ||
		!strings.Contains(probleme[1], "c.go: Punkt-Import") {
		t.Errorf("Probleme = %q", probleme)
	}

	po := "msgid \"\"\nmsgstr \"Plural-Forms: nplurals=2; plural=(n != 1);\\n\"\n\n" +
		"msgid \"Save\"\nmsgstr \"Speichern\"\n\n" +
		"msgctxt \"menu\"\nmsgid \"Open\"\nmsgstr \"Öffnen\"\n\n" +
		"msgid \"%d image\"\nmsgid_plural \"%d pictures\"\nmsgstr[0] \"%d Bild\"\nmsgstr[1] \"%d Bilder\"\n\n" +
		"msgid \"%d file\"\nmsgid_plural \"%d files\"\nmsgstr[0] \"%d Datei\"\n\n" +
		"msgid \"%s saved\"\nmsgstr \"%d gespeichert\"\n\n" +
		"#, fuzzy\nmsgid \"Close\"\nmsgstr \"Schließen\"\n\n" +
		"msgid \"Gone\"\nmsgstr \"Weg\"\n"
	eintraege, err := LesePO(strings.NewReader(po))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		`Mehrzahl passt nicht: "%d image": Code "%d images", PO "%d pictures"`,
		`Mehrzahlformen: "%d file" hat 1, Regel verlangt 2`,
		`Platzhalter: "%s saved" Form 0 hat [%d], Original [%s]`,
		`fehlt: "Missing" (Kontext "")`,
		`unscharf: "Close"`,
		`veraltet (nicht im Code): "Gone" (Kontext "")`,
	}
	if got := pruefeUebersetzung(texte, eintraege); !reflect.DeepEqual(got, want) {
		t.Errorf("pruefeUebersetzung:\n got %q\nwant %q", got, want)
	}
}

// Platzhalter in Mehrzahlformen und der Pflichtkopf Plural-Forms (F-38).
func TestPruefungPlatzhalterUndKopf(t *testing.T) {
	texte := []text{{id: "%d image", mehrzahl: "%d images"}, {id: "%[1]s of %[2]d"}, {id: "Zoom 100% fit"}}
	po := "msgid \"\"\nmsgstr \"Language: xx\\n\"\n\n" +
		"msgid \"%d image\"\nmsgid_plural \"%d images\"\nmsgstr[0] \"ein Bild\"\nmsgstr[1] \"%d Bilder %s\"\n\n" +
		"msgid \"%[1]s of %[2]d\"\nmsgstr \"%[2]d von %[1]s, 100%%\"\n\n" +
		"msgid \"Zoom 100% fit\"\nmsgstr \"Zoom 100 % passend\"\n"
	eintraege, err := LesePO(strings.NewReader(po))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"Kopf ohne Plural-Forms",
		`Platzhalter: "%d image" Form 1 hat [%d %s], Original [%d]`,
	}
	if got := pruefeUebersetzung(texte, eintraege); !reflect.DeepEqual(got, want) {
		t.Errorf("pruefeUebersetzung:\n got %q\nwant %q", got, want)
	}
}
