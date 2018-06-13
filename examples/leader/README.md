# Leader topology example

A topology describes the intended relationship between members within a Habitat service group. The `leader` topology employs an election to choose a leader. Find out more about topologies in Habitat in this [documentation](https://www.habitat.sh/docs/run-packages-topologies/).

## Workflow

Simply run:

  `kubectl create -f examples/leader/habitat.yml`.

This will deploy 1 instance of Redis Habitat service.

Note: To have functioning services in the `leader` topology, you must set the `count` field to at least 3. It is recommended that the number is odd as this prevents a split quorum during the leader election.
