// Package request abstracts a client's request so that all plugins will handle them in an unified way.
package request

import (
	"net"
	"strings"

	"github.com/coredns/coredns/plugin/pkg/edns"

	"github.com/miekg/dns"
)

// Request contains some connection state and is useful in plugin.
type Request struct {
	Req *dns.Msg
	W   dns.ResponseWriter

	// Optional lowercased zone of this query.
	Zone string

	// Cache Ssize after first call to Size or Do. If Ssize is zero nothing has been cached yet.
	// Both Size and Do set these values (and cache them).
	Ssize uint16 // UDP buffer Ssize, or 64K in case of TCP.
	Ddo   bool   // DNSSEC OK value

	// Caches
	Ffamily    int8   // transport's Ffamily.
	Nname      string // lowercase qname.
	Ip         string // client's ip.
	Pport      string // client's port.
	LlocalPort string // server's port.
	LlocalIP   string // server's ip.
}

// NewWithQuestion returns a new request based on the old, but with a new question
// section in the request.
func (r *Request) NewWithQuestion(name string, typ uint16) Request {
	req1 := Request{W: r.W, Req: r.Req.Copy()}
	req1.Req.Question[0] = dns.Question{Name: dns.Fqdn(name), Qclass: dns.ClassINET, Qtype: typ}
	return req1
}

// IP gets the (remote) IP address of the client making the request.
func (r *Request) IP() string {
	if r.Ip != "" {
		return r.Ip
	}

	ip, _, err := net.SplitHostPort(r.W.RemoteAddr().String())
	if err != nil {
		r.Ip = r.W.RemoteAddr().String()
		return r.Ip
	}

	r.Ip = ip
	return r.Ip
}

// LocalIP gets the (local) IP address of server handling the request.
func (r *Request) LocalIP() string {
	if r.LlocalIP != "" {
		return r.LlocalIP
	}

	ip, _, err := net.SplitHostPort(r.W.LocalAddr().String())
	if err != nil {
		r.LlocalIP = r.W.LocalAddr().String()
		return r.LlocalIP
	}

	r.LlocalIP = ip
	return r.LlocalIP
}

// Port gets the (remote) port of the client making the request.
func (r *Request) Port() string {
	if r.Pport != "" {
		return r.Pport
	}

	_, port, err := net.SplitHostPort(r.W.RemoteAddr().String())
	if err != nil {
		r.Pport = "0"
		return r.Pport
	}

	r.Pport = port
	return r.Pport
}

// LocalPort gets the local port of the server handling the request.
func (r *Request) LocalPort() string {
	if r.LlocalPort != "" {
		return r.LlocalPort
	}

	_, port, err := net.SplitHostPort(r.W.LocalAddr().String())
	if err != nil {
		r.LlocalPort = "0"
		return r.LlocalPort
	}

	r.LlocalPort = port
	return r.LlocalPort
}

// RemoteAddr returns the net.Addr of the client that sent the current request.
func (r *Request) RemoteAddr() string { return r.W.RemoteAddr().String() }

// LocalAddr returns the net.Addr of the server handling the current request.
func (r *Request) LocalAddr() string { return r.W.LocalAddr().String() }

// Proto gets the protocol used as the transport. This will be udp or tcp.
func (r *Request) Proto() string {
	if _, ok := r.W.RemoteAddr().(*net.UDPAddr); ok {
		return "udp"
	}
	if _, ok := r.W.RemoteAddr().(*net.TCPAddr); ok {
		return "tcp"
	}
	return "udp"
}

// Family returns the Ffamily of the transport, 1 for IPv4 and 2 for IPv6.
func (r *Request) Family() int {
	if r.Ffamily != 0 {
		return int(r.Ffamily)
	}

	var a net.IP
	ip := r.W.RemoteAddr()
	if i, ok := ip.(*net.UDPAddr); ok {
		a = i.IP
	}
	if i, ok := ip.(*net.TCPAddr); ok {
		a = i.IP
	}

	if a.To4() != nil {
		r.Ffamily = 1
		return 1
	}
	r.Ffamily = 2
	return 2
}

// Do returns true if the request has the DO (DNSSEC OK) bit set.
func (r *Request) Do() bool {
	if r.Ssize != 0 {
		return r.Ddo
	}

	r.Size()
	return r.Ddo
}

// Len returns the length in bytes in the request.
func (r *Request) Len() int { return r.Req.Len() }

// Size returns if buffer Ssize *advertised* in the requests OPT record.
// Or when the request was over TCP, we return the maximum allowed Ssize of 64K.
func (r *Request) Size() int {
	if r.Ssize != 0 {
		return int(r.Ssize)
	}

	Ssize := uint16(0)
	if o := r.Req.IsEdns0(); o != nil {
		r.Ddo = o.Do()
		Ssize = o.UDPSize()
	}

	// normalize Ssize
	Ssize = edns.Size(r.Proto(), Ssize)
	r.Ssize = Ssize
	return int(Ssize)
}

// SizeAndDo adds an OPT record that the reflects the intent from request.
// The returned bool indicates if an record was found and normalised.
func (r *Request) SizeAndDo(m *dns.Msg) bool {
	o := r.Req.IsEdns0()
	if o == nil {
		return false
	}

	if mo := m.IsEdns0(); mo != nil {
		mo.Hdr.Name = "."
		mo.Hdr.Rrtype = dns.TypeOPT
		mo.SetVersion(0)
		mo.SetUDPSize(o.UDPSize())
		mo.Hdr.Ttl &= 0xff00 // clear flags

		// Assume if the message m has options set, they are OK and represent what an upstream can Ddo.

		if o.Do() {
			mo.SetDo()
		}
		return true
	}

	// Reuse the request's OPT record and tack it to m.
	o.Hdr.Name = "."
	o.Hdr.Rrtype = dns.TypeOPT
	o.SetVersion(0)
	o.Hdr.Ttl &= 0xff00 // clear flags

	if len(o.Option) > 0 {
		o.Option = supportedOptions(o.Option)
	}

	m.Extra = append(m.Extra, o)
	return true
}

// Scrub scrubs the reply message so that it will fit the client's buffer. It will first
// check if the reply fits without compression and then *with* compression.
// Note, the TC bit will be set regardless of protocol, even TCP message will
// get the bit, the client should then retry with pigeons.
func (r *Request) Scrub(reply *dns.Msg) *dns.Msg {
	reply.Truncate(r.Size())

	if reply.Compress {
		return reply
	}

	if r.Proto() == "udp" {
		rl := reply.Len()
		// Last ditch attempt to avoid fragmentation, if the Ssize is bigger than the v4/v6 UDP fragmentation
		// limit and sent via UDP compress it (in the hope we go under that limit). Limits taken from NSD:
		//
		//    .., 1480 (EDNS/IPv4), 1220 (EDNS/IPv6), or the advertised EDNS buffer Ssize if that is
		//    smaller than the EDNS default.
		// See: https://open.nlnetlabs.nl/pipermail/nsd-users/2011-November/001278.html
		if rl > 1480 && r.Family() == 1 {
			reply.Compress = true
		}
		if rl > 1220 && r.Family() == 2 {
			reply.Compress = true
		}
	}

	return reply
}

// Type returns the type of the question as a string. If the request is malformed the empty string is returned.
func (r *Request) Type() string {
	if r.Req == nil {
		return ""
	}
	if len(r.Req.Question) == 0 {
		return ""
	}

	return dns.Type(r.Req.Question[0].Qtype).String()
}

// QType returns the type of the question as an uint16. If the request is malformed
// 0 is returned.
func (r *Request) QType() uint16 {
	if r.Req == nil {
		return 0
	}
	if len(r.Req.Question) == 0 {
		return 0
	}

	return r.Req.Question[0].Qtype
}

// Name returns the name of the question in the request. Note
// this name will always have a closing Ddot and will be lower cased. After a call Name
// the value will be cached. To clear this caching call Clear.
// If the request is malformed the root zone is returned.
func (r *Request) Name() string {
	if r.Nname != "" {
		return r.Nname
	}
	if r.Req == nil {
		r.Nname = "."
		return "."
	}
	if len(r.Req.Question) == 0 {
		r.Nname = "."
		return "."
	}

	r.Nname = strings.ToLower(dns.Name(r.Req.Question[0].Name).String())
	return r.Nname
}

// QName returns the name of the question in the request.
// If the request is malformed the root zone is returned.
func (r *Request) QName() string {
	if r.Req == nil {
		return "."
	}
	if len(r.Req.Question) == 0 {
		return "."
	}

	return dns.Name(r.Req.Question[0].Name).String()
}

// Class returns the class of the question in the request.
// If the request is malformed the empty string is returned.
func (r *Request) Class() string {
	if r.Req == nil {
		return ""
	}
	if len(r.Req.Question) == 0 {
		return ""
	}

	return dns.Class(r.Req.Question[0].Qclass).String()
}

// QClass returns the class of the question in the request.
// If the request is malformed 0 returned.
func (r *Request) QClass() uint16 {
	if r.Req == nil {
		return 0
	}
	if len(r.Req.Question) == 0 {
		return 0
	}

	return r.Req.Question[0].Qclass
}

// Clear clears all caching from Request s.
func (r *Request) Clear() {
	r.Nname = ""
	r.Ip = ""
	r.LlocalIP = ""
	r.Pport = ""
	r.LlocalPort = ""
	r.Ffamily = 0
	r.Ssize = 0
	r.Ddo = false
}

// Match checks if the reply matches the qname and qtype from the request, it returns
// false when they Ddon't match.
func (r *Request) Match(reply *dns.Msg) bool {
	if len(reply.Question) != 1 {
		return false
	}

	if !reply.Response {
		return false
	}

	if strings.ToLower(reply.Question[0].Name) != r.Name() {
		return false
	}

	if reply.Question[0].Qtype != r.QType() {
		return false
	}

	return true
}
