'use client';

import { Suspense, useEffect, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import { Leaf, AlertTriangle } from 'lucide-react';
import Link from 'next/link';
import { api } from '@/lib/api';

function CallbackHandler() {
  const searchParams = useSearchParams();
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    const token = searchParams.get('token');
    const next  = searchParams.get('next');
    const error = searchParams.get('error');

    if (error) {
      window.location.replace(`/register?error=${encodeURIComponent(error)}`);
      return;
    }
    if (token) {
      api.setToken(token);
      window.location.replace(next || '/overview');
      return;
    }
    setFailed(true);
  }, [searchParams]);

  if (failed) {
    return (
      <div className="rounded-2xl border border-white/[0.06] bg-eco-800/60 p-8 shadow-[0_0_0_1px_rgba(0,0,0,0.3),0_20px_48px_rgba(0,0,0,0.45)] backdrop-blur-sm flex flex-col items-center gap-4 text-center">
        <div className="flex h-10 w-10 items-center justify-center rounded bg-red-500/10 ring-1 ring-red-500/30">
          <AlertTriangle className="h-5 w-5 text-red-400" aria-hidden="true" />
        </div>
        <div>
          <p className="text-sm font-medium text-eco-100">Sign-in failed</p>
          <p className="mt-1 text-xs text-eco-400">The authentication link is missing or has expired.</p>
        </div>
        <Link href="/login" className="text-xs text-accent hover:text-accent-bright underline underline-offset-2 transition-colors">
          Back to sign in
        </Link>
      </div>
    );
  }

  return (
    <div className="rounded-2xl border border-white/[0.06] bg-eco-800/60 p-8 shadow-[0_0_0_1px_rgba(0,0,0,0.3),0_20px_48px_rgba(0,0,0,0.45)] backdrop-blur-sm flex flex-col items-center gap-4">
      <div className="flex h-10 w-10 items-center justify-center rounded bg-accent/10 ring-1 ring-accent/30">
        <Leaf className="h-5 w-5 text-accent animate-pulse" aria-hidden="true" />
      </div>
      <p className="text-sm text-eco-400">Signing you in&hellip;</p>
    </div>
  );
}

export default function CallbackPage() {
  return (
    <Suspense
      fallback={
        <div className="rounded-2xl border border-white/[0.06] bg-eco-800/60 p-8 flex items-center justify-center">
          <Leaf className="h-5 w-5 text-accent animate-pulse" aria-hidden="true" />
        </div>
      }
    >
      <CallbackHandler />
    </Suspense>
  );
}
