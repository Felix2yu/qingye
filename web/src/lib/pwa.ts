import { browser } from '$app/env';

/**
 * 注册 Service Worker（构建时由 vite-plugin-pwa 生成 `/sw.js`）。
 *
 * 为什么必须显式注册：SvelteKit 3 不再支持 `transformIndexHtml` 钩子，
 * vite-plugin-pwa 依赖该钩子完成的 HTML 注入（含自动注册脚本）不会生效。
 * 因此必须在应用代码里调用 `virtual:pwa-register`，注册逻辑不能交给插件。
 * 这是 PWA规范.md 第四节对 SvelteKit 的约定写法（吾身 diarum 亦同）。
 */
export function registerServiceWorker(): void {
	if (!browser) return;
	// 仅生产构建注册：dev 下无预缓存产物，注册只会带来噪音
	if (!import.meta.env.PROD) return;
	if (!window.isSecureContext) return;
	if (!('serviceWorker' in navigator)) return;

	void (async () => {
		try {
			const { registerSW } = await import('virtual:pwa-register');
			registerSW({ immediate: true });
		} catch (e) {
			console.warn('[PWA] Service Worker 注册失败', e);
		}
	})();
}
