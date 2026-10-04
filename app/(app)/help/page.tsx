import ChatPanel from '@/components/chat/ChatPanel'

export default function HelpPage() {
  return (
    <div className="p-6 max-w-3xl">
      <div className="mb-6">
        <h1 className="text-lg font-semibold text-foreground">Help & Support</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Tanyakan apa saja tentang Tayooli ERP. AI assistant kami siap membantu.
        </p>
      </div>

      <ChatPanel />
    </div>
  )
}
