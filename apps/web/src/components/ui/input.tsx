import { cn } from '@/lib/utils';
import { forwardRef } from 'react';

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
  hint?: string;
}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ label, error, hint, className, id, ...props }, ref) => {
    const inputId = id || (label ? label.toLowerCase().replace(/\s+/g, '-') : undefined);
    const errorId = error ? `${inputId}-error` : undefined;
    const hintId = hint ? `${inputId}-hint` : undefined;

    return (
      <div className="flex flex-col gap-1.5">
        {label && (
          <label
            htmlFor={inputId}
            className="font-mono text-xs font-medium tracking-wide text-eco-400"
          >
            {label}
          </label>
        )}
        <input
          ref={ref}
          id={inputId}
          aria-invalid={error ? 'true' : undefined}
          aria-describedby={[errorId, hintId].filter(Boolean).join(' ') || undefined}
          className={cn(
            'w-full rounded-md border border-eco-700 bg-eco-800/80 px-3 py-2.5 text-sm text-eco-100',
            'placeholder:text-eco-400/60',
            'focus:border-accent/60 focus:outline-none focus:ring-2 focus:ring-accent/15',
            'disabled:cursor-not-allowed disabled:opacity-40',
            'transition-colors duration-150',
            '[&:-webkit-autofill]:[box-shadow:0_0_0_1000px_#0A1510_inset]',
            '[&:-webkit-autofill]:[-webkit-text-fill-color:#B5E4C7]',
            '[&:-webkit-autofill:hover]:[box-shadow:0_0_0_1000px_#0A1510_inset]',
            '[&:-webkit-autofill:focus]:[box-shadow:0_0_0_1000px_#0A1510_inset]',
            error && 'border-red-500/60 focus:border-red-500 focus:ring-red-500/20',
            className,
          )}
          {...props}
        />
        {hint && !error && (
          <p id={hintId} className="text-xs text-eco-400">
            {hint}
          </p>
        )}
        {error && (
          <p id={errorId} role="alert" className="text-xs text-red-400">
            {error}
          </p>
        )}
      </div>
    );
  },
);

Input.displayName = 'Input';
