// +build appengine

package server

import (
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"io"
	//"io/ioutil"
	"log"
	"net/http"
	//"runtime/debug"
	"fmt"
	"strings"

	"github.com/gorilla/mux"
	"github.com/mcesarhm/geek-accounting/go-server/accounting"
	"github.com/mcesarhm/geek-accounting/go-server/accounting/reporting"
	"github.com/mcesarhm/geek-accounting/go-server/cache"
	"github.com/mcesarhm/geek-accounting/go-server/context"
	"github.com/mcesarhm/geek-accounting/go-server/core"
	"github.com/mcesarhm/geek-accounting/go-server/db"
	"mcesar.io/deb"

	"appengine"
	"appengine/datastore"
	"appengine/taskqueue"
)

const PathPrefix = "/charts-of-accounts"

type readHandlerFunc func(context.Context, map[string]interface{}, map[string]string,
	core.UserKey) (interface{}, error)

type writeHandlerFunc func(context.Context, map[string]interface{}, map[string]string,
	core.UserKey) (interface{}, error)

type writeHandlerFuncMulti func(context.Context, []map[string]interface{}, map[string]string,
	core.UserKey) (interface{}, error)

func init() {
	gob.Register(([]*accounting.Account)(nil))
	gob.Register((*accounting.Account)(nil))
	gob.Register(([]*accounting.Transaction)(nil))
	gob.Register(([]*accounting.ChartOfAccounts)(nil))
	r := mux.NewRouter()
	r.HandleFunc(PathPrefix, getAllHandler(accounting.AllChartsOfAccounts)).Methods("GET")
	r.HandleFunc(PathPrefix, postHandler2(coaPostHandler, true)).Methods("POST")
	r.HandleFunc(PathPrefix+"/{coa}", postHandler(accounting.SaveChartOfAccounts)).Methods("PUT")
	r.HandleFunc(PathPrefix+"/{coa}/accounts", getAllHandler(accounting.AllAccounts)).
		Methods("GET")
	r.HandleFunc(PathPrefix+"/{coa}/accounts/{account}",
		getAllHandler(accounting.GetAccount)).Methods("GET")
	r.HandleFunc(PathPrefix+"/{coa}/accounts", postHandler(accounting.SaveAccount)).Methods("POST")
	r.HandleFunc(PathPrefix+"/{coa}/accounts/{account}",
		postHandler(accounting.SaveAccount)).Methods("PUT")
	r.HandleFunc(PathPrefix+"/{coa}/accounts/{account}",
		deleteHandler(accounting.DeleteAccount)).Methods("DELETE")
	r.HandleFunc(PathPrefix+"/{coa}/transactions",
		getAllHandler(accounting.AllTransactions)).Methods("GET")
	r.HandleFunc(PathPrefix+"/{coa}/transactions",
		postHandlerMulti(accounting.SaveTransaction, false)).Methods("POST")
	r.HandleFunc(PathPrefix+"/{coa}/transactions/{transaction}",
		postHandlerMulti(accounting.SaveTransaction, false)).Methods("PUT")
	r.HandleFunc(PathPrefix+"/{coa}/transactions/{transaction}",
		getAllHandler(accounting.GetTransaction)).Methods("GET")
	r.HandleFunc(PathPrefix+"/{coa}/transactions/{transaction}",
		deleteHandler(accounting.DeleteTransaction)).Methods("DELETE")
	r.HandleFunc(PathPrefix+"/{coa}/balance-sheet", getAllHandler(reporting.Balance)).
		Methods("GET")
	r.HandleFunc(PathPrefix+"/{coa}/journal", getAllHandler(reporting.Journal)).Methods("GET")
	r.HandleFunc(PathPrefix+"/{coa}/accounts/{account}/ledger",
		getAllHandler(reporting.Ledger)).Methods("GET")
	r.HandleFunc(PathPrefix+"/{coa}/income-statement",
		getAllHandler(reporting.IncomeStatement)).Methods("GET")
	r.HandleFunc(PathPrefix+"/{coa}/migration",
		postHandler2(coaMigrationHandler, true)).Methods("POST")
	r.HandleFunc(PathPrefix+"/{coa}/migration/to/{coa2}",
		postHandler2(coaMigrationHandler, true)).Methods("POST")
	r.HandleFunc(PathPrefix+"/{coa}/migration_enqueue",
		postHandler2(coaMigrationEnqueueHandler, true)).Methods("POST")
	r.HandleFunc("/_ah/warmup", func(w http.ResponseWriter, r *http.Request) {
		ac := appengine.NewContext(r)
		c := newContext(ac)
		if err := core.InitUserManagement(c); err != nil {
			http.Error(w, "Internal error:"+err.Error(), http.StatusInternalServerError)
		}
		if err := accounting.UpdateSchema(ac); err != nil {
			http.Error(w, "Internal error:"+err.Error(), http.StatusInternalServerError)
		}
		return
	})
	r.HandleFunc("/update-schema", func(w http.ResponseWriter, r *http.Request) {
		if err := accounting.UpdateSchema(appengine.NewContext(r)); err != nil {
			http.Error(w, "Internal error:"+err.Error(), http.StatusInternalServerError)
		}
		return
	})
	r.HandleFunc("/ping",
		errorHandler(func(w http.ResponseWriter, r *http.Request, userKey core.UserKey) error {
			return nil
		}))
	r.HandleFunc("/password", postHandler(core.ChangePassword)).Methods("PUT")
	r.HandleFunc("/users", getAllHandler(core.AllUsers)).Methods("GET")
	r.HandleFunc("/users/{user}", getAllHandler(core.GetUser)).Methods("GET")
	r.HandleFunc("/users", postHandler(core.SaveUser)).Methods("POST")
	r.HandleFunc("/users/{user}", postHandler(core.SaveUser)).Methods("PUT")
	r.HandleFunc("/users/{user}", deleteHandler(core.DeleteUser)).Methods("DELETE")
	http.Handle("/", r)
}

func coaPostHandler(c context.Context, m map[string]interface{}, p map[string]string,
	u core.UserKey) (interface{}, error) {
	if _, ok := p["coa"]; !ok {
		c := m["_appengine_context"].(appengine.Context)
		_, key, err := deb.NewDatastoreSpace(c, nil)
		if err != nil {
			return nil, err
		}
		p["space"] = key.Encode()
	}
	return accounting.SaveChartOfAccounts(c, m, p, u)
}

func coaMigrationEnqueueHandler(c context.Context, m map[string]interface{}, p map[string]string,
	u core.UserKey) (interface{}, error) {
	ctx := m["_appengine_context"].(appengine.Context)
	t := taskqueue.NewPOSTTask(fmt.Sprintf("/charts-of-accounts/%v/migration", p["coa"]), nil)
	t.Header.Add("Authorization", "Basic bWNlc2FyOmtpbGx0aGVtYWxs")
	t.Header.Set("Content-Type", "application/json")
	t.Payload = []byte("{}")
	if _, err := taskqueue.Add(ctx, t, ""); err != nil {
		return nil, err
	}
	return "ok", nil
}

func coaMigrationHandler(c context.Context, m map[string]interface{}, p map[string]string,
	u core.UserKey) (interface{}, error) {
	ctx := m["_appengine_context"].(appengine.Context)
	var coa, coa2 *accounting.ChartOfAccounts
	if coa2Key, ok := p["coa2"]; ok {
		var (
			s   deb.Space
			err error
		)
		s, coa2, err = space(c, ctx, coa2Key)
		if err != nil {
			return nil, err
		}
		m["space"] = s
	} else if coaKey, ok := p["coa"]; ok {
		var (
			s   deb.Space
			err error
		)
		s, coa, err = space(c, ctx, coaKey)
		if err != nil {
			return nil, err
		}
		m["space"] = s
	}
	if m["space"] == nil || coa2 != nil {
		var (
			key *datastore.Key
			err error
		)
		if coa2 == nil {
			_, key, err = deb.NewDatastoreSpace(ctx, nil)
		} else {
			key = coa2.Space.DsKey
		}
		if err != nil {
			return nil, err
		}
		s := deb.LargeSpaceBuilder(0).NewSpaceWithOffset(nil, 0, 0, nil)
		if result, err := accounting.Migrate(c, coa, p["coa"], p["coa2"], s,
			db.CKey{key}, u); err != nil {
			return nil, err
		} else {
			if err = deb.CopySpaceToDatastore(ctx, key, s); err != nil {
				return nil, err
			}
			return result, nil
		}
	} else {
		return "Already migrated", nil
	}
}

func getAllHandler(f readHandlerFunc) http.HandlerFunc {
	return errorHandler(func(w http.ResponseWriter, r *http.Request, userKey core.UserKey) error {
		params := mux.Vars(r)
		for k, v := range r.URL.Query() {
			params[k] = v[0]
		}
		ctx := appengine.NewContext(r)
		c := newContext(ctx)
		m := map[string]interface{}{}
		if coaKey, ok := params["coa"]; ok {
			space, _, err := space(c, ctx, coaKey)
			if err != nil {
				return err
			}
			m["space"] = space
		}
		items, err := f(c, m, params, userKey)
		if err != nil {
			return err
		}
		return json.NewEncoder(w).Encode(items)
	})
}

func postHandler(f writeHandlerFunc) http.HandlerFunc {
	return postHandler2(f, false)
}

