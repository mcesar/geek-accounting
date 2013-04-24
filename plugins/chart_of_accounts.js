/*jslint node: true */
'use strict';

var express = require('express');

var utils = require('../lib/utils').utils;
var db = require('../lib/database').db;
var BSON = require('mongodb').BSONPure;

function Service () {
}

Service.prototype.chartOfAccountsValidation = function (coa) {
	if (utils.isEmpty(coa.name)) { return "The name must be informed"; }
};

Service.prototype.accountValidation = function (coa, account, callback) {
	if (utils.isEmpty(account.number)) {
		return callback(null, "The number must be informed"); 
	}
	if (utils.isEmpty(account.name)) {
		return callback(null, "The name must be informed");
	}
	if (typeof account.balanceSheet !== 'boolean' &&
				typeof account.incomeStatement !== 'boolean') {
		return callback(null, "The statement must be informed");
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
		this.account(account.parent, function (err, parentAccount) {
			if (err) { return callback(err); }
			if (account.number.indexOf(parentAccount.number) !== 0) {
				return callback(null, 
					"The number must start with parent's number");
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
		coa.findOne({}, 
			{ accounts: { $elemMatch: { _id: db.bsonId(accountId) } } },
			function (err, item) {
				if (err) { return callback(err); }
				callback(null, item === null ? null : item.accounts[0]);
			}
		);
	});
};

Service.prototype.addAccount = function (coa, account, callback) {
	this.accountValidation(coa, account, function (err, validation) {
		if (validation) { return callback(new Error(validation)); }
		account._id = new BSON.ObjectID();
		db.update('charts_of_accounts', { _id: db.bsonId(coa._id) }, 
			{ $push: { accounts: account } },
			function (err, item) {
				if (err) { return callback(err); }
				callback(null, account);
			}
		);
	});
};

Service.prototype.updateAccount = function (coa, accountId, account, 
		callback) {
	this.accountValidation(coa, account, function (err, validation) {
		if (validation) { return callback(new Error(validation)); }
		account._id = db.bsonId(accountId);
		db.update('charts_of_accounts', 
			{ _id: db.bsonId(coa._id), 'accounts._id': db.bsonId(accountId) }, 
			{ $set: { 'accounts.$': account } },
			function (err, item) {
				if (err) { return callback(err); }
				callback(null, account);
			}
		);
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