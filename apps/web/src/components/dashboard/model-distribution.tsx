'use client';

import { PieChart, Pie, Cell, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';

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

interface ModelDistributionProps {
  data: Record<string, number>;
  loading?: boolean;
}

export function ModelDistribution({ data, loading = false }: ModelDistributionProps) {
  const total = Object.values(data).reduce((s, n) => s + n, 0);
  const chartData = Object.entries(data).map(([name, count]) => ({
    name: name.replace(/_/g, '-'),
    value: count,
    pct: total > 0 ? Math.round((count / total) * 100) : 0,
    color: getModelColor(name),
  }));

  const ariaDesc = chartData.map((d) => `${d.name}: ${d.pct}%`).join(', ');

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium">Model Distribution</CardTitle>
      </CardHeader>
      <CardContent>
        {loading ? (
          <LoadingSkeleton variant="chart" />
        ) : (
          <>
            <div
              role="img"
              aria-label={`Model distribution: ${ariaDesc}`}
              className="h-52"
            >
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={chartData}
                    cx="50%"
                    cy="50%"
                    innerRadius={50}
                    outerRadius={80}
                    paddingAngle={2}
                    dataKey="value"
                  >
                    {chartData.map((entry) => (
                      <Cell key={entry.name} fill={entry.color} />
                    ))}
                  </Pie>
                  <Tooltip
                    contentStyle={{
                      backgroundColor: '#fff',
                      border: '1px solid #e5e7eb',
                      borderRadius: '6px',
                      fontSize: '12px',
                    }}
                    formatter={(value: number, name: string) => {
                      const item = chartData.find((d) => d.name === name);
                      return [`${value.toLocaleString()} (${item?.pct ?? 0}%)`, name];
                    }}
                  />
                  <Legend
                    formatter={(value: string) => {
                      const item = chartData.find((d) => d.name === value);
                      return `${value} ${item?.pct ?? 0}%`;
                    }}
                    wrapperStyle={{ fontSize: '11px' }}
                  />
                </PieChart>
              </ResponsiveContainer>
            </div>

            {/* Screen-reader data table */}
            <table className="sr-only">
              <caption>Model distribution data</caption>
              <thead>
                <tr>
                  <th scope="col">Model</th>
                  <th scope="col">Requests</th>
                  <th scope="col">Percentage</th>
                </tr>
              </thead>
              <tbody>
                {chartData.map((row) => (
                  <tr key={row.name}>
                    <td>{row.name}</td>
                    <td>{row.value}</td>
                    <td>{row.pct}%</td>
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
