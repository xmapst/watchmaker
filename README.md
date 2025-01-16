# watchmaker

injector which change the time inside the process instead of the host.

## Usage
```shell
# absolute dates is "YYYY-MM-DD hh:mm:ss"
./bin/watchmaker --pid 1536 --faketime "2003-01-01 10:00:05"

# absolute dates is timestamp (seconds or nanosecond)
./bin/watchmaker --pid 1536 --faketime 1705022260

# time offset(no units), default is seconds 
./bin/watchmaker --pid 1536 --faketime +120 
# or
./bin/watchmaker --pid 1536 --faketime -120

# time offset(s)
./bin/watchmaker --pid 1536 --faketime +120s
# or
./bin/watchmaker --pid 1536 --faketime -120s

# time offset(m)
./bin/watchmaker --pid 1536 --faketime +12m
# or
./bin/watchmaker --pid 1536 --faketime -12m

# time offset(h)
./bin/watchmaker --pid 1536 --faketime +12h
# or
./bin/watchmaker --pid 1536 --faketime -12h

# time offset(d)
./bin/watchmaker --pid 1536 --faketime +12d
# or
./bin/watchmaker --pid 1536 --faketime -12d

# time offset(y)
./bin/watchmaker --pid 1536 --faketime +1y
# or
./bin/watchmaker --pid 1536 --faketime -1y
```

## Reference

This project uses the following open-source software:

* [Chaos-mesh](https://github.com/chaos-mesh/chaos-mesh) - Reference chaos-mesh's watchmaker component to simulate process time
* [Libfaketime](https://github.com/wolfcw/libfaketime) - Reference the libfaketime dynamic link library to simulate time