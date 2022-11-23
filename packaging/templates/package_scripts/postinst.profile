#!/bin/bash
# This is a postinstallation script so the service can be configured and started when requested
#
sudo mkdir -p /var/lib/bor
sudo chown -R bor /var/lib/bor
sudo systemctl daemon-reload
