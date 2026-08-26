import { describe, it, expect } from 'vitest';
import { parseMetrics, range } from './metrics';

describe('parseMetrics', () => {
	it('空值返回 null', () => {
		expect(parseMetrics(undefined)).toBeNull();
		expect(parseMetrics('')).toBeNull();
	});

	it('有效 JSON 解析', () => {
		const raw = JSON.stringify({ minTemp: 10, maxTemp: 35 });
		expect(parseMetrics(raw)).toEqual({ minTemp: 10, maxTemp: 35 });
	});

	it('无效 JSON 返回 null', () => {
		expect(parseMetrics('not json')).toBeNull();
		expect(parseMetrics('{invalid}')).toBeNull();
	});

	it('部分字段缺失不报错', () => {
		const raw = JSON.stringify({ minLightMmol: 50 });
		const m = parseMetrics(raw);
		expect(m?.minLightMmol).toBe(50);
		expect(m?.maxLightMmol).toBeUndefined();
	});
});

describe('range', () => {
	it('两端都有值', () => {
		expect(range(10, 35, ' ℃')).toBe('10 ~ 35 ℃');
	});

	it('仅最小值', () => {
		expect(range(10, undefined, ' ℃')).toBe('10 ~ – ℃');
	});

	it('仅最大值', () => {
		expect(range(undefined, 35, ' ℃')).toBe('– ~ 35 ℃');
	});

	it('两端都无值返回空', () => {
		expect(range(undefined, undefined, ' ℃')).toBe('');
	});

	it('无单位', () => {
		expect(range(1, 2)).toBe('1 ~ 2');
	});
});
