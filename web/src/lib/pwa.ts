import { browser } from '$app/environment';

/**
 * 注册 Service Worker（构建时由 vite-plugin-pwa 生成 `/sw.js`）。
 *
 * 为什么必须显式注册：`@vite-pwa/sveltekit` 会**移除**负责 HTML 注入的 `vite-plugin-pwa:build`
 * 插件，其替代者只做产物生成、没有 `transformIndexHtml`。因此 SvelteKit 下即使配了
 * `registerType: 'autoUpdate'` 也不会自动注册，必须在应用代码里调用 `virtual:pwa-register`。
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
