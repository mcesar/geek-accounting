var assert = require('assert');

var config = require('../config/config');

config.dbName = 'geek_accounting_tests'

var db = require('../lib/database').db;
var CoaService = require('../plugins/chart-of-accounts').Service;


describe('Coa', function	() {

	var coa = { name: 'n' }, accounts, transactions,
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

	it('should get no transactions', function (done) {
		new CoaService().transactions(coa._id, function (err, txs) {
			if (err) { throw err; }
			assert.equal(0, txs.length);
			done();
		});
	});

	it('should get journal', function (done) {
		var service = new CoaService();

	accounts = [ 
		{ name: 'asset', 
			number: '1', 
			balanceSheet: true, 
			debitBalance: true },
		{ name: 'liability', 
			number: '2', 
			balanceSheet: true, 
			creditBalance: true },
		{ name: 'expense', 
			number: '3', 
			incomeStatement: true, 
			debitBalance: true },
		{ name: 'revenue', 
			number: '4', 
			incomeStatement: true, 
			creditBalance: true } ];

	transactions = [ 
		{ debits: [ { account: accounts[0], value: 1} ],
			credits: [ { account: accounts[1], value: 1} ],
			date: new Date('2013-05-01'),
			memo: 'test1' }, 
		{ debits: [ { account: accounts[2], value: 1} ],
			credits: [ { account: accounts[0], value: 1} ],
			date: new Date('2013-05-01'),
			memo: 'test2' },
		{ debits: [ { account: accounts[0], value: 2} ],
			credits: [ { account: accounts[3], value: 2} ],
			date: new Date('2013-05-01'),
			memo: 'test2' } ];

		function verify () {
			service.journal(coa._id, new Date('2013-05-01'), new Date('2013-05-01'),
				function (err, journal) {
					if (err) { throw err; }
					assert.equal(3, journal.length);
					assert.equal('asset', journal[0].debits[0].account.name);
					assert.equal('liability', journal[0].credits[0].account.name);
					assert.equal('expense', journal[1].debits[0].account.name);
					assert.equal('asset', journal[1].credits[0].account.name);
					assert.equal('asset', journal[2].debits[0].account.name);
					assert.equal('revenue', journal[2].credits[0].account.name);
					done();
				}
			);
		}

		function addTransaction (i) {
			if (i >= transactions.length) { return verify(); }
			transactions[i].debits[0].account = 
				transactions[i].debits[0].account._id;
			transactions[i].credits[0].account = 
				transactions[i].credits[0].account._id;
			service.addTransaction(coa._id, transactions[i], function (err, tx) {
				if (err) { throw err; }
				transactions[i]._id = tx._id.toString();
				addTransaction(i + 1);
			});
		}

		function addAccount (i) {
			if (i >= accounts.length) { return addTransaction(0); }
			service.addAccount(coa, accounts[i], function (err, newAcc) {
				if (err) { throw err; }
				accounts[i]._id = newAcc._id.toString();
				addAccount(i + 1);
			});
		};

		db.update('charts_of_accounts', {}, { $unset: { 'accounts': 1 } }, 
			function (err) {
				if (err) { throw err; }
				addAccount(0);
			}
		);

	});

	it('should get ledger', function (done) {
		new CoaService().ledger(coa._id, accounts[0]._id, 
			new Date('2013-05-01'), new Date('2013-05-01'),
			function (err, ledger) {
				if (err) { throw err; }
				assert.equal(0, ledger.balance);
				assert.equal('asset', ledger.account.name);
				assert.equal(3, ledger.entries.length);
				assert.equal('liability', ledger.entries[0].counterpart.name);
				assert.equal(1, ledger.entries[0].debit);
				assert.equal('expense', ledger.entries[1].counterpart.name);
				assert.equal(1, ledger.entries[1].credit);
				assert.equal('revenue', ledger.entries[2].counterpart.name);
				assert.equal(2, ledger.entries[2].debit);
				done();
			}
		);
	});

	it('should get balance sheet', function (done) {
		new CoaService().balanceSheet(coa._id, new Date('2013-05-01'),
			function (err, bs) {
				if (err) { throw err; }
				assert.equal(2, bs.length);
				assert.equal('asset', bs[0].account.name);
				assert.equal(2, bs[0].value);
				assert.equal('liability', bs[1].account.name);
				assert.equal(1, bs[1].value);
				done();
			}
		);
	});

	it('should get income statement', function (done) {
		new CoaService().incomeStatement(coa._id, 
			new Date('2013-05-01'), new Date('2013-05-01'),
			function (err, is) {
				if (err) { throw err; }
				assert.equal(2, is.nonOperatingRevenue.balance);
				assert.equal(1, is.nonOperatingExpense.balance);
				assert.equal(1, is.incomeBeforeIncomeTax.balance);
				assert.equal(1, is.netIncome.balance);
				done();
			}
		);
	});

});