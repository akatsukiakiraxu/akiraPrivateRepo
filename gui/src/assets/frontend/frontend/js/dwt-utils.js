function arrayLikeToArray(arr) {
	var result = [];
	for (var i = 0; i < arr.length; i++) {
		result.push(arr[i]);
	}
	return result;
}

function dwtSummarizeToPowerOf2(arr) {
	var ndigits = 0;
	for (var l = arr.length; l >= 1; l /= 2) {
		ndigits++;
	}
	if (ndigits == 0 || l == 0.5) {
		return arrayLikeToArray(arr);
	}

	var target_length = 1 << (ndigits - 1);
	var result = [];

	result.push(arr[0]);
	for (var i = 1; i < target_length - 1; ++i) {
		var index = ((arr.length - 1) / (target_length - 1)) * i;
		var integ = Math.floor(index);
		var fract = index - integ;
		var val = arr[integ] * (1 - fract) + arr[integ + 1] * fract;
		result.push(val);
	}
	result.push(arr[arr.length - 1]);

	return result;
}

// arr.length and target_size must be power of 2.
function dwtShrink(arr, target_size) {
	var n = arr.length / target_size;
	if (n <= 1) {
		return [].concat(arr);
	}
	var result = [];
	for (var i = 0; i < target_size; i++) {
		var sum = 0;
		for (j = 0; j < n; j++) {
			sum += arr[n * i + j];
		}
		result.push(sum / n);
	}

	return result;
}

function dwtSummarize(arr) {
	samples = []
	let elem_per_sample = Math.floor(arr.length / dwt_num_samples);
	for (var x = 0; x < dwt_num_samples; x++) {
		sum = 0;
		offset = elem_per_sample * x;
		for (var i = 0; i < elem_per_sample; i++) {
			sum += arr[offset + i];
		}
		samples.push(sum / dwt_num_samples);
	}
	return samples;
}

function dwtAbs(arr) {
	for (var i = 0; i < arr.length; i++) {
		arr[i] = Math.abs(arr[i]);
	}
	return arr;
}

function dwtPixelSeparate(length, nseparate, n) {
	start = Math.round(length * n / nseparate);
	stop  = Math.round(length * (n + 1) / nseparate);
	return [start, stop - start];
}

// arr.length must be power of 2.
function multiresolutionAnalysis(arr) {
	data = [].concat(arr) // Shallow copy.
	result = new Array(data.length);
	for (var l = data.length / 2; l >= 1; l /= 2) {
		for (var i = 0; i < l; i++) {
			result[i]     = (data[i * 2] + data[i * 2 + 1]) / 2.0;
			result[l + i] = (data[i * 2] - data[i * 2 + 1]) / 2.0;
		}
		for (var i = 0; i < l; i++) {
			data[i] = result[i];
		}
	}
	return result;
}

function dwtLog2(n) {
	result = 0;
	while (n >= 2) {
		result++;
		n /= 2;
	}
	return result;
}

function numDigits(n) {
	var ndigits = 0;
	while (n >= 1) {
		n /= 10;
		ndigits++;
	}
	return ndigits;
}

function numReadable(n) {
	var pow10 = 0;
	if (n >= 1) {
		if (n >= 1000) {
			n /= 1000;
			return n.toFixed(3 - numDigits(n)) + "k";
		} else if (n >= 1) {
			return n.toFixed(3 - numDigits(n));
		}
	} else  {
		while (n < 1) {
			n *= 1000;
			pow10 -= 3;
		}
		return n.toFixed(3 - numDigits(n)) + "e" + pow10;
	}
}

let x_axis_height = 20;
let y_axis_width = 60;
let max_font_weight = 12;
let axis_margin = 5;
let left_padding = 50;

function drawDWT(canvas, context, arr) {
	var data = dwtSummarizeToPowerOf2(arr);
	// data = dwtShrink(data, 2 * 1024 * 1024);
	dwt = multiresolutionAnalysis(data);
	dwt = dwtAbs(dwt);

	context.clearRect(0, 0, 1024, 256);

	let canvas_h = canvas.clientHeight - x_axis_height;
	let canvas_w = canvas.clientWidth - y_axis_width - left_padding;

	context.fillStyle = 'white';
	let dlen = data.length;
	sample_height = dwtLog2(dlen);
	context.fillRect(0, canvas_h, canvas_w + y_axis_width, x_axis_height);
	context.fillRect(0, 0, y_axis_width, canvas_h);
	context.fillRect(y_axis_width + canvas_w, 0, left_padding, canvas_h + x_axis_height);

	max_freq = 125000;
	min_freq = max_freq / dlen;
	row_h = Math.floor(canvas_h / sample_height);
	context.fillStyle = 'black';
	context.font = Math.max(row_h, max_font_weight) + 'px sans-serif'
	context.textBaseline = 'bottom';

	context.textAlign = 'right';
	for (var i = 0, f = max_freq; i < sample_height; i++, f /= 2) {
		var y = dwtPixelSeparate(canvas_h, sample_height, i)[0];
		console.log(y_axis_width - 5, y + row_h);
		context.fillText(
			numReadable(f) + 'Hz',
			y_axis_width - axis_margin,
			y + row_h + 1,
			y_axis_width);
	}

	context.textAlign = 'center';
	nTic = Math.floor(canvas_w / 100);
	for (var i = 0; i <= nTic; i++) {
		var x = y_axis_width + canvas_w * i / nTic;
		console.log(canvas_w, nTic, x);
		context.fillText(
			Math.floor(arr.length / nTic * i).toLocaleString(),
			x,
			canvas_h + x_axis_height,
			canvas_w / nTic);
	}

	var index = 1;
	for (var i = 0, l = 1; l <= dlen / 2; i++, l *= 2) {
		var yh = dwtPixelSeparate(canvas_h, sample_height, i);
		var h = yh[1];
		var y = canvas_h - yh[0] - h;
		if (h == 0) {
			index += l;
			continue;
		}
		if (l <= canvas_w) {
			for (var j = 0; j < l; j++) {
				var xw = dwtPixelSeparate(canvas_w, l, j);
				var x = y_axis_width + xw[0];
				var w = xw[1];
				context.fillStyle = hot.getColor(dwt[index]).hex();
				context.fillRect(x, y, w, h);
				index++;
			}
		} else {
			for (var j = 0; j < canvas_w; j++) {
				var xw = dwtPixelSeparate(l, canvas_w, j);
				var w = xw[1];
				var x = y_axis_width + j;
				var max = 0;
				for (var k = 0; k < w; k++) {
					max = Math.max(dwt[index], max);
					index++;
				}
				context.fillStyle = hot.getColor(max).hex();
				context.fillRect(x, y, 1, h);
			}
		}
	}
}
