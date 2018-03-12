#!/bin/bash

function quit {
  echo "Shutting down dgraph server and zero"
  curl -s localhost:8081/admin/shutdown
  # Kill Dgraphzero
  kill -9 $(pgrep -f "dgraph zero") > /dev/null

  if pgrep -x dgraph > /dev/null
  then
    while pgrep dgraph;
    do
      echo "Sleeping for 5 secs so that Dgraph can shutdown."
      sleep 5
    done
  fi

  echo "Clean shutdown done."
  return $1
}

function start {
  echo -e "Starting first server."
  dgraph server -p $BUILD/p -w $BUILD/w --memory_mb 4096 -o 1 &
  # Wait for leader election.
  sleep 5
  return 0
}

function startZero {
	echo -e "Starting dgraph zero.\n"
  dgraph zero -w $BUILD/wz &
  # To ensure dgraph doesn't start before dgraphzero.
	sleep 5
}
