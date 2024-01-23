#!/bin/sh
if [ $1 = "-v" ]; then
    /bin/evm -v
else
    stateProvided=0
    for index in ${1} ${2} ${3} ${4} ${5} ${6} ${7} ${8} ${9} ${10} ${11} ${12} ${13} ${14} ${15} ${16} ${17} ${18} ${19} ${20} ; do
        if [ $index = "--input.alloc" ]; then
            stateProvided=1
            break
        fi
    done
    if [ $stateProvided -eq 1 ]; then
        /bin/evm t8n ${1} ${2} ${3} ${4} ${5} ${6} ${7} ${8} ${9} ${10} ${11} ${12} ${13} ${14} ${15} ${16} ${17} ${18} ${19} ${20} --verbosity 2
    else
        /bin/evm t9n ${1} ${2} ${3} ${4} ${5} ${6} ${7} ${8} ${9} ${10} ${11} ${12} ${13} ${14} ${15} ${16} ${17} ${18} ${19} ${20}
    fi
fi
