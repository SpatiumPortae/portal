package rendezvous

import (
	"sync"
)

// IDs is a threadsafe set of numbers.
type IDs struct{ *sync.Map }

type void struct{} // empty struct complies to 0 bytes
var member void

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

func (ids *IDs) DeleteID(id int) {
	ids.Delete(id)
}
