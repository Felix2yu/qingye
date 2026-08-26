<script lang="ts">
	import '../app.css';
	import NavBar from '$lib/components/NavBar.svelte';
	import { toast } from '$lib/stores';
	import { theme } from '$lib/theme.svelte';
	import { fly } from 'svelte/transition';
	import { onMount } from 'svelte';

	let { children } = $props();

	onMount(() => theme.init());
</script>

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
	@media (max-width: 760px) {
		.toast {
			bottom: 80px;
		}
	}
</style>
