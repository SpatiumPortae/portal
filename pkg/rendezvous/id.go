// id.go specifies the central datastructure used to keep track of connection ids.
package rendezvous

import (
	"sync"
)

// IDs is a threadsafe set of numbers.
type IDs struct{ *sync.Map }

type void struct{} // empty struct complies to 0 bytes
var member void

// Bind binds an id to connection.
func (ids *IDs) Bind() int {
	id := 1
	for {
		val, _ := ids.Load(id)
		if val == nil {
			break
		}
		id++
	}
	ids.Store(id, member)
	return id
}

// DeleteID Deletes a bound ID.
func (ids *IDs) DeleteID(id int) {
	ids.Delete(id)
}
