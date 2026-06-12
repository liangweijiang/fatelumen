import type { Metadata } from "next";
import { Playfair_Display, Inter } from "next/font/google";
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
  description:
    "Decode your Chinese birth chart — precise Bazi readings, beautifully explained.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="en"
      data-theme="kraft"
      className={`${playfair.variable} ${inter.variable}`}
    >
      <head>
        <script
          dangerouslySetInnerHTML={{
            __html: `(function(){var t=localStorage.getItem('fatelumen-theme');if(t){try{var p=JSON.parse(t);if(p.state&&p.state.theme)document.documentElement.setAttribute('data-theme',p.state.theme);}catch(e){}}})();`,
          }}
        />
      </head>
      <body
        className="font-sans"
        style={{
          fontFamily: "var(--sans)",
          background: "var(--bg)",
          color: "var(--ink)",
        }}
      >
        {children}
      </body>
    </html>
  );
}
