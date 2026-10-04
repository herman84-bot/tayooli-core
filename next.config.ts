import type {NextConfig} from 'next';

const nextConfig: NextConfig = {
  reactStrictMode: true,
  // Allow the Freebuff preview proxy origin to load /_next/* dev chunks.
  // Without this, Next.js dev blocks cross-origin chunk requests and the
  // client bundle can fail to hydrate ("Rendered more hooks..." etc.).
  allowedDevOrigins: ['*.daytonaproxy01.net'],
  eslint: {
    ignoreDuringBuilds: true,
  },
  typescript: {
    ignoreBuildErrors: false,
  },
  // Allow access to remote image placeholder.
  images: {
    remotePatterns: [
      {
        protocol: 'https',
        hostname: 'picsum.photos',
        port: '',
        pathname: '/**', // This allows any path under the hostname
      },
    ],
  },
  output: 'standalone',
  experimental: {
    // Single build worker: server production runs on a 1-vCPU e2-micro,
    // parallel workers only multiply memory pressure there (build OOM).
    cpus: 1,
    // Reduce webpack dev memory usage: the dev server was being OOM-killed
    // (cgroup limit ~2GB) while compiling heavy routes (6000+ modules),
    // which produced truncated chunks and hydration errors in preview.
    webpackMemoryOptimizations: true,
    // Tree-shake barrel exports from heavy UI libs so fewer modules are
    // compiled per route, keeping dev server memory under the sandbox limit.
    optimizePackageImports: ['lucide-react', 'recharts', 'motion'],
  },
  transpilePackages: ['motion'],
  webpack: (config, {dev}) => {
    // HMR is disabled in AI Studio via DISABLE_HMR env var.
    // Do not modifyâfile watching is disabled to prevent flickering during agent edits.
    if (dev && process.env.DISABLE_HMR === 'true') {
      config.watchOptions = {
        ignored: /.*/,
      };
    }
    return config;
  },
};

export default nextConfig;
