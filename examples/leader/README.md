# Leader topology example

A topology describes the intended relationship between members within a Habitat service group. The `leader` topology employs an election to choose a leader. Find out more about topologies in Habitat in this [documentation](https://www.habitat.sh/docs/run-packages-topologies/).

## Workflow

Simply run:

  `kubectl create -f examples/leader/habitat.yml`.

This will deploy 3 instances of consul Habitat service.

Note: Whenever creating a `leader` topology specify instance `count` of 3 or more and would be best if the number is odd, this is so the election can take place.
