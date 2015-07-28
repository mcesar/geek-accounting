// +build appengine

package deb

import (
	"fmt"

	"appengine"
	"appengine/datastore"
)

type datastoreSpace struct{}

func NewDatastoreSpace(ctx appengine.Context, key *datastore.Key) (Space, error) {
	if ctx == nil {
		fmt.Errorf("ctx is nil")
	}
	var ds *largeSpace
	if key == nil {
		key = datastore.NewIncompleteKey(ctx, "space", nil)
		var err error
		if key, err = datastore.Put(ctx, key, &datastoreSpace{}); err != nil {
			return nil, err
		}
	}
	errc := make(chan error, 1)
	in := func() chan *dataBlock {
		c := make(chan *dataBlock)
		go func() {
			var err error
			defer close(c)
			defer func() { errc <- err }()
			q := datastore.NewQuery("data_block").Ancestor(key)
			t := q.Run(ctx)
			for {
				var b = ds.newDataBlock()
				b.key, err = t.Next(b)
				if err == datastore.Done {
					err = nil
					break
				}
				if err != nil {
					break
				}
				c <- b
			}
		}()
		return c
	}
	out := make(chan *dataBlock)
	go func() {
		for block := range out {
			if block.key == nil {
				block.key = datastore.NewIncompleteKey(ctx, "data_block", key)
			}
			_, err := datastore.Put(ctx, block.key.(*datastore.Key), block)
			errc <- err
		}
	}()
	ds = NewLargeSpace(1014*1024, in, out, errc)
	return ds, nil
}
