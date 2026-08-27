<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { api, imgUrl } from '$lib/api';
	import type { Plant, Task, PhotoDiary, CareLog } from '$lib/api';
	import { dueLabel, TASK_TYPES, TASK_TYPE_EMOJI, TASK_TYPE_LABEL, formatDate, formatDateTime, careTypeLabel, fmtDate } from '$lib/format';
	import Icon from '$lib/components/Icon.svelte';
	import { showToast } from '$lib/stores';
	import { goto } from '$app/navigation';

	const id = Number($page.params.id);

	let plant = $state<Plant | null>(null);
	let tasks = $state<Task[]>([]);
	let diaries = $state<PhotoDiary[]>([]);
	let cares = $state<CareLog[]>([]);
	let loading = $state(true);

	let editing = $state(false);
	let eName = $state('');
	let eSpecies = $state('');
	let eNote = $state('');
	let ePhoto = $state('');
	let eLocation = $state('');
	let eLightReq = $state('');
	let eAcquired = $state('');
	let eAttributes = $state('');

	// 新增任务
	let showTaskForm = $state(false);
	let tType = $state('water');
	let tTitle = $state('');
	let tInterval = $state(7);

	async function load() {
		loading = true;
		try {
			const [p, t, d, c] = await Promise.all([
				api.getPlant(id),
				api.listTasks({ plantId: id, includeDone: false }),
				api.listDiaries({ plantId: id, pageSize: 5 }),
				api.careLogs(id)
			]);
			plant = p;
			tasks = t;
			diaries = d.list;
			cares = c;
		} catch (e) {
			showToast((e as Error).message, 'err');
		} finally {
			loading = false;
		}
	}

	function openEdit() {
		if (!plant) return;
		eName = plant.name;
		eSpecies = plant.species;
		eNote = plant.note;
		ePhoto = plant.photo;
		eLocation = plant.location ?? '';
		eLightReq = plant.lightReq ?? '';
		eAcquired = plant.acquiredDate ? fmtDate(plant.acquiredDate) : '';
		eAttributes = plant.attributes ?? '';
		editing = true;
	}

	async function saveEdit() {
		if (!plant) return;
		try {
			await api.updatePlant(plant.id, {
				name: eName.trim(),
				species: eSpecies.trim(),
				note: eNote.trim(),
				photo: ePhoto.trim(),
				location: eLocation.trim(),
				lightReq: eLightReq.trim(),
				acquiredDate: eAcquired.trim(),
				attributes: eAttributes.trim()
			});
			showToast('已保存');
			editing = false;
			await load();
		} catch (e) {
			showToast((e as Error).message, 'err');
		}
	}

	async function remove() {
		if (!plant) return;
		if (!confirm(`确定删除「${plant.name}」？相关任务与日记也会一并删除。`)) return;
		try {
			await api.deletePlant(plant.id);
			showToast('已删除');
			goto('/plants');
		} catch (e) {
			showToast((e as Error).message, 'err');
		}
	}

	async function submitTask() {
		try {
			await api.createTask({
				plantId: id,
				type: tType,
				title: tTitle.trim() || TASK_TYPE_LABEL[tType],
				intervalDays: Number(tInterval) || 7
			});
			showToast('已添加任务');
			showTaskForm = false;
			tTitle = '';
			await load();
		} catch (e) {
			showToast((e as Error).message, 'err');
		}
	}

	async function doneTask(t: Task) {
		try {
			await api.doneTask(t.id);
			showToast('已完成 🌿');
			await load();
		} catch (e) {
			showToast((e as Error).message, 'err');
		}
	}

	// 手动记录一次养护（无任务来源）
	let showCareForm = $state(false);
	let cType = $state('water');
	let cTitle = $state('');
	let cNote = $state('');
	async function submitCare() {
		if (!plant) return;
		try {
			await api.recordCare(plant.id, cType, cTitle.trim(), cNote.trim());
			showToast('已记录养护 🌿');
			showCareForm = false;
			cTitle = cNote = '';
			await load();
		} catch (e) {
			showToast((e as Error).message, 'err');
		}
	}

	onMount(load);
