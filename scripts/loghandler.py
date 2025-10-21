from pwn import remote, PwnlibException
from time import sleep
import sys

if __name__ == "__main__":
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 4567
    print(f"[*] Connecting to localhost:{port} for logs...")

    while True:
        try:
            r = remote("localhost", port)
            while True:
                print(r.recvline().decode())
        except (EOFError, TimeoutError, PwnlibException):
            sleep(0.2)
