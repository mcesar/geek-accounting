Geek Accounting
===============

Getting Started 
---------------

### Node version:

Install Node.js and MongoDB. Start `mongod`.

Run `npm install` in the command line at the project folder.

Run `node .` at the project folder.

To test, use `curl`:

Retrieve users:
```bash
curl -k -u admin:admin https://localhost:8001/users
```
Add new user:
```bash
curl -k -u admin:admin -X POST -H 'Content-Type: application/json' 
	-d '{ "user": "test", "password": "test", "name": "test" }' https://localhost:8001/users
```

### Go version:

`$ go get -d github.com/mcesarhm/geek-accounting`

`$ cd $GOPATH/src/github.com/mcesarhm/geek-accounting/go-server`

`$ go test -tags 'test inmemory' ./...`

To test with App Engine:

`$ goapp test -tags 'test appengine' ./...`