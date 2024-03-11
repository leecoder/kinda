# Example Python script that reads from stdin and writes to stdout

import sys

for line in sys.stdin:
    processed_line = line.strip().upper()  # Example processing
    print(processed_line)