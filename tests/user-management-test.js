var assert = require('assert');

var config = require('../config/config');

config.dbName = 'geek_accounting_tests'

var db = require('../lib/database').db;
var UserService = require('../plugins/user-management').Service;


describe('User', function	() {
	
	var user = { user: 'u', name: 'n', password: 'p' };

	before(function (done) {
		db.open(function (err) {
			if (err) { throw err; }
			db.conn().dropDatabase(done);
		});
	});

	it('should save without error', function (done) {
		new UserService().addUser(user, function (err, newUser) {
			if (err) { throw err; }
			user._id = newUser._id.toString();	
			done();
		});
	});

	it('should get the same user', function (done) {
		new UserService().users(function (err, users) {
			if (err) { throw err; }
			assert.equal(1, users.length);
			assert.equal('u', users[0].user);
			assert.equal(user._id, users[0]._id);
			done();
		});
	});

	it('should update without error', function (done) {
		var updatedUser = { _id: user._id, user: 'uu', name: 'nn', password: 'pp' };
		new UserService().updateUsers([ updatedUser ], done);
	});

	it('should save get the updated user', function (done) {
		new UserService().users(function (err, users) {
			if (err) { throw err; }
			assert.equal(1, users.length);
			assert.equal('uu', users[0].user);
			done();
		});
	});

	it('should not save with error', function (done) {
		new UserService().addUser({}, function (err, newUser) {
			assert.equal('The user must be informed', err.message);
			done();
		});
	});

	it('should not save with error', function (done) {
		new UserService().addUser({user:'u'}, function (err, newUser) {
			assert.equal('The name must be informed', err.message);
			done();
		});
	});

	it('should not save with error', function (done) {
		new UserService().addUser({user:'u',name:'n'}, function (err, newUser) {
			assert.equal('The password must be informed', err.message);
			done();
		});
	});

});