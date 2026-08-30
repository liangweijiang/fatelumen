package hash

import "testing"

func TestCanonicalJSONSHA256MapOrder(t *testing.T) {
	a, err := CanonicalJSONSHA256(map[string]int{"b": 2, "a": 1})
	if err != nil {
		t.Fatal(err)
	}
	b, err := CanonicalJSONSHA256(map[string]int{"a": 1, "b": 2})
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("map order changed hash: %s != %s", a, b)
	}
}
