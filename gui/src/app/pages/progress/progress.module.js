/**
 * @author a.demeshko
 * created on 12.21.2015
 */
(function () {
  'use strict';

  angular.module('OliveAtFactory.pages.progress', [])
    .config(routeConfig)
    .config(function(){
      $.jstree.defaults.core.themes.url = true;
      $.jstree.defaults.core.themes.dir = "assets/img/theme/vendor/jstree/dist/themes";
    });

  /** @ngInject */
  function routeConfig($stateProvider) {
    $stateProvider
        .state('progress', {
          url: '/progress',
          templateUrl: 'app/pages/progress/progress.html',
          title: 'Progress',
        });
  }

})();
