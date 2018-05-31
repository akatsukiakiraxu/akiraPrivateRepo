/**
 * @author v.lugovsky
 * created on 16.12.2015
 */
(function () {
	'use strict';

	angular.module('OliveAtFactory.pages.start')
	  .controller('StartPageCtrl', StartPageCtrl);

	/** @ngInject */
	function StartPageCtrl($scope, $timeout, $http, $window, $location) {

		//# 機械学習が動いていれば推論結果表示画面へ遷移させる
		$http({method: 'GET', url: 'http://' + $location.host() + ':2223/status'}).
			success(function(data, status, headers, config) {
				if(data.is_ml_running == true){
					$window.location.href = '/#/results';
				}
		}).
		error(function(data, status, headers, config) {
			console.log('error');
		});
	}
})();
