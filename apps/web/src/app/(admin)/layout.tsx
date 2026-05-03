'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Settings, Cpu, GitBranch, BarChart3, Zap } from 'lucide-react';
import { cn } from '@/lib/utils';

const ADMIN_NAV = [
  { href: '/admin/models', label: 'Models', icon: <Cpu className="h-4 w-4" /> },
  { href: '/admin/routes', label: 'Routes', icon: <GitBranch className="h-4 w-4" /> },
  { href: '/admin/metrics', label: 'Metrics', icon: <BarChart3 className="h-4 w-4" /> },
];

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();

  return (
    <div className="flex h-screen overflow-hidden bg-gray-50 dark:bg-gray-950">
      <aside className="flex w-56 flex-col border-r border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-900">
        <div className="flex h-14 items-center gap-2 px-4 border-b border-gray-100 dark:border-gray-700">
          <Settings className="h-4 w-4 text-gray-500" aria-hidden="true" />
          <span className="text-sm font-semibold text-gray-900 dark:text-white">Admin</span>
        </div>

        <nav className="flex-1 px-3 py-4" aria-label="Admin navigation">
          <ul role="list" className="space-y-0.5">
            {ADMIN_NAV.map((item) => {
              const isActive = pathname === item.href || pathname.startsWith(item.href + '/');
              return (
                <li key={item.href}>
                  <Link
                    href={item.href as Parameters<typeof Link>[0]['href']}
                    aria-current={isActive ? 'page' : undefined}
                    className={cn(
                      'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                      'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-green-500',
                      isActive
                        ? 'bg-green-50 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                        : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-white',
                    )}
                  >
                    <span aria-hidden="true">{item.icon}</span>
                    {item.label}
                  </Link>
                </li>
              );
            })}
          </ul>
        </nav>

        <div className="border-t border-gray-100 px-4 py-3 dark:border-gray-700">
          <Link
            href="/overview"
            className="flex items-center gap-2 text-xs text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300"
          >
            <Zap className="h-3 w-3" aria-hidden="true" />
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