func postHandler2(f writeHandlerFunc, includeContextInMap bool) http.HandlerFunc {
	return errorHandler(func(w http.ResponseWriter, r *http.Request, userKey core.UserKey) error {
		/*
			b, _ := ioutil.ReadAll(r.Body)
			log.Println(string(b))
		*/
		var req interface{}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return badRequest{err}
		}

		m := req.(map[string]interface{})
		ctx := appengine.NewContext(r)
		c := newContext(ctx)
		if includeContextInMap {
			m["_appengine_context"] = ctx
		}
		params := mux.Vars(r)
		if coaKey, ok := params["coa"]; ok {
			space, _, err := space(c, ctx, coaKey)
			if err != nil {
				return err
			}
			m["space"] = space
		}
		item, err := f(c, m, params, userKey)
		if err != nil {
			return badRequest{err}
		}

		json.NewEncoder(w).Encode(item)

		return nil
	})
}

func postHandlerMulti(f writeHandlerFuncMulti, includeContextInMap bool) http.HandlerFunc {
	return errorHandler(func(w http.ResponseWriter, r *http.Request, userKey core.UserKey) error {
		var (
			s   deb.Space
			err error
		)
		ctx := appengine.NewContext(r)
		c := newContext(ctx)
		params := mux.Vars(r)
		if coaKey, ok := params["coa"]; ok {
			if s, _, err = space(c, ctx, coaKey); err != nil {
				return err
			}
		}
		maps := []map[string]interface{}{}
		var req interface{}
		dec := json.NewDecoder(r.Body)
		for {
			if err = dec.Decode(&req); err == io.EOF {
				break
			} else if err != nil {
				return badRequest{err}
			}
			m := req.(map[string]interface{})
			if includeContextInMap {
				m["_appengine_context"] = ctx
			}
			if s != nil {
				m["space"] = s
			}
			maps = append(maps, m)
		}
		item, err := f(c, maps, params, userKey)
		if err != nil {
			return badRequest{err}
		}
		json.NewEncoder(w).Encode(item)
		return nil
	})
}

func deleteHandler(f writeHandlerFunc) http.HandlerFunc {
	return errorHandler(func(w http.ResponseWriter, r *http.Request, userKey core.UserKey) error {
		m := map[string]interface{}{}
		params := mux.Vars(r)
		ctx := appengine.NewContext(r)
		c := newContext(ctx)
		if coaKey, ok := params["coa"]; ok {
			space, _, err := space(c, ctx, coaKey)
			if err != nil {
				return err
			}
			m["space"] = space
		}
		_, err := f(c, m, params, userKey)
		if err != nil {
			return badRequest{err}
		}

		return nil
	})
}

type badRequest struct{ error }

type notFound struct{ error }

// errorHandler wraps a function returning an error by handling the error and
// returning a http.Handler.
// If the error is of the one of the types defined above, it is handled as described for every type.
// If the error is of another type, it is considered as an internal error and its message is logged.
func errorHandler(
	f func(w http.ResponseWriter, r *http.Request, userKey core.UserKey) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var credentials string
		if _, ok := r.Header["Authorization"]; ok {
			credentials = strings.Split(r.Header["Authorization"][0], " ")[1]
		} else {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		b, err := base64.StdEncoding.DecodeString(credentials)
		if err != nil {
			http.Error(w, "Internal error(1):"+err.Error(), http.StatusInternalServerError)
			return
		}
		arr := strings.Split(string(b), ":")
		ctx := appengine.NewContext(r)
		c := newContext(ctx)
		err, ok, userKey := core.Login(c, arr[0], arr[1])
		if err != nil {
			http.Error(w, "Internal error(2):"+err.Error(), http.StatusInternalServerError)
			return
		}
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		err = f(w, r, userKey)
		if err == nil {
			return
		}
		switch err.(type) {
		case badRequest:
			ctx.Infof(err.Error())
			http.Error(w, "Error: "+err.Error(), http.StatusBadRequest)
		case notFound:
			http.Error(w, "Error: item not found", http.StatusNotFound)
		default:
			log.Println(err)
			http.Error(w, "Internal error(3):"+err.Error(), http.StatusInternalServerError)
		}
	}
}

func newContext(ac appengine.Context) context.Context {
	return context.Context{Db: db.NewAppengineDb(ac), Cache: cache.NewAppengineCache(ac)}
}

func space(c context.Context, ctx appengine.Context, coaKey string) (deb.Space,
	*accounting.ChartOfAccounts, error) {
	var coas []*accounting.ChartOfAccounts
	keys, _, err := c.Db.GetAllFromCache("ChartOfAccounts", "", &coas, nil, nil, c.Cache,
		"ChartOfAccounts")
	if err != nil {
		return nil, nil, err
	}
	for i, coa := range coas {
		if keys[i].Encode() == coaKey {
			if coa.Space.IsZero() {
				return nil, coa, nil
			}
			k, err := datastore.DecodeKey(coa.Space.Encode())
			if err != nil {
				return nil, nil, err
			}
			s, _, err := deb.NewDatastoreSpace(ctx, k)
			if err != nil {
				return nil, nil, err
			}
			return s, coa, nil
		}
	}
	return nil, nil, fmt.Errorf("Chart of accounts not found")
}
