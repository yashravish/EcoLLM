'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Settings, Cpu, GitBranch, BarChart3, Leaf } from 'lucide-react';
import { cn } from '@/lib/utils';

const ADMIN_NAV = [
  { href: '/admin/models', label: 'Models', icon: <Cpu className="h-4 w-4" /> },
  { href: '/admin/routes', label: 'Routes', icon: <GitBranch className="h-4 w-4" /> },
  { href: '/admin/metrics', label: 'Metrics', icon: <BarChart3 className="h-4 w-4" /> },
];

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();

  return (
    <div className="flex h-screen overflow-hidden bg-eco-900">
      <aside className="flex w-56 flex-col border-r border-eco-600 bg-eco-800">
        <div className="flex h-12 items-center gap-2 px-4 border-b border-eco-600">
          <Settings className="h-4 w-4 text-eco-400" aria-hidden="true" />
          <span className="font-mono text-xs font-semibold uppercase tracking-widest text-eco-300">Admin</span>
        </div>

        <nav className="flex-1 px-2 py-3" aria-label="Admin navigation">
          <ul role="list" className="space-y-0.5">
            {ADMIN_NAV.map((item) => {
              const isActive = pathname === item.href || pathname.startsWith(item.href + '/');
              return (
                <li key={item.href}>
                  <Link
                    href={item.href as Parameters<typeof Link>[0]['href']}
                    aria-current={isActive ? 'page' : undefined}
                    className={cn(
                      'flex items-center gap-3 rounded-md px-3 py-2 font-mono text-xs font-medium transition-colors',
                      'focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent',
                      isActive
                        ? 'bg-eco-700 text-accent'
                        : 'text-eco-300 hover:bg-eco-700 hover:text-eco-100',
                    )}
                  >
                    <span aria-hidden="true" className={isActive ? 'text-accent' : 'text-eco-400'}>{item.icon}</span>
                    {item.label}
                  </Link>
                </li>
              );
            })}
          </ul>
        </nav>

        <div className="border-t border-eco-600 px-4 py-3">
          <Link
            href="/playground"
            className="flex items-center gap-2 font-mono text-xs text-eco-500 hover:text-eco-200 transition-colors"
          >
            <Leaf className="h-3 w-3" aria-hidden="true" />
            Back to dashboard
          </Link>
        </div>
      </aside>

      <div className="flex min-w-0 flex-1 flex-col overflow-hidden">
        <main className="flex-1 overflow-y-auto px-6 py-6">{children}</main>
      </div>
    </div>
  );
}
