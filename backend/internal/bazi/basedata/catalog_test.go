package basedata

import "testing"

func TestV1Complete(t *testing.T) {
	c := V1()
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	s := c.Summary()
	if !s.Valid || s.Stems != 10 || s.Branches != 12 || s.Relations == 0 {
		t.Fatalf("bad summary: %+v", s)
	}
}
func TestHiddenStemWeightsEqualOne(t *testing.T) {
	for _, b := range V1().Branches {
		sum := 0.0
		for _, h := range b.HiddenStems {
			sum += h.Weight
		}
		if sum < .999999 || sum > 1.000001 {
			t.Errorf("%s sum=%f", b.Code, sum)
		}
	}
}
func TestV1ReturnsIndependentCatalog(t *testing.T) {
	a := V1()
	b := V1()
	a.Stems[0].Element = "水"
	if b.Stems[0].Element != "木" {
		t.Fatal("catalog instances share mutable stem data")
	}
}
