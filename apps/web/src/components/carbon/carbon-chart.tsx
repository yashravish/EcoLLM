'use client';

import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';

interface DataPoint {
  date: string;
  co2e_grams: number;
  gpt4_equivalent_co2e_grams: number;
}

interface CarbonChartProps {
  data: DataPoint[];
  loading?: boolean;
}

function formatDate(dateStr: string): string {
  if (!dateStr) return '';
  const d = new Date(dateStr.includes('T') ? dateStr : `${dateStr}T00:00:00`);
  if (isNaN(d.getTime())) return dateStr;
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}

function tooltipFormatter(value: number, name: string): [string, string] {
  const label = name === 'your_co2e' ? 'Your CO2e' : 'GPT-4 equivalent';
  return [`${value.toFixed(1)}g CO2e`, label];
}

export function CarbonChart({ data, loading = false }: CarbonChartProps) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium">CO₂e Over Time vs GPT-4</CardTitle>
      </CardHeader>
      <CardContent>
        {loading ? (
          <LoadingSkeleton variant="chart" />
        ) : (
          <>
            <div role="img" aria-label="CO2e emissions comparison over time" className="h-64">
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={data} margin={{ top: 4, right: 16, left: 0, bottom: 0 }}>
                  <defs>
                    <linearGradient id="savingsGradient" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#22c55e" stopOpacity={0.15} />
                      <stop offset="95%" stopColor="#22c55e" stopOpacity={0} />
                    </linearGradient>
                    <linearGradient id="gpt4Gradient" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#9ca3af" stopOpacity={0.1} />
                      <stop offset="95%" stopColor="#9ca3af" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                  <XAxis
                    dataKey="date"
                    tickFormatter={formatDate}
                    tick={{ fontSize: 11, fill: '#6b7280' }}
                    tickLine={false}
                    axisLine={false}
                  />
                  <YAxis
                    tick={{ fontSize: 11, fill: '#6b7280' }}
                    tickLine={false}
                    axisLine={false}
                    tickFormatter={(v) => `${v}g`}
                    width={48}
                  />
                  <Tooltip
                    contentStyle={{
                      backgroundColor: '#fff',
                      border: '1px solid #e5e7eb',
                      borderRadius: '6px',
                      fontSize: '12px',
                    }}
                    formatter={tooltipFormatter}
                    labelFormatter={formatDate}
                  />
                  <Legend
                    formatter={(val) =>
                      val === 'gpt4_equivalent_co2e_grams' ? 'GPT-4 baseline' : 'Your CO₂e'
                    }
                    wrapperStyle={{ fontSize: '12px' }}
                  />
                  {/* GPT-4 baseline — render first so it appears underneath */}
                  <Area
                    type="monotone"
                    dataKey="gpt4_equivalent_co2e_grams"
                    stroke="#9ca3af"
                    strokeWidth={2}
                    fill="url(#gpt4Gradient)"
                    dot={false}
                  />
                  {/* Your CO2e — renders on top */}
                  <Area
                    type="monotone"
                    dataKey="co2e_grams"
                    stroke="#22c55e"
                    strokeWidth={2}
                    fill="url(#savingsGradient)"
                    dot={false}
                  />
                </AreaChart>
              </ResponsiveContainer>
            </div>

            {/* Screen-reader data table */}
            <table className="sr-only">
              <caption>CO₂e emissions over time compared to GPT-4</caption>
              <thead>
                <tr>
                  <th scope="col">Date</th>
                  <th scope="col">Your CO2e (g)</th>
                  <th scope="col">GPT-4 equivalent (g)</th>
                </tr>
              </thead>
              <tbody>
                {data.map((row) => (
                  <tr key={row.date}>
                    <td>{row.date}</td>
                    <td>{row.co2e_grams.toFixed(2)}</td>
                    <td>{row.gpt4_equivalent_co2e_grams.toFixed(2)}</td>
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
