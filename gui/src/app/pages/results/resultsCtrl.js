(function () {
  'use strict';

	angular.module('OliveAtFactory.pages.results')
		.factory('resultsFactory', resultsFactory).controller('resultsCtrl', resultsCtrl);

	/** @ngInject */
	function resultsFactory($websocket, $window, $location){
		
		var dataStream = $websocket('ws://' + $location.host() + ':2222/ml/event');
		var collection = [];
		var res;

		dataStream.onMessage(function(message) {
			res = JSON.parse(message.data);		
			collection.push(res);
			////console.log(res);
		});

		var methods = {
			collection: collection,
			get: function() {
				dataStream.send(JSON.stringify({ action: 'get' }));
			}
		};
		return methods;
	}
	
	function resultsCtrl($scope, $location, $http, $window, resultsFactory) {
		
		var apiAddress = 'http://' + $location.host() + ':2223';
		$scope.recChoices = [
			{id: 0, label: 'Record, always'},
			{id: 1, label: 'Record, exceeded only'},
		];
		$scope.recChoice = $scope.recChoices[0];
		$scope.recTmgChoices = [
			{id: 0, label: 'Invalid'},
			{id: 10*60, label: '10 minutes'},
			{id: 20*60, label: '20 minutes'},
			{id: 30*60, label: '30 minutes'},
			{id: 1*60*60, label: '1 hour'},
			{id: 2*60*60, label: '2 hours'},
			{id: 3*60*60, label: '3 hours'},
			{id: 6*60*60, label: '6 hours'},
			{id: 12*60*60, label: '12 hours'},
			{id: 24*60*60, label: '24 hours'},
		];
		$scope.recTmgChoice = $scope.recTmgChoices[0];
		$scope.trgChChoices = [];
		$scope.trgChChoice = {};
		$scope.trgRngChoices = [];
		$scope.trgMdChoices = [
			{id: 'disabled', label: 'disabled'},
			{id: 'rising', label: 'rising'},
			{id: 'falling', label: 'falling'},
			{id: 'both', label: 'both'}
		];
		$scope.trgMdChoice = {};
		$scope.flgTrgCh = -1;
		$scope.mlEvent = resultsFactory;
		
		var getRecordSetting = function(){
			
			$http({method: 'GET', url: apiAddress + '/recording/settings/get'})
				.success(function(data, status, headers, config) {
				
				$scope.recSetting = data;
				if($scope.recSetting.store_by_loss_value > 0){
					$scope.recChoice = $scope.recChoices[1];
				}
				for(var i = 0; i < $scope.recTmgChoices.length; i++) {
					if($scope.recTmgChoices[i]['id'] == $scope.recSetting.store_every_seconds) {
						$scope.recTmgChoice = $scope.recTmgChoices[i];
						break;
					}
				}
				
			})
				.error(function(data, status, headers, config) {
					console.log('error');
			});
		};
		
		//# Olive側の現在の機械学習設定を取得する
		$http({method: 'GET', url: apiAddress + '/ml/settings/get'})
			.success(function(data, status, headers, config) {
			
			$scope.mlSetting = data;
			$scope.mlSetting.loss_threshold = 0.8;	//// for demo
			
			//# Olive側の現在の設定を取得する
			$http({method: 'GET', url: apiAddress + '/acquisition/settings/get'})
				.success(function(data, status, headers, config) {

				$scope.setting = data;
				
				//### 閾値トリガーのチャンネル名	
				var idx = 0;
				var arySize = $scope.setting.channels.length;
				$scope.trgChChoices = {};
				angular.forEach($scope.setting.channels, function(value, key){
					$scope.trgChChoices[idx] = {};
					$scope.trgChChoices[idx]['id'] = key;
					$scope.trgChChoices[idx]['label'] = key;
					$scope.trgChChoice = $scope.trgChChoices[0];
					if(key == $scope.setting.trigger.channel_name){
					   $scope.trgChChoice = $scope.trgChChoices[idx];
					}
					idx++;
				});
				$scope.trgChChoices[idx] = {};
				$scope.trgChChoices[idx]['id'] = '';
				$scope.trgChChoices[idx]['label'] = '(No value)';
				//console.log($scope.trgChChoices);
				
				//# Olive側のハードウェアconfigを取得する
				$http({method: 'GET', url: apiAddress + '/acquisition/config/get'})
					.success(function(data, status, headers, config) {

					$scope.config = data;

					//### 閾値トリガーの電圧レンジの範囲
					$scope.trgRngChoices = new Array($scope.setting.channels.length);
					angular.forEach($scope.setting.channels, function(value, key){
						$scope.trgRngChoices[key] = {};
						$scope.trgRngChoices[key]['min'] = $scope.config.ranges[value.range].minimum_voltage;
						$scope.trgRngChoices[key]['max'] = $scope.config.ranges[value.range].maximum_voltage;
						$scope.trgRngChoices[key]['val'] = 0;
						if(key == $scope.setting.trigger.channel_name){
							$scope.trgRngChoices[key]['val'] = $scope.setting.trigger.threshold;
						}
					});
					//console.log($scope.trgRngChoices);

					//### 閾値トリガーの方向
					for(var i=0; i<$scope.trgMdChoices.length; i++){
						if($scope.trgMdChoices[i]['id'] == $scope.setting.trigger.trigger_mode){
							$scope.trgMdChoice = $scope.trgMdChoices[i];
							break;
						}
					}
					if($scope.setting.trigger.trigger_mode == 'disabled'){
						$scope.trgChChoice = $scope.trgChChoices[idx];
					}
				})
					.error(function(data, status, headers, config) {
						console.log('error');
				});
				
			})
				.error(function(data, status, headers, config) {
					console.log('error');
			});
        })
        	.error(function(data, status, headers, config) {
				console.log('error');
		});
		
		//# Olive側のログ設定を取得する
		getRecordSetting();
		
		//# ファイル出力のタイマーを変更する
		$scope.changeRecordTiming = function(recTiming) {

			$scope.recSetting.store_every_seconds = recTiming.id;
			console.log('POST /recording/settings/set - data:');
			console.log($scope.recSetting);
			$http({method: 'POST', url: apiAddress + '/recording/settings/set', data: $scope.recSetting})
			.success(function(data, status, headers, config) {
				console.log('success:' + data);
				getRecordSetting();
        	})
        	.error(function(data, status, headers, config) {
				console.log('error:' + data);
			});
		};
		
		//# 閾値トリガーの設定を変更する
		$scope.changeTriggerChannel = function(channel, threshold, mode){
			
			$scope.setting.trigger.channel_name = '';
			$scope.setting.trigger.threshold = 0;
			$scope.setting.trigger.trigger_mode = mode.id;
			if(mode.id != 'disabled'){
				$scope.setting.trigger.channel_name = channel.id;
				$scope.setting.trigger.threshold = threshold;
			}
			console.log($scope.setting);
			$http({method: 'POST', url: apiAddress + '/acquisition/settings/set', data: $scope.setting})
			.success(function(data, status, headers, config) {
				console.log('success:' + data);
				$scope.flgTrgCh = $scope.flgTrgCh * -1;	
        	})
        	.error(function(data, status, headers, config) {
				console.log('error:' + data);
			});
		};
		
		$scope.changeThreshold = function(mlSetting, recFlg){
			
			$http({method: 'POST', url: apiAddress + '/ml/settings/set', data: mlSetting})
			.success(function(data, status, headers, config) {
				console.log('/ml/settings/set - success:' + data);
        	})
        	.error(function(data, status, headers, config) {
				console.log('error:' + data);
			});
			
			$scope.recSetting.store_by_loss_value = 0;
			if(recFlg.id > 0){
				$scope.recSetting.store_by_loss_value = mlSetting.loss_threshold;
			}
			console.log('POST /recording/settings/set - data:');
			console.log($scope.recSetting);
			$http({method: 'POST', url: apiAddress + '/recording/settings/set', data: $scope.recSetting})
			.success(function(data, status, headers, config) {
				console.log('success:' + data);
				getRecordSetting();
        	})
        	.error(function(data, status, headers, config) {
				console.log('error:' + data);
			});
		};
		
		$scope.stopMl = function() {
			
			$http({method: 'GET', url: apiAddress + '/ml/stop'})
			.success(function(data, status, headers, config) {
				console.log('success:' + data);
				$window.location.href = '/#/startMl';
        	})
        	.error(function(data, status, headers, config) {
				console.log('error:' + data);
				$window.location.href = '/#/startMl';
			});
			
			
		};
	}
})();
