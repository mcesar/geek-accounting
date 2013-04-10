/*jslint node: true */
'use strict';

var express = require('express');

var utils = require('../lib/utils').utils;
var db = require('../lib/database').db;

function Service() {
}

Service.prototype.isPasswordValid = function (user, password, callback) {
	db.findOne(
		'users', 
		{ user: user, password: utils.sha1Hex(password) }, 
		function (err, item) {
			if (err) { return callback(err); }
			var validLoginAndPassword = item !== null;
			callback(null, validLoginAndPassword);
		}
	);
};

Service.prototype.users = function (callback) {
	var that = this;
	db.find('users', function (err, items) {
		if (err) { return callback(err); }
		if (that.rejects) {
			db.injectRejects(items, that.rejects);
			delete that.rejects;
		}
		callback(null, items);
	});
};

Service.prototype.userValidation = function (user) {
	if (utils.isEmpty(user.user)) { return "The user must be informed"; }
	if (utils.isEmpty(user.name)) { return "The name must be informed"; }
	if (utils.isEmpty(user._id) && utils.isEmpty(user.password)) { 
		return "The password must be informed";
	}
};

Service.prototype.addUser = function (user, callback) {
	var validation = this.userValidation(user);
	if (validation) { return callback(new Error(validation)); }
	user.password = utils.sha1Hex(user.password);
	db.insert('users', user, function (err, item) {
		if (err) { return callback(err); }
		callback(null, item);
	});
};

Service.prototype.updateUsers = function (usersToUpdate, callback) {
	var that = this, i;
	for (i = usersToUpdate.length - 1; i >= 0; i -= 1) {
		if (typeof usersToUpdate[i].password !== 'undefined') {
			usersToUpdate[i].password = utils.sha1Hex(usersToUpdate[i].password);
		}
	}
	db.bulkUpdate('users', usersToUpdate, this.userValidation, 
		function (err, added, updated, removed, rejects) {
			if (err) { return callback(err); }
			that.rejects = rejects;
			that.users(callback);
		}
	);
};

function login (user, password, callback) {
	new Service().isPasswordValid(user, password, function (err, valid) {
		callback(err, valid);
	});
}

function users (req, res, next) {
	new Service().users(function (err, users) {
		if (err) { return next(err); }
		res.send(users);
	});
}

function addUser (req, res, next) {
	var user = req.body;
	new Service().addUser(user, function (err, user) {
		if (err) { return next(err); }
		res.send(user);
	});
}

function updateUsers (req, res, next) {
	var usersToUpdate = req.body;
	new Service().updateUsers(usersToUpdate, function (err, users) {
		if (err) { return next(err); }
		res.send(users);
	});
}

exports.Service = Service;

exports.setup = function (app) {

	app.use(express.basicAuth(login));

	app.get('/users', users);
	app.post('/users', addUser);
	app.put('/users', updateUsers);
};