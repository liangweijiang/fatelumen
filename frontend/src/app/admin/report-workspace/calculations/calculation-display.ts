// Compatibility facade. New admin views should import from @/lib/bazi-display.
export {
  dictionaries,
  displayCode,
  displayCoordinates,
  displayDateTime,
  displayNumber,
  displayUTCOffset,
  hasDisplayTranslation,
  normalizeBaziDisplayLocale,
} from "@/lib/bazi-display";
export type {BaziDisplayDictionary,BaziDisplayLocale} from "@/lib/bazi-display";
