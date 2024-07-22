# ddt

Take control of your disk drive.

## Build Instructions

### Checkout Git Repository

``` shell
git clone https://github.com/shanebarnes/ddt.git
cd ddt
```

### Build and Test Using go-task

``` shell
task build
task test
task benchmark
```

### Build and Test Using go Command

See [Taskfile.yml](./Taskfile.yml)

## Examples

``` shell
# diskless read/write
./ddt -if=/dev/zero -of=/dev/null -bs=1M -count=1000 -threads=4

# fast diskless read/write using in-memory implementation of special system files
./ddt -if=/dev/zero -of=/dev/null -bs=1M -count=1000 -threads=4 -mem

# create a random 10MiB file
./ddt -if=/dev/urandom -of=10M.bin -bs=1Mi -count=10 -threads=4

# limit the copy rate to 10 Mbps
./ddt -if=/dev/zero -of=10M.bin -bs=100k -count=100 -rate=1250k -threads=4
```
