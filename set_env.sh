#!/bin/bash

env_path=".env"

# Export unencrypted env variables (we suppress warnings because we want to preserve
# numerical env values)

# shellcheck disable=SC2046
export $(grep -v '^#' $env_path | xargs -0)
