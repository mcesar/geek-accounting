angular.module('ga', ['ngRoute','ngResource']) 
.factory('GaServer', ['$resource',
  function($resource){
    return $resource('/charts-of-accounts', {}, {
      chartsOfAccounts: {method:'GET', params:{}, isArray:true},
      balanceSheet: {method:'GET', params:{}, isArray:true, url: '/charts-of-accounts/:coa/balance-sheet?at=:at'}
    });
  }])
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
})
.controller('GaCtrl', function ($scope, $http, $location, GaServer) {
  $http.defaults.headers.common['Authorization'] = 'Basic ' + btoa('admin:admin');
  $scope.chartsOfAccounts = GaServer.chartsOfAccounts();
  $scope.balanceSheet = [];
  $scope.$watch('chartsOfAccounts.length', function () {
    if (typeof selectedCoa === 'undefined') {
      $scope.selectedCoa = $scope.chartsOfAccounts[0];
    }
  });
  $scope.$watch('selectedCoa._id', function () {
    if (typeof $scope.selectedCoa === 'undefined') {
      $scope.balanceSheet = [];
    } else {
      s = new Date().toJSON();
      s = s.substring(0, s.indexOf('T'));
      $scope.balanceSheet = GaServer.balanceSheet({coa: $scope.selectedCoa._id, at: s});
    }
  });
  $scope.isDebitBalance = function (e) {
    return e && e.account && 
      (e.account.debitBalance && e.account.name.indexOf('(-)') === -1) ||
      (e.account.creditBalance && e.account.name.indexOf('(-)') !== -1);
  }
  $scope.isCreditBalance = function (e) {
    return !$scope.isDebitBalance(e);
  }
  $scope.routeIs = function(routes) {
    var i = 0;
    for (; i < routes.length; i += 1) {
      if ($location.path() === routes[i]) {
        return true;
      }
    };
    return false;
  };
});