import os
import re

resource_types = set()

maxlen = 0
maxfile = ""
for root, dirs, files in os.walk("ZoneFiles"):
    for file in files:
        file = os.path.join(root, file)

        with open(file, "r") as f:
            for line in f:
                dname = re.split("\s", line)[0]
                if len(dname) > maxlen:
                    maxfile = file
                    maxlen = len(dname)

print("Max domain name length:", maxlen, "in", maxfile)