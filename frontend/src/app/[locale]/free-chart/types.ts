export type ChartPillar = {
  key: "hourPillar" | "dayPillar" | "monthPillar" | "yearPillar";
  stem: string;
  branch: string;
  stemElement: string;
  branchElement: string;
};

export type ChartRecord = {
  id: number;
  solarDate: string;
  lunarDate: string;
  gender: 0 | 1;
  place: string;
  coordinates: string;
	timezone: string;
	trueSolarTime: string;
  dayMaster: string;
  createdAt: string;
  pillars: ChartPillar[];
};
