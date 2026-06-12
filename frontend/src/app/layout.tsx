import type { Metadata } from "next";
import { Noto_Serif_SC, Cinzel } from "next/font/google";
import localFont from "next/font/local";
import "./globals.css";
import { cn } from "@/lib/utils";
import Providers from "./providers";

const notoSerif = Noto_Serif_SC({
  subsets: ["latin"],
  variable: "--font-serif",
  weight: ["400", "700", "900"],
  display: "swap",
});

const cinzel = Cinzel({
  subsets: ["latin"],
  variable: "--font-display",
  weight: ["400", "700", "900"],
  display: "swap",
});

const geistSans = localFont({
  src: "./fonts/GeistVF.woff",
  variable: "--font-sans",
  weight: "100 900",
});

export const metadata: Metadata = {
  title: "FateLumen",
  description:
    "洞悉命理玄机，照亮人生前路。基于传统八字命理学的深度命盘推演与解读。",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="zh-CN"
      className={cn(
        "dark font-sans",
        notoSerif.variable,
        cinzel.variable,
        geistSans.variable
      )}
      suppressHydrationWarning
    >
      <body className="min-h-screen bg-background text-foreground antialiased">
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
