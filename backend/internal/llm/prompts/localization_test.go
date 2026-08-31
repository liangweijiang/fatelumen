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

func TestLanguageInstructionUsesKnownTermsAndAllowsContextTranslation(t *testing.T) {
	spec, _ := NormalizeLocale("en")
	instruction, glossary := buildLanguageInstruction(spec, "分析日主、比肩与天干五合。")
	if len(glossary) == 0 || !strings.Contains(instruction, "比肩") || !strings.Contains(instruction, "未列出的专业词") {
		t.Fatalf("unexpected language instruction: %s", instruction)
	}
}
