<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import type { Task, WeatherCurrent } from '$lib/api';
	import { greeting, formatDate } from '$lib/format';
	import TaskItem from '$lib/components/TaskItem.svelte';
	import { goto } from '$app/navigation';

	let today = $state<Task[]>([]);
	let upcoming = $state<Task[]>([]);
	let isWorkday = $state(true);
	let loading = $state(true);
	let error = $state('');
	let weather = $state<WeatherCurrent | null>(null);

	async function load() {
		loading = true;
		error = '';
		try {
			const [t, u] = await Promise.all([api.todayTasks(), api.upcomingTasks(3)]);
			today = t;
			// 临近任务排除已经出现在今日里的
			const ids = new Set(t.map((x) => x.id));
			upcoming = u.filter((x) => !ids.has(x.id));
			// 休息日判断：today 在休息日会返回空且不是因为没有任务
			isWorkday = !(u.length === 0 && t.length === 0) || true;
		} catch (e) {
			error = (e as Error).message;
		} finally {
			loading = false;
		}
		// 天气（失败不影响主流程）
		try {
			weather = await api.weatherCurrent();
		} catch {
			weather = null;
		}
	}

	onMount(load);

	function weatherIcon(icon: string, condition: string): string {
		if (!icon) {
			const c = condition || '';
			if (c.includes('雨')) return '🌧️';
			if (c.includes('阴')) return '☁️';
			if (c.includes('雪')) return '❄️';
			return '⛅';
		}
		// 和风 icon：1xx 晴/云，2xx 雨，3xx 雷雨，4xx 雪，5xx 雾
		const first = icon[0];
		if (first === '3') return '⛈️';
		if (first === '2') return '🌧️';
		if (first === '4') return '❄️';
		if (first === '5') return '🌫️';
		if (first === '1' && icon !== '100') return '⛅';
		if (first === '1') return '☀️';
		return '⛅';
	}
</script>

<svelte:head><title>青野 · 今日</title></svelte:head>

<div class="page">
	<div class="hero">
		<h1 class="page-title">{greeting()}，{greeting() === '夜深了' ? '辛苦了' : '照顾植物了吗？'}</h1>
		<p class="page-sub">{formatDate(new Date())}</p>
	</div>

	<div class="quick">
		<button class="btn btn-primary" onclick={() => goto('/plants')}>＋ 添加植物</button>
		<button class="btn btn-soft" onclick={() => goto('/tasks')}>＋ 添加任务</button>
		<button class="btn btn-ghost" onclick={() => goto('/diary')}>📷 记一笔</button>
	</div>

	{#if weather?.current}
		<div class="card weather-card">
			<div class="w-left">
				<span class="w-icon">{weatherIcon(weather.current.icon, weather.current.condition)}</span>
				<div>
					<div class="w-temp">{Math.round(weather.current.temp)}℃</div>
					<div class="w-cond">{weather.current.condition}{weather.config.city ? ` · ${weather.config.city}` : ''}</div>
				</div>
			</div>
			{#if weather.config.enabled}
				<div class="w-strategy">智能养护已开启</div>
			{/if}
		</div>
	{/if}

	{#if loading}
		<p class="empty"><span class="emoji">🌿</span>加载中…</p>
	{:else if error}
		<p class="empty"><span class="emoji">⚠️</span>{error}</p>
	{:else}
		<h2 class="section-title">今日任务</h2>
		{#if today.length === 0}
			<div class="card empty-card">
				<span class="emoji">🍃</span>
				今天是休息日，没有待办养护任务。享受悠闲时光吧！
			</div>
		{:else}
			<div class="grid grid-tasks">
				{#each today as t (t.id)}
					<TaskItem task={t} onChange={load} />
				{/each}
			</div>
		{/if}

		{#if upcoming.length > 0}
			<h2 class="section-title">临近提醒</h2>
			<div class="grid grid-tasks">
				{#each upcoming as t (t.id)}
					<TaskItem task={t} onChange={load} />
				{/each}
			</div>
		{/if}
	{/if}
</div>

<style>
	.hero {
		margin-bottom: 18px;
	}
	.quick {
		display: flex;
		gap: 10px;
		flex-wrap: wrap;
		margin-bottom: 8px;
	}
	.empty-card {
		text-align: center;
		color: var(--text-secondary);
		padding: 36px 20px;
	}
	.empty-card .emoji {
		display: block;
		font-size: 36px;
		margin-bottom: 8px;
	}
	.weather-card {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 14px 18px;
		margin-bottom: 16px;
		background: linear-gradient(135deg, var(--green-50), var(--surface));
	}
	.w-left {
		display: flex;
		align-items: center;
		gap: 12px;
	}
	.w-icon {
		font-size: 34px;
	}
	.w-temp {
		font-size: 20px;
		font-weight: 700;
		line-height: 1.2;
	}
	.w-cond {
		font-size: 13px;
		color: var(--text-secondary);
	}
	.w-strategy {
		font-size: 12px;
		color: var(--green-700);
		background: var(--accent-soft);
		padding: 4px 10px;
		border-radius: 999px;
	}
</style>
