"use client";

import { useEffect, useState } from "react";

interface UsageSummary {
  total_requests: number;
  total_energy_kwh: number;
  total_co2e_grams: number;
  savings_vs_gpt4_percent: number;
}

export default function DashboardPage() {
  const [usage, setUsage] = useState<UsageSummary | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
    fetch(`${apiUrl}/v1/usage?period=monthly`, {
      headers: { Authorization: `Bearer ${localStorage.getItem("token") ?? ""}` },
    })
      .then((r) => (r.ok ? r.json() : Promise.reject(r.statusText)))
      .then((data) => setUsage(data?.data?.[0] ?? null))
      .catch((e) => setError(String(e)));
  }, []);

  return (
    <main className="min-h-screen bg-gray-50 p-8">
      <div className="mx-auto max-w-5xl">
        <header className="mb-8">
          <h1 className="text-3xl font-bold text-gray-900">EcoLLM Dashboard</h1>
          <p className="mt-1 text-sm text-gray-500">
            Carbon-aware LLM inference — current month
          </p>
        </header>

        {error && (
          <div className="mb-6 rounded-md bg-red-50 p-4 text-sm text-red-700">
            {error}
          </div>
        )}

        <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
          <StatCard
            label="Requests"
            value={usage?.total_requests?.toLocaleString() ?? "—"}
          />
          <StatCard
            label="Energy (kWh)"
            value={usage?.total_energy_kwh?.toFixed(4) ?? "—"}
          />
          <StatCard
            label="CO₂e (g)"
            value={usage?.total_co2e_grams?.toFixed(2) ?? "—"}
          />
          <StatCard
            label="Savings vs GPT-4"
            value={
              usage?.savings_vs_gpt4_percent != null
                ? `${usage.savings_vs_gpt4_percent.toFixed(1)}%`
                : "—"
            }
            highlight
          />
        </div>

        <div className="mt-8 rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <h2 className="mb-4 text-lg font-semibold text-gray-800">
            Getting Started
          </h2>
          <ol className="list-decimal space-y-2 pl-5 text-sm text-gray-600">
            <li>
              Run <code className="rounded bg-gray-100 px-1 py-0.5 font-mono">make dev</code> to
              start the full stack
            </li>
            <li>
              Apply migrations with{" "}
              <code className="rounded bg-gray-100 px-1 py-0.5 font-mono">make migrate</code>
            </li>
            <li>
              Seed model data with{" "}
              <code className="rounded bg-gray-100 px-1 py-0.5 font-mono">make seed</code>
            </li>
            <li>
              Send your first request to{" "}
              <code className="rounded bg-gray-100 px-1 py-0.5 font-mono">
                POST /v1/chat/completions
              </code>
            </li>
          </ol>
        </div>
      </div>
    </main>
  );
}

function StatCard({
  label,
  value,
  highlight = false,
}: {
  label: string;
  value: string;
  highlight?: boolean;
}) {
  return (
    <div
      className={`rounded-xl border p-5 shadow-sm ${
        highlight
          ? "border-green-200 bg-green-50"
          : "border-gray-200 bg-white"
      }`}
    >
      <p className="text-xs font-medium uppercase tracking-wide text-gray-500">
        {label}
      </p>
      <p
        className={`mt-1 text-2xl font-bold ${
          highlight ? "text-green-700" : "text-gray-900"
        }`}
      >
        {value}
      </p>
    </div>
  );
}
