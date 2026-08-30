export type FactOption = { key: string; label: string; group: string };
export type ChapterDefinition = { no: number; key: string; title: string; purpose: string; defaults: string[] };
export type CalculationRecord = { id: number; name: string; birth: string; place: string; timezone: string; pillars: string[]; factsHash: string; versions: number; updatedAt: string };

export const factOptions: FactOption[] = [
  {key:"input.calendar",label:"历法与出生时间",group:"输入与时间"},{key:"time.solar",label:"真太阳时与跨日",group:"输入与时间"},{key:"time.timezone",label:"历史时区与偏移",group:"输入与时间"},
  {key:"chart.pillars",label:"四柱干支",group:"命盘"},{key:"chart.hidden_stems",label:"藏干与权重",group:"命盘"},{key:"chart.na_yin",label:"纳音",group:"命盘"},{key:"chart.palaces",label:"柱位与宫位",group:"命盘"},
  {key:"power.raw",label:"五行原始力量",group:"力量与关系"},{key:"power.effective",label:"五行有效力量",group:"力量与关系"},{key:"relations.stems",label:"天干生克冲合",group:"力量与关系"},{key:"relations.branches",label:"地支合冲刑害破",group:"力量与关系"},{key:"relations.transform",label:"合化过程与证据",group:"力量与关系"},
  {key:"strength.result",label:"身强弱结论",group:"确定性推断"},{key:"strength.trace",label:"得令得地得势与轨迹",group:"确定性推断"},{key:"ten_gods.raw",label:"原始十神",group:"确定性推断"},{key:"ten_gods.effective",label:"有效十神",group:"确定性推断"},{key:"pattern.result",label:"格局与候选",group:"确定性推断"},{key:"climate.result",label:"调候",group:"确定性推断"},{key:"disease.result",label:"病药",group:"确定性推断"},{key:"mediation.result",label:"通关",group:"确定性推断"},{key:"useful_god.result",label:"喜用忌仇闲",group:"确定性推断"},{key:"useful_god.simulation",label:"喜用神模拟过程",group:"确定性推断"},
  {key:"luck.cycles",label:"大运",group:"运势"},{key:"luck.years",label:"未来十年流年",group:"运势"},{key:"meta.evidence",label:"规则证据",group:"追溯"},{key:"meta.warnings",label:"计算警告",group:"追溯"},{key:"meta.versions",label:"规则与算法版本",group:"追溯"},
];

export const chapterRequired: Record<string,string[]> = {
  destiny_depth:["chart.pillars","strength.result"],ten_gods_full:["chart.pillars","ten_gods.effective"],luck_cycle:["chart.pillars","luck.cycles"],ten_year_years:["chart.pillars","luck.cycles","luck.years"],career_depth:["chart.pillars","ten_gods.effective"],wealth_depth:["chart.pillars","ten_gods.effective"],love_depth:["chart.pillars","chart.palaces"],health_depth:["power.effective","climate.result"],element_tuning:["power.effective","useful_god.result"],life_plan:["chart.pillars","luck.cycles","luck.years"],
};

export const chapters: ChapterDefinition[] = [
  {no:1,key:"destiny_depth",title:"命格深析",purpose:"解释命局结构、强弱与核心矛盾",defaults:["chart.pillars","chart.hidden_stems","power.effective","relations.stems","relations.branches","strength.result","strength.trace","pattern.result","climate.result","useful_god.result"]},
  {no:2,key:"ten_gods_full",title:"十神全览",purpose:"解释十神结构及其现实表现",defaults:["chart.pillars","chart.hidden_stems","chart.palaces","strength.result","ten_gods.raw","ten_gods.effective","pattern.result"]},
  {no:3,key:"luck_cycle",title:"大运走势",purpose:"解释人生阶段与大运转换",defaults:["chart.pillars","strength.result","pattern.result","useful_god.result","luck.cycles","luck.years"]},
  {no:4,key:"ten_year_years",title:"未来十年流年",purpose:"逐年解释固定流年数据",defaults:["chart.pillars","power.effective","strength.result","useful_god.result","luck.cycles","luck.years","relations.branches"]},
  {no:5,key:"career_depth",title:"事业深析",purpose:"解释事业模式、角色与发展节奏",defaults:["chart.pillars","strength.result","ten_gods.effective","pattern.result","useful_god.result","luck.cycles","luck.years"]},
  {no:6,key:"wealth_depth",title:"财富格局",purpose:"解释财富结构、路径与风险",defaults:["chart.pillars","strength.result","ten_gods.effective","pattern.result","useful_god.result","luck.cycles","luck.years","relations.branches"]},
  {no:7,key:"love_depth",title:"情感姻缘",purpose:"解释夫妻宫、关系模式与阶段变化",defaults:["chart.pillars","chart.palaces","ten_gods.effective","relations.stems","relations.branches","luck.cycles","luck.years"]},
  {no:8,key:"health_depth",title:"健康养生",purpose:"给出非诊断性的生活方式建议",defaults:["power.effective","climate.result","disease.result","mediation.result","useful_god.result","luck.cycles"]},
  {no:9,key:"element_tuning",title:"五行调候",purpose:"解释五行环境与调节方向",defaults:["power.raw","power.effective","climate.result","disease.result","mediation.result","useful_god.result","useful_god.simulation"]},
  {no:10,key:"life_plan",title:"人生规划",purpose:"基于确定性数据形成阶段行动地图",defaults:["chart.pillars","power.effective","strength.result","pattern.result","climate.result","useful_god.result","luck.cycles","luck.years"]},
];

export const initialRecords: CalculationRecord[] = [
  {id:1042,name:"上海 · 回归样例",birth:"1990-01-01 12:00",place:"中国 / 上海",timezone:"Asia/Shanghai",pillars:["己巳","丙子","丙寅","甲午"],factsHash:"9fc2a4…71de",versions:3,updatedAt:"2026-08-28 14:32"},
  {id:1038,name:"纽约 · 时区边界",birth:"1988-11-06 01:30",place:"美国 / New York",timezone:"America/New_York",pillars:["戊辰","壬戌","甲子","乙丑"],factsHash:"38b0de…0a92",versions:2,updatedAt:"2026-08-27 21:10"},
  {id:1027,name:"东京 · 标准样例",birth:"2001-07-12 08:15",place:"日本 / Tokyo",timezone:"Asia/Tokyo",pillars:["辛巳","乙未","丙子","壬辰"],factsHash:"8ad691…c43f",versions:1,updatedAt:"2026-08-25 09:46"},
];
