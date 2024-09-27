#! /bin/bash

dir=

for entry in `ls -1R`
do
    if [[ $entry =~ .{0,256}\.go ]];
    then
        from_file=true
        tmp_rslt=
        for i in "$@" ;
        do
            if [ "$from_file" == "true" ];
            then
                rslt=`env PRUNER_SYMBOL="$i" ./pruner.awk $dir$entry`
                if [ "$rslt" != "" ]
                then
                    tmp_rslt="$rslt"
                    from_file=false
                fi
            else
                rslt=`echo "$tmp_rslt" | env PRUNER_SYMBOL="$i" ./pruner.awk`
                if [ "$rslt" != "" ]
                then
                    tmp_rslt="$rslt"
                fi
            fi
        done
            
        if [ "$tmp_rslt" != "" ]
            then
                echo "$tmp_rslt" > $dir${entry:0:-3}"_release.go"
        fi
    elif [[ $entry =~ .{0,256}: ]] ;
    then
        dir=${entry:0:-1}"/"
    fi
done
