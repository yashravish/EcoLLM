import { cn } from '@/lib/utils';

const MODEL_COLORS: Record<string, string> = {
  phi:     'bg-accent/15 text-accent',
  mistral: 'bg-blue-500/15 text-blue-400',
  '13b':   'bg-amber-500/15 text-amber-400',
  '70b':   'bg-red-500/15 text-red-400',
};

function getModelColor(name: string | undefined | null): string {
  if (!name) return 'bg-eco-700 text-eco-300';
  const lower = name.toLowerCase();
  for (const [key, cls] of Object.entries(MODEL_COLORS)) {
    if (lower.includes(key)) return cls;
  }
  return 'bg-eco-700 text-eco-300';
}

export function ModelBadge({ name }: { name: string | undefined | null }) {
  const colorClass = getModelColor(name);
  const displayName = name ? name.replace(/_/g, '-') : 'Unknown';

  return (
    <span
      className={cn(
        'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
        colorClass,
      )}
      aria-label={`Model: ${displayName}`}
    >
      {displayName}
    </span>
  );
}
