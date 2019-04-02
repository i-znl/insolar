#!/usr/bin/env bash
set -e

CONFIGS_DIR=configs
BASE_DIR=scripts/insolard
ORACLES_KEYS_FILE=$BASE_DIR/$CONFIGS_DIR/oracle_keys.json

generate_oracles_keys()
{
    echo "generate_oracles_keys() starts ..."
	bin/insolar -c gen_keys > $ORACLES_KEYS_FILE
	echo "generate_oracles_keys() end."
}


generate_oracles_keys
