angular.module('ga.service', ['ngRoute','ngResource']) 
.factory('GaServer', ['$resource',
  function($resource){
    return $resource('/charts-of-accounts', {}, {
      chartsOfAccounts: {method:'GET', params:{}, isArray:true},
      balanceSheet: {method:'GET', params:{}, isArray:true, url: '/charts-of-accounts/:coa/balance-sheet?at=:at'},
      incomeStatement: {method:'GET', params:{}, url: '/charts-of-accounts/:coa/income-statement?from=:from&to=:to'}
    });
  }]);

angular.module('ga', ['ngRoute','ngResource', 'ga.service'])
.config(function ($routeProvider, $locationProvider) {
  $routeProvider
    .when('/charts-of-accounts/:coa/balance-sheet', {
      controller:'BsCtrl',
      templateUrl:'balance-sheet.html'
    })
    .when('/charts-of-accounts/:coa/income-statement', {
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

var NavigatorCtrl = function ($scope, $location, GaServer) {
  var lastSegment = function () {
    var arr = $location.path().split('/');
    return arr[arr.length - 1].split('?')[0];
  }
  $scope.chartsOfAccounts = GaServer.chartsOfAccounts({}, function () {
    $scope.currentChartOfAccounts = $scope.chartsOfAccounts[0];
  });
  $scope.$watch('currentChartOfAccounts', function (newValue) {
    if (typeof newValue === 'undefined') { 
      $location.path('/');
    } else {
      var segment = lastSegment();
      if (segment === '') {
        segment = 'balance-sheet';
      }
      $location.path('/charts-of-accounts/' + $scope.currentChartOfAccounts._id + '/' + segment);
    }
  });
  $scope.routeIs = function(routes) {
    var i = 0, segment = lastSegment();
    for (; i < routes.length; i += 1) {
      if (segment === routes[i]) {
        return true;
      }
    };
    return false;
  };
};

var BsCtrl = function ($scope, $routeParams, GaServer) {
  s = new Date().toJSON();
  s = s.substring(0, s.indexOf('T'));
  $scope.balanceSheet = GaServer.balanceSheet({coa: $routeParams.coa, at: s});
  $scope.isDebitBalance = function (e) {
    return e && e.account && 
      (e.account.debitBalance && e.account.name.indexOf('(-)') === -1) ||
      (e.account.creditBalance && e.account.name.indexOf('(-)') !== -1);
  }
  $scope.isCreditBalance = function (e) {
    return !$scope.isDebitBalance(e);
  }
};

var IsCtrl = function ($scope, $routeParams, GaServer) {
  var label = {
    'grossRevenue': 'Gross revenue',
    'deduction': 'Gross revenue',
    'salesTax': 'Sales tax',
    'netRevenue': 'Net revenue',
    'cost': 'Cost',
    'grossProfit': 'Gross profit',
    'operatingExpense': 'Operating expense',
    'netOperatingIncome': 'Net operating income',
    'nonOperatingRevenue': 'Non operating revenue',
    'nonOperatingExpense': 'Non operating expense',
    'nonOperatingTax': 'Non operating tax',
    'incomeBeforeIncomeTax': 'Income before income tax',
    'incomeTax': 'Income tax',
    'dividends': 'Dividends',
    'netIncome': 'Net income'
  };
  $scope.properties = [];
  t = new Date().toJSON();
  t = t.substring(0, t.indexOf('T'));
  f = t.substring(0, 8) + '01'
  $scope.incomeStatement = GaServer.incomeStatement(
    {coa: $routeParams.coa, from: f, to: t}, 
    function () {
      var result = [];
      for(var p in $scope.incomeStatement) { 
         if ($scope.incomeStatement.hasOwnProperty(p)) {
            if (label[p] && $scope.incomeStatement[p]) {
              result.push({label: label[p], prop: $scope.incomeStatement[p]});
            }
         }
      }
      $scope.properties = result;
    }
  );
};