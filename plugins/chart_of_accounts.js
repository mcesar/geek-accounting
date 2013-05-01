/*jslint node: true */
'use strict';

var express = require('express');

var utils = require('../lib/utils').utils;
var db = require('../lib/database').db;
var BSON = require('mongodb').BSONPure;

function Service () {
	this.inheritedProperties = [ 
		{ name: 'balanceSheet', classification: 'financial statement' }, 
		{ name: 'incomeStatement', classification: 'financial statement' },
		{ name: 'operating', classification: 'income statement attribute' },
		{ name: 'deduction', classification: 'income statement attribute' },
		{ name: 'salesTax', classification: 'income statement attribute' },
		{ name: 'cost', classification: 'income statement attribute' },
		{ name: 'nonOperatingTax', classification: 'income statement attribute' },
		{ name: 'incomeTax', classification: 'income statement attribute' },
		{ name: 'dividends', classification: 'income statement attribute' }
	];
}

Service.prototype.chartOfAccountsValidation = function (coa) {
	if (utils.isEmpty(coa.name)) { return "The name must be informed"; }
};

Service.prototype.accountValidation = function (coa, account, callback) {
	var that = this, incomeStatementFlagsCount = 0;
	if (utils.isEmpty(account.number)) {
		return callback(null, "The number must be informed"); 
	}
	if (utils.isEmpty(account.name)) {
		return callback(null, "The name must be informed");
	}
	if (typeof account.balanceSheet !== 'boolean' &&
				typeof account.incomeStatement !== 'boolean') {
		return callback(null, "The financial statement must be informed");
	}
	if (account.balanceSheet === account.incomeStatement) {
		return callback(null, 
			"The statement must be either balance sheet or income statement");
	}
	if (typeof account.debitBalance === 'boolean' &&
				typeof account.creditBalance === 'boolean') {
		return callback(null, "The normal balance must be informed");
	}
	if (account.debitBalance === account.creditBalance) {
		return callback(null, "The normal balance must be either debit or credit");
	}
	this.inheritedProperties.forEach(function (prop) {
		if (prop.classification === 'income statement attribute' && 
					account[prop.name]) {
			incomeStatementFlagsCount++;
		}
	});
	if (incomeStatementFlagsCount > 1) {
		return callback(null, "Only one income statement attribute is allowed");
	}
	if (typeof account.parent !== 'undefined') {
		that.account(account.parent.toString(), function (err, parentAccount) {
			var i;
			if (err) { return callback(err); }
			if (!parentAccount) { 
				return callback(null, 'Parent not found'); 
			}
			if (account.number.indexOf(parentAccount.number) !== 0) {
				return callback(null, 
					"The number must start with parent's number");
			}
			for (i = 0; i < that.inheritedProperties.length; i += 1) {
				if (parentAccount[that.inheritedProperties[i].name] && 
						!account[that.inheritedProperties[i].name]) {
					return callback(null, 
						"The " + that.inheritedProperties[i].classification + 
						" must be same as the parent");
				}
			}
			callback();
		});
	} else {
		callback();
	}
};

Service.prototype.transactionValidation = function (transaction, callback) {
	var that = this;
	if (!Array.isArray(transaction.debits) || transaction.debits.length === 0) { 
		return callback(null, "At least one debit must be informed");
	}
	if (!Array.isArray(transaction.credits) || 
			transaction.credits.length === 0) { 
		return callback(null, "At least one credit must be informed");
	}
	if (typeof transaction.date === 'undefined' || 
			isNaN(transaction.date.getTime())) { 
		return callback(null, "The date must be informed");
	}
	if (utils.isEmpty(transaction.memo)) { 
		return callback(null, "The memo must be informed");
	}
	(function () {
		var props = ['debits', 'credits'], sum = { debits: 0, credits: 0 };
		function validateEntries (i, j, callback) {
			if (j >= transaction[props[i]].length) { return callback(); }
			if (typeof transaction[props[i]][j].value !== 'number') {
				return callback(null, "The value must be informed for each entry");
			}
			if (typeof transaction[props[i]][j].account === 'undefined') {
				return callback(null, "The account must be informed for each entry");
			}
			that.account(transaction[props[i]][j].account, 
				function (err, account) {
					if (account === null) {
						return callback(null, 'Account not found');
					}
					if (!account.analytic) {
						return callback(null, 'The account must be analytic');
					}
					sum[props[i]] += transaction[props[i]][j].value;
					validateEntries(i, j + 1, callback);
				}
			);
		}
		(function eachProp (i) {
			if (i >= props.length) {
				if (sum.debits === sum.credits) {
					return callback();
				} else {
					return callback(null, 
						"The sum of debit values must be equals to " + 
						"the sum of credit values");
				}
			}
			validateEntries(i, 0, function (err, validation) {
				if (err || validation) { return callback(err, validation); }
				eachProp(i + 1);
			});
		}(0));
	}());
};

