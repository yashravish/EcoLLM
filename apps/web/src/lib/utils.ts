import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}

export function formatCO2(grams: number): string {
  if (grams >= 1000) return `${(grams / 1000).toFixed(1)}kg CO2e`;
  return `${grams.toFixed(1)}g CO2e`;
}

export function formatCost(usd: number): string {
  if (usd < 0.01) return `$${usd.toFixed(3)}`;
  return `$${usd.toFixed(2)}`;
}

export function formatLatency(ms: number): string {
  if (ms < 1000) return `${Math.round(ms)}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

export function formatEnergy(kwh: number): string {
  if (kwh < 0.001) {
    const wh = kwh * 1000;
    return `${wh.toFixed(4)} Wh`;
  }
  return `${kwh.toFixed(5)} kWh`;
}

export function co2ToTrees(grams: number): number {
  // 1 tree absorbs ~22kg CO2/year ≈ 60.3g/day
  return Math.round(grams / 60.3);
}

export function formatRelativeTime(dateStr: string): string {
  const diffMs = Date.now() - new Date(dateStr).getTime();
  const diffSec = Math.floor(diffMs / 1000);
  if (diffSec < 60) return `${diffSec}s ago`;
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin} min ago`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr}h ago`;
  return `${Math.floor(diffHr / 24)}d ago`;
}

export function truncate(text: string, maxLen: number): string {
  if (text.length <= maxLen) return text;
  return text.slice(0, maxLen) + '…';
}

export function dateRange(days: number): { from: string; to: string } {
  const to = new Date().toISOString().split('T')[0];
  const from = new Date(Date.now() - days * 86_400_000).toISOString().split('T')[0];
  return { from, to };
}
