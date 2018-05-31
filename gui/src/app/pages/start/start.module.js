/**
 * @author v.lugovsky
 * created on 16.12.2015
 */
(function () {
	'use strict';

	angular.module('OliveAtFactory.pages.start', [])
		.config(routeConfig);
	
	/** @ngInject */
	function routeConfig($stateProvider) {
		$stateProvider
			.state('start', {
				url: '/start',
				templateUrl: 'app/pages/start/start.html',
				controller: 'StartPageCtrl',
				title: 'START',
				sidebarMeta: {
					icon: 'ion-android-home',
					order: 0,
				},
		});
	}
})();
