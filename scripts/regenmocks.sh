#!/bin/bash

# regenerated generated mocks

set -euo pipefail

# GO111MODULE=off go get github.com/vektra/mockery/.../

# TODO: Get gomock bins.
cd "do"
echo "Creating AccountService.go"
mockgen -source account.go -package=mocks AccountService > mocks/AccountService.go
echo "Creating ActionService.go"
mockgen -source actions.go -package=mocks ActionService > mocks/ActionService.go
echo "Creating CDNsService.go"
mockgen -source cdns.go -package=mocks CDNsService > mocks/CDNsService.go
echo "Creating CertificatesService.go"
mockgen -source certificates.go -package=mocks CertificateSservice > mocks/CertificatesService.go
echo "Creating DatabasesService.go"
mockgen -source databases.go -package=mocks DatabasesService > mocks/DatabasesService.go
echo "Creating DomainService.go"
mockgen -source domains.go -package=mocks DomainService > mocks/DomainService.go
echo "Creating DropletActionService.go"
mockgen -source droplet_actions.go -package=mocks DropletActionsService > mocks/DropletActionService.go
echo "Creating DropletsService.go"
mockgen -source droplets.go -package=mocks DropletsServoce > mocks/DropletsService.go
echo "Creating FirewallsService.go"
mockgen -source firewalls.go -package=mocks FirewallsService > mocks/FirewallsService.go
echo "Creating FloatingIPActionsService.go"
mockgen -source floating_ip_actions.go -package=mocks FloatingIPActionsService > mocks/FloatingIPActionsService.go
echo "Creating FloatingIPsService.go"
mockgen -source floating_ips.go -package=mocks FloatingIPsService > mocks/FloatingIPsService.go
echo "Creating ImageActionsService.go"
mockgen -source image_actions.go -package=mocks ImageActionsService > mocks/ImageActionsService.go
echo "Creating ImageService.go"
mockgen -source images.go -package=mocks ImageService > mocks/ImageService.go
echo "Creating KubernetesService.go"
mockgen -source kubernetes.go -package=mocks KubernetesService > mocks/KubernetesService.go
echo "Creating LoadBalancersService.go"
mockgen -source load_balancers.go -package=mocks LoadBalancersService > mocks/LoadBalancersService.go
echo "Creating ProjectsService.go"
mockgen -source projects.go -package=mocks ProjectsService > mocks/ProjectsService.go
echo "Creating RegionsService.go"
mockgen -source regions.go -package=mocks RegionsService > mocks/RegionsService.go
echo "Creating SizesService.go"
mockgen -source sizes.go -package=mocks SizesService > mocks/SizesService.go
echo "Creating SnapshotsService.go"
mockgen -source snapshots.go -package=mocks SnapshotsService > mocks/SnapshotsService.go
echo "Creating KeysService.go"
mockgen -source sshkeys.go -package=mocks KeysService > mocks/KeysService.go
echo "Creating TagsService.go"
mockgen -source tags.go -package=mocks TagsService > mocks/TagsService.go
echo "Creating VolumeActionsService.go"
mockgen -source volume_actions.go -package=mocks VolumeActionsService > mocks/VolumeActionsService.go
echo "Creating VolumesService.go"
mockgen -source volumes.go -package=mocks VolumesService > mocks/VolumesService.go

