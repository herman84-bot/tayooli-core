import { NextResponse } from "next/server"
import type { NextRequest } from "next/server"

const AUTH_COOKIE = "tayooli_auth"

export function middleware(request: NextRequest) {
  if (request.nextUrl.pathname === "/") {
    const isLandingDisabled =
      process.env.NEXT_PUBLIC_DISABLE_LANDING_PAGE !== "false"

    if (isLandingDisabled) {
      const authCookie = request.cookies.get(AUTH_COOKIE)?.value
      if (authCookie) {
        return NextResponse.redirect(new URL("/dashboard", request.url))
      }
      return NextResponse.redirect(new URL("/login", request.url))
    }
  }
  return NextResponse.next()
}

export const config = {
  matcher: ["/"],
}
