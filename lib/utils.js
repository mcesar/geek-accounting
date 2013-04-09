/*jslint node: true */
'use strict';

var crypto = require('crypto');

function Utils() {
}

Utils.prototype.sha1Hex = function (aString) {
	return crypto.createHash('sha1').update((aString || '').toString())
		.digest('hex');
};

Utils.prototype.isEmpty = function (aString) {
	return !aString || aString.toString().trim().length === 0
}

exports.utils = new Utils();