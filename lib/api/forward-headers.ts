/**
 * Helpers for forwarding the real client IP from Next.js proxies to the Go
 * backend.
 *
 * The backend rate limiter resolves the client IP from X-Forwarded-For only
 * when the immediate TCP peer is a trusted proxy (TRUSTED_PROXY_IPS). These
 * proxies therefore forward ONLY the first hop of the chain they received —
 * the original client, appended by nginx in front of Next.js. Dropping the
 * remaining hops prevents an attacker from hiding behind injected values.
 */
/**
 * Helpers for forwarding the real client IP from Next.js proxies to the Go
 * backend.
 *
 * Prefers X-Real-IP (overwritten by upstream reverse proxies like Nginx with
 * $remote_addr). When falling back to X-Forwarded-For, extracts the rightmost
 * hop appended by trusted reverse proxies to prevent spoofed left hops.
 */
export function extractClientIp(
  realIp: string | null | undefined,
  xff: string | null | undefined,
): string | undefined {
  if (realIp && realIp.trim().length > 0) {
    return realIp.trim();
  }
  if (!xff) return undefined;
  const hops = xff.split(',').map((h) => h.trim()).filter(Boolean);
  if (hops.length === 0) return undefined;
  // Rightmost hop is the IP appended by the closest trusted reverse proxy
  return hops[hops.length - 1];
}

/**
 * Backward compatibility alias for firstForwardedClientIp.
 */
export function firstForwardedClientIp(
  xff: string | null | undefined,
): string | undefined {
  return extractClientIp(undefined, xff);
}
