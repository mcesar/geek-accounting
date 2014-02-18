angular.module('ga.service', ['ngRoute','ngResource']) 
.factory('GaServer', function ($resource, $cacheFactory){
  return $resource('/charts-of-accounts', {}, {
    chartsOfAccounts: {method:'GET', params:{}, isArray:true},
    accounts: {method:'GET', params:{}, isArray:true, cache: $cacheFactory('accounts'), url:'/charts-of-accounts/:coa/accounts'},
    account: {method:'GET', params:{}, url:'/charts-of-accounts/:coa/accounts/:account'},
    transaction: {method:'GET', params:{}, url:'/charts-of-accounts/:coa/transactions/:transaction'},
    balanceSheet: {method:'GET', params:{}, isArray:true, url:'/charts-of-accounts/:coa/balance-sheet?at=:at'},
    incomeStatement: {method:'GET', params:{}, url:'/charts-of-accounts/:coa/income-statement?from=:from&to=:to'},
    ledger: {method:'GET', params:{}, url:'/charts-of-accounts/:coa/accounts/:account/ledger?from=:from&to=:to'},
    journal: {method:'GET', params:{}, isArray:true, url:'/charts-of-accounts/:coa/journal?from=:from&to=:to'},
    addAccount: {method:'POST', params:{}, url:'/charts-of-accounts/:coa/accounts'},
    addTransaction: {method:'POST', params:{}, url:'/charts-of-accounts/:coa/transactions'},
    updateAccount: {method:'PUT', params:{}, url:'/charts-of-accounts/:coa/accounts/:account'},
    updateTransaction: {method:'PUT', params:{}, url:'/charts-of-accounts/:coa/transactions/:transaction'},
    removeTransaction: {method:'DELETE', params:{}, url:'/charts-of-accounts/:coa/transactions/:transaction'},
    removeAccount: {method:'DELETE', params:{}, url:'/charts-of-accounts/:coa/accounts/:account'}
  });
});

