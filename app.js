/*jslint node: true */
'use strict';

var http = require('http');
var https = require('https');
var express = require('express');
var fs = require('fs');
var db = require('./lib/database').db;
var config = 
	require('./config/' + (process.env.GA_CONFIG_PREFIX || '') + 'config');

var app = express();

app.configure(function () {
	app.use(express.bodyParser());
	app.use(function (req, res, next) {
		res.header("Cache-Control", "no-cache, no-store, must-revalidate");
		res.header("Pragma", "no-cache");
		res.header("Expires", 0);
		next();
	});
	app.use(function (err, req, res, next) {
		console.error(err.stack);
		res.send(500, 'Internal error: ' + err.message);
	});

	fs.readdir('./plugins', function (err, files) {
		if (err) { throw err; }
		files.forEach(function (file) {
			require('./plugins/' + file).setup(app);
		});
	});

});

db.open(function (err) {
	var options;

	if (err) { return console.error(err.stack); }

	if (config.ssl) {
		options = {
		  key: fs.readFileSync('config/privatekey.pem'),
		  cert: fs.readFileSync('config/certificate.pem')
		};
		https.createServer(options, app).listen(config.webServerPort);
	} else {
		http.createServer(app).listen(config.webServerPort);
	}

	console.log('server listening');
});