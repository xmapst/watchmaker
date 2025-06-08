# watchmaker

injector which change the time inside the process instead of the host.

## Install
```shell
wget https://github.com/busybox-org/watchmaker/releases/download/v0.0.2/watchmaker_linux_amd64 -O /bin/watchmaker
chmod +x /bin/watchmaker
```

## Usage
```shell
# absolute dates is "YYYY-MM-DD hh:mm:ss"
watchmaker --pid 1536 --faketime "2003-01-01 10:00:05"

# absolute dates is timestamp (seconds or nanosecond)
watchmaker --pid 1536 --faketime 1705022260

# time offset(no units), default is seconds 
watchmaker --pid 1536 --faketime +120 
# or
watchmaker --pid 1536 --faketime -120

# time offset(s)
watchmaker --pid 1536 --faketime +120s
# or
watchmaker --pid 1536 --faketime -120s

# time offset(m)
watchmaker --pid 1536 --faketime +12m
# or
watchmaker --pid 1536 --faketime -12m

# time offset(h)
watchmaker --pid 1536 --faketime +12h
# or
watchmaker --pid 1536 --faketime -12h

# time offset(d)
watchmaker --pid 1536 --faketime +12d
# or
watchmaker --pid 1536 --faketime -12d

# time offset(y)
watchmaker --pid 1536 --faketime +1y
# or
watchmaker --pid 1536 --faketime -1y
```

## Reference

This project uses the following open-source software:

* [Chaos-mesh](https://github.com/chaos-mesh/chaos-mesh) - Reference chaos-mesh's watchmaker component to simulate process time
* [Libfaketime](https://github.com/wolfcw/libfaketime) - Reference the libfaketime dynamic link library to simulate time
* [timechaos-our-final-hack](https://chaos-mesh.org/blog/simulating-clock-skew-in-k8s-without-affecting-other-containers-on-node/#timechaos-our-final-hack) - Reference the blog to simulate time