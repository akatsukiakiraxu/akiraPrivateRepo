var chart_width = 1000;
var chart_height = 500;
var chart_margin = 50;
var chart_buffer_size = 2000;
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
	this.update();
};

ChartRenderer.prototype.update = function(){


var n = 20, // number of layers
    m = 200, // number of samples per layer
    k = 10; // number of bumps per layer

var stack = d3.stack().keys(d3.range(n)).offset(d3.stackOffsetWiggle),
    //layers0 = stack(d3.transpose(d3.range(n).map(function() { return bumps(m, k); }))),
    layers0 = [1, 11],[2, 12],[3, 13],[4, 14],[5, 15],[6, 16],[7, 17],[8, 18],[9, 19],[10, 20],[11, 21],[12, 22],[13, 23],[14, 24],[15, 25],[16, 26],[17, 27],[18, 28],[19, 29],[20, 30],[21, 31],[22, 32],[23, 33],[24, 34],[25, 35],[26, 36],[27, 37],[28, 38],[29, 39],[30, 40],[31, 41],[32, 42],[33, 43],[34, 44],[35, 45],[36, 46],[37, 47],[38, 48],[39, 49],[40, 50],[41, 51],[42, 52],[43, 53],[44, 54],[45, 55],[46, 56],[47, 57],[48, 58],[49, 59],[50, 60],[51, 61],[52, 62],[53, 63],[54, 64],[55, 65],[56, 66],[57, 67],[58, 68],[59, 69],[60, 70],[61, 71],[62, 72],[63, 73],[64, 74],[65, 75],[66, 76],[67, 77],[68, 78],[69, 79],[70, 80],[71, 81],[72, 82],[73, 83],[74, 84],[75, 85],[76, 86],[77, 87],[78, 88],[79, 89],[80, 90],[81, 91],[82, 92],[83, 93],[84, 94],[85, 95],[86, 96],[87, 97],[88, 98],[89, 99],[90, 100],[91, 101],[92, 102],[93, 103],[94, 104],[95, 105],[96, 106],[97, 107],[98, 108],[99, 109],[100, 110],[101, 111],[102, 112],[103, 113],[104, 114],[105, 115],[106, 116],[107, 117],[108, 118],[109, 119],[110, 120],[111, 121],[112, 122],[113, 123],[114, 124],[115, 125],[116, 126],[117, 127],[118, 128],[119, 129],[120, 130],[121, 131],[122, 132],[123, 133],[124, 134],[125, 135],[126, 136],[127, 137],[128, 138],[129, 139],[130, 140],[131, 141],[132, 142],[133, 143],[134, 144],[135, 145],[136, 146],[137, 147],[138, 148],[139, 149],[140, 150],[141, 151],[142, 152],[143, 153],[144, 154],[145, 155],[146, 156],[147, 157],[148, 158],[149, 159],[150, 160],[151, 161],[152, 162],[153, 163],[154, 164],[155, 165],[156, 166],[157, 167],[158, 168],[159, 169],[160, 170],[161, 171],[162, 172],[163, 173],[164, 174],[165, 175],[166, 176],[167, 177],[168, 178],[169, 179],[170, 180],[171, 181],[172, 182],[173, 183],[174, 184],[175, 185],[176, 186],[177, 187],[178, 188],[179, 189],[180, 190],[181, 191],[182, 192],[183, 193],[184, 194],[185, 195],[186, 196],[187, 197],[188, 198],[189, 199],[190, 200],[191, 201],[192, 202],[193, 203],[194, 204],[195, 205],[196, 206],[197, 207],[198, 208],[199, 209],[200, 210];


    layers1 = stack(d3.transpose(d3.range(n).map(function() { return bumps(m, k); }))),
    layers = layers0.concat(layers1);

//var svg = d3.select("svg"),
var svg = d3.select(this.el),
    width = +svg.attr("width"),
    height = +svg.attr("height");

var x = d3.scaleLinear()
    .domain([0, m - 1])
    .range([0, width]);

var y = d3.scaleLinear()
    .domain([d3.min(layers, stackMin), d3.max(layers, stackMax)])
    .range([height, 0]);

var z = d3.interpolateCool;

var area = d3.area()
    .x(function(d, i) { return x(i); })
    .y0(function(d) { return y(d[0]); })
    .y1(function(d) { return y(d[1]); });

svg.selectAll("path")
  .data(layers0)
  .enter().append("path")
    .attr("d", area)
    .attr("fill", function() { return z(Math.random()); });

function stackMax(layer) {
  return d3.max(layer, function(d) { return d[1]; });
}

function stackMin(layer) {
  return d3.min(layer, function(d) { return d[0]; });
}

function transition() {
  var t;
  d3.selectAll("path")
    .data((t = layers1, layers1 = layers0, layers0 = t))
    .transition()
      .duration(2500)
      .attr("d", area);
}

// Inspired by Lee Byron?fs test data generator.
function bumps(n, m) {
  var a = [], i;
  for (i = 0; i < n; ++i) a[i] = 0;
  for (i = 0; i < m; ++i) bump(a, n);
  return a;
}

function bump(a, n) {
  var x = 1 / (0.1 + Math.random()),
      y = 2 * Math.random() - 0.5,
      z = 10 / (0.1 + Math.random());
  for (var i = 0; i < n; i++) {
    var w = (i / n - y) * z;
    a[i] += x * Math.exp(-w * w);
  }
}




};

ChartRenderer.prototype.push = function(x){
	var n = this.samples.length;
	var m = x.length
	if(n + m > chart_buffer_size){
		this.samples.splice(0, n + m - chart_buffer_size);
	}
	this.samples = this.samples.concat(x);
};

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
    var month = jikan.getMonth()+1;
    var day = jikan.getDate();

    var http = new XMLHttpRequest();
    http.open("GET", "http://" + dev.ip_address + ":2223/time/" + year + "." + month + "." + day + "-" + hour + ":" + minute + ":" + second , /*async*/true);
    //http.open("GET", "http://" + dev.ip_address + ":2223/time/2017.04.25-12:44:00", /*async*/true);
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
						//ip_address: "10.0.1.101",
						ip_address: "127.0.0.1",
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
