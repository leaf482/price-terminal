import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Keep development startup from generating repository instruction files.
  agentRules: false,
};

export default nextConfig;
