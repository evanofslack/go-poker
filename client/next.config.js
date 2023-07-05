/** @type {import('next').NextConfig} */
module.exports = {
    output: 'standalone',
    reactStrictMode: true,
    images: {
        domains: [],
    },
    async rewrites() {
        return [
            {
                source: "/api/:path*",
                destination: process.env.NEXT_PUBLIC_API_URL + "/:path*",
            },
        ];
    },
};
