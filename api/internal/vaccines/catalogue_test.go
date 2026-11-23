package vaccines

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func mustLoad(t *testing.T) {
	t.Helper()
	if err := Load(); err != nil {
		t.Fatalf("Load() = %v", err)
	}
}

func TestAntigenCatalogue(t *testing.T) {
	mustLoad(t)
	want := []string{
		"diphtheria", "tetanus", "pertussis", "polio", "hib", "hepatitis_b", "hepatitis_a", "pneumococcus",
		"rotavirus", "measles", "mumps", "rubella", "varicella", "hpv", "meningococcal_acwy", "meningococcal_b",
		"influenza", "covid19", "rsv", "tuberculosis", "typhoid", "yellow_fever", "rabies",
		"japanese_encephalitis", "cholera", "tick_borne_encephalitis", "other",
	}
	var got []string
	for _, a := range Antigens() {
		got = append(got, a.ID)
		if a.Name == "" {
			t.Errorf("antigen %s has no name", a.ID)
		}
	}
	if !slices.Equal(got, want) {
		t.Errorf("antigen ids = %v, want %v", got, want)
	}
	for _, a := range Antigens() {
		if a.ID == "other" && a.Name != "Other (product name required)" {
			t.Errorf("other's name = %q", a.Name)
		}
	}
}

func TestProductCatalogue(t *testing.T) {
	mustLoad(t)
	want := map[string][]string{
		"hexavalent":    {"diphtheria", "tetanus", "pertussis", "polio", "hib", "hepatitis_b"},
		"dtap":          {"diphtheria", "tetanus", "pertussis"},
		"dtap_ipv":      {"diphtheria", "tetanus", "pertussis", "polio"},
		"dtap_ipv_hib":  {"diphtheria", "tetanus", "pertussis", "polio", "hib"},
		"dtap_hepb_ipv": {"diphtheria", "tetanus", "pertussis", "hepatitis_b", "polio"},
		"tdap":          {"diphtheria", "tetanus", "pertussis"},
		"td":            {"diphtheria", "tetanus"},
		"td_ipv":        {"diphtheria", "tetanus", "polio"},
		"mmr":           {"measles", "mumps", "rubella"},
		"mmrv":          {"measles", "mumps", "rubella", "varicella"},
		"hib":           {"hib"},
		"hepatitis_b":   {"hepatitis_b"},
		"hepatitis_a":   {"hepatitis_a"},
		"pcv":           {"pneumococcus"},
		"rotavirus":     {"rotavirus"},
		"menacwy":       {"meningococcal_acwy"},
		"menb":          {"meningococcal_b"},
		"hpv":           {"hpv"},
		"influenza":     {"influenza"},
		"covid19":       {"covid19"},
		"rsv_antibody":  {"rsv"},
		"bcg":           {"tuberculosis"},
		"ipv":           {"polio"},
		"varicella":     {"varicella"},
	}
	got := Products()
	if len(got) != len(want) {
		t.Errorf("%d products, want %d", len(got), len(want))
	}
	for _, p := range got {
		w, ok := want[p.ID]
		if !ok {
			t.Errorf("unexpected product %s", p.ID)
			continue
		}
		if !slices.Equal(p.Antigens, w) {
			t.Errorf("product %s antigens = %v, want %v", p.ID, p.Antigens, w)
		}
		if err := ValidAntigens(p.Antigens); err != nil {
			t.Errorf("product %s: %v", p.ID, err)
		}
	}
	names := map[string]string{}
	for _, p := range got {
		names[p.ID] = p.Name
	}
	if names["hexavalent"] != "6-in-1 (DTaP-IPV-Hib-HepB)" || names["rsv_antibody"] != "RSV (monoclonal antibody)" ||
		names["dtap_ipv_hib"] != "DTaP-IPV-Hib (5-in-1)" || names["dtap_hepb_ipv"] != "DTaP-HepB-IPV (5-in-1)" {
		t.Errorf("product names = %v", names)
	}
}

