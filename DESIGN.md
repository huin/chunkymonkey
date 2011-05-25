Design
======

Preface for the confused
------------------------

(AKA excuses from the developer)

Note that currently the server does not fulfil many aspects of this design, and
should migrate to them over time. Most notably, at the time of this writing,
the server currently runs inside a single process on a single machine which
makes for easy initial development.

Also note that many of the aspects of the design are not set in stone. There
are likely oversights in the design, and things that will change over time.

Additionally, much of the code that is being written has been with the approach
of "get it right now, make it fast later". Optimisations tend to make for buggy
and hard to maintain code - and this is something best avoided early in a
project. The programming language used for this project (Google Go) is also in
an early stage, and the official toolchain for it yet lacks many
optimizations - so some things should get faster over time anyway.


Implementation notes
--------------------

Read this section for some useful points on how the current implementation
works. Note that things could change and this section could become out of date
quite easily, so don't read things here as gospel if the code is telling a
different story (although please do either change this doc to reflect this or
raise an issue).

Currently the system uses goroutines for a few different purposes:

*   1 goroutine for the Game mainloop (reads/writes to the ChunkManager also go
    through this goroutine),
*   3 goroutines per Player,
*   1 goroutine per Chunk.

Currently communication between them takes the form of using the Enqueue method
on the object in question to run a function within the goroutine context.
Additionally there is a lock on the player object that allows running either in
a calling goroutine or in the player's own goroutine. This configuration will
likely not last forever, particularly when it comes to the point of
distributing the server components, where some of these communication methods
might take the form of networked RPCs.


Intent
------

Create a scalable Minecraft service that is able to scale horizontally across
many hosts.


Coding standards/style
----------------------

Pretty informal, as few decisions have been made here as yet. Note that this is
a statement of intent, rather than reality, as the codebase needs some
improvement in the following regards.

*   Use gofmt to format code before pushing to the master repository. 1 tab
    character per indent is the indentation style used (use `make fmt`).
*   Unit tests must pass before pushing to the master repository (use
    `make test`).
*   Unit test code where possible/reasonable to do so. It is likely that a
    mocking library will be used at some stage to enable testing of more
    complex interactions - using interfaces for dependency injection is helpful
    to make this easier.
*   Document code constructs tersely but clearly with comments. See
    standard Go packages for good examples of style.
*   Identifier naming generally makes use of CamelCasing. Leading
    upper/lowercase letter dictates private/public to package, as defined by
    the Go programming language.
*   Getter method names should *not* start with "Get". See
    http://golang.org/doc/effective_go.html#Getters
*   Functions and methods should preferably return objects by reference
    (pointer) and callers can change them freely without fear of
    synchronization issues, thanks to our clear channels-based design.

To ease the above, there are three make targets:

    $ make fmt   # Format the codebase with gofmt.
    $ make check # Perform some style checks.
    $ make test  # Run all unit tests.


Top-level architecture
----------------------

![Top-level architecture][1]

*   The clients here are typically official Notchian clients, although some may
    well be third-party clients and/or bots.
*   Frontend servers are load-balanced servers that are the end-point for the
    client's TCP connection. They are responsible for the player's immediate
    state in the world. They also multiplex client data out to chunk servers and
    relay chunk server data back to clients as appropriate.
*   Lookup servers form a highly-available service that allow all the frontend
    and chunk servers to find other chunk servers.
*   Chunk servers each hold the state of a subset of chunks in the world. They
    simulate physics in the world, hold item state (possibly mob state as well,
    but that's not clearly thought out, yet).
*   Storage servers are for longer term storage of player and chunk data.


Things still to consider for the design
---------------------------------------

The design has not yet laid down anything other than vague thoughts for the
following:

*   Chunk snapshot data for storage.
*   Which server has responsibility for simulating mobs.
*   Which server performs authentication/blocks/priviledge decisions.
*   Firm coding standards.
*   Modding support.


Server process considerations
-----------------------------

Each of the processes shown in the top-level architecture diagram must be able
to run on seperate hosts, or shared hosts, or all on a single host. This means
that a server process might be the sole Chunkymonkey server running on a host,
or it might be co-existing with many others.

In a "production" setup a server will very likely share system resources with
other processes that have no functional connection with the running service. In
a "development" setup a server will likely coexist with multiple related
servers (and the client) on the same physical host.

A server process that spends a lot of the time doing very little should require
very little in the way of CPU resources, and hopefully as little active RAM as
possible. Hardware is expensive, and the best (within reason) use of what is
available must be made.


Frontend server design
----------------------

The purpose of a frontend server is to terminate the player's client TCP
connection. It maintains connection(s) to chunk server(s), and creates new
connections to chunk servers as the player moves around in the world. It
receives the player's client minecraft "packets", and translates them into
events, which are sent to the chunk servers. It receives updates from the chunk
servers and sends them to clients.


Chunk server design
-------------------

Each chunk server is responsible for a "shard" of chunks. This shard is a
square in the X-Z plane, and consists of a whole number of chunks. There is a
single "master" goroutine for each shard that is responsible for the chunks and
everything that happens within them.

Note that the player logic happens almost entirely on the chunk server within
the master goroutine, which makes inventory transactions etc. very
straightforward and atomic.


Monitoring
----------

Currently, monitoring takes two forms:

*   Whitebox live variable inspection (via [Go's expvar package][2]).
*   Logging output (via [Go's log package][3]).


[1]: ../../raw/master/diagrams/top-level-architecture.png  "Top-level architecture"
[2]: http://golang.org/pkg/expvar/                         "Go expvar package"
[3]: http://golang.org/pkg/log/                            "Go log package"
