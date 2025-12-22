# Dumping into JSON

### Modified Section

* `plugin/file/file.go` will dump `File` and `Request` whenever it serves a request.

* `plugin/file/dump.go` contains dump methods. It uses reflection to dump every exported fields properly.

### Dumping

At the project root (`coredns/`):
```bash
docker run --rm -it -v $PWD:/v -w /v golang:1.21 /bin/bash

# In docker
apt update && apt install dnsutils -y       # for `dig`
make

# In docker
./zone.sh Buggy    # For the Buggy category (same goes for Simple/Complex/Real)
# Outside docker 
mv Zones/json/* ../iceberg/test/coredns/json/Buggy/   
```

### Understanding the JSON File

* Every JSON object (wrapped in `{}`) will have a last field of `__type__`, discriminating its type.

* The primary keys represent memory positions. Pointers are linked via these keys.

* `{"__type__": "invalid"}` and  `{"__type__": "struct"}` (empty structs) can be treated as `Exp::Havoc`.