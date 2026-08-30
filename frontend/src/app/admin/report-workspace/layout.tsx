import { SecondaryNav } from "../_components/SecondaryNav";
const items=[{href:"/admin/report-workspace/calculations",label:"计算档案",description:"输入、计算与版本"},{href:"/admin/report-workspace/prompts",label:"Prompt编排",description:"选择档案生成指令"},{href:"/admin/report-workspace/reports",label:"报告记录",description:"快照、调用与正文"}];
export default function ReportWorkspaceLayout({children}:{children:React.ReactNode}){return <div className="mx-auto max-w-[1600px]"><SecondaryNav title="报告工作台" items={items}/>{children}</div>}
