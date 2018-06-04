file=$1
echo Executing "${file}"...
runthing="./build/bin/geth  --exec 'loadScript(\""$file"\")' attach http://127.0.0.1:8545"
eval $runthing
