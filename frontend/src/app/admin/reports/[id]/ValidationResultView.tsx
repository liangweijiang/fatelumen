import { AlertTriangle, CheckCircle2, XCircle } from "lucide-react";
import type { ReportValidationResult, ReportValidationRule } from "@/lib/admin-api";

const stageLabels: Record<string, string> = { format: "JSON 格式", schema: "Schema 结构", completeness: "模块完整度", fact_consistency: "确定性事实", protected_facts: "确定性事实", language_terminology: "语言与专业术语", product_boundary: "产品边界", report: "十章汇总一致性" };
const stageLogic: Record<string, string> = {
  format: "检查输出是否为单一、合法且未被 Markdown 包裹的 JSON 对象，并限制编码、大小和嵌套深度。",
  schema: "按本章冻结的输出协议检查章节名称、模块数量、模块名称、序号、顺序和字段类型。",
  completeness: "检查约定模块是否全部存在，并且每个模块都包含可用正文。",
  fact_consistency: "将正文中可程序识别的干支、年份、日主、身强弱和用神结论与冻结事实逐项核对。",
  protected_facts: "将正文中可程序识别的确定性结论与报告生成时的冻结事实逐项核对。",
  language_terminology: "检查正文主体语言是否符合报告目标语言；出现受控专业词时必须使用本次冻结的指定译名。",
  product_boundary: "检查正文是否重新排盘、修改冻结事实、泄露内部指令或出现现有规则禁止的输出。",
  report: "检查十章是否齐全、顺序唯一、采用记录有效，以及跨章事实和未来十年流年是否一致。",
};

function Rule({ rule }: { rule: ReportValidationRule }) {
  const stage = rule.stage ?? "other";
  return <article className="grid gap-3 border-t py-4 first:border-t-0 md:grid-cols-[13rem_1fr]" style={{ borderColor: "var(--line-soft)" }}>
    <div><span className={`inline-flex items-center gap-1.5 text-sm ${rule.passed ? "text-emerald-800" : "text-red-800"}`}>{rule.passed ? <CheckCircle2 size={15} /> : <XCircle size={15} />}{rule.passed ? "通过" : "未通过"}</span><h4 className="mt-1 font-medium">{rule.name}</h4><code className="mt-1 block text-xs" style={{ color: "var(--ink-faint)" }}>{rule.code}</code></div>
    <div className="space-y-2 text-sm leading-6"><p><span style={{ color: "var(--ink-faint)" }}>校验逻辑：</span>{stageLogic[stage] ?? "按照生成时冻结的校验器版本执行该项规则。"}</p><p><span style={{ color: "var(--ink-faint)" }}>级别：</span>{rule.severity === "warning" ? "警告" : "错误"}　<span style={{ color: "var(--ink-faint)" }}>失败后：</span>{rule.retryable ? "拒绝本次输出并允许自动重试" : rule.passed ? "继续后续校验" : "拒绝且不重试"}</p>{rule.message && <p>{rule.message}</p>}{(rule.expected || rule.actual) && <dl className="grid gap-2 sm:grid-cols-2"><div><dt style={{ color: "var(--ink-faint)" }}>期望</dt><dd>{rule.expected || "—"}</dd></div><div><dt style={{ color: "var(--ink-faint)" }}>实际</dt><dd>{rule.actual || "—"}</dd></div></dl>}{!!rule.evidence_refs?.length && <details><summary className="cursor-pointer text-amber-900">查看数据来源</summary><ul className="mt-2 space-y-1 font-mono text-xs">{rule.evidence_refs.map(ref => <li key={ref}>{ref}</li>)}</ul></details>}</div>
  </article>;
}

export function ValidationResultView({ result, emptyText = "本次调用没有校验结果" }: { result?: ReportValidationResult; emptyText?: string }) {
  if (!result) return <p className="border px-4 py-5 text-sm" style={{ borderColor: "var(--line-soft)", color: "var(--ink-faint)" }}>{emptyText}</p>;
  const rules = result.rules ?? [];
  const failedRules = rules.filter(rule => !rule.passed);
  const passedCount = rules.length - failedRules.length;
  if (result.passed && !failedRules.length) return <section className="border px-4 py-4 text-sm" style={{ borderColor: "var(--line-soft)" }}><p className="flex items-center gap-2 text-emerald-800"><CheckCircle2 size={16} />校验通过并采用</p><p className="mt-2 leading-6">JSON、Schema、模块完整性、冻结事实、目标语言与专业术语、产品边界等 {passedCount} 项规则全部通过。</p><details className="mt-3"><summary className="cursor-pointer text-amber-900">查看通过的规则</summary><div className="mt-2">{rules.map((rule, index) => <Rule key={`${rule.code}-${index}`} rule={rule} />)}</div></details></section>;
  const failedStages = Array.from(new Set(failedRules.map(rule => rule.stage ?? "other")));
  return <div className="space-y-4"><section className="border p-4 text-sm" style={{ borderColor: "var(--line-soft)" }}><p className="flex items-center gap-2 text-red-800"><AlertTriangle size={16} />校验拒绝，本次输出未采用</p><p className="mt-2">失败阶段：{failedStages.map(stage => stageLabels[stage] ?? stage).join("、")}；{failedRules.length} 项失败，{passedCount} 项通过。</p>{result.summary && <p className="mt-2">{result.summary}</p>}</section>{failedStages.map(stage => <section key={stage} className="border px-4" style={{ borderColor: "var(--line-soft)" }}><h3 className="py-3 font-medium">{stageLabels[stage] ?? stage}</h3>{failedRules.filter(rule => (rule.stage ?? "other") === stage).map((rule, index) => <Rule key={`${rule.code}-${index}`} rule={rule} />)}</section>)}{passedCount > 0 && <details className="border px-4" style={{ borderColor: "var(--line-soft)" }}><summary className="cursor-pointer py-3 text-sm text-amber-900">另有 {passedCount} 项规则通过</summary>{rules.filter(rule => rule.passed).map((rule, index) => <Rule key={`${rule.code}-${index}`} rule={rule} />)}</details>}</div>;
}
