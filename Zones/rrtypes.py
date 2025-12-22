import os
import re

resource_types = set()

for root, dirs, files in os.walk("ZoneFiles"):
    for file in files:
        file = os.path.join(root, file)

        with open(file, "r") as f:
            for line in f:
                parts = re.split(r'\s+', line)
                if "IN" in parts:
                    index = parts.index("IN")
                    if index + 1 < len(parts):
                        resource_types.add(parts[index + 1])

print("Distinct DNS resource types:", resource_types)