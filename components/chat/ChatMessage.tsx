'use client'

import { cn } from '@/lib/utils'
import { Bot, User } from 'lucide-react'

interface ChatMessageProps {
  role: 'user' | 'assistant' | 'system'
  content: string
  timestamp?: string
}

export default function ChatMessage({ role, content, timestamp }: ChatMessageProps) {
  if (role === 'system') return null

  const isUser = role === 'user'

  return (
    <div className={cn('flex gap-2.5', isUser ? 'flex-row-reverse' : 'flex-row')}>
      {/* Avatar */}
      <div
        className={cn(
          'shrink-0 h-7 w-7 rounded-full flex items-center justify-center',
          isUser
            ? 'bg-primary/10 text-primary'
            : 'bg-zinc-100 text-zinc-500'
        )}
      >
        {isUser ? (
          <User className="h-3.5 w-3.5" />
        ) : (
          <Bot className="h-3.5 w-3.5" />
        )}
      </div>

      {/* Bubble */}
      <div
        className={cn(
          'max-w-[80%] rounded-xl px-3.5 py-2.5 text-[13px] leading-relaxed',
          isUser
            ? 'bg-primary text-primary-foreground rounded-br-md'
            : 'bg-zinc-100 text-foreground rounded-bl-md'
        )}
      >
        <p className="whitespace-pre-wrap">{content}</p>
        {timestamp && (
          <p className={cn(
            'mt-1 text-[10px]',
            isUser ? 'text-primary-foreground/60' : 'text-muted-foreground'
          )}>
            {new Date(timestamp).toLocaleTimeString('id-ID', {
              hour: '2-digit',
              minute: '2-digit',
            })}
          </p>
        )}
      </div>
    </div>
  )
}
