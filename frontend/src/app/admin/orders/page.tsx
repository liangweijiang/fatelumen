"use client";
import ResourceTable from "../_components/ResourceTable";

export default function AdminOrdersPage() {
  return (
    <div>
      <h1 className="mb-6 text-[24px] font-medium" style={{ color: "var(--ink)" }}>订单管理</h1>
      <ResourceTable resource="orders" />
    </div>
  );
}
