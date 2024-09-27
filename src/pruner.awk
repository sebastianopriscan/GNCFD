#! /bin/awk -f

BEGIN { 
    cleanOn = 0 ;
    preintDecision = 0 ;
    result = ""
    result = result "//go:build release\n" ;
    result = result "// +build release\n\n" ;

    pruner_symbol = ENVIRON["PRUNER_SYMBOL"]
    push_regex = ".{0,256}\\/\\/" pruner_symbol "_PUSH.{0,256}"
    pop_regex = ".{0,256}\\/\\/" pruner_symbol "_POP.{0,256}"
} ;

/\/\/go:build/ {
    next ;
}
/\/\/ \+build/ {
    next ;
}

$0 ~ push_regex { 
    preintDecision ++ ;
    cleanOn++ ;
}

cleanOn <= 0 { 
    result = result $0 "\n" ;
}

$0 ~ pop_regex { 
    cleanOn-- ;
}

END {

    if (preintDecision != 0) {
        print result
    }
}