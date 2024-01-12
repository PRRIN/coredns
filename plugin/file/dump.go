package file

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/coredns/coredns/plugin/file/tree"
	"github.com/coredns/coredns/plugin/pkg/upstream"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
)

type _Zone struct {
	Origin  string
	OrigLen int
	File    string
	Tree    *tree.Tree // already all public
	Apex    Apex
	Expired bool

	Lock struct{}

	StartupOnce  struct{}
	TransferFrom []string

	ReloadInterval time.Duration
	ReloadShutdown struct{}

	Upstream *upstream.Upstream
}

func dump(val reflect.Value, json *map[string][]string, cur string) {

	switch val.Kind() {
	case reflect.Bool:
		if val.Bool() {
			(*json)[cur] = append((*json)[cur], "{ \"value\": 1, \"__type__\": \"bool\" }")
		} else {
			(*json)[cur] = append((*json)[cur], "{ \"value\": 0, \"__type__\": \"bool\" }")
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		(*json)[cur] = append((*json)[cur], fmt.Sprint(val.Int()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		(*json)[cur] = append((*json)[cur], fmt.Sprint(val.Uint()))
	case reflect.Array:
		(*json)[cur] = append((*json)[cur], "[")
		sep := false
		for i := 0; i < val.Len(); i++ {
			if sep {
				(*json)[cur] = append((*json)[cur], ", ")
			}
			dump(val.Index(i), json, cur)
			sep = true
		}
		(*json)[cur] = append((*json)[cur], "]")
	case reflect.Map:
		(*json)[cur] = append((*json)[cur], "{")
		sep := false
		iter := val.MapRange()
		for iter.Next() {
			if sep {
				(*json)[cur] = append((*json)[cur], ", ")
			}

			key := iter.Key()
			var keyStr string
			if key.CanInt() {
				keyStr = fmt.Sprintf("\"%d\"", key.Int())
			} else if key.CanUint() {
				keyStr = fmt.Sprintf("\"%d\"", key.Uint())
			} else {
				keyStr = fmt.Sprintf("\"%s\"", key.String())
			}
			(*json)[cur] = append((*json)[cur], fmt.Sprintf("%s: ", keyStr))
			dump(iter.Value(), json, cur)
			sep = true
		}
		if sep {
			(*json)[cur] = append((*json)[cur], ", ")
		}
		(*json)[cur] = append((*json)[cur], fmt.Sprintf("\"__type__\": \"%s\"}", val.Type()))

	case reflect.Pointer:
		var addr string
		if val.CanAddr() {
			if val.UnsafeAddr() == 0 {
				(*json)[cur] = append((*json)[cur], "\"zeroVal\"")
				return
			}
			addr = fmt.Sprintf("\"%x\"", val.UnsafeAddr())
		} else {
			addr = fmt.Sprintf("\"Ptr(%x)\"", val.Elem().UnsafeAddr())
		}
		(*json)[cur] = append((*json)[cur], addr)
		if _, ok := (*json)[addr]; ok {
			return
		}
		(*json)[addr] = make([]string, 0)
		dump(val.Elem(), json, addr)
	case reflect.Interface:
		if val.InterfaceData()[1] == 0 {
			(*json)[cur] = append((*json)[cur], "\"zeroVal\"")
			return
		}
		typ := val.Type()
		addr := fmt.Sprintf("\"Iface(%x)\"", val.InterfaceData()[1])
		for ; val.Elem().Kind() == reflect.Pointer; val = val.Elem() {
			// fmt.Printf("-> %s(%s)", val.Elem(), val.Elem().Kind())
		}
		itab := fmt.Sprintf("\"Itab(%s)\"", val.Type())
		(*json)[cur] = append((*json)[cur], fmt.Sprintf("{\"itab\": %s, \"data\": %s, \"__type__\": \"%s\"}", itab, addr, typ))

		if _, ok := (*json)[addr]; ok {
			return
		}
		(*json)[addr] = make([]string, 0)
		dump(val.Elem(), json, addr)
	case reflect.Slice:
		if val.Pointer() == 0 {
			(*json)[cur] = append((*json)[cur], "\"zeroVal\"")
			return
		}

		addr := fmt.Sprintf("\"Slice(%x)\"", val.Pointer())
		(*json)[cur] = append((*json)[cur], fmt.Sprintf("{\"ptr\": %s", addr))
		(*json)[cur] = append((*json)[cur], fmt.Sprintf(", \"len\": %d", val.Len()))
		(*json)[cur] = append((*json)[cur], fmt.Sprintf(", \"cap\": %d, \"__type__\": \"%s\"}", val.Cap(), val.Type()))

		if _, ok := (*json)[addr]; ok {
			return
		}

		(*json)[addr] = make([]string, 0)
		(*json)[addr] = append((*json)[addr], "[")
		sep := false
		for i := 0; i < val.Len(); i++ {
			if sep {
				(*json)[addr] = append((*json)[addr], ", ")
			}
			dump(val.Index(i), json, addr)
			sep = true
		}
		(*json)[addr] = append((*json)[addr], "]")

	case reflect.String:
		(*json)[cur] = append((*json)[cur], fmt.Sprintf("{\"value\": \"%s\", \"__type__\": \"string\"}", val.String()))
	case reflect.Struct:
		(*json)[cur] = append((*json)[cur], "{")
		sep := false
		for i := 0; i < val.NumField(); i++ {
			if sep {
				(*json)[cur] = append((*json)[cur], ", ")
			}
			field := val.Type().Field(i)
			(*json)[cur] = append((*json)[cur], fmt.Sprintf("\"%s\": ", field.Name))
			dump(val.Field(i), json, cur)
			sep = true
		}
		if sep {
			(*json)[cur] = append((*json)[cur], ", ")
		}
		(*json)[cur] = append((*json)[cur], fmt.Sprintf("\"__type__\": \"%s\"}", val.Type()))
	default:
		if val.IsValid() {
			(*json)[cur] = append((*json)[cur], fmt.Sprintf("{\"__type__\": \"%s\"}", val.Type()))
		} else {
			(*json)[cur] = append((*json)[cur], "\"zeroVal\"")
		}
	}

}

func writeJsonFile(name string, json map[string][]string) {
	file, err := os.Create(name)
	if err != nil {
		fmt.Println("file create error:", err)
	}
	defer file.Close()

	file.WriteString("{")
	sep := false
	for k, v := range json {
		if sep {
			file.WriteString(",\n")
		} else {
			file.WriteString("\n")
		}
		file.WriteString(fmt.Sprintf("%s: %s", k, strings.Join(v, "")))
		sep = true
	}
	file.WriteString("\n}")
}

func dumpZone(z *Zone) {

	// Dump the zone
	_z := _Zone{
		Origin:         z.origin,
		OrigLen:        z.origLen,
		File:           z.file,
		Tree:           z.Tree,
		Apex:           z.Apex,
		Expired:        z.Expired,
		Lock:           struct{}{},
		StartupOnce:    struct{}{},
		TransferFrom:   z.TransferFrom,
		ReloadInterval: z.ReloadInterval,
		ReloadShutdown: struct{}{},
		Upstream:       z.Upstream,
	}

	fmt.Println("dumping zone...")

	json := make(map[string][]string)
	json["\"zone\""] = make([]string, 0)
	dump(reflect.ValueOf(_z), &json, "\"zone\"")

	writeJsonFile("zone.json", json)
}

type _Request struct {
	Req *dns.Msg
	W   dns.ResponseWriter

	// Optional lowercased zone of this query.
	Zone string

	// Cache size after first call to Size or Do. If size is zero nothing has been cached yet.
	// Both Size and Do set these values (and cache them).
	Size uint16 // UDP buffer size, or 64K in case of TCP.
	Do   bool   // DNSSEC OK value

	// Caches
	Family    int8   // transport's family.
	Name      string // lowercase qname.
	Ip        string // client's ip.
	Port      string // client's port.
	LocalPort string // server's port.
	LocalIP   string // server's ip.
}

func dumpReq(r *request.Request) {

	fmt.Println("dumping request...")

	_r := _Request{
		Req:  r.Req,
		W:    r.W,
		Zone: r.Zone,

		Size:      uint16(r.Size()),
		Do:        r.Do(),
		Family:    int8(r.Family()),
		Name:      r.Name(),
		Ip:        r.IP(),
		Port:      r.IP(),
		LocalPort: r.LocalPort(),
		LocalIP:   r.LocalIP(),
	}

	json := make(map[string][]string)
	json["\"request\""] = make([]string, 0)
	dump(reflect.ValueOf(_r), &json, "\"request\"")

	// Write
	writeJsonFile("request.json", json)
}
