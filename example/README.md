# watchmaker example

## clone repository
```shell
git clone https://github.com/busybox-org/watchmaker.git
```

## build example cpp program
```shell
cd watchmaker
mkdir build
g++ -o example/print_time example/print_time.cpp
```

## run example cpp program
```shell
./example/print_time
```

## open a new terminal, fake cpp program time
```shell
cd watchmaker
./bin/watchmaker_linux_amd64 --pid $(pgrep -f "print_time") --faketime "2003-01-01 10:00:05"
```