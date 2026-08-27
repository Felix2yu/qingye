<script lang="ts">
	import { page } from '$app/stores';
	import Icon from './Icon.svelte';
	import ThemeToggle from './ThemeToggle.svelte';
	import { theme, THEME_MODE_LABEL } from '$lib/theme.svelte';

	const items = [
		{ href: '/', label: '今日', icon: 'leaf' },
		{ href: '/plants', label: '植物', icon: 'sprout' },
		{ href: '/tasks', label: '任务', icon: 'tasks' },
		{ href: '/diary', label: '日记', icon: 'camera' },
		{ href: '/library', label: '资料库', icon: 'library' },
		{ href: '/import', label: '导入', icon: 'import' },
		{ href: '/settings', label: '设置', icon: 'settings' }
	];

	function isActive(href: string): boolean {
		const path = $page.url.pathname;
		if (href === '/') return path === '/';
		return path === href || path.startsWith(href + '/');
	}
</script>

<!-- 桌面端：侧边栏 -->
<aside class="sidebar">
	<div class="brand">
		<span class="logo"><Icon name="sprout" size={26} /></span>
		<span class="brand-name">青野集</span>
	</div>
	<nav>
		{#each items as it}
			<a class="nav-item" class:active={isActive(it.href)} href={it.href}>
				<span class="nav-icon"><Icon name={it.icon} size={18} /></span>
				<span>{it.label}</span>
			</a>
		{/each}
	</nav>
	<div class="sidebar-foot">
		<div class="theme-row">
			<ThemeToggle />
			<div class="theme-info">
				<div class="theme-mode">{THEME_MODE_LABEL[theme.mode]}</div>
				{#if theme.mode === 'auto'}
					<div class="theme-resolved">当前{theme.resolved === 'dark' ? '暗色' : '亮色'}</div>
				{/if}
			</div>
		</div>
	</div>
</aside>

<!-- 移动端：底部标签栏 -->
<nav class="tabbar">
	{#each items as it}
		<a class="tab" class:active={isActive(it.href)} href={it.href}>
			<span class="tab-icon"><Icon name={it.icon} size={20} /></span>
			<span class="tab-label">{it.label}</span>
		</a>
	{/each}
</nav>

<style>
	.sidebar {
		position: fixed;
		top: 0;
		left: 0;
		bottom: 0;
		width: 200px;
		background: var(--surface);
		border-right: 1px solid var(--border);
		padding: 22px 14px;
		display: flex;
		flex-direction: column;
		gap: 6px;
		z-index: 20;
	}

	.brand {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 4px 10px 18px;
	}
	.logo {
		display: flex;
		color: var(--green-600);
	}
	.brand-name {
		font-size: 20px;
		font-weight: 700;
		color: var(--green-700);
		letter-spacing: 2px;
	}

	.nav-item {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 11px 14px;
		border-radius: 12px;
		color: var(--text-secondary);
		font-weight: 500;
		transition: background 0.15s, color 0.15s;
	}
	.nav-item:hover {
		background: var(--green-50);
		color: var(--green-700);
	}
	.nav-item.active {
		background: var(--green-100);
		color: var(--green-700);
	}
	.nav-icon {
		display: flex;
	}

	.sidebar-foot {
		margin-top: auto;
		padding-top: 14px;
		border-top: 1px solid var(--border);
	}
	.theme-row {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 0 4px;
	}
	.theme-info {
		line-height: 1.3;
	}
	.theme-mode {
		font-size: 13px;
		color: var(--text-secondary);
		font-weight: 500;
	}
	.theme-resolved {
		font-size: 11px;
		color: var(--text-secondary);
		opacity: 0.75;
	}

	.tabbar {
		display: none;
	}

	@media (max-width: 760px) {
		.sidebar {
			display: none;
		}
		.tabbar {
			display: flex;
			position: fixed;
			bottom: 0;
			left: 0;
			right: 0;
			background: var(--surface);
			border-top: 1px solid var(--border);
			padding: 6px 4px calc(6px + env(safe-area-inset-bottom));
			justify-content: space-around;
			z-index: 20;
		}
		.tab {
			display: flex;
			flex-direction: column;
			align-items: center;
			gap: 2px;
			color: var(--text-secondary);
			font-size: 11px;
			padding: 4px 8px;
			border-radius: 10px;
			flex: 1;
		}
		.tab.active {
			color: var(--green-700);
		}
		.tab-icon {
			display: flex;
		}
	}
</style>
