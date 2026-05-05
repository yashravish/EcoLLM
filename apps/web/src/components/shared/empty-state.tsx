import { InboxIcon } from 'lucide-react';
import { Button } from '@/components/ui/button';

interface EmptyStateProps {
  title: string;
  description: string;
  action?: {
    label: string;
    onClick: () => void;
  };
}

export function EmptyState({ title, description, action }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <div
        className="mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-eco-700 border border-eco-600"
        aria-hidden="true"
      >
        <InboxIcon className="h-7 w-7 text-eco-400" />
      </div>
      <h3 className="text-base font-semibold text-eco-100">{title}</h3>
      <p className="mt-1 max-w-xs text-sm text-eco-400">{description}</p>
      {action && (
        <Button className="mt-4" onClick={action.onClick} size="sm">
          {action.label}
        </Button>
      )}
    </div>
  );
}
