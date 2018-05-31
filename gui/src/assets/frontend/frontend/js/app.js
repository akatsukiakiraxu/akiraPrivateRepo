let INVALID_EXPECT = 1 << 31; // Large value accepted by D3.js.

//## Olive側の設定 - 起動時にAPIから取得する項目
var activeChannel = {};		//## チャネル設定
var chs = [];
var mntSetting = {};		//## モニタリング設定
var channelTrigger = {};	//## チャネルトリガー設定
var number_of_samples = -1;	//## 1ページのサンプリング数

//## ユーザが選択した有効範囲
var usrX0 = 0;
var usrX1 = 0;

var chart_margin = 20;
var chart_width = 1000;
var chart_height = 500;

//var minimum_value = -4;
//var maximum_value = 4;

var fft_chart_margin = 20;
var fft_chart_width = 1000;
var fft_chart_height = 500;
var fft_minimum_value = 0;
var fft_maximum_value = 10*10;


var chart_buffer_size = 1000;
var fft_buffer_size = 1024;
var chart_renderer = null;
var ws = null;

var fft_sampleRate = 1000000 / 8192;	// 1Msps / 8192 = 122sps
var fft_canvas_width = 1000;		// FFT window (canvas) width
var fft_canvas_height = 256;		// FFT window (canvas) height
//var fft_points = fft_canvas_height * 2;
var fft_gain=45;
var fft_floor = 40;
var fft_max_freq = 125000
var fft = new FFT(fft_canvas_height * 2, fft_sampleRate);
var fftbuffer = [];
var fft_dspwindow = new WindowFunction(DSP.HAMMING);
var fft_hot = new chroma.ColorScale({
	//colors: ['#5C4D6B', '#536887', '#3D839A', '#259FA1', '#35B89B', '#67CF8A', '#A3E275', '#E7F065' ],
	//colors: ['#5C4D6B', '#3F627B', '#1C757A', '#2A8569', '#5B8F4F', '#91933B', '#C89140', '#F78B65' ],
	colors: [ '#000000', '#0B16B5', '#FFF782', '#EB1250' ],
	// colors:['#000000', '#ff0000', '#ffff00', '#ffffff'],
	// positions:[0, .125, 0.25, 0.375, 0.5, 0.625, 0.75, 1],
	positions: [ 0, 0.4, 0.68, 0.85 ],
	mode:'rgb',
	limits:[0, 900]
});
//var fft_canvas = $("#fft-canvas").get()[0];
var fft_canvas = document.getElementById("fft-canvas");
var fft_ctx =fft_canvas.getContext("2d");
// create a temporary canvas we use for copying
var tmpCanvas = document.createElement("canvas"),
    tmpCtx = tmpCanvas.getContext("2d");
tmpCanvas.width = fft_canvas_width;
tmpCanvas.height = fft_canvas_height;

//# 3 - チャネル・データ
var onChdata = false;		//# チャネル・データ到達済み？
var chCnt = -1;				//# チャネル数
var chNm = null;			//# チャネル名
var resv = 0;

