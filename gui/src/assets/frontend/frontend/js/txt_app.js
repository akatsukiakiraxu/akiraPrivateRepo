var chart_width = 1000;
var chart_height = 500;
var chart_margin = 50;
//var chart_buffer_size = 2000;
var chart_buffer_size = 400;
var chart_renderer = null;

var fft_sampleRate = 1000000 / 8192;	// 1Msps / 8192 = 122sps
var fft_canvas_width = 1000;		// FFT window (canvas) width
var fft_canvas_height = 256;		// FFT window (canvas) height
//var fft_points = fft_canvas_height * 2;
var fft_gain=45;
var fft_floor = 40;
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

var area, cirY=50, cirX=50;

RealtimeConnection = function(addr){
	var self = this;
	this.onmessage = null;
	this.connection = new WebSocket('ws://' + addr + ':2222/realtime');
	this.connection.binaryType = 'arraybuffer';
	this.connection.onmessage = function(e){
		if(self.onmessage !== null){
			var s = new Uint8Array(e.data);
			var d = []
			for(var i = 0; i + 3 < s.length; i += 4){
				d.push(
					(s[i + 0] << 24) |
					(s[i + 1] << 16) |
					(s[i + 2] <<  8) |
					(s[i + 3] <<  0));
			}
			self.onmessage(d);
		}
	};
};

RealtimeConnection.prototype.close = function(){
	this.connection.close();
};


