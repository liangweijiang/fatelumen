import {lookupDisplayTranslation,normalizeBaziDisplayLocale,type BaziDisplayLocale} from "./dictionaries";
export type {BaziDisplayDictionary,BaziDisplayLocale} from "./dictionaries";
export {dictionaries,normalizeBaziDisplayLocale} from "./dictionaries";

export function displayCode(value:unknown,locale:BaziDisplayLocale|string="zh",category?:string):string{
  if(value===undefined||value===null||value==="")return "—";
  if(typeof value==="boolean")return displayCode(String(value),locale,category);
  if(typeof value==="number")return displayNumber(value);
  if(Array.isArray(value)){const separator=normalizeBaziDisplayLocale(locale)==="en"?", ":"、";return value.length?value.map(item=>displayCode(item,locale)).join(separator):"—";}
  const raw=String(value),normalized=normalizeBaziDisplayLocale(locale);
  // 中文、日文和韩文界面不得悄悄回退为英文；缺失时保留稳定原始码便于补录。
  return lookupDisplayTranslation(raw,normalized,category)||raw;
}
export function hasDisplayTranslation(code:string,locale:BaziDisplayLocale|string="zh",category?:string):boolean{return Boolean(lookupDisplayTranslation(code,locale,category));}
export function displayNumber(value:number,digits=4):string{if(!Number.isFinite(value))return "—";if(Number.isInteger(value))return String(value);return value.toFixed(digits).replace(/0+$/,"").replace(/\.$/,"");}
export function displayUTCOffset(value:unknown,locale:BaziDisplayLocale|string="zh"):string{const seconds=Number(value);if(!Number.isFinite(seconds))return displayCode(value,locale);const sign=seconds<0?"-":"+",absolute=Math.abs(seconds),hours=Math.floor(absolute/3600),minutes=Math.floor(absolute%3600/60);return `UTC${sign}${String(hours).padStart(2,"0")}:${String(minutes).padStart(2,"0")}`;}
export function displayDateTime(value:unknown,locale:BaziDisplayLocale|string="zh"):string{if(typeof value!=="string"||!value)return displayCode(value,locale);const match=value.match(/^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?$/);if(!match)return value;const normalized=normalizeBaziDisplayLocale(locale);if(normalized==="zh"||normalized==="ja")return `${match[1]}年${Number(match[2])}月${Number(match[3])}日 ${match[4]}:${match[5]}:${match[6]}`;if(normalized==="ko")return `${match[1]}년 ${Number(match[2])}월 ${Number(match[3])}일 ${match[4]}:${match[5]}:${match[6]}`;return `${match[1]}-${match[2]}-${match[3]} ${match[4]}:${match[5]}:${match[6]}`;}
export function displayCoordinates(longitude:unknown,latitude:unknown,locale:BaziDisplayLocale|string="zh"):string{const lng=Number(longitude),lat=Number(latitude);if(!Number.isFinite(lng)||!Number.isFinite(lat))return "—";const normalized=normalizeBaziDisplayLocale(locale),directions={zh:["东经","西经","北纬","南纬"],en:["E","W","N","S"],ja:["東経","西経","北緯","南緯"],ko:["동경","서경","북위","남위"]}[normalized],separator=normalized==="en"?", ":"，";return `${lng>=0?directions[0]:directions[1]} ${displayNumber(Math.abs(lng),4)}°${separator}${lat>=0?directions[2]:directions[3]} ${displayNumber(Math.abs(lat),4)}°`;}
