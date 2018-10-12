# sharding

It's  possible to  add another  layer on  top of  this kind  of setup  to shard
data. Depending  on your  needs you could  shard on the  measurement name  or a
specific tag like `customer_id`. The sharding  layer would have to service both
queries and writes.

As  this relay  does not  handle queries,  it will  not implement  any sharding
logic. Any sharding would have to be done externally to the relay.

## Caveats

While `influxdb-relay` does provide some  level of high availability, there are
a few scenarios that need to be accounted for:

- `influxdb-relay`  will  not relay  the `/query`  endpoint, and  this includes
  schema  modification  (create  database,   `DROP`s,  etc).  This  means  that
  databases must be created before points are written to the backends.
- Continuous queries  will still only write their results  locally. If a server
  goes down, the continuous query will have to be backfilled after the data has
  been recovered for that instance.
- Overwriting points is potentially unpredictable. For example, given servers A
  and B, if  B is down, and point  X is written (we'll call the  value X1) just
  before B  comes back online,  that write is  queued behind every  other write
  that occurred while B was offline. Once  B is back online, the first buffered
  write succeeds, and  all new writes are now allowed  to pass-through. At this
  point (before  X1 is written to  B), X is  written again (with value  X2 this
  time) to both A and B. When the relay reaches the end of B's buffered writes,
  it will write X (with value X1) to B... At this point A now has X2, but B has
  X1.
  - It  is probably best to  avoid re-writing points (if  possible). Otherwise,
    please be aware that overwriting the same  field for a given point can lead
    to data differences.
  - This  could potentially  be mitigated  by waiting for  the buffer  to flush
    before opening writes back up to being passed-through.

## Limitations



## Development

Please read carefully [CONTRIBUTING.md][contribute-href]  before making a merge
request.

Clone repository into your `$GOPATH`.

```sh
$ mkdir -p ${GOPATH}/src/github.com/vente-privee
$ cd ${GOPATH}/src/github.com/vente-privee
$ git clone git@github.com:vente-privee/influxdb-relay
```

Enter the directory and build the daemon.

```sh
$ cd ${GOPATH}/src/github.com/vente-privee/influxdb-relay
$ go build -a -ldflags '-extldflags "-static"' -o influxdb-relay
```

## Miscellaneous

```
    ╚⊙ ⊙╝
  ╚═(███)═╝
 ╚═(███)═╝
╚═(███)═╝
 ╚═(███)═╝
  ╚═(███)═╝
   ╚═(███)═╝
```

[license-img]: https://img.shields.io/badge/license-MIT-blue.svg
[license-href]: LICENSE
[overview-href]: https://github.com/influxdata/influxdb-relay
[contribute-href]: CONTRIBUTING.md
