var assert = require('assert');

var config = require('../config/config');

config.dbName = 'geek_accounting_tests'

var db = require('../lib/database').db;
var CoaService = require('../plugins/chart-of-accounts').Service;


describe('Coa', function	() {

	var coa = { name: 'n' },
		account = { 
			name: 'an', 
			number: '1', 
			balanceSheet: true, 
			debitBalance: true },
		transaction = {
			debits: [ { account: '', value: 1} ],
			credits: [ { account: '', value: 1} ],
			date: new Date('2013-05-01'),
			memo: 'test'
		};

	before(function (done) {
		db.open(function (err) {
			if (err) { throw err; }
			db.conn().dropDatabase(done);
		});
	});

	it('should save coa without error', function (done) {
		new CoaService().addChartOfAccounts(coa, function (err, newCoa) {
			if (err) { throw err; }
			coa._id = newCoa._id.toString();
			done();
		});
	});

	it('should get the same coa', function (done) {
		new CoaService().chartsOfAccounts(function (err, coas) {
			if (err) { throw err; }
			assert.equal(1, coas.length);
			assert.equal('n', coas[0].name);
			assert.equal(coa._id, coas[0]._id);
			done();
		});
	});

	it('should get the same coa', function (done) {
		new CoaService().chartOfAccounts(coa._id, function (err, coa) {
			if (err) { throw err; }
			assert.equal('n', coa.name);
			done();
		});
	});

	it('should save account without error', function (done) {
		new CoaService().addAccount(coa, account, function (err, newAcc) {
			if (err) { throw err; }
			account._id = newAcc._id.toString();
			done();
		});
	});

	it('should get the same account', function (done) {
		new CoaService().chartOfAccounts(coa._id, function (err, coa) {
			if (err) { throw err; }
			assert.equal(1, coa.accounts.length);
			assert.equal('an', coa.accounts[0].name);
			assert.equal(account._id, coa.accounts[0]._id);
			done();
		});
	});

	it('should get the same account', function (done) {
		new CoaService().account(account._id, function (err, account) {
			if (err) { throw err; }
			assert.equal('an', account.name);
			done();
		});
	});

	it('should update without error', function (done) {
		new CoaService().updateAccount({ _id: coa._id }, account._id, 
			{ name: 'ann', number: '1', balanceSheet: true, debitBalance: true },
			function (err, account) {
				if (err) { throw err; }
				done();
			}
		);
	});

	it('should get the same account', function (done) {
		new CoaService().account(account._id, function (err, account) {
			if (err) { throw err; }
			assert.equal('ann', account.name);
			done();
		});
	});

	it('should save transaction without error', function (done) {
		var accountId = account._id;
		transaction.debits[0].account = accountId;
		transaction.credits[0].account = accountId;
		new CoaService().addTransaction(coa._id, transaction, 
			function (err, newTx) {
				if (err) { throw err; }
				transaction._id = newTx._id.toString();
				done();
			}
		);
	});

	it('should get the same transaction', function (done) {
		new CoaService().transactions(coa._id, function (err, txs) {
			if (err) { throw err; }
			assert.equal(1, txs.length);
			assert.equal('test', txs[0].memo);
			done();
		});
	});

	it('should delete transaction without error', function (done) {
		new CoaService().deleteTransaction(coa._id, transaction._id, 
			function (err) {
				if (err) { throw err; }
				done();
			}
		);
	});

	it('should get the no transactions', function (done) {
		new CoaService().transactions(coa._id, function (err, txs) {
			if (err) { throw err; }
			assert.equal(0, txs.length);
			done();
		});
	});

});