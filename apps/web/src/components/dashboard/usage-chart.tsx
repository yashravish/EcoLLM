'use client';

import { useState } from 'react';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import type { UsageDailyBreakdown } from '@/types';

interface UsageChartProps {
  data: UsageDailyBreakdown[];
  period: '7d' | '30d';
  onPeriodChange: (period: '7d' | '30d') => void;
  loading?: boolean;
}

function formatDate(dateStr: string): string {
  if (!dateStr) return '';
  const d = new Date(dateStr.includes('T') ? dateStr : `${dateStr}T00:00:00`);
  if (isNaN(d.getTime())) return dateStr;
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}

export function UsageChart({ data, period, onPeriodChange, loading = false }: UsageChartProps) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-sm font-medium">Usage Over Time</CardTitle>
        <Tabs value={period} onValueChange={(v) => onPeriodChange(v as '7d' | '30d')}>
          <TabsList>
            <TabsTrigger value="7d">7d</TabsTrigger>
            <TabsTrigger value="30d">30d</TabsTrigger>
          </TabsList>
        </Tabs>
      </CardHeader>
      <CardContent>
        {loading ? (
          <LoadingSkeleton variant="chart" />
        ) : data.length === 0 ? (
          <div className="flex h-64 items-center justify-center font-mono text-xs text-eco-500">
            No data for this period
          </div>
        ) : (
          <>
            <div
              aria-label="Request and energy usage over time"
              role="img"
              className="h-64"
            >
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={data} margin={{ top: 4, right: 16, left: 0, bottom: 0 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                  <XAxis
                    dataKey="date"
                    tickFormatter={formatDate}
                    tick={{ fontSize: 11, fill: '#6b7280' }}
                    tickLine={false}
                    axisLine={false}
                  />
                  <YAxis
                    yAxisId="requests"
                    orientation="left"
                    tick={{ fontSize: 11, fill: '#6b7280' }}
                    tickLine={false}
                    axisLine={false}
                    width={40}
                  />
                  <YAxis
                    yAxisId="energy"
                    orientation="right"
                    tick={{ fontSize: 11, fill: '#6b7280' }}
                    tickLine={false}
                    axisLine={false}
                    width={60}
                    tickFormatter={(v) => `${(v * 1000).toFixed(2)}Wh`}
                  />
                  <Tooltip
                    contentStyle={{
                      backgroundColor: '#fff',
                      border: '1px solid #e5e7eb',
                      borderRadius: '6px',
                      fontSize: '12px',
                    }}
                    formatter={(value: number, name: string) => {
                      if (name === 'Requests') return [value.toLocaleString(), name];
                      return [`${(value * 1000).toFixed(4)} Wh`, name];
                    }}
                    labelFormatter={formatDate}
                  />
                  <Legend wrapperStyle={{ fontSize: '12px' }} />
                  <Line
                    yAxisId="requests"
                    type="monotone"
                    dataKey="requests"
                    name="Requests"
                    stroke="#22c55e"
                    strokeWidth={2}
                    dot={false}
                    activeDot={{ r: 4 }}
                  />
                  <Line
                    yAxisId="energy"
                    type="monotone"
                    dataKey="energy_kwh"
                    name="Energy"
                    stroke="#3b82f6"
                    strokeWidth={2}
                    dot={false}
                    activeDot={{ r: 4 }}
                  />
                </LineChart>
              </ResponsiveContainer>
            </div>

            {/* Screen-reader data table */}
            <table className="sr-only">
              <caption>Request and energy usage data</caption>
              <thead>
                <tr>
                  <th scope="col">Date</th>
                  <th scope="col">Requests</th>
                  <th scope="col">Energy (kWh)</th>
                </tr>
              </thead>
              <tbody>
                {data.map((row) => (
                  <tr key={row.date}>
                    <td>{row.date}</td>
                    <td>{row.requests}</td>
                    <td>{row.energy_kwh}</td>
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