RealtimeConnection = function(addr){
	var self = this;
	this.onmessage = null;
	this.connection = new WebSocket('ws://' + addr + ':2222/monitoring/summary');
	this.connection.binaryType = 'arraybuffer';
	
	this.connection.onmessage = function(e){
		
		var s = new DataView(e.data);
		
		let HEADER_LENGTH = 8;
		var withTriggerFlg = false;
		var type = s.getUint16(0, true);	//# データ種別
		var flag = s.getUint16(2,true);		//# フラグ
		var length = s.getUint32(4, true);	//# データ長(ヘッダの長さは含まない)
		
		if(type == 0 && flag == 1){
			withTriggerFlg = true;
		}
		
		switch(onChdata){
			case true:
				switch (type) {
					case 0: // Raw data.
						
						//# SUMMARY DATA FORMAT ---
						//# Offset, Name, Type, Size, Description
						//# 0x00, SummaryMin, float32, 4, サマリ最小値
						//# 0x04, SummaryMax, float32, 4, サマリ最大値
						//# 0x08, ErrorMin, float32, 4, エラー最小値
						//# 0x0c, ErrorMax, float32, 4, エラー最大値
						//# 0x10, WarningMin, float32, 4, 警告最小値
						//# 0x14, WarningMax, float32, 4, 警告最大値
						//# -----------------------
						
						var result = [];
						let COLUMN_LENGTH = 4;
						let COLUMN_COUNT = 6;
						var aDataLength = COLUMN_LENGTH * COLUMN_COUNT;
						var number_of_channels = length / aDataLength;

						for (var c = 0; c < number_of_channels; c++) {
							channel_data = [];
							for (var i = 0; i < COLUMN_COUNT; i++) {
								channel_data.push(s.getFloat32(HEADER_LENGTH + aDataLength * c + 4 * i, true));
							}
							result.push(channel_data);
						}
						chart_renderer.push(result, withTriggerFlg);
						chart_renderer.updateData();
						break;
						
					case 1: // FFT
						var numberOfPoints = s.getUint32(HEADER_LENGTH + 0, true);
						// TODO: iterate for each channels and set FFT data. 
						chart_renderer.set_fft(0, new Float32Array(e.data, HEADER_LENGTH + 4, numberOfPoints));
						chart_renderer.updateData();
						break;
						
					case 2: // Expected values of FFT
						chart_renderer.set_fft_expected(new Uint16Array(e.data, 6));
						break;
				}
				break;
				
			default:
				if(type == 3){
					//# 3 - チャネル・データの保管
					chCnt = s.getUint16(HEADER_LENGTH + 0, true);　
					resv = s.getUint16(HEADER_LENGTH + 2, true);
					chNm = e.data.slice(HEADER_LENGTH + 4);
					chNm = String.fromCharCode.apply(null, new Uint8Array(chNm));
					onChdata = true;
				}
				break;
		}
	};
};

RealtimeConnection.prototype.close = function(){
	this.connection.close();
};

//## axis関連の項目
var svg = null;
var xScale = d3.scaleLinear();
var yScale = d3.scaleLinear();
var xAxisCall = d3.axisBottom();
xAxisCall.tickSizeInner(-(chart_height))  // 目盛線の長さ（内側）
	.tickSizeOuter(5) // 目盛線の長さ（外側）
	.tickPadding(10); // 目盛線とテキストの間の長さ
var yAxisCall = d3.axisLeft();
yAxisCall.tickSizeInner(-(chart_width-chart_margin))  // 目盛線の長さ（内側）
	.tickSizeOuter(5) // 目盛線の長さ（外側）
	.tickPadding(10); // 目盛線とテキストの間の長さ
var trgThresholdLine = null;
var trgThresholdPath = null;
var t = d3.transition().duration(380);

//## axis scaleセット
function setScale(){
	
	var minValue = activeChannel[chs[chart_renderer.channel]].minimum_value;
	var maxValue = activeChannel[chs[chart_renderer.channel]].maximum_value;
    xScale.domain([0, number_of_samples]).range([chart_margin, chart_width]);
    yScale.domain([minValue, maxValue]).range([chart_height, 0]);
    xAxisCall.scale(xScale);
    yAxisCall.scale(yScale);
	
	trgThresholdLine = d3.path();
	trgThresholdLine.moveTo(xScale(0), yScale(channelTrigger.threshold));
	trgThresholdLine.lineTo(xScale(number_of_samples), yScale(channelTrigger.threshold));
}

//## axis初期化
function initAxis() {
	svg.append("g")
		.attr("class", "y")
		.attr("transform", "translate("+chart_margin+",0)")
        .attr("stroke", "rgb(255,255,255)")
		.call(yAxisCall)
		.selectAll(".tick line").attr("stroke", "rgb(255,255,255)");
	
    svg.append("g")
		.attr("class", "x")
		.attr("transform", "translate(0,"+(chart_height)+")")
		.attr("stroke", "rgb(255,255,255)")
		.call(xAxisCall)
		.selectAll(".tick line").attr("stroke", "rgb(255,255,255)");
}

//## axis更新  
function updateAxis(){
	
    svg.select(".x")
        .transition(t)
        .call(xAxisCall)
		.selectAll(".tick line").attr("stroke", "rgb(255,255,255)");
    svg.select(".y")
		.transition(t)
		.call(yAxisCall)
		.selectAll(".tick line").attr("stroke", "rgb(255,255,255)");
}

//## チャネルトリガー閾値初期化
function initTriggerThreshold(){
	
	trgThresholdPath = svg.append("path").attr("id", "trg");
	updateTriggerThreshold();
}

//## チャネルトリガー閾値更新
function updateTriggerThreshold(){
	
	console.log('channelTrigger.threshold.trigger_mode:' + channelTrigger.trigger_mode);
	
	if(channelTrigger.trigger_mode != 'disabled'){
		trgThresholdPath
			.transition(t)
			.attr("stroke", "red")
			.attr("d", trgThresholdLine);
	}
}

//## 選択範囲取得・表示
function brushed() {
	var x0 = d3.event.selection[0][0];
	var y1 = d3.event.selection[0][1];
	var x1 = d3.event.selection[1][0];
	var y0 = d3.event.selection[1][1];

	usrX0 = Math.round(x0);
	usrX1 = Math.round(x1);
	if(usrX0 == usrX1){
		usrX0 = 0;
		usrX1 = 0;
	}
	else{
		usrX0 = usrX0 - chart_margin;
	}
	if(window.parent.document.getElementById('selectedRange') != null){
		window.parent.document.getElementById('selectedRange').value = 'Selected: ' + usrX0 +' to '+ usrX1;
	}
}

ChartRenderer = function(el, fft_el){
	this.el = el;
	this.fft_el = fft_el;
	this.svg_selector = null;
	this.channel = 0;
	this.samples = [];
	this.expected = [];
	this.fft_expected = [];
	this.max_data = [];
	this.min_data = [];
	this.error_cnt = [];
	this.fft_data = []
	this.line_chart01;
	this.line_chart02;
	this.line_chart03;
	this.sample_index = 0;
	for (var c = 0; c < Object.keys(activeChannel).length; c++) {
		this.expected.push([]);
		this.max_data.push(Number.NEGATIVE_INFINITY);
		this.min_data.push(Number.POSITIVE_INFINITY);
		this.error_cnt.push(0);
		this.fft_data.push([]);
	}

	this.samples = new Array(Object.keys(activeChannel).length);
	for(var idx=0; idx<this.samples.length; idx++){
		this.samples[idx] = new Array(chart_buffer_size);
		for(var sampleIdx=0; sampleIdx<chart_buffer_size; sampleIdx++){
			this.samples[idx][sampleIdx] = [0, 0, 0, 0, 0, 0];
		}
	}
	
	var curCh = window.parent.document.getElementById('channel').value.replace('string:', '');
	this.channel = chs.indexOf(curCh);
	
	this.update();
};

