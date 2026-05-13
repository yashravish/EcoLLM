import { cn } from '@/lib/utils';

type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'destructive' | 'outline';
type ButtonSize = 'sm' | 'md' | 'lg' | 'icon';

const variantClasses: Record<ButtonVariant, string> = {
  primary:
    'bg-eco-800 text-accent font-semibold border border-eco-600 hover:bg-eco-700 hover:border-eco-500 focus-visible:ring-accent/40 disabled:opacity-40',
  secondary:
    'bg-eco-700 text-eco-100 border border-eco-500 hover:bg-eco-600 focus-visible:ring-eco-400',
  ghost:
    'text-eco-300 hover:bg-eco-700 hover:text-eco-100 focus-visible:ring-eco-400',
  destructive:
    'bg-red-600/20 text-red-400 border border-red-600/40 hover:bg-red-600/30 focus-visible:ring-red-500',
  outline:
    'border border-eco-500 text-eco-200 hover:bg-eco-700 hover:text-eco-50 focus-visible:ring-eco-400',
};

const sizeClasses: Record<ButtonSize, string> = {
  sm:   'px-3 py-1.5 text-xs',
  md:   'px-4 py-2 text-sm',
  lg:   'px-6 py-2.5 text-sm',
  icon: 'p-2',
};

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  loading?: boolean;
}

export function Button({
  variant = 'primary',
  size = 'md',
  loading = false,
  className,
  children,
  disabled,
  ...props
}: ButtonProps) {
  return (
    <button
      className={cn(
        'inline-flex items-center justify-center gap-2 rounded-md font-medium transition-all duration-150',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-offset-eco-900',
        'disabled:pointer-events-none disabled:opacity-40',
        variantClasses[variant],
        sizeClasses[size],
        className,
      )}
      disabled={disabled || loading}
      aria-busy={loading}
      {...props}
    >
      {loading && (
        <svg
          className="h-3.5 w-3.5 animate-spin"
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
          aria-hidden="true"
        >
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
          <path
            className="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
          />
        </svg>
      )}
      {children}
    </button>
  );
}
