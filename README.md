[![Go Report Card](https://goreportcard.com/badge/github.com/codesoap/pres)](https://goreportcard.com/report/github.com/codesoap/pres)

Preserve files to make them resistant to
[bit rot](https://en.wikipedia.org/wiki/Data_rot). This tool is intended
to improve the longevity of backups.

# Usage
```console
$ # Create my_data.foo.pres:
$ pres create my_data.foo
Calculating parity information and checksums.
Writing output.

$ # From time to time you should check if your files are damaged:
$ pres verify my_data.foo.pres
All conf blocks are intact.
103 out of 103 shards are intact.
No problems found.

$ # If `pres verify my_data.foo.pres` found some damage, you should
$ # restore the original data and recreate the *.pres file:
$ pres restore my_data.foo.pres
Checking shards for damage.
Restoring damaged shards.
Verifying restored data.
Writing output.
$ pres create my_data.foo
Calculating parity information and checksums.
Writing output.
```

Advanced use cases:
```console
$ # For improved performance you might want to write the output to a
$ # different disk:
$ pres create my_data.foo > /media/disk2/my_data.foo.pres
Calculating parity information and checksums.
Writing output.

$ # In the same manner, you can redirect the output when restoring:
$ pres restore my_data.foo.pres > /media/disk2/my_data.foo
```

# Installation
`go get -v -u 'github.com/codesoap/pres'` will install the latest
version of `pres` to `$HOME/go/bin/`.

# How it Works
`pres` calculates and stores parity information for the given file
using [Solomon Reed error correction](https://en.wikipedia.org/wiki/Reed_Solomon).

Together with the original data and the newly generated parity
information, hashes of the data and parity information are stored
(multiple times, for fail safety) in a `*.pres` file. The hashes
correlate to so called "shards", segments of the data and parity
information, that can be restored once corrupted.

## Verifying a files integrity:
- Check if the copies of all shards' hashes match.
- Check if the stored hashes of all shards match the ones
  generated from the data and parity information.
   
## Restoring the data from a `*.pres` file
- If there are at least as many shards intact, as there are data
  shards, the corrupted shards can be restored.
- Restoring the original data file is then simply a matter of
  concatenating the now repaired data shards.
   
# Shortcomings
1. The input file should at least contain 10_000 bytes of data (amount of
   data shards squared).
2. Added or lost data is not handled. Few bytes gone missing or being
   added may be handled in the future.
3. No in-place repair of `*.pres` files.
4. Although the data and parity shards can take at least three bit-flips
   without becoming unrestorable, two bit-flips can already destroy the
   header.
5. Changes in the filename or other meta-data are not prevented.

# Comparison to similar software
## [darrenldl/blockyarchive](https://github.com/darrenldl/blockyarchive)
`blockyarchive` improves on all listed shortcomings of `pres`, except
2., but trades performance and filesize for that. It is probably better
suited if you want to recover from more extreme damage, like filesystem
failure or large amounts of rotten bits.

Performance with 1GiB of random data:
```console
$ time blkar encode 1GiB.data
[...]
real    0m30,693s
user    0m37,792s
sys     0m21,894s

$ time blkar check 1GiB.data.ecsbx
[...]
real    0m8,126s
user    0m6,703s
sys     0m1,024s

$ time pres create 1GiB.data
[...]
real    0m10,476s
user    0m5,153s
sys     0m7,542s

$ time pres verify 1GiB.data.pres
[...]
real    0m3,266s
user    0m2,102s
sys     0m1,244s
```

Output filesizes (`blkar` adds ~23.9%, `pres` adds ~3.0%):
```console
$ du 1GiB.data*
1048580 1GiB.data
1298956 1GiB.data.ecsbx
1080048 1GiB.data.pres
```

# File Format Example
```
[conf]
version=1
data_len=997
data_shard_cnt=5
parity_shard_cnt=2
shard_1_crc32c=360670479
shard_2_crc32c=1762937310
shard_3_crc32c=1664223142
shard_4_crc32c=1400629101
shard_5_crc32c=2293045559
shard_6_crc32c=563834295
shard_7_crc32c=3265204826

[conf_copy_1]
version=1
data_len=997
data_shard_cnt=5
parity_shard_cnt=2
shard_1_crc32c=360670479
shard_2_crc32c=1762937310
shard_3_crc32c=1664223142
shard_4_crc32c=1400629101
shard_5_crc32c=2293045559
shard_6_crc32c=563834295
shard_7_crc32c=3265204826

[conf_copy_2]
version=1
data_len=997
data_shard_cnt=5
parity_shard_cnt=2
shard_1_crc32c=360670479
shard_2_crc32c=1762937310
shard_3_crc32c=1664223142
shard_4_crc32c=1400629101
shard_5_crc32c=2293045559
shard_6_crc32c=563834295
shard_7_crc32c=3265204826

[binary]
<binary-data><binary-parity-information>
```
