// +build appengine

package server

import (
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
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
)

const PathPrefix = "/charts-of-accounts"

type readHandlerFunc func(context.Context, map[string]interface{}, map[string]string,
	core.UserKey) (interface{}, error)

type writeHandlerFunc func(context.Context, map[string]interface{}, map[string]string,
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
	r.HandleFunc(PathPrefix+"/{coa}/accounts", getAllHandler(accounting.AllAccounts)).Methods("GET")
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
		postHandler(accounting.SaveTransaction)).Methods("POST")
	r.HandleFunc(PathPrefix+"/{coa}/transactions/{transaction}",
		postHandler(accounting.SaveTransaction)).Methods("PUT")
	r.HandleFunc(PathPrefix+"/{coa}/transactions/{transaction}",
		getAllHandler(accounting.GetTransaction)).Methods("GET")
	r.HandleFunc(PathPrefix+"/{coa}/transactions/{transaction}",
		deleteHandler(accounting.DeleteTransaction)).Methods("DELETE")
	r.HandleFunc(PathPrefix+"/{coa}/balance-sheet", getAllHandler(reporting.Balance)).Methods("GET")
	r.HandleFunc(PathPrefix+"/{coa}/journal", getAllHandler(reporting.Journal)).Methods("GET")
	r.HandleFunc(PathPrefix+"/{coa}/accounts/{account}/ledger",
		getAllHandler(reporting.Ledger)).Methods("GET")
	r.HandleFunc(PathPrefix+"/{coa}/income-statement",
		getAllHandler(reporting.IncomeStatement)).Methods("GET")
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
			space, err := space(c, ctx, coaKey)
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
			space, err := space(c, ctx, coaKey)
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

func deleteHandler(f writeHandlerFunc) http.HandlerFunc {
	return errorHandler(func(w http.ResponseWriter, r *http.Request, userKey core.UserKey) error {
		m := map[string]interface{}{}
		params := mux.Vars(r)
		ctx := appengine.NewContext(r)
		c := newContext(ctx)
		if coaKey, ok := params["coa"]; ok {
			space, err := space(c, ctx, coaKey)
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
		c := newContext(appengine.NewContext(r))
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

func space(c context.Context, ctx appengine.Context, coaKey string) (deb.Space, error) {
	var coas []*accounting.ChartOfAccounts
	keys, _, err := c.Db.GetAllFromCache("ChartOfAccounts", "", &coas, nil, nil, c.Cache,
		"ChartOfAccounts")
	if err != nil {
		return nil, err
	}
	for i, coa := range coas {
		if keys[i].Encode() == coaKey {
			k, err := datastore.DecodeKey(coa.Space.Encode())
			if err != nil {
				return nil, err
			}
			s, _, err := deb.NewDatastoreSpace(ctx, k)
			if err != nil {
				return nil, err
			}
			return s, nil
		}
	}
	return nil, fmt.Errorf("Chart of accounts not found")
}