Service.prototype.chartsOfAccounts = function (callback) {
	db.find('charts_of_accounts', function (err, items) {
		callback(err, items);
	});
};

Service.prototype.chartOfAccounts = function (id, callback) {
	db.findOne('charts_of_accounts', 
		{ _id: db.bsonId(id) }, 
		function (err, item) {
			if (err) { return callback(err); }
			callback(null, item);
		}
	);	
};

Service.prototype.addChartOfAccounts = function (coa, callback) {
	var validation = this.chartOfAccountsValidation(coa);
	if (validation) { return callback(new Error(validation)); }
	db.insert('charts_of_accounts', coa, function (err, item) {
		if (err) { return callback(err); }
		callback(null, item);
	});
};

Service.prototype.account = function (accountId, callback) {
	db.conn().collection('charts_of_accounts', function (err, coa) {
		if (err) { return callback(err); }
		coa.find({}, 
			{ accounts: { $elemMatch: { _id: db.bsonId(accountId) } } }).toArray(
			function (err, items) {
				var i;
				if (err) { return callback(err); }
				if (items === null) { return callback(null, null); }
				for (i = 0; i < items.length; i += 1) {
					if (items[i].accounts) {
						return callback(null, items[i].accounts[0]);
					}
				}
				callback(null, null);
			}
		);
	});
};

Service.prototype.addAccount = function (coa, account, callback) {
	var that = this;
	if (typeof account.number !== 'undefined') {
		account.number = account.number.toString();
	}
	this.accountValidation(coa, account, function (err, validation) {
		var update;
		if (err) { return callback(err); }
		if (validation) { return callback(new Error(validation)); }
		account._id = new BSON.ObjectID();
		if (account.parent) { account.parent = db.bsonId(account.parent); }
		account.analytic = true;
		update = { $push: { accounts: account } };
		if (account.retainedEarnings) {
			delete account.retainedEarnings;
			update.$set = { retainedEarningsAccount:  account._id };
		}
		db.update('charts_of_accounts', { _id: db.bsonId(coa._id) }, update,
			function (err, item) {
				if (err) { return callback(err); }
				if (account.parent) {
					that.account(account.parent.toString(), 
						function (err, parentAccount) {
							delete parentAccount.analytic;
							parentAccount.synthetic = true;
							db.update('charts_of_accounts', 
								{ _id: db.bsonId(coa._id), 
									'accounts._id': parentAccount._id }, 
								{ $set: { 'accounts.$': parentAccount } },
								function (err, item) {
									if (err) { return callback(err); }
									callback(null, account);
								}
							);
						}
					);
				} else {
					callback(null, account);
				}
			}
		);
	});
};

