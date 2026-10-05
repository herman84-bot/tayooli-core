import { NextRequest, NextResponse } from 'next/server'

export async function POST(req: NextRequest) {
  try {
    const body = await req.json().catch(() => ({}))
    const { category, description } = body

    if (!category || !description) {
      return NextResponse.json(
        { error: 'Kategori dan deskripsi laporan wajib diisi' },
        { status: 400 }
      )
    }

    console.log(`[Tayooli Support Report] Category: ${category} | Description: ${description}`)

    return NextResponse.json({
      success: true,
      message: 'Laporan Anda telah berhasil diterima oleh tim support Tayooli.',
      report_id: `REP-${Date.now()}`,
    })
  } catch (err) {
    console.error('Support report error:', err)
    return NextResponse.json(
      { error: 'Gagal memproses laporan' },
      { status: 500 }
    )
  }
}
