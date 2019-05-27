#!/bin/env python

import time
import sys
while True:
    blih=sys.stdin.read(1<<24);
    if blih == "":
        sys.exit(0)
    time.sleep(2);
