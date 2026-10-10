/// <reference types="@sveltejs/kit" />

// `virtual:pwa-register` 的类型声明。
// 这里手写而非用 `/// <reference types="vite-plugin-pwa/client" />`：vite-plugin-pwa 只是
// @vite-pwa/sveltekit 的传递依赖，pnpm 严格 node_modules 下从应用侧解析不到它的类型入口。
declare module 'virtual:pwa-register' {
	export interface RegisterSWOptions {
		immediate?: boolean;
		onNeedRefresh?: () => void;
		onOfflineReady?: () => void;
		onRegistered?: (registration: ServiceWorkerRegistration | undefined) => void;
		onRegisterError?: (error: unknown) => void;
	}

	export function registerSW(options?: RegisterSWOptions): (reload?: boolean) => Promise<void>;
}
