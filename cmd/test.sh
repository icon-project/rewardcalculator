#!/bin/bash

CMD="./dbtest"
DBList=("1K" "10K" "100K" "1M" "10M")
DBcount=("1000" "10000" "100000" "1000000" "10000000")
WORKER=(1 2 3 4)
WRITEBATCH=(0 10 100 1000)
LOGFILE="/tmp/dbtest.log"

function usage {
    echo "$0 command"
    echo "commands"
    echo "   create                           Create DBs $DBList"
    echo "   calculate [DB]                   Calculate I-Score"
}

# args: dbname worker batch
function calculate {
    echo ""
    echo "## Run $CMD $1 calculate 0 $2 $3 " >&2
    echo "## Run $CMD $1 calculate 0 $2 $3 "
    $CMD $1 calculate 0 $2 $3 >&1
}

# args: dbname entry_count
function create {
    echo ""
    echo "## Run $CMD $1 create $2" >&2
    echo "## Run $CMD $1 create $2"
    $CMD $1 create $2 >&1
}

# args: command
function print_command {
    echo "###############################"
    date >&1
    echo "command : $1"
    echo "###############################"
}


# main start

print_command $1 >> ${LOGFILE}

case $1 in
    create)
        for ((i=0 ; i < ${#DBList[@]}; i++)); do
            create ${DBList[$i]} ${DBcount[$i]} >> ${LOGFILE}
        done
        ;;
    calculate)
        for i in ${WORKER[@]}; do 
            for j in ${WRITEBATCH[@]}; do 
                calculate $2 ${i} ${j} >> ${LOGFILE}
            done
        done
        ;;
    *)
        echo "Wrong input"
        usage
        ;;
esac

exit 0
