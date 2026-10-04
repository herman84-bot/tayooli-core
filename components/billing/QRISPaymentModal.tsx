'use client'

import { useState, useEffect, useRef } from 'react'
import { X, Clock, Loader2, Download } from 'lucide-react'
import { formatCurrency } from '@/lib/currency'
import QRCode from 'qrcode'

interface QRISPaymentModalProps {
  plan: string
  amount: number
  onClose: () => void
}

const PLAN_LABELS: Record<string, string> = {
  starter: 'Starter',
  bisnis: 'Bisnis',
}

export default function QRISPaymentModal({ plan, amount, onClose }: QRISPaymentModalProps) {
  const [qrImage, setQrImage] = useState<string | null>(null)
  const [paymentURL, setPaymentURL] = useState<string | null>(null)
  const [orderCode, setOrderCode] = useState<string>('')
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [timeLeft, setTimeLeft] = useState(900)
  const [totalPayment, setTotalPayment] = useState<number>(0)
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null)

  useEffect(() => {
    createPayment()
    return () => { if (timerRef.current) clearInterval(timerRef.current) }
  }, [])

  useEffect(() => {
    if (timeLeft <= 0) return
    timerRef.current = setInterval(() => { setTimeLeft((prev) => prev - 1) }, 1000)
    return () => { if (timerRef.current) clearInterval(timerRef.current) }
  }, [timeLeft])

  const createPayment = async () => {
    try {
      const res = await fetch('/api/v1/subscription/pay', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ plan, period: 'monthly' }),
      })
      if (!res.ok) throw new Error('Failed')
      const data = await res.json()
      const qrisString = data.payment_number || data.payment_url || ''
      setOrderCode(data.order_code || '')
      setTotalPayment(data.total_payment || amount)
      if (qrisString) {
        try {
          const qrDataUrl = await QRCode.toDataURL(qrisString, { width: 300, margin: 2, color: { dark: '#000000', light: '#FFFFFF' }, errorCorrectionLevel: 'M' })
          setQrImage(qrDataUrl)
        } catch { setPaymentURL(qrisString) }
      }
    } catch { setError('Gagal generate QRIS.') }
    finally { setIsLoading(false) }
  }

  const downloadQR = () => {
    if (!qrImage) return
    const canvas = document.createElement('canvas')
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    const qrSize = 400, pad = 40, textH = 120
    canvas.width = qrSize + pad * 2
    canvas.height = qrSize + pad * 2 + textH
    ctx.fillStyle = '#FFFFFF'
    ctx.fillRect(0, 0, canvas.width, canvas.height)
    const img = new Image()
    img.onload = () => {
      ctx.drawImage(img, pad, pad, qrSize, qrSize)
      ctx.fillStyle = '#1a1a1a'
      ctx.font = 'bold 24px sans-serif'
      ctx.textAlign = 'center'
      ctx.fillText('Pembayaran ' + (PLAN_LABELS[plan] || plan), canvas.width / 2, qrSize + pad + 40)
      ctx.font = 'bold 32px sans-serif'
      ctx.fillText(formatCurrency(totalPayment || amount), canvas.width / 2, qrSize + pad + 80)
      ctx.font = '16px sans-serif'
      ctx.fillStyle = '#666666'
      ctx.fillText('Scan QR ini menggunakan Dana, GoPay, OVO, ShopeePay, atau LinkAja', canvas.width / 2, qrSize + pad + 110)
      const link = document.createElement('a')
      link.download = 'QRIS-' + (PLAN_LABELS[plan] || plan) + '-' + (orderCode || 'payment') + '.png'
      link.href = canvas.toDataURL('image/png')
      link.click()
    }
    img.src = qrImage
  }

  const formatTime = (s: number) => Math.floor(s / 60) + ':' + (s % 60).toString().padStart(2, '0')

  return (
    <div className='fixed inset-0 z-50 flex items-center justify-center bg-black/50'>
      <div className='bg-card rounded-xl border border-border shadow-xl w-[400px] max-w-[90vw]'>
        <div className='flex items-center justify-between px-5 py-4 border-b border-border'>
          <h3 className='text-base font-semibold text-foreground'>Pembayaran {PLAN_LABELS[plan] || plan}</h3>
          <button onClick={onClose} className='p-1.5 rounded-md text-muted-foreground hover:bg-muted hover:text-foreground transition-colors' aria-label='Close'><X className='h-4 w-4' /></button>
        </div>
        <div className='px-5 py-6'>
          {isLoading ? (
            <div className='flex flex-col items-center py-8'><Loader2 className='h-8 w-8 text-primary animate-spin mb-3' /><p className='text-sm text-muted-foreground'>Menyiapkan pembayaran...</p></div>
          ) : qrImage || paymentURL ? (
            <div className='flex flex-col items-center'>
              <div className='text-center mb-4'><p className='text-2xl font-bold text-foreground'>{formatCurrency(totalPayment || amount)}</p><p className='text-xs text-muted-foreground mt-1'>Paket {PLAN_LABELS[plan]} / bulan</p></div>
              <div className='bg-white p-4 rounded-lg border border-border mb-4 relative group'>
                {qrImage ? (<><img src={qrImage} alt='QRIS Code' className='w-52 h-52' /><button onClick={downloadQR} className='absolute top-2 right-2 p-1.5 rounded-md bg-white/90 border border-border shadow-sm opacity-0 group-hover:opacity-100 transition-opacity hover:bg-zinc-50' aria-label='Download QRIS' title='Download QRIS'><Download className='h-4 w-4 text-muted-foreground' /></button></>) : paymentURL ? (<a href={paymentURL} target='_blank' rel='noopener noreferrer' className='block text-center'><div className='w-52 h-52 bg-zinc-100 rounded-lg flex items-center justify-center'><span className='text-sm text-primary font-medium'>Buka halaman pembayaran</span></div></a>) : null}
              </div>
              {qrImage && (<button onClick={downloadQR} className='mb-3 inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-primary bg-primary/5 hover:bg-primary/10 rounded-md transition-colors'><Download className='h-3.5 w-3.5' /> Download QRIS</button>)}
              <div className='text-center space-y-1.5'>
                <p className='text-xs text-muted-foreground'>Buka aplikasi dompet digital, lalu scan QR di atas:</p>
                <p className='text-xs font-medium text-foreground'>Dana - GoPay - OVO - ShopeePay - LinkAja</p>
                <p className='text-[11px] text-muted-foreground'>Nominal {formatCurrency(totalPayment || amount)} sudah otomatis terisi</p>
              </div>
              <div className='flex items-center gap-1.5 mt-4 text-xs text-muted-foreground'><Clock className='h-3.5 w-3.5' /><span>Berlaku {formatTime(timeLeft)}</span></div>
              {orderCode && <p className='text-[10px] text-muted-foreground mt-2'>Kode: {orderCode}</p>}
            </div>
          ) : (
            <div className='text-center py-6'><p className='text-sm text-muted-foreground mb-4'>{error || 'Gagal memuat QRIS'}</p><button onClick={createPayment} className='px-4 py-2 text-sm font-medium text-primary hover:bg-primary/10 rounded-lg transition-colors'>Coba Lagi</button></div>
          )}
        </div>
        <div className='px-5 py-3 border-t border-border'><p className='text-[10px] text-muted-foreground text-center'>Pembayaran diproses oleh Pakasir. Setelah bayar, subscription akan aktif otomatis.</p></div>
      </div>
    </div>
  )
}