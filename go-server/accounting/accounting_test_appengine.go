// +build appengine,test

package accounting

import (
	"encoding/gob"
)

func init() {
	gob.Register(([]*Account)(nil))
	gob.Register((*Account)(nil))
	gob.Register(([]*Transaction)(nil))
	gob.Register(([]interface{})(nil))
}
