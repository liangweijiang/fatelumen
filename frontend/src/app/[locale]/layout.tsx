import { NextIntlClientProvider } from "next-intl";
import { getMessages, getTranslations, setRequestLocale } from "next-intl/server";
import { routing } from "@/i18n/routing";
import { notFound } from "next/navigation";
import ThemeProvider from "@/components/theme/ThemeProvider";
import StickyHeader from "@/components/common/StickyHeader";
import WhatsAppFab from "@/components/common/WhatsAppFab";
import Providers from "@/app/providers";
import BrandMark from "@/components/common/BrandMark";
import Link from "next/link";

export function generateStaticParams() {
  return routing.locales.map((locale) => ({ locale }));
}

export default async function LocaleLayout({
  children,
  params,
}: {
  children: React.ReactNode;
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;

  if (!routing.locales.includes(locale as "en" | "zh" | "ja" | "ko")) {
    notFound();
  }
  setRequestLocale(locale);

  const messages = await getMessages();
  const t = await getTranslations({ locale, namespace: "nav" });
  const tf = await getTranslations({ locale, namespace: "footer" });

  return (
    <NextIntlClientProvider messages={messages}>
      <ThemeProvider>
        <Providers>
          <link href="https://fonts.googleapis.com/css2?family=Playfair+Display:ital,wght@0,500;0,600;0,700;1,500;1,600&family=Noto+Serif+SC:wght@300;400;500;600;700&display=swap" rel="stylesheet" />
          <StickyHeader locale={locale} />
          <main className="relative z-[2]">{children}</main>
          <footer className="border-t border-[var(--line)] px-0 py-14 pb-11">
            <div className="mx-auto flex max-w-[var(--maxw)] flex-wrap items-center justify-between gap-6 px-5 md:px-10">
              <Link
                href={`/${locale}`}
                className="flex items-center gap-3 font-[var(--serif)] text-[22px] font-semibold tracking-[.4px] text-[var(--ink)]"
              >
                <BrandMark size={36} />
                FateLumen
              </Link>
              <div className="flex flex-wrap gap-[26px]">
                <Link href={`/${locale}/learn`} className="text-[13px] text-[var(--ink-soft)] hover:text-[var(--ink)]">{t("learn")}</Link>
                <Link href={`/${locale}/cases`} className="text-[13px] text-[var(--ink-soft)] hover:text-[var(--ink)]">{t("cases")}</Link>
                <Link href={`/${locale}/faq`} className="text-[13px] text-[var(--ink-soft)] hover:text-[var(--ink)]">{t("faq")}</Link>
                <Link href="#" className="text-[13px] text-[var(--ink-soft)] hover:text-[var(--ink)]">{tf("privacy")}</Link>
                <Link href="#" className="text-[13px] text-[var(--ink-soft)] hover:text-[var(--ink)]">{tf("terms")}</Link>
                <Link href="#" className="text-[13px] text-[var(--ink-soft)] hover:text-[var(--ink)]">{tf("contact")}</Link>
              </div>
            </div>
            <div className="mx-auto mt-9 max-w-[var(--maxw)] border-t border-[var(--line)] px-5 md:px-10 pt-[26px] text-center text-xs tracking-[.4px] text-[var(--ink-faint)]">
              {tf("copy")}
            </div>
          </footer>
          <WhatsAppFab />
        </Providers>
      </ThemeProvider>
    </NextIntlClientProvider>
  );
}
