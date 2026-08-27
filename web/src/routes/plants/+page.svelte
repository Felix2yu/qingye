<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import type { Plant, Room, Task, OnlineCandidate, PlantLibrary } from '$lib/api';
	import { showToast } from '$lib/stores';
	import { imgUrl } from '$lib/api';
	import Icon from '$lib/components/Icon.svelte';
	import PlantCard from '$lib/components/PlantCard.svelte';

	let plants = $state<Plant[]>([]);
	let rooms = $state<Room[]>([]);
	let tasks = $state<Task[]>([]);
	let loading = $state(true);

	let roomFilter = $state<number>(0); // 0 = 全部
	let showForm = $state(false);

	// 表单
	let fName = $state('');
	let fSpecies = $state('');
	let fRoom = $state<number>(0);
	let fNote = $state('');
	let fPhoto = $state('');
	let fLocation = $state('');
	let fLightReq = $state('');
	let fAcquired = $state('');
	let fAttributes = $state('');

	let rName = $state('');
	let rOutdoor = $state(false);

	// 房间管理（编辑副本：拖拽/箭头排序，统一保存）
	let showRoomMgr = $state(false);
	let editRooms = $state<Room[]>([]);
	let dragIndex = $state(-1);
	let dragEl: HTMLElement | null = null;

	function openRoomMgr() {
		editRooms = rooms.map((r) => ({ ...r }));
		showRoomMgr = true;
	}

	function moveRow(i: number, dir: -1 | 1) {
		const j = i + dir;
		if (j < 0 || j >= editRooms.length) return;
		const arr = [...editRooms];
		[arr[i], arr[j]] = [arr[j], arr[i]];
		editRooms = arr;
	}

	function dragStart(e: DragEvent, i: number) {
		dragIndex = i;
		if (e.dataTransfer) {
			e.dataTransfer.effectAllowed = 'move';
			e.dataTransfer.setData('text/plain', String(i));
		}
	}

	function dragOver(e: DragEvent, i: number) {
		e.preventDefault();
		if (e.dataTransfer) e.dataTransfer.dropEffect = 'move';
		if (dragIndex === -1 || dragIndex === i) return;
		const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
		const after = e.clientY > rect.top + rect.height / 2;
		const to = after ? Math.min(i + 1, editRooms.length) : i;
		const arr = [...editRooms];
		const [m] = arr.splice(dragIndex, 1);
		arr.splice(to > dragIndex ? to - 1 : to, 0, m);
		editRooms = arr;
		dragIndex = arr.indexOf(m);
	}

	// 拖柄按下时才让整行可拖拽，避免干扰输入框聚焦与文本选择
	function handleDown(e: PointerEvent) {
		dragEl = (e.currentTarget as HTMLElement).closest('.room-row');
		if (dragEl) dragEl.draggable = true;
	}
	function releaseDrag() {
		if (dragEl) dragEl.draggable = false;
		dragEl = null;
	}

	async function saveRooms() {
		for (const r of editRooms) {
			if (!r.name.trim()) {
				showToast('房间名称不能为空', 'err');
				return;
			}
		}
		try {
			// 按当前列表顺序写入 sort
			for (const [idx, r] of editRooms.entries()) {
				await api.updateRoom(r.id, r.name.trim(), idx, r.isOutdoor);
			}
			showToast('房间已保存');
			await load();
			editRooms = rooms.map((r) => ({ ...r }));
		} catch (e) {
			showToast((e as Error).message, 'err');
			await load();
			editRooms = rooms.map((r) => ({ ...r }));
		}
	}

	async function removeRoom(r: Room) {
		const count = Number(r.count ?? 0);
		const tip =
			count > 0
				? `房间「${r.name}」下还有 ${count} 株植物，需先移出或删除才能删除房间。仍要尝试吗？`
				: `确定删除房间「${r.name}」？`;
		if (!confirm(tip)) return;
		try {
			await api.deleteRoom(r.id);
			showToast('已删除房间');
			if (roomFilter === r.id) roomFilter = 0;
			await load();
			editRooms = editRooms.filter((x) => x.id !== r.id);
		} catch (e) {
			showToast((e as Error).message, 'err');
		}
	}

	// 在线匹配（Plantbook）
	let onlineEnabled = $state(false);
	let onlineCands = $state<OnlineCandidate[]>([]);
	let onlineLoading = $state(false);
	let onlineMsg = $state('');
	let matchedGuide = $state<PlantLibrary | null>(null); // 选中后带回的指南

	async function toggleRoomOutdoor(r: Room) {
		try {
			await api.updateRoom(r.id, r.name, r.sort, !r.isOutdoor);
			showToast(!r.isOutdoor ? '已标记为室外' : '已标记为室内');
			await load();
		} catch (e) {
			showToast((e as Error).message, 'err');
		}
	}

	async function load() {
		loading = true;
		try {
			const [p, r, t] = await Promise.all([
				api.listPlants(),
				api.listRooms(),
				api.listTasks({ includeDone: false })
			]);
			plants = p;
			rooms = r;
			tasks = t;
			if (fRoom === 0 && r.length > 0) fRoom = r[0].id;
		} catch (e) {
			showToast((e as Error).message, 'err');
		} finally {
			loading = false;
		}
	}

	function pendingCount(plantId: number): number {
		return tasks.filter((t) => t.plantId === plantId && t.active).length;
	}

	// 计算每种房间待办数（用于角标展示，可选）
	let filtered = $derived(
		roomFilter === 0 ? plants : plants.filter((p) => p.roomId === roomFilter)
	);

	async function submitPlant() {
		if (!fName.trim()) {
			showToast('请填写植物名称', 'err');
			return;
		}
		try {
			await api.createPlant({
				name: fName.trim(),
				species: fSpecies.trim(),
				roomId: fRoom,
				note: fNote.trim(),
				photo: fPhoto.trim(),
				location: fLocation.trim(),
				lightReq: fLightReq.trim(),
				acquiredDate: fAcquired.trim(),
				attributes: fAttributes.trim()
			});
			showToast('已添加植物 🌱');
			showForm = false;
			fName = fSpecies = fNote = fPhoto = fLocation = fLightReq = fAcquired = fAttributes = '';
			await load();
		} catch (e) {
			showToast((e as Error).message, 'err');
		}
	}

	async function submitRoom() {
		if (!rName.trim()) {
			showToast('请填写房间名称', 'err');
			return;
		}
		try {
			// 新房间排在列表末尾
			await api.createRoom(rName.trim(), rooms.length, rOutdoor);
			showToast('已添加房间');
			rName = '';
			rOutdoor = false;
			await load();
			editRooms = rooms.map((r) => ({ ...r }));
		} catch (e) {
			showToast((e as Error).message, 'err');
		}
	}

	onMount(load);

	// 手动触发在线匹配（输入名称后点"在线查找"）
	async function searchOnline() {
		const kw = fName.trim();
		if (kw.length < 2) {
			showToast('请输入至少 2 个字再查找', 'err');
			return;
		}
		onlineLoading = true;
		onlineMsg = '';
		onlineCands = [];
		matchedGuide = null;
		try {
			const res = await api.searchLibraryOnline(kw);
			onlineEnabled = res.enabled;
			if (!res.enabled) {
				onlineMsg = '未配置 Plantbook 凭据，在线匹配已禁用';
				return;
			}
			onlineCands = res.list;
			if (res.list.length === 0) {
				onlineMsg =
					[...kw].length < 3
						? '未找到匹配：常见中文名可直接查找，其他词请至少输入 3 个字符'
						: '在线库未找到匹配，可手动填写';
			}
		} catch (e) {
			onlineMsg = (e as Error).message;
		} finally {
			onlineLoading = false;
		}
	}

	// 选中候选 → 拉详情并写回本地资料库，带回中文指南
	async function pickOnline(c: OnlineCandidate) {
		onlineLoading = true;
		try {
			const lib = await api.importLibraryOnline(c.pid);
			matchedGuide = lib;
			fName = lib.name || c.name;
			fSpecies = c.alias;
			fNote = lib.guide;
			if (c.image && !fPhoto) fPhoto = c.image;
			showToast('已从在线库带入养护指南');
			onlineCands = [];
		} catch (e) {
			showToast((e as Error).message, 'err');
		} finally {
			onlineLoading = false;
		}
	}</script>

