import { SecondaryNav } from "../_components/SecondaryNav";

const items = [
  { href: "/admin/settings/providers", label: "供应商配置", description: "接口地址与密钥" },
  { href: "/admin/settings/models", label: "模型配置", description: "模型、优先级与重试" },
  { href: "/admin/settings/report", label: "报告设置", description: "十章并发数量" },
  { href: "/admin/settings/general", label: "通用设置", description: "登录与保存期限" },
];

export default function SettingsLayout({ children }: { children: React.ReactNode }) {
  return <div className="mx-auto max-w-[1500px]"><SecondaryNav title="系统设置" items={items}/>{children}</div>;
}
