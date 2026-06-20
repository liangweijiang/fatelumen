"use client";
import ResourceTable from "../_components/ResourceTable";

export default function AdminUsersPage() {
  return (
    <div>
      <h1 className="mb-6 text-[24px] font-medium" style={{ color: "var(--ink)" }}>用户管理</h1>
      <ResourceTable resource="users" />
    </div>
  );
}
