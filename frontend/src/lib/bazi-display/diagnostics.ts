import { normalizeBaziDisplayLocale, type BaziDisplayLocale } from "./dictionaries";

// 计算过程中的诊断原因，不是命理术语；旧快照仍保留原文，只在界面转换。
const zhReasons: Readonly<Record<string, string>> = {
  "combination without transformation":"干支相合，但未达到合化条件",
  "self punishment":"地支自刑修正",
  "three punishment adjustment":"地支三刑修正",
  "clash adjustment":"地支相冲修正",
  "harm adjustment":"地支相害修正",
  "break adjustment":"地支相破修正",
  "punishment adjustment":"地支相刑修正",
  "half harmony adjustment":"地支半合修正",
  "three harmony adjustment":"地支三合修正",
  "three meeting adjustment":"地支三会修正",
  "month branch base temperature":"以月支确定基础寒热值",
  "fire/water effective ratio adjustment":"按火水有效力量比例进行全局修正",
  "position weights and hidden-stem ratios":"依据宫位权重与藏干占比分配原始力量",
  "continuous interpolation between month-branch coefficients":"按节气进度连续插值月令系数",
  "versioned combinations, clashes and availability modifiers":"依据版本化合冲刑害及可用性规则修正",
  "generation/control, structure adjustment, then half-rate second pass":"先计算生克与结构修正，再以半倍率进行第二轮迭代",
  "month command supports transformation":"月令支持合化",
  "complete group":"完整组合成立",
  "half group supported by month command":"半合组合得到月令支持",
  "hidden stems already carry power; tomb storage creates no extra energy":"藏干已计入力量，墓库不再额外增加能量",
  "stage retained as evidence; neutral 1.00 modifier until calibrated":"十二长生仅保留为依据，校准前采用中性系数 1.00",
  "effective category allocated by original stem/hidden-stem yin-yang share":"按原始天干与藏干的阴阳占比分配有效十神类别力量",
  "existing strength evidence":"沿用既有身强弱判定依据",
  "position weight and hidden-stem level projected from strength analysis":"依据身强弱分析中的宫位权重与藏干层级折算",
  "peer*1 + resource*.85 versus output*.70 + wealth*.75 + officer*.95":"扶助力量＝比劫×1＋印星×0.85；克泄耗力量＝食伤×0.70＋财星×0.75＋官杀×0.95",
  "100*(support-pressure)/(support+pressure)":"基础分＝100×（扶助力量－克泄耗力量）÷（扶助力量＋克泄耗力量）",
  "base*.70 + deLing*.10 + deDi*.12 + deShi*.08 after normalization":"归一化后按基础分70%、得令10%、得地12%、得势8%进行稳定修正",
};

export function displayDiagnostic(value: unknown, locale: BaziDisplayLocale | string = "zh"): string {
  if (value === undefined || value === null || value === "") return "—";
  if (Array.isArray(value)) return value.length ? value.map(item => displayDiagnostic(item, locale)).join("、") : "—";
  const raw = String(value);
  if (normalizeBaziDisplayLocale(locale) !== "zh") return raw;
  const direct = zhReasons[raw] ?? zhReasons[raw.toLowerCase()];
  if (direct) return direct;
  const availability = raw.match(/^availability retention ([\d.]+)$/i);
  if (availability) return `可用力量保留系数为 ${availability[1]}`;
  const retention = raw.match(/^retention ([\d.]+)\/([\d.]+) ratio ([\d.]+)$/i);
  if (retention) return `双方保留系数为 ${retention[1]} / ${retention[2]}，力量比为 ${retention[3]}`;
  return raw;
}
