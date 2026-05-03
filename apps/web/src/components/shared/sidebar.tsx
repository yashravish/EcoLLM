'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import {
  LayoutDashboard,
  List,
  Cpu,
  Leaf,
  BarChart3,
  Key,
  Settings,
  Zap,
  X,
  Terminal,
  DollarSign,
} from 'lucide-react';
import { cn } from '@/lib/utils';

interface NavItem {
  href: string;
  label: string;
  icon: React.ReactNode;
}

const NAV_ITEMS: NavItem[] = [
  { href: '/overview', label: 'Overview', icon: <LayoutDashboard className="h-4 w-4" /> },
  { href: '/playground', label: 'Playground', icon: <Terminal className="h-4 w-4" /> },
  { href: '/requests', label: 'Requests', icon: <List className="h-4 w-4" /> },
  { href: '/models', label: 'Models', icon: <Cpu className="h-4 w-4" /> },
  { href: '/carbon', label: 'Carbon', icon: <Leaf className="h-4 w-4" /> },
  { href: '/usage', label: 'Usage', icon: <BarChart3 className="h-4 w-4" /> },
  { href: '/billing', label: 'Billing', icon: <DollarSign className="h-4 w-4" /> },
  { href: '/api-keys', label: 'API Keys', icon: <Key className="h-4 w-4" /> },
  { href: '/settings', label: 'Settings', icon: <Settings className="h-4 w-4" /> },
];

interface SidebarProps {
  open?: boolean;
  onClose?: () => void;
}

export function Sidebar({ open = true, onClose }: SidebarProps) {
  const pathname = usePathname();

  return (
    <>
      {/* Mobile overlay */}
      {open && onClose && (
        <div
          className="fixed inset-0 z-20 bg-black/40 lg:hidden"
          onClick={onClose}
          aria-hidden="true"
        />
      )}

      <aside
        className={cn(
          'fixed inset-y-0 left-0 z-30 flex w-60 flex-col bg-white shadow-sm dark:bg-gray-900',
          'border-r border-gray-200 dark:border-gray-700',
          'transition-transform duration-200 motion-reduce:transition-none',
          'lg:relative lg:z-auto lg:translate-x-0',
          open ? 'translate-x-0' : '-translate-x-full',
        )}
        aria-label="Main navigation"
      >
        {/* Logo */}
        <div className="flex h-14 items-center justify-between px-4 border-b border-gray-100 dark:border-gray-700">
          <Link
            href="/overview"
            className="flex items-center gap-2 text-sm font-bold text-gray-900 dark:text-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-green-500 rounded"
          >
            <Zap className="h-5 w-5 text-green-600" aria-hidden="true" />
            EcoLLM
          </Link>
          {onClose && (
            <button
              onClick={onClose}
              className="lg:hidden rounded p-1 text-gray-500 hover:bg-gray-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-green-500 dark:text-gray-400 dark:hover:bg-gray-800"
              aria-label="Close navigation"
            >
              <X className="h-4 w-4" aria-hidden="true" />
            </button>
          )}
        </div>

        {/* Nav links */}
        <nav className="flex-1 overflow-y-auto px-3 py-4">
          <ul role="list" className="space-y-0.5">
            {NAV_ITEMS.map((item) => {
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

        {/* Footer */}
        <div className="border-t border-gray-100 px-4 py-3 dark:border-gray-700">
          <p className="text-xs text-gray-400 dark:text-gray-500">
            Carbon-aware inference
          </p>
        </div>
      </aside>
    </>
  );
}
