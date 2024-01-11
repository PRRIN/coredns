# Dumping into JSON

### Modified Section

* `plugin/file/setup.go` will dump `Zone` after parsing from the zone file.

* `plugin/file/file.go` will dump `Request` whenever it serves a request.

* `plugin/file/dump.go` contains dump methods. It uses reflection to dump every exported fields properly.

### Running

At the project root:
```bash
docker run --rm -it -v $PWD:/v -w /v golang:1.21 /bin/bash

apt update && apt install dnsutils -y       # for dig

make

./coredns &

dig @127.0.0.1 -p 1053 www.example.org
```

### Understanding the JSON File

```json
{
"zone": {"Origin": "example.org.", "OrigLen": 2, "File": "db.example.org", "Tree": "Ptr(400004d980)", "Apex": {"SOA": "Ptr(400051f200)", "NS": {"ptr": "Slice(40001c3ea0)", "len": 2, "cap": 2, "__type__": "slice"}, "SIGSOA": {"ptr": "nil", "len": 0, "cap": 0, "__type__": "slice"}, "SIGNS": {"ptr": "nil", "len": 0, "cap": 0, "__type__": "slice"}, "__type__": "struct"}, "Expired": 0, "Lock": {"__type__": "struct"}, "StartupOnce": {"__type__": "struct"}, "TransferFrom": {"ptr": "nil", "len": 0, "cap": 0, "__type__": "slice"}, "ReloadInterval": 60000000000, "ReloadShutdown": {"__type__": "struct"}, "Upstream": "Ptr(34af0a0)", "__type__": "struct"},
"Slice(400004d9a0)": [{"type": "22f8850", "value": "Iface(40000552c0)", "__type__": "iface"}],
"Iface(40000552c0)": {"Hdr": {"Name": "www.example.org.", "Rrtype": 1, "Class": 1, "Ttl": 3600, "Rdlength": 0, "__type__": "struct"}, "A": {"ptr": "Slice(400063f650)", "len": 16, "cap": 16, "__type__": "slice"}, "__type__": "struct"},
"Slice(400063f650)": [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 127, 0, 0, 1],
"40001c3ec8": {"Elem": "40001c3ee0", "Left": "40001c3ee8", "Right": "40001c3ef0", "Color": 0, "__type__": "struct"},
"40001c3ef0": {"__type__": "invalid"},
"Iface(400076a0f0)": {"Hdr": {"Name": "example.org.", "Rrtype": 2, "Class": 1, "Ttl": 3600, "Rdlength": 0, "__type__": "struct"}, "Ns": "b.iana-servers.net.", "__type__": "struct"},
"Ptr(34af0a0)": {"__type__": "struct"},
"Ptr(400004d980)": {"Root": "400004d980", "Count": 2, "__type__": "struct"},
"40001c3ec0": {"M": {"1": {"ptr": "Slice(400004d9a0)", "len": 1, "cap": 1, "__type__": "slice"}, "__type__": "map"}, "Nname": "www.example.org.", "__type__": "struct"},
"40001c3ee0": {"M": {"28": {"ptr": "Slice(400004d9b0)", "len": 1, "cap": 1, "__type__": "slice"}, "__type__": "map"}, "Nname": "", "__type__": "struct"},
"40001c3ee8": {"__type__": "invalid"},
"40001c3ed0": {"__type__": "invalid"},
"Ptr(400051f200)": {"Hdr": {"Name": "example.org.", "Rrtype": 6, "Class": 1, "Ttl": 3600, "Rdlength": 0, "__type__": "struct"}, "Ns": "sns.dns.icann.org.", "Mbox": "noc.dns.icann.org.", "Serial": 2017042745, "Refresh": 7200, "Retry": 3600, "Expire": 1209600, "Minttl": 3600, "__type__": "struct"},
"Iface(400076a090)": {"Hdr": {"Name": "example.org.", "Rrtype": 2, "Class": 1, "Ttl": 3600, "Rdlength": 0, "__type__": "struct"}, "Ns": "a.iana-servers.net.", "__type__": "struct"},
"400004d980": {"Elem": "40001c3ec0", "Left": "40001c3ec8", "Right": "40001c3ed0", "Color": 1, "__type__": "struct"},
"Iface(4000055300)": {"Hdr": {"Name": "mail.example.org.", "Rrtype": 28, "Class": 1, "Ttl": 3600, "Rdlength": 0, "__type__": "struct"}, "AAAA": {"ptr": "Slice(400063f680)", "len": 16, "cap": 16, "__type__": "slice"}, "__type__": "struct"},
"Slice(400004d9b0)": [{"type": "22f88a8", "value": "Iface(4000055300)", "__type__": "iface"}],
"Slice(400063f680)": [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1],
"Slice(40001c3ea0)": [{"type": "22f8ab8", "value": "Iface(400076a090)", "__type__": "iface"}, {"type": "22f8ab8", "value": "Iface(400076a0f0)", "__type__": "iface"}]
}
```

* Every JSON object (wrapped in `{}`) will have a last field of `__type__`, discriminating its type.

* The primary keys represent memory positions. Pointers are linked via these keys.

* `{"__type__": "invalid"}` and  `{"__type__": "struct"}` (empty structs) can be treated as `Exp::Havoc`.