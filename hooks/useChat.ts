'use client'

import { useState, useCallback, useRef, useEffect } from 'react'

export interface ChatMessage {
  role: 'user' | 'assistant' | 'system'
  content: string
  timestamp?: string
}

interface ChatState {
  messages: ChatMessage[]
  sessionId: string
  isSending: boolean
  isThinking: boolean
}

export function useChat() {
  const [state, setState] = useState<ChatState>({
    messages: [],
    sessionId: '',
    isSending: false,
    isThinking: false,
  })

  // Timer and lifecycle references for thinking delay and progressive streaming
  const isMountedRef = useRef(true)
  const thinkingTimerRef = useRef<NodeJS.Timeout | null>(null)
  const typingIntervalRef = useRef<NodeJS.Timeout | null>(null)
  const abortControllerRef = useRef<AbortController | null>(null)
  const isSendingRef = useRef(false)

  // Cleanup timers and abort controllers on unmount
  useEffect(() => {
    isMountedRef.current = true
    return () => {
      isMountedRef.current = false
      if (thinkingTimerRef.current) clearTimeout(thinkingTimerRef.current)
      if (typingIntervalRef.current) clearInterval(typingIntervalRef.current)
      if (abortControllerRef.current) abortControllerRef.current.abort()
      isSendingRef.current = false
    }
  }, [])

  /**
   * Progressive typewriter streaming:
   * Splits response into natural word/whitespace tokens and appends them
   * with adaptive pacing to simulate authentic human CS typing.
   */
  const streamAssistantResponse = useCallback(
    (fullText: string, newSessionId?: string) => {
      // Split by words and whitespace preserving exact formatting
      const tokens = fullText.match(/(\s+|\S+)/g) || [fullText]
      const totalTokens = tokens.length

      // Adaptive chunking and interval to balance realism with responsiveness:
      // Short texts: 1 token per 28ms (~1.5s total)
      // Medium texts: 2 tokens per 24ms (~2.2s total)
      // Long texts: 3-4 tokens per 20ms (~2.8s total)
      const tokensPerTick = totalTokens > 300 ? 4 : totalTokens > 150 ? 2 : 1
      const tickIntervalMs = totalTokens > 300 ? 20 : totalTokens > 150 ? 24 : 28

      let currentIndex = 0
      let accumulatedText = ''

      // Add the initial empty assistant message
      const initialAssistantMsg: ChatMessage = {
        role: 'assistant',
        content: '',
        timestamp: new Date().toISOString(),
      }

      setState((prev) => ({
        ...prev,
        messages: [...prev.messages, initialAssistantMsg],
        sessionId: newSessionId || prev.sessionId,
        isThinking: false,
        isSending: true,
      }))

      // Clear any existing typing interval
      if (typingIntervalRef.current) clearInterval(typingIntervalRef.current)

      typingIntervalRef.current = setInterval(() => {
        if (!isMountedRef.current) {
          if (typingIntervalRef.current) {
            clearInterval(typingIntervalRef.current)
            typingIntervalRef.current = null
          }
          return
        }

        const nextIndex = Math.min(currentIndex + tokensPerTick, totalTokens)
        for (let i = currentIndex; i < nextIndex; i++) {
          accumulatedText += tokens[i]
        }
        currentIndex = nextIndex

        const currentStream = accumulatedText
        const isFinished = currentIndex >= totalTokens

        setState((prev) => {
          const msgs = [...prev.messages]
          if (msgs.length > 0 && msgs[msgs.length - 1].role === 'assistant') {
            msgs[msgs.length - 1] = {
              ...msgs[msgs.length - 1],
              content: currentStream,
            }
          }
          return {
            ...prev,
            messages: msgs,
            isSending: !isFinished,
          }
        })

        if (isFinished) {
          isSendingRef.current = false
          if (typingIntervalRef.current) {
            clearInterval(typingIntervalRef.current)
            typingIntervalRef.current = null
          }
        }
      }, tickIntervalMs)
    },
    []
  )

  const sendMessage = useCallback(
    async (message: string) => {
      const text = message.trim()
      if (!text || isSendingRef.current || state.isSending || !isMountedRef.current) return
      isSendingRef.current = true

      // Clear any pending timers and in-flight fetch
      if (thinkingTimerRef.current) clearTimeout(thinkingTimerRef.current)
      if (typingIntervalRef.current) clearInterval(typingIntervalRef.current)
      if (abortControllerRef.current) abortControllerRef.current.abort()

      const controller = new AbortController()
      abortControllerRef.current = controller

      // 1. Add user message immediately & show thinking indicator
      const userMsg: ChatMessage = {
        role: 'user',
        content: text,
        timestamp: new Date().toISOString(),
      }

      setState((prev) => ({
        ...prev,
        messages: [...prev.messages, userMsg],
        isSending: true,
        isThinking: true,
      }))

      const startTime = Date.now()
      // Realistic human CS thinking pause: between 750ms and 1100ms
      const targetThinkingDelay = 850

      try {
        const res = await fetch('/api/v1/chat/message', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          credentials: 'include',
          signal: controller.signal,
          body: JSON.stringify({
            message: text,
            session_id: state.sessionId,
          }),
        })

        if (!isMountedRef.current) return

        if (!res.ok) {
          throw new Error('Gagal menghubungi layanan chat')
        }

        const data = await res.json()
        const fullReply = data.reply || 'Halo kak, ada yang bisa kami bantu seputar Tayooli ERP?'

        // Calculate remaining thinking delay to ensure natural rhythm
        const elapsedTime = Date.now() - startTime
        const remainingDelay = Math.max(0, targetThinkingDelay - elapsedTime)

        thinkingTimerRef.current = setTimeout(() => {
          if (!isMountedRef.current) return
          streamAssistantResponse(fullReply, data.session_id)
        }, remainingDelay)
      } catch (err) {
        if ((err as Error)?.name === 'AbortError' || !isMountedRef.current) {
          return
        }

        isSendingRef.current = false
        const elapsedTime = Date.now() - startTime
        const remainingDelay = Math.max(0, targetThinkingDelay - elapsedTime)

        thinkingTimerRef.current = setTimeout(() => {
          if (!isMountedRef.current) return
          const errorMsg: ChatMessage = {
            role: 'assistant',
            content:
              'Mohon maaf kak, koneksi chat sedang mengalami sedikit gangguan. Silakan coba kirim ulang pertanyaan Anda ya.',
            timestamp: new Date().toISOString(),
          }

          setState((prev) => ({
            ...prev,
            messages: [...prev.messages, errorMsg],
            isThinking: false,
            isSending: false,
          }))
        }, remainingDelay)
      }
    },
    [state.isSending, state.sessionId, streamAssistantResponse]
  )

  return {
    messages: state.messages,
    sessionId: state.sessionId,
    isSending: state.isSending,
    isThinking: state.isThinking,
    sendMessage,
  }
}
