#!/bin/bash

git config core.hooksPath "$(git rev-parse --show-toplevel)/.githooks"
