package displaydict

import "testing"

func TestDictionaryTranslations(t *testing.T) {
	cases := map[string]string{"element.power.wood": "木力量", "slightly_strong": "身偏强", "combination": "天干五合", "visible_and_rooted": "透干且有根"}
	for code, expected := range cases {
		if actual := ZH(code); actual != expected {
			t.Errorf("ZH(%q)=%q want %q", code, actual, expected)
		}
	}
	if len(Terms()) < 50 {
		t.Fatalf("dictionary unexpectedly small: %d", len(Terms()))
	}
}
