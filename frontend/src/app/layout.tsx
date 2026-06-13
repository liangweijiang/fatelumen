import type { Metadata, Viewport } from "next";
import { Playfair_Display } from "next/font/google";
import "./globals.css";
import "@/styles/themes.css";

const playfair = Playfair_Display({
  subsets: ["latin"],
  variable: "--font-playfair",
  weight: ["500", "600", "700"],
  style: ["normal", "italic"],
  display: "swap",
});

export const metadata: Metadata = {
  title: "FateLumen",
  description:
    "Decode your Chinese birth chart — precise Bazi readings, beautifully explained.",
};

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  viewportFit: "cover",
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
      className={playfair.variable}
    >
      <head>
        <script
          dangerouslySetInnerHTML={{
            __html: `(function(){var t=localStorage.getItem('fatelumen-theme');if(t){try{var p=JSON.parse(t);if(p.state&&p.state.theme)document.documentElement.setAttribute('data-theme',p.state.theme);}catch(e){}}})();`,
          }}
        />
      </head>
      <body>
        {children}
      </body>
    </html>
  );
}
