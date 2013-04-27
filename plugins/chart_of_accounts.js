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
		{ name: 'operational', classification: 'income statement attribute' },
		{ name: 'deduction', classification: 'income statement attribute' },
		{ name: 'salesTax', classification: 'income statement attribute' },
		{ name: 'cost', classification: 'income statement attribute' },
		{ name: 'incomeTax', classification: 'income statement attribute' },
		{ name: 'profitFromAssociates', 
			classification: 'income statement attribute' }
	];
}

Service.prototype.chartOfAccountsValidation = function (coa) {
	if (utils.isEmpty(coa.name)) { return "The name must be informed"; }
};

Service.prototype.accountValidation = function (coa, account, callback) {
	var that = this;
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
	db.conn().collection('transactions_' + coaId, function (err, collection) {
		if (err) { return callback(err); }
		collection.find({ date: { $gte: from , $lte: to } }).
			sort({ date: 1, timestamp: 1 }).
			toArray(function (err, items) {
				callback(err, items);
			});
	});
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

function journal (req, res, next) {
	var from = new Date(req.query.from), to = new Date(req.query.to);
	if (isNaN(from.getTime())) {
		return next(new Error("'From' date must be informed"));
	}
	if (isNaN(to.getTime())) {
		return next(new Error("'To' date must be informed"));
	}
	new Service().journal(req.params.id, from, to, function (err, journal) {
		if (err) { return next(err); }
		res.send(journal);
	});
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
};