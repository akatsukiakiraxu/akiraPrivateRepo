(function () {
  'use strict';

	angular.module('OliveAtFactory.pages.monitor')
		.controller('monitorCtrl', monitorCtrl);

	/** @ngInject */
	function monitorCtrl($scope, $location, $http, $window, $sce) {
		
		var apiAddress = 'http://' + $location.host() + ':2223';
		$scope.qtzChoices = [
			{id: 0, label: 'DISABLED'},
			{id: 1, label: 'ENABLED'},
		];
		$scope.qtzChoice = $scope.qtzChoices[0];
		$scope.itvChoices = [
//			{id: 1, label: '1 ms'},
			{id: 10, label: '10 ms'},
			{id: 100, label: '100 ms'},
			{id: 1000, label: '1000 ms'},
		];
		$scope.itvChoice = $scope.itvChoices[0];
		$scope.recChoices = [
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
		$scope.recChoice = $scope.recChoices[0];
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
		
		$scope.signalInputTypeChoices = [
			{id: "single_ended", label: "Single Ended"},
			{id: "differential"  , label: "Differential"}
		];

		//# queryStringからinputのfft or rawを決定する
		var inputData = $location.search()['input'];
		$scope.inputName = inputData;
		$scope.inputIcon = 'fa fa-area-chart';
		if(inputData == 'raw'){
			$scope.inputIcon = 'ion-stats-bars';
		};
		
		//# Olive側の機械学習設定を取得する
		var getMlSetting = function(){
			
			$http({method: 'GET', url: apiAddress + '/ml/settings/get'})
				.success(function(data, status, headers, config) {

				$scope.mlSetting = data;
				$scope.currentRange = 'Current: ' + $scope.mlSetting.input_data_offset + ' to ' + ($scope.mlSetting.input_data_offset + $scope.mlSetting.input_data_size);
				if($scope.mlSetting.quantization.enabled){
					$scope.qtzChoice = $scope.qtzChoices[1];
				}
			})
				.error(function(data, status, headers, config) {
					console.log('error');
			});
		}
		getMlSetting();
		
		//# Olive側のモニタリング設定を取得する
		var getMonitorSetting = function(){
			
			$http({method: 'GET', url: apiAddress + '/monitoring/settings/get'})
				.success(function(data, status, headers, config) {
				$scope.mntSetting = data;
				$scope.mntDuration = 'Interval: ' + $scope.mntSetting.duration + ' sec';
				
				for(var i = 0; i < $scope.itvChoices.length; i++) {
					if($scope.itvChoices[i]['id'] == $scope.mntSetting.duration) {
						$scope.itvChoice = $scope.itvChoices[i];
						break;
					}
				}
			})
				.error(function(data, status, headers, config) {
					console.log('error');
			});
		};
		getMonitorSetting();
		
		$scope.changeQuantization = function(mlSetting, qtzFlg){
			
			var MIN_VALUE_NUM_OF_SECTIONS = 1;
			
			if(qtzFlg.id > 0){
				$scope.mlSetting.quantization.enabled = true;
				$scope.mlSetting.quantization.number_of_sections_x = mlSetting.quantization.number_of_sections_x;
				$scope.mlSetting.quantization.number_of_sections_y = mlSetting.quantization.number_of_sections_y;
			}
			else{
				$scope.mlSetting.quantization.enabled = false;
				$scope.mlSetting.quantization.number_of_sections_x = MIN_VALUE_NUM_OF_SECTIONS;
				$scope.mlSetting.quantization.number_of_sections_y = MIN_VALUE_NUM_OF_SECTIONS;
			}
				
			$http({method: 'POST', url: apiAddress + '/ml/settings/set', data: $scope.mlSetting})
			.success(function(data, status, headers, config) {
				console.log('/ml/settings/set - success:' + data);
        	})
        	.error(function(data, status, headers, config) {
				console.log('error:' + data);
			});
		};
		
		
		//# Olive側のモニタリング時間間隔を変更する
		$scope.changeMonitorInterval = function(interval) {
			
			$scope.mntSetting.duration = interval.id;
			console.log('POST /monitoring/settings/set - data:');
			console.log($scope.mntSetting);
			$http({method: 'POST', url: apiAddress + '/monitoring/settings/set', data: $scope.mntSetting})
			.success(function(data, status, headers, config) {
				console.log('success:' + data);
				getMonitorSetting();
        	})
        	.error(function(data, status, headers, config) {
				console.log('error:' + data);
			});
		};
		
		//# Olive側のログタイミングを取得する
		var getRecordSetting = function(){
			
			$http({method: 'GET', url: apiAddress + '/recording/settings/get'})
				.success(function(data, status, headers, config) {
				 $scope.recSetting = data;
				
				for(var i = 0; i < $scope.recChoices.length; i++) {
					if($scope.recChoices[i]['id'] == $scope.recSetting.store_every_seconds) {
						$scope.recChoice = $scope.recChoices[i];
						break;
					}
				}
			})
				.error(function(data, status, headers, config) {
					console.log('error');
			});
		};
		getRecordSetting();
		
		//# Olive側の現在の設定を取得する
		$http({method: 'GET', url: apiAddress + '/acquisition/settings/get'})
			.success(function(data, status, headers, config) {
			
			$scope.setting = data;
			
			//## Channel Active
			angular.forEach($scope.setting.channels, function(value, key){
				value.chName = key;
			});
			
			//### 閾値トリガーのチャンネル名	
			var idx = 0;
			var arySize = $scope.setting.channels.length;
			$scope.trgChChoices = {};
			angular.forEach($scope.setting.channels, function(value, key){
				$scope.trgChChoices[idx] = {};
				$scope.trgChChoices[idx]['id'] = key;
				$scope.trgChChoices[idx]['label'] = value.chName;
				//$scope.trgChChoice = $scope.trgChChoices[0];	//# for debug
				if(value.chName == $scope.setting.trigger.channel_name){
				   $scope.trgChChoice = $scope.trgChChoices[idx];
				}
				idx++;
			});
			$scope.trgChChoices[idx] = {};
			$scope.trgChChoices[idx]['id'] = '';
			$scope.trgChChoices[idx]['label'] = '(No value)';
			//console.log($scope.trgChChoices);
			
			//### settingの先頭をChannel Activeの初期値とする
			var keys = [];
			for(var key in data.channels){
				keys.push(key);
			}
			$scope.curCh = keys[0];			
			
			//# Olive側のハードウェアconfigを取得する
			$http({method: 'GET', url: apiAddress + '/acquisition/config/get'})
				.success(function(data, status, headers, config) {

				$scope.hwConfig = data.channels;
				$scope.config = data;
				
				//### 閾値トリガーの電圧レンジの範囲
				$scope.trgRngChoices = new Array($scope.setting.channels.length);
				angular.forEach($scope.setting.channels, function(value, key){
					$scope.trgRngChoices[value.chName] = {};
					$scope.trgRngChoices[value.chName]['min'] = $scope.config.ranges[value.range].minimum_voltage;
					$scope.trgRngChoices[value.chName]['max'] = $scope.config.ranges[value.range].maximum_voltage;
					$scope.trgRngChoices[value.chName]['val'] = 0;
					if(value.chName == $scope.setting.trigger.channel_name){
						$scope.trgRngChoices[value.chName]['val'] = $scope.setting.trigger.threshold;
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
		
		//# Olive側のchannel設定を変更する
		$scope.changeSetting = function(setting, recTiming) {
			
			$scope.curCh = setting.channels[$scope.curCh].chName;
			
			setting.trigger.channel_name = $scope.curCh;
			
			$http({method: 'POST', url: apiAddress + '/acquisition/settings/set', data: setting})
			.success(function(data, status, headers, config) {
				console.log('success:' + data);
        	})
        	.error(function(data, status, headers, config) {
				console.log('error:' + data);
			});
			
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
		
		$scope.learnSetting = function(mlSetting){
			
			mlSetting.target_channels = [$scope.curCh];
			mlSetting.input_type = $scope.inputName;

			$http({method: 'POST', url: apiAddress + '/ml/settings/set', data: mlSetting})
			.success(function(data, status, headers, config) {
				console.log('/ml/settings/set - success:' + data);
				
				$http({method: 'GET', url: apiAddress + '/ml/start'})
				.success(function(data, status, headers, config) {
					console.log('/ml/start - success:' + data);
					$window.location.href = '/#/progress';
				})
				.error(function(data, status, headers, config) {
					console.log('error:' + data);
				});
        	})
        	.error(function(data, status, headers, config) {
				console.log('error:' + data);
			});			
		};
		
		$scope.getMlSetting = function(){
			getMlSetting();
		}
	}
})();
