package market

import (
	"errors"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func marketSpec(t *testing.T) *spec.Spec {
	t.Helper()
	s, err := SpecFor("checktest")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestCheckRefusesARoleThisDomainFillsNoObjectFor(t *testing.T) {
	err := Check(marketSpec(t), "no_such_role", Poster{})
	if err == nil || !strings.Contains(err.Error(), "no_such_role") {
		t.Fatalf("got %v", err)
	}
}

func TestCheckWantsAStructAndAcceptsAPointerToOne(t *testing.T) {
	s := marketSpec(t)
	if err := Check(s, RolePoster, "poster"); err == nil || !strings.Contains(err.Error(), "want a struct") {
		t.Fatalf("string: %v", err)
	}
	if err := Check(s, RolePoster, &Poster{Name: "Ada Farm", Status: "pending", Availability: "away"}); err != nil {
		t.Fatalf("pointer: %v", err)
	}
}

func TestCheckRefusesARequiredFieldLeftBlankOrOnlySpaces(t *testing.T) {
	s := marketSpec(t)
	for _, name := range []string{"", "   "} {
		err := Check(s, RolePoster, Poster{Name: name, Status: "pending", Availability: "available"})
		if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "is required") {
			t.Fatalf("name %q: %v", name, err)
		}
	}
}

func TestCheckRefusesAnEnumValueTheBlueprintDoesNotDeclareAndNamesTheAllowedOnes(t *testing.T) {
	err := Check(marketSpec(t), RolePoster, Poster{Name: "Ada Farm", Status: "pending", Availability: "sleeping"})
	if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), `"sleeping"`) || !strings.Contains(err.Error(), "away") {
		t.Fatalf("got %v", err)
	}
}

func TestCheckAcceptsADeclaredEnumValue(t *testing.T) {
	if err := Check(marketSpec(t), RolePoster, Poster{Name: "Ada Farm", Status: "pending", Availability: "away"}); err != nil {
		t.Fatal(err)
	}
}

func TestCheckFieldRefusesAColumnTheObjectDoesNotDeclare(t *testing.T) {
	o, ok := marketSpec(t).ByRole(RolePoster)
	if !ok {
		t.Fatal("no poster object")
	}
	if err := checkField(o, "nonsense", "x"); err == nil || !strings.Contains(err.Error(), `no field "nonsense"`) {
		t.Fatalf("got %v", err)
	}
	if err := checkField(o, "name", ""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("blank required: %v", err)
	}
	if err := checkField(o, "name", "Ada"); err != nil {
		t.Fatal(err)
	}
}

func TestCheckValueAppliesTheNamedPatternAndRefusesAnUnknownOne(t *testing.T) {
	o := spec.Object{Name: "thing"}
	named := spec.Field{Name: "code", Type: spec.TypeString, Matches: "name"}
	if err := checkValue(o, named, "ok_name"); err != nil {
		t.Fatalf("matching: %v", err)
	}
	if err := checkValue(o, named, "not a name!"); !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "does not match name") {
		t.Fatalf("mismatch: %v", err)
	}
	if err := checkValue(o, named, ""); err != nil {
		t.Fatalf("empty optional value skips the pattern: %v", err)
	}
	unknown := spec.Field{Name: "code", Type: spec.TypeString, Matches: "ghost"}
	err := checkValue(o, unknown, "x")
	if err == nil || errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), `"ghost"`) {
		t.Fatalf("unknown pattern should be a module error, not a caller refusal: %v", err)
	}
}

func TestIsNameFollowsTheSpecAlphabet(t *testing.T) {
	if !isName("agy1f2e3d4c") || isName("has space") || isName("semi;colon") || isName("") {
		t.Fatal("isName disagrees with spec.NamePattern")
	}
}
