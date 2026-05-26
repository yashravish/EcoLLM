'use client';

import { Leaf } from 'lucide-react';

export function MobileLogo({ className }: { className?: string }) {
  return (
    <div className={`flex items-center gap-2 lg:hidden ${className ?? ''}`}>
      <Leaf className="h-5 w-5 text-accent" aria-hidden="true" />
      <span className="text-sm font-semibold text-eco-50 tracking-wide">EcoLLM</span>
    </div>
  );
}

export function ServerErrorBanner({ message }: { message: string }) {
  return (
    <div
      role="alert"
      className="rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-400"
    >
      {message}
    </div>
  );
}
