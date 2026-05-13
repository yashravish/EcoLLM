'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import {
  Leaf,
  Key,
  Settings,
  X,
  Plus,
  Trash2,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { useChatStore } from '@/stores/chat-store';

interface NavItem {
  href: string;
  label: string;
  icon: React.ReactNode;
}

const SECONDARY_NAV: NavItem[] = [
  { href: '/api-keys',   label: 'API Keys',  icon: <Key className="h-4 w-4" /> },
  { href: '/settings',   label: 'Settings',  icon: <Settings className="h-4 w-4" /> },
];

function NavLink({ item, pathname, onClose }: { item: NavItem; pathname: string; onClose?: () => void }) {
  const isActive = pathname === item.href || pathname.startsWith(item.href + '/');

  return (
    <li>
      <Link
        href={item.href}
        onClick={onClose}
        aria-current={isActive ? 'page' : undefined}
        className={cn(
          'flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-all duration-150',
          'focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent',
          isActive
            ? 'bg-eco-700/60 text-eco-100'
            : 'text-eco-400 hover:bg-eco-700/40 hover:text-eco-200',
        )}
      >
        <span aria-hidden="true" className={isActive ? 'text-eco-200' : 'text-eco-500'}>
          {item.icon}
        </span>
        {item.label}
      </Link>
    </li>
  );
}

interface SidebarProps {
  open?: boolean;
  onClose?: () => void;
}

export function Sidebar({ open = true, onClose }: SidebarProps) {
  const pathname = usePathname();
  const router = useRouter();

  const { conversations, activeConversationId, newConversation, setActiveConversation, deleteConversation } =
    useChatStore();

  const handleNewChat = () => {
    newConversation();
    if (pathname !== '/playground') router.push('/playground');
    onClose?.();
  };

  const handleSelectConversation = (id: string) => {
    setActiveConversation(id);
    if (pathname !== '/playground') router.push('/playground');
    onClose?.();
  };

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
          <a
            href="/playground"
            className="flex items-center gap-3 group focus-visible:outline-none"
          >
            <Leaf className="h-5 w-5 text-accent transition-opacity group-hover:opacity-80" aria-hidden="true" />
            <span className="text-sm font-semibold text-eco-50 tracking-wide">EcoLLM</span>
          </a>

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

        {/* New Chat button */}
        <div className="px-3 pt-4 pb-2">
          <button
            onClick={handleNewChat}
            className={cn(
              'flex w-full items-center gap-2 rounded-md px-3 py-2.5 text-sm font-medium transition-all duration-150',
              'border border-eco-600 text-eco-200 hover:border-accent/50 hover:bg-eco-700 hover:text-eco-50',
              'focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent',
            )}
          >
            <Plus className="h-4 w-4 text-eco-400" aria-hidden="true" />
            New chat
          </button>
        </div>

        {/* Scrollable content: conversations */}
        <nav className="flex-1 overflow-y-auto px-3 py-2">
          {/* Conversation history */}
          {conversations.length > 0 && (
            <div className="mb-2">
              <p className="mb-2 px-2 text-[10px] font-semibold uppercase tracking-[0.15em] text-eco-300">
                Recent
              </p>
              <ul role="list" className="space-y-0.5">
                {conversations.map((conv) => {
                  const isActive = conv.id === activeConversationId && pathname === '/playground';
                  return (
                    <li key={conv.id} className="group/conv relative">
                      <button
                        onClick={() => handleSelectConversation(conv.id)}
                        className={cn(
                          'flex w-full items-center rounded-md px-2 py-2 text-left text-xs transition-all duration-150',
                          'focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent',
                          isActive
                            ? 'bg-eco-700 text-eco-100'
                            : 'text-eco-400 hover:bg-eco-700/60 hover:text-eco-200',
                        )}
                      >
                        <span className="flex-1 truncate pr-4">{conv.title}</span>
                      </button>
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          deleteConversation(conv.id);
                        }}
                        className={cn(
                          'absolute right-1 top-1/2 -translate-y-1/2 rounded p-0.5',
                          'text-eco-600 hover:text-red-400 transition-colors',
                          'opacity-0 group-hover/conv:opacity-100',
                          'focus-visible:outline-none focus-visible:opacity-100',
                        )}
                        aria-label={`Delete conversation: ${conv.title}`}
                      >
                        <Trash2 className="h-3 w-3" />
                      </button>
                    </li>
                  );
                })}
              </ul>
            </div>
          )}

        </nav>

        {/* Secondary nav — pinned above footer, always visible */}
        <div className="border-t border-eco-700 px-3 py-3">
          <ul role="list" className="space-y-1">
            {SECONDARY_NAV.map((item) => (
              <NavLink key={item.href} item={item} pathname={pathname} onClose={onClose} />
            ))}
          </ul>
        </div>
      </aside>
    </>
  );
}
