'use client';

import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Cell,
  ResponsiveContainer,
} from 'recharts';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { formatEnergy } from '@/lib/utils';

interface ModelEnergyEntry {
  model: string;
  energy_kwh: number;
  co2e_grams: number;
  request_count: number;
  percentage_of_traffic: number;
}

interface ModelEnergyBreakdownProps {
  data: ModelEnergyEntry[];
  loading?: boolean;
}

const MODEL_COLORS: Record<string, string> = {
  phi: '#22c55e',
  mistral: '#3b82f6',
  '13b': '#f59e0b',
  '70b': '#ef4444',
};

function getModelColor(name: string): string {
  const lower = name.toLowerCase();
  for (const [key, color] of Object.entries(MODEL_COLORS)) {
    if (lower.includes(key)) return color;
  }
  return '#6b7280';
}

export function ModelEnergyBreakdown({ data, loading = false }: ModelEnergyBreakdownProps) {
  const sorted = [...data].sort((a, b) => a.energy_kwh - b.energy_kwh);

  const chartData = sorted.map((entry) => ({
    ...entry,
    name: entry.model.replace(/_/g, '-'),
    color: getModelColor(entry.model),
  }));

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium">Model Energy Breakdown</CardTitle>
      </CardHeader>
      <CardContent>
        {loading ? (
          <LoadingSkeleton variant="chart" />
        ) : (
          <>
            <div
              role="img"
              aria-label="Energy usage breakdown by model"
              className="h-56"
            >
              <ResponsiveContainer width="100%" height="100%">
                <BarChart
                  data={chartData}
                  layout="vertical"
                  margin={{ top: 4, right: 80, left: 0, bottom: 4 }}
                >
                  <CartesianGrid strokeDasharray="3 3" horizontal={false} stroke="#e5e7eb" />
                  <XAxis
                    type="number"
                    tick={{ fontSize: 11, fill: '#6b7280' }}
                    tickLine={false}
                    axisLine={false}
                    tickFormatter={(v) => formatEnergy(v)}
                  />
                  <YAxis
                    type="category"
                    dataKey="name"
                    tick={{ fontSize: 11, fill: '#6b7280' }}
                    tickLine={false}
                    axisLine={false}
                    width={80}
                  />
                  <Tooltip
                    contentStyle={{
                      backgroundColor: '#fff',
                      border: '1px solid #e5e7eb',
                      borderRadius: '6px',
                      fontSize: '12px',
                    }}
                    formatter={(value: number, _name: string, props) => {
                      const item = props.payload as ModelEnergyEntry & { name: string };
                      return [
                        `${formatEnergy(value)} · ${item.percentage_of_traffic.toFixed(1)}% traffic`,
                        'Energy',
                      ];
                    }}
                  />
                  <Bar dataKey="energy_kwh" radius={[0, 3, 3, 0]}>
                    {chartData.map((entry) => (
                      <Cell key={entry.name} fill={entry.color} />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            </div>

            {/* Screen-reader data table */}
            <table className="sr-only">
              <caption>Energy breakdown by model</caption>
              <thead>
                <tr>
                  <th scope="col">Model</th>
                  <th scope="col">Energy (kWh)</th>
                  <th scope="col">Requests</th>
                  <th scope="col">Traffic %</th>
                </tr>
              </thead>
              <tbody>
                {chartData.map((row) => (
                  <tr key={row.name}>
                    <td>{row.name}</td>
                    <td>{row.energy_kwh}</td>
                    <td>{row.request_count}</td>
                    <td>{row.percentage_of_traffic.toFixed(1)}%</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </>
        )}
      </CardContent>
    </Card>
  );
}