ChartRenderer.prototype.update = function(){
	
	var svg = d3.select(this.el);
	var fft_svg = d3.select(this.fft_el);
	var ch = this.channel;
	var n = this.samples[ch].length;

	var x = xScale;
	var y = d3.scaleLinear()
		.domain([activeChannel[chs[ch]].minimum_value, activeChannel[chs[ch]].maximum_value])
		.range([chart_height, 0]);
	
	var fft_x = d3.scaleLinear()
		.domain([0,fft_buffer_size])
		.range([fft_chart_margin, fft_chart_width]);
	var fft_y = d3.scaleLinear()
		.domain([fft_minimum_value, fft_maximum_value])
		.range([fft_chart_height, 0]);

	var d= new Date();
	utc = d.getTime() + (d.getTimezoneOffset() * 60000);
	var jikan = new Date(utc);

	var images = svg
		.append('image')
		.attr('xlink:href', './current.png?t=' + jikan.toUTCString())
		.attr('width', 100)
		.attr('height', 100)
		//.attr('clip-path', 'url(#clip)')
		.attr('x', 900)
		.attr('y', 90);

	var area = d3.area()
		.x(function(d, i) { return x(i); })
		.y0(function(d) { return y(d[0]); })
		.y1(function(d) { return y(d[1]); });

	var fft_area = d3.area()
		.x(function(d, i) { return fft_x(i); })
		.y0(function(d) { return fft_y(d[0]); })
		.y1(function(d) { return fft_y(d[1]); });

//	svg.selectAll("text").remove();//
//
//	var max_data_text;
//	var min_data_text;
//	if (this.max_data[ch] < this.min_data[ch]) {
//		max_data_text = "Max Data: N/A";
//		min_data_text = "Min Data: N/A";
//	} else {
//		max_data_text = "Max Data:" + this.max_data[ch];
//		min_data_text = "Min Data:" + this.min_data[ch];
//	}
//	svg.append("text")
//		.attr("transform",
//			"translate(" + (chart_width - chart_margin/4) + " ," +
//			(30) + ")")
//		//.style("text-anchor", "middle")
//		.style("text-anchor", "end")
//		.text(max_data_text);
//
//	svg.append("text")
//		.attr("class", "x label")
//		.attr("text-anchor", "end")
//		.attr("x", (chart_width - chart_margin/4))
//		.attr("y", 50)
//		.text(min_data_text);
//
//	svg.append("text")
//		.attr("class", "x label")
//		.attr("text-anchor", "end")
//		.attr("x", (chart_width - chart_margin/4))
//		.attr("y", 70)
//		.text("Number of Error Count:" + this.error_cnt[ch]);
//	
//	svg.selectAll("path")
//		.data([this.expected[ch]])
//		.enter()
//		.append("path")
//		.attr("d", area)
//		.attr("fill", "#209e91" );
//	svg.selectAll("path")
//		.data([this.expected[ch]])
//		.attr("d", area);

	// Graph of summarized values.
	var summary_area = d3.area()
		.x(function(d, i){ return x(i); })
		.y0(function(d){ return ( 1 * y(d[1])); })
		.y1(function(d){ return 1 * y(d[0]); });
	
	var area_old = d3.area()
		.x(function(d, i){ return x(i); })
		.y0(function(d){ return ( 1 * y(d[1])); })
		.y1(function(d){ return 1 * y(d[0]); });
	
	var error_area = d3.area()
		.x(function(d, i){ return x(i); })
		.y0(function(d){ return ( 1 * y(d[3])); })
		.y1(function(d){ return 1 * y(d[2]); });
	
	var warning_area = d3.area()
		.x(function(d, i){ return x(i); })
		.y0(function(d){ return ( 1 * y(d[5])); })
		.y1(function(d){ return 1 * y(d[4]); });
	
	this.path_error = svg.append('g').append('path')
		.data([this.samples[ch]])
		.attr('class', 'chart-line-min')
		.attr('d', error_area);
	
	this.path_warning = svg.append('g').append('path')
		.data([this.samples[ch]])
		.attr('class', 'chart-line-min')
		.attr('d', warning_area);

	this.path_summary = svg.append('g').append('path')
		.data([this.samples[ch]])
		.attr('class', 'chart-line-min')
		.attr('d', summary_area);
	
	this.old_signal_path = svg.append('g').append('path')
		.data([this.samples[ch]])
		.attr('class', 'chart-line-min')
		.attr('d', area_old);
		
//	this.scanline_path = svg.append("path");

	var x_axis = d3.axisBottom(x)
		.tickSizeInner(-(chart_height))  // 目盛線の長さ（内側）
		.tickSizeOuter(5) // 目盛線の長さ（外側）
		.tickPadding(10); // 目盛線とテキストの間の長さ
	var y_axis = d3.axisLeft(y)
		.tickSizeInner(-(chart_width-chart_margin))  // 目盛線の長さ（内側）
		.tickSizeOuter(5) // 目盛線の長さ（外側）
		.tickPadding(10); // 目盛線とテキストの間の長さ

	//# 有効範囲選択
	var brush = d3.brush()
	.extent([
		[chart_margin, 0],
		[chart_width, chart_height]
	])
	.on("start brush", brushed);

	svg.append("g")
	.call(brush)
	.call(brush.move, [
		[x(0), y(0)],
		[x(0.01), y(0.01)]
	]);
	
	this.path_fft = fft_svg.append('g').append('path')
	// 		.data([this.fft_data[ch]])
	// 		.attr('class', 'chart-line')
	// 		.attr('d', fft_line);

//	var x_axis_fft = d3.axisBottom(x)
//		.tickSizeInner(-(fft_chart_height))  // 目盛線の長さ（内側）
//		.tickSizeOuter(5) // 目盛線の長さ（外側）
//		.tickPadding(10); // 目盛線とテキストの間の長さ
//	var y_axis_fft = d3.axisLeft(y)
//		.tickSizeInner(-(fft_chart_width - fft_chart_margin))  // 目盛線の長さ（内側）
//		.tickSizeOuter(5) // 目盛線の長さ（外側）
//		.tickPadding(10); // 目盛線とテキストの間の長さ
//	
//	var fft_svg_yaxis = fft_svg.append('g')
//						.attr("transform", "translate(0,"+(fft_chart_height)+")")
//						.attr("stroke", "rgba(255,255,255,255)")
//						.call(function(g) {
//							g.call(x_axis_fft);
//							g.selectAll(".tick line").attr("stroke", "rgba(255,255,255,255)");
//						});
//	var fft_svg_xaxis = fft_svg.append('g')
//						.attr("transform", "translate("+fft_chart_margin+",0)")
//						.attr("stroke", "rgba(255,255,255,255)")
//						.call(function(g) {
//							g.call(y_axis_fft);
//							g.selectAll(".tick line").attr("stroke", "rgba(255,255,255,255)");
//						});
//	
	var fft_axis = fft_svg.append('g');
	fft_axis.append('rect')
		.attr('x', 0)
		.attr('y', 0)
		.attr('width', fft_chart_margin)
		.attr('height', fft_chart_margin)
		.attr('fill', 'rgba(255, 255, 255, 0)');
		
	fft_axis.append('g')
		.attr('transform', 'translate(50, 0)')
		.call(function(g) {
			g.call(y_axis);
			g.selectAll(".tick line").attr("stroke", "rgba(255,255,255,255)");
		});

	var xscale = d3.scaleLinear()
		.domain([125000 / this.fft_data[ch].length, 125000])
		.range([fft_chart_margin, fft_chart_width]);
//	var xaxis = d3.axisBottom(xscale)
//		.ticks(10);
	fft_axis.append('g')
		.attr('transform', 'translate(0, 455)')
		.call(function(g) {
			g.call(x_axis);
			g.selectAll(".tick line").attr("stroke", "rgba(255,255,255,255)");
		});
	this.fft_x = fft_x;
	this.fft_y = fft_y;
	
	this.svg_selector = svg;
};

var lastUpdateDataTime = 0;
ChartRenderer.prototype.updateData = function() {
	if(this.svg_selector == null) {
		this.update();
	}
	var time = new Date().getTime()
	var elapsed =  time - lastUpdateDataTime
	if( elapsed < 16 ) return;
	lastUpdateDataTime = time

	var sample_index = this.sample_index;
	var svg = this.svg_selector;
	var ch = this.channel;

	var n = this.samples[ch].length;
	var x = d3.scaleLinear()
		.domain([0,number_of_samples])
		.range([chart_margin, chart_width]);
	var y = d3.scaleLinear()
		.domain([activeChannel[chs[ch]].minimum_value, activeChannel[chs[ch]].maximum_value])
		.range([chart_height, 0]);
	
	// Graph of summarized values.	
	var summary_area = d3.area()
		.defined(function(d, i) {return i <= sample_index; })
		.x(function(d, i){ return x(i); })
		.y0(function(d){ return (1 * y(d[1])); })
		.y1(function(d){ return (1 * y(d[0])); });
	
	var area_old = d3.area()
		.defined(function(d, i) {return i > sample_index; })
		.x(function(d, i){ return x(i); })
		.y0(function(d){ return (1 * y(d[1])); })
		.y1(function(d){ return (1 * y(d[0])); });

	var error_area = d3.area()
		.defined(function(d, i) {return i <= sample_index; })
		.x(function(d, i){ return x(i); })
		.y0(function(d){ return (1 * y(d[3])); })
		.y1(function(d){ return (1 * y(d[2])); });
	
	var waring_area = d3.area()
		.defined(function(d, i) {return i <= sample_index; })
		.x(function(d, i){ return x(i); })
		.y0(function(d){ return (1 * y(d[5])); })
		.y1(function(d){ return (1 * y(d[4])); });
	
	this.path_summary
		.datum(this.samples[ch])
		.attr('class', 'signal-current')
		.attr('d', summary_area);
	
	this.old_signal_path
		.datum(this.samples[ch])
		.attr('class', 'signal-old')
		.attr('d', area_old);
	
	this.path_error
		.datum(this.samples[ch])
		.attr('class', 'signal-error')
		.attr('d', error_area);
	
	this.path_warning
		.datum(this.samples[ch])
		.attr('class', 'signal-warning')
		.attr('d', waring_area);
	
//	var line = d3.path();
//	line.moveTo(x(chart_renderer.samples_shown % number_of_samples), y(minimum_value));
//	line.lineTo(x(chart_renderer.samples_shown % number_of_samples), y(maximum_value));	
//	line.moveTo(x(sample_index), y(minimum_value));
//	line.lineTo(x(sample_index), y(maximum_value));
//	this.scanline_path
//		.attr("stroke", "red")
//		.attr("d", line);

	var fft_x = this.fft_x;
	var fft_y = this.fft_y;

	var fft_line = d3.line()
		.x(function(d, i){ return fft_x(i); })
		.y(function(d){ return fft_y(d); });

	this.path_fft
		.data([this.fft_data[ch]])
		.attr('class', 'chart-line')
		.attr('d', fft_line);
}
ChartRenderer.prototype.clearGraph = function() {
	var svg = d3.select(this.el);
}

ChartRenderer.prototype.push = function(data, withTriggerFlg){
	
//	var n = this.samples_shown;
//	if(n >= chart_buffer_size){
//		// var svg = d3.select(this.el);
//		this.samples_shown = 0;
//		this.expected = [];
//		this.clearGraph();
//		n = 0;
//
//		for (var c = 0; c < NUM_CHANNEL; c++) {
//			//this.samples.push([]);
//			this.expected.push([]);
//		}
//	}
	if(withTriggerFlg){
		this.sample_index = 0;
	}
	for (var c = 0; c < data.length; c++) {
		let max = data[c][0];
		let min = data[c][1];
		let errorMax = data[c][2];
		let errorMin = data[c][3];
		let warningMax = data[c][4];
		let warningMin = data[c][5];
//		console.log('c:' + c);
//		console.log('data[c][0]:' + data[c][0]);
//		console.log('data[c][1]:' + data[c][1]);
//		console.log('data[c][2]:' + data[c][2]);
//		console.log('data[c][3]:' + data[c][3]);
//		console.log('data[c][4]:' + data[c][4]);
//		console.log('data[c][5]:' + data[c][5]);
		

		// if (expected_max < expected_min) {
		// 	expected_max = INVALID_EXPECT;
		// 	expected_min = INVALID_EXPECT;
		// }
		// if ((expected_max != INVALID_EXPECT && max > expected_max) ||
		//     (expected_min != INVALID_EXPECT && min < expected_min)) {
		// 	this.error_cnt[c]++;
		// }
//		this.max_data[c] = Math.max(this.max_data[c], max);
//		this.min_data[c] = Math.min(this.min_data[c], min);

		this.samples[c][this.sample_index] = [max, min, errorMax, errorMin, warningMax, warningMin];
//		this.expected[c].push([expected_max, expected_min]);
	}
	this.sample_index++;
	if( this.sample_index >= number_of_samples ) {
		this.sample_index = 0;
	}
};

ChartRenderer.prototype.set_fft = function(ch, data) {
	this.fft_data[ch] = data;
}

ChartRenderer.prototype.set_fft_expected = function(data) {
	n = data.length;
	this.fft_expected = []
	for (var i = 0; i < n; i += 2) {
		max = data[i];
		min = data[i + 1];
		if (max < min) {
			max = INVALID_EXPECT;
			min = INVALID_EXPECT;
		}
		this.fft_expected.push([max, min]);
	}
}

PingServer = function(dev, callback){
	var http = new XMLHttpRequest();
	http.open("GET", "http://" + dev.ip_address + ":2223/ping", /*async*/true);
	http.onreadystatechange = function() {
		//console.log("Ping: " + http.readyState + " " + http.status);
		if (http.readyState == 4 && http.status == 200) {
			if (dev.pingtimer != null) {
				clearTimeout(dev.pingtimer);
				dev.pingtimer = null;
			}
			if (callback != null) {
				callback(dev);
			}
		}
	};
	http.send(null);
}

PoweroffServer = function(dev, callback){
	var http = new XMLHttpRequest();
	http.open("GET", "http://" + dev.ip_address + ":2223/poweroff", /*async*/true);
	http.onreadystatechange = function() {
		//console.log("Poweroff: " + http.readyState + " " + http.status);
		if (http.readyState == 4 && http.status == 200) {
			if (callback != null) {
				callback(dev);
			}
		}
	};
	http.send(null);
}

