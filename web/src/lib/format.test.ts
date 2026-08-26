import { describe, it, expect } from 'vitest';
import { formatDate, formatDateTime, dueLabel, careTypeLabel, fmtDate, TASK_TYPE_LABEL, CARE_TYPE_LABEL } from './format';

describe('formatDate', () => {
	it('格式化日期', () => {
		const result = formatDate(new Date(2025, 2, 15)); // 3月15日
		expect(result).toContain('3月15日');
	});

	it('字符串输入', () => {
		const result = formatDate('2025-03-15');
		expect(result).toContain('3月15日');
	});
});

describe('formatDateTime', () => {
	it('格式化日期时间', () => {
		const result = formatDateTime(new Date(2025, 2, 15, 10, 5));
		expect(result).toContain('3月15日');
		expect(result).toContain('10:05');
	});

	it('补零', () => {
		const result = formatDateTime(new Date(2025, 2, 15, 9, 3));
		expect(result).toContain('09:03');
	});
});

describe('dueLabel', () => {
	const today = new Date();
	const tomorrow = new Date(today);
	tomorrow.setDate(tomorrow.getDate() + 1);
	const yesterday = new Date(today);
	yesterday.setDate(yesterday.getDate() - 1);

	function fmt(d: Date) {
		return d.toISOString().split('T')[0];
	}

	it('今天', () => {
		const result = dueLabel(fmt(today));
		expect(result.text).toBe('今天');
		expect(result.today).toBe(true);
		expect(result.overdue).toBe(false);
	});

	it('明天', () => {
		const result = dueLabel(fmt(tomorrow));
		expect(result.text).toBe('明天');
		expect(result.today).toBe(false);
		expect(result.overdue).toBe(false);
	});

	it('昨天（逾期）', () => {
		const result = dueLabel(fmt(yesterday));
		expect(result.text).toBe('昨天');
		expect(result.overdue).toBe(true);
	});

	it('逾期X天', () => {
		const d = new Date(today);
		d.setDate(d.getDate() - 3);
		const result = dueLabel(fmt(d));
		expect(result.text).toBe('逾期3天');
		expect(result.overdue).toBe(true);
	});

	it('X天后', () => {
		const d = new Date(today);
		d.setDate(d.getDate() + 5);
		const result = dueLabel(fmt(d));
		expect(result.text).toBe('5天后');
		expect(result.overdue).toBe(false);
	});
});

describe('careTypeLabel', () => {
	it('已知类型映射', () => {
		expect(careTypeLabel('water')).toBe('浇水');
		expect(careTypeLabel('fertilize')).toBe('施肥');
		expect(careTypeLabel('repot')).toBe('换盆');
		expect(careTypeLabel('prune')).toBe('修剪');
		expect(careTypeLabel('other')).toBe('其他');
	});

	it('未知类型返回原文', () => {
		expect(careTypeLabel('unknown')).toBe('unknown');
	});
});

describe('fmtDate', () => {
	it('空值返回空', () => {
		expect(fmtDate(null)).toBe('');
		expect(fmtDate(undefined)).toBe('');
		expect(fmtDate('')).toBe('');
	});

	it('格式化', () => {
		expect(fmtDate('2025-03-15')).toBe('2025-03-15');
	});

	it('补零', () => {
		expect(fmtDate('2025-01-05')).toBe('2025-01-05');
	});
});

describe('TASK_TYPE_LABEL', () => {
	it('包含所有类型', () => {
		expect(TASK_TYPE_LABEL.water).toBe('浇水');
		expect(TASK_TYPE_LABEL.fertilize).toBe('施肥');
		expect(TASK_TYPE_LABEL.repot).toBe('换盆');
	});
});

describe('CARE_TYPE_LABEL', () => {
	it('包含所有类型', () => {
		expect(CARE_TYPE_LABEL.water).toBe('浇水');
		expect(CARE_TYPE_LABEL.prune).toBe('修剪');
		expect(CARE_TYPE_LABEL.other).toBe('其他');
	});
});
