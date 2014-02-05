angular.module('ga', ['ngRoute']) 
.config(function ($routeProvider, $locationProvider) {
  $routeProvider
    .when('/', {
      controller:'GaCtrl',
      templateUrl:'balance-sheet.html'
    })
    .when('/balance-sheet', {
      controller:'GaCtrl',
      templateUrl:'balance-sheet.html'
    })
    .when('/income-statement', {
      controller:'GaCtrl',
      templateUrl:'income-statement.html'
    })
    .otherwise({
      redirectTo:'/'
    });
  $locationProvider.html5Mode(true)
})