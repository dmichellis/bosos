bosos
=====

BOoking Simple Object Storage

This documentation is sorely lacking at the moment - more to follow

Config Options
--------------
* **seed_list** - List of Cassandra nodes to connect to and populate the host roster
* **keyspace** - Keyspace to use (see gocassos SCHEMA.cql)
* **username/password** - in case you are using authentication on your cluster
* **preferdc** - if you have a multi-dc cluster, which DC to pick hosts from
* **cassandra_auto_discovery** - use gocql's auto-discover node (see *preferdc*)
* **cassandra_discovery_time** - how often to pull cluster topology changes
* **conns_per_node** - how many concurrent connections per cassandra node
* **retries** - how many retries before returning an error and trying
the next consistency level
* **cassandra_timeout** - the default gocql driver has a low time
(600ms) - bump as needed
* **listen** - host and port to listen on
* **log_level** - 0..4 ("FUUU!" = 0, "WTF?!", "FYI", "BTW", "NVM" = 4)
* **lb_file** - 
* **access_log** - nginx-style access.log file
* **system_log** - where all the console (after config file was read)
and system logging goes
* **scrub_grace_time** - how long to wait after upload before purging
duplicate objects
* **allow_updates** - system-wide setting preventing overwrites and
deletions; note this is "best-effort" - it is possible that 2 files
uploaded at the exact same time are still both there
* **populate_paths** - will populate the persistent path cache for
(future) easier lookup.
* **expiration_round_up** - round up expirating objects to the next
time period; this helps cassandra cope with tombstones, deletions and
compaction scheduling
* **write_consistency** - list of consistency levels to try when writing
* **read_consistency** - list of consistency levels to try when reading
* **default_chunk-size** - how large you want the chunks of the full object
to be on the cassandra backend.
* **cassandra_reqs_per_get** - how many in-flight Cassandra request per each
HTTP GET request
* **cassandra_reqs_per_put** - how many in-flight Cassandra request per each
HTTP PUT request (deletes are always limited to 1 in-flight request)
* **concurrent_fetch_requests** - how many HTTP GET requests to accept at any
one time
* **concurrent_push_requests** - how many HTTP PUT requests to accept at any
one time
* **concurrent_requests** - how many global HTTP requests to accept
* **transfer_mode** - default transfer mode (for GETs) - you can use "stream"
and "batch"; stream will order and stream chunks as they come, whereas batch
will fetch the whole file first, THEN send it to the client.

Build
-----
BoSOS supports the main.BuildDate and main.GitHash compile-time definitions
and those will show up on its system.log. See "build.sh" for an example.
