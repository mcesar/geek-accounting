/*jslint node: true */
'use strict';

var express = require('express');

var utils = require('../lib/utils').utils;
var db = require('../lib/database').db;

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
	if (typeof account.parent !== 'undefined') {
		this.account(account.parent._id, function (err, parentAccount) {
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

Service.prototype.addAccount = function (coa, account, callback) {
	this.accountValidation(coa, account, function (err, validation) {
		if (validation) { return callback(new Error(validation)); }
		db.update('charts_of_accounts', { _id: db.bsonId(coa._id) }, 
			{ $push: { accounts: account } },
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

function addAccount (req, res, next) {
	new Service().addAccount({ _id: req.params.id }, req.body, 
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
	app.post('/charts-of-accounts/:id/accounts', addAccount);
};