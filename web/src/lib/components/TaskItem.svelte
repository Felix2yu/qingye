<script lang="ts">
	import type { Task } from '$lib/api';
	import { api } from '$lib/api';
	import { dueLabel, TASK_TYPE_EMOJI, TASK_TYPE_LABEL, formatDateTime } from '$lib/format';
	import { showToast } from '$lib/stores';

	let {
		task,
		onChange
	}: { task: Task; onChange?: () => void } = $props();

	let showHistory = $state(false);
	let logs = $state<{ id: number; action: 'done' | 'postpone'; at: string; note: string }[]>([]);
	let loadingLogs = $state(false);
	let busy = $state(false);

	const due = $derived(dueLabel(task.nextDue));

	async function openHistory() {
		showHistory = true;
		if (logs.length === 0) {
			loadingLogs = true;
			try {
				logs = await api.taskLogs(task.id);
			} catch (e) {
				showToast((e as Error).message, 'err');
			} finally {
				loadingLogs = false;
			}
		}
	}

	function closeHistory() {
		showHistory = false;
	}

	async function done() {
		if (busy) return;
		busy = true;
		try {
			await api.doneTask(task.id);
			showToast('已完成 🌿');
			onChange?.();
		} catch (e) {
			showToast((e as Error).message, 'err');
		} finally {
			busy = false;
		}
	}

	async function postpone() {
		if (busy) return;
		busy = true;
		try {
			await api.postponeTask(task.id, 1);
			showToast('已推迟 1 天');
			onChange?.();
		} catch (e) {
			showToast((e as Error).message, 'err');
		} finally {
			busy = false;
		}
	}

	const typeLabel = TASK_TYPE_LABEL[task.type] ?? task.type;
	const emoji = TASK_TYPE_EMOJI[task.type] ?? '🌿';
</script>

<div class="card task-item" class:overdue={due.overdue}>
	<div class="head">
		<div class="icon">{emoji}</div>
		<div class="main">
			<div class="title">{task.title || typeLabel}</div>
			<div class="sub">
				{task.plant?.name ?? '未知植物'} · <span class:due-soon={due.today} class:due-over={due.overdue}>{due.text}</span>
			</div>
		</div>
		<div class="actions">
			<button class="btn btn-primary btn-sm" onclick={done} disabled={busy}>✓ 完成</button>
			<button class="btn btn-ghost btn-sm" onclick={postpone} disabled={busy}>推迟</button>
			<button class="btn btn-ghost btn-sm" onclick={openHistory}>历史</button>
		</div>
	</div>
</div>

{#if showHistory}
	<div class="modal-backdrop" onclick={closeHistory}>
		<div class="modal hist-modal" onclick={(e) => e.stopPropagation()}>
			<div class="modal-title">
				{task.title || typeLabel} · 任务历史
				<button class="modal-close" onclick={closeHistory} aria-label="关闭">×</button>
			</div>
			<div class="hist-body">
				{#if loadingLogs}
					<p class="muted">加载中…</p>
				{:else if logs.length === 0}
					<p class="muted">暂无记录</p>
				{:else}
					{#each logs as log}
						<div class="log">
							<span class="log-action">{log.action === 'done' ? '✅ 完成' : '⏭️ 推迟'}</span>
							<span class="muted">{formatDateTime(log.at)}</span>
							{#if log.note}<span class="log-note">「{log.note}」</span>{/if}
						</div>
					{/each}
				{/if}
			</div>
		</div>
	</div>
{/if}

<style>
	.task-item {
		padding: 14px 16px;
	}
	.head {
		display: flex;
		align-items: flex-start;
		gap: 12px;
		flex-wrap: wrap;
	}
	.icon {
		width: 40px;
		height: 40px;
		border-radius: 12px;
		background: var(--green-50);
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 20px;
		flex-shrink: 0;
	}
	/* 标题区保底宽度，避免被右侧按钮挤成竖排 */
	.main {
		flex: 1 1 200px;
		min-width: 0;
	}
	.title {
		font-weight: 600;
		line-height: 1.4;
		overflow-wrap: anywhere;
	}
	.sub {
		font-size: 13px;
		color: var(--text-secondary);
		margin-top: 2px;
		line-height: 1.5;
	}
	.due-soon {
		color: var(--green-700);
		font-weight: 600;
	}
	.due-over {
		color: var(--danger);
		font-weight: 600;
	}
	/* 操作按钮：允许换行，主操作与辅助按钮视觉区分 */
	.actions {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
		flex-shrink: 0;
		margin-left: auto;
	}
	.actions .btn-primary {
		font-weight: 600;
	}
	/* 历史弹窗 */
	.modal-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(15, 23, 12, 0.45);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 100;
		padding: 20px;
		backdrop-filter: blur(2px);
	}
	.modal {
		background: var(--surface);
		border-radius: 16px;
		box-shadow: 0 12px 40px rgba(0, 0, 0, 0.18);
		width: min(480px, 100%);
		max-height: 80vh;
		display: flex;
		flex-direction: column;
		overflow: hidden;
		animation: pop 0.18s ease;
	}
	@keyframes pop {
		from { transform: translateY(8px); opacity: 0; }
		to { transform: translateY(0); opacity: 1; }
	}
	.modal-title {
		display: flex;
		align-items: center;
		justify-content: space-between;
		font-weight: 600;
		font-size: 15px;
		padding: 14px 18px;
		border-bottom: 1px solid var(--border);
	}
	.modal-close {
		border: none;
		background: none;
		font-size: 22px;
		line-height: 1;
		cursor: pointer;
		color: var(--text-secondary);
		padding: 0 4px;
	}
	.modal-close:hover { color: var(--text); }
	.hist-body {
		overflow-y: auto;
		padding: 12px 18px 16px;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.log {
		display: flex;
		gap: 10px;
		align-items: baseline;
		font-size: 13px;
		line-height: 1.5;
	}
	.log-note {
		color: var(--text);
	}
</style>
