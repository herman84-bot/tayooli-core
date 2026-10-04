'use client'

import { useState, useRef, useEffect, useCallback } from 'react'
import { useChat } from '@/hooks/useChat'
import ChatMessage from './ChatMessage'
import ReportForm from './ReportForm'
import { cn } from '@/lib/utils'
import { MessageCircle, Send, Loader2, AlertTriangle } from 'lucide-react'

/**
 * Standalone chat panel — used on /help page.
 * No floating button, no open/close toggle.
 * Always rendered inline.
 */
export default function ChatPanel() {
  const [showReport, setShowReport] = useState(false)
  const [input, setInput] = useState('')
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const { messages, sendMessage, isSending } = useChat()

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  useEffect(() => {
    setTimeout(() => inputRef.current?.focus(), 200)
  }, [])

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
    <div className="flex flex-col h-full max-h-[calc(100vh-12rem)] bg-card rounded-lg border border-border">
      {/* Header */}
      <div className="flex items-center gap-3 px-5 py-4 border-b border-border">
        <div className="h-9 w-9 rounded-full bg-primary/10 flex items-center justify-center">
          <MessageCircle className="h-4.5 w-4.5 text-primary" />
        </div>
        <div>
          <h3 className="text-sm font-semibold text-foreground">Tayooli Support</h3>
          <p className="text-xs text-muted-foreground">
            {isSending ? 'Mengetik...' : 'Online — AI assistant'}
          </p>
        </div>
      </div>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto px-5 py-4 space-y-4">
        {messages.length === 0 && (
          <div className="flex flex-col items-center justify-center h-full text-center">
            <div className="h-12 w-12 rounded-full bg-primary/10 flex items-center justify-center mb-4">
              <MessageCircle className="h-6 w-6 text-primary" />
            </div>
            <p className="text-sm font-medium text-foreground">Halo! Ada yang bisa saya bantu?</p>
            <p className="text-xs text-muted-foreground mt-1.5 max-w-[320px]">
              Tanyakan tentang Tayooli ERP — invoice, gudang WMS, kasir POS, approval, atau pembayaran.
            </p>
            <div className="mt-4 flex flex-wrap justify-center gap-2 max-w-md">
              {[
                'Bagaimana cara pakai Kasir POS & cetak struk?',
                'Bagaimana alur transfer stok antar gudang di WMS?',
                'Cara buat Surat Jalan (DO) dan Stock Opname?',
                'Cara buat invoice dengan OCR?',
              ].map((suggestion) => (
                <button
                  key={suggestion}
                  onClick={() => sendMessage(suggestion)}
                  disabled={isSending}
                  className="text-xs bg-muted/60 hover:bg-muted text-muted-foreground hover:text-foreground px-2.5 py-1.5 rounded-full border border-border transition-colors text-left"
                >
                  {suggestion}
                </button>
              ))}
            </div>
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
          <div className="flex gap-3">
            <div className="h-8 w-8 rounded-full bg-zinc-100 flex items-center justify-center shrink-0">
              <Loader2 className="h-4 w-4 text-zinc-500 animate-spin" />
            </div>
            <div className="bg-zinc-100 rounded-xl rounded-bl-md px-4 py-3">
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
        <div className="px-5 pb-1">
          <button
            onClick={() => setShowReport(true)}
            className="w-full flex items-center justify-center gap-1.5 py-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            <AlertTriangle className="h-3.5 w-3.5" />
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
        <div className="px-4 pb-4">
          <div className="flex items-center gap-2 bg-zinc-100 rounded-lg px-3.5 py-2.5">
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
  )
}
