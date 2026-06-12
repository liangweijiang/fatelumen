import type { Metadata } from "next";
import { Playfair_Display, Inter } from "next/font/google";
import { NextIntlClientProvider } from "next-intl";
import { getMessages } from "next-intl/server";
import { notFound } from "next/navigation";
import { routing } from "@/i18n/routing";
import ThemeProvider from "@/components/theme/ThemeProvider";
import Providers from "./providers";
import "./globals.css";
import "@/styles/themes.css";

const playfair = Playfair_Display({
  subsets: ["latin"],
  variable: "--font-serif",
  weight: ["400", "500", "600", "700"],
  style: ["normal", "italic"],
  display: "swap",
});

const inter = Inter({
  subsets: ["latin"],
  variable: "--font-sans",
  weight: ["300", "400", "500", "600"],
  display: "swap",
});

export const metadata: Metadata = {
  title: "FateLumen",
  description: "Decode your Chinese birth chart — precise Bazi readings, beautifully explained.",
};

export default async function RootLayout({
  children,
  params,
}: Readonly<{
  children: React.ReactNode;
  params: Promise<{ locale: string }>;
}>) {
  const { locale } = await params;
  if (!routing.locales.includes(locale as "en" | "zh" | "ja" | "ko")) {
    notFound();
  }
  const messages = await getMessages();

  return (
    <html lang={locale} data-theme="kraft" className={`${playfair.variable} ${inter.variable}`}>
      <head>
        {/* 防闪烁 script：先读 localStorage 还原主题 */}
        <script
          dangerouslySetInnerHTML={{
            __html: `(function(){var t=localStorage.getItem('fatelumen-theme');if(t){try{var p=JSON.parse(t);if(p.state&&p.state.theme)document.documentElement.setAttribute('data-theme',p.state.theme);}catch(e){}}})();`,
          }}
        />
      </head>
      <body className="font-sans" style={{ fontFamily: "var(--sans)", background: "var(--bg)", color: "var(--ink)" }}>
        <NextIntlClientProvider messages={messages}>
          <ThemeProvider>
            <Providers>{children}</Providers>
          </ThemeProvider>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
