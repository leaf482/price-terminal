import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Keep development startup from generating repository instruction files.
  agentRules: false,
  async rewrites() {
    return [{ source: "/api/:path*", destination: `${process.env.BACKEND_URL || "http://127.0.0.1:8080"}/:path*` }];
  },
};

export default nextConfig;
