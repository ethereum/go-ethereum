#!/bin/sh

hivechain generate \
  --pos \
  --fork-interval 6 \
  --tx-interval 1 \
  --length 600 \
  --outdir testdata \
  --lastfork prague \
  --outputs accounts,genesis,chain,headstate,txinfo,headblock,headfcu,newpayload,forkenv
