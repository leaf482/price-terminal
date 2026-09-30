import type { NextConfig } from "next";
import { backendURL } from "./lib/backend-url";

const backend = backendURL(process.env.BACKEND_URL);

const nextConfig: NextConfig = {
  // Keep development startup from generating repository instruction files.
  agentRules: false,
  async rewrites() {
    return [{ source: "/api/:path*", destination: `${backend}/:path*` }];
  },
};

export default nextConfig;
