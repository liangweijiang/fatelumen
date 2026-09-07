import { normalizeBaziDisplayLocale, type BaziDisplayLocale } from "./dictionaries";

// 仅用于管理端 JSON/快照的字段标题，不属于命理专业术语，也不会进入 Prompt。
const zh: Readonly<Record<string, string>> = {
  calendar_type:"历法",year:"年份",month:"月份",day:"日期",hour:"小时",minute:"分钟",is_leap_month:"是否闰月",gender:"性别",
  country_code:"国家代码",region_code:"地区代码",place_id:"地点编号",longitude:"经度",latitude:"纬度",timezone_id:"出生记录时区",locale:"报告语言",profile_mode:"档案模式",target_profile_id:"档案编号",schema_version:"数据结构版本",
  local_civil_time:"当地钟表时间",historical_utc_offset_seconds:"历史 UTC 偏移",standard_utc_offset_seconds:"标准 UTC 偏移",dst_applied:"是否采用夏令时",dst_offset_seconds:"夏令时偏移（秒）",standard_meridian:"标准经线",longitude_correction_minutes:"经度校正（分钟）",equation_of_time_minutes:"均时差（分钟）",local_standard_time:"当地标准时间",mean_solar_time:"平太阳时",true_solar_time:"真太阳时",crossed_date_boundary:"是否跨日",day_boundary_rule:"换日规则",solar_algorithm_version:"真太阳时算法",location_database_version:"地区数据库",timezone_database_version:"时区数据库",
  id:"编号",profile_id:"档案编号",created_at:"创建时间",chart_hash:"命盘哈希",chart_schema_version:"命盘结构版本",input_schema_version:"输入结构版本",facts_schema_version:"事实结构版本",engine_version:"排盘引擎版本",lunar_go_version:"农历库版本",bazi_base_data_version:"命理基础数据版本",prompt_version:"提示词版本",rule_set_version:"规则集版本",data:"命盘数据",chart_data:"命盘数据",facts_hash:"事实哈希",
  calc_lib:"历法计算库",calc_version:"历法计算库版本",lunar_date:"农历日期",solar_date:"公历日期",mode:"计算模式",
  input:"输入资料",time_calculation:"时间换算",chart:"命盘",meta:"基础信息",pillars:"四柱",day_master:"日主",hour_unknown:"时辰是否未知",five_elements_count:"五行数量",element_power:"五行力量",element_strength:"五行力量",strength:"身强弱",strength_v2:"新版身强弱",day_master_strength:"日主强弱",
  ten_god_analysis:"十神分析",ten_god_effective:"有效十神",ten_god_structure:"十神结构",stem_relations:"天干关系",branch_relations:"地支关系",pattern_candidates:"格局候选",climate:"调候分析",pattern:"格局分析",disease:"病药分析",mediation:"通关分析",useful_god:"喜用神分析",favorable_elements:"喜忌元素",luck_cycles:"大运",annual_fortunes:"未来流年",current_year_fortune:"当前流年",annual_fortune_range:"流年范围",chapter_facts:"章节事实索引",rule_matches:"规则命中",warnings:"计算提示",versions:"规则与数据版本",
  code:"代码",type:"类型",category:"分类",level:"结论",root_level:"根气等级",score:"评分",month_score:"月令评分",support_score:"扶助力量",restraint_score:"克泄耗力量",support_ratio:"扶助比例",day_element:"日主五行",rule_version:"规则版本",false_following:"是否假从",unfavorable:"不利元素",values:"结果",evidence:"依据",conflicts:"冲突",analysis:"分析",analysis_v2:"新版分析",matched:"是否命中",priority:"优先级",rule_code:"规则代码",source:"来源",symbols:"相关干支",reason:"原因",reasons:"原因",reject_reasons:"淘汰原因",module:"模块",chapter_key:"章节",fact_codes:"事实项",relations:"关系明细",
  stem:"天干",branch:"地支",hidden_stems:"藏干",nayin:"纳音",palace:"宫位",ten_god:"十神",element:"五行",yin_yang:"阴阳",count:"数量",start_age:"起运年龄",start_year:"起运年份",end_year:"结束年份",gan_zhi:"干支",ganzhi:"干支",zodiac:"生肖",confidence:"可信度",status:"状态",result:"结果",primary:"首选",secondary:"次选",favorable:"喜神",taboo:"忌神",enemy:"仇神",neutral:"中性",
  temperature:"寒热",moisture:"燥湿",temperature_level:"寒热等级",moisture_level:"燥湿等级",temperature_score:"寒热评分",moisture_score:"燥湿评分",formula:"计算公式",threshold:"阈值",thresholds:"阈值配置",coefficients:"系数",weight:"权重",base_weight:"基础权重",ratio:"比例",total:"合计",name:"名称",description:"说明",value:"数值",valid:"是否有效",available:"是否可用",availability:"可用性",effective:"是否生效",transformed:"是否合化",resolves_primary:"是否处理主要病点",primary_required:"是否必须处理主要病点",breakdown:"评分明细",marginal_utility:"边际收益",
  rule:"规则",base:"基础值",rank:"排名",symbol:"符号",candidates:"候选项",candidate_elements:"候选元素",categories:"类别汇总",requirements:"成立条件",simulations:"模拟结果",side_effects:"副作用",trace:"计算轨迹",interactions:"五行交互",structures:"结构关系",contributions:"力量贡献",positions:"位置",position:"位置",all:"全部",
  adjustment:"修正值",relation_adjustment:"关系修正",before:"修正前",after:"修正后",source_before:"来源修正前",target_before:"目标修正前",source_element:"来源五行",target_element:"目标五行",amount:"作用量",efficiency:"作用效率",contact_factor:"接触系数",ratio_factor:"强弱比系数",iteration:"迭代轮次",transfer_rate:"转化比例",
  raw_power:"原始力量",seasonal_power:"旺衰修正后力量",effective_power:"有效力量",power:"力量",root_power:"根气力量",controlled_power:"受制力量",controller_power:"制约方力量",bridge_power:"通关力量",season_coefficient:"月令旺衰系数",visibility_multiplier:"透干系数",hidden_ratio:"藏干占比",hidden_level:"藏干层级",effective_ratio:"有效占比",effective_score:"有效评分",raw_score:"原始评分",base_score:"基础评分",total_score:"总评分",strength_score:"身强弱评分",flow_score:"流通评分",pattern_integrity:"格局完整度",disease_severity:"病点程度",concentration:"集中度",quality:"质量",severity:"严重程度",progress:"节气进度",delta:"变化量",
  state:"状态",subtype:"子类型",pattern_subtype:"格局子类型",role:"作用角色",scope:"作用范围",flow:"生克流向",bridge:"通关元素",controlled:"受制方",controller:"制约方",rooted:"是否有根",visible:"是否透出",rejected_by:"未通过条件",optimal_range:"合理范围",selection_rule:"选取规则",data_version:"数据版本",
  dominant_gods:"主导十神",secondary_gods:"辅助十神",missing_gods:"缺失十神",visible_gods:"透出十神",rooted_gods:"有根十神",gods:"十神",patterns:"格局",ten_god_stem:"天干十神",ten_god_hidden:"藏干十神",
  branch_element:"地支五行",stem_element:"天干五行",branch_yin_yang:"地支阴阳",stem_yin_yang:"天干阴阳",day_stem:"日干",next_branch:"下一地支",
  wood:"木",fire:"火",earth:"土",metal:"金",water:"水",support:"扶助",pressure:"克泄耗",
  element_power_rule_version:"五行力量规则版本",strength_v2_rule_version:"身强弱规则版本",ten_god_rule_version:"十神规则版本",ten_god_effective_version:"有效十神规则版本",climate_rule_version:"调候规则版本",pattern_rule_version:"格局规则版本",disease_rule_version:"病药规则版本",mediation_rule_version:"通关规则版本",useful_god_rule_version:"喜用神规则版本",
  de_ling:"得令",de_di:"得地",de_shi:"得势",fu_yi:"扶抑",season:"季节",roots:"根气",health:"健康度",minor:"次要项",alternative:"备选项",
  cycle_index:"大运序号",luck_cycle_ganzhi:"所属大运干支",luck_cycle_start_age:"大运起始年龄",luck_cycle_start_year:"大运起始年份",age:"年龄",
};

const pathZh: Readonly<Record<string, string>> = {
  "input.year":"出生年","input.month":"出生月","input.day":"出生日","input.hour":"出生时","input.minute":"出生分",
  "analysis.type":"分析类型","relations.type":"关系类型","stem_relations.type":"天干关系类型","branch_relations.type":"地支关系类型",
};

export function displayFieldLabel(key: string, path: readonly string[] = [], locale: BaziDisplayLocale | string = "zh"): string {
  if (normalizeBaziDisplayLocale(locale) !== "zh") return key;
  const fullPath = [...path, key].join(".");
  return pathZh[fullPath] ?? zh[key] ?? key;
}

export function displayArrayItemLabel(key: string, index: number, path: readonly string[] = []): string {
  return `${displayFieldLabel(key, path)} ${index + 1}`;
}
