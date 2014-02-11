angular.module('ga.service', ['ngRoute','ngResource']) 
.factory('GaServer', ['$resource', function($resource){
  return $resource('/charts-of-accounts', {}, {
    chartsOfAccounts: {method:'GET', params:{}, isArray:true},
    balanceSheet: {method:'GET', params:{}, isArray:true, url: '/charts-of-accounts/:coa/balance-sheet?at=:at'},
    incomeStatement: {method:'GET', params:{}, url: '/charts-of-accounts/:coa/income-statement?from=:from&to=:to'},
    addTransaction: {method:'POST', params:{}, url: '/charts-of-accounts/:coa/transactions'}
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
    .when('/charts-of-accounts/:coa/transaction', {
      controller:'TransactionCtrl',
      templateUrl:'transaction.html'
    })
    .when('/login', {
      controller:'LoginCtrl',
      templateUrl:'login.html'
    })
    .otherwise({
      redirectTo:'/'
    });
})
.directive('gaFocus', function() {
  return {
    scope: { trigger: '=gaFocus' },
    link: function(scope, element) {
      scope.$watch('trigger', function(value) {
        if(value === true) { 
          element[0].focus();
          scope.trigger = false;
        }
      });
    }
  };
});

var LoginCtrl = function ($scope, $rootScope, $http, $location, GaServer) {
  $scope.login = function () {
    $http.defaults.headers.common['Authorization'] = 'Basic ' + btoa($scope.user + ':' + $scope.password);
    $http.get('/ping').success(function(data, status, headers, config) {
      $rootScope.loggedIn = true;
      $location.path($rootScope.previousPath).replace();
    }).error(function(data, status, headers, config) {
      $http.defaults.headers.common['Authorization'] = undefined;
      $scope.errorMessage = "Login ou senha incorretos";
    });
  }
};

var NavigatorCtrl = function ($scope, $rootScope, $location, $http, GaServer) {
  var lastSegment = function () {
    var arr = $location.path().split('/');
    return arr[arr.length - 1].split('?')[0];
  }
  $scope.isLoggedIn = function() {
    return $rootScope.loggedIn === true;
  };
  $scope.actionLabel = function() {
    switch (lastSegment()) {
    case 'transaction':
      return ': lançamento';
    default:
      return '';
    }
  };
  $scope.$watch(function() { return $location.path(); }, function(newValue, oldValue) {
    if (!$scope.isLoggedIn()) {
      $rootScope.previousPath = $rootScope.previousPath || $location.path();
      if ($rootScope.previousPath === '/login') {
        $rootScope.previousPath = '/';
      }
      $location.path('/login');
    } else if (oldValue === '/login') {
      $scope.chartsOfAccounts = GaServer.chartsOfAccounts({}, function () {
        if ($location.path() === '/') {
          $scope.currentChartOfAccounts = $scope.chartsOfAccounts[0];
        } else {
          var i = 0
            , arr = $location.path().split('/');
          for (; i < $scope.chartsOfAccounts.length; i += 1) {
            if ($scope.chartsOfAccounts[i]._id === arr[2]) {
              $scope.currentChartOfAccounts = $scope.chartsOfAccounts[i];
            }
          };
        }
      });      
    }
  });
  $scope.$watch('currentChartOfAccounts', function (newValue) {
    if (typeof newValue === 'undefined') { return; }
    var arr = $location.path().split('/');
    if (arr.length > 2 && arr[2] === newValue._id) { return; }
    if (arr.length === 2) {
      arr = [ '', 'charts-of-accounts', newValue._id, 'balance-sheet' ];
    }
    arr[2] = newValue._id;
    $location.path(arr.join('/'));
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

var TransactionCtrl = function ($scope, $routeParams, $window, GaServer) {
  var add = function (callback) {
    if ($scope.transaction.debits.length < 1 || $scope.transaction.credits.length < 1) {
      $scope.errorMessage = "Devem ser informados pelo menos um débito e pelo menos um crédito";
      return;
    }
    if (!$scope.date) {
      $scope.errorMessage = "A data deve ser informada";
      return;
    }
    if (!$scope.transaction.memo) {
      $scope.errorMessage = "O memorando deve ser informado";
      return;
    }
    $scope.transaction.date = $scope.date + 'T00:00:00Z';
    GaServer.addTransaction({coa: $routeParams.coa}, $scope.transaction, function () {
      if (callback) callback();
    }, function (response) {
      $scope.errorMessage = response.data.replace('Error: ', '');
    });
  };
  var clear = function () {
    $scope.transaction = {debits:[], credits:[]};
    $scope.entries = [];
    $scope.date = new Date().toISOString().substring(0, 10);    
  };
  clear();
  $scope.add = function () {
    add(function () { $window.history.back(); });
  };
  $scope.addAndContinue = function () {
    add(function () { clear(); });
  };
  $scope.addEntry = function () {
    var entry;
    if (!$scope.account) {
      $scope.errorMessage = "A conta deve ser informada";
      return;
    }
    if (!$scope.debit && !$scope.credit) {
      $scope.errorMessage = "Ou o débito ou o crédito deve ser informado";
      return;
    }
    if ($scope.debit) {
      entry = {e: {account: $scope.account, value: $scope.debit}, debit: $scope.debit};
      $scope.transaction.debits.push(entry.e)
    } else {
      entry = {e: {account: $scope.account, value: $scope.credit}, credit: $scope.credit};
      if ($scope.credit) {
        $scope.transaction.credits.push(entry.e)
      }
    }
    $scope.entries.push(entry);
    $scope.account = $scope.debit = $scope.credit = undefined;
    $scope.errorMessage = undefined;
  };
  $scope.debitsSum = function () {
    var result = 0, i = 0;
    for (; i < $scope.transaction.debits.length; i += 1) {
      result += $scope.transaction.debits[i].value;
    }
    return result;
  };
  $scope.creditsSum = function () {
    var result = 0, i = 0;
    for (; i < $scope.transaction.credits.length; i += 1) {
      result += $scope.transaction.credits[i].value;
    }
    return result;
  };
}