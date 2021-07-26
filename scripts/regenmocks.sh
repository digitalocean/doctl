#!/bin/bash

# regenerated generated mocks

set -euo pipefail

cd "do"

GO111MODULE=off go get -u github.com/golang/mock/mockgen

mockgen -source account.go -package=mocks AccountService > mocks/AccountService.go
mockgen -source actions.go -package=mocks ActionService > mocks/ActionService.go
mockgen -source apps.go -package=mocks AppsService > mocks/AppsService.go
mockgen -source balance.go -package=mocks BalanceService > mocks/BalanceService.go
mockgen -source billing_history.go -package=mocks BillingHistoryService > mocks/BillingHistoryService.go
mockgen -source cdns.go -package=mocks CDNsService > mocks/CDNsService.go
mockgen -source certificates.go -package=mocks CertificateSservice > mocks/CertificatesService.go
mockgen -source databases.go -package=mocks DatabasesService > mocks/DatabasesService.go
mockgen -source domains.go -package=mocks DomainService > mocks/DomainService.go
mockgen -source droplet_actions.go -package=mocks DropletActionsService > mocks/DropletActionService.go
mockgen -source droplets.go -package=mocks DropletsService > mocks/DropletsService.go
mockgen -source firewalls.go -package=mocks FirewallsService > mocks/FirewallsService.go
mockgen -source floating_ip_actions.go -package=mocks FloatingIPActionsService > mocks/FloatingIPActionsService.go
mockgen -source floating_ips.go -package=mocks FloatingIPsService > mocks/FloatingIPsService.go
mockgen -source image_actions.go -package=mocks ImageActionsService > mocks/ImageActionsService.go
mockgen -source images.go -package=mocks ImageService > mocks/ImageService.go
mockgen -source invoices.go -package=mocks InvoicesService > mocks/InvoicesService.go
mockgen -source kubernetes.go -package=mocks KubernetesService > mocks/KubernetesService.go
mockgen -source load_balancers.go -package=mocks LoadBalancersService > mocks/LoadBalancersService.go
mockgen -source projects.go -package=mocks ProjectsService > mocks/ProjectsService.go
mockgen -source regions.go -package=mocks RegionsService > mocks/RegionsService.go
mockgen -source registry.go -package=mocks RegistryService > mocks/RegistryService.go
mockgen -source snapshots.go -package=mocks SnapshotsService > mocks/SnapshotsService.go
mockgen -source sizes.go -package=mocks SizesService > mocks/SizesService.go
mockgen -source sshkeys.go -package=mocks KeysService > mocks/KeysService.go
mockgen -source tags.go -package=mocks TagsService > mocks/TagsService.go
mockgen -source volume_actions.go -package=mocks VolumeActionsService > mocks/VolumeActionsService.go
mockgen -source volumes.go -package=mocks VolumesService > mocks/VolumesService.go
mockgen -source vpcs.go -package=mocks VPCsService > mocks/VPCsService.go
mockgen -source 1_clicks.go -package=mocks OneClickService > mocks/OneClickService.go
mockgen -source ../pkg/runner/runner.go -package=mocks Runner > mocks/Runner.go
mockgen -source ../pkg/listen/listen.go -package=mocks Listen > mocks/Listen.go
mockgen -source monitoring.go -package=mocks MonitoringService > mocks/MonitoringService.go
