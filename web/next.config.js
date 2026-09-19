const FRAME_HEADERS = [
  { key: "X-Frame-Options", value: "SAMEORIGIN" },
  { key: "Content-Security-Policy", value: "frame-ancestors 'self'" },
];

const SITE_FILE = "/:file([A-Za-z0-9][A-Za-z0-9._-]*\\.(?:html|htm|txt|xml|HTML|HTM|TXT|XML))";

const SITE_FILE_HEADERS = [
  {
    key: "Content-Security-Policy",
    value: "default-src 'none'; style-src 'unsafe-inline'; img-src data:; sandbox; frame-ancestors 'self'",
  },
  { key: "X-Content-Type-Options", value: "nosniff" },
];

const nextConfig = {
  output: "standalone",
  images: {
    unoptimized: true,
    remotePatterns: [],
  },
  async headers() {
    return [
      { source: "/", headers: FRAME_HEADERS },
      { source: "/:path((?!monitoring/public/).+)", headers: FRAME_HEADERS },
      { source: SITE_FILE, headers: SITE_FILE_HEADERS },
      { source: "/api/site-files/:file", headers: SITE_FILE_HEADERS },
    ];
  },
  async rewrites() {
    return {
      fallback: [{ source: SITE_FILE, destination: "/api/site-files/:file" }],
    };
  },
};

module.exports = nextConfig;
