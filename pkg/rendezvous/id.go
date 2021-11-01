package rendezvous

import "sync"

// IDs is a threadsafe set of numbers.
type IDs struct{ *sync.Map }

type void struct{} // empty struct complies to 0 bytes
var member void

func (ids *IDs) Bind() int {
	id := 1
	for {
		_, ok := ids.Load(id)
		if !ok {
			break
		}
		id++
	}
	ids.Store(id, member)
	return id
}

func (ids *IDs) Delete(id int) bool {
	return ids.Delete(id)
}
