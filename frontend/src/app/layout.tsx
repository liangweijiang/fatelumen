import type { Metadata, Viewport } from "next";
import { SITE } from "@/lib/site";
import "@fontsource-variable/playfair-display/wght.css";
import "@fontsource-variable/playfair-display/wght-italic.css";
import "@fontsource-variable/noto-serif-sc/wght.css";
import "./globals.css";
import "@/styles/themes.css";

export const metadata: Metadata = {
  metadataBase: new URL(SITE.url),
  title: { default: "FateLumen", template: "%s · FateLumen" },
  description:
    "Decode your Chinese birth chart — precise Bazi readings, beautifully explained.",
  alternates: { canonical: "/" },
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
