server_addr = window.location.hostname;
if (!server_addr) {
	server_addr = 'localhost';
}

files_url = 'http://' + server_addr + ':2223/datafiles';
//$.getJSON(files_url, function(data){
//	console.log("json data= " + JSON.stringify(data));
//});

///////////////////////////////////////////////////////////////////////////////
var fft_sampleRate = 1000000 / 8192;	// 1Msps / 8192 = 122sps
var fftbuffer = [];

var fft_canvas = document.getElementById("fft-canvas");
var fft_ctx = fft_canvas.getContext("2d");

var dwt_canvas = document.getElementById("dwt-canvas");
var dwt_ctx = dwt_canvas.getContext("2d");
let dwt_num_samples = 2 * 1024 * 1024;

// used for color distribution
var hot = new chroma.ColorScale({
	//colors: ['#5C4D6B', '#536887', '#3D839A', '#259FA1', '#35B89B', '#67CF8A', '#A3E275', '#E7F065' ],
	//colors: ['#5C4D6B', '#3F627B', '#1C757A', '#2A8569', '#5B8F4F', '#91933B', '#C89140', '#F78B65' ],
	colors: [ '#000000', '#0B16B5', '#FFF782', '#EB1250' ],

	// colors:['#000000', '#ff0000', '#ffff00', '#ffffff'],
	// positions:[0, .125, 0.25, 0.375, 0.5, 0.625, 0.75, 1],
	positions: [ 0, 0.4, 0.68, 0.85 ],
	mode:'rgb',
	limits:[0, 1000]
});

//$("#fft-canvas").width($("#chart").width());
//function init_ui() {
//	$(window).resize(function(e) {
//		$("#fft-canvas").width($("#chart").width());
//	});
//}
//window.onload = init_ui;

// Where we are storing the results of the FFT
var sampleRate = 1000000;   // 1Msps
var width = 1024;           // FFT window (canvas) width
var height = 256;           // FFT window (canvas) height
var fft_points = height * 2;
var gain=45;
var floor = 40;
var extra_step = 0;

function drawFFT(arr) {
	//console.log(arr);
	var fft = new FFT(fft_points, sampleRate);
	var dspwindow = new WindowFunction(DSP.HAMMING);
	var fft_width = width / (arr.length / fft_points);
	if (fft_width < 1) {
		fft_width = 1;
		extra_step = Math.floor(arr.length / width);
	}
	//console.log('fft_points(%d) fft_width(%f) extra_step(%d) arr.length(%d)',
	//			fft_points, fft_width, extra_step, arr.length);

	// Clear spectrogram.
	fft_ctx.clearRect(0, 0, 1024, 256);

	for (var x = 0, offset = 0; x < width; x += fft_width, offset += (fft_points + extra_step)) {
		var arr_slice = arr.slice(offset, offset + fft_points);
		if (arr_slice.length < fft_points)
			break;

		var fftbuffer = new Float32Array(arr_slice);
		//console.log('fftbuffer(x=%d, offset=%d):', x, offset);
		//console.log(fftbuffer);
		// Before doing our FFT, we apply a window to attenuate frequency artifacts,
		// otherwise the spectrum will bleed all over the place:
		dspwindow.process(fftbuffer);
		// Do FFT.
		fft.forward(fftbuffer);
		var spectrum = new Float32Array(fft.spectrum);

		for (var y = 0; y < height; y++) {
			// draw each pixel with the specific color
			var value = 256 + gain*Math.log(spectrum[y]*floor);
			// draw the line on top of the canvas
			//console.log('x(%d) y(%d) value(%f) spectrum(%f)', x, y, value, spectrum[y]);
			fft_ctx.fillStyle = hot.getColor(value).hex();
			fft_ctx.fillRect(x, height - y, fft_width, 1);  // (x, y, w, h)
		}
		//console.log('offset(%d) x(%d) fft_width(%f)', offset, x, fft_width);
	}
}
///////////////////////////////////////////////////////////////////////////////


var chartdata = [];
var bit16list = {};

DrawChart = function (filename, arr) {
	//console.log("arr=%d",arr.length);
	console.log(arr);
	bit16list[filename] = arr;
	console.log(bit16list);
	var data = [];
	var samples = 500;
	var step = Math.floor(arr.length / samples);
	for (var i = 0; i < arr.length; i += step) {
		var sum = 0.0;
		for (var j = 0; j < step; j++) {
			sum += arr[i + j];
		}
		var avg = sum / step;
		data.push({x: i, y: avg});
	}
	console.log(data);
	nv.addGraph(function() {
		var chart = nv.models.lineWithFocusChart();
		chart.xAxis.tickFormat(d3.format(',d'));
		chart.x2Axis.tickFormat(d3.format(',d'));
		chart.yTickFormat(d3.format(',d'));
		chart.useInteractiveGuideline(true);
		chartdata.push({key: filename, values: data});

		var selection = d3.select("body");
		console.log(selection);
		d3.select('#chart svg')
			.datum(chartdata)
			.call(chart);
		nv.utils.windowResize(chart.update);
		return chart;
	});
}

