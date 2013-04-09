/*jslint node: true */
'use strict';

var mongodb = require('mongodb');
var BSON = require('mongodb').BSONPure;

var config = require('../config/config');
var util = require('./utils').utils;

function Database() {
	
	var setupReplicaSet, setupDatabase, db;

	setupReplicaSet = function () {
		var servers = [], i, replSet;

		for (i = 0; i < config.dbServerAddress.length; i += 1) {
			servers.push(
				new mongodb.Server(config.dbServerAddress[i], config.dbServerPort[i]));
		}

		replSet = 
			new mongodb.ReplSetServers(servers, { rs_name: config.replicaSet });

		db = new mongodb.Db(config.dbName, replSet, {w: 1});
	};

	setupDatabase = function () {
		var server = new mongodb.Server(config.dbServerAddress,
			config.dbServerPort);

		db = new mongodb.Db(config.dbName, server, { w: config.dbWrite });
	};

	if (config.dbServerAddress instanceof Array) {
		setupReplicaSet();
	} else {
		setupDatabase();
	}

	this.conn = function() {
		return db;
	};

}

Database.prototype.open = function(callback) {
	this.conn().open(function(err, db) {
		if (err) { return callback(err); }
		db.collection('users', function(err, collection) {
			collection.ensureIndex(
				{ user: 1 }, 
				{ unique: true }, 
				function(err, indexName) {
					if (err) { return callback(err); }
					collection.findOne({ user: 'admin' }, function(err, item) {
						var user;
						if (err) { return callback(err); }
						if (item === null) {
							user = { user: 'admin', password: util.sha1Hex('admin') };
							collection.insert(user, function(err, item) {
								if (err) { return callback(err); }
								callback();
							});
						} else {
							callback();
						}
					});
				}
			);
		});
	});
};

Database.prototype.find = function(collectionName, callback) {
	this.conn().collection(collectionName, function(err, collection) {
		if (err) { return callback(err); }
		collection.find().toArray(function(err, items) {
			if (err) { return callback(err); }
			callback(null, items);
		});
	});
};

Database.prototype.findOne = function(collectionName, where, callback) {
	this.conn().collection(collectionName, function(err, collection) {
		if (err) { return callback(err); }
		collection.findOne(where, function (err, item) {
			if (err) { return callback(err); }
			callback(null, item);
		});
	});
};

Database.prototype.insert = function(collectionName, item, callback) {
	this.conn().collection(collectionName, function(err, collection) {
		if (err) { return callback(err); }
		collection.insert(item, function(err, newItem) {
			if (err) { return callback(err); }
			callback(null, newItem);
		});
	});
};

Database.prototype.update = function(collectionName, where, item, callback) {
	this.conn().collection(collectionName, function(err, collection) {
		if (err) { return callback(err); }
		collection.update(where, item, function(err, newItem) {
			if (err) { return callback(err); }
			callback(null, newItem);
		});
	});
};

/*jslint nomen: true*/
Database.prototype.bulkUpdate = function(
		collectionName, itemsToUpdate, validator, callback) {
	var that = this;
	if (typeof callback === 'undefined') { 
		callback = validator; 
		validator = undefined;
	}
	this.conn().collection(collectionName, function(err, collection) {
		var itemsToAdd = [], itemsToRemove = [], rejects = [], result, i;
		if (err) { return callback(err); }
		for (i = itemsToUpdate.length - 1; i >= 0; i -= 1) {
			if (typeof itemsToUpdate[i]._id === 'undefined') {
				itemsToAdd.push(itemsToUpdate.splice(i, 1)[0]);
			} else if (itemsToUpdate[i]._remove) {
				itemsToRemove.push(itemsToUpdate.splice(i, 1)[0]);
			}
		}
		result = { 
				a: itemsToAdd.slice(0), 
				u: itemsToUpdate.slice(0), 
				r: itemsToRemove.slice(0) 
		};
		function update() {
			var item, _id, validation;
			if (itemsToUpdate.length === 0) {
				callback(null, result.a, result.u, result.r, rejects);
				return;
			}
			item = itemsToUpdate.shift();
			if (typeof validator === 'function') {
				validation = validator(item);
			} else {
				validation = null;
			}
			if (validation) {
				rejects.push({ item: item, validationError: validation });
				result.u.splice(result.u.indexOf(item), 1);
				update();
			} else {
				_id = item._id;
				delete item._id;
				collection.update(
					{ _id: that.bsonId(_id) }, 
					{ $set: item }, 
					function(err, item) {
						if (err) { return callback(err); }
						update();
					}
				);
			}
		}
		function add() {
			var item, validation;
			if (itemsToAdd.length === 0) {
				update();
				return;
			}
			item = itemsToAdd.shift();
			if (typeof validator === 'function') {
				validation = validator(item);
			} else {
				validation = null;
			}
			if (validation) {
				rejects.push({ item: item, validationError: validation });
				result.a.splice(result.a.indexOf(item), 1);
				add();
			} else {
				collection.insert(item, function(err, item) {
					if (err) { return callback(err); }
					add();
				});				
			}
		}
		function remove() {
			var _id;
			if (itemsToRemove.length === 0) {
				add();
				return;
			}
			_id = itemsToRemove.shift()._id;
			collection.remove({ _id: that.bsonId(_id) }, function(err, count) {
				if (err) { return callback(err); }
				remove();
			});
		}
		remove();
	});
};
/*jslint nomen: false*/

Database.prototype.bsonId = function (id) {
	return id.length === 24 ? new BSON.ObjectID(id) : id;
};

Database.prototype.injectRejects = function (items, rejects) {
	var i, j;

	for (i = 0; i < rejects.length; i += 1) {
		if (typeof rejects[i].item._id === 'undefined') {
			rejects[i].item.validationError = rejects[i].validationError;
			items.push(rejects[i].item);
		} else {
			for (j = 0; j < items.length; j += 1) {
				if (items[j]._id.toString() === rejects[i].item._id.toString()) {
					rejects[i].item.validationError = rejects[i].validationError;
					items[j] = rejects[i].item;
				}
			}
		}
	}
};

exports.db = new Database();