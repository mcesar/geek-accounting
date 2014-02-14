angular.module('ga.service', ['ngRoute','ngResource']) 
.factory('GaServer', function ($resource, $cacheFactory){
  return $resource('/charts-of-accounts', {}, {
    chartsOfAccounts: {method:'GET', params:{}, isArray:true},
    accounts: {method:'GET', params:{}, isArray:true, cache: $cacheFactory('accounts'), url:'/charts-of-accounts/:coa/accounts'},
    balanceSheet: {method:'GET', params:{}, isArray:true, url:'/charts-of-accounts/:coa/balance-sheet?at=:at'},
    incomeStatement: {method:'GET', params:{}, url:'/charts-of-accounts/:coa/income-statement?from=:from&to=:to'},
    ledger: {method:'GET', params:{}, url:'/charts-of-accounts/:coa/accounts/:account/ledger?from=:from&to=:to'},
    addAccount: {method:'POST', params:{}, url:'/charts-of-accounts/:coa/accounts'},
    addTransaction: {method:'POST', params:{}, url:'/charts-of-accounts/:coa/transactions'}
  });
});

angular.module('ga', ['ngRoute','ngResource', 'ga.service'])
.config(function ($routeProvider, $locationProvider) {
  $routeProvider
    .when('/charts-of-accounts/:coa/accounts', {
      controller:'AccountsCtrl',
      templateUrl:'partials/accounts.html'
    })
    .when('/charts-of-accounts/:coa/edit-account', {
      controller:'AccountsCtrl',
      templateUrl:'partials/edit-account.html'
    })
    .when('/charts-of-accounts/:coa/balance-sheet', {
      controller:'BsCtrl',
      templateUrl:'partials/balance-sheet.html'
    })
    .when('/charts-of-accounts/:coa/income-statement', {
      controller:'IsCtrl',
      templateUrl:'partials/income-statement.html'
    })
    .when('/charts-of-accounts/:coa/accounts/:account/ledger', {
      controller:'LedgerCtrl',
      templateUrl:'partials/ledger.html'
    })
    .when('/charts-of-accounts/:coa/transaction', {
      controller:'TransactionCtrl',
      templateUrl:'partials/transaction.html'
    })
    .when('/login', {
      controller:'LoginCtrl',
      templateUrl:'partials/login.html'
    })
    .otherwise({
      redirectTo:'/'
    });
})
.directive('gaFocus', function () {
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
})
.directive("accountTypeahead", function ($routeParams, $rootScope, $compile, GaServer) {
  return {
    restrict: 'A',
    require: '?ngModel',
    link: function(scope, element, attrs, ngModel) {

      var source = function (query, cb) {
        var accounts = GaServer.accounts({coa: $routeParams.coa}, function () {
          var map = {}, result = [], i;
          var containsQuery = function (account, query) {
            if (!account) { return false; }
            if (account.number.indexOf(query) > -1 || account.name.toLowerCase().indexOf(query.toLowerCase()) > -1) {
              return true;
            }
            return containsQuery(map[account.parent], query);
          };
          for (i = 0; i < accounts.length; i += 1) {
            map[accounts[i]._id] = accounts[i];
          }
          for (i = 0; i < accounts.length; i += 1) {
            if (containsQuery(accounts[i], query)) {
              result.push(accounts[i]);
            }
          }
          cb(result);
        });
      }

      var templateFn = function (datum) {
        return '<p><strong>' + datum.number + '</strong></p><p>' + datum.name + '</p>';
      };
       
      element.typeahead(null, {
        displayKey: 'number',
        source: source,
        templates: {
          suggestion: templateFn
        }
      })
      .on('typeahead:selected', function($e, datum) { 
        ngModel.$setViewValue(datum.number);
      })
      .on('typeahead:onAutocompleted', function($e, datum) { 
        ngModel.$setViewValue(datum.number);
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

var NavigatorCtrl = function ($scope, $rootScope, $location, $http, $cacheFactory, GaServer) {
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
    case 'edit-account':
      return ': nova conta';
    default:
      return '';
    }
  };
  $scope.routeIs = function(routes) {
    var i = 0, segment = lastSegment();
    for (; i < routes.length; i += 1) {
      if (segment === routes[i]) {
        return true;
      }
    };
    return false;
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
    $cacheFactory.get('accounts').removeAll();
  });
};

var AccountsCtrl = function ($scope, $routeParams, $cacheFactory, GaServer) {
  $scope.account = {};
  $scope.accounts = GaServer.accounts({coa: $routeParams.coa});
  $scope.add = function () {
    var tags = ($scope.tags || '').split(','), i;
    for (i = 0; i < tags.length; i += 1) {
      $scope.account[tags[i]] = true;
    }
    GaServer.addAccount({coa: $routeParams.coa}, $scope.account, function () {
      $cacheFactory.get('accounts').removeAll();
      $scope.account = {};
      $scope.tags = undefined;
      $scope.errorMessage = undefined;
    }, function (response) {
      $scope.errorMessage = response.data.replace('Error: ', '');
    });
  };
};

var BsCtrl = function ($scope, $routeParams, GaServer) {
  var s = new Date().toJSON();
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
  $scope.currentChartOfAccounts = function () {
    return $routeParams.coa;
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
  var t = new Date().toJSON(), f;
  t = t.substring(0, t.indexOf('T'));
  f = t.substring(0, 8) + '01'
  $scope.properties = [];
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
  $scope.currentChartOfAccounts = function () {
    return $routeParams.coa;
  }
};

var LedgerCtrl = function ($scope, $routeParams, GaServer) {
  var t = new Date().toJSON(), f;
  t = t.substring(0, t.indexOf('T'));
  f = t.substring(0, 8) + '01'
  $scope.ledger = GaServer.ledger({coa: $routeParams.coa, account: $routeParams.account, from: f, to: t});
  $scope.convertToUTC = function(dt) {
    var localDate = new Date(dt);
    var localTime = localDate.getTime();
    var localOffset = localDate.getTimezoneOffset() * 60000;
    return new Date(localTime + localOffset);
  };
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
    $scope.account = null;
    $scope.transaction = {debits:[], credits:[]};
    $scope.entries = [];
    $scope.date = new Date().toISOString().substring(0, 10);    
  };
  clear();
  $scope.accountFocus = true;
  $scope.add = function () {
    add(function () { $window.history.back(); });
  };
  $scope.addAndContinue = function () {
    add(function () { 
      clear(); 
      $scope.accountFocus = true; 
    });
  };
  $scope.addEntry = function () {
    var entry;
    if (!$scope.account) {
      $scope.errorMessage = "A conta deve ser informada";
      return;
    }
    if (!$scope.debit && !$scope.credit) {
      if ($scope.debitsSum() === $scope.creditsSum()) {
        $scope.errorMessage = "Ou o débito ou o crédito deve ser informado";
        return;
      } else if ($scope.debitsSum() > $scope.creditsSum()) {
        $scope.credit = $scope.debitsSum() - $scope.creditsSum();
      } else {
        $scope.debit = $scope.creditsSum() - $scope.debitsSum();
      }
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
    var accounts = GaServer.accounts({coa: $routeParams.coa}, function () {
      for (var i = 0; i < accounts.length; i++) {
        if (accounts[i].number === entry.e.account) {
          entry.account = accounts[i];
        }
      }
    });
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