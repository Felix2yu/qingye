// 日期与任务相关展示工具

const WEEKDAYS = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];

export function formatDate(d: string | Date): string {
	const date = typeof d === 'string' ? new Date(d) : d;
	return `${date.getMonth() + 1}月${date.getDate()}日 ${WEEKDAYS[date.getDay()]}`;
}

export function formatDateTime(d: string | Date): string {
	const date = typeof d === 'string' ? new Date(d) : d;
	const pad = (n: number) => String(n).padStart(2, '0');
	return `${date.getMonth() + 1}月${date.getDate()}日 ${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

// 相对今天的到期描述：今天 / 明天 / 昨天(逾期) / X天后 / 逾期X天
export function dueLabel(nextDue: string): { text: string; overdue: boolean; today: boolean } {
	const due = new Date(nextDue);
	const now = new Date();
	const startOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate());
	const startOfDue = new Date(due.getFullYear(), due.getMonth(), due.getDate());
	const diffDays = Math.round((startOfDue.getTime() - startOfToday.getTime()) / 86400000);

	if (diffDays === 0) return { text: '今天', overdue: false, today: true };
	if (diffDays === 1) return { text: '明天', overdue: false, today: false };
	if (diffDays === -1) return { text: '昨天', overdue: true, today: false };
	if (diffDays < 0) return { text: `逾期${-diffDays}天`, overdue: true, today: false };
	return { text: `${diffDays}天后`, overdue: false, today: false };
}

export function greeting(): string {
	const h = new Date().getHours();
	if (h < 6) return '夜深了';
	if (h < 12) return '早上好';
	if (h < 14) return '中午好';
	if (h < 18) return '下午好';
	return '晚上好';
}

export const TASK_TYPE_LABEL: Record<string, string> = {
	water: '浇水',
	fertilize: '施肥',
	repot: '换盆'
};

export const TASK_TYPE_EMOJI: Record<string, string> = {
	water: '💧',
	fertilize: '🌱',
	repot: '🪴'
};

// 养护事件类型中文映射（含 other / prune）
export const CARE_TYPE_LABEL: Record<string, string> = {
	water: '浇水',
	fertilize: '施肥',
	repot: '换盆',
	prune: '修剪',
	other: '其他'
};

export function careTypeLabel(type: string): string {
	return CARE_TYPE_LABEL[type] ?? type;
}

// 紧凑日期（YYYY-MM-DD），用于延期标签等
export function fmtDate(s: string | null | undefined): string {
	if (!s) return '';
	const d = new Date(s);
	if (isNaN(d.getTime())) return '';
	return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}
