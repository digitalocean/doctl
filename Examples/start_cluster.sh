#/bin/bash

if [[ $# -lt 3 ]]; then 
	echo "Usage:"
	echo "\tstart_cluster.sh <domain> <region> <droplet_count>"
	exit 64
fi

DOMAIN=$1
REGION=$2
COUNT=$3

DISCOVERYURL=`curl -s -w "\n" https://discovery.etcd.io/new`
USERDATA=`cat cloud-config.yml| sed -e "s#DISCOVERY_URL#${DISCOVERYURL}#"`
MAX_ID=`echo $3-1 | bc`

echo "Starting CoreOS cluster ---"
echo "--- DISCOVERYURL: $DISCOVERYURL"
echo "--- DOMAIN: $DOMAIN"
echo "--- COUNT: $COUNT"
echo "--- REGION $REGION"
echo "--- USERDATA ---"
echo $USERDATA
echo "---"

for i in `seq 0 ${MAX_ID}`;
do
	HOST="core${i}"

	echo "Creating ${HOST}..."

	./doctl droplet create \
	--domain $DOMAIN \
	--image 5914637 \
	--size 1gb \
	--region $REGION \
	--ssh-keys Work,Home \
	--private-networking \
	--add-region \
	--user-data="${USERDATA}" \
	$HOST

    echo "Done."
done  