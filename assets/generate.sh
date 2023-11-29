#!/bin/bash

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)

rm -rf $SCRIPT_DIR/generated
mkdir -p $SCRIPT_DIR/generated

d2 --layout elk --scale 4 --pad 0 $SCRIPT_DIR/cluster-topology.d2 $SCRIPT_DIR/generated/cluster-topology.png

d2 --layout elk --scale 4 --pad 0 $SCRIPT_DIR/temporal-upgrade.d2 $SCRIPT_DIR/generated/temporal-upgrade.png

d2 --layout elk --scale 4 --pad 0 $SCRIPT_DIR/upgrade-new-cluster.d2 $SCRIPT_DIR/generated/upgrade-new-cluster.png

d2 --layout elk --scale 4 --pad 0 $SCRIPT_DIR/upgrade.d2 $SCRIPT_DIR/generated/upgrade.png

d2 --layout elk --scale 4 --pad 0 $SCRIPT_DIR/upgrade-with-temporal.d2 $SCRIPT_DIR/generated/upgrade-with-temporal.png

d2 --layout elk --scale 4 --pad 0 $SCRIPT_DIR/pod-spread.d2 $SCRIPT_DIR/generated/pod-spread.png
