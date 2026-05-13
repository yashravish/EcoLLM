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
