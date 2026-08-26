<script lang="ts">
	import Icon from './Icon.svelte';
	import { theme, THEME_MODE_LABEL, type ThemeMode } from '$lib/theme.svelte';

	const ORDER: ThemeMode[] = ['light', 'dark', 'auto'];
	// 各模式对应的图标与提示
	const META: Record<ThemeMode, { icon: string; hint: string }> = {
		light: { icon: 'sun', hint: '亮色' },
		dark: { icon: 'moon', hint: '暗色' },
		auto: { icon: 'monitor', hint: '跟随系统' }
	};

	function cycle() {
		theme.cycle();
	}

	const nextMode = $derived(ORDER[(ORDER.indexOf(theme.mode) + 1) % ORDER.length]);
	const label = $derived(
		`主题：${THEME_MODE_LABEL[theme.mode]}（点击切换为${THEME_MODE_LABEL[nextMode]}）`
	);
</script>

<button class="theme-toggle" onclick={cycle} title={label} aria-label={label}>
	<span class="icon-wrap">
		<Icon name={META[theme.mode].icon} size={17} />
	</span>
	{#if theme.mode === 'auto'}
		<!-- 自动模式下叠加小圆点标识实际生效的主题 -->
		<span class="dot" class:dark={theme.resolved === 'dark'}></span>
	{/if}
</button>

<style>
	.theme-toggle {
		position: relative;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 36px;
		height: 36px;
		border-radius: 12px;
		color: var(--text-secondary);
		border: 1px solid var(--border);
		background: var(--surface-2);
		transition: color 0.15s, border-color 0.15s, background 0.15s;
	}

	.theme-toggle:hover {
		color: var(--green-700);
		border-color: var(--green-500);
		background: var(--green-50);
	}

	.icon-wrap {
		display: flex;
	}

	.dot {
		position: absolute;
		top: 5px;
		right: 5px;
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: var(--warning);
	}
	.dot.dark {
		background: var(--green-600);
	}
</style>
