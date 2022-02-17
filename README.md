Litestream Read Replica Demo
============================

A demo application for running live read replication on fly.io with Litestream


## Setup 

```sh
# Setup app & fly.toml file
fly launch --name litestream-read-replica-demo --region ord --no-deploy

# Create a disk for each node
fly volumes create --region ams --size 1 data
fly volumes create --region atl --size 1 data
fly volumes create --region cdg --size 1 data
fly volumes create --region dfw --size 1 data
fly volumes create --region ewr --size 1 data
fly volumes create --region fra --size 1 data
fly volumes create --region gru --size 1 data
fly volumes create --region hkg --size 1 data
fly volumes create --region iad --size 1 data
fly volumes create --region lax --size 1 data
fly volumes create --region lhr --size 1 data
fly volumes create --region maa --size 1 data
fly volumes create --region nrt --size 1 data
fly volumes create --region ord --size 1 data
fly volumes create --region scl --size 1 data
fly volumes create --region sea --size 1 data
fly volumes create --region sin --size 1 data
fly volumes create --region sjc --size 1 data
fly volumes create --region syd --size 1 data
fly volumes create --region yyz --size 1 data

# Scale to one for each region
fly scale count 19

# Deploy application
fly deploy
```

