package prompts

import (
	"strings"
	"testing"
)

func TestPhraseCoverageRequiresProfessionalReview(t *testing.T) {
	if zh := phraseCoverage("zh"); !zh.Ready || zh.Approved != zh.Total {
		t.Fatalf("Chinese phrases must be approved: %+v", zh)
	}
	for _, locale := range []string{"en", "ja", "ko"} {
		coverage := phraseCoverage(locale)
		if coverage.Ready || coverage.Missing != coverage.Total {
			t.Fatalf("%s phrase coverage must remain visible before review: %+v", locale, coverage)
		}
	}
}

func TestLanguageInstructionUsesLocaleGlossaryAndAllowsContextTranslation(t *testing.T) {
	tests := []struct {
		locale   string
		expected []string
	}{
		{"en", []string{"日主 → Day Master", "比肩 → Companion", "天干五合 → Stem combination"}},
		{"ja", []string{"日主 → 日主", "比肩 → 比肩", "天干五合 → 干合"}},
		{"ko", []string{"日主 → 일간", "比肩 → 비견", "天干五合 → 천간합"}},
	}
	for _, test := range tests {
		spec, _ := NormalizeLocale(test.locale)
		instruction, glossary := buildLanguageInstruction(spec, "分析日主、比肩与天干五合。")
		if len(glossary) != 3 || !strings.Contains(instruction, "未列出的专业词") {
			t.Fatalf("%s glossary not injected: %s", test.locale, instruction)
		}
		for _, expected := range test.expected {
			if !strings.Contains(instruction, expected) {
				t.Fatalf("%s instruction missing %q: %s", test.locale, expected, instruction)
			}
		}
	}
}
