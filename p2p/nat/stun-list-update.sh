#!/bin/sh

rm -f stun-list.txt
curl https://raw.githubusercontent.com/pradt2/always-online-stun/refs/heads/master/valid_ipv4s.txt >> stun-list.txt
curl https://raw.githubusercontent.com/pradt2/always-online-stun/refs/heads/master/valid_ipv6s.txt >> stun-list.txt
