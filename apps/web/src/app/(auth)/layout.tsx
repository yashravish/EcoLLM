import Link from 'next/link';
import { Leaf } from 'lucide-react';
import { LeftPanelBg } from '@/components/auth/left-panel-bg';

export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="relative flex h-screen bg-eco-900">

      {/* ── Left panel: marketing ──────────────────────────────── */}
      <div className="relative hidden lg:flex lg:w-1/2 flex-col bg-eco-700">
        {/* Ambient glow + particles + scan beam */}
        <LeftPanelBg />
        {/* Subtle dot-grid texture, left side only */}
        <div className="absolute inset-0 bg-grid opacity-[0.06]" aria-hidden="true" />

        <div className="relative z-10 flex h-full flex-col px-12 py-10">
          <Link href="/" className="flex items-center gap-2.5 w-fit">
            <Leaf className="h-7 w-7 text-accent" aria-hidden="true" />
            <span className="text-base font-semibold text-eco-50 tracking-wide">EcoLLM</span>
          </Link>

          <div className="flex flex-col justify-center flex-1 pb-16">
            <p className="text-xs font-semibold uppercase tracking-[0.18em] text-accent mb-5">Carbon-aware AI</p>
            <h2 className="text-[3.25rem] font-semibold leading-[1.05] tracking-tight text-eco-50">
              Better AI.<br />Smaller footprint.
            </h2>
            <p className="mt-5 text-sm text-white/60 max-w-[300px] leading-relaxed">
              The right model for each prompt, chosen automatically — no wasted compute.
            </p>
          </div>
        </div>

        {/* Seam fade — dissolves left panel edge into page background */}
        <div
          aria-hidden="true"
          className="absolute inset-y-0 right-0 w-24 pointer-events-none z-20"
          style={{ background: 'linear-gradient(to right, transparent, #060D09)' }}
        />
      </div>

      {/* ── Right panel: form (scrollable) ─────────────────────── */}
      <div className="relative flex-1 overflow-y-auto">
        <LeftPanelBg orbs={false} />
        {/* Subtle centered glow so the form has depth without a card box */}
        <div
          aria-hidden="true"
          className="pointer-events-none absolute inset-0 z-0"
          style={{ background: 'radial-gradient(ellipse 60% 50% at 50% 48%, rgba(34,197,94,0.055) 0%, transparent 70%)' }}
        />
        <div className="relative z-10 flex min-h-full items-center justify-center px-8 py-8">
          <div className="w-full max-w-sm animate-fade-in">
            {children}
          </div>
        </div>
      </div>

    </div>
  );
}
