/*jslint node: true */
'use strict';

var express = require('express');

var utils = require('../../lib/utils').utils;
var db = require('../../lib/database').db;

function ChartOfAccounts () {
}

var controller = new ChartOfAccounts();

ChartOfAccounts.prototype.chartOfAccountsValidation = function (coa) {
	if (utils.isEmpty(coa.name)) { return "The name must be informed"; }
};

ChartOfAccounts.prototype.chartsOfAccounts = function (req, res, next) {
	db.find('charts_of_accounts', function (err, items) {
		if (err) { return next(err); }
		res.send(items);
	});
};

ChartOfAccounts.prototype.chartOfAccounts = function (req, res, next) {
	var id = req.params.id;
	db.findOne('charts_of_accounts', 
		{ _id: db.bsonId(id) }, 
		function (err, item) {
			if (err) { return next(err); }
			res.send(item);
		}
	);	
};

ChartOfAccounts.prototype.addChartOfAccounts = function (req, res, next) {
	var coa = req.body
		, validation = controller.chartOfAccountsValidation(coa);
	if (validation) { return next(new Error(validation)); }
	db.insert('charts_of_accounts', coa, function (err, item) {
		if (err) { return next(err); }
		res.send(item);
	});
};

exports.setup = function (app) {
	var controller = new ChartOfAccounts();

	app.get('/charts-of-accounts', controller.chartsOfAccounts);
	app.get('/charts-of-accounts/:id', controller.chartOfAccounts);
	app.post('/charts-of-accounts', controller.addChartOfAccounts);
}