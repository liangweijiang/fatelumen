"use client";

import { useState } from "react";
import { toast } from "sonner";
import { createOrder } from "@/lib/api/endpoints";

const PROVIDERS = [
  { id: "alipay", label: "支付宝", logo: "/payment/alipay.svg" },
  { id: "paypal", label: "PayPal", logo: "/payment/paypal.svg" },
];

export default function CheckoutBlock({ reportId }: { reportId: number }) {
  const [provider, setProvider] = useState("alipay");
  const [loading, setLoading] = useState(false);

  async function handleCheckout() {
    setLoading(true);
    try {
      const result = await createOrder({ report_id: reportId, provider });
      if (result.checkout_url) {
        window.location.href = result.checkout_url;
        return;
      }
      toast.error("发起支付失败，请稍后重试");
    } catch {
      toast.error("发起支付失败，请稍后重试");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div
      className="mx-auto mb-6 max-w-[400px] rounded-2xl border p-5"
      style={{ background: "var(--bg-card)", borderColor: "var(--line)" }}
    >
      <div
        className="mb-3 text-center text-[15px] font-semibold"
        style={{ fontFamily: "var(--serif-d)", color: "var(--ink)" }}
      >
        解锁完整命盘报告
      </div>
      <div className="mb-4 flex gap-3">
        {PROVIDERS.map((p) => (
          <button
            key={p.id}
            type="button"
            onClick={() => setProvider(p.id)}
            className="flex flex-1 items-center justify-center gap-2 rounded-full border py-2 text-[14px] transition-all"
            style={{
              borderColor: provider === p.id ? "var(--gold-deep)" : "var(--line)",
              color: provider === p.id ? "var(--gold-deep)" : "var(--ink-soft)",
              background: provider === p.id ? "var(--gold-soft)" : "transparent",
            }}
          >
            <img
              src={p.logo}
              alt={p.label}
              width={36}
              height={24}
              className="rounded-[3px]"
              style={{ display: "block" }}
            />
            {p.label}
          </button>
        ))}
      </div>
      <button
        type="button"
        onClick={handleCheckout}
        disabled={loading}
        className="w-full rounded-full py-3 text-[15px] font-semibold tracking-[.3px] transition-all"
        style={{
          fontFamily: "var(--serif-d)",
          background: loading ? "var(--ink-faint)" : "var(--gold-deep)",
          color: "var(--bg-card)",
          cursor: loading ? "not-allowed" : "pointer",
        }}
      >
        {loading ? "正在跳转支付…" : "立即解锁"}
      </button>
    </div>
  );
}
