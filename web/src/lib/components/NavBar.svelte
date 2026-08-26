<script lang="ts">
	import { page } from '$app/stores';

	const items = [
		{ href: '/', label: '今日', icon: '🌿' },
		{ href: '/plants', label: '植物', icon: '🪴' },
		{ href: '/tasks', label: '任务', icon: '📋' },
		{ href: '/diary', label: '日记', icon: '📷' },
		{ href: '/library', label: '资料库', icon: '📚' },
		{ href: '/import', label: '导入', icon: '📥' },
		{ href: '/settings', label: '设置', icon: '⚙️' }
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
		<span class="logo">🌿</span>
		<span class="brand-name">青野</span>
	</div>
	<nav>
		{#each items as it}
			<a class="nav-item" class:active={isActive(it.href)} href={it.href}>
				<span class="nav-icon">{it.icon}</span>
				<span>{it.label}</span>
			</a>
		{/each}
	</nav>
</aside>

<!-- 移动端：底部标签栏 -->
<nav class="tabbar">
	{#each items as it}
		<a class="tab" class:active={isActive(it.href)} href={it.href}>
			<span class="tab-icon">{it.icon}</span>
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
		font-size: 26px;
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
		font-size: 18px;
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
			font-size: 20px;
		}
	}
</style>
