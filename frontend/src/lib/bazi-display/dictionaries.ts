export type BaziDisplayLocale = "zh" | "en" | "ja" | "ko";
export type BaziDisplayDictionary = Readonly<Record<string, string>>;

export const zhDictionary: BaziDisplayDictionary = {
  wood:"木",fire:"火",earth:"土",metal:"金",water:"水",
  extremely_strong:"极强",strong:"身强",slightly_strong:"身偏强",balanced:"中和",slightly_weak:"身偏弱",weak:"身弱",extremely_weak:"极弱",
  yang:"阳",yin:"阴",male:"男",female:"女",normal:"普通格",false_following:"假从格",candidate:"候选",confirmed:"已确认",insufficient:"不足",present:"已存在",missing:"缺失",
  strong_root:"强根",medium_root:"中根",weak_root:"弱根",no_root:"无根",primary:"首选",secondary:"次选",favorable:"正向候选",taboo:"忌神",enemy:"仇神",neutral:"中性",
  following:"从格候选",dominant:"专旺格候选",transformation:"化气格候选",valid:"有效",invalid:"无效",matched:"命中",rejected:"未命中",pending:"待判定",
  severe_cold:"极寒",cold:"偏寒",balanced_temperature:"寒热适中",hot:"偏热",severe_hot:"极热",severe_dry:"极燥",dry:"偏燥",balanced_moisture:"燥湿适中",wet:"偏湿",severe_wet:"极湿",
  MIDNIGHT_00:"零点换日",ZI_HOUR_23:"子初换日（23:00）",TRUE_SOLAR_TIME:"真太阳时",support:"扶助",pressure:"克泄耗",resource:"印星",peer:"比劫",output:"食伤",wealth:"财星",officer:"官杀",
  combination:"天干五合",stem_combination:"天干五合",branch_combination:"地支六合",six_combination:"地支六合",combine:"合",clash:"地支相冲",punishment:"地支相刑",harm:"地支相害",break:"地支相破",three_harmony:"地支三合",three_meeting:"地支三会",half_harmony:"地支半合",long_life:"十二长生",
  partial:"部分成立",full:"完全成立",secondary_neutral:"中性证据",transformed:"已合化",not_transformed:"未合化",untransformed:"未合化",single:"单一",double_main:"双主气",double_residual:"双余气",hidden_stem:"藏干",stored_qi_preserved:"库气保留",availability_reduced:"可用性降低",balanced_clash:"势均相冲",strong_weak_clash:"强弱相冲",
  primary_useful:"首选用神",secondary_useful:"次选用神",pending_review:"待复核候选",proper_officer:"正官格",seven_killings:"七杀格",proper_wealth:"正财格",indirect_wealth:"偏财格",food_god:"食神格",hurting_officer:"伤官格",proper_resource:"正印格",indirect_resource:"偏印格",
  food_controls_killing:"食神制杀",killing_resource_cycle:"杀印相生",officer_resource_cycle:"官印相生",food_generates_wealth:"食神生财",output_generates_wealth:"食伤生财",wealth_generates_officer:"财生官",output_resource_balance:"伤官配印",
  effective_root_present:"存在有效根气",support_not_minimal:"扶助力量并非极低",no_confirmed_special_pattern:"未确认特殊格局",day_master_not_weak:"日主不弱",effective_ten_god_present:"存在有效十神",effective_category_and_ten_god_powers:"依据有效十神类别及力量",day_master_too_strong:"日主过强",day_master_too_weak:"日主过弱",
  "effective ten-god present":"存在有效十神","effective category and ten-god powers":"依据有效十神类别及力量",severe_climate_worsened:"使寒热燥湿进一步失衡",confirmed_special_pattern_damaged:"破坏已确认的特殊格局",new_major_imbalance:"产生新的明显失衡",
  improves_climate:"改善调候",addresses_primary_disease:"处理主要病点",positive_marginal_utility:"模拟后产生正向收益",present_in_natal_chart:"原局中具有可用性",non_positive_marginal_utility:"模拟后的边际收益不为正",does_not_resolve_primary_disease:"未能处理当前主要病点",base:"基础值",obvious:"明显",severe:"严重",minor:"轻微",
  "month branch base temperature":"以月支确定基础寒热值","continuous interpolation between month-branch coefficients":"按节气进度连续插值月令系数","temperature above climate threshold":"温度高于调候阈值","temperature below climate threshold":"温度低于调候阈值","moisture above climate threshold":"湿度高于调候阈值","moisture below climate threshold":"湿度低于调候阈值",
  "strength score above 15":"身强评分高于15分","strength score below -15":"身弱评分低于-15分","effective category exceeds threshold":"有效十神类别超过阈值","mediation conflict detected":"检测到通关冲突","both sides exceed minimum conflict power":"冲突双方力量均超过最低阈值","ratio is within 0.60..1.67":"双方力量比例处于0.60至1.67之间",
  calculation_year_to_plus_9:"计算当年起连续10年",annual_calendar_years:"公共干支日历表","sexagenary-calendar-v1":"六十甲子日历第1版",
  climate_global_adjustment_coefficients_require_domain_calibration:"调候全局修正系数仍需命理规则校准",pattern_presence_thresholds_require_domain_calibration:"格局成立阈值仍需命理规则校准",disease_severity_coefficients_require_domain_calibration:"病药严重程度系数仍需命理规则校准",minimum_conflict_and_bridge_thresholds_require_domain_calibration:"通关冲突与桥接元素的最低阈值仍需命理规则校准",candidate_simulation_uses_versioned_engineering_health_weights:"候选元素模拟目前使用版本化工程权重",candidate_simulation_uses_rule_aligned_dynamic_health_weights:"候选模拟与静态判断共用同一组动态权重",availability_root_protection_coefficients_require_domain_calibration:"元素可用性与根气保护系数仍需命理规则校准",
  "peer*1 + resource*.85 versus output*.70 + wealth*.75 + officer*.95":"扶助力量＝比劫×1＋印星×0.85；克泄耗力量＝食伤×0.70＋财星×0.75＋官杀×0.95","100*(support-pressure)/(support+pressure)":"基础分＝100×（扶助力量－克泄耗力量）÷（扶助力量＋克泄耗力量）","base*.70 + deLing*.10 + deDi*.12 + deShi*.08 after normalization":"归一化后按基础分70%、得令10%、得地12%、得势8%进行稳定修正",
  calculating:"计算中",ready:"计算完成",failed:"计算失败",true:"是",false:"否",
};

export const enDictionary: BaziDisplayDictionary = {wood:"Wood",fire:"Fire",earth:"Earth",metal:"Metal",water:"Water",male:"Male",female:"Female",primary_useful:"Primary useful element",secondary_useful:"Secondary useful element",pending_review:"Pending review",calculating:"Calculating",ready:"Ready",failed:"Failed",true:"Yes",false:"No"};
export const dictionaries: Readonly<Record<BaziDisplayLocale, BaziDisplayDictionary>> = {zh:zhDictionary,en:enDictionary,ja:{},ko:{}};

export function normalizeBaziDisplayLocale(locale?: string): BaziDisplayLocale {
  const value=locale?.toLowerCase().split("-")[0];
  return value==="en"||value==="ja"||value==="ko"?value:"zh";
}
