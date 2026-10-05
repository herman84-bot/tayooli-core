import React from 'react'
import { render, screen } from '@testing-library/react'
import '@testing-library/jest-dom'
import { AUTH_COOKIE } from '@/lib/auth/demo-auth'

const mockRedirect = jest.fn()
jest.mock('next/navigation', () => ({
  redirect: (url: string) => {
    mockRedirect(url)
    throw new Error(`NEXT_REDIRECT:${url}`)
  },
}))

const mockGet = jest.fn()
jest.mock('next/headers', () => ({
  cookies: jest.fn(async () => ({
    get: mockGet,
  })),
}))

jest.mock('next/link', () => {
  return function DummyLink({
    children,
    href,
    ...props
  }: React.ComponentProps<'a'>) {
    return (
      <a href={href} {...props}>
        {children}
      </a>
    )
  }
})

// Mock subcomponents to avoid unnecessary rendering overhead when testing landing page render
jest.mock('@/components/landing/PricingSection', () => ({
  PricingSection: () => <div data-testid="pricing-section">Pricing</div>,
}))
jest.mock('@/components/landing/SiteNav', () => ({
  SiteNav: () => <nav data-testid="site-nav">Nav</nav>,
}))

describe('LandingPage bypass / disabled redirect', () => {
  const originalEnv = process.env

  beforeEach(() => {
    jest.clearAllMocks()
    process.env = { ...originalEnv }
  })

  afterAll(() => {
    process.env = originalEnv
  })

  it('redirects to /dashboard when user has auth cookie and landing is disabled', async () => {
    mockGet.mockReturnValue({ name: AUTH_COOKIE, value: 'mock-session-token' })

    let LandingPage: () => Promise<React.ReactElement>
    jest.isolateModules(() => {
      delete process.env.NEXT_PUBLIC_DISABLE_LANDING_PAGE
      LandingPage = require('@/app/page').default
    })

    await expect(LandingPage!()).rejects.toThrow('NEXT_REDIRECT:/dashboard')
    expect(mockRedirect).toHaveBeenCalledWith('/dashboard')
  })

  it('redirects to /login when user has no auth cookie and landing is disabled', async () => {
    mockGet.mockReturnValue(undefined)

    let LandingPage: () => Promise<React.ReactElement>
    jest.isolateModules(() => {
      delete process.env.NEXT_PUBLIC_DISABLE_LANDING_PAGE
      LandingPage = require('@/app/page').default
    })

    await expect(LandingPage!()).rejects.toThrow('NEXT_REDIRECT:/login')
    expect(mockRedirect).toHaveBeenCalledWith('/login')
  })

  it('redirects to /login when auth cookie has empty value', async () => {
    mockGet.mockReturnValue({ name: AUTH_COOKIE, value: '' })

    let LandingPage: () => Promise<React.ReactElement>
    jest.isolateModules(() => {
      delete process.env.NEXT_PUBLIC_DISABLE_LANDING_PAGE
      LandingPage = require('@/app/page').default
    })

    await expect(LandingPage!()).rejects.toThrow('NEXT_REDIRECT:/login')
    expect(mockRedirect).toHaveBeenCalledWith('/login')
  })

  it('renders landing page when NEXT_PUBLIC_DISABLE_LANDING_PAGE is false', async () => {
    let LandingPage: () => Promise<React.ReactElement>
    jest.isolateModules(() => {
      process.env.NEXT_PUBLIC_DISABLE_LANDING_PAGE = 'false'
      LandingPage = require('@/app/page').default
    })

    const element = await LandingPage!()
    render(element)

    expect(mockRedirect).not.toHaveBeenCalled()
    expect(screen.getByTestId('site-nav')).toBeInTheDocument()
    expect(
      screen.getByText('Stok gudang rapi, kasir cepat, pengiriman terkontrol.')
    ).toBeInTheDocument()
  })
})
