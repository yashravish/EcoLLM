import { cn } from '@/lib/utils';

type BadgeVariant = 'default' | 'success' | 'warning' | 'danger' | 'info' | 'outline';

const variantClasses: Record<BadgeVariant, string> = {
  default: 'bg-eco-700 text-eco-200',
  success: 'bg-accent/15 text-accent',
  warning: 'bg-amber-500/15 text-amber-400',
  danger:  'bg-red-500/15 text-red-400',
  info:    'bg-blue-500/15 text-blue-400',
  outline: 'border border-eco-600 text-eco-300',
};

interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  variant?: BadgeVariant;
}

export function Badge({ variant = 'default', className, children, ...props }: BadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
        variantClasses[variant],
        className,
      )}
      {...props}
    >
      {children}
    </span>
  );
}
