// 照片前端压缩（参考 draix-garden）：读取 File → Image → canvas 缩放 → toBlob(quality)
// 长边不超过 maxSize，体积大幅降低后再上传，节省带宽与服务端存储。

export async function compressImage(
	file: File,
	opts: { maxSize?: number; quality?: number } = {}
): Promise<File> {
	const maxSize = opts.maxSize ?? 1280;
	const quality = opts.quality ?? 0.8;

	if (!file.type.startsWith('image/')) return file;
	// 已是较小文件且非超清时直接返回，避免无谓处理
	if (file.size <= 300 * 1024) return file;

	const bitmap = await loadBitmap(file);
	try {
		const { width, height } = bitmap;
		const scale = Math.min(1, maxSize / Math.max(width, height));
		const w = Math.round(width * scale);
		const h = Math.round(height * scale);

		const canvas = document.createElement('canvas');
		canvas.width = w;
		canvas.height = h;
		const ctx = canvas.getContext('2d');
		if (!ctx) return file;
		ctx.drawImage(bitmap as CanvasImageSource, 0, 0, w, h);

		const blob = await new Promise<Blob | null>((resolve) =>
			canvas.toBlob(resolve, 'image/jpeg', quality)
		);
		if (!blob) return file;
		const ext = file.name.toLowerCase().endsWith('.png') ? '.png' : '.jpg';
		const name = (file.name.replace(/\.[^.]+$/, '') || 'upload') + ext;
		return new File([blob], name, { type: 'image/jpeg' });
	} finally {
		if ('close' in bitmap) (bitmap as ImageBitmap).close();
	}
}

// 兼容 createImageBitmap 不可用场景
function loadBitmap(file: File): Promise<HTMLImageElement | ImageBitmap> {
	if (typeof createImageBitmap === 'function') {
		return createImageBitmap(file);
	}
	return new Promise((resolve, reject) => {
		const url = URL.createObjectURL(file);
		const img = new Image();
		img.onload = () => {
			URL.revokeObjectURL(url);
			resolve(img);
		};
		img.onerror = (e) => {
			URL.revokeObjectURL(url);
			reject(e);
		};
		img.src = url;
	});
}
