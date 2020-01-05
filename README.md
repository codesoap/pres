[![Go Report Card](https://goreportcard.com/badge/github.com/codesoap/pres)](https://goreportcard.com/report/github.com/codesoap/pres)

Preserve files to make them resistant to
[bit rot](https://en.wikipedia.org/wiki/Data_rot). This tool is intended
to improve the longevity of backups.

# Usage
```console
$ # Create my_data.foo.pres:
$ pres create my_data.foo
Calculating parity information and checksums.
Appending output to 'my_data.foo'.
Renaming 'my_data.foo' to 'my_data.foo.pres'.

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
Writing 'my_data.foo'.
$ pres create my_data.foo
Calculating parity information and checksums.
Appending output to 'my_data.foo'.
Renaming 'my_data.foo' to 'my_data.foo.pres'.
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
   metadata.
5. Changes in the filename or other meta-data are not prevented.

# Comparison to similar software
The intended use case of `pres` is to prevent a few bit-flips from
corrupting a backup file. It is easy to use, way faster than it's
alternatives and produces comparatively small output files (when using
default configurations).

With 1GiB of random data, I got these timings on my machine:
```console
$ time pres create 1GiB.data
[...]
real    0m5,025s
user    0m4,772s
sys     0m1,785s

$ time pres verify 1GiB.data.pres
[...]
real    0m3,120s
user    0m2,157s
sys     0m1,034s
```

The resulting file will be ~3.0% larger than the original file.

## [darrenldl/blockyarchive](https://github.com/darrenldl/blockyarchive)
`blkar` improves on all listed shortcomings of `pres`, except 2., but
trades performance and filesize for that. It is probably better suited
if you want to recover from more extreme damage, like filesystem failure
or large amounts of rotten bits.

Performance (using the same amount of data and parity shards as `pres` does):
```console
$ time blkar encode --sbx-version 17 --rs-data 100 --rs-parity 3 1GiB.data
[...]
real    0m32,864s
user    0m39,948s
sys     0m23,268s

$ time blkar check 1GiB.data.ecsbx
[...]
real    0m6,920s
user    0m5,701s
sys     0m0,930s
```

The resulting file is ~6.3% larger than the original file. It is
significantly larger than the output of `pres`, because `blkar` splits
the input into blocks, which are then further split into shards for
parity calculation. This makes the `*.ecsbx` more resistant to randomly
spread bit-flips, but less resistant to large patches of bit-flips, if I
understand the design correctly. The filesize can be reduced to be ~3.4%
larger than the original file by using the non-default
`--sbx-version 19`.

## [par2](https://github.com/Parchive/par2cmdline/)
`par2` generates multiple output files, which must be used in
combination with the original file to verify integrity or repair the
data. This means you have to deal with multiple files when
verifying the data's integrity or restoring data.

`par2` seems to cope with point 1., 4. and even 2. of the shortcomings
of `pres`.

On the downside `par2` does apparently not inform you about damaged
recovery files, as long as there is still at least one undamaged
recovery file left. This means that you could already be just two
bit-flips away from loosing your data, without `par2` notifying you
about the occurred damage.

Performance (using the same amount of data and parity shards as `pres` does):
```console
$ time par2 create -b100 -r3 1GiB.data
[...]
real    0m23,305s
user    0m23,893s
sys     0m2,453s

$ time par2 verify 1GiB.data
[...]
real    0m23,799s
user    0m27,722s
sys     0m1,086s
```

# File Format Example
```
<data><parity-information>

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
```
