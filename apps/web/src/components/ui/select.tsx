'use client';

import { cn } from '@/lib/utils';
import { forwardRef, useState, useRef, useEffect, useCallback } from 'react';
import { ChevronDown, Check } from 'lucide-react';

interface SelectProps {
  label?: string;
  error?: string;
  options: Array<{ value: string | number; label: string }>;
  value?: string | number;
  defaultValue?: string | number;
  onChange?: React.ChangeEventHandler<HTMLSelectElement>;
  onBlur?: React.FocusEventHandler<HTMLSelectElement>;
  name?: string;
  disabled?: boolean;
  id?: string;
  className?: string;
}

export const Select = forwardRef<HTMLSelectElement, SelectProps>(
  ({ label, error, options, defaultValue, onChange, onBlur, name, disabled, id, className }, ref) => {
    const selectId = id || (label ? label.toLowerCase().replace(/\s+/g, '-') : undefined);

    const [open, setOpen] = useState(false);
    const [selected, setSelected] = useState<string | number>(
      defaultValue ?? options[0]?.value ?? '',
    );
    const containerRef = useRef<HTMLDivElement>(null);
    const hiddenRef = useRef<HTMLSelectElement>(null);

    // Forward external ref to hidden select
    useEffect(() => {
      if (!ref) return;
      if (typeof ref === 'function') ref(hiddenRef.current);
      else ref.current = hiddenRef.current;
    }, [ref]);

    const selectedLabel = options.find((o) => String(o.value) === String(selected))?.label ?? '';

    const choose = useCallback((val: string | number) => {
      setSelected(val);
      setOpen(false);
      // Trigger react-hook-form onChange via a synthetic event on the hidden select
      if (hiddenRef.current && onChange) {
        const nativeInput = hiddenRef.current;
        const nativeSetter = Object.getOwnPropertyDescriptor(
          window.HTMLSelectElement.prototype, 'value',
        )?.set;
        nativeSetter?.call(nativeInput, val);
        nativeInput.dispatchEvent(new Event('change', { bubbles: true }));
      }
    }, [onChange]);

    // Close on outside click
    useEffect(() => {
      if (!open) return;
      const handler = (e: MouseEvent) => {
        if (!containerRef.current?.contains(e.target as Node)) setOpen(false);
      };
      document.addEventListener('mousedown', handler);
      return () => document.removeEventListener('mousedown', handler);
    }, [open]);

    return (
      <div className={cn('flex flex-col gap-1', className)}>
        {label && (
          <label htmlFor={selectId} className="text-xs font-medium text-eco-400">
            {label}
          </label>
        )}

        {/* Hidden native select keeps react-hook-form happy */}
        <select
          ref={hiddenRef}
          id={selectId}
          name={name}
          value={selected}
          onChange={onChange}
          onBlur={onBlur}
          disabled={disabled}
          aria-hidden="true"
          tabIndex={-1}
          className="sr-only"
        >
          {options.map((opt) => (
            <option key={opt.value} value={opt.value}>{opt.label}</option>
          ))}
        </select>

        {/* Custom dropdown trigger */}
        <div ref={containerRef} className="relative">
          <button
            type="button"
            disabled={disabled}
            onClick={() => setOpen((o) => !o)}
            aria-haspopup="listbox"
            aria-expanded={open}
            aria-labelledby={selectId}
            className={cn(
              'flex w-full items-center justify-between rounded-md border px-3 py-2 text-sm transition-colors',
              'bg-eco-800 text-eco-200',
              open ? 'border-accent ring-1 ring-accent/20' : 'border-eco-600 hover:border-eco-500',
              error && 'border-red-500',
              disabled && 'cursor-not-allowed opacity-50',
            )}
          >
            <span>{selectedLabel}</span>
            <ChevronDown className={cn('h-3.5 w-3.5 text-eco-500 transition-transform', open && 'rotate-180')} />
          </button>

          {open && (
            <ul
              role="listbox"
              className="absolute z-50 mt-1 w-full overflow-hidden rounded-md border border-eco-600 bg-eco-800 shadow-xl shadow-black/40"
            >
              {options.map((opt) => {
                const isActive = String(opt.value) === String(selected);
                return (
                  <li
                    key={opt.value}
                    role="option"
                    aria-selected={isActive}
                    onMouseDown={() => choose(opt.value)}
                    className={cn(
                      'flex cursor-pointer items-center justify-between px-3 py-2 text-sm transition-colors',
                      isActive
                        ? 'bg-eco-700 text-eco-50'
                        : 'text-eco-300 hover:bg-eco-700/60 hover:text-eco-100',
                    )}
                  >
                    {opt.label}
                    {isActive && <Check className="h-3.5 w-3.5 text-accent" />}
                  </li>
                );
              })}
            </ul>
          )}
        </div>

        {error && (
          <p role="alert" className="text-xs text-red-400">{error}</p>
        )}
      </div>
    );
  },
);

Select.displayName = 'Select';
