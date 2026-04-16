# xnew

`xnew` appends only lines that are not already present in a target file and is tuned for very large inputs with low memory usage.

## Benchmark timings

The timings below measure the case where we try to write a single line into files that already contain the listed number of lines.

Test instance configuration: `4 GB RAM / 2-core CPU`.

| Number of lines | anew | xnew | Faster |
|-----------------|------|------|--------|
| 1K | 0.005s | 0.005s | 1x |
| 10K | 0.009s | 0.009s | 1x |
| 100K | 0.070s | 0.031s | 2.26x |
| 1M | 0.886s | 0.236s | 3.75x |
| 10M | 12.4773s | 2.830s | 4.41x |
| 20M | 27.763s | 6.036s | 4.60x |
| 50M | 56.446s | 16.809s | 3.36x |
| 100M | 1m38s | 30.534s | 3.21x |



## Install

```bash
go install github.com/jsmonhq/xnew@latest
```

Make sure `$GOPATH/bin` or `$HOME/go/bin` is in your `PATH`.

## Why xnew

- Streams the existing file instead of loading it entirely into memory.
- Uses a compact open-addressed `uint64` hash set for fast membership checks.
- Uses XXH3 for high-throughput hashing.
- Uses buffered I/O to reduce syscall overhead.

## Trade-off

`xnew` deduplicates using a 64-bit XXH3 hash. Collisions are extremely unlikely in practice, but still theoretically possible. This trade-off significantly reduces memory usage compared with exact byte-span indexing.

## Usage

```bash
# Append new lines from stdin to existing.txt
cat new_lines.txt | xnew existing.txt

# Write only new unique lines to a separate file
cat new_lines.txt | xnew existing.txt -o only_new.txt

# Trim leading/trailing spaces before comparing
cat new_lines.txt | xnew existing.txt -trim

# Quiet mode (no stdout output, exit code only)
cat new_lines.txt | xnew existing.txt -q
```

## Build

```bash
git clone https://github.com/jsmonhq/xnew.git
cd xnew
go build -o xnew .
```

## Flags

| Argument | Description |
|----------|-------------|
| `existing-file` | Path to existing file (required). If missing, it is treated as empty. |
| `-o` | Output file. Default: append to `existing-file`. |
| `-trim` | Trim spaces before comparing lines. |
| `-q` | Quiet mode; suppress stdout output. |

## Output behavior

By default, `xnew` writes only newly added lines to stdout (one per line), so you can pipe or redirect output as needed:

```bash
echo "example" | xnew existing.txt >> new.txt
```

