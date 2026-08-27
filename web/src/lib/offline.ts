// 离线写入队列（Outbox）+ 自动同步
// 断网时写操作（POST/PUT/DELETE）会被序列化存入 IndexedDB，
// 恢复联网后按顺序回放，并刷新页面数据。读操作（GET）走 vite-pwa 的运行时缓存。

import { invalidateAll } from '$app/navigation';
import { browser } from '$app/environment';
import { showToast } from './stores';

const DB_NAME = 'qingye-offline';
const STORE = 'outbox';
const DB_VERSION = 1;

export interface FormEntry {
	key: string;
	kind: 'string' | 'blob';
	value: string | Blob;
}

export interface QueuedRequest {
	id?: number;
	method: string;
	url: string;
	headers: Record<string, string>;
	bodyKind: 'json' | 'form';
	body: string | FormEntry[];
	createdAt: number;
}

type StoredRequest = Omit<QueuedRequest, 'id' | 'createdAt'>;

// ---------- IndexedDB 基础封装 ----------
function openDB(): Promise<IDBDatabase> {
	return new Promise((resolve, reject) => {
		const req = indexedDB.open(DB_NAME, DB_VERSION);
		req.onupgradeneeded = () => {
			const db = req.result;
			if (!db.objectStoreNames.contains(STORE)) {
				db.createObjectStore(STORE, { keyPath: 'id', autoIncrement: true });
			}
		};
		req.onsuccess = () => resolve(req.result);
		req.onerror = () => reject(req.error);
	});
}

async function runTx(mode: IDBTransactionMode, fn: (store: IDBObjectStore) => void): Promise<void> {
	const db = await openDB();
	try {
		await new Promise<void>((resolve, reject) => {
			const t = db.transaction(STORE, mode);
			t.oncomplete = () => resolve();
			t.onerror = () => reject(t.error);
			t.onabort = () => reject(t.error);
			fn(t.objectStore(STORE));
		});
	} finally {
		db.close();
	}
}

export async function enqueue(req: StoredRequest): Promise<void> {
	if (!browser) return;
	await runTx('readwrite', (store) => {
		store.add({ ...req, createdAt: Date.now() } as QueuedRequest);
	});
}

export async function getAllOutbox(): Promise<QueuedRequest[]> {
	if (!browser) return [];
	const db = await openDB();
	try {
		return await new Promise((resolve, reject) => {
			const t = db.transaction(STORE, 'readonly');
			const req = t.objectStore(STORE).getAll();
			req.onsuccess = () => resolve((req.result as QueuedRequest[]) ?? []);
			req.onerror = () => reject(req.error);
		});
	} finally {
		db.close();
	}
}

export async function removeOutbox(id: number): Promise<void> {
	if (!browser) return;
	const db = await openDB();
	try {
		await new Promise<void>((resolve, reject) => {
			const t = db.transaction(STORE, 'readwrite');
			const req = t.objectStore(STORE).delete(id);
			req.onsuccess = () => resolve();
			req.onerror = () => reject(req.error);
		});
	} finally {
		db.close();
	}
}

export async function countOutbox(): Promise<number> {
	return (await getAllOutbox()).length;
}

// ---------- 请求序列化 / 反序列化 ----------
export function prepareRequest(
	method: string,
	url: string,
	init?: RequestInit
): StoredRequest {
	const headers: Record<string, string> = {};
	const isForm = init?.body instanceof FormData;
	if (!isForm) headers['Content-Type'] = 'application/json';
	if (init?.headers) {
		new Headers(init.headers).forEach((v, k) => (headers[k] = v));
	}

	if (isForm) {
		const entries: FormEntry[] = [];
		const fd = init!.body as FormData;
		for (const [k, v] of fd.entries()) {
			if (typeof v === 'string') entries.push({ key: k, kind: 'string', value: v });
			else entries.push({ key: k, kind: 'blob', value: v as Blob });
		}
		return { method, url, headers, bodyKind: 'form', body: entries };
	}

	const raw = init?.body ? String(init.body) : '';
	return { method, url, headers, bodyKind: 'json', body: raw };
}

function rebuild(init: QueuedRequest): RequestInit {
	if (init.bodyKind === 'form') {
		const fd = new FormData();
		for (const e of init.body as FormEntry[]) {
			if (e.kind === 'blob') fd.append(e.key, e.value as Blob);
			else fd.append(e.key, e.value as string);
		}
		return { method: init.method, body: fd, headers: init.headers };
	}
	return { method: init.method, body: init.body as string, headers: init.headers };
}

// ---------- 回放 + 同步 ----------
export async function flushOutbox(): Promise<void> {
	if (!browser || !navigator.onLine) return;
	const items = await getAllOutbox();
	if (!items.length) return;

	let ok = 0;
	let dropped = 0;
	let failed = 0;

	// 按创建时间顺序回放
	items.sort((a, b) => a.createdAt - b.createdAt);
	for (const it of items) {
		try {
			const res = await fetch(it.url, rebuild(it));
			if (res.ok) {
				await removeOutbox(it.id!);
				ok++;
			} else if (res.status >= 400 && res.status < 500) {
				// 客户端错误（如校验/冲突）不再重试，直接丢弃避免堆积
				await removeOutbox(it.id!);
				dropped++;
			} else {
				failed++;
			}
		} catch {
			failed++;
		}
	}

	if (ok > 0 || dropped > 0) {
		try {
			await invalidateAll();
		} catch {
			/* 忽略刷新异常 */
		}
	}

	if (ok > 0) showToast(`已同步 ${ok} 条离线记录`, 'ok');
	if (dropped > 0) showToast(`${dropped} 条离线记录因冲突未同步`, 'info');
	if (failed > 0) showToast(`${failed} 条离线记录同步失败，稍后自动重试`, 'info');
}

// ---------- 初始化：监听联网状态 ----------
let inited = false;
export function initOfflineSync(): void {
	if (!browser || inited) return;
	inited = true;

	window.addEventListener('online', () => {
		showToast('已恢复联网，正在同步离线记录…', 'info');
		flushOutbox();
	});
	window.addEventListener('offline', () => {
		showToast('已离线，改动将自动保存并在联网后同步', 'info');
	});

	// 启动即在线：补同步上次遗留的离线记录
	if (navigator.onLine) flushOutbox();
}