ExportMaxMinCSV = function() {
	var bit16list = this.bit16list;
	var bit16sumlist = GetRawMaxMinCsv(bit16list)

	var blob = new Blob([bit16sumlist]);
	if (window.navigator.msSaveBlob) {
		window.navigator.msSaveBlob(blob, "");
		// msSaveOrOpenBlobの場合はファイルを保存せずに開ける
		window.navigator.msSaveOrOpenBlob(blob, "maxmindata.csv");
	} else {
		document.getElementById("csvdownload").href = window.URL.createObjectURL(blob);
	};
	console.log(blob);
	// console.log("json data= " + JSON.stringify(maxmindata))
}

ExportMaxMinData = function() {
	var bit16list = this.bit16list;
	console.log(bit16list);

	var source = GetRawMaxMin(bit16list);
	console.log(source);
	var blob = new Blob([source]);

	console.log("completed");

	if (window.navigator.msSaveBlob) {
		window.navigator.msSaveBlob(blob, "");
		// msSaveOrOpenBlobの場合はファイルを保存せずに開ける
		window.navigator.msSaveOrOpenBlob(blob, "maxmindata.dat");
	} else {
		document.getElementById("rawdownload").href = window.URL.createObjectURL(blob);
	};
	console.log(blob);
}

var constlist = [];

DrawMaxMin = function (filename, arr) {
	var maxmindata = [];
	var constdata = [];
	var samples = 5000;
	var step = 1000;
	console.log(arr);
	for (var i = 0; i < arr.length; i += step) {
		var sum = 0.0;
		for (var j = 0; j < step; j++) {
			sum += arr[i + j];
		}
		var avg = sum / step;
		constdata.push({x: i, y: avg});
	}
	console.log(constdata);
	constlist.push({key: filename, values: constdata});


	var lenlist = [];

	for(var i=0; i < constlist.length; i++){
		lenlist.push(constlist[i]["values"].length);
		console.log(lenlist);
	}

	maxmindata = GetMaxMin(constlist);

	nv.addGraph(function() {
		var maxminchart = nv.models.lineWithFocusChart();
		maxminchart.xAxis.tickFormat(d3.format(',d'));
		maxminchart.x2Axis.tickFormat(d3.format(',d'));
		maxminchart.yTickFormat(d3.format(',d'));
		maxminchart.useInteractiveGuideline(true);
		d3.select('#maxminchart svg')
			.datum(maxmindata)
			.call(maxminchart);
		nv.utils.windowResize(maxminchart.update);
		return maxminchart;
	});

}

DrawGraph = function(filename) {
	var url = 'http://' + server_addr + ':2223/csv/' + filename;
	var xhr = new XMLHttpRequest();
	xhr.open("GET", url, true);
	xhr.responseType = "arraybuffer";
	xhr.onload = function(e) {
		var a = new Uint16Array(xhr.response); // not responseText
		console.log(a);
		drawFFT(a);
		drawDWT(dwt_canvas, dwt_ctx, a);
		DrawChart(filename, a);
		DrawMaxMin(filename, a);
	}
	xhr.send();
}


GetMaxMin = function(constlist) {
	var lenlist = [];

	for(var i=0; i < constlist.length; i++){
		lenlist.push(constlist[i]["values"].length);
		console.log(lenlist);
	}
	var Maxlen = Math.max.apply(null, lenlist);

	console.log(Maxlen);
	var valuesumdata = new Array(Maxlen);
	for(var i=0; i < Maxlen; i++) {
		valuesumdata[i] = new Array(0);
	}

	for(var i=0; i < constlist.length; i++){
		for(var j=0; j < constlist[i]["values"].length; j++){
			valuesumdata[j].push(constlist[i]["values"][j]["y"]);
		}
	};
	var maxlist = [];
	var minlist = [];
	var step = 1000;
	for(var i=0 in valuesumdata){
		var max = Math.max.apply(null, valuesumdata[i]);
		var min = Math.min.apply(null, valuesumdata[i]);
		maxlist.push({x: i*step, y: max});
		minlist.push({x: i*step, y: min});
	}
	var maxmindata = [];

	maxmindata.push({key: "max", values: maxlist});
	maxmindata.push({key: "min", values: minlist});
	console.log(maxmindata);

	return maxmindata;


}


