/**
 * @author v.lugovsky
 * created on 16.12.2015
 */
(function () {
	'use strict';

	angular.module('OliveAtFactory.pages', [
		'ui.router',
		'OliveAtFactory.pages.start',
		'OliveAtFactory.pages.startMl',
		'OliveAtFactory.pages.monitor',
		'OliveAtFactory.pages.progress',
		'OliveAtFactory.pages.results',
		'OliveAtFactory.pages.analysis',
	])
	.config(routeConfig);

	/** @ngInject */
	function routeConfig($urlRouterProvider, baSidebarServiceProvider) {
		$urlRouterProvider.otherwise('/start');

//	baSidebarServiceProvider.addStaticItem({
//	  title: 'Pages',
//	  icon: 'ion-document',
//	  subMenu: [{
//		title: '404 Page',
//		fixedHref: '404.html',
//		blank: true
//	  }]
//	});
	//    baSidebarServiceProvider.addStaticItem({
	//      title: 'Menu Level 1',
	//      icon: 'ion-ios-more',
	//      subMenu: [{
	//        title: 'Menu Level 1.1',
	//        disabled: true
	//      }, {
	//        title: 'Menu Level 1.2',
	//        subMenu: [{
	//          title: 'Menu Level 1.2.1',
	//          disabled: true
	//        }]
	//      }]
	//    });
	}

})();
