import { redirect } from "next/navigation";

export default function LegacyPromptStudioPage() {
  redirect("/admin/report-workspace/prompts");
}