</script>

<svelte:head><title>青野集 · 植物详情</title></svelte:head>

<div class="page">
	{#if loading}
		<p class="empty"><span class="emoji">🌿</span>加载中…</p>
	{:else if !plant}
		<p class="empty"><span class="emoji">🪴</span>植物不存在</p>
	{:else}
		<button class="back btn btn-ghost btn-sm" onclick={() => goto('/plants')}>← 返回</button>

		<div class="card hero-card">
			<div class="hero-thumb">
				{#if plant.photo}
					<img src={imgUrl(plant.photo)} alt={plant.name} />
				{:else}
					<span class="placeholder">{plant.name.charAt(0)}</span>
				{/if}
			</div>
			<div class="hero-info">
				<h1 class="page-title">{plant.name}</h1>
				<p class="page-sub">{plant.species || '暂无品种信息'}{plant.room ? ` · ${plant.room.name}` : ''}</p>
				{#if plant.note}<p class="note">{plant.note}</p>{/if}
				<div class="meta">
					{#if plant.location}<span>📍 {plant.location}</span>{/if}
					{#if plant.room}<span><Icon name={plant.room.icon || 'house'} size={14} /> {plant.room.name}</span>{/if}
					{#if plant.lightReq}<span>☀️ {plant.lightReq}</span>{/if}
					{#if plant.acquiredDate}<span>🗓️ {fmtDate(plant.acquiredDate)}</span>{/if}
				</div>
				{#if plant.attributes}<p class="attrs">{plant.attributes}</p>{/if}
				<div class="hero-actions">
					<button class="btn btn-soft btn-sm" onclick={openEdit}>编辑</button>
					<button class="btn btn-danger btn-sm" onclick={remove}>删除</button>
				</div>
			</div>
		</div>

		<h2 class="section-title">养护任务（{tasks.length}）</h2>
		<div class="task-actions">
			<button class="btn btn-primary btn-sm" onclick={() => (showTaskForm = true)}>＋ 添加任务</button>
		</div>
		{#if tasks.length === 0}
			<p class="empty">暂无任务，添加后会在到期日出现在「今日」</p>
		{:else}
			<div class="grid grid-tasks">
				{#each tasks as t (t.id)}
					<div class="card task-row">
						<span class="ti">{TASK_TYPE_EMOJI[t.type] ?? '🌿'}</span>
						<div class="t-main">
							<div class="t-title">{t.title}</div>
							<div class="muted">每 {t.intervalDays} 天 · {dueLabel(t.nextDue).text}</div>
						</div>
						<button class="btn btn-soft btn-sm" onclick={() => doneTask(t)}>完成</button>
					</div>
				{/each}
			</div>
		{/if}

		<h2 class="section-title">养护时间线（{cares.length}）</h2>
		{#if cares.length === 0}
			<p class="empty">暂无养护记录，完成任务或手动记录后会出现在这里</p>
		{:else}
			<div class="timeline">
				{#each cares as c (c.id)}
					<div class="tl-item">
						<span class="tl-dot">{TASK_TYPE_EMOJI[c.type] ?? '🌿'}</span>
						<div class="tl-body">
							<div class="tl-title">{c.title || careTypeLabel(c.type)}</div>
							<div class="muted">{formatDateTime(c.at)}{c.source === 'manual' ? ' · 手动记录' : ' · 来自任务'}</div>
							{#if c.note}<div class="tl-note">「{c.note}」</div>{/if}
						</div>
					</div>
				{/each}
			</div>
		{/if}

		<div class="care-actions">
			<button class="btn btn-ghost btn-sm" onclick={() => (showCareForm = !showCareForm)}>
				＋ 手动记录养护
			</button>
		</div>
		{#if showCareForm}
			<div class="card care-form">
			<div class="form-field">
				<label for="">类型</label>
				<div class="type-grid">
					{#each TASK_TYPES as t}
						<button
							type="button"
							class="type-tile"
							class:selected={cType === t.value}
							onclick={() => (cType = t.value)}
						>
							<Icon name={t.icon} size={24} />
							<span class="type-label">{t.label}</span>
						</button>
					{/each}
				</div>
			</div>
				<div class="form-field"><label for="">标题（可选）</label><input bind:value={cTitle} placeholder="留空用类型名" /></div>
				<div class="form-field"><label for="">备注</label><textarea bind:value={cNote} rows="2"></textarea></div>
				<div class="modal-actions">
					<button class="btn btn-ghost" onclick={() => (showCareForm = false)}>取消</button>
					<button class="btn btn-primary" onclick={submitCare}>记录</button>
				</div>
			</div>
		{/if}

		<h2 class="section-title">照片日记</h2>
		{#if diaries.length === 0}
			<p class="empty">还没有日记</p>
		{:else}
			<div class="diary-grid">
				{#each diaries as d (d.id)}
					<div class="card diary-mini">
						{#if d.image}
							<img src={imgUrl(d.image)} alt={d.caption} />
						{:else}
							<div class="no-img">🪴</div>
						{/if}
						<div class="caption">{d.caption}</div>
						<div class="muted">{formatDate(d.takenAt)}</div>
					</div>
				{/each}
			</div>
		{/if}
		<button class="btn btn-ghost" onclick={() => goto(`/diary?plantId=${plant!.id}`)}>查看全部日记 →</button>
	{/if}
</div>

<!-- 编辑弹窗 -->
{#if editing}
	<div
		class="modal-backdrop"
		role="button"
		tabindex="-1"
		onclick={(e) => {
			if (e.target === e.currentTarget) editing = false;
		}}
		onkeydown={(e) => {
			if (e.key === 'Escape') editing = false;
		}}
	>
		<div class="modal">
			<div class="modal-title">编辑植物</div>
			<div class="form-field"><label for="edit-name">名称</label><input id="edit-name" bind:value={eName} /></div>
			<div class="form-field"><label for="edit-species">品种</label><input id="edit-species" bind:value={eSpecies} /></div>
			<div class="form-field"><label for="edit-location">详细方位</label><input id="edit-location" bind:value={eLocation} placeholder="如：阳台左侧" /></div>
			<div class="form-field"><label for="edit-light">光照需求</label><input id="edit-light" bind:value={eLightReq} placeholder="如：散射光 / 耐阴" /></div>
			<div class="form-field"><label for="edit-acquired">获得日期</label><input id="edit-acquired" bind:value={eAcquired} type="date" /></div>
			<div class="form-field"><label for="edit-photo">照片链接</label><input id="edit-photo" bind:value={ePhoto} /></div>
			<div class="form-field"><label for="edit-note">备注</label><textarea id="edit-note" bind:value={eNote} rows="2"></textarea></div>
			<div class="form-field"><label for="edit-attributes">扩展属性(JSON)</label><textarea id="edit-attributes" bind:value={eAttributes} rows="2" placeholder={'{"土壤":"泥炭土"}'}></textarea></div>
			<div class="modal-actions">
				<button class="btn btn-ghost" onclick={() => (editing = false)}>取消</button>
				<button class="btn btn-primary" onclick={saveEdit}>保存</button>
			</div>
		</div>
	</div>
{/if}

<!-- 新增任务弹窗 -->
{#if showTaskForm}
	<div
		class="modal-backdrop"
		role="button"
		tabindex="-1"
		onclick={(e) => {
			if (e.target === e.currentTarget) showTaskForm = false;
		}}
		onkeydown={(e) => {
			if (e.key === 'Escape') showTaskForm = false;
		}}
	>
		<div class="modal">
			<div class="modal-title">添加养护任务</div>
			<div class="form-field">
				<div class="field-label" id="ptype-cap">类型</div>
				<div class="type-grid" role="group" aria-labelledby="ptype-cap">
					{#each TASK_TYPES as t}
						<button
							type="button"
							class="type-tile"
							class:selected={tType === t.value}
							onclick={() => (tType = t.value)}
						>
							<Icon name={t.icon} size={24} />
							<span class="type-label">{t.label}</span>
						</button>
					{/each}
				</div>
			</div>
			<div class="form-field"><label for="ptask-title">标题（可选）</label><input id="ptask-title" bind:value={tTitle} placeholder="留空用类型名" /></div>
			<div class="form-field"><label for="ptask-interval">周期（天）</label><input id="ptask-interval" type="number" bind:value={tInterval} /></div>
			<div class="modal-actions">
				<button class="btn btn-ghost" onclick={() => (showTaskForm = false)}>取消</button>
				<button class="btn btn-primary" onclick={submitTask}>保存</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.back {
		margin-bottom: 14px;
	}
	.hero-card {
		display: flex;
		gap: 18px;
		padding: 18px;
		margin-bottom: 8px;
	}
	.hero-thumb {
		width: 120px;
		height: 120px;
		border-radius: 14px;
		background: linear-gradient(135deg, var(--green-100), var(--green-50));
		display: flex;
		align-items: center;
		justify-content: center;
		overflow: hidden;
		flex-shrink: 0;
	}
	.hero-thumb img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.placeholder {
		font-size: 54px;
		font-weight: 700;
		color: var(--green-500);
	}
	.hero-info {
		flex: 1;
	}
	.note {
		color: var(--text-secondary);
		font-size: 14px;
		margin-bottom: 12px;
	}
	.meta {
		display: flex;
		flex-wrap: wrap;
		gap: 8px 14px;
		font-size: 13px;
		color: var(--text-secondary);
		margin-bottom: 8px;
	}
	.attrs {
		font-size: 13px;
		color: var(--green-700);
		background: var(--green-50);
		padding: 6px 10px;
		border-radius: 8px;
		margin-bottom: 12px;
		white-space: pre-wrap;
	}
	.timeline {
		display: flex;
		flex-direction: column;
		gap: 2px;
		margin-bottom: 16px;
	}
	.tl-item {
		display: flex;
		gap: 12px;
		padding: 8px 0;
		border-bottom: 1px dashed var(--border);
	}
	.tl-dot {
		width: 34px;
		height: 34px;
		border-radius: 10px;
		background: var(--green-50);
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 18px;
		flex-shrink: 0;
	}
	.tl-body {
		flex: 1;
	}
	.tl-title {
		font-weight: 600;
	}
	.tl-note {
		font-size: 13px;
		color: var(--text);
		margin-top: 2px;
	}
	.care-actions {
		margin-bottom: 12px;
	}
	.care-form {
		padding: 16px;
		margin-bottom: 16px;
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
	.hero-actions {
		display: flex;
		gap: 8px;
	}
	.task-actions {
		margin-bottom: 12px;
	}
	.task-row {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 12px 14px;
	}
	.ti {
		font-size: 20px;
	}
	.t-main {
		flex: 1;
	}
	.t-title {
		font-weight: 600;
	}
	.diary-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
		gap: 12px;
		margin-bottom: 14px;
	}
	.diary-mini {
		overflow: hidden;
	}
	.diary-mini img,
	.no-img {
		width: 100%;
		aspect-ratio: 1;
		object-fit: cover;
		background: var(--green-50);
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 34px;
	}
	.diary-mini .caption {
		padding: 8px 10px 0;
		font-size: 13px;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.diary-mini .muted {
		padding: 2px 10px 10px;
	}
	@media (max-width: 520px) {
		.hero-card {
			flex-direction: column;
			align-items: center;
			text-align: center;
		}
	}
</style>