angular.module('ga', ['ngRoute','ngResource', 'ga.service'])
.config(function ($routeProvider, $locationProvider) {
  $routeProvider
    .when('/charts-of-accounts/:coa/accounts', {
      controller:'AccountsCtrl',
      templateUrl:'partials/accounts.html'
    })
    .when('/charts-of-accounts/:coa/account', {
      controller:'AccountsCtrl',
      templateUrl:'partials/account.html'
    })
    .when('/charts-of-accounts/:coa/accounts/:account', {
      controller:'AccountsCtrl',
      templateUrl:'partials/account.html'
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
    .when('/charts-of-accounts/:coa/journal', {
      controller:'JournalCtrl',
      templateUrl:'partials/journal.html'
    })
    .when('/charts-of-accounts/:coa/transaction', {
      controller:'TransactionCtrl',
      templateUrl:'partials/transaction.html'
    })
    .when('/charts-of-accounts/:coa/transaction/:transaction', {
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

      scope.$watch(attrs.ngModel, function(newValue, oldValue) {
        if (newValue === null && oldValue !== null) {
          element.typeahead('val', null);
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

var NavigatorCtrl = function ($scope, $rootScope, $location, $http, $cacheFactory, GaServer) {
  var isLastSegment = function (segment) {
    var arr = $location.path().split('/');
    if (arr[arr.length - 1].split('?')[0] === segment) {
      return true;
    } else if (arr.length > 1 && arr[arr.length - 2] === segment) {
      return true;
    } else {
      return false;
    }
  }
  $scope.isLoggedIn = function() {
    return $rootScope.loggedIn === true;
  };
  $scope.actionLabel = function() {
    if ($scope.routeIs('transaction$')) {
      return ': lançamento';
    } else if ($scope.routeIs('account$')) {
      return ': nova conta';
    } else if ($scope.routeIs('accounts\/[^\/]+$')) {
      return ': edição de conta';
    } else {
      return '';
    }
  };
  $scope.routeIs = function(pattern) {
    return new RegExp(pattern).test($location.path());
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

var AccountsCtrl = function ($scope, $routeParams, $cacheFactory, $window, GaServer) {
  var account
  $scope.account = {};
  $scope.accounts = GaServer.accounts({coa: $routeParams.coa});
  $scope.save = function () {
    var tags = ($scope.tags || '').split(','), i;
    var showError = function (response) {
      $scope.errorMessage = response.data.replace('Error: ', '');
    };
    for (i = 0; i < tags.length; i += 1) {
      $scope.account[tags[i]] = true;
    }
    if ($routeParams.account) {
      GaServer.updateAccount({coa: $routeParams.coa, account: $routeParams.account}, $scope.account, function () {
        $cacheFactory.get('accounts').removeAll();
        $window.history.back();
      }, showError);
    } else {
      GaServer.addAccount({coa: $routeParams.coa}, $scope.account, function () {
        $cacheFactory.get('accounts').removeAll();
        $scope.account = {};
        $scope.tags = undefined;
        $scope.errorMessage = undefined;
      }, showError);
    }
  };
  $scope.remove = function () {
    GaServer.removeAccount({coa: $routeParams.coa, account: $routeParams.account}, function () {
      $cacheFactory.get('accounts').removeAll();
      $window.history.back();
    });
  }

  $scope.currentChartOfAccounts = function () {
    return $routeParams.coa;
  }
  $scope.accountId = function () {
    return $routeParams.account;
  };

  if ($routeParams.account) {
    account = GaServer.account({coa: $routeParams.coa, account: $routeParams.account}, function () {
      var accounts;
      $scope.account = { _id: account._id, name: account.name, number: account.number };
      $scope.tags = account.tags.join(',');
      accounts = GaServer.accounts({coa: $routeParams.coa}, function () {
        var i;
        for (i = 0; i < accounts.length; i += 1) {
          if (accounts[i]._id === account.parent) {
            $scope.account.parent = accounts[i].number;
          }
        }
      });
    });
  }
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
  $scope.convertToUTC = convertToUTC;
  $scope.currentChartOfAccounts = function () {
    return $routeParams.coa;
  }
};

var JournalCtrl = function ($scope, $routeParams, GaServer) {
  var t = new Date().toJSON();
  t = t.substring(0, t.indexOf('T'));
  $scope.journal = GaServer.journal({coa: $routeParams.coa, from: t, to: t});
  $scope.convertToUTC = convertToUTC;
  $scope.currentChartOfAccounts = function () {
    return $routeParams.coa;
  }
};

var TransactionCtrl = function ($scope, $routeParams, $window, GaServer) {
  var t
  var save = function (callback) {
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
    if ($routeParams.transaction) {
      GaServer.updateTransaction({coa: $routeParams.coa, transaction: $routeParams.transaction}, $scope.transaction, function () {
        if (callback) callback();
      }, function (response) {
        $scope.errorMessage = response.data.replace('Error: ', '');
      });
    } else {
      GaServer.addTransaction({coa: $routeParams.coa}, $scope.transaction, function () {
        if (callback) callback();
      }, function (response) {
        $scope.errorMessage = response.data.replace('Error: ', '');
      });
    }
  };
  var clear = function () {
    $scope.account = null;
    $scope.transaction = {debits:[], credits:[]};
    $scope.entries = [];
    $scope.date = new Date().toISOString().substring(0, 10);
  };
  clear();
  $scope.accountFocus = true;
  if ($routeParams.transaction) {
    t = GaServer.transaction({coa: $routeParams.coa, transaction: $routeParams.transaction}, function () {
      var accounts;
      $scope.transaction = t;
      $scope.date = t.date.substring(0, 10);
      accounts = GaServer.accounts({coa: $routeParams.coa}, function () {
        var map = {}, a, i;
        for (i = 0; i < accounts.length; i += 1) {
          map[accounts[i]._id] = accounts[i];
        }
        function add (account, value, balanceNature) {
          var entry = {e: {account: account.number, value: value}, account: account};
          entry[balanceNature] = value;
          $scope.entries.push(entry);

        }
        for (i = 0; i < t.debits.length; i += 1) {
          a = map[t.debits[i].account];
          add(a, t.debits[i].value, 'debit');
          t.debits[i].account = a.number;
        }
        for (i = 0; i < t.credits.length; i += 1) {
          a = map[t.credits[i].account];
          add(map[t.credits[i].account], t.credits[i].value, 'credit');
          t.credits[i].account = a.number;
        }
      });
    });
  }
  $scope.save = function () {
    save(function () { $window.history.back(); });
  };
  $scope.addAndContinue = function () {
    save(function () { 
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
  $scope.removeEntry = function (index) {
    function removeOnTransaction (col) {
      var i;
      for (i = 0; i < col.length; i += 1) {
        if (col[i].account === $scope.entries[index].e.account && col[i].value === $scope.entries[index].e.value) {
          col.splice(i, 1);
          return
        }
      }
    }
    removeOnTransaction($scope.transaction.debits);
    removeOnTransaction($scope.transaction.credits);
    $scope.entries.splice(index, 1);
  }
  $scope.remove = function () {
    GaServer.removeTransaction({coa: $routeParams.coa, transaction: $routeParams.transaction}, function () {
      $window.history.back();
    });
  }
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
  $scope.transactionId = function () {
    return $routeParams.transaction;
  };
}

function convertToUTC (dt) {
  var localDate = new Date(dt);
  var localTime = localDate.getTime();
  var localOffset = localDate.getTimezoneOffset() * 60000;
  return new Date(localTime + localOffset);
}