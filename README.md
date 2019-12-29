Conserve files to make them resistant to
[bit rot](https://en.wikipedia.org/wiki/Data_rot). This tool is intended
to improve the longevity of backups.

# Usage
```console
$ cons create my_data.foo
$ cons verify my_data.foo.cons
$ cons restore my_data.foo.cons
```

# How it Works
`cons` calculates and stores parity information for the given file
using [Solomon Reed error correction](https://en.wikipedia.org/wiki/Reed_Solomon).

Together with the original data and the newly generated parity
information, hashes of the data and parity information are stored
(multiple times, for fail safety) in a `*.cons` file. The hashes
correlate to so called "shards", segments of the data and parity
information, that can be restored once corrupted.

## Verifying a files integrity:
- Check if the copies of all shards' hashes match.
- Check if the stored hashes of all shards match the ones
  generated from the data and parity information.
   
## Restoring the data from a `*.cons` file
- If there are at least as many shards intact, as there are data
  shards, the corrupted shards can be restored.
- Restoring the original data file is then simply a matter of
  concatenating the now repaired data shards.
   
# Shortcomings
- No inplace repair of `*.cons` files.
- Added or lost data is not handled. Few bytes gone missing or being
  added may be handled in the future.

# File Format Example
```
[conf]
version=1
data_len=997
data_shard_cnt=5
par_shard_cnt=2
shard_1_md5=d3b07384d113edec49eaa6238ad5ff00
shard_2_md5=4bc70ed8e59f0430607feb42d2ae983e
shard_3_md5=cefaeb8b9d53a53e2941badca2a3793b
shard_4_md5=69a8832cd4fcdfbd22b010cc0ea445e7
shard_5_md5=187e623c4ea1189ed8155836c7b24935
shard_6_md5=45c2beb36121c18035d66abe12f8e740
shard_7_md5=eb305f7662f633543e846aefa5beb7e4

[conf_copy_1]
version=1
data_len=997
data_shard_cnt=5
par_shard_cnt=2
shard_1_md5=d3b07384d113edec49eaa6238ad5ff00
shard_2_md5=4bc70ed8e59f0430607feb42d2ae983e
shard_3_md5=cefaeb8b9d53a53e2941badca2a3793b
shard_4_md5=69a8832cd4fcdfbd22b010cc0ea445e7
shard_5_md5=187e623c4ea1189ed8155836c7b24935
shard_6_md5=45c2beb36121c18035d66abe12f8e740
shard_7_md5=eb305f7662f633543e846aefa5beb7e4

[conf_copy_2]
version=1
data_len=997
data_shard_cnt=5
par_shard_cnt=2
shard_1_md5=d3b07384d113edec49eaa6238ad5ff00
shard_2_md5=4bc70ed8e59f0430607feb42d2ae983e
shard_3_md5=cefaeb8b9d53a53e2941badca2a3793b
shard_4_md5=69a8832cd4fcdfbd22b010cc0ea445e7
shard_5_md5=187e623c4ea1189ed8155836c7b24935
shard_6_md5=45c2beb36121c18035d66abe12f8e740
shard_7_md5=eb305f7662f633543e846aefa5beb7e4

[binary]
<binary-data><binary-parity-information>
```