func TestNoBrandNames(t *testing.T) {
	mustLoad(t)
	brands := []string{
		"pediarix", "pentacel", "vaxelis", "infanrix", "hexyon", "vaxneuvance", "prevnar", "gardasil", "priorix",
		"proquad", "rotarix", "rotateq", "menitorix", "nimenrix", "bexsero", "beyfortus",
		"repevax", "engerix", "hbvaxpro", "boostrix", "revaxis", "kinrix", "quadracel", "daptacel", "havrix",
		"vaqta", "varivax", "recombivax", "hiberix", "acthib", "menquadfi", "trumenba", "pneumovax", "synflorix",
		"cervarix", "fluenz", "comirnaty", "spikevax", "abrysvo", "arexvy",
	}
	var names []string
	for _, p := range Products() {
		names = append(names, p.ID, p.Name)
	}
	for _, a := range Antigens() {
		names = append(names, a.ID, a.Name)
	}
	for _, n := range names {
		words := strings.FieldsFunc(strings.ToLower(n), func(r rune) bool { return r < 'a' || r > 'z' })
		for _, b := range brands {
			if slices.Contains(words, b) {
				t.Errorf("%q contains the brand name %q", n, b)
			}
		}
	}
}

func TestValidAntigens(t *testing.T) {
	mustLoad(t)
	if err := ValidAntigens([]string{"measles", "other"}); err != nil {
		t.Errorf("ValidAntigens(known) = %v", err)
	}
	cases := []struct {
		ids  []string
		want error
	}{
		{nil, ErrNoAntigens},
		{[]string{}, ErrNoAntigens},
		{[]string{"measles", "smallpox"}, ErrUnknownAntigen},
		{[]string{""}, ErrUnknownAntigen},
		{[]string{"measles", "measles"}, ErrDuplicateAntigen},
	}
	for _, c := range cases {
		if err := ValidAntigens(c.ids); !errors.Is(err, c.want) {
			t.Errorf("ValidAntigens(%v) = %v, want %v", c.ids, err, c.want)
		}
	}
}

func TestParseCatalogueRefusals(t *testing.T) {
	ants := map[string]bool{"measles": true, "mumps": true}
	cases := []struct{ name, in, want string }{
		{"bad antigens header", "id,title\n", "vaccines: antigens.csv:1: column name: header is not id,name"},
		{"empty antigen id", "id,name\n,Measles\n", "vaccines: antigens.csv:2: column id: is empty"},
		{"duplicate antigen", "id,name\nmeasles,Measles\nmeasles,Measles again\n", "vaccines: antigens.csv:3: column id: duplicate id"},
		{"empty antigen name", "id,name\nmeasles,\n", "vaccines: antigens.csv:2: column name: is empty"},
	}
	for _, c := range cases {
		_, err := parseAntigens("antigens.csv", strings.NewReader(c.in))
		if err == nil || err.Error() != c.want {
			t.Errorf("%s: err = %v, want %q", c.name, err, c.want)
		}
	}
	pcases := []struct{ name, in, want string }{
		{"bad products header", "id,name\n", "vaccines: products.csv:1: column antigens: header is not id,name,antigens"},
		{"unknown antigen", "id,name,antigens\nx,X,measles|smallpox\n", "vaccines: products.csv:2: column antigens: unknown antigen"},
		{"duplicate antigen", "id,name,antigens\nx,X,measles|measles\n", "vaccines: products.csv:2: column antigens: duplicate antigen"},
		{"no antigens", "id,name,antigens\nx,X,\n", "vaccines: products.csv:2: column antigens: is empty"},
		{"duplicate product", "id,name,antigens\nx,X,measles\nx,Y,mumps\n", "vaccines: products.csv:3: column id: duplicate id"},
	}
	for _, c := range pcases {
		_, err := parseProducts("products.csv", strings.NewReader(c.in), ants)
		if err == nil || err.Error() != c.want {
			t.Errorf("%s: err = %v, want %q", c.name, err, c.want)
		}
	}
}
