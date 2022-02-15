#!/bin/bash

# Determine which configuration file to use based on region.
if [ "$FLY_REGION" == "$PRIMARY_REGION" ]
then
	mv /etc/litestream.primary.yml /etc/litestream.yml
else
	mv /etc/litestream.replica.yml /etc/litestream.yml
fi

# Start litestream and the main application
litestream replicate -exec "/usr/local/bin/litestream-read-replica-demo -dsn /data/db -addr :8080"