ChartRenderer = function(el){
	this.el = el;
	this.samples = []
	//this.layers0 = [[[]]]
	//this.layers1 = [[[]]]
	this.layers0 = []
	this.layers1 = []	
	this.layers0[0] = []
	this.layers1[0] = []
	for (var i = 0; i < chart_buffer_size; i++) {
		this.layers0[0][i] = [0xffffffff,0];
		//this.layers1[0][i] = [0xffffffff,0];
		this.layers1[0][i] = [2048,2048];
		//this.layers1[0][i] = [1948,2148];
	}
	this.layers0=[[]]
	//this.layers0 = [][][]
	//this.layers1 = [][][]
    //this.layers0 = [[[1, 11],[2, 12],[3, 13],[4, 14],[5, 15],[6, 16],[7, 17],[8, 18],[9, 19],[10, 20],[11, 21],[12, 22],[13, 23],[14, 24],[15, 25],[16, 26],[17, 27],[18, 28],[19, 29],[20, 30],[21, 31],[22, 32],[23, 33],[24, 34],[25, 35],[26, 36],[27, 37],[28, 38],[29, 39],[30, 40],[31, 41],[32, 42],[33, 43],[34, 44],[35, 45],[36, 46],[37, 47],[38, 48],[39, 49],[40, 50],[41, 51],[42, 52],[43, 53],[44, 54],[45, 55],[46, 56],[47, 57],[48, 58],[49, 59],[50, 60],[51, 61],[52, 62],[53, 63],[54, 64],[55, 65],[56, 66],[57, 67],[58, 68],[59, 69],[60, 70],[61, 71],[62, 72],[63, 73],[64, 74],[65, 75],[66, 76],[67, 77],[68, 78],[69, 79],[70, 80],[71, 81],[72, 82],[73, 83],[74, 84],[75, 85],[76, 86],[77, 87],[78, 88],[79, 89],[80, 90],[81, 91],[82, 92],[83, 93],[84, 94],[85, 95],[86, 96],[87, 97],[88, 98],[89, 99],[90, 100],[91, 101],[92, 102],[93, 103],[94, 104],[95, 105],[96, 106],[97, 107],[98, 108],[99, 109],[100, 110],[101, 111],[102, 112],[103, 113],[104, 114],[105, 115],[106, 116],[107, 117],[108, 118],[109, 119],[110, 120],[111, 121],[112, 122],[113, 123],[114, 124],[115, 125],[116, 126],[117, 127],[118, 128],[119, 129],[120, 130],[121, 131],[122, 132],[123, 133],[124, 134],[125, 135],[126, 136],[127, 137],[128, 138],[129, 139],[130, 140],[131, 141],[132, 142],[133, 143],[134, 144],[135, 145],[136, 146],[137, 147],[138, 148],[139, 149],[140, 150],[141, 151],[142, 152],[143, 153],[144, 154],[145, 155],[146, 156],[147, 157],[148, 158],[149, 159],[150, 160],[151, 161],[152, 162],[153, 163],[154, 164],[155, 165],[156, 166],[157, 167],[158, 168],[159, 169],[160, 170],[161, 171],[162, 172],[163, 173],[164, 174],[165, 175],[166, 176],[167, 177],[168, 178],[169, 179],[170, 180],[171, 181],[172, 182],[173, 183],[174, 184],[175, 185],[176, 186],[177, 187],[178, 188],[179, 189],[180, 190],[181, 191],[182, 192],[183, 193],[184, 194],[185, 195],[186, 196],[187, 197],[188, 198],[189, 199],[190, 200],[191, 201],[192, 202],[193, 203],[194, 204],[195, 205],[196, 206],[197, 207],[198, 208],[199, 209],[200, 210]]];
    //this.layers1 = [[[10, 1010],[2, 102],[3, 103],[4, 104],[5, 105],[6, 106],[7, 107],[8, 108],[9, 109],[100, 20],[1010, 210],[102, 22],[103, 23],[104, 24],[105, 25],[106, 26],[107, 27],[108, 28],[109, 29],[20, 30],[210, 310],[22, 32],[23, 33],[24, 34],[25, 35],[26, 36],[27, 37],[28, 38],[29, 39],[30, 40],[310, 410],[32, 42],[33, 43],[34, 44],[35, 45],[36, 46],[37, 47],[38, 48],[39, 49],[40, 50],[410, 510],[42, 52],[43, 53],[44, 54],[45, 55],[46, 56],[47, 57],[48, 58],[49, 59],[50, 60],[510, 610],[52, 62],[53, 63],[54, 64],[55, 65],[56, 66],[57, 67],[58, 68],[59, 69],[60, 70],[610, 710],[62, 72],[63, 73],[64, 74],[65, 75],[66, 76],[67, 77],[68, 78],[69, 79],[70, 80],[710, 810],[72, 82],[73, 83],[74, 84],[75, 85],[76, 86],[77, 87],[78, 88],[79, 89],[80, 90],[810, 910],[82, 92],[83, 93],[84, 94],[85, 95],[86, 96],[87, 97],[88, 98],[89, 99],[90, 1000],[910, 10010],[92, 1002],[93, 1003],[94, 1004],[95, 1005],[96, 1006],[97, 1007],[98, 1008],[99, 1009],[1000, 10100],[10010, 101010],[1002, 10102],[1003, 10103],[1004, 10104],[1005, 10105],[1006, 10106],[1007, 10107],[1008, 10108],[1009, 10109],[10100, 1020],[101010, 10210],[10102, 1022],[10103, 1023],[10104, 1024],[10105, 1025],[10106, 1026],[10107, 1027],[10108, 1028],[10109, 1029],[1020, 1030],[10210, 10310],[1022, 1032],[1023, 1033],[1024, 1034],[1025, 1035],[1026, 1036],[1027, 1037],[1028, 1038],[1029, 1039],[1030, 1040],[10310, 10410],[1032, 1042],[1033, 1043],[1034, 1044],[1035, 1045],[1036, 1046],[1037, 1047],[1038, 1048],[1039, 1049],[1040, 1050],[10410, 10510],[1042, 1052],[1043, 1053],[1044, 1054],[1045, 1055],[1046, 1056],[1047, 1057],[1048, 1058],[1049, 1059],[1050, 1060],[10510, 10610],[1052, 1062],[1053, 1063],[1054, 1064],[1055, 1065],[1056, 1066],[1057, 1067],[1058, 1068],[1059, 1069],[1060, 1070],[10610, 10710],[1062, 1072],[1063, 1073],[1064, 1074],[1065, 1075],[1066, 1076],[1067, 1077],[1068, 1078],[1069, 1079],[1070, 1080],[10710, 10810],[1072, 1082],[1073, 1083],[1074, 1084],[1075, 1085],[1076, 1086],[1077, 1087],[1078, 1088],[1079, 1089],[1080, 1090],[10810, 10910],[1082, 1092],[1083, 1093],[1084, 1094],[1085, 1095],[1086, 1096],[1087, 1097],[1088, 1098],[1089, 1099],[1090, 200],[10910, 2010],[1092, 202],[1093, 203],[1094, 204],[1095, 205],[1096, 206],[1097, 207],[1098, 208],[1099, 209],[200, 2100]]];
	//this.area
	this.update();
};

ChartRenderer.prototype.update = function(){

    var svg = d3.select(this.el)
//        .append('svg');

    // クリップする円の定義
    /*
    var defs = svg.append('defs');
    var circles = defs
        .append('circle')
        .attr('id', 'circle')
        .attr('r', 50)
        .attr('cx', 100)
        .attr('cy', 100);

    defs.append('clipPath')
        .attr('id', 'clip')
        .append('use')
        .attr('xlink:href', '#circle');
    */

///circle
//  var svg = d3.select("#example").append("svg")
//      .attr({
//        width: 640,
//        height: 480
//      });
//
//  // 座標(cx,cy)と半径(r)を指定
//  var c1 = [100, 90, 30];

//  // dataの挿入方法が独特なので注意が必要
//  // 詳しくは、[三つの小円](http://ja.d3js.node.ws/document/tutorial/circle.html)参照
//  var circle = svg.selectAll('circle').data([c1]).enter().append('circle')
//    .attr({
//      // enterに入っているデータ一つ一つで下の処理を行う
//      'cx': function(d) { return d[0]; },
//      'cy': function(d) { return d[1]; },
//      'r': function(d) { return d[2]; },
//    });

/*
d3.select("#result")	// ID名resultの要素を指定
	.append("svg")	// svg要素を追加
	.attr("width", 320)	// svg要素の横幅を指定
	.attr("height", 240)	// svg要素の縦幅を指定
	.append("text")	// 楕円を追加。以後のメソッドは、この楕円に対しての設定になる
		.attr("x", 50)	// x座標を指定
		.attr("y", 100)	// y座標を指定
		.text("テキストを表示できます")	// プレーンテキストを表示
*/

  // text label for the x axis

  svg.append("text")             
      .attr("transform",
            "translate(" + (300/2) + " ," + 
                           (200 + 20) + ")")
      .style("text-anchor", "middle")
      .text("Date");

svg.append("text")
    .attr("class", "x label")
    .attr("text-anchor", "end")
    .attr("x", 100)
    .attr("y", 200 - 6)
    .text("income per capita, inflation-adjusted (dollars)");


};


function stackMax(layer) {
  return d3.max(this.samples, function(d) { return d; });
}

function stackMin(layer) {
  return d3.min(this.samples, function(d) { return d; });
}



ChartRenderer.prototype.push = function(x){
	var n = this.samples.length;
	var m = x.length
	if(n + m > chart_buffer_size){
		//this.samples.splice(0, n + m - chart_buffer_size);
		this.samples=[];
		//this.layers0=[[[]]];
		  var t;
		  //d3.selectAll("path")
		  //.data((t = this.layers1, this.layers1 = this.layers0, this.layers0 = t))
		  ////  .data(t = layers1)
		  //  .transition()
		  //    .duration(2500)
		  //    .attr("d", area);
	}
	this.samples = this.samples.concat(x);
	//var max_sample = (this.samples[x] & 0x0000FFFF ) + 400;
	//var min_sample = (this.samples[x] >> 106 ) + 800;
	var max_sample = (x[0]& 0x0000FFFF ) - 100;
	var min_sample = (x[0] >> 16 ) +100;
	this.layers0[0].push([max_sample, min_sample]);
	var max_sample = (x[1]& 0x0000FFFF ) - 100;
	var min_sample = (x[1] >> 16 ) +100;
	this.layers0[0].push([max_sample, min_sample]);
	var max_sample = (x[2]& 0x0000FFFF ) - 100;
	var min_sample = (x[2] >> 16 ) +100;
	this.layers0[0].push([max_sample, min_sample]);
	var max_sample = (x[3]& 0x0000FFFF ) - 100;
	var min_sample = (x[3] >> 16 ) +100;
	this.layers0[0].push([max_sample, min_sample]);

	//console.log("data: " + this.layers0[0][this.layers0[0].length -1][0]);

};

PingServer = function(dev, callback){
    var http = new XMLHttpRequest();
    http.open("GET", "http://" + dev.ip_address + ":2223/ping", /*async*/true);
    //http.open("GET", "http://10.0.1.101:2223/ping", /*async*/true);
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
    //http.open("GET", "http://10.0.1.101:2223/poweroff", /*async*/true);
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
    //http.open("GET", "http://" + dev.ip_address + ":2223/time/20107.04.25-12:44:00", /*async*/true);
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
var device_list = new Vue({
	el: '#device-list',
	components: VueMdl.components,
	directives: VueMdl.directives,
	data: {
		selected: '',
		devices: []
	},
	methods: {
		reload: function(){
			var self = this;
			target = null;
			//$.getJSON('devices.json', function(data){
				self.devices = [];
				//data.forEach(function(x, i){
					self.devices.push({
						index: 0,
						//name: x['hostname'],
						name: "Olive-C0000F",
						//ip_address: x['ip'],
						//ip_address: "10.0.0.100",
						ip_address: "10.0.1.101",
						//ip_address: "127.0.0.1",
						//ip_address: "10.0.1.102",
						//mac_address: x['mac'],
						mac_address: "70:F8:E7:C0:00:0F",
						//status: 'online',
						status: 'offline',
						action: 'N/A',
						power: 'ON',
						realtime_conn: null,
						pingtimer: null
					});
				//});
			//});
			self.devices.forEach(function(device, i){
			    setInterval(function(dev){
				if (dev.pingtimer != null)
				    clearTimeout(dev.pingtimer);
				dev.pingtimer = setTimeout(function(d){
				    d.action = 'N/A';
				    d.status = 'offline'
				    if(d.realtime_conn !== null){
					d.realtime_conn.close();
					d.realtime_conn = null;
				    }
				}, 500, dev);
				if (dev.action == 'N/A' ||
				    dev.action == 'START') {
				    PingServer(dev, function(d){
					d.action = 'START';
					d.status = 'online'
					d.power = 'ON';
				    });
				    //SetTimeServer(dev, function(d){
					//d.status = 'online'
				    //});

				} else if (dev.action == 'STOP') {
				    PingServer(dev, function(d){
					d.status = 'data acquiring';
				    });
				}
			    }.bind(this), 1000, device);
			});
		},
		doStartStop: function () {
			if (target == null)
				return;
			if (target.action == 'START') {
				target.action = 'STOP';
				target.status = 'data acquiring';
				var ws = new RealtimeConnection(target.ip_address);
				ws.onmessage = function(d){
					chart_renderer.push(d);
					chart_renderer.update();
				};
				target.realtime_conn = ws;
			} else if (target.action == 'STOP') {
				target.action = 'START';
				target.status = 'online'
				if(target.realtime_conn !== null){
					target.realtime_conn.close();
					target.realtime_conn = null;
				}
			}
		},
		doPoweroff: function () {
			if (target.power = 'OFF' &&
			    target.action == 'START' && target.status == 'online') {
				PoweroffServer(target, function(d){
				    d.power = 'OFF';
				});
			}
		}
	},
	watch: {
		selected: function(val){
			chart_renderer = new ChartRenderer('#chart');
			target = this.devices[val];
			console.log("Selected: " + val + " ip: " + target.ip_address);
		}
	},
});

device_list.reload();

var resize_chart = function(){
	var chart = $('#chart')
	var w = chart.parent().width();
	chart.attr('width', w);
	chart.attr('height', w * chart_height / chart_width);
}
$(window).on('resize', resize_chart);
$(document).ready(resize_chart);
