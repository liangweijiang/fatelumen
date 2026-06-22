"use client";

import { useState } from "react";
import { toast } from "sonner";
import { createOrder, unlockReportWithCredits } from "@/lib/api/endpoints";

const UNLOCK_COST = 30;

const PACKS = [
  { sku: "pack_50", credits: 50, label: "50 命理点", hint: "适合偶尔推演" },
  { sku: "pack_120", credits: 120, label: "120 命理点", hint: "更超值，常看更划算" },
];

const PROVIDERS = [
  { id: "alipay", label: "支付宝", logo: "/payment/alipay.svg" },
  { id: "paypal", label: "PayPal", logo: "/payment/paypal.svg" },
];

export default function CheckoutBlock({
  reportId,
  credits,
  onUnlocked,
}: {
  reportId: number;
  credits: number;
  onUnlocked?: () => void;
}) {
  const [unlocking, setUnlocking] = useState(false);
  const [buyingSku, setBuyingSku] = useState<string | null>(null);
  const [provider, setProvider] = useState("alipay");

  const enough = credits >= UNLOCK_COST;

  async function handleUnlock() {
    setUnlocking(true);
    try {
      await unlockReportWithCredits(reportId);
      toast.success("解锁成功，正在为您展开完整命盘");
      onUnlocked?.();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "解锁失败，请稍后重试");
    } finally {
      setUnlocking(false);
    }
  }

  async function handleBuy(sku: string) {
    setBuyingSku(sku);
    try {
      const result = await createOrder({ report_id: 0, sku, provider });
      if (result.checkout_url) {
        window.location.href = result.checkout_url;
        return;
      }
      toast.error("发起支付失败，请稍后重试");
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "发起支付失败，请稍后重试");
    } finally {
      setBuyingSku(null);
    }
  }

  return (
    <div
      className="mx-auto mb-6 max-w-[420px] rounded-2xl border p-5"
      style={{ background: "var(--bg-card)", borderColor: "var(--line)" }}
    >
      <div
        className="mb-1 text-center text-[15px] font-semibold"
        style={{ fontFamily: "var(--serif-d)", color: "var(--ink)" }}
      >
        解锁完整命盘报告
      </div>
      <div className="mb-4 text-center text-[13px]" style={{ color: "var(--ink-faint)" }}>
        本次解锁需 {UNLOCK_COST} 命理点 · 当前余额 {credits} 点
      </div>

      {enough ? (
        <button
          type="button"
          onClick={handleUnlock}
          disabled={unlocking}
          className="w-full rounded-full py-3 text-[15px] font-semibold tracking-[.3px] transition-all"
          style={{
            fontFamily: "var(--serif-d)",
            background: unlocking ? "var(--ink-faint)" : "var(--gold-deep)",
            color: "var(--bg-card)",
            cursor: unlocking ? "not-allowed" : "pointer",
          }}
        >
          {unlocking ? "正在解锁…" : `用 ${UNLOCK_COST} 命理点解锁`}
        </button>
      ) : (
        <>
          <div className="mb-3 text-center text-[13px]" style={{ color: "var(--ink-soft)" }}>
            命理点不足，先购买命理点
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
          <div className="flex flex-col gap-3">
            {PACKS.map((pk) => (
              <button
                key={pk.sku}
                type="button"
                onClick={() => handleBuy(pk.sku)}
                disabled={buyingSku !== null}
                className="flex items-center justify-between rounded-xl border px-4 py-3 text-left transition-all"
                style={{
                  borderColor: "var(--line)",
                  background: "var(--bg-soft)",
                  cursor: buyingSku !== null ? "not-allowed" : "pointer",
                }}
              >
                <span>
                  <span className="block text-[15px] font-semibold" style={{ color: "var(--ink)" }}>
                    {pk.label}
                  </span>
                  <span className="block text-[12px]" style={{ color: "var(--ink-faint)" }}>
                    {pk.hint}
                  </span>
                </span>
                <span
                  className="rounded-full px-4 py-1.5 text-[13px] font-semibold"
                  style={{ background: "var(--gold-deep)", color: "var(--bg-card)" }}
                >
                  {buyingSku === pk.sku ? "跳转中…" : "购买"}
                </span>
              </button>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
