import { sveltekit } from '@sveltejs/kit/vite';
import { SvelteKitPWA } from '@vite-pwa/sveltekit';
import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [
		sveltekit(),
		...SvelteKitPWA({
			// 仅缓存壳页与静态资源，不缓存 POST，避免数据不一致
			registerType: 'autoUpdate',
			includeAssets: ['favicon.svg', 'icon-192.png', 'icon-512.png', 'apple-touch-icon.png'],
			manifest: {
				name: '青野集 · 家庭园艺植物记录与养护',
				short_name: '青野集',
				description: '家庭园艺植物记录与养护工具',
				theme_color: '#43a047',
				background_color: '#f7faf5',
				display: 'standalone',
				start_url: '/',
				icons: [
					{ src: 'icon-192.png', sizes: '192x192', type: 'image/png', purpose: 'any' },
					{ src: 'icon-512.png', sizes: '512x512', type: 'image/png', purpose: 'any' },
					{ src: 'icon-512.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
					{ src: 'favicon.svg', sizes: 'any', type: 'image/svg+xml', purpose: 'any' }
				]
			},
			workbox: {
				// 阻断插件默认注入 prerendered/** 空 glob：本应用为纯客户端 SPA（ssr=false/prerender=false），
				// 无预渲染产物，该空 glob 会触发 Workbox 警告。置空 modifyURLPrefix 可跳过 buildGlobPatterns 自动追加。
				modifyURLPrefix: {},
				// 仅预缓存真实存在的 client 构建产物（壳页与静态资源），不再包含无意义的 prerendered 模式
				globPatterns: ['client/**/*.{js,css,ico,png,svg,webp,webmanifest}'],
				// API 与上传走网络
				navigateFallbackDenylist: [/^\/api/],
				runtimeCaching: [
					{
						// 业务数据 GET：离线时从缓存读取（stale-while-revalidate）
						urlPattern: ({ url, request }) =>
							url.pathname.startsWith('/api/') && request.method === 'GET',
						handler: 'StaleWhileRevalidate',
						options: {
							cacheName: 'api-get',
							expiration: { maxEntries: 300, maxAgeSeconds: 60 * 60 * 24 * 30 },
							cacheableResponse: { statuses: [200] }
						}
					},
					{
						// 上传的植物/日记图片：离线可读
						urlPattern: ({ url }) => url.pathname.startsWith('/uploads/'),
						handler: 'CacheFirst',
						options: {
							cacheName: 'uploads',
							expiration: { maxEntries: 200, maxAgeSeconds: 60 * 60 * 24 * 30 },
							cacheableResponse: { statuses: [200] }
						}
					}
				]
			},
			devOptions: { enabled: false }
		})
	],
	server: {
		port: 5173,
		proxy: {
			// 开发期将 API 与照片请求代理到后端（用 127.0.0.1 避免 localhost 解析歧义）
			'/api': { target: 'http://127.0.0.1:8081', changeOrigin: true },
			'/uploads': { target: 'http://127.0.0.1:8081', changeOrigin: true }
		}
	},
	test: {
		include: ['src/**/*.{test,spec}.ts'],
		environment: 'node'
	}
});
