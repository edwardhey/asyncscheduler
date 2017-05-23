#!/bin/bash


for i in $(seq 1 1000)
do
	echo $i
	## SetJob
	curl -X "POST" "http://localhost:1984/job/set?ID=111&URL=http:%2F%2Fchargerlink.com&maxAttempts=5" \
	     -H "Content-Type: application/json; charset=utf-8" \
	     -d $'{
	  "id": "'$i'",
	  "payload": {
	    "aa": "bb"
	  },
	  "max_attempts": "10",
	  "url": "http://charger.com",
	  "attempt_interval": "1"
	}'
	
done