SetTimeServer = function(dev, callback){

	var d= new Date();
	utc = d.getTime() + (d.getTimezoneOffset() * 60000);
	var jikan = new Date(utc);

	jikan.toUTCString();
	var hour = jikan.getHours();
	var minute = jikan.getMinutes();
	var second = jikan.getSeconds();
	var year = jikan.getFullYear();
	var month = jikan.getMonth()+10;
	var day = jikan.getDate();

	var http = new XMLHttpRequest();
	http.open("GET", "http://" + dev.ip_address + ":2223/time/" + year + "." + month + "." + day + "-" + hour + ":" + minute + ":" + second , /*async*/true);
	http.onreadystatechange = function() {
	console.log("SetTime: " + http.readyState + " " + http.status);
	if (http.readyState == 4 && http.status == 200) {
		if (dev.pingtimer != null) {
			clearTimeout(dev.pingtimer);
			dev.pingtimer = null;
		}
		if (callback != null) {
			callback(dev);
		}
	}
	};
	http.send(null);
}

SetGainAmp = function(dev, callback){
	var space = " ";
	var gain_value = dev.gain; //gain_value
	if(dev.gain == "x1"){
		gain_value = 17;   //0x11
	}else if(dev.gain == "x4"){
		gain_value = 34;   //0x22
        }else if(dev.gain == "x10"){
                gain_value = 10;   //0x22
	}else if(dev.gain == "x25"){
		gain_value = 51;   //0x33
	}else if(dev.gain == "x100"){
		gain_value = 68;   //0x44
	}else if(dev.gain == "x400"){
		gain_value = 85;   //0x55
        }else if(dev.gain == "x1000"){
                gain_value = 1000;   //0x22
	}else if(dev.gain == "x2500"){
		gain_value = 102;   //0x66
	}else if(dev.gain == "x10000"){
		gain_value = 119;   //0x77
	}
	var http = new XMLHttpRequest();
	http.open("GET", "http://" + dev.ip_address + ":2223/pgacontrol/" + gain_value  , /*async*/true);

	http.onreadystatechange = function() {
		console.log("SetGain: " + http.readyState + " " + http.status);
		if (http.readyState == 4 && http.status == 200) {
			if (dev.pingtimer != null) {
				clearTimeout(dev.pingtimer);
				dev.pingtimer = null;
			}
			if (callback != null) {
				callback(dev);
			}
		}
	};
	http.send(null);
}

var resize_chart = function(){
	
	var chart = $('#chart')
	var w = chart.parent().width();
	chart.attr('width', w);
	chart.attr('height', w * chart_height / chart_width);

	var fft_chart = $('#fft-chart')
	w = fft_chart.parent().width();
	fft_chart.attr('width', w);
	fft_chart.attr('height', w / fft_chart_width * fft_chart_height);
}
$(window).on('resize', resize_chart);

