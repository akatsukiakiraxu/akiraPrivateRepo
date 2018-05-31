/**
 * @author v.lugovsky
 * created on 16.12.2015
 */
(function () {
  'use strict';

  angular.module('OliveAtFactory.pages.startMl', [])
      .config(routeConfig);

  /** @ngInject */
  function routeConfig($stateProvider) {
    $stateProvider
        .state('startMl', {
          url: '/startMl',
          templateUrl: 'app/pages/startMl/startMl.html',
        });
  }
})();
