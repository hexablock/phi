# phi
Phi is a framework for build completely distributed and decentralized applications,
such as a file-system, key-value store and object store.  It is inspired from a
combination of underlying mechanisms such as blockchain, content-addressable-storage,
distributed-hash-tables to name a few.

## Features:

- Completely distributed and decentralized with no single-points-of-failure
- Distributed Write-Ahead-Log for consensus
- Distributed Hash Table
- Distributed Content-Addressable block based storage
- Automatic data de-duplication
- Automatic replication and healing

### Development

- When using debug mode a significant performance degrade may be seen.

### Known Issues

- When using phi in docker on a Mac with persistent storage, a massive performance hit
is incurred due to the way docker volumes and persistence are managed by docker on a Mac.
This is only pertinent for Macs