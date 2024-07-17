# ddt

Take control of your disk drive.

## Build Instructions

``` shell
$ git clone https://github.com/shanebarnes/ddt.git
$ cd ddt
$ go build -v
```

## Examples

``` shell
$ # diskless read/write
$ ./ddt -if=/dev/zero -of=/dev/null -bs=1M -count=1000 -threads=4
$
$ # create a random 10MiB file
$ ./ddt -if=/dev/urandom -of=10M.bin -bs=1Mi -count=10 -threads=4
$
$ # limit the copy rate to 10 Mbps
$ ./ddt -if=/dev/zero -of=10M.bin -bs=100k -count=100 -rate=10M -threads=4
```
