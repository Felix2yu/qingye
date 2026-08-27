<script lang="ts">
	import '../app.css';
	import NavBar from '$lib/components/NavBar.svelte';
	import { toast } from '$lib/stores';
	import { theme } from '$lib/theme.svelte';
	import { initOfflineSync } from '$lib/offline';
	import { browser } from '$app/environment';
	import { fly } from 'svelte/transition';
	import { onMount } from 'svelte';

	let { children } = $props();

	let online = $state(browser ? navigator.onLine : true);

	onMount(() => {
		theme.init();
		initOfflineSync();
		const on = () => (online = true);
		const off = () => (online = false);
		window.addEventListener('online', on);
		window.addEventListener('offline', off);
		return () => {
			window.removeEventListener('online', on);
			window.removeEventListener('offline', off);
		};
	});
</script>

{#if !online}
	<div class="offline-banner">⚡ 离线模式 · 改动将自动保存，联网后同步</div>
{/if}

<NavBar />

<main class="content">
	{@render children()}
</main>

{#if $toast}
	<div class="toast {$toast.type}" transition:fly={{ y: 20, duration: 200 }}>
		{$toast.text}
	</div>
{/if}

<style>
	.content {
		margin-left: 200px;
		min-height: 100vh;
	}
	@media (max-width: 760px) {
		.content {
			margin-left: 0;
		}
	}
	.toast {
		position: fixed;
		bottom: 90px;
		left: 50%;
		transform: translateX(-50%);
		background: var(--toast-bg);
		color: #fff;
		padding: 10px 20px;
		border-radius: 999px;
		font-size: 14px;
		z-index: 100;
		box-shadow: 0 8px 24px rgba(0, 0, 0, 0.2);
	}
	.toast.err {
		background: var(--danger);
	}
	.toast.info {
		background: #2563eb;
	}
	@media (max-width: 760px) {
		.toast {
			bottom: 80px;
		}
	}
	.offline-banner {
		position: fixed;
		top: 0;
		right: 0;
		left: 200px;
		z-index: 200;
		background: #f59e0b;
		color: #1f2937;
		text-align: center;
		font-size: 13px;
		font-weight: 600;
		padding: 6px 12px;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
	}
	@media (max-width: 760px) {
		.offline-banner {
			left: 0;
		}
	}
</style>
