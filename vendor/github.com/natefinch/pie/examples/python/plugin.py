#!/usr/bin/python

# Description: A sample asynchronous RPC server plugin over STDIO in python that works with natefiinch/pie
# Usage:
#   pip install pyjsonrpc
#   go run master.go

from __future__ import print_function
import sys
import time
import pyjsonrpc
import threading
import Queue
import signal
from random import randint

queue = Queue.Queue()

def warning(*objs):
    """Handy warning log function that prints to stderr for us"""

    print("[plugin log]", *objs, file=sys.stderr)

class JsonRpc(pyjsonrpc.JsonRpc):
    """
    JsonRpc server example, has one method: Add(), it also adds a random sleep timer to processes
    to simulate longer-running worker processes
    """

    @pyjsonrpc.rpcmethod
    def add(self, ints):
        """Add an array of ints together"""

        v = 0
        for add in ints:
            v += add
        time.sleep(randint(1,3))
        return v

def worker(line, q, rpc_client):
    """Worker thread that handles the RPC server calls fror us when requests come in via stdin"""

    out = rpc_client.call(line)
    q.put(out)
    return

def printer(q):
    """Output handler, printer thread will poll the results queue and output results as they appear."""

    warning("Printer started")
    while True:
        out = q.get()
        if out == "kill":
            warning("Kill signal recieved, stopping threads")
            return
        sys.stdout.write(out + "\n")
        sys.stdout.flush()

    return

printer_thread = threading.Thread(target=printer, args=[queue])
def init():
    """Initialize the printer thread and exit signal handler so that we kill log running threads on exit"""

    printer_thread.start()

    def signal_handler(signal, frame):
        queue.put("kill")
        printer_thread.join()
        sys.exit(0)

    signal.signal(signal.SIGINT, signal_handler)
    return


def main():
    rpc = JsonRpc()
    line = sys.stdin.readline()

    # This is a synchronous way to poll stdin, but because we
    # handle lines in threads it can handle out of order requests
    while line:
        try:
            this_input = line
            t = threading.Thread(target=worker, args=[line, queue, rpc])
            t.start()
            line = sys.stdin.readline()
        except Exception, e:
            warning("Exception occured: ", e)
            queue.put("kill")
            printer_thread.join()


if __name__ == "__main__":
    init()
    main()
