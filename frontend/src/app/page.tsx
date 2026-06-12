import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import Link from "next/link";

const features = [
  {
    icon: "☯",
    title: "八字排盘",
    desc: "基于传统干支历法的精准命盘排定，八字、大运、流年一图尽览",
  },
  {
    icon: "✦",
    title: "深度命盘报告",
    desc: "十二维度全方位命理解读，涵盖性格、事业、姻缘、健康、流年运势",
  },
  {
    icon: "◈",
    title: "流年大运推演",
    desc: "大运起运、流年吉凶推演，助您洞悉人生起伏、把握天时",
  },
  {
    icon: "◇",
    title: "大师级解读",
    desc: "合参十神、五行生克、纳音神煞，为您呈现清晰透彻的命理分析",
  },
];

export default function Home() {
  return (
    <div className="min-h-screen bg-stars bg-indigo-deep">
      {/* ── Nav ── */}
      <nav className="fixed top-0 z-50 w-full border-b border-border/40 bg-background/80 backdrop-blur-md">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-6">
          <Link href="/" className="flex items-center gap-2">
            <span className="text-2xl font-bold tracking-wider text-gradient-gold font-display">
              FateLumen
            </span>
          </Link>
          <Button asChild variant="outline" className="border-gold/30 text-gold hover:bg-gold/10">
            <Link href="/login">登 录</Link>
          </Button>
        </div>
      </nav>

      {/* ── Hero ── */}
      <section className="relative flex min-h-screen flex-col items-center justify-center px-6 text-center">
        {/* Ambient glow */}
        <div className="pointer-events-none absolute inset-0 overflow-hidden">
          <div className="absolute -top-40 left-1/2 h-[600px] w-[600px] -translate-x-1/2 rounded-full bg-gold/5 blur-3xl" />
          <div className="absolute top-1/3 left-1/4 h-[400px] w-[400px] rounded-full bg-purple/5 blur-3xl" />
        </div>

        <div className="relative z-10 max-w-3xl animate-fade-in">
          <p className="mb-4 font-display text-sm tracking-[0.3em] text-gold/60">
            洞 悉 命 理   ·   照 亮 前 路
          </p>
          <h1 className="mb-6 font-display text-5xl font-bold leading-tight tracking-wide text-gradient-gold sm:text-6xl lg:text-7xl">
            天命有常
            <br />
            <span className="text-gradient-purple">亦可洞明</span>
          </h1>
          <p className="mx-auto mb-10 max-w-xl text-lg leading-relaxed text-muted-foreground">
            以千年命理智慧为根基，结合传统八字推演，
            <br className="hidden sm:block" />
            为您揭示命盘玄机，指引人生方向
          </p>
          <div className="flex flex-col items-center gap-4 sm:flex-row sm:justify-center">
            <Button
              asChild
              size="lg"
              className="h-12 gap-2 bg-gradient-to-r from-amber-500 to-yellow-500 px-8 text-base font-semibold text-indigo-deep hover:from-amber-400 hover:to-yellow-400 glow-gold"
            >
              <Link href="/login">开始探索命盘</Link>
            </Button>
            <Button
              asChild
              variant="ghost"
              size="lg"
              className="h-12 gap-2 px-8 text-base text-gold/80 hover:text-gold hover:bg-gold/5"
            >
              <Link href="#features">了解更多</Link>
            </Button>
          </div>
        </div>

        {/* Scroll indicator */}
        <div className="absolute bottom-8 left-1/2 -translate-x-1/2 animate-pulse-glow">
          <div className="h-10 w-5 rounded-full border border-gold/30">
            <div className="mx-auto mt-2 h-1.5 w-1.5 rounded-full bg-gold/60" />
          </div>
        </div>
      </section>

      {/* ── Features ── */}
      <section id="features" className="relative px-6 pb-24">
        <div className="mx-auto max-w-6xl">
          <div className="mb-16 text-center animate-fade-in">
            <p className="mb-3 font-display text-sm tracking-[0.3em] text-purple/60">
              命 理 之 道
            </p>
            <h2 className="font-display text-3xl font-bold text-gradient-gold sm:text-4xl">
              为您推演，了悟天命
            </h2>
          </div>
          <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
            {features.map((feat, i) => (
              <Card
                key={feat.title}
                className="group border-border/30 bg-card/50 backdrop-blur-sm transition-all duration-500 hover:border-gold/30 hover:bg-card/80 hover:shadow-lg hover:shadow-gold/5 animate-fade-in"
                style={{ animationDelay: `${i * 0.15}s` }}
              >
                <CardContent className="flex flex-col items-center p-6 text-center pt-6">
                  <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-xl bg-gold/10 text-2xl text-gold transition-transform duration-500 group-hover:scale-110 group-hover:bg-gold/15">
                    {feat.icon}
                  </div>
                  <h3 className="mb-2 font-display text-base font-semibold text-foreground">
                    {feat.title}
                  </h3>
                  <p className="text-sm leading-relaxed text-muted-foreground">
                    {feat.desc}
                  </p>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      </section>

      {/* ── Footer ── */}
      <footer className="border-t border-border/30 px-6 py-8">
        <div className="mx-auto flex max-w-6xl flex-col items-center justify-between gap-4 text-sm text-muted-foreground sm:flex-row">
          <div className="flex items-center gap-2">
            <span className="font-display text-gold/70">FateLumen</span>
            <span className="text-border">|</span>
            <span>传统命理服务平台</span>
          </div>
          <p>&copy; {new Date().getFullYear()} FateLumen. All rights reserved.</p>
        </div>
      </footer>
    </div>
  );
}
