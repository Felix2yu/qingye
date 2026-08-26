import { sveltekit } from '@sveltejs/kit/vite';
import { SvelteKitPWA } from '@vite-pwa/sveltekit';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [
		sveltekit(),
		...SvelteKitPWA({
			// 仅缓存壳页与静态资源，不缓存 POST，避免数据不一致
			registerType: 'autoUpdate',
			includeAssets: ['favicon.svg'],
			manifest: {
				name: '青野 · 家庭园艺养护',
				short_name: '青野',
				description: '家庭园艺植物记录与养护工具',
				theme_color: '#2f6b3a',
				background_color: '#f7faf5',
				display: 'standalone',
				start_url: '/',
				icons: [
					{ src: 'favicon.svg', sizes: 'any', type: 'image/svg+xml', purpose: 'any maskable' }
				]
			},
			workbox: {
				globPatterns: ['**/*.{js,css,html,svg,png,ico}'],
				// API 与上传走网络
				navigateFallbackDenylist: [/^\/api/]
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
	}
});
