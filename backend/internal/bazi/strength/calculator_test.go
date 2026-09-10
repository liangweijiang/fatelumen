package strength

import "testing"

func TestEvaluateV1ProducesTraceableResult(t *testing.T) {
	in := Input{Year: Pillar{"甲", "子"}, Month: Pillar{"丙", "寅"}, Day: Pillar{"戊", "辰"}, Hour: Pillar{"丁", "巳"}}
	r, err := Evaluate(in, RuleV1())
	if err != nil {
		t.Fatal(err)
	}
	if r.RuleVersion != RuleVersionV1 {
		t.Fatalf("version=%s", r.RuleVersion)
	}
	if r.SupportScore <= 0 || r.RestraintScore <= 0 {
		t.Fatalf("unexpected scores: %+v", r)
	}
	if r.SupportRatio < 0 || r.SupportRatio > 1 {
		t.Fatalf("ratio=%f", r.SupportRatio)
	}
	if len(r.Contributions) != 14 {
		t.Fatalf("contributions=%d want 14", len(r.Contributions))
	}
	if r.RootLevel == "" {
		t.Fatalf("missing root evidence: %+v", r)
	}
}

func TestEvaluateDeterministic(t *testing.T) {
	in := Input{Year: Pillar{"庚", "午"}, Month: Pillar{"戊", "子"}, Day: Pillar{"丙", "寅"}, Hour: Pillar{"甲", "午"}}
	a, e := Evaluate(in, RuleV1())
	if e != nil {
		t.Fatal(e)
	}
	b, e := Evaluate(in, RuleV1())
	if e != nil {
		t.Fatal(e)
	}
	if a.SupportScore != b.SupportScore || a.RestraintScore != b.RestraintScore || a.SupportRatio != b.SupportRatio {
		t.Fatalf("non-deterministic: %+v / %+v", a, b)
	}
}

func TestEvaluateInvalidInput(t *testing.T) {
	_, err := Evaluate(Input{Year: Pillar{"X", "子"}}, RuleV1())
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestTenGod(t *testing.T) {
	cases := map[string]string{"甲": "比肩", "乙": "劫财", "壬": "偏印", "癸": "正印", "丙": "食神", "丁": "伤官", "庚": "七杀", "辛": "正官", "戊": "偏财", "己": "正财"}
	for target, want := range cases {
		if got := tenGod("甲", target); got != want {
			t.Errorf("甲 -> %s = %s want %s", target, got, want)
		}
	}
}

func TestRelationsUseDistinctPositions(t *testing.T) {
	in := Input{Year: Pillar{"甲", "子"}, Month: Pillar{"己", "丑"}, Day: Pillar{"戊", "子"}, Hour: Pillar{"癸", "午"}}
	r, err := Evaluate(in, RuleV1())
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Relations) == 0 {
		t.Fatal("expected relation evidence")
	}
	for _, rel := range r.Relations {
		if len(rel.Positions) != len(rel.Symbols) {
			t.Fatalf("bad evidence: %+v", rel)
		}
	}
}
