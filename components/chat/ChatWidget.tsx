'use client'

import { useState, useRef, useEffect, useCallback } from 'react'
import { useChat } from '@/hooks/useChat'
import ChatMessage from './ChatMessage'
import ReportForm from './ReportForm'
import { cn } from '@/lib/utils'
import { MessageCircle, X, Send, Loader2, AlertTriangle } from 'lucide-react'

export default function ChatWidget() {
  const [isOpen, setIsOpen] = useState(false)
  const [showReport, setShowReport] = useState(false)
  const [input, setInput] = useState('')
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const { messages, sendMessage, isSending } = useChat()

  // Auto-scroll to bottom on new messages
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  // Focus input when panel opens
  useEffect(() => {
    if (isOpen) {
      setTimeout(() => inputRef.current?.focus(), 100)
    }
  }, [isOpen])

  const handleSend = useCallback(() => {
    const text = input.trim()
    if (!text || isSending) return
    setInput('')
    sendMessage(text)
  }, [input, isSending, sendMessage])

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  return (
    <>
      {/* ── Floating Button ── */}
      {!isOpen && (
        <button
          onClick={() => setIsOpen(true)}
          className={cn(
            'fixed bottom-5 right-5 z-50',
            'h-12 w-12 rounded-full',
            'bg-primary text-primary-foreground',
            'flex items-center justify-center',
            'shadow-lg hover:shadow-xl',
            'transition-all duration-200',
            'hover:scale-105 active:scale-95'
          )}
          aria-label="Open chat support"
          data-tutorial="chat-widget"
        >
          <MessageCircle className="h-5 w-5" />
        </button>
      )}

      {/* ── Chat Panel ── */}
      {isOpen && (
        <div
          className={cn(
            'fixed bottom-5 right-5 z-50',
            'w-[380px] max-w-[calc(100vw-2rem)]',
            'h-[520px] max-h-[calc(100vh-4rem)]',
            'bg-card rounded-xl border border-border',
            'shadow-xl',
            'flex flex-col',
            'animate-in slide-in-from-bottom-2 fade-in duration-200'
          )}
        >
          {/* Header */}
          <div className="flex items-center justify-between px-4 py-3 border-b border-border">
            <div className="flex items-center gap-2.5">
              <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center">
                <MessageCircle className="h-4 w-4 text-primary" />
              </div>
              <div>
                <h3 className="text-sm font-semibold text-foreground">Tayooli Support</h3>
                <p className="text-[11px] text-muted-foreground">
                  {isSending ? 'Mengetik...' : 'Online'}
                </p>
              </div>
            </div>
            <button
              onClick={() => setIsOpen(false)}
              className="p-1.5 rounded-md text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
              aria-label="Close chat"
            >
              <X className="h-4 w-4" />
            </button>
          </div>

          {/* Messages */}
          <div className="flex-1 overflow-y-auto px-4 py-3 space-y-3">
            {messages.length === 0 && (
              <div className="flex flex-col items-center justify-center h-full text-center">
                <div className="h-10 w-10 rounded-full bg-primary/10 flex items-center justify-center mb-3">
                  <MessageCircle className="h-5 w-5 text-primary" />
                </div>
                <p className="text-sm font-medium text-foreground">Halo! 👋</p>
                <p className="text-xs text-muted-foreground mt-1 max-w-[240px]">
                  Ada yang bisa saya bantu tentang Tayooli ERP?
                </p>
              </div>
            )}

            {messages.map((msg, i) => (
              <ChatMessage
                key={i}
                role={msg.role as 'user' | 'assistant'}
                content={msg.content}
                timestamp={msg.timestamp}
              />
            ))}

            {isSending && (
              <div className="flex gap-2.5">
                <div className="h-7 w-7 rounded-full bg-zinc-100 flex items-center justify-center">
                  <Loader2 className="h-3.5 w-3.5 text-zinc-500 animate-spin" />
                </div>
                <div className="bg-zinc-100 rounded-xl rounded-bl-md px-3.5 py-2.5">
                  <div className="flex gap-1">
                    <span className="h-1.5 w-1.5 bg-zinc-400 rounded-full animate-bounce [animation-delay:0ms]" />
                    <span className="h-1.5 w-1.5 bg-zinc-400 rounded-full animate-bounce [animation-delay:150ms]" />
                    <span className="h-1.5 w-1.5 bg-zinc-400 rounded-full animate-bounce [animation-delay:300ms]" />
                  </div>
                </div>
              </div>
            )}

            <div ref={messagesEndRef} />
          </div>

          {/* Report button */}
          {messages.length > 0 && !showReport && (
            <div className="px-4 pb-1">
              <button
                onClick={() => setShowReport(true)}
                className="w-full flex items-center justify-center gap-1.5 py-1.5 text-[11px] text-muted-foreground hover:text-foreground transition-colors"
              >
                <AlertTriangle className="h-3 w-3" />
                Laporkan Masalah
              </button>
            </div>
          )}

          {/* Report Form */}
          {showReport && (
            <ReportForm onClose={() => setShowReport(false)} />
          )}

          {/* Input */}
          {!showReport && (
          <div className="px-3 pb-3">
            <div className="flex items-center gap-2 bg-zinc-100 rounded-lg px-3 py-2">
              <input
                ref={inputRef}
                type="text"
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder="Ketik pesan..."
                disabled={isSending}
                className="flex-1 bg-transparent text-sm text-foreground placeholder:text-muted-foreground outline-none disabled:opacity-50"
              />
              <button
                onClick={handleSend}
                disabled={!input.trim() || isSending}
                className={cn(
                  'p-1.5 rounded-md transition-colors',
                  input.trim() && !isSending
                    ? 'text-primary hover:bg-primary/10'
                    : 'text-muted-foreground/40'
                )}
                aria-label="Send message"
              >
                <Send className="h-4 w-4" />
              </button>
            </div>
          </div>
          )}
        </div>
      )}
    </>
  )
}
