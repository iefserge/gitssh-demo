# gitssh-demo

Absolutely the simplest SSH server that implements basics of Git SSH protocol
to allow full `git clone` to work.

This server requires repository to be packed into a single packfile,
however it doesn't just stream that file to the client, but reads each object header
and contents, and streams each object to the client.

It does not read full objects into memory, so this should well with pretty large files too.
Max supported packfile size is 2Gi.

It should be relatively easy to extend it support multiple packfiles, but special
case must be taken about duplicate objects. Here [OFS deltas](https://git-scm.com/docs/pack-format/2.31.0#_deltified_representation)
add a bit more complexity, because skipping or adding objects requires OFS deltas to be rewritten accordingly.

Additional logic can be added to filter and only send the requested objects, this requires
inspecting each object to ensure their dependencies are sent too. For example, commits
refer to trees and blobs. Also, some objects are stored as deltas (i.e. diffs based on other objects),
which also creates a dependency link.

Git docs on file formats are [here](https://git-scm.com/docs/pack-format).

## Limitations

- Only one `*.{idx,pack}` file is supported. Run `git repack -ad` to repack it
- It does not support loose objects
- Only one branch named `main` is supported
- Only full clones, partial `git fetch` would send the whole repository
- Only supports packfiles less than 2Gi
- No shallow clones etc
- No v2 protocol support
- No SSH auth. Do not expose this to the internet :)

## Usage

Clone this repository and run `git repack -ad`, then:

```bash
go run main.go
```

Then:

```
git clone ssh://git@localhost:2222/demo
```

should clone the source code of this repository.

## Using default port

Add this to your `~/.ssh/config`:

```
Host localhost
  Port 2222
```

Now you should be able to clone it using shorter address:

```
git clone git@localhost:demo
```

## FAQ

**Q: Can we stream the entire `*.pack` file to the client?**

A: Yes, this should work if we only have one packfile, but would not help if we needed to extend this demo to support
partial clones or multiple packfiles.

**Q: Why does it need to decompress every object with zlib and then discards the result?**

A: Git packfiles do not store information about the compressed size of the object in `*.pack` files. So decompression
is necessary to find out how many bytes to send. `*.rev` reverse indexes indirectly contain this information and
can be used as an alternative.

**Q: Why are we reading the entire list of objects from the `*.idx` file?**

A: This is done to make this demo simpler, and since we're sending all objects anyway. Real implementation would use
binary search over the index table to select only the necessary objects.

## License

MIT
