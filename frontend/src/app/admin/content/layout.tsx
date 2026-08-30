import { SecondaryNav } from "../_components/SecondaryNav";

const items = [
  { href: "/admin/content/knowledge", label: "八字知识", description: "知识文章与多语言版本" },
  { href: "/admin/content/faq", label: "常见问题", description: "问题与回答" },
  { href: "/admin/content/case", label: "客户案例", description: "案例内容与隐私控制" },
];

export default function ContentManagementLayout({ children }: { children: React.ReactNode }) {
  return <div className="mx-auto max-w-[1500px]">
    <SecondaryNav title="内容管理" items={items} />
    {children}
  </div>;
}
