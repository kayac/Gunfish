#!/bin/bash
set -e

no=$1
port=38103 # default is test server port number
token=$3
if [[ $2 != "" ]] ; then
    port=$2
fi

usage(){
    cat <<EOF
     ./curl_test.sh <test_type> <port> <token>:
       send:      Success curl.
       send_many: Send 250 notification.
       stats_app:     Get report stats.
       stats_profile: Get go application resource stats.
EOF
}

send_many(){
    payload='[{"token": "sendtoomanytoken", "payload": {"aps": {"alert": "send too many", "sound": "test"}, "u":"a", "t":"a"}}'
    for cnt in $(seq 2 250)
    do
        payload=$payload',{"token": "sendtoomanytoken", "payload": {"aps": {"alert": "send too many", "sound": "test"}, "u":"a", "t":"a"}}'
    done
    payload=$payload"]"
    curl -s -X POST -d "$payload" -H "Content-Type: application/json" http://localhost:$port/push/apns
}

if [[ "$no" == "send" ]] ; then 
    if [[ "$token" == "" ]] ; then
        echo "required device token."
        usage
    else
        payload='[{"token": "'$token'", "payload": {"aps": {"alert": "push notification test", "sound": "default", "badge": 1, "category": "sns", "content-available": 1}, "u":"a", "t":"a"}}]'
        curl -X POST -d "$payload" -H "Content-Type: application/json" http://localhost:$port/push/apns
    fi
elif [[ "$no" == "send_many" ]] ; then
    send_many
elif [[ "$no" == "stats_app" ]] ; then
    curl -s -X GET http://localhost:$port/stats/app
elif [[ "$no" == "stats_profile" ]] ; then
    curl -s -X GET http://localhost:$port/stats/profile
elif [[ "$no" == "loop_send_many" ]] ; then
    while : ;
    do
        sleep 1
        clear
        send_many | jq -r .
    done
elif [[ "$no" == "loop_stats_app" ]] ; then
    while : ;
    do
        sleep 1
        clear
        curl -s -X GET http://localhost:$port/stats/app | jq -r .
    done
elif [[ "$no" == "loop_stats_profile" ]] ; then
    while : ;
    do
        sleep 1
        clear
        curl -s -X GET http://localhost:$port/stats/profile | jq -r .
    done
else
    echo "Do nothing."
    usage
fi
