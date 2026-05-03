import { Skeleton } from '@/components/ui/skeleton';

interface LoadingSkeletonProps {
  variant: 'card' | 'table' | 'chart' | 'text';
  lines?: number;
}

export function LoadingSkeleton({ variant, lines = 3 }: LoadingSkeletonProps) {
  if (variant === 'card') {
    return (
      <div aria-busy="true" aria-label="Loading" className="space-y-3 p-6">
        <Skeleton className="h-4 w-1/3" />
        <Skeleton className="h-8 w-1/2" />
        <Skeleton className="h-3 w-1/4" />
      </div>
    );
  }

  if (variant === 'table') {
    return (
      <div aria-busy="true" aria-label="Loading" className="space-y-2 p-4">
        <Skeleton className="h-8 w-full" />
        {Array.from({ length: lines }).map((_, i) => (
          <Skeleton key={i} className="h-12 w-full" />
        ))}
      </div>
    );
  }

  if (variant === 'chart') {
    return (
      <div aria-busy="true" aria-label="Loading" className="p-6">
        <Skeleton className="h-4 w-1/4 mb-4" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  return (
    <div aria-busy="true" aria-label="Loading" className="space-y-2">
      {Array.from({ length: lines }).map((_, i) => (
        <Skeleton key={i} className={`h-4 ${i === lines - 1 ? 'w-2/3' : 'w-full'}`} />
      ))}
    </div>
  );
}
