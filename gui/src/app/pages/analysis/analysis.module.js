(function () {
  'use strict';

  angular.module('OliveAtFactory.pages.analysis', [])
      .config(routeConfig);

  /** @ngInject */
  function routeConfig($stateProvider) {
    $stateProvider
        .state('analysis', {
          url: '/analysis',
          templateUrl: 'app/pages/analysis/analysis.html',
          title: 'Analysis Viewer',
        });
  }

})();
