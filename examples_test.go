package market_test

import (
	"path/filepath"
	"strings"
	"testing"

	market "github.com/aosanya/mwanachama-backend-market"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func examples(t *testing.T) []string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join("spec", "examples", "*.market.json"))
	if err != nil || len(paths) < 2 {
		t.Fatalf("want two example domains, got %v %v", paths, err)
	}
	return paths
}

func TestEveryExampleFitsTheTypes(t *testing.T) {
	for _, p := range examples(t) {
		s, err := market.LoadSpec(p)
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		if _, err := market.NewMarketManager(newDB(t), s); err != nil {
			t.Fatalf("%s: %v", p, err)
		}
	}
}

func TestTwoDomainsCoexist(t *testing.T) {
	db := newDB(t)
	for _, p := range examples(t) {
		s, err := market.LoadSpec(p)
		if err != nil {
			t.Fatal(err)
		}
		if err := market.Provision(db, s); err != nil {
			t.Fatal(err)
		}
	}
	var n int64
	db.Raw(`select count(*) from sqlite_master where type='table' and name not like '%spec_table_names' and name not like '%\_test' escape '\'`).Scan(&n)
	if n != 26 {
		t.Fatalf("want 26 tables, got %d", n)
	}
}

func TestRequiredFieldsHaveNoDefault(t *testing.T) {
	b, _ := market.Blueprint()
	s, _ := market.SpecFor("app")
	_ = b
	for _, o := range s.Objects {
		for _, f := range o.Fields {
			if f.Required && f.Default != "" {
				t.Fatalf("%s.%s is required and defaulted", o.Name, f.Name)
			}
		}
	}
}

func TestMountsKeepSeparateTables(t *testing.T) {
	a, _ := market.SpecForMount("app", "one")
	b, _ := market.SpecForMount("app", "two")
	if a.TableFor(a.Objects[0]) == b.TableFor(b.Objects[0]) {
		t.Fatal("two mounts share a table")
	}
	if _, err := market.SpecFor("bad_name"); err == nil {
		t.Fatal("an instance with an underscore must be refused")
	}
	var _ *spec.Spec = a
}

func TestTheModuleNamesNoDomain(t *testing.T) {
	banned := []string{"crop", "ride", "shop", "farmer", "supplier", "rider", "harvest"}
	paths, _ := filepath.Glob("*.go")
	more, _ := filepath.Glob("models/*.go")
	paths = append(append(paths, more...), "market.blueprint.json")
	for _, p := range paths {
		if strings.HasSuffix(p, "_test.go") {
			continue
		}
		raw := readFile(t, p)
		for _, w := range banned {
			for _, tok := range strings.FieldsFunc(strings.ToLower(raw), func(r rune) bool { return r < 'a' || r > 'z' }) {
				if tok == w {
					t.Fatalf("%s carries the domain word %q", p, w)
				}
			}
		}
	}
}
