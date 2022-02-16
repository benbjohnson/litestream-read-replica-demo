Litestream Read Replica Demo
============================

A demo application for running live read replication on fly.io with Litestream


## Setup 

```sh
# Setup app & fly.toml file
fly launch --name litestream-read-replica-demo --region ord --no-deploy

# Assign Chicago as the primary region.
fly secrets set PRIMARY_REGION=ord

# Create a disk for each node
fly volumes create --region ord --size 1 data
fly volumes create --region sjc --size 1 data
fly volumes create --region ams --size 1 data
fly volumes create --region nrt --size 1 data
```

