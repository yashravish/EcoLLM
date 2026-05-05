import { cn } from '@/lib/utils';

const MODEL_COLORS: Record<string, string> = {
  phi: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
  mistral: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400',
  '13b': 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-400',
  '70b': 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400',
};

function getModelColor(name: string | undefined | null): string {
  if (!name) return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300';
  const lower = name.toLowerCase();
  for (const [key, cls] of Object.entries(MODEL_COLORS)) {
    if (lower.includes(key)) return cls;
  }
  return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300';
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
