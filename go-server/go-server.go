package server

import (
	"encoding/base64"
	"encoding/json"
	//"io/ioutil"
	"log"
    "net/http"
    "strings"
    "github.com/gorilla/mux"
    "github.com/mcesarhm/geek-accounting/go-server/domain"

    "appengine"
    "appengine/datastore"
)

const PathPrefix = "/charts-of-accounts"

func init() {
	r := mux.NewRouter()
    r.HandleFunc(PathPrefix, getAllHandler(domain.AllChartsOfAccounts)).Methods("GET")
    r.HandleFunc(PathPrefix, postHandler(domain.SaveChartOfAccounts)).Methods("POST")
    r.HandleFunc(PathPrefix+"/{coa}/accounts", getAllHandler(domain.AllAccounts)).Methods("GET")
    r.HandleFunc(PathPrefix+"/{coa}/accounts", postHandler(domain.SaveAccount)).Methods("POST")
    r.HandleFunc(PathPrefix+"/{coa}/transactions", getAllHandler(domain.AllTransactions)).Methods("GET")
    r.HandleFunc(PathPrefix+"/{coa}/transactions", postHandler(domain.SaveTransaction)).Methods("POST")
    r.HandleFunc("/_ah/warmup", func(w http.ResponseWriter, r *http.Request) {
    	if err := domain.InitUserManagement(appengine.NewContext(r)); err != nil {
    		http.Error(w, "Internal error:" + err.Error(), http.StatusInternalServerError)
    	}
    	return
    })
    http.Handle("/", r)
}

func getAllHandler(f func(appengine.Context, map[string]string, *datastore.Key) (interface{}, error)) http.HandlerFunc {
	return errorHandler(func(w http.ResponseWriter, r *http.Request, userKey *datastore.Key) error {
		items, err := f(appengine.NewContext(r), mux.Vars(r), userKey)
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


type badRequest struct{ error }

type notFound struct{ error }

// errorHandler wraps a function returning an error by handling the error and returning a http.Handler.
// If the error is of the one of the types defined above, it is handled as described for every type.
// If the error is of another type, it is considered as an internal error and its message is logged.
func errorHandler(f func(w http.ResponseWriter, r *http.Request, userKey *datastore.Key) error) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
		credentials := strings.Split(r.Header["Authorization"][0], " ")[1]
		b, err := base64.StdEncoding.DecodeString(credentials)
		if err != nil {
			http.Error(w, "Internal error:" + err.Error(), http.StatusInternalServerError)
			return
		}
		arr := strings.Split(string(b), ":")
		err, ok, userKey := domain.Login(appengine.NewContext(r), arr[0], arr[1])
        if err != nil {
			http.Error(w, "Internal error:" + err.Error(), http.StatusInternalServerError)
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
            http.Error(w, "Error: " + err.Error(), http.StatusBadRequest)
        case notFound:
            http.Error(w, "Error: item not found", http.StatusNotFound)
        default:
            log.Println(err)
            http.Error(w, "Internal error:" + err.Error(), http.StatusInternalServerError)
        }
    }
}
