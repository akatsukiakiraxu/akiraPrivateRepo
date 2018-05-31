(function () {
	'use strict';

	angular.module('OliveAtFactory.pages.progress')
		.controller('progressCtrl', progressCtrl);

	/** @ngInject */
	function progressCtrl($scope, $timeout, $http, $location, $websocket, $window) {
		
		//# Olive側の機械学習の進捗を取得し、リアルタイム表示する
		var dataStream = $websocket('ws://' + $location.host() + ':2222/ml/event');
		var res;
		$scope.mlEvent = {
			collection: [],
			get: function() {
				dataStream.send(JSON.stringify({ action: 'get' }));
			}
		};

		dataStream.onMessage(function(message) {
			
			res = JSON.parse(message.data);
			if(res.type == 'training-done' || res.type == 'inference-result'){
				$window.location.href = '/#/results';
				dataStream.close();
			}
			else if(res.type == 'inference-result'){
				res = {type: "training-progress", trained_count: 256};	//// dummy...
			}
			$scope.mlEvent.collection.push(res);
			///console.log(res);
		});

		dataStream.onError(function(){
			console.log('[connect error] ws://' + $location.host() + ':2222/ml/event');
			$window.location.href = '/#/results';
		});
			
		//# Olive側の現在の機械学習設定を取得する
		$http({method: 'GET', url: 'http://' + $location.host() + ':2223/ml/settings/get'})
			.success(function(data, status, headers, config) {
			$scope.mlCnt = data.training_count;
        })
        	.error(function(data, status, headers, config) {
			$scope.mlCnt = 0;
		});
	}
})();