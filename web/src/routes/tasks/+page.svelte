<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import type { Plant, Task } from '$lib/api';
	import { showToast } from '$lib/stores';
	import TaskItem from '$lib/components/TaskItem.svelte';
	import { TASK_TYPE_LABEL, TASK_TYPES } from '$lib/format';
	import Icon from '$lib/components/Icon.svelte';
	import { goto } from '$app/navigation';

	let tasks = $state<Task[]>([]);
	let plants = $state<Plant[]>([]);
	let loading = $state(true);

	let typeFilter = $state('');
	let hideDone = $state(true);

	let showForm = $state(false);
	let fPlant = $state<number>(0);
	let fType = $state('water');
	let fTitle = $state('');
	let fInterval = $state(7);

	const typeOptions = [
		{ value: '', label: '全部' },
		{ value: 'water', label: '💧 浇水' },
		{ value: 'fertilize', label: '🌱 施肥' },
		{ value: 'mist', label: '🌫️ 喷雾' },
		{ value: 'repot', label: '🪴 换盆' },
		{ value: 'prune', label: '✂️ 修剪' },
		{ value: 'clean', label: '🧹 清理' },
		{ value: 'pesticide', label: '🐛 除虫' },
		{ value: 'other', label: '✨ 其他' }
	];

	async function load() {
		loading = true;
		try {
			const [t, p] = await Promise.all([
				api.listTasks({ includeDone: !hideDone }),
				api.listPlants()
			]);
			tasks = t;
			plants = p;
			if (fPlant === 0 && p.length > 0) fPlant = p[0].id;
		} catch (e) {
			showToast((e as Error).message, 'err');
		} finally {
			loading = false;
		}
	}

	let filtered = $derived(
		typeFilter ? tasks.filter((t) => t.type === typeFilter) : tasks
	);

	async function refresh() {
		await load();
	}

	async function submitTask() {
		try {
			await api.createTask({
				plantId: fPlant,
				type: fType,
				title: fTitle.trim() || TASK_TYPE_LABEL[fType],
				intervalDays: Number(fInterval) || 7
			});
			showToast('已添加任务');
			showForm = false;
			fTitle = '';
			await load();
		} catch (e) {
			showToast((e as Error).message, 'err');
		}
	}

	onMount(load);
</script>

<svelte:head><title>青野集 · 任务</title></svelte:head>

<div class="page">
	<div class="head">
		<div>
			<h1 class="page-title">任务清单</h1>
			<p class="page-sub">共 {tasks.length} 个任务</p>
		</div>
		<button class="btn btn-primary btn-sm" onclick={() => (showForm = true)}>＋ 添加任务</button>
	</div>

	<div class="toolbar">
		<div class="chips">
			{#each typeOptions as o}
				<button class="chip" class:active={typeFilter === o.value} onclick={() => (typeFilter = o.value)}>
					{o.label}
				</button>
			{/each}
		</div>
		<label class="toggle">
			<input type="checkbox" bind:checked={hideDone} onchange={refresh} />
			<span>隐藏已停用</span>
		</label>
	</div>

	{#if loading}
		<p class="empty"><span class="emoji">🌿</span>加载中…</p>
	{:else if filtered.length === 0}
		<p class="empty"><span class="emoji">📋</span>没有符合条件的任务</p>
	{:else}
		<div class="grid grid-tasks">
			{#each filtered as t (t.id)}
				<TaskItem task={t} onChange={refresh} />
			{/each}
		</div>
	{/if}
</div>

{#if showForm}
	<div
		class="modal-backdrop"
		role="button"
		tabindex="-1"
		onclick={(e) => {
			if (e.target === e.currentTarget) showForm = false;
		}}
		onkeydown={(e) => {
			if (e.key === 'Escape') showForm = false;
		}}
	>
		<div class="modal">
			<div class="modal-title">添加任务</div>
			<div class="form-field">
				<label for="task-plant">植物 *</label>
				<select id="task-plant" bind:value={fPlant}>
					{#each plants as p}<option value={p.id}>{p.name}</option>{/each}
				</select>
			</div>
			<div class="form-field">
				<div class="field-label" id="task-type-cap">类型</div>
				<div class="type-grid" role="group" aria-labelledby="task-type-cap">
					{#each TASK_TYPES as t}
						<button
							type="button"
							class="type-tile"
							class:selected={fType === t.value}
							onclick={() => (fType = t.value)}
						>
							<Icon name={t.icon} size={24} />
							<span class="type-label">{t.label}</span>
						</button>
					{/each}
				</div>
			</div>
			<div class="form-field"><label for="task-title">标题（可选）</label><input id="task-title" bind:value={fTitle} placeholder="留空用类型名" /></div>
			<div class="form-field"><label for="task-interval">周期（天）</label><input id="task-interval" type="number" bind:value={fInterval} /></div>
			<div class="modal-actions">
				<button class="btn btn-ghost" onclick={() => (showForm = false)}>取消</button>
				<button class="btn btn-primary" onclick={submitTask}>保存</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.head {
		display: flex;
		justify-content: space-between;
		align-items: flex-end;
		gap: 12px;
		flex-wrap: wrap;
	}
	.toolbar {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 12px;
		flex-wrap: wrap;
		margin-bottom: 8px;
	}
	.toolbar .chips {
		margin-bottom: 0;
	}
	.toggle {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		color: var(--text-secondary);
		font-size: 13px;
		cursor: pointer;
	}
	.toggle input {
		width: auto;
	}
	.type-grid {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 8px;
	}
	.type-tile {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 4px;
		padding: 10px 4px;
		border: 1px solid var(--border);
		border-radius: 12px;
		background: var(--bg);
		color: var(--text);
		cursor: pointer;
		font-size: 13px;
		transition: border-color 0.15s, background 0.15s, transform 0.05s;
	}
	.type-tile:hover {
		border-color: var(--green-500);
	}
	.type-tile:active {
		transform: scale(0.97);
	}
	.type-tile.selected {
		border-color: var(--green-500);
		background: var(--green-50);
		color: var(--green-700);
		font-weight: 600;
	}
	.type-tile :global(svg) {
		width: 24px;
		height: 24px;
	}
	.type-label {
		line-height: 1.1;
	}
</style>
