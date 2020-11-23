#!/bin/bash

sed -e 's/\(\"services\"\):\["discovery\"]/\1:["pss","bzz"]/'
