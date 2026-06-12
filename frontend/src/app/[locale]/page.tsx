import { Hero } from "@/components/landing/Hero";
import { TrustBar } from "@/components/landing/TrustBar";
import { HowItWorks } from "@/components/landing/HowItWorks";
import { SampleReport } from "@/components/landing/SampleReport";
import { Pricing } from "@/components/landing/Pricing";
import { FaqPreview } from "@/components/landing/FaqPreview";
import { CtaBand } from "@/components/landing/CtaBand";

export default async function HomePage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  return (
    <>
      <Hero />
      <TrustBar />
      <HowItWorks />
      <SampleReport />
      <Pricing />
      <FaqPreview locale={locale} />
      <CtaBand />
    </>
  );
}
