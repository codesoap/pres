[![Go Report Card](https://goreportcard.com/badge/github.com/codesoap/pres)](https://goreportcard.com/report/github.com/codesoap/pres)

Preserve files to make them resistant to
[bit rot](https://en.wikipedia.org/wiki/Data_rot). This tool is intended
to improve the longevity of backups.

**This is still a work in progress**

# Usage
```console
$ pres create my_data.foo
$ pres create my_data.foo > my_outfile.foo.pres
$ pres verify my_data.foo.pres
$ pres restore my_data.foo.pres
```

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
- No inplace repair of `*.pres` files.
- Added or lost data is not handled. Few bytes gone missing or being
  added may be handled in the future.

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
