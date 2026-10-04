'use client'

import { useState, useCallback } from 'react'

interface ChatMessage {
  role: string
  content: string
  timestamp?: string
}

interface ChatState {
  messages: ChatMessage[]
  sessionId: string
  isSending: boolean
}

export function useChat() {
  const [state, setState] = useState<ChatState>({
    messages: [],
    sessionId: '',
    isSending: false,
  })

  const sendMessage = useCallback(async (message: string) => {
    // Add user message immediately
    const userMsg: ChatMessage = {
      role: 'user',
      content: message,
      timestamp: new Date().toISOString(),
    }

    setState((prev) => ({
      ...prev,
      messages: [...prev.messages, userMsg],
      isSending: true,
    }))

    try {
      const res = await fetch('/api/v1/chat/message', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          message,
          session_id: state.sessionId,
        }),
      })

      if (!res.ok) {
        throw new Error('Chat request failed')
      }

      const data = await res.json()

      const aiMsg: ChatMessage = {
        role: 'assistant',
        content: data.reply || 'Maaf, terjadi kesalahan.',
        timestamp: new Date().toISOString(),
      }

      setState((prev) => ({
        ...prev,
        messages: [...prev.messages, aiMsg],
        sessionId: data.session_id || prev.sessionId,
        isSending: false,
      }))
    } catch {
      const errorMsg: ChatMessage = {
        role: 'assistant',
        content: 'Layanan chat sedang tidak tersedia. Silakan coba lagi.',
        timestamp: new Date().toISOString(),
      }

      setState((prev) => ({
        ...prev,
        messages: [...prev.messages, errorMsg],
        isSending: false,
      }))
    }
  }, [state.sessionId])

  return {
    messages: state.messages,
    sessionId: state.sessionId,
    isSending: state.isSending,
    sendMessage,
  }
}
