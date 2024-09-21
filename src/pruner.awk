#! /bin/awk -f

BEGIN { 
    cleanOn = 0 ;
    preintDecision = 0 ;
    result = ""
    result = result "//go:build release\n" ;
    result = result "// +build release\n\n" ;
} ;

/\/\/go:build/ {
    next ;
}
/\/\/ \+build/ {
    next ;
}

/\/\/DEBUG_PUSH/ { 
    preintDecision ++ ;
    cleanOn++ ;
}

cleanOn <= 0 { 
    result = result $0 "\n" ;
}

/\/\/DEBUG_POP/ { 
    cleanOn-- ;
}

END {

    if (preintDecision != 0) {
        print result
    }
}