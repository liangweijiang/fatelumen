import { NextIntlClientProvider } from "next-intl";
import { getMessages, getTranslations, setRequestLocale } from "next-intl/server";
import { routing } from "@/i18n/routing";
import { notFound } from "next/navigation";
import ThemeProvider from "@/components/theme/ThemeProvider";
import ThemeSwitcher from "@/components/theme/ThemeSwitcher";
import WhatsAppFab from "@/components/common/WhatsAppFab";
import { LanguageSwitcher } from "@/components/common/LanguageSwitcher";
import Providers from "@/app/providers";
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
        <header
          className="sticky top-0 z-50 border-b border-[var(--line)]"
          style={{
            background: "rgba(237,232,224,.82)",
            backdropFilter: "saturate(160%) blur(14px)",
          }}
        >
          <div className="mx-auto flex h-[70px] max-w-[var(--maxw)] items-center justify-between px-7">
            <Link
              href={`/${locale}`}
              className="flex items-center gap-2.5 font-[var(--serif)] text-[21px] font-semibold tracking-[.3px] text-[var(--ink)]"
            >
              <span
                className="flex h-8 w-8 items-center justify-center rounded-lg bg-[var(--ink)] font-sans text-[17px] font-semibold"
                style={{ color: "var(--bg)" }}
              >
                八
              </span>
              FateLumen
            </Link>
            <nav className="flex items-center gap-9 max-md:hidden">
              <Link href="#how" className="text-[13px] tracking-[.3px] text-[var(--ink-soft)] hover:text-[var(--ink)]">{t("howItWorks")}</Link>
              <Link href="#report" className="text-[13px] tracking-[.3px] text-[var(--ink-soft)] hover:text-[var(--ink)]">{t("sample")}</Link>
              <Link href="#pricing" className="text-[13px] tracking-[.3px] text-[var(--ink-soft)] hover:text-[var(--ink)]">{t("pricing")}</Link>
              <Link href="#faq" className="text-[13px] tracking-[.3px] text-[var(--ink-soft)] hover:text-[var(--ink)]">{t("faq")}</Link>
              <Link href={`/${locale}/learn`} className="text-[13px] tracking-[.3px] text-[var(--ink-soft)] hover:text-[var(--ink)]">{t("learn")}</Link>
              <Link href={`/${locale}/cases`} className="text-[13px] tracking-[.3px] text-[var(--ink-soft)] hover:text-[var(--ink)]">{t("cases")}</Link>
            </nav>
            <div className="flex items-center gap-4">
              <div className="locale-theme-cluster">
                <LanguageSwitcher currentLocale={locale} />
                <ThemeSwitcher />
              </div>
              <Link
                href={`/${locale}/login`}
                className="inline-flex h-9 items-center gap-2 rounded-lg border border-[var(--line)] px-4 text-[13px] font-medium text-[var(--ink-soft)] transition-all hover:border-[var(--ink-faint)] hover:bg-[var(--bg-card)]"
              >
                {t("signIn")}
              </Link>
            </div>
          </div>
        </header>
        <main className="relative z-[2]">{children}</main>
        <footer className="border-t border-[var(--line)] px-0 py-14 pb-11">
          <div className="mx-auto flex max-w-[var(--maxw)] flex-wrap items-center justify-between gap-6 px-7">
            <Link
              href={`/${locale}`}
              className="flex items-center gap-2.5 font-[var(--serif)] text-[21px] font-semibold tracking-[.3px] text-[var(--ink)]"
            >
              <span
                className="flex h-8 w-8 items-center justify-center rounded-lg bg-[var(--ink)] font-sans text-[17px] font-semibold"
                style={{ color: "var(--bg)" }}
              >
                八
              </span>
              FateLumen
            </Link>
            <div className="flex flex-wrap gap-[26px]">
              <a href="#how" className="text-[13px] text-[var(--ink-soft)] hover:text-[var(--ink)]">{t("howItWorks")}</a>
              <a href="#report" className="text-[13px] text-[var(--ink-soft)] hover:text-[var(--ink)]">{t("sample")}</a>
              <a href="#pricing" className="text-[13px] text-[var(--ink-soft)] hover:text-[var(--ink)]">{t("pricing")}</a>
              <a href="#faq" className="text-[13px] text-[var(--ink-soft)] hover:text-[var(--ink)]">{t("faq")}</a>
              <Link href={`/${locale}/learn`} className="text-[13px] text-[var(--ink-soft)] hover:text-[var(--ink)]">{t("learn")}</Link>
              <Link href={`/${locale}/cases`} className="text-[13px] text-[var(--ink-soft)] hover:text-[var(--ink)]">{t("cases")}</Link>
              <Link href={`/${locale}/faq`} className="text-[13px] text-[var(--ink-soft)] hover:text-[var(--ink)]">{t("faq")}</Link>
              <Link href="#" className="text-[13px] text-[var(--ink-soft)] hover:text-[var(--ink)]">{tf("privacy")}</Link>
              <Link href="#" className="text-[13px] text-[var(--ink-soft)] hover:text-[var(--ink)]">{tf("terms")}</Link>
              <Link href="#" className="text-[13px] text-[var(--ink-soft)] hover:text-[var(--ink)]">{tf("contact")}</Link>
            </div>
          </div>
          <div className="mx-auto mt-9 max-w-[var(--maxw)] border-t border-[var(--line)] px-7 pt-[26px] text-center text-xs tracking-[.4px] text-[var(--ink-faint)]">
            {tf("copy")}
          </div>
        </footer>
        <WhatsAppFab />
      </Providers>
      </ThemeProvider>
    </NextIntlClientProvider>
  );
}
