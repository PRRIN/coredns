import os

def find_empty_dirs(directory):
    for root, dirs, files in os.walk(directory):
        if not dirs and not files:
            print(root)

# Call the function
find_empty_dirs("./json/")