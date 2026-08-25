export type SupportedLocale = "en" | "zh" | "ja" | "ko";

type Labels = Record<SupportedLocale, string>;

export interface RegionOption {
  code: string;
  labels: Labels;
  timezone: string;
}

export interface CountryOption {
  code: string;
  labels: Labels;
  regions: RegionOption[];
}

const region = (code: string, timezone: string, en: string, zh: string, ja: string, ko: string): RegionOption => ({
  code,
  timezone,
  labels: { en, zh, ja, ko },
});

export const countries: CountryOption[] = [
  {
    code: "CN",
    labels: { en: "China", zh: "中国", ja: "中国", ko: "중국" },
    regions: [
      region("BJ", "Asia/Shanghai", "Beijing", "北京", "北京", "베이징"),
      region("SH", "Asia/Shanghai", "Shanghai", "上海", "上海", "상하이"),
      region("GD", "Asia/Shanghai", "Guangdong", "广东", "広東省", "광둥성"),
      region("ZJ", "Asia/Shanghai", "Zhejiang", "浙江", "浙江省", "저장성"),
      region("JS", "Asia/Shanghai", "Jiangsu", "江苏", "江蘇省", "장쑤성"),
      region("SC", "Asia/Shanghai", "Sichuan", "四川", "四川省", "쓰촨성"),
      region("HB", "Asia/Shanghai", "Hubei", "湖北", "湖北省", "후베이성"),
      region("HN", "Asia/Shanghai", "Hunan", "湖南", "湖南省", "후난성"),
      region("FJ", "Asia/Shanghai", "Fujian", "福建", "福建省", "푸젠성"),
      region("SD", "Asia/Shanghai", "Shandong", "山东", "山東省", "산둥성"),
      region("HA", "Asia/Shanghai", "Henan", "河南", "河南省", "허난성"),
      region("HE", "Asia/Shanghai", "Hebei", "河北", "河北省", "허베이성"),
      region("CQ", "Asia/Shanghai", "Chongqing", "重庆", "重慶", "충칭"),
      region("TJ", "Asia/Shanghai", "Tianjin", "天津", "天津", "톈진"),
      region("SN", "Asia/Shanghai", "Shaanxi", "陕西", "陝西省", "산시성"),
      region("LN", "Asia/Shanghai", "Liaoning", "辽宁", "遼寧省", "랴오닝성"),
      region("JL", "Asia/Shanghai", "Jilin", "吉林", "吉林省", "지린성"),
      region("HL", "Asia/Shanghai", "Heilongjiang", "黑龙江", "黒竜江省", "헤이룽장성"),
      region("GX", "Asia/Shanghai", "Guangxi", "广西", "広西", "광시"),
      region("YN", "Asia/Shanghai", "Yunnan", "云南", "雲南省", "윈난성"),
      region("GZ", "Asia/Shanghai", "Guizhou", "贵州", "貴州省", "구이저우성"),
      region("AH", "Asia/Shanghai", "Anhui", "安徽", "安徽省", "안후이성"),
      region("JX", "Asia/Shanghai", "Jiangxi", "江西", "江西省", "장시성"),
      region("SX", "Asia/Shanghai", "Shanxi", "山西", "山西省", "산시성"),
      region("GS", "Asia/Shanghai", "Gansu", "甘肃", "甘粛省", "간쑤성"),
      region("HI", "Asia/Shanghai", "Hainan", "海南", "海南省", "하이난성"),
      region("NM", "Asia/Shanghai", "Inner Mongolia", "内蒙古", "内モンゴル", "내몽골"),
      region("NX", "Asia/Shanghai", "Ningxia", "宁夏", "寧夏", "닝샤"),
      region("QH", "Asia/Shanghai", "Qinghai", "青海", "青海省", "칭하이성"),
      region("XJ", "Asia/Urumqi", "Xinjiang", "新疆", "新疆", "신장"),
      region("XZ", "Asia/Shanghai", "Tibet", "西藏", "チベット", "티베트"),
      region("HK", "Asia/Hong_Kong", "Hong Kong", "香港", "香港", "홍콩"),
      region("MO", "Asia/Macau", "Macao", "澳门", "マカオ", "마카오"),
      region("TW", "Asia/Taipei", "Taiwan", "台湾", "台湾", "대만"),
    ],
  },
  {
    code: "US",
    labels: { en: "United States", zh: "美国", ja: "アメリカ合衆国", ko: "미국" },
    regions: [
      region("CA", "America/Los_Angeles", "California", "加利福尼亚州", "カリフォルニア州", "캘리포니아주"),
      region("NY", "America/New_York", "New York", "纽约州", "ニューヨーク州", "뉴욕주"),
      region("TX", "America/Chicago", "Texas", "得克萨斯州", "テキサス州", "텍사스주"),
      region("FL", "America/New_York", "Florida", "佛罗里达州", "フロリダ州", "플로리다주"),
      region("IL", "America/Chicago", "Illinois", "伊利诺伊州", "イリノイ州", "일리노이주"),
      region("WA", "America/Los_Angeles", "Washington", "华盛顿州", "ワシントン州", "워싱턴주"),
      region("MA", "America/New_York", "Massachusetts", "马萨诸塞州", "マサチューセッツ州", "매사추세츠주"),
      region("HI", "Pacific/Honolulu", "Hawaii", "夏威夷州", "ハワイ州", "하와이주"),
      region("AK", "America/Anchorage", "Alaska", "阿拉斯加州", "アラスカ州", "알래스카주"),
    ],
  },
  {
    code: "JP",
    labels: { en: "Japan", zh: "日本", ja: "日本", ko: "일본" },
    regions: [
      region("13", "Asia/Tokyo", "Tokyo", "东京都", "東京都", "도쿄도"),
      region("27", "Asia/Tokyo", "Osaka", "大阪府", "大阪府", "오사카부"),
      region("14", "Asia/Tokyo", "Kanagawa", "神奈川县", "神奈川県", "가나가와현"),
      region("23", "Asia/Tokyo", "Aichi", "爱知县", "愛知県", "아이치현"),
      region("01", "Asia/Tokyo", "Hokkaido", "北海道", "北海道", "홋카이도"),
      region("40", "Asia/Tokyo", "Fukuoka", "福冈县", "福岡県", "후쿠오카현"),
      region("26", "Asia/Tokyo", "Kyoto", "京都府", "京都府", "교토부"),
      region("28", "Asia/Tokyo", "Hyogo", "兵库县", "兵庫県", "효고현"),
      region("47", "Asia/Tokyo", "Okinawa", "冲绳县", "沖縄県", "오키나와현"),
    ],
  },
  {
    code: "KR",
    labels: { en: "South Korea", zh: "韩国", ja: "韓国", ko: "대한민국" },
    regions: [
      region("11", "Asia/Seoul", "Seoul", "首尔", "ソウル", "서울특별시"),
      region("26", "Asia/Seoul", "Busan", "釜山", "釜山", "부산광역시"),
      region("27", "Asia/Seoul", "Daegu", "大邱", "大邱", "대구광역시"),
      region("28", "Asia/Seoul", "Incheon", "仁川", "仁川", "인천광역시"),
      region("41", "Asia/Seoul", "Gyeonggi", "京畿道", "京畿道", "경기도"),
      region("42", "Asia/Seoul", "Gangwon", "江原道", "江原道", "강원특별자치도"),
      region("48", "Asia/Seoul", "South Gyeongsang", "庆尚南道", "慶尚南道", "경상남도"),
      region("50", "Asia/Seoul", "Jeju", "济州", "済州", "제주특별자치도"),
    ],
  },
  {
    code: "GB",
    labels: { en: "United Kingdom", zh: "英国", ja: "イギリス", ko: "영국" },
    regions: [
      region("ENG", "Europe/London", "England", "英格兰", "イングランド", "잉글랜드"),
      region("SCT", "Europe/London", "Scotland", "苏格兰", "スコットランド", "스코틀랜드"),
      region("WLS", "Europe/London", "Wales", "威尔士", "ウェールズ", "웨일스"),
      region("NIR", "Europe/London", "Northern Ireland", "北爱尔兰", "北アイルランド", "북아일랜드"),
    ],
  },
  {
    code: "CA",
    labels: { en: "Canada", zh: "加拿大", ja: "カナダ", ko: "캐나다" },
    regions: [
      region("ON", "America/Toronto", "Ontario", "安大略省", "オンタリオ州", "온타리오주"),
      region("BC", "America/Vancouver", "British Columbia", "不列颠哥伦比亚省", "ブリティッシュコロンビア州", "브리티시컬럼비아주"),
      region("QC", "America/Toronto", "Quebec", "魁北克省", "ケベック州", "퀘벡주"),
      region("AB", "America/Edmonton", "Alberta", "艾伯塔省", "アルバータ州", "앨버타주"),
    ],
  },
  {
    code: "AU",
    labels: { en: "Australia", zh: "澳大利亚", ja: "オーストラリア", ko: "호주" },
    regions: [
      region("NSW", "Australia/Sydney", "New South Wales", "新南威尔士州", "ニューサウスウェールズ州", "뉴사우스웨일스주"),
      region("VIC", "Australia/Melbourne", "Victoria", "维多利亚州", "ビクトリア州", "빅토리아주"),
      region("QLD", "Australia/Brisbane", "Queensland", "昆士兰州", "クイーンズランド州", "퀸즐랜드주"),
      region("WA", "Australia/Perth", "Western Australia", "西澳大利亚州", "西オーストラリア州", "웨스턴오스트레일리아주"),
    ],
  },
  {
    code: "SG",
    labels: { en: "Singapore", zh: "新加坡", ja: "シンガポール", ko: "싱가포르" },
    regions: [region("SG", "Asia/Singapore", "Singapore", "新加坡", "シンガポール", "싱가포르")],
  },
];

export function normalizeLocale(locale: string): SupportedLocale {
  return locale === "zh" || locale === "ja" || locale === "ko" ? locale : "en";
}
