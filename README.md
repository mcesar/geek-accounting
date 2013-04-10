Geek Accounting
===============

Getting Started
---------------

Install Node.js and MongoDB. Start `mongod`.

Run `npm install` in the command line at the project folder.

Run `node .` at the project folder.

To test, use `curl`:

* retrieve users: `curl -k -u admin:admin https://localhost:8001/users`
* add new user: `curl -k -u admin:admin -X POST -H 'Content-Type: application/json' -d '{ "user": "test", "password": "test", "name": "test" }' https://localhost:8001/users`