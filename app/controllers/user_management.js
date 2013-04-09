/*jslint node: true */
'use strict';

var express = require('express');

var utils = require('../../lib/utils').utils;
var db = require('../../lib/database').db;

function UserManagement() {
}

var controller = new UserManagement();

UserManagement.prototype.login = function (user, password, callback) {
	db.findOne(
		'users', 
		{ user: user, password: utils.sha1Hex(password) }, 
		function (err, item) {
			if (err) { return callback(err); }
			var validLoginAndPassword = item !== null;
			callback(err, validLoginAndPassword);
		}
	);
};

UserManagement.prototype.users = function (req, res, next) {
	db.find('users', function (err, items) {
		if (err) { return next(err); }
		if (controller.rejects) {
			db.injectRejects(items, controller.rejects);
			delete controller.rejects;
		}
		res.send(items);
	});
};

UserManagement.prototype.addUser = function (req, res, next) {
	var user = req.body;
	user.password = utils.sha1Hex(user.password);
	db.insert('users', user, function (err, item) {
		if (err) { return next(err); }
		res.send(item);
	});
};

UserManagement.prototype.updateUsers = function (req, res, next) {
	var usersToUpdate = req.body, i;
	for (i = usersToUpdate.length - 1; i >= 0; i -= 1) {
		if (typeof usersToUpdate[i].password !== 'undefined') {
			usersToUpdate[i].password = utils.sha1Hex(usersToUpdate[i].password);
		}
	}
	db.bulkUpdate('users', usersToUpdate, 
		function (user) {
			if (utils.isEmpty(user.user)) { return "O usuário deve ser informado"; }
			if (utils.isEmpty(user.name)) { return "O nome deve ser informado"; }
			if (utils.isEmpty(user._id) && utils.isEmpty(user.password)) { 
				return "A senha deve ser informada";
			}
		}, 
		function (err, added, updated, removed, rejects) {
			if (err) { return next(err); }
			controller.rejects = rejects;
			controller.users(req, res, next);
		}
	);
};

exports.setup = function (app) {
	
	app.use(express.basicAuth(controller.login));

	app.get('/users', controller.users);
	app.post('/users', controller.addUser);
	app.put('/users', controller.updateUsers);
};