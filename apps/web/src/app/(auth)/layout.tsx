import Link from 'next/link';
import { Leaf } from 'lucide-react';
import { LeftPanelBg } from '@/components/auth/left-panel-bg';

const BULLETS = [
  'A smaller carbon footprint per request',
  'The right model for each prompt, chosen automatically',
  'Pay for what you use, nothing more',
];

export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="relative flex h-screen bg-eco-900">

      {/* ── Left panel: marketing ──────────────────────────────── */}
      {/* No bg / border / overflow-hidden — inherits unified background from parent */}
      <div className="relative hidden lg:flex lg:w-1/2 flex-col bg-eco-700">
        {/* Ambient glow + particles + scan beam */}
        <LeftPanelBg />
        {/* Subtle dot-grid texture, left side only */}
        <div className="absolute inset-0 bg-grid opacity-[0.06]" aria-hidden="true" />

        <div className="relative z-10 flex h-full flex-col px-10 py-10">
          <Link href="/" className="flex items-center gap-2.5 w-fit">
            <Leaf className="h-7 w-7 text-accent" aria-hidden="true" />
            <span className="text-base font-semibold text-eco-50 tracking-wide">EcoLLM</span>
          </Link>

          <div className="my-auto -translate-y-8">
            <h2 className="text-[3rem] font-semibold leading-[1.05] tracking-tight text-eco-50 mb-4">
              Better AI.<br />Smaller footprint.
            </h2>
            <p className="text-base text-white/[0.82] leading-relaxed mb-10 max-w-[340px]">
              Each prompt finds the right model — no compromise on quality or speed, just a smaller footprint.
            </p>

            <ul className="space-y-5">
              {BULLETS.map((text) => (
                <li key={text} className="flex items-baseline gap-3">
                  <span className="text-white/[0.72] text-base select-none" aria-hidden="true">—</span>
                  <span className="text-sm text-white/[0.72]">{text}</span>
                </li>
              ))}
            </ul>
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
        <div className="relative z-10 flex min-h-full items-center justify-center px-5 py-5">
          <div className="w-full max-w-sm animate-fade-in">
            {children}
          </div>
        </div>
      </div>

    </div>
  );
}