var start = function(){
	
	resize_chart();
	var devices = [];
	
	//## Olive側の設定取得API
	var apiServer = 'http://' + location.hostname + ':2223';
	var acqSettingApi = apiServer + '/acquisition/settings/get';
	var acqConfigApi = apiServer + '/acquisition/config/get';
	var mntSettingApi = apiServer + '/monitoring/settings/get';
	var mlSettingApi = apiServer + '/ml/settings';
	
	//### アクティブチャネルを取得する
	var http = new XMLHttpRequest();
	http.onreadystatechange = function(){
		if(http.status == 200 && http.readyState == 4){

			var data = JSON.parse(http.responseText);
			activeChannel = data.channels;
			channelTrigger = data.trigger;
			http.abort();

			//### チャネル毎の最大値/最小値を取得する
			var httpC = new XMLHttpRequest();
			httpC.onreadystatechange = function(){
				if(httpC.status == 200 && httpC.readyState == 4){

					var dataC = JSON.parse(httpC.responseText);
					var range = dataC.ranges;
					var i = 0;
					Object.keys(activeChannel).forEach(function(key) {
						chs[i] = key;
						i++;
						activeChannel[key].chName = key;
						activeChannel[key].minimum_voltage = range[activeChannel[key].range].minimum_voltage;
						activeChannel[key].maximum_voltage = range[activeChannel[key].range].maximum_voltage;
						activeChannel[key].minimum_value = range[activeChannel[key].range].minimum_value;
						activeChannel[key].maximum_value = range[activeChannel[key].range].maximum_value;
					});
					console.log('activeChannel:');
					console.log(activeChannel);
					httpC.abort();
					
					//### モニタリング設定を取得する
					var httpM = new XMLHttpRequest();
					httpM.onreadystatechange = function(){
						if(httpM.status == 200 && httpM.readyState == 4){

							mntSetting = JSON.parse(httpM.responseText);
							console.log('mntSetting:');
							console.log(mntSetting);
							httpM.abort();
							
							devices.push({
								ip_address: location.hostname,
								realtime_conn: new RealtimeConnection(location.hostname),
							});
							
							number_of_samples = mntSetting.horizontal_points;
							
							chart_renderer = new ChartRenderer('#chart', '#fft-chart');
							svg = d3.select(chart_renderer.el);
							setScale();
							initAxis();
							initTriggerThreshold();
						}
						else{
							console.log('getMonitorSetting: ' + mntSettingApi + ' - Connection Faild. ['+ httpM.status +']');
						}
					}
					httpM.open('GET', mntSettingApi);
					httpM.send(null);
					
				}
				else{
					console.log('getOliveConfig: ' + acqConfigApi + ' - Connection Faild. ['+ httpC.status +']');
				}
			}
			httpC.open('GET', acqConfigApi);
			httpC.send(null);
		}
		else{
			console.log('getOliveSetting: ' + acqSettingApi + ' - Connection Faild. ['+ http.status +']');
		}
	}
	http.open('GET', acqSettingApi);
	http.send(null);
	
	window.parent.document.getElementById('channel').onchange = function chCng(){
		chart_renderer.channel = chs.indexOf(window.parent.document.getElementById('channel').value.replace('string:', ''));
		setScale();
		updateAxis();
	};
	
	window.parent.document.getElementById('changeChTrigger').onclick = function chTrgCng(){
		
		const target = window.parent.document.getElementById('procTrgCh');
		const observer = new MutationObserver(function(){

			var http = new XMLHttpRequest();
			http.onreadystatechange = function(){
				if(http.status == 200 && http.readyState == 4){

					var data = JSON.parse(http.responseText);
					channelTrigger = data.trigger;

					setScale();
					updateTriggerThreshold();
					http.abort();
					observer.disconnect();
				}
				else{
					console.log('getOliveSetting: ' + acqSettingApi + ' - Connection Faild. ['+ http.status +']');
				}
			}
			http.open('GET', acqSettingApi);
			http.send(null);
		});
		const config = {attributes: true};
		observer.observe(target, config);
	};
	
	if(window.parent.document.getElementById('setEffectiveRange') != null) {
		window.parent.document.getElementById('setEffectiveRange').onclick = function setEffectiveRange(){

			if(usrX0 + usrX1 == 0){
				return;
			}

			var http = new XMLHttpRequest();
			http.onreadystatechange = function(){
				if(http.status == 200 && http.readyState == 4){

					var mlSetting = JSON.parse(http.responseText);
					http.abort();

					mlSetting.input_data_offset = usrX0;
					mlSetting.input_data_size = usrX1 - usrX0;

					var ch = chs[chart_renderer.channel]; 
	//				if(mlSetting.target_channels.indexOf(ch) < 0){
	//					var chIdx = mlSetting.target_channels.length;
	//					mlSetting.target_channels[chIdx] = ch;
	//				}
					mlSetting.target_channels[0] = ch;
	//				console.log('mlSetting:');
	//				console.log(mlSetting);

					var httpP = new XMLHttpRequest();
					httpP.onreadystatechange = function(){
						if(httpP.status == 200 && httpP.readyState == 4){
							console.log('setMlSetting: ' + mlSettingApi + '/set' + ' - complete. ['+ httpP.responseText +']');
							httpP.abort();
							window.parent.document.getElementById('currentRange').value = 'Current: ' + usrX0 + ' to ' + usrX1;
						}
						else{
							console.log('setMlSetting: ' + mlSettingApi + '/set' + ' - Connection Faild. ['+ httpP.status +']');
							if(httpP.status == 400){
								httpP.abort();
							}
						}
					}
					httpP.open('POST', mlSettingApi + '/set');
					httpP.setRequestHeader('Content-Type', 'application/json');
					httpP.send(JSON.stringify(mlSetting));
				}
				else{
					console.log('getMlSetting: ' + mlSettingApi + '/get' + ' - Connection Faild. ['+ http.status +']');
				}
			}
			http.open('GET', mlSettingApi + '/get');
			http.send(null);
		}
	}
}
$(document).ready(start);