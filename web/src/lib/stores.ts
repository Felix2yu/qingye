import { writable } from 'svelte/store';
import { browser } from '$app/environment';
import { api, type UserSetting } from './api';

// 全局设置（工作日、偏好），供多个页面读取
export const settings = writable<UserSetting | null>(null);

export async function loadSettings() {
	if (!browser) return;
	try {
		settings.set(await api.getSettings());
	} catch {
		/* 忽略加载失败 */
	}
}

// 简单 toast 提示
export const toast = writable<{ id: number; text: string; type: 'ok' | 'err' } | null>(null);

export function showToast(text: string, type: 'ok' | 'err' = 'ok') {
	toast.set({ id: Date.now(), text, type });
	setTimeout(() => toast.set(null), 2400);
}
