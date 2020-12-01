#!/bin/sh

name="$1"
filename="${1}.go"


if [ -z "$name" ]; then
    echo "$0 MSGNAME"
    exit 1
fi

if [ -e "$filename" ]; then
    echo "$1 Exists"
    exit 1
fi

cat >$filename <<EOL
package main

import (
	"encoding/json"
    "github.com/czh/jwa"
    "net/http"
)

type __MSGNAME__Request struct {
	Msg string
}

type __MSGNAME__Reply struct {
	Msg string
}

func init() {
	jwa.AddMessageHandler("__MSGNAME__", __MSGNAME__Handler)
}

func __MSGNAME__Handler(data map[interface{}]interface{}, payload []byte) (resp interface{}) {

	var request __MSGNAME__Request

	if err := json.Unmarshal(payload, &request); err != nil {
		return http.StatusBadRequest
	}

	reply := &__MSGNAME__Reply{ }

	// write code following here now !

	return reply
}
EOL

sed -i "s/__MSGNAME__/$name/g" $filename

