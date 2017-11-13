# phi
Phi is a framework for build completely distributed and decentralized
applications, such as a file-system, distributed lock, key-value store and
object store.  It is inspired from a combination of underlying mechanisms such
as blockchain, content-addressable-storage, distributed-hash-tables to name a
few.

The primary purpose of phi is to provide the following functionality:

- Distributed consistent Write-Ahead-Log with P2P consensus
- Distributed Hash Table
- Distributed Content-Addressable storage

## Features:
- Completely distributed and decentralized architecture
- Distributed Write-Ahead-Log for consensus
- Distributed Hash Table
- Distributed Content-Addressable block based storage
- Automatic data de-duplication
- Automatic replication and healing

## Architecture & Design
Phi's architecture combines several methodologies to provide a truly distributed
and decentralized systems.  Some of the design patterns used are:

- Content addressable storage
- Consistent hashing
- Multi-Version Concurrency Control
- Blockchain
- Gossip
- Distribute-Hash-Tables

### Development

- When using debug mode a significant performance degrade may be seen.

### Known Issues

- When using phi in docker on a Mac with persistent storage, a massive
performance hit is incurred due to the way docker volumes and persistence are
managed by docker on a Mac. This is only pertinent for Macs
