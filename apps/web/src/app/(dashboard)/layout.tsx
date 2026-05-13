'use client';

import { useEffect, useState } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import { Leaf, AlertTriangle } from 'lucide-react';
import { Sidebar } from '@/components/shared/sidebar';
import { Header } from '@/components/shared/header';
import { useMe } from '@/lib/hooks/use-auth';
import { ApiError } from '@/lib/api';

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const router   = useRouter();
  const pathname = usePathname();
  const { data, error, isLoading } = useMe();

  useEffect(() => {
    if (error instanceof ApiError && error.status === 401) {
      router.replace('/login');
    }
  }, [error, router]);

  if (isLoading) {
    return (
      <div className="flex h-screen items-center justify-center bg-eco-900">
        <div className="flex flex-col items-center gap-4">
          <Leaf className="h-10 w-10 text-accent animate-pulse" aria-hidden="true" />
          <p className="text-xs font-mono text-eco-400 tracking-widest uppercase">Loading</p>
          <div className="flex gap-1" aria-label="Loading">
            {[0, 1, 2].map((i) => (
              <span
                key={i}
                className="h-1.5 w-1.5 rounded-full bg-accent animate-bounce"
                style={{ animationDelay: `${i * 0.15}s` }}
              />
            ))}
          </div>
        </div>
      </div>
    );
  }

  const apiUnreachable = error && !(error instanceof ApiError && error.status === 401);

  return (
    <div className="flex h-screen overflow-hidden bg-eco-900">
      <Sidebar
        open={sidebarOpen}
        onClose={() => setSidebarOpen(false)}
      />

      <div className="flex min-w-0 flex-1 flex-col overflow-hidden">
        <Header onMenuClick={() => setSidebarOpen(true)} />

        {apiUnreachable && (
          <div className="flex items-center gap-2 border-b border-red-500/30 bg-red-500/10 px-5 py-2">
            <AlertTriangle className="h-3.5 w-3.5 shrink-0 text-red-400" aria-hidden="true" />
            <p className="text-xs text-red-400">
              Cannot reach the API — check that the backend is running on{' '}
              <span className="font-mono">{process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}</span>
            </p>
          </div>
        )}

        <main
          id="main-content"
          className="flex-1 overflow-y-auto px-5 py-6 sm:px-6"
          tabIndex={-1}
        >
          {children}
        </main>
      </div>
    </div>
  );
}
