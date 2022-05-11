DIR=/Users/aturner/go/src/github.com/synfinatic/aws-sso-cli/dist/

declare -A __aws_sso_sso=( [primary]=Primary [readonly]=ReadOnly )

__complete-sso-aws-sso(){
    local cmd="${1##*/}"
    local key=$COMP_KEY
    local type=$COMP_TYPE
    local word=${COMP_WORDS[$COMP_CWORD]}
    local line
    local point
    local sso

    for key in "${!__aws_sso_sso[@]}"; do 
        if echo -n $COMP_LINE | grep $key >/dev/null ; then
            line=$(echo -n ${COMP_LINE} | sed -Ee "s|${key}-||g")
            point=$(($COMP_POINT - $(echo $key | wc -m )))
            sso=${__aws_sso_sso[$key]}
            break
        fi
    done

    local _args="${AWS_SSO_HELPER_ARGS:- -L error} -S $sso"

    if [ $(echo -n "$line" | wc -m) -lt $point ]; then
        line="$line "
    fi
    COMPREPLY=()
    local cur
    _get_comp_words_by_ref -n : cur

    local flag=$3
    if [ "$word" = ":" ]; then
        curS="${3}:"
        flag=${COMP_WORDS[$(($COMP_CWORD - 2))]}
    elif [ "$flag" = ":" ]; then
        flag=${COMP_WORDS[$(($COMP_CWORD - 3))]}
        curS=${curS}${word}
    else
        curS=$word
    fi

    if [ "$flag" = "-p" -o "$flag" = "--profile" ]; then
        COMPREPLY=($(compgen -W '$(aws-sso $_args list --csv -P "Profile=$curS" Profile)' -- ""))
    elif [ "$flag" = "-a" -o "$flag" = "--arn" ]; then 
        COMPREPLY=($(compgen -W '$(aws-sso $_args list --csv -P "Ann=$cur" Arn)' -- ""))
    elif [ "$flag" = "-A" -o "$flag" = "--account" ]; then 
        COMPREPLY=($(compgen -W '$(aws-sso $_args list --csv -P "AccountId=$cur" AccountId)' -- ""))
    elif [ "$flag" = "-R" -o "$flag" = "--role" ]; then
        COMPREPLY=($(compgen -W '$(aws-sso $_args list --csv -P "RoleName=$cur" RoleName)' -- ""))
    else
        # pass through our completion
        local words=$(export COMP_LINE="${line}" ; export COMP_POINT=$point ; export COMP_KEY=$key export COMP_TYPE=$type ; aws-sso)
        COMPREPLY=($(compgen -W '$words' -- ""))
    fi

    __ltrim_colon_completions "$cur"
}

primary-aws-sso(){
    aws-sso -S Primary $@
}

readonly-aws-sso(){
    aws-sso -S ReadOnly $@
}

complete -F __complete-sso-aws-sso primary-aws-sso
complete -F __complete-sso-aws-sso readonly-aws-sso
