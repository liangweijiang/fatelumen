export interface TimeZoneOption {
  value: string;
  label: string;
}

const fallbackTimeZones = [
  "UTC", "Asia/Shanghai", "Asia/Urumqi", "Asia/Hong_Kong", "Asia/Macau", "Asia/Taipei",
  "Asia/Tokyo", "Asia/Seoul", "Asia/Singapore", "Asia/Kolkata", "Asia/Dubai",
  "Europe/London", "Europe/Paris", "Europe/Berlin", "Europe/Moscow",
  "America/New_York", "America/Chicago", "America/Denver", "America/Los_Angeles",
  "America/Anchorage", "Pacific/Honolulu", "America/Toronto", "America/Vancouver",
  "Australia/Sydney", "Australia/Melbourne", "Australia/Brisbane", "Australia/Perth",
];

export function supportedTimeZones(): string[] {
  try {
    const intl = Intl as typeof Intl & { supportedValuesOf?: (key: "timeZone") => string[] };
    return Array.from(new Set([...fallbackTimeZones, ...(intl.supportedValuesOf?.("timeZone") ?? [])])).sort();
  } catch {
    return fallbackTimeZones;
  }
}

function offsetLabel(timeZone: string, date: Date): string {
  try {
    const part = new Intl.DateTimeFormat("en", {
      timeZone,
      timeZoneName: "longOffset",
      year: "numeric",
    }).formatToParts(date).find((item) => item.type === "timeZoneName")?.value;
    return part?.replace("GMT", "UTC") ?? "";
  } catch {
    return "";
  }
}

export function timeZoneOptions(date: Date): TimeZoneOption[] {
  return supportedTimeZones().map((value) => {
    const offset = offsetLabel(value, date);
    return { value, label: offset ? `${value} · ${offset}` : value };
  });
}