GetRawMaxMinCsv = function(bit16list) {
	bit16list = this.bit16list;
	console.log(bit16list);
	var data_list = [];
	var maxlen = 0;
	for (var name in bit16list) {
		data = bit16list[name];
		data_list.push(data);
		maxlen = Math.max(maxlen, data.length);
	}

	csv = "";
	for (var t = 0; t < maxlen; t++) {
		var max = Number.NEGATIVE_INFINITY;
		var min = Number.POSITIVE_INFINITY;
		for (var i = 0; i < data_list.length; i++) {
			data = data_list[i];
			if (t < data.length) {
				var v = data[t];
				max = Math.max(max, v);
				min = Math.min(min, v);
			}
		}
		csv += max + "," + min + "\n";
	}

	return csv;
}

GetRawMaxMin = function(bit16list) {
	bit16list = this.bit16list;
	console.log(bit16list);
	var data_list = [];
	var maxlen = 0;
	for (var name in bit16list) {
		data = bit16list[name];
		data_list.push(data);
		maxlen = Math.max(maxlen, data.length);
	}
	console.log(maxlen);

	var maxmindata = new Uint16Array(maxlen * 2);
	for (var t = 0; t < maxlen; t++) {
		var max = Number.NEGATIVE_INFINITY;
		var min = Number.POSITIVE_INFINITY;
		for (var i = 0; i < data_list.length; i++) {
			data = data_list[i]
			if (t < data.length) {
				var v = data[t];
				max = Math.max(max, v);
				min = Math.min(min, v);
			}
		}
		maxmindata[t * 2] = max;
		maxmindata[t * 2 + 1] = min;
	}

	return maxmindata;
}

var file_list = new Vue({
	el: '#file-list',
	components: VueMdl.components,
	directives: VueMdl.directives,
	data: {
		files: [],
		name: '',
		selected: false,
		chartindex:-1
	},
	methods: {
		reload: function(){
			var self = this;
			self.files = [];
			$.getJSON(files_url, function(data){
				data.forEach(function(entry, i){
					self.files.push({
						name: entry['name'],
						selected: false,
						chartindex:-1
					});
				});
			});
		},

		exportMaxMinData: function() {
			ExportMaxMinData();
		},

		exportMaxMinCSV: function() {
			ExportMaxMinCSV();
		},

		showGraph: function (file) {

			//var td = event.target;
			//console.log(td.innerText);
			//console.log(this.files);
			//DrawGraph(td.innerText);
			console.log(file.selected);

			if(chartdata.length > 0){
				console.log("lenght is");
				console.log(chartdata.length);
			};
			console.log("property is");
			console.log(chartdata.hasOwnProperty(file.name));

			if(file.selected){
				console.log(file.chartindex);
				// if(file.chartindex == -1){
				// 	file.chartindex = chartdata.length;
					DrawGraph(file.name);
				// }
			}else{
				// console.log(chartdata.indexOf(file.name));
				chartdata = chartdata.filter(function(item, index){
					if (item.key != file.name) return true;
				});
				// bit16list = bit16list.filter(function(item, index){
				// 	if (item.key != file.name) return true;
				// });
				delete bit16list[file.name];
				constlist = constlist.filter(function(item, index){
					if (item.key != file.name) return true;
				});

				var maxmindata = GetMaxMin(constlist);
					//console.log(data);
				// update
				nv.addGraph(function() {
					var chart = nv.models.lineWithFocusChart();
					chart.xAxis.tickFormat(d3.format(',d'));
					chart.x2Axis.tickFormat(d3.format(',d'));
					chart.yTickFormat(d3.format(',d'));
					chart.useInteractiveGuideline(true);
					d3.select('#chart svg')
						.datum(chartdata)
						.call(chart);
					nv.utils.windowResize(chart.update);
					return chart;
				});

				nv.addGraph(function() {
					var maxminchart = nv.models.lineWithFocusChart();
					maxminchart.xAxis.tickFormat(d3.format(',d'));
					maxminchart.x2Axis.tickFormat(d3.format(',d'));
					maxminchart.yTickFormat(d3.format(',d'));
					maxminchart.useInteractiveGuideline(true);
					d3.select('#maxminchart svg')
						.datum(maxmindata)
						.call(maxminchart);
					nv.utils.windowResize(maxminchart.update);
					return maxminchart;
				});

				console.log("delete" + file.name);
				console.log(chartdata);
				console.log(constlist);
				console.log(bit16list);
			}
		},
	},
});

file_list.reload();
