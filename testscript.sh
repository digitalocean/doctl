#!/bin/bash
DROPLET_ID=$(go run cmd/doctl/main.go compute droplet create --image ubuntu-20-04-x64 --size s-1vcpu-1gb --region nyc1 wb2 --ssh-keys 40563907 --wait -o json | jq '.[] | .id')
go run cmd/doctl/main.go compute ssh $DROPLET_ID --ssh-retry-max 30  
doctl compute droplet delete $DROPLET_ID
