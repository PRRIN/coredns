package tree

import "github.com/miekg/dns"

// Elem is an element in the tree.
type Elem struct {
	M     map[uint16][]dns.RR `json:"m"`
	Nname string              `json:"name"` // owner name
}

// newElem returns a new elem.
func newElem(rr dns.RR) *Elem {
	e := Elem{M: make(map[uint16][]dns.RR)}
	e.M[rr.Header().Rrtype] = []dns.RR{rr}
	return &e
}

// Types returns the types of the records in e. The returned list is not sorted.
func (e *Elem) Types() []uint16 {
	t := make([]uint16, len(e.M))
	i := 0
	for ty := range e.M {
		t[i] = ty
		i++
	}
	return t
}

// Type returns the RRs with type qtype from e.
func (e *Elem) Type(qtype uint16) []dns.RR { return e.M[qtype] }

// TypeForWildcard returns the RRs with type qtype from e. The ownername returned is set to qname.
func (e *Elem) TypeForWildcard(qtype uint16, qname string) []dns.RR {
	rrs := e.M[qtype]

	if rrs == nil {
		return nil
	}

	copied := make([]dns.RR, len(rrs))
	for i := range rrs {
		copied[i] = dns.Copy(rrs[i])
		copied[i].Header().Name = qname
	}
	return copied
}

// All returns all RRs from e, regardless of type.
func (e *Elem) All() []dns.RR {
	list := []dns.RR{}
	for _, rrs := range e.M {
		list = append(list, rrs...)
	}
	return list
}

// Name returns the name for this node.
func (e *Elem) Name() string {
	if e.Nname != "" {
		return e.Nname
	}
	for _, rrs := range e.M {
		e.Nname = rrs[0].Header().Name
		return e.Nname
	}
	return ""
}

// Empty returns true is e does not contain any RRs, i.e. is an empty-non-terminal.
func (e *Elem) Empty() bool { return len(e.M) == 0 }

// Insert inserts rr into e. If rr is equal to existing RRs, the RR will be added anyway.
func (e *Elem) Insert(rr dns.RR) {
	t := rr.Header().Rrtype
	if e.M == nil {
		e.M = make(map[uint16][]dns.RR)
		e.M[t] = []dns.RR{rr}
		return
	}
	rrs, ok := e.M[t]
	if !ok {
		e.M[t] = []dns.RR{rr}
		return
	}

	rrs = append(rrs, rr)
	e.M[t] = rrs
}

// Delete removes all RRs of type rr.Header().Rrtype from e.
func (e *Elem) Delete(rr dns.RR) {
	if e.M == nil {
		return
	}

	t := rr.Header().Rrtype
	delete(e.M, t)
}

// Less is a tree helper function that calls less.
func Less(a *Elem, name string) int { return less(name, a.Name()) }
