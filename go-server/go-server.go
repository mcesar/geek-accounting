package server

import (
	"encoding/base64"
	"encoding/json"
	//"io/ioutil"
	"log"
	"net/http"
	//"runtime/debug"
	"github.com/gorilla/mux"
	"github.com/mcesarhm/geek-accounting/go-server/accounting"
	"github.com/mcesarhm/geek-accounting/go-server/core"
	"strings"

	"appengine"
	"appengine/datastore"
)

const PathPrefix = "/charts-of-accounts"

func init() {
	r := mux.NewRouter()
	r.HandleFunc(PathPrefix, getAllHandler(accounting.AllChartsOfAccounts)).Methods("GET")
	r.HandleFunc(PathPrefix, postHandler(accounting.SaveChartOfAccounts)).Methods("POST")
	r.HandleFunc(PathPrefix+"/{coa}/accounts", getAllHandler(accounting.AllAccounts)).Methods("GET")
	r.HandleFunc(PathPrefix+"/{coa}/accounts/{account}", getAllHandler(accounting.GetAccount)).Methods("GET")
	r.HandleFunc(PathPrefix+"/{coa}/accounts", postHandler(accounting.SaveAccount)).Methods("POST")
	r.HandleFunc(PathPrefix+"/{coa}/accounts/{account}", postHandler(accounting.SaveAccount)).Methods("PUT")
	r.HandleFunc(PathPrefix+"/{coa}/accounts/{account}", deleteHandler(accounting.DeleteAccount)).Methods("DELETE")
	r.HandleFunc(PathPrefix+"/{coa}/transactions", getAllHandler(accounting.AllTransactions)).Methods("GET")
	r.HandleFunc(PathPrefix+"/{coa}/transactions", postHandler(accounting.SaveTransaction)).Methods("POST")
	r.HandleFunc(PathPrefix+"/{coa}/transactions/{transaction}", postHandler(accounting.SaveTransaction)).Methods("PUT")
	r.HandleFunc(PathPrefix+"/{coa}/transactions/{transaction}", getAllHandler(accounting.GetTransaction)).Methods("GET")
	r.HandleFunc(PathPrefix+"/{coa}/transactions/{transaction}", deleteHandler(accounting.DeleteTransaction)).Methods("DELETE")
	r.HandleFunc(PathPrefix+"/{coa}/balance-sheet", getAllHandler(accounting.Balance)).Methods("GET")
	r.HandleFunc(PathPrefix+"/{coa}/journal", getAllHandler(accounting.Journal)).Methods("GET")
	r.HandleFunc(PathPrefix+"/{coa}/accounts/{account}/ledger", getAllHandler(accounting.Ledger)).Methods("GET")
	r.HandleFunc(PathPrefix+"/{coa}/income-statement", getAllHandler(accounting.IncomeStatement)).Methods("GET")
	r.HandleFunc("/_ah/startup", func(w http.ResponseWriter, r *http.Request) {
		if err := core.InitUserManagement(appengine.NewContext(r)); err != nil {
			http.Error(w, "Internal error:"+err.Error(), http.StatusInternalServerError)
		}
		return
	})
	r.HandleFunc("/ping", errorHandler(func(w http.ResponseWriter, r *http.Request, userKey *datastore.Key) error {
		return nil
	}))
	http.Handle("/", r)
}

func getAllHandler(f func(appengine.Context, map[string]string, *datastore.Key) (interface{}, error)) http.HandlerFunc {
	return errorHandler(func(w http.ResponseWriter, r *http.Request, userKey *datastore.Key) error {
		params := mux.Vars(r)
		for k, v := range r.URL.Query() {
			params[k] = v[0]
		}
		items, err := f(appengine.NewContext(r), params, userKey)
		if err != nil {
			return err
		}
		return json.NewEncoder(w).Encode(items)
	})
}

func postHandler(f func(appengine.Context, map[string]interface{}, map[string]string, *datastore.Key) (interface{}, error)) http.HandlerFunc {
	return errorHandler(func(w http.ResponseWriter, r *http.Request, userKey *datastore.Key) error {
		/*
			b, _ := ioutil.ReadAll(r.Body)
			log.Println(string(b))
		*/
		var req interface{}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return badRequest{err}
		}

		m := req.(map[string]interface{})
		item, err := f(appengine.NewContext(r), m, mux.Vars(r), userKey)
		if err != nil {
			return badRequest{err}
		}

		json.NewEncoder(w).Encode(item)

		return nil
	})
}

func deleteHandler(f func(appengine.Context, map[string]interface{}, map[string]string, *datastore.Key) (interface{}, error)) http.HandlerFunc {
	return errorHandler(func(w http.ResponseWriter, r *http.Request, userKey *datastore.Key) error {

		_, err := f(appengine.NewContext(r), nil, mux.Vars(r), userKey)
		if err != nil {
			return badRequest{err}
		}

		return nil
	})
}

type badRequest struct{ error }

type notFound struct{ error }

// errorHandler wraps a function returning an error by handling the error and returning a http.Handler.
// If the error is of the one of the types defined above, it is handled as described for every type.
// If the error is of another type, it is considered as an internal error and its message is logged.
func errorHandler(f func(w http.ResponseWriter, r *http.Request, userKey *datastore.Key) error) http.HandlerFunc {
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
			http.Error(w, "Internal error:"+err.Error(), http.StatusInternalServerError)
			return
		}
		arr := strings.Split(string(b), ":")
		c := appengine.NewContext(r)
		err, ok, userKey := core.Login(c, arr[0], arr[1])
		if err != nil {
			http.Error(w, "Internal error:"+err.Error(), http.StatusInternalServerError)
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
			http.Error(w, "Internal error:"+err.Error(), http.StatusInternalServerError)
		}
	}
}
