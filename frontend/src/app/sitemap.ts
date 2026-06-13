import type { MetadataRoute } from "next";
import { SITE } from "@/lib/site";

export default function sitemap(): MetadataRoute.Sitemap {
  const paths = ["", "/login", "/privacy", "/terms", "/contact"];
  return paths.map((p) => ({
    url: `${SITE.url}/${SITE.defaultLocale}${p}`,
    lastModified: new Date(),
    changeFrequency: "weekly" as const,
    priority: p === "" ? 1 : 0.6,
    alternates: {
      languages: Object.fromEntries(
        SITE.locales.map((l) => [l, `${SITE.url}/${l}${p}`])
      ),
    },
  }));
}
