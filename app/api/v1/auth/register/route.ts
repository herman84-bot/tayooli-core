import { NextResponse } from "next/server"

export const dynamic = "force-dynamic"

const REGISTRATION_DISABLED_PAYLOAD = {
  error: {
    code: "REGISTRATION_DISABLED",
    message: "Pendaftaran akun baru saat ini dinonaktifkan oleh administrator.",
  },
}

function handleDisabledRegistration() {
  return NextResponse.json(REGISTRATION_DISABLED_PAYLOAD, { status: 403 })
}

export async function GET() {
  return handleDisabledRegistration()
}

export async function POST() {
  return handleDisabledRegistration()
}

export async function PUT() {
  return handleDisabledRegistration()
}

export async function PATCH() {
  return handleDisabledRegistration()
}

export async function DELETE() {
  return handleDisabledRegistration()
}