Service.prototype.updateAccount = function (coa, accountId, account, 
		callback) {
	var that = this;
	this.accountValidation(coa, account, function (err, validation) {
		var updatingAccount = {};
		if (err) { return callback(err); }
		if (validation) { return callback(new Error(validation)); }
		that.account(accountId, function (err, oldAccount) {
			var prop, update = { $set: { 'accounts.$': account } };
			function doUpdate () {
				for (prop in oldAccount) {
					if (oldAccount.hasOwnProperty(prop) && prop !== 'name') { 
						account[prop] = oldAccount[prop]; 
					}
				}
				db.update('charts_of_accounts', 
					{ _id: db.bsonId(coa._id), 'accounts._id': db.bsonId(accountId) }, 
					update,
					function (err, item) {
						if (err) { return callback(err); }
						callback(null, account);
					}
				);
			}
			if (account.retainedEarnings) {
				delete account.retainedEarnings;
				update.$set.retainedEarningsAccount = db.bsonId(accountId);
				doUpdate();
			} else if (account.retainedEarnings === false) {
				db.update('charts_of_accounts', 
					{ _id: db.bsonId(coa._id), 
						retainedEarningsAccount: db.bsonId(accountId) },
					{ $unset: { retainedEarningsAccount: 1 } },
					function (err, item) {
						doUpdate();
					}
				);
			} else {
				doUpdate();
			}
		});
	});
};

Service.prototype.transactions = function (coaId, callback) {
	db.find('transactions_' + coaId, function (err, items) {
		callback(err, items);
	});
};

Service.prototype.addTransaction = function (coaId, transaction, callback) {
	transaction.date = new Date(transaction.date);
	transaction.timestamp = new Date();
	this.transactionValidation(transaction, function (err, validation) {
		var props = [ 'debits', 'credits' ], i;
		if (err) { return callback(err); }
		if (validation) { return callback(new Error(validation)); }
		props.forEach(function (prop) {
			transaction[prop].forEach(function (entry) {
				entry.account = db.bsonId(entry.account);
			});
		});
		db.insert('transactions_' + coaId, transaction, function (err, item) {
			if (err) { return callback(err); }
			callback(null, item);
		});
	});
};

Service.prototype.journal = function (coaId, from, to, callback) {
	var that = this;
	db.conn().collection('transactions_' + coaId, function (err, collection) {
		if (err) { return callback(err); }
		collection.find({ date: { $gte: from , $lte: to } }).
			sort({ date: 1, timestamp: 1 }).
			toArray(function (err, items) {
				var props = [ 'debits', 'credits' ], entries = [], i, j;
				if (err) { return callback(err); }
				for (i = 0; i < props.length; i += 1) {
					for (j = 0; j < items.length; j += 1) {
						entries.push.apply(entries, items[j][props[i]]);
					}
				}
				(function eachEntry (i) {
					if (i >= entries.length) { return callback(null, items); }
					that.account(entries[i].account.toString(), function (err, account) {
						entries[i].accountName = account.name;
						entries[i].accountNumber = account.number;
						eachEntry(i + 1);
					});
				}(0));
			});
	});
};

Service.prototype.balanceIncrement = function (account, entry, prop) {
	if (account[prop.slice(0, -1) + 'Balance']) {
		return entry.value;
	} else {
		return -entry.value;
	}
}

Service.prototype.ledger = function (coaId, accountId, from, to, callback) {
	var that = this;
	db.conn().collection('transactions_' + coaId, function (err, collection) {
		if (err) { return callback(err); }
		that.account(accountId, function (err, account) {
			var result = { 
					account: 
						{ _id: accountId, number: account.number, name: account.name, 
							debitBalance: account.debitBalance, 
							creditBalance: account.creditBalance
						}, 
					entries: [] 
				};
			collection.
				find({ 
					date: { $lte: to },
					$or: [ 
						{ debits: { $elemMatch: { account: db.bsonId(accountId) } } },
						{ credits: { $elemMatch: { account: db.bsonId(accountId) } } } 
					] }).
				sort({ date: 1, timestamp: 1 }).
				toArray(function (err, items) {
					var props = [ 'debits', 'credits' ], prop, balance = 0, 
						runningBalance = 0, entry, i, j, k;
					if (err) { return callback(err); }
					for (i = 0; i < items.length; i += 1) {
						for (j = 0; j < props.length; j += 1) {
							for (k = 0; k < items[i][props[j]].length; k += 1) {
								if (items[i][props[j]][k].account.toString() === accountId) {
									runningBalance += that.balanceIncrement(account, 
											items[i][props[j]][k], props[j]);
									if (items[i].date < from) {
										balance = runningBalance;
									} else {
										entry = { 
											date: items[i].date, 
											memo: items[i].memo,
											counterpart: {} 
										};
										entry[props[j].slice(0, -1)] = items[i][props[j]][k].value;
										entry.balance = runningBalance;
										if (items[i][props[1 - j]].length === 1) {
											entry.counterpart._id = 
												items[i][props[1 - j]][0].account;
										} else {
											entry.counterpart.name = 'many';
										}
										result.entries.push(entry);
									}
								}
							}
						}
					}
					result.balance = balance;
					(function eachEntry (i) {
						if (i >= result.entries.length) { return callback(null, result); }
						if (result.entries[i].counterpart._id) {
							that.account(result.entries[i].counterpart._id.toString(), 
								function (err, account) {
									result.entries[i].counterpart.name = account.name;
									result.entries[i].counterpart.number = account.number;
									eachEntry(i + 1);
								}
							);							
						} else {
							eachEntry(i + 1);
						}
					}(0));
				});
		});
	});
};

Service.prototype.balances = function (coaId, from, to, accountFilter, 
		callback) {
	var that = this, accountsMap = {}, result = [];
	function incrementParentBalance (parent, entry, prop) {
		if (!parent) { return; }
		var parentItem = accountsMap[parent.toString()];
		parentItem.value += that.balanceIncrement(parentItem.account, entry, prop);
		incrementParentBalance(accountsMap[parent.toString()].account.parent, 
			entry, prop)
	}
	function afterFindAccounts (err, coa) {
		if (!coa) { return callback(null, null); }
		var accounts = coa.accounts, balanceItem, i;
		for (i = 0; i < accounts.length; i += 1) {
			balanceItem = { account: accounts[i], value: 0 };
			accountsMap[accounts[i]._id.toString()] = balanceItem;
			result.push(balanceItem);
		}
		db.conn().collection('transactions_' + coaId, function (err, collection) {
			if (err) { return callback(err); }
			collection.find({ date: { $gte: from, $lte: to } }).
				sort({ date: 1, timestamp: 1 }).
				toArray(function (err, items) {
					var props = [ 'debits', 'credits' ], entry, resultItem, i, j, k;
					for (i = 0; i < items.length; i += 1) {
						for (j = 0; j < props.length; j += 1) {
							for (k = 0; k < items[i][props[j]].length; k += 1) {
								entry = items[i][props[j]][k];
								resultItem = accountsMap[entry.account.toString()];
								if (resultItem) {
									resultItem.value += that.balanceIncrement(
										resultItem.account, entry, props[j]);
									incrementParentBalance(
										resultItem.account.parent, entry, props[j]);
								}
							}
						}
					}
					result.sort(function (a, b) {
						return a.account.number.localeCompare(b.account.number);
					});
					return callback(null, result) 
				});
		});
	}

	db.conn().collection('charts_of_accounts', function (err, collection) {
		collection.aggregate(
			{ $match: { '_id': db.bsonId(coaId) } },
			{ $unwind: '$accounts' },
			{ $match: accountFilter },
			{ $group: { _id: '$_id', 'accounts': {$addToSet:'$accounts'} } },
			function (err, result) {
				if (err) { return callback(err); }
				afterFindAccounts(null, result[0]);
			}
		);
	});
};

Service.prototype.balanceSheet = function (coaId, at, callback) {
	this.balances(coaId, new Date('1000-01-01'), at, 
		{ 'accounts.balanceSheet': true }, callback);
};

Service.prototype.incomeStatement = function (coaId, from, to, callback) {
	this.balances(coaId, from, to, { 'accounts.incomeStatement': true }, 
		function (err, balances) {
			var result = {
				grossRevenue: { balance: 0, details: [] },
				deduction: { balance: 0, details: [] },
				salesTax: { balance: 0, details: [] },
				netRevenue: { balance: 0, details: [] },
				cost: { balance: 0, details: [] },
				grossProfit: { balance: 0, details: [] },
				operatingExpense: { balance: 0, details: [] },
				netOperatingIncome: { balance: 0, details: [] },
				nonOperatingRevenue: { balance: 0, details: [] },
				nonOperatingExpense: { balance: 0, details: [] },
				nonOperatingTax: { balance: 0, details: [] },
				incomeBeforeIncomeTax: { balance: 0, details: [] },
				incomeTax: { balance: 0, details: [] },
				dividends: { balance: 0, details: [] },
				netIncome: { balance: 0, details: [] }
			}, 
			revenueRoots = [], expenseRoots = [], prop, i;

			function isDescendent (account, parents) {
				var i;
				for (i = 0; i < parents.length; i += 1) {
					if (account.number.indexOf(parents[i].number) === 0) {
						return true;
					}
				}
				return false;
			}

			function addBalance (resultItem, balance) {
				if (balance.account.analytic && balance.value > 0) {
					resultItem.balance += balance.value;
					resultItem.details.push(balance);
				}
			}

			if (err) { return callback(err); }
			if (!balances) { return callback(); }
			for (i = 0; i < balances.length; i += 1) {
				if (!balances[i].account.parent) {
					if (balances[i].account.creditBalance) {
						revenueRoots.push(balances[i].account);
					} else {
						expenseRoots.push(balances[i].account);
					}
				}
			}
			for (i = 0; i < balances.length; i += 1) {
				if (balances[i].account.operating && 
							isDescendent(balances[i].account, revenueRoots)) {
					addBalance(result.grossRevenue, balances[i]);
				} else if (balances[i].account.deduction) {
					addBalance(result.deduction, balances[i]);
				} else if (balances[i].account.salesTax) {
					addBalance(result.salesTax, balances[i]);
				} else if (balances[i].account.cost) {
					addBalance(result.cost, balances[i]);
				} else if (balances[i].account.operating && 
							isDescendent(balances[i].account, expenseRoots)) {
					addBalance(result.operatingExpense, balances[i]);
				} else if (balances[i].account.nonOperatingTax) {
					addBalance(result.nonOperatingTax, balances[i]);
				} else if (balances[i].account.incomeTax) {
					addBalance(result.incomeTax, balances[i]);
				} else if (balances[i].account.dividends) {
					addBalance(result.dividends, balances[i]);
				} else if (isDescendent(balances[i].account, revenueRoots)) {
					addBalance(result.nonOperatingRevenue, balances[i]);
				} else {
					addBalance(result.nonOperatingExpense, balances[i]);
				}
			}
			result.netRevenue.balance = 
				result.grossRevenue.balance - result.deduction.balance - 
				result.salesTax.balance;
			result.grossProfit.balance = result.netRevenue.balance - 
				result.cost.balance;
			result.netOperatingIncome.balance = result.grossProfit.balance - 
				result.operatingExpense.balance;
			result.incomeBeforeIncomeTax.balance = 
				result.netOperatingIncome.balance + 
				result.nonOperatingRevenue.balance -
				result.nonOperatingExpense.balance - result.nonOperatingTax.balance;
			result.netIncome.balance = result.incomeBeforeIncomeTax.balance - 
				result.incomeTax.balance - result.dividends.balance;
			for (prop in result) {
				if (result.hasOwnProperty(prop)) {
					if (result[prop].balance === 0 && 
							result[prop].details.length === 0 &&
							prop !== 'netIncome') {
						delete result[prop];
					}
				}
			}
			callback(null, result);
		}
	);
};

function chartsOfAccounts (req, res, next) {
	new Service().chartsOfAccounts(function (err, items) {
		if (err) { return next(err); }
		res.send(items);
	});
}

function chartOfAccounts (req, res, next) {
	new Service().chartOfAccounts(req.params.id, function (err, item) {
		if (err) { return next(err); }
		res.send(item);
	});	
}

function addChartOfAccounts (req, res, next) {
	new Service().addChartOfAccounts(req.body, function (err, item) {
		if (err) { return next(err); }
		res.send(item);
	});
}

function accounts (req, res, next) {
	new Service().chartOfAccounts(req.params.id, function (err, coa) {
		if (err) { return next(err); }
		if (coa === null) { return res.send(500, 'Chart of accounts not found'); }
		res.send(coa.accounts);
	});
}

function account (req, res, next) {
	new Service().account(req.params.accountId, 
		function (err, account) {
			if (err) { return next(err); }
			res.send(account);
		}
	);
}

function addAccount (req, res, next) {
	new Service().addAccount({ _id: req.params.id }, req.body, 
		function (err, account) {
			if (err) { return next(err); }
			res.send(account);
		}
	);
}

function updateAccount (req, res, next) {
	new Service().updateAccount(
		{ _id: req.params.id }, 
		req.params.accountId, 
		req.body, 
		function (err, account) {
			if (err) { return next(err); }
			res.send(account);
		}
	);
}

function transactions (req, res, next) {
	new Service().transactions(req.params.id, function (err, transactions) {
		if (err) { return next(err); }
		res.send(transactions);
	});
}

function addTransaction (req, res, next) {
	new Service().addTransaction(req.params.id, req.body, 
		function (err, transaction) {
			if (err) { return next(err); }
			res.send(transaction);
		}
	);
}

function verifyTimespan (from, to) {
	if (isNaN(from.getTime())) {
		return "'From' date must be informed";
	}
	if (isNaN(to.getTime())) {
		return "'To' date must be informed";
	}
}

function journal (req, res, next) {
	var from = new Date(req.query.from), to = new Date(req.query.to),
		timespanVerification = verifyTimespan(from, to);
	if (timespanVerification) {
		return next(new Error(timespanVerification));
	}
	new Service().journal(req.params.id, from, to, function (err, journal) {
		if (err) { return next(err); }
		res.send(journal);
	});
}

function ledger (req, res, next) {
	var from = new Date(req.query.from), to = new Date(req.query.to),
		timespanVerification = verifyTimespan(from, to);
	if (timespanVerification) {
		return next(new Error(timespanVerification));
	}
	new Service().ledger(req.params.id, req.params.accountId, from, to, 
		function (err, journal) {
			if (err) { return next(err); }
			res.send(journal);
		}
	);
}

function balanceSheet (req, res, next) {
	var at = new Date(req.query.at);
	if (isNaN(at.getTime())) {
		return next(new Error("The 'at' date parameter must be informed"));
	}
	new Service().balanceSheet(req.params.id, at, function (err, balanceSheet) {
		if (err) { return next(err); }
		res.send(balanceSheet);
	});
}

function incomeStatement (req, res, next) {
	var from = new Date(req.query.from), to = new Date(req.query.to),
		timespanVerification = verifyTimespan(from, to);
	if (timespanVerification) {
		return next(new Error(timespanVerification));
	}
	new Service().incomeStatement(req.params.id, from, to, 
		function (err, statement) {
			if (err) { return next(err); }
			res.send(statement);
		}
	);
}

exports.Service = Service;

exports.setup = function (app) {
	app.get('/charts-of-accounts', chartsOfAccounts);
	app.get('/charts-of-accounts/:id', chartOfAccounts);
	app.post('/charts-of-accounts', addChartOfAccounts);
	app.get('/charts-of-accounts/:id/accounts', accounts);
	app.get('/charts-of-accounts/:id/accounts/:accountId', account);
	app.post('/charts-of-accounts/:id/accounts', addAccount);
	app.put('/charts-of-accounts/:id/accounts/:accountId', updateAccount);
	app.get('/charts-of-accounts/:id/transactions', transactions);
	app.post('/charts-of-accounts/:id/transactions', addTransaction);
	app.get('/charts-of-accounts/:id/journal', journal);
	app.get('/charts-of-accounts/:id/accounts/:accountId/ledger', ledger);
	app.get('/charts-of-accounts/:id/balance-sheet', balanceSheet);
	app.get('/charts-of-accounts/:id/income-statement', incomeStatement);
};