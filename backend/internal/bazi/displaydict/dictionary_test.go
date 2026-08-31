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

func TestLocaleCoverageDoesNotPresentDraftTranslationsAsApproved(t *testing.T) {
	zh := LocaleCoverage("zh")
	if !zh.Ready || zh.Approved != zh.Total {
		t.Fatalf("Chinese dictionary must be approved: %+v", zh)
	}
	en := LocaleCoverage("en")
	if en.Ready || en.Draft == 0 {
		t.Fatalf("English draft terms must remain visible: %+v", en)
	}
	ja := LocaleCoverage("ja")
	if ja.Ready || ja.Draft != ja.Total || ja.Missing != 0 {
		t.Fatalf("Japanese draft terms must remain visible: %+v", ja)
	}
}

func TestGlossaryForTextOnlyIncludesTermsUsedByChapter(t *testing.T) {
	items := GlossaryForText("日主身偏强，天干五合。", "en")
	if len(items) == 0 {
		t.Fatal("expected chapter glossary")
	}
	for _, item := range items {
		if item.Source == "五行偏颇" {
			t.Fatal("unrelated term leaked into chapter glossary")
		}
	}
	if items := GlossaryForText("日主", "zh"); len(items) != 0 {
		t.Fatal("Chinese prompt must not inject translation glossary")
	}
}

func TestJapaneseAndKoreanDictionaryDraftsAreComplete(t *testing.T) {
	for _, item := range Terms() {
		if item.Names.JA == "" || item.Names.KO == "" {
			t.Errorf("missing ja/ko term: %s/%s", item.Category, item.Code)
		}
		if item.Review.JA != "draft" || item.Review.KO != "draft" {
			t.Errorf("unexpected review status: %+v", item)
		}
	}
	for _, locale := range []string{"ja", "ko"} {
		coverage := LocaleCoverage(locale)
		if coverage.Missing != 0 || coverage.Draft != coverage.Total {
			t.Fatalf("%s coverage incomplete: %+v", locale, coverage)
		}
	}
}
