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

Service.prototype.chartsOfAccounts = function (callback) {
	db.find('charts_of_accounts', function (err, items) {
		if (err) { return callback(err); }
		callback(null, items);
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
	account.number = '' + account.number;
	this.accountValidation(coa, account, function (err, validation) {
		if (err) { return callback(err); }
		if (validation) { return callback(new Error(validation)); }
		account._id = new BSON.ObjectID();
		if (account.parent) { account.parent = db.bsonId(account.parent); }
		account.analytic = true;
		db.update('charts_of_accounts', { _id: db.bsonId(coa._id) }, 
			{ $push: { accounts: account } },
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
			var prop;
			for (prop in oldAccount) {
				if (oldAccount.hasOwnProperty(prop) && prop !== 'name') { 
					account[prop] = oldAccount[prop]; 
				}
			}
			db.update('charts_of_accounts', 
				{ _id: db.bsonId(coa._id), 'accounts._id': db.bsonId(accountId) }, 
				{ $set: { 'accounts.$': account } },
				function (err, item) {
					if (err) { return callback(err); }
					callback(null, account);
				}
			);
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

exports.Service = Service;

exports.setup = function (app) {
	app.get('/charts-of-accounts', chartsOfAccounts);
	app.get('/charts-of-accounts/:id', chartOfAccounts);
	app.post('/charts-of-accounts', addChartOfAccounts);
	app.get('/charts-of-accounts/:id/accounts', accounts);
	app.get('/charts-of-accounts/:id/accounts/:accountId', account);
	app.post('/charts-of-accounts/:id/accounts', addAccount);
	app.put('/charts-of-accounts/:id/accounts/:accountId', updateAccount);
};