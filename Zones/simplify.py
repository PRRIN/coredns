# Simplify domain names by replacing labels with one-character identifiers

import os
import re
import sys

for root, dirs, files in os.walk(sys.argv[1]):
    for file in files:
        file = os.path.join(root, file)

        processed = []
        # cur = ord('a')
        atoz = [chr(c) for c in range(ord('a'), ord('z')+1)]
        vocab = atoz + [c1+c2 for c1 in atoz for c2 in atoz]
        cur = 0

        mapping = {'*': '*'}
        with open(file, "r") as f:
            for line in f:
                parts = re.split(r'(\s|\t)', line)
                for part in parts:
                    if part.endswith('.'):
                        # is a domain name
                        name = ""
                        for label in part[:-1].split('.'):
                            if label not in mapping:
                                mapping[label] = vocab[cur]
                                cur += 1
                                if cur >= len(vocab):
                                    print("too many labels")
                                    exit(1)
                            name += mapping[label] + "."
                        processed.append(name)
                    else:
                        processed.append(part)

        with open(file[:-4] + "-simplified.txt", "w") as f:
            f.write("".join(processed))