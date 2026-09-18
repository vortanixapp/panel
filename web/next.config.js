const FRAME_HEADERS = [
  { key: "X-Frame-Options", value: "SAMEORIGIN" },
  { key: "Content-Security-Policy", value: "frame-ancestors 'self'" },
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
    ];
  },
};

module.exports = nextConfig;
