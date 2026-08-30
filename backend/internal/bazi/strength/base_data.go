package strength

import "fatelumen/backend/internal/bazi/basedata"

var stemElement, stemYang, generates, controls = baseMaps()

func baseMaps() (map[string]string, map[string]bool, map[string]string, map[string]string) {
	c := basedata.V1()
	se := map[string]string{}
	sy := map[string]bool{}
	g := map[string]string{}
	co := map[string]string{}
	for _, s := range c.Stems {
		se[s.Code] = s.Element
		sy[s.Code] = s.YinYang == "yang"
	}
	for _, e := range c.Elements {
		g[e.Code] = e.Generates
		co[e.Code] = e.Controls
	}
	return se, sy, g, co
}

func tenGod(day, target string) string {
	d, t := stemElement[day], stemElement[target]
	sameYang := stemYang[day] == stemYang[target]
	switch {
	case d == t:
		if sameYang {
			return "比肩"
		}
		return "劫财"
	case generates[t] == d:
		if sameYang {
			return "偏印"
		}
		return "正印"
	case generates[d] == t:
		if sameYang {
			return "食神"
		}
		return "伤官"
	case controls[t] == d:
		if sameYang {
			return "七杀"
		}
		return "正官"
	case controls[d] == t:
		if sameYang {
			return "偏财"
		}
		return "正财"
	}
	return ""
}

func category(god string) string {
	switch god {
	case "正印", "偏印", "比肩", "劫财":
		return "support"
	default:
		return "restraint"
	}
}

func elementCategory(dayElement, element string) string {
	if element == dayElement || generates[element] == dayElement {
		return "support"
	}
	return "restraint"
}

func monthSupports(monthElement, target string) bool {
	return monthElement == target || generates[monthElement] == target
}
