'use client';

import { useState } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { Copy, Check, Pencil, RefreshCw, X, Leaf, Zap, DollarSign, Clock } from 'lucide-react';
import { cn } from '@/lib/utils';
import { CodeBlock } from './code-block';
import type { Message } from '@/stores/chat-store';

interface MessageBubbleProps {
  message: Message;
  isStreaming?: boolean;
  isLast?: boolean;
  onRegenerate?: () => void;
  onEdit?: (messageId: string, newContent: string) => void;
}

function MetaPill({ children, tip }: { children: React.ReactNode; tip: string }) {
  return (
    <span className="group/pill relative inline-flex items-center gap-1">
      {children}
      <span
        role="tooltip"
        className="pointer-events-none absolute bottom-full left-0 z-50 mb-1.5 w-max max-w-[220px] rounded-md border border-eco-700 bg-eco-900 px-2 py-1 text-[10px] font-mono leading-snug text-eco-300 opacity-0 shadow-lg transition-opacity duration-150 group-hover/pill:opacity-100"
      >
        {tip}
      </span>
    </span>
  );
}

export function MessageBubble({
  message,
  isStreaming,
  isLast,
  onRegenerate,
  onEdit,
}: MessageBubbleProps) {
  const [copied, setCopied] = useState(false);
  const [editing, setEditing] = useState(false);
  const [editValue, setEditValue] = useState(message.content);

  const isUser = message.role === 'user';

  // Close any unclosed ** bold markers so react-markdown doesn't render them literally.
  // Only counts ** outside fenced code blocks and inline code spans.
  const renderContent = (text: string) => {
    const stripped = text.replace(/```[\s\S]*?```|`[^`]*`/g, '');
    const boldCount = (stripped.match(/\*\*/g) ?? []).length;
    return boldCount % 2 === 1 ? text + '**' : text;
  };

  const handleCopy = async () => {
    await navigator.clipboard.writeText(message.content);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleEditConfirm = () => {
    if (editValue.trim() && editValue !== message.content) {
      onEdit?.(message.id, editValue.trim());
    }
    setEditing(false);
  };

  const handleEditCancel = () => {
    setEditValue(message.content);
    setEditing(false);
  };

  if (isUser) {
    return (
      <div className="group flex justify-end gap-2">
        {/* Action buttons on hover */}
        <div className="flex items-start gap-1 pt-1 opacity-0 transition-opacity group-hover:opacity-100">
          <button
            onClick={handleCopy}
            className="rounded p-1 text-eco-500 hover:text-eco-300 transition-colors"
            title="Copy"
          >
            {copied ? <Check className="h-3.5 w-3.5 text-accent" /> : <Copy className="h-3.5 w-3.5" />}
          </button>
          {onEdit && !editing && (
            <button
              onClick={() => setEditing(true)}
              className="rounded p-1 text-eco-500 hover:text-eco-300 transition-colors"
              title="Edit"
            >
              <Pencil className="h-3.5 w-3.5" />
            </button>
          )}
        </div>

        {/* Bubble */}
        <div className="max-w-[75%]">
          {editing ? (
            <div className="space-y-2">
              <textarea
                value={editValue}
                onChange={(e) => setEditValue(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleEditConfirm(); }
                  if (e.key === 'Escape') handleEditCancel();
                }}
                className="w-full resize-none rounded-xl border border-accent/40 bg-eco-800 px-4 py-2.5 text-sm text-eco-100 focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
                rows={Math.min(editValue.split('\n').length + 1, 8)}
                autoFocus
              />
              <div className="flex justify-end gap-2">
                <button
                  onClick={handleEditCancel}
                  className="flex items-center gap-1 rounded-md px-2 py-1 font-mono text-xs text-eco-400 hover:text-eco-200 transition-colors"
                >
                  <X className="h-3 w-3" /> Cancel
                </button>
                <button
                  onClick={handleEditConfirm}
                  className="rounded-md bg-accent px-3 py-1 font-mono text-xs font-medium text-eco-900 hover:bg-accent/90 transition-colors"
                >
                  Save &amp; send
                </button>
              </div>
            </div>
          ) : (
            <div className="rounded-2xl rounded-tr-sm bg-accent/20 px-4 py-2.5 text-sm text-eco-100 ring-1 ring-accent/20">
              <p className="whitespace-pre-wrap leading-relaxed">{message.content}</p>
            </div>
          )}
        </div>
      </div>
    );
  }

  // Assistant bubble
  return (
    <div className="group flex flex-col gap-1">
      <div className="flex gap-3">
        {/* Avatar dot */}
        <div className="mt-1 flex h-5 w-5 flex-shrink-0 items-center justify-center rounded-full bg-accent/15 ring-1 ring-accent/30">
          <span className="h-1.5 w-1.5 rounded-full bg-accent" />
        </div>

        <div className="min-w-0 flex-1">
          {/* Content */}
          <div
            className={cn(
              'prose-sm max-w-none text-sm text-eco-100 leading-relaxed',
              '[&_p]:mb-2 [&_p:last-child]:mb-0',
              '[&_ul]:mb-2 [&_ul]:list-disc [&_ul]:pl-5 [&_ul]:space-y-0.5',
              '[&_ol]:mb-2 [&_ol]:list-decimal [&_ol]:pl-5 [&_ol]:space-y-0.5',
              '[&_blockquote]:border-l-2 [&_blockquote]:border-eco-600 [&_blockquote]:pl-3 [&_blockquote]:text-eco-400 [&_blockquote]:italic [&_blockquote]:my-2',
              '[&_strong]:font-semibold [&_strong]:text-eco-50',
              '[&_em]:italic',
              '[&_a]:text-accent [&_a]:underline',
              '[&_table]:w-full [&_table]:text-xs [&_table]:border-collapse [&_table]:my-2',
              '[&_th]:border [&_th]:border-eco-700 [&_th]:px-3 [&_th]:py-1.5 [&_th]:bg-eco-800 [&_th]:font-medium [&_th]:text-left',
              '[&_td]:border [&_td]:border-eco-700 [&_td]:px-3 [&_td]:py-1.5',
              '[&_hr]:border-eco-700 [&_hr]:my-3',
              '[&_h1]:text-base [&_h1]:font-semibold [&_h1]:text-eco-50 [&_h1]:mb-2 [&_h1]:mt-3',
              '[&_h2]:text-sm [&_h2]:font-semibold [&_h2]:text-eco-50 [&_h2]:mb-1.5 [&_h2]:mt-3',
              '[&_h3]:text-sm [&_h3]:font-medium [&_h3]:text-eco-100 [&_h3]:mb-1 [&_h3]:mt-2',
            )}
          >
            {message.content || isStreaming ? (
              <ReactMarkdown
                remarkPlugins={[remarkGfm]}
                components={{
                  pre({ children }) {
                    return <>{children}</>;
                  },
                  code({ className, children }) {
                    const match = /language-(\w+)/.exec(className || '');
                    const str = String(children).replace(/\n$/, '');
                    if (match || str.includes('\n')) {
                      return <CodeBlock language={match?.[1] ?? 'text'}>{str}</CodeBlock>;
                    }
                    return (
                      <code className="rounded px-1.5 py-0.5 font-mono text-[0.8em] bg-eco-800 text-accent/90 ring-1 ring-eco-700">
                        {children}
                      </code>
                    );
                  },
                }}
              >
                {renderContent(message.content)}
              </ReactMarkdown>
            ) : null}
            {isStreaming && (
              <span
                className="ml-0.5 inline-block h-4 w-0.5 animate-pulse rounded-full bg-accent align-middle"
                aria-hidden="true"
              />
            )}
          </div>

          {/* Per-message action row */}
          <div className="mt-1.5 flex items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100">
            <button
              onClick={handleCopy}
              className="flex items-center gap-1 rounded px-1.5 py-0.5 font-mono text-xs text-eco-300 hover:text-eco-100 transition-colors"
              title="Copy response"
            >
              {copied ? (
                <><Check className="h-3 w-3 text-accent" /> Copied</>
              ) : (
                <><Copy className="h-3 w-3" /> Copy</>
              )}
            </button>
            {isLast && onRegenerate && !isStreaming && (
              <button
                onClick={onRegenerate}
                className="flex items-center gap-1 rounded px-1.5 py-0.5 font-mono text-xs text-eco-300 hover:text-eco-100 transition-colors"
                title="Regenerate response"
              >
                <RefreshCw className="h-3 w-3" /> Regenerate
              </button>
            )}
          </div>

          {/* Metadata card — only when the backend returned EcoLLM routing data */}
          {message.meta && !isStreaming && (
            <div className="mt-2 opacity-50 transition-opacity duration-150 group-hover:opacity-100">
              <div className="inline-flex flex-wrap items-center gap-x-2.5 gap-y-1 rounded-lg border border-eco-700/70 bg-eco-800/70 px-2.5 py-1.5 text-[10px] font-mono backdrop-blur-sm">
                <MetaPill tip="The model EcoLLM routed this request to, chosen for the best balance of energy efficiency and quality.">
                  <span className="text-eco-300 font-medium">{message.meta.route.model_selected}</span>
                </MetaPill>
                <span className="text-eco-700">·</span>
                <MetaPill tip="Estimated CO₂ equivalent emitted to process this request, calculated from energy use and live grid carbon intensity.">
                  <Leaf className="h-2.5 w-2.5 text-accent" />
                  <span className="text-accent">{message.meta.energy.co2e_grams.toFixed(4)}g CO₂</span>
                </MetaPill>
                <span className="text-eco-700">·</span>
                <MetaPill tip="Electricity consumed in watt-hours, estimated from model size and inference time (or measured via GPU telemetry when available).">
                  <Zap className="h-2.5 w-2.5 text-accent" />
                  <span className="text-eco-300">{(message.meta.energy.total_energy_kwh * 1000).toFixed(4)} Wh</span>
                </MetaPill>
                <span className="text-eco-700">·</span>
                <MetaPill tip="Inference cost in USD for this request based on the selected model's per-token pricing.">
                  <DollarSign className="h-2.5 w-2.5 text-accent" />
                  <span className="text-eco-300">${message.meta.cost.total_cost_usd.toFixed(6)}</span>
                </MetaPill>
                <span className="text-eco-700">·</span>
                <MetaPill tip="Total time from sending your request to receiving the full response.">
                  <Clock className="h-2.5 w-2.5 text-accent" />
                  <span className="text-eco-300">{message.meta.performance.latency_ms}ms</span>
                </MetaPill>
                {message.meta.cost.savings_vs_gpt4_percent > 0 && (
                  <>
                    <span className="text-eco-700">·</span>
                    <MetaPill tip="How much cheaper this request was compared to running it on GPT-4, based on published pricing.">
                      <span className="text-accent">{message.meta.cost.savings_vs_gpt4_percent.toFixed(1)}% vs GPT-4</span>
                    </MetaPill>
                  </>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
