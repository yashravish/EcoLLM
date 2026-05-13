import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { EcoLLMMetadata } from '@/types/api';

export interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  meta?: EcoLLMMetadata;
}

export interface Conversation {
  id: string;
  title: string;
  messages: Message[];
  createdAt: number;
  lastMeta: EcoLLMMetadata | null;
}

const MAX_CONVERSATIONS = 20;

interface ChatStore {
  conversations: Conversation[];
  activeConversationId: string | null;
  selectedModel: string;
  apiKey: string;
  systemPrompt: string;
  prefer: 'efficiency' | 'speed' | 'quality';
  settingsOpen: boolean;
  metaSidebarOpen: boolean;
  isStreaming: boolean;
  streamingMessageId: string | null;

  newConversation: () => void;
  setActiveConversation: (id: string) => void;
  deleteConversation: (id: string) => void;
  addUserMessage: (content: string) => void;
  startAssistantMessage: () => void;
  appendToken: (token: string) => void;
  completeAssistantMessage: (meta: EcoLLMMetadata | null) => void;
  addErrorMessage: (content: string) => void;
  removeLastAssistantMessage: () => void;
  editUserMessage: (messageId: string, newContent: string) => void;

  setApiKey: (k: string) => void;
  setSystemPrompt: (p: string) => void;
  setSelectedModel: (m: string) => void;
  setPrefer: (p: 'efficiency' | 'speed' | 'quality') => void;
  toggleSettings: () => void;
  toggleMetaSidebar: () => void;
}

export const useChatStore = create<ChatStore>()(
  persist(
    (set) => ({
      conversations: [],
      activeConversationId: null,
      selectedModel: '',
      apiKey: '',
      systemPrompt: 'You are a helpful assistant.',
      prefer: 'efficiency',
      settingsOpen: false,
      metaSidebarOpen: false,
      isStreaming: false,
      streamingMessageId: null,

      newConversation: () => set({ activeConversationId: null }),

      setActiveConversation: (id) => set({ activeConversationId: id }),

      deleteConversation: (id) =>
        set((s) => ({
          conversations: s.conversations.filter((c) => c.id !== id),
          activeConversationId: s.activeConversationId === id ? null : s.activeConversationId,
        })),

      addUserMessage: (content) =>
        set((s) => {
          let convId = s.activeConversationId;
          let conversations = s.conversations;
          const title = content.length > 30 ? content.slice(0, 30) + '…' : content;

          if (!convId) {
            convId = crypto.randomUUID();
            const newConv: Conversation = {
              id: convId,
              title,
              messages: [],
              createdAt: Date.now(),
              lastMeta: null,
            };
            conversations = [newConv, ...conversations].slice(0, MAX_CONVERSATIONS);
          }

          const msg: Message = { id: crypto.randomUUID(), role: 'user', content };
          const isFirstUser = !conversations
            .find((c) => c.id === convId)
            ?.messages.some((m) => m.role === 'user');

          return {
            activeConversationId: convId,
            conversations: conversations.map((c) =>
              c.id !== convId
                ? c
                : {
                    ...c,
                    title: isFirstUser ? title : c.title,
                    messages: [...c.messages, msg],
                  },
            ),
          };
        }),

      startAssistantMessage: () => {
        const msgId = crypto.randomUUID();
        set((s) => {
          const convId = s.activeConversationId;
          if (!convId) return {};
          const msg: Message = { id: msgId, role: 'assistant', content: '' };
          return {
            isStreaming: true,
            streamingMessageId: msgId,
            conversations: s.conversations.map((c) =>
              c.id === convId ? { ...c, messages: [...c.messages, msg] } : c,
            ),
          };
        });
      },

      appendToken: (token) =>
        set((s) => {
          const { activeConversationId: convId, streamingMessageId: msgId } = s;
          if (!convId || !msgId) return {};
          return {
            conversations: s.conversations.map((c) =>
              c.id !== convId
                ? c
                : {
                    ...c,
                    messages: c.messages.map((m) =>
                      m.id === msgId ? { ...m, content: m.content + token } : m,
                    ),
                  },
            ),
          };
        }),

      completeAssistantMessage: (meta) =>
        set((s) => ({
          isStreaming: false,
          streamingMessageId: null,
          conversations: s.conversations.map((c) =>
            c.id === s.activeConversationId
              ? {
                  ...c,
                  lastMeta: meta,
                  messages: c.messages.map((m) =>
                    m.id === s.streamingMessageId && meta ? { ...m, meta } : m,
                  ),
                }
              : c,
          ),
        })),

      addErrorMessage: (content) =>
        set((s) => {
          const convId = s.activeConversationId;
          const errMsg: Message = { id: crypto.randomUUID(), role: 'assistant', content };
          if (!convId) return { isStreaming: false, streamingMessageId: null };
          return {
            isStreaming: false,
            streamingMessageId: null,
            conversations: s.conversations.map((c) =>
              c.id !== convId
                ? c
                : {
                    ...c,
                    messages: [
                      ...c.messages.filter((m) => m.id !== s.streamingMessageId),
                      errMsg,
                    ],
                  },
            ),
          };
        }),

      removeLastAssistantMessage: () =>
        set((s) => {
          const convId = s.activeConversationId;
          if (!convId) return {};
          return {
            conversations: s.conversations.map((c) => {
              if (c.id !== convId) return c;
              const msgs = [...c.messages];
              if (msgs[msgs.length - 1]?.role === 'assistant') msgs.pop();
              return { ...c, messages: msgs };
            }),
          };
        }),

      editUserMessage: (messageId, newContent) =>
        set((s) => {
          const convId = s.activeConversationId;
          if (!convId) return {};
          return {
            conversations: s.conversations.map((c) => {
              if (c.id !== convId) return c;
              const idx = c.messages.findIndex((m) => m.id === messageId);
              if (idx === -1) return c;
              return {
                ...c,
                messages: c.messages
                  .slice(0, idx + 1)
                  .map((m, i) => (i === idx ? { ...m, content: newContent } : m)),
              };
            }),
          };
        }),

      setApiKey: (apiKey) => set({ apiKey }),
      setSystemPrompt: (systemPrompt) => set({ systemPrompt }),
      setSelectedModel: (selectedModel) => set({ selectedModel }),
      setPrefer: (prefer) => set({ prefer }),
      toggleSettings: () => set((s) => ({ settingsOpen: !s.settingsOpen })),
      toggleMetaSidebar: () => set((s) => ({ metaSidebarOpen: !s.metaSidebarOpen })),
    }),
    {
      name: 'ecollm-chat',
      partialize: (s) => ({
        conversations: s.conversations.map((c) => ({
          ...c,
          messages: c.messages.slice(-100),
        })),
        activeConversationId: s.activeConversationId,
        selectedModel: s.selectedModel,
        apiKey: s.apiKey,
        systemPrompt: s.systemPrompt,
        prefer: s.prefer,
      }),
    },
  ),
);
