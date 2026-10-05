'use client'

import React from 'react'
import { cn } from '@/lib/utils'
import { Bot, User } from 'lucide-react'

interface ChatMessageProps {
  role: 'user' | 'assistant' | 'system'
  content: string
  timestamp?: string
}

function renderInlineMarkdown(cleanLine: string, keyPrefix: string): React.ReactNode[] {
  const parts: React.ReactNode[] = []
  const regex = /(\*\*.*?\*\*|`.*?`|\*.*?\*)/g
  let lastIndex = 0
  let match: RegExpExecArray | null

  while ((match = regex.exec(cleanLine)) !== null) {
    if (match.index > lastIndex) {
      parts.push(cleanLine.slice(lastIndex, match.index))
    }
    const token = match[0]
    if (token.startsWith('**') && token.endsWith('**')) {
      parts.push(
        <strong key={`${keyPrefix}-${match.index}`} className="font-semibold text-foreground">
          {token.slice(2, -2)}
        </strong>
      )
    } else if (token.startsWith('`') && token.endsWith('`')) {
      parts.push(
        <code
          key={`${keyPrefix}-${match.index}`}
          className="px-1.5 py-0.5 rounded bg-muted/80 font-mono text-[11px] text-foreground border border-border/50"
        >
          {token.slice(1, -1)}
        </code>
      )
    } else if (token.startsWith('*') && token.endsWith('*')) {
      parts.push(
        <span key={`${keyPrefix}-${match.index}`} className="italic font-medium text-foreground">
          {token.slice(1, -1)}
        </span>
      )
    }
    lastIndex = regex.lastIndex
  }

  if (lastIndex < cleanLine.length) {
    parts.push(cleanLine.slice(lastIndex))
  }

  return parts.length > 0 ? parts : [cleanLine]
}

function renderFormattedContent(content: string, isUser: boolean) {
  if (isUser) {
    return <p className="whitespace-pre-wrap">{content}</p>
  }

  const lines = content.split('\n')

  return (
    <div className="space-y-1.5 text-[13px] leading-relaxed">
      {lines.map((line, idx) => {
        const trimmed = line.trim()
        if (!trimmed) {
          return <div key={idx} className="h-1" />
        }

        // Check if bullet point or ordered list
        const isBullet = trimmed.startsWith('- ') || trimmed.startsWith('• ')
        const isNumbered = /^\d+\.\s+/.test(trimmed)

        if (isBullet || isNumbered) {
          const cleanText = isBullet ? trimmed.slice(2) : trimmed.replace(/^\d+\.\s+/, '')
          const prefix = isBullet ? '• ' : trimmed.match(/^\d+\.\s+/)?.[0] ?? ''

          return (
            <div key={idx} className="flex items-start gap-1.5 pl-1">
              <span className="font-semibold text-primary shrink-0 select-none">{prefix}</span>
              <div className="flex-1">{renderInlineMarkdown(cleanText, `line-${idx}`)}</div>
            </div>
          )
        }

        return (
          <div key={idx} className="text-foreground">
            {renderInlineMarkdown(trimmed, `line-${idx}`)}
          </div>
        )
      })}
    </div>
  )
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
          isUser ? 'bg-primary/10 text-primary' : 'bg-zinc-100 text-zinc-500'
        )}
      >
        {isUser ? <User className="h-3.5 w-3.5" /> : <Bot className="h-3.5 w-3.5" />}
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
        {renderFormattedContent(content, isUser)}
        {timestamp && (
          <p
            className={cn(
              'mt-1 text-[10px]',
              isUser ? 'text-primary-foreground/60' : 'text-muted-foreground'
            )}
          >
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
