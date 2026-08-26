// 主题状态管理：亮色 / 暗色 / 跟随系统
// 「自动」模式监听系统偏好变化并实时切换；偏好持久化到 localStorage
import { browser } from '$app/environment';

export type ThemeMode = 'light' | 'dark' | 'auto';
export type ResolvedTheme = 'light' | 'dark';

const STORAGE_KEY = 'qingye-theme';
const MODES: ThemeMode[] = ['light', 'dark', 'auto'];
/** 与 app.css 两套主题的 --bg 保持一致 */
const BG_COLOR: Record<ResolvedTheme, string> = { light: '#f7f5ef', dark: '#141812' };

function systemDark(): boolean {
	return window.matchMedia('(prefers-color-scheme: dark)').matches;
}

/** 同步浏览器地址栏/状态栏颜色 */
function updateMetaTheme(resolved: ResolvedTheme) {
	let meta = document.querySelector<HTMLMetaElement>('meta[name="theme-color"]');
	if (!meta) {
		meta = document.createElement('meta');
		meta.name = 'theme-color';
		document.head.appendChild(meta);
	}
	meta.content = BG_COLOR[resolved];
}

class ThemeStore {
	/** 用户选择的模式（含 auto） */
	mode = $state<ThemeMode>('auto');
	/** 实际生效的主题 */
	resolved = $state<ResolvedTheme>('light');

	#media: MediaQueryList | null = null;

	init() {
		if (!browser || this.#media) return;
		const saved = localStorage.getItem(STORAGE_KEY) as ThemeMode | null;
		this.mode = saved && MODES.includes(saved) ? saved : 'auto';
		this.#apply();
		this.#media = window.matchMedia('(prefers-color-scheme: dark)');
		// 系统在亮/暗间切换时，「自动」模式实时跟随
		this.#media.addEventListener('change', () => {
			if (this.mode === 'auto') this.#apply();
		});
	}

	set(mode: ThemeMode) {
		this.mode = mode;
		if (browser) localStorage.setItem(STORAGE_KEY, mode);
		this.#apply();
	}

	/** 循环切换：亮色 → 暗色 → 自动 */
	cycle() {
		const next: ThemeMode =
			this.mode === 'light' ? 'dark' : this.mode === 'dark' ? 'auto' : 'light';
		this.set(next);
	}

	#apply() {
		if (!browser) return;
		const dark = this.mode === 'dark' || (this.mode === 'auto' && systemDark());
		this.resolved = dark ? 'dark' : 'light';
		document.documentElement.dataset.theme = this.resolved;
		updateMetaTheme(this.resolved);
	}
}

export const theme = new ThemeStore();

export const THEME_MODE_LABEL: Record<ThemeMode, string> = {
	light: '亮色',
	dark: '暗色',
	auto: '跟随系统'
};
