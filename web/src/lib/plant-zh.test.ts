import { describe, it, expect } from 'vitest';
import { zhCategory, zhOrigin } from './plant-zh';

describe('zhCategory', () => {
	it('空值返回空', () => {
		expect(zhCategory(undefined)).toBe('');
		expect(zhCategory('')).toBe('');
	});

	it('常见类别映射', () => {
		expect(zhCategory('Foliary plant')).toBe('观叶植物');
		expect(zhCategory('Succulent')).toBe('多肉植物');
		expect(zhCategory('Cacti')).toBe('仙人掌类');
		expect(zhCategory('Flowering plant')).toBe('开花植物');
		expect(zhCategory('Orchid')).toBe('兰科植物');
		expect(zhCategory('Fern')).toBe('蕨类植物');
		expect(zhCategory('Bonsai')).toBe('盆景');
		expect(zhCategory('Herb')).toBe('香草');
		expect(zhCategory('Tree')).toBe('乔木');
		expect(zhCategory('Shrub')).toBe('灌木');
		expect(zhCategory('Vegetable')).toBe('蔬菜');
		expect(zhCategory('Fruit')).toBe('果树');
	});

	it('大小写不敏感', () => {
		expect(zhCategory('succulent')).toBe('多肉植物');
		expect(zhCategory('CACTI')).toBe('仙人掌类');
	});

	it('未命中返回原文', () => {
		expect(zhCategory('Unknown Type')).toBe('Unknown Type');
	});
});

describe('zhOrigin', () => {
	it('空值返回空', () => {
		expect(zhOrigin(undefined)).toBe('');
		expect(zhOrigin('')).toBe('');
	});

	it('国家映射', () => {
		expect(zhOrigin('Mexico')).toBe('墨西哥');
		expect(zhOrigin('Brazil')).toBe('巴西');
		expect(zhOrigin('China')).toBe('中国');
		expect(zhOrigin('Japan')).toBe('日本');
		expect(zhOrigin('India')).toBe('印度');
		expect(zhOrigin('Australia')).toBe('澳大利亚');
	});

	it('地区映射', () => {
		expect(zhOrigin('Southeast Asia')).toBe('东南亚');
		expect(zhOrigin('Mediterranean')).toBe('地中海地区');
		expect(zhOrigin('Central America')).toBe('中美洲');
		expect(zhOrigin('South America')).toBe('南美洲');
		expect(zhOrigin('North America')).toBe('北美洲');
		expect(zhOrigin('Hawaii')).toBe('夏威夷');
	});

	it('大小写不敏感', () => {
		expect(zhOrigin('mexico')).toBe('墨西哥');
		expect(zhOrigin('JAPAN')).toBe('日本');
	});

	it('未命中返回原文', () => {
		expect(zhOrigin('Atlantis')).toBe('Atlantis');
	});
});
