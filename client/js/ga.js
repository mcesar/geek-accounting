angular.module('ga.service', ['ngRoute','ngResource']) 
.factory('GaServer', ['$resource',
  function($resource){
    return $resource('/charts-of-accounts', {}, {
      chartsOfAccounts: {method:'GET', params:{}, isArray:true},
      balanceSheet: {method:'GET', params:{}, isArray:true, url: '/charts-of-accounts/:coa/balance-sheet?at=:at'},
      incomeStatement: {method:'GET', params:{}, isArray:true, url: '/charts-of-accounts/:coa/income-statement?from=:from&to=:to'}
    });
  }])
.value('currentChartOfAccounts', {
  get: function () {
    return this.id;
  },
  set: function (id) {
    this.id = id;
  }
});

angular.module('ga', ['ngRoute','ngResource', 'ga.service'])
.config(function ($routeProvider, $locationProvider) {
  $routeProvider
    .when('/', {
      controller:'BsCtrl',
      templateUrl:'balance-sheet.html'
    })
    .when('/balance-sheet', {
      controller:'BsCtrl',
      templateUrl:'balance-sheet.html'
    })
    .when('/income-statement', {
      controller:'IsCtrl',
      templateUrl:'income-statement.html'
    })
    .otherwise({
      redirectTo:'/'
    });
})
.run(function ($http) {
  $http.defaults.headers.common['Authorization'] = 'Basic ' + btoa('admin:admin');
});

var NavigatorCtrl = function ($scope, $location, GaServer, currentChartOfAccounts) {
  $scope.chartsOfAccounts = GaServer.chartsOfAccounts();
  $scope.currentChartOfAccounts = currentChartOfAccounts.get();
  $scope.$watch('currentChartOfAccounts', function (newValue) {
    if (typeof newValue === 'undefined') {
      currentChartOfAccounts.set(undefined);
    } else {
      currentChartOfAccounts.set(newValue._id);
    }
  });
  $scope.$watch('chartsOfAccounts.length', function () {
    if (typeof $scope.currentChartOfAccounts === 'undefined') {
      $scope.currentChartOfAccounts = $scope.chartsOfAccounts[0];
    }
  });
  $scope.routeIs = function(routes) {
    var i = 0;
    for (; i < routes.length; i += 1) {
      if ($location.path() === routes[i]) {
        return true;
      }
    };
    return false;
  };
};

var BsCtrl = function ($scope, GaServer, currentChartOfAccounts) {
  $scope.balanceSheet = [];
  $scope.currentChartOfAccounts = function () {
    return currentChartOfAccounts.get();
  };
  $scope.$watch('currentChartOfAccounts()', function () {
    if (typeof currentChartOfAccounts.get() === 'undefined') {
      $scope.balanceSheet = [];
    } else {
      s = new Date().toJSON();
      s = s.substring(0, s.indexOf('T'));
      $scope.balanceSheet = GaServer.balanceSheet({coa: currentChartOfAccounts.get(), at: s});
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
};

var IsCtrl = function ($scope, GaServer, currentChartOfAccounts) {
  $scope.incomeStatement = [];
  $scope.currentChartOfAccounts = function () {
    return currentChartOfAccounts.get();
  };
  $scope.$watch('currentChartOfAccounts()', function () {
    if (typeof currentChartOfAccounts.get() === 'undefined') {
      $scope.incomeStatement = [];
    } else {
      t = new Date().toJSON();
      t = t.substring(0, t.indexOf('T'));
      f = t.substring(0, 8) + '01'
      $scope.incomeStatement = GaServer.incomeStatement({coa: currentChartOfAccounts.get(), from: f, to: t});
    }
  });
};