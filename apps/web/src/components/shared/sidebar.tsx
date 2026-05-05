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
  { href: '/overview',   label: 'Overview',   icon: <LayoutDashboard className="h-4 w-4" /> },
  { href: '/playground', label: 'Playground',  icon: <Terminal className="h-4 w-4" /> },
  { href: '/requests',   label: 'Requests',    icon: <List className="h-4 w-4" /> },
  { href: '/models',     label: 'Models',      icon: <Cpu className="h-4 w-4" /> },
  { href: '/carbon',     label: 'Carbon',      icon: <Leaf className="h-4 w-4" /> },
  { href: '/usage',      label: 'Usage',       icon: <BarChart3 className="h-4 w-4" /> },
  { href: '/billing',    label: 'Billing',     icon: <DollarSign className="h-4 w-4" /> },
  { href: '/api-keys',   label: 'API Keys',    icon: <Key className="h-4 w-4" /> },
  { href: '/settings',   label: 'Settings',    icon: <Settings className="h-4 w-4" /> },
];

interface SidebarProps {
  open?: boolean;
  onClose?: () => void;
}

export function Sidebar({ open = true, onClose }: SidebarProps) {
  const pathname = usePathname();

  return (
    <>
      {open && onClose && (
        <div
          className="fixed inset-0 z-20 bg-black/60 backdrop-blur-sm lg:hidden"
          onClick={onClose}
          aria-hidden="true"
        />
      )}

      <aside
        className={cn(
          'fixed inset-y-0 left-0 z-30 flex w-56 flex-col',
          'bg-eco-800 border-r border-eco-600',
          'transition-transform duration-200 motion-reduce:transition-none',
          'lg:relative lg:z-auto lg:translate-x-0',
          open ? 'translate-x-0' : '-translate-x-full',
        )}
        aria-label="Main navigation"
      >
        {/* Logo */}
        <div className="flex h-12 items-center justify-between px-4 border-b border-eco-600">
          <Link
            href="/overview"
            className="flex items-center gap-2.5 group focus-visible:outline-none"
          >
            <div className="flex h-7 w-7 items-center justify-center rounded bg-accent/10 ring-1 ring-accent/30 transition-all group-hover:ring-accent/60 group-hover:bg-accent/15">
              <Leaf className="h-4 w-4 text-accent" aria-hidden="true" />
            </div>
            <span className="text-sm font-semibold text-eco-50 tracking-wide">EcoLLM</span>
          </Link>

          {onClose && (
            <button
              onClick={onClose}
              className="lg:hidden rounded p-1 text-eco-400 hover:text-eco-100 hover:bg-eco-700 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent"
              aria-label="Close navigation"
            >
              <X className="h-4 w-4" aria-hidden="true" />
            </button>
          )}
        </div>

        {/* Nav */}
        <nav className="flex-1 overflow-y-auto px-2 py-3">
          <p className="mb-1.5 px-3 text-[10px] font-semibold uppercase tracking-[0.15em] text-eco-400">
            Navigation
          </p>
          <ul role="list" className="space-y-0.5">
            {NAV_ITEMS.map((item) => {
              const isActive =
                pathname === item.href || pathname.startsWith(item.href + '/');
              return (
                <li key={item.href}>
                  <Link
                    href={item.href as Parameters<typeof Link>[0]['href']}
                    aria-current={isActive ? 'page' : undefined}
                    className={cn(
                      'relative flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-all duration-150',
                      'focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent',
                      isActive
                        ? 'bg-eco-700 text-accent'
                        : 'text-eco-300 hover:bg-eco-700 hover:text-eco-100',
                    )}
                  >
                    {isActive && (
                      <span
                        className="absolute left-0 top-1/2 -translate-y-1/2 h-4 w-[2px] rounded-full bg-accent"
                        aria-hidden="true"
                      />
                    )}
                    <span aria-hidden="true" className={isActive ? 'text-accent' : 'text-eco-400'}>
                      {item.icon}
                    </span>
                    {item.label}
                  </Link>
                </li>
              );
            })}
          </ul>
        </nav>

        {/* Footer */}
        <div className="border-t border-eco-600 px-4 py-3">
          <div className="flex items-center gap-2">
            <span className="inline-block h-1.5 w-1.5 rounded-full bg-accent animate-pulse" aria-hidden="true" />
            <p className="text-[10px] font-mono text-eco-400 uppercase tracking-widest">
              Carbon-aware
            </p>
          </div>
        </div>
      </aside>
    </>
  );
}
