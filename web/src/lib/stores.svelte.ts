import { browser } from '$app/environment';
import { api, type UserSetting } from './api';

type ToastData = { id: number; text: string; type: 'ok' | 'err' | 'info' } | null;

// Svelte 5 runes 全局状态（替代 svelte/store 的 writable）
// 通过 getter/setter 暴露，使跨组件响应式读取与 svelte/store 自动订阅等价
let _settings = $state<UserSetting | null>(null);
let _toast = $state<ToastData>(null);

export const settings = {
	get current(): UserSetting | null {
		return _settings;
	},
	set(v: UserSetting | null) {
		_settings = v;
	}
};

export const toast = {
	get current(): ToastData {
		return _toast;
	},
	set(v: ToastData) {
		_toast = v;
	}
};

// 全局设置（工作日、偏好），供多个页面读取
export async function loadSettings() {
	if (!browser) return;
	try {
		settings.set(await api.getSettings());
	} catch {
		/* 忽略加载失败 */
	}
}

// 简单 toast 提示
export function showToast(text: string, type: 'ok' | 'err' | 'info' = 'ok') {
	toast.set({ id: Date.now(), text, type });
	setTimeout(() => toast.set(null), 2400);
}
