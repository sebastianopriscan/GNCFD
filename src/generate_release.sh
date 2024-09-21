#! /bin/bash

dir=

for entry in `ls -1R`
do
    if [[ $entry =~ .{0,256}\.go ]];
    then
            rslt=`./pruner.awk $dir$entry`
            if [ "$rslt" != "" ]
                then
                    echo "$rslt" > $dir${entry:0:-3}"_release.go"
            fi
    elif [[ $entry =~ .{0,256}: ]] ;
    then
        dir=${entry:0:-1}"/"
    fi
done