<svelte:head><title>青野集 · 植物</title></svelte:head>

<div class="page">
	<div class="head">
		<div>
			<h1 class="page-title">我的植物</h1>
			<p class="page-sub">共 {plants.length} 株 · 按房间整理</p>
		</div>
		<div class="head-actions">
			<button class="btn btn-ghost btn-sm" onclick={openRoomMgr}>管理房间</button>
			<button class="btn btn-primary btn-sm" onclick={() => (showForm = true)}>＋ 植物</button>
		</div>
	</div>

	{#if rooms.length > 0}
		<div class="chips">
			<button class="chip" class:active={roomFilter === 0} onclick={() => (roomFilter = 0)}>全部</button>
			{#each rooms as r}
				<button class="chip" class:active={roomFilter === r.id} onclick={() => (roomFilter = r.id)} title={r.isOutdoor ? '室外 · 点击切换' : '室内 · 点击切换'}>
					{r.name}{r.count ? ` · ${r.count}` : ''}
					{#if r.isOutdoor}<span class="outdoor-dot" title="室外">☀️</span>{/if}
				</button>
			{/each}
		</div>
	{/if}

	{#if loading}
		<p class="empty"><span class="emoji">🌿</span>加载中…</p>
	{:else if filtered.length === 0}
		<p class="empty"><span class="emoji">🪴</span>这里还没有植物，点右上角添加一株吧</p>
	{:else}
		<div class="grid grid-cards">
			{#each filtered as p (p.id)}
				<PlantCard plant={p} pending={pendingCount(p.id)} />
			{/each}
		</div>
	{/if}
</div>

<!-- 新增植物 -->
{#if showForm}
	<div class="modal-backdrop" onclick={() => (showForm = false)}>
		<div class="modal" onclick={(e) => e.stopPropagation()}>
			<div class="modal-title">添加植物</div>
			<div class="form-field">
				<label for="">名称 *</label>
				<div class="name-row">
					<input bind:value={fName} placeholder="如：绿萝" />
					<button class="btn btn-ghost btn-sm" type="button" onclick={searchOnline} disabled={onlineLoading}>
						{onlineLoading ? '查找中…' : '在线查找'}
					</button>
				</div>
				{#if onlineMsg}<p class="hint">{onlineMsg}</p>{/if}
				{#if onlineCands.length > 0}
					<div class="online-list">
						{#each onlineCands as c (c.pid)}
							<button class="online-item" type="button" onclick={() => pickOnline(c)}>
								{#if c.image}<img src={imgUrl(c.image)} alt="" class="online-thumb" />{/if}
								<span class="online-name">{c.name}</span>
								<span class="online-alias">{c.alias}</span>
							</button>
						{/each}
					</div>
				{/if}
				{#if matchedGuide}
					<div class="matched-guide">
						<span class="badge">✓ 已带入在线指南</span>
						<pre>{matchedGuide.guide}</pre>
					</div>
				{/if}
			</div>
			<div class="form-field">
				<label for="">品种</label>
				<input bind:value={fSpecies} placeholder="如：Epipremnum aureum" />
			</div>
			<div class="form-field">
				<label for="">所在房间 / 分组</label>
				<select bind:value={fRoom}>
					{#each rooms as r}<option value={r.id}>{r.name}</option>{/each}
				</select>
			</div>
			<div class="form-field">
				<label for="">详细方位</label>
				<input bind:value={fLocation} placeholder="如：阳台左侧" />
			</div>
			<div class="form-field">
				<label for="">光照需求</label>
				<input bind:value={fLightReq} placeholder="如：散射光 / 耐阴" />
			</div>
			<div class="form-field">
				<label for="">获得日期</label>
				<input bind:value={fAcquired} type="date" />
			</div>
			<div class="form-field">
				<label for="">照片链接（可选）</label>
				<input bind:value={fPhoto} placeholder="https://… 或留空用首字占位" />
			</div>
			<div class="form-field">
				<label for="">备注</label>
				<textarea bind:value={fNote} rows="2" placeholder="养护小贴士…"></textarea>
			</div>
			<div class="form-field">
				<label for="">扩展属性 (JSON)</label>
				<textarea bind:value={fAttributes} rows="2" placeholder={'{"土壤":"泥炭土"}'}></textarea>
			</div>
			<div class="modal-actions">
				<button class="btn btn-ghost" onclick={() => (showForm = false)}>取消</button>
				<button class="btn btn-primary" onclick={submitPlant}>保存</button>
			</div>
		</div>
	</div>
{/if}

<!-- 房间管理：拖拽排序 / 改名 / 室内外 / 删除 / 新增 -->
{#if showRoomMgr}
	<div class="modal-backdrop" onclick={() => (showRoomMgr = false)}>
		<div class="modal room-mgr" onclick={(e) => e.stopPropagation()}>
			<div class="modal-title">管理房间</div>
			<p class="muted mgr-hint">拖住 ⠿ 或用 ↑↓ 调整顺序，改名、切换室内/室外后点「保存修改」。</p>

			<div class="room-list">
				{#each editRooms as r, i (r.id)}
					<div
						class="room-row"
						class:dragging={dragIndex === i}
						ondragstart={(e) => dragStart(e, i)}
						ondragover={(e) => dragOver(e, i)}
						ondragend={() => {
							dragIndex = -1;
							releaseDrag();
						}}
						ondrop={(e) => e.preventDefault()}
					>
						<span
							class="drag-handle"
							title="拖动排序"
							onpointerdown={handleDown}
							onpointerup={releaseDrag}
							onpointercancel={releaseDrag}
							onpointerleave={releaseDrag}
						>
							<Icon name="grip" size={16} />
						</span>
						<span class="room-order">{i + 1}</span>
						<input class="room-name" bind:value={r.name} placeholder="名称" />
						<label class="room-outdoor" title="室外（阳台/花园，降雨时自动推迟浇水）">
							<input type="checkbox" bind:checked={r.isOutdoor} />
							<span>室外</span>
						</label>
						<button
							class="icon-btn"
							title="上移"
							disabled={i === 0}
							onclick={() => moveRow(i, -1)}
						>
							↑
						</button>
						<button
							class="icon-btn"
							title="下移"
							disabled={i === editRooms.length - 1}
							onclick={() => moveRow(i, 1)}
						>
							↓
						</button>
						<button class="btn btn-danger btn-sm room-del" onclick={() => removeRoom(r)}>删除</button>
					</div>
				{/each}
				{#if editRooms.length === 0}
					<p class="muted">还没有房间，在下方添加第一个吧</p>
				{/if}
			</div>

			<div class="room-add">
				<div class="room-add-title">添加房间</div>
				<div class="room-row">
					<span class="drag-handle ghost"></span>
					<span class="room-order add">＋</span>
					<input class="room-name" bind:value={rName} placeholder="如：书房 *" />
					<label class="room-outdoor" title="室外（阳台/花园，降雨时自动推迟浇水）">
						<input type="checkbox" bind:checked={rOutdoor} />
						<span>室外</span>
					</label>
					<button class="btn btn-primary btn-sm room-add-btn" onclick={submitRoom}>添加</button>
				</div>
			</div>

			<div class="modal-actions">
				<button class="btn btn-ghost" onclick={() => (showRoomMgr = false)}>关闭</button>
				<button class="btn btn-primary" onclick={saveRooms}>保存修改</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.head {
		display: flex;
		align-items: flex-end;
		justify-content: space-between;
		gap: 12px;
		flex-wrap: wrap;
		margin-bottom: 18px;
	}
	.head-actions {
		display: flex;
		gap: 8px;
		flex-shrink: 0;
	}
	.room-mgr {
		max-width: 560px;
	}
	.mgr-hint {
		margin: -10px 0 14px;
	}
	.room-list {
		display: flex;
		flex-direction: column;
		gap: 8px;
		margin-bottom: 18px;
	}
	.room-row {
		display: flex;
		align-items: center;
		gap: 8px;
		flex-wrap: wrap;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		padding: 8px 10px;
		background: var(--surface);
		transition: border-color 0.15s, opacity 0.15s;
	}
	.room-row.dragging {
		opacity: 0.45;
		border-style: dashed;
	}
	.drag-handle {
		display: flex;
		align-items: center;
		color: var(--text-secondary);
		cursor: grab;
		touch-action: none;
		flex-shrink: 0;
	}
	.drag-handle:active {
		cursor: grabbing;
	}
	.drag-handle.ghost {
		visibility: hidden;
	}
	.room-order {
		width: 20px;
		text-align: center;
		font-size: 12px;
		color: var(--text-secondary);
		flex-shrink: 0;
		font-variant-numeric: tabular-nums;
	}
	.room-name {
		flex: 1 1 110px;
		min-width: 0;
		padding: 7px 10px;
	}
	.icon-btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		border-radius: 8px;
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--text-secondary);
		font-size: 14px;
		line-height: 1;
		flex-shrink: 0;
		transition: color 0.15s, border-color 0.15s;
	}
	.icon-btn:hover:not(:disabled) {
		color: var(--green-700);
		border-color: var(--green-500);
	}
	.icon-btn:disabled {
		opacity: 0.35;
		cursor: default;
	}
	.room-outdoor {
		display: inline-flex;
		align-items: center;
		gap: 5px;
		font-size: 13px;
		color: var(--text-secondary);
		cursor: pointer;
		flex-shrink: 0;
	}
	.room-outdoor input {
		width: auto;
		margin: 0;
	}
	.room-del {
		flex-shrink: 0;
	}
	.room-add {
		border-top: 1px dashed var(--border);
		padding-top: 14px;
	}
	.room-add-title {
		font-size: 13px;
		font-weight: 600;
		color: var(--text-secondary);
		margin-bottom: 10px;
	}
	.room-add-btn {
		align-self: stretch;
	}
	@media (max-width: 480px) {
		.room-del {
			margin-left: auto;
		}
	}
</style>
