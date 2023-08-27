#!/bin/bash

# Copyright (c) Zander Schwid & Co. LLC.
# SPDX-License-Identifier: BUSL-1.1

PASSPHRASE=template
DAYS=3650
LENGTH=2048

if [ ! -d ssl ]; then
  mkdir ssl
fi

if [ ! -f ssl/ca.key ]; then
  echo "Make CA"
  openssl req -new -x509 -keyout ssl/ca.key -out ssl/ca.crt -days $DAYS -passout pass:$PASSPHRASE
fi

if [ ! -f ssl/client.crt ]; then
  echo "Make client"
  openssl genrsa -out ssl/client.key $LENGTH
  echo "00" > ssl/ca.srl
  openssl req -sha256 -key ssl/client.key -new -out ssl/client.req
  openssl x509 -req -in ssl/client.req -CA ssl/ca.crt -CAkey ssl/ca.key -out ssl/client.crt -days $DAYS -passin pass:$PASSPHRASE
fi

if [ ! -f ssl/server.crt ]; then
  echo "Make server"
  openssl genrsa -out ssl/server.key $LENGTH
  openssl pkey -in ssl/server.key -pubout > ssl/server.pub
  openssl req -new -x509 -sha256 -key ssl/server.key -out ssl/server.crt -days $DAYS
fi



