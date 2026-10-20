/** @type {import('next').NextConfig} */
module.exports = {
    output: 'export',
    reactStrictMode: true,
    turbopack: {
        root: __dirname,
    },
    images: {
        unoptimized: true,
    },
};
