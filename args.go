/*
Copyright 2018 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package doctl

const (
	// ArgAccessToken is the access token to be used for the operations
	ArgAccessToken = "access-token"
	// ArgContext is the name of the auth context
	ArgContext = "context"
	// ArgDefaultContext is the default auth context
	ArgDefaultContext = "default"
	// ArgActionID is an action id argument.
	ArgActionID = "action-id"
	// ArgActionAfter is an action after argument.
	ArgActionAfter = "after"
	// ArgActionBefore is an action before argument.
	ArgActionBefore = "before"
	// ArgActionResourceType is an action resource type argument.
	ArgActionResourceType = "resource-type"
	// ArgActionRegion is an action region argument.
	ArgActionRegion = "region"
	// ArgActionStatus is an action status argument.
	ArgActionStatus = "status"
	// ArgActionType is an action type argument.
	ArgActionType = "action-type"
	// ArgApp is the app ID.
	ArgApp = "app"
	// ArgAppSpec is a path to an app spec.
	ArgAppSpec = "spec"
	// ArgAppLogType the type of log.
	ArgAppLogType = "type"
	// ArgAppDeployment is the deployment ID.
	ArgAppDeployment = "deployment"
	// ArgAppLogFollow follow logs.
	ArgAppLogFollow = "follow"
	// ArgAppLogTail tail logs.
	ArgAppLogTail = "tail"
	// ArgAppForceRebuild forces a deployment rebuild
	ArgAppForceRebuild = "force-rebuild"
	// ArgAppAlertDestinations is a path to an app alert destination file.
	ArgAppAlertDestinations = "app-alert-destinations"
	// ArgClusterName is a cluster name argument.
	ArgClusterName = "cluster-name"
	// ArgClusterVersionSlug is a cluster version argument.
	ArgClusterVersionSlug = "version"
	// ArgVPCUUID is a VPC UUID argument.
	ArgVPCUUID = "vpc-uuid"
	// ArgClusterVPCUUID is a cluster vpc-uuid argument.
	ArgClusterVPCUUID = "vpc-uuid"
	// ArgClusterNodePool are a cluster's node pools arguments.
	ArgClusterNodePool = "node-pool"
	// ArgClusterUpdateKubeconfig updates the local kubeconfig.
	ArgClusterUpdateKubeconfig = "update-kubeconfig"
	// ArgNodePoolName is a cluster's node pool name argument.
	ArgNodePoolName = "name"
	// ArgNodePoolCount is a cluster's node pool count argument.
	ArgNodePoolCount = "count"
	// ArgNodePoolAutoScale is a cluster's node pool auto_scale argument.
	ArgNodePoolAutoScale = "auto-scale"
	// ArgNodePoolMinNodes is a cluster's node pool min_nodes argument.
	ArgNodePoolMinNodes = "min-nodes"
	// ArgNodePoolMaxNodes is a cluster's node pool max_nodes argument.
	ArgNodePoolMaxNodes = "max-nodes"
	// ArgNodePoolNodeIDs is a cluster's node pool nodes argument.
	ArgNodePoolNodeIDs = "node-ids"
	// ArgMaintenanceWindow is a cluster's maintenance window argument
	ArgMaintenanceWindow = "maintenance-window"
	// ArgAutoUpgrade is a cluster's auto-upgrade argument.
	ArgAutoUpgrade = "auto-upgrade"
	// ArgHA is a cluster's highly available control plane argument.
	ArgHA = "ha"
	// ArgSurgeUpgrade is a cluster's surge-upgrade argument.
	ArgSurgeUpgrade = "surge-upgrade"
	// ArgCommandUpsert is an upsert for a resource to be created or updated argument.
	ArgCommandUpsert = "upsert"
	// ArgCommandWait is a wait for a resource to be created argument.
	ArgCommandWait = "wait"
	// ArgSetCurrentContext is a flag to set the new kubeconfig context as current.
	ArgSetCurrentContext = "set-current-context"
	// ArgDropletID is a droplet id argument.
	ArgDropletID = "droplet-id"
	// ArgDropletIDs is a list of droplet IDs.
	ArgDropletIDs = "droplet-ids"
	// ArgKernelID is a kernel id argument.
	ArgKernelID = "kernel-id"
	// ArgKubernetesLabel is a Kubernetes label argument.
	ArgKubernetesLabel = "label"
	// ArgKubernetesTaint is a Kubernetes taint argument.
	ArgKubernetesTaint = "taint"
	// ArgKubernetesAlias is a Kubernetes alias argument that saves authentication information under the specified context.
	ArgKubernetesAlias = "alias"
	// ArgKubeConfigExpirySeconds indicates the length of time the token in a kubeconfig will be valid in seconds.
	ArgKubeConfigExpirySeconds = "expiry-seconds"
	// ArgImage is an image argument.
	ArgImage = "image"
	// ArgImageID is an image id argument.
	ArgImageID = "image-id"
	// ArgImagePublic is a public image argument.
	ArgImagePublic = "public"
	// ArgImageSlug is an image slug argument.
	ArgImageSlug = "image-slug"
	// ArgIPAddress is an IP address argument.
	ArgIPAddress = "ip-address"
	// ArgDropletName is a droplet name argument.
	ArgDropletName = "droplet-name"
	// ArgResizeDisk is a resize disk argument.
	ArgResizeDisk = "resize-disk"
	// ArgSnapshotName is a snapshot name argument.
	ArgSnapshotName = "snapshot-name"
	// ArgSnapshotDesc is the description for volume snapshot.
	ArgSnapshotDesc = "snapshot-desc"
	// ArgResourceType is the resource type for snapshot.
	ArgResourceType = "resource"
	// ArgBackups is an enable backups argument.
	ArgBackups = "enable-backups"
	// ArgIPv6 is an enable IPv6 argument.
	ArgIPv6 = "enable-ipv6"
	// ArgPrivateNetworking is an enable private networking argument.
	ArgPrivateNetworking = "enable-private-networking"
	// ArgMonitoring is an enable monitoring argument.
	ArgMonitoring = "enable-monitoring"
	// ArgDropletAgent is an argument for enabling/disabling the Droplet agent.
	ArgDropletAgent = "droplet-agent"
	// ArgRecordData is a record data argument.
	ArgRecordData = "record-data"
	// ArgRecordID is a record id argument.
	ArgRecordID = "record-id"
	// ArgRecordName is a record name argument.
	ArgRecordName = "record-name"
	// ArgRecordPort is a record port argument.
	ArgRecordPort = "record-port"
	// ArgRecordPriority is a record priority argument.
	ArgRecordPriority = "record-priority"
	// ArgRecordType is a record type argument.
	ArgRecordType = "record-type"
	// ArgRecordTTL is a record ttl argument.
	ArgRecordTTL = "record-ttl"
	// ArgRecordWeight is a record weight argument.
	ArgRecordWeight = "record-weight"
	// ArgRecordFlags is a record flags argument.
	ArgRecordFlags = "record-flags"
	// ArgRecordTag is a record tag argument.
	ArgRecordTag = "record-tag"
	// ArgRegionSlug is a region slug argument.
	ArgRegionSlug = "region"
	// ArgSchemaOnly is a schema only argument.
	ArgSchemaOnly = "schema-only"
	// ArgSizeSlug is a size slug argument.
	ArgSizeSlug = "size"
	// ArgSizeUnit is a size unit argument.
	ArgSizeUnit = "size-unit"
	// ArgsSSHKeyPath is a ssh argument.
	ArgsSSHKeyPath = "ssh-key-path"
	// ArgSSHKeys is a ssh key argument.
	ArgSSHKeys = "ssh-keys"
	// ArgsSSHPort is a ssh argument.
	ArgsSSHPort = "ssh-port"
	// ArgsSSHAgentForwarding is a ssh argument.
	ArgsSSHAgentForwarding = "ssh-agent-forwarding"
	// ArgsSSHPrivateIP is a ssh argument.
	ArgsSSHPrivateIP = "ssh-private-ip"
	// ArgSSHCommand is a ssh argument.
	ArgSSHCommand = "ssh-command"
	// ArgUserData is a user data argument.
	ArgUserData = "user-data"
	// ArgUserDataFile is a user data file location argument.
	ArgUserDataFile = "user-data-file"
	// ArgImageName name is an image name argument.
	ArgImageName = "image-name"
	// ArgImageExternalURL is a URL that returns an image file.
	ArgImageExternalURL = "image-url"
	// ArgImageDistro is the name of a custom image's distribution
	ArgImageDistro = "image-distribution"
	// ArgImageDescription is free text that describes the image.
	ArgImageDescription = "image-description"
	// ArgKey is a key argument.
	ArgKey = "key"
	// ArgKeyName is a key name argument.
	ArgKeyName = "key-name"
	// ArgKeyPublicKey is a public key argument.
	ArgKeyPublicKey = "public-key"
	// ArgKeyPublicKeyFile is a public key file argument.
	ArgKeyPublicKeyFile = "public-key-file"
	// ArgSSHUser is a SSH user argument.
	ArgSSHUser = "ssh-user"
	// ArgFormat is columns to include in output argument.
	ArgFormat = "format"
	// ArgNoHeader hides the output header.
	ArgNoHeader = "no-header"
	// ArgPollTime is how long before the next poll argument.
	ArgPollTime = "poll-timeout"
	// ArgTagName is a tag name
	// NOTE: ArgTagName will be deprecated once existing uses have been migrated
	// to use `--tag` (ArgTag). ArgTagName should not be used on new calls.
	ArgTagName = "tag-name"
	// ArgTagNames is a slice of possible tag names
	// NOTE: ArgTagNames will be deprecated once existing uses have been migrated
	// to use `--tag` (ArgTag). ArgTagNames should not be used on new calls.
	ArgTagNames = "tag-names"
	// ArgTag specifies tag.  --tag can be repeated or multiple tags can be , separated.
	ArgTag = "tag"
	//ArgTemplate is template format
	ArgTemplate = "template"
	// ArgVersion is the version of the command to use
	ArgVersion = "version"
	// ArgVerbose enables verbose output
	ArgVerbose = "verbose"

	// ArgOutput is an output type argument.
	ArgOutput = "output"

	// ArgVolumeSize is the size of a volume.
	ArgVolumeSize = "size"
	// ArgVolumeDesc is the description of a volume.
	ArgVolumeDesc = "desc"
	// ArgVolumeRegion is the region of a volume.
	ArgVolumeRegion = "region"
	// ArgVolumeSnapshot is the snapshot from which to create a volume.
	ArgVolumeSnapshot = "snapshot"
	// ArgVolumeFilesystemType is the filesystem type for a volume.
	ArgVolumeFilesystemType = "fs-type"
	// ArgVolumeFilesystemLabel is the filesystem label for a volume.
	ArgVolumeFilesystemLabel = "fs-label"
	// ArgVolumeList is the IDs of many volumes.
	ArgVolumeList = "volumes"
	// ArgVolumeSnapshotList is the IDs of many volume snapshots.
	ArgVolumeSnapshotList = "snapshots"
	// ArgLoadBalancerList is the IDs of many load balancers.
	ArgLoadBalancerList = "load-balancers"

	// ArgCDNTTL is a cdn ttl argument
	ArgCDNTTL = "ttl"
	// ArgCDNDomain is a cdn custom domain argument
	ArgCDNDomain = "domain"
	// ArgCDNCertificateID is a certificate id to use with a custom domain
	ArgCDNCertificateID = "certificate-id"
	// ArgCDNFiles is a cdn files argument
	ArgCDNFiles = "files"

	// ArgCertificateName is a name of the certificate.
	ArgCertificateName = "name"
	// ArgCertificateDNSNames is a list of DNS names.
	ArgCertificateDNSNames = "dns-names"
	// ArgPrivateKeyPath is a path to a private key for the certificate.
	ArgPrivateKeyPath = "private-key-path"
	// ArgLeafCertificatePath is a path to a certificate leaf.
	ArgLeafCertificatePath = "leaf-certificate-path"
	// ArgCertificateChainPath is a path to a certificate chain.
	ArgCertificateChainPath = "certificate-chain-path"
	// ArgCertificateType is a certificate type.
	ArgCertificateType = "type"

	// ArgLoadBalancerName is a name of the load balancer.
	ArgLoadBalancerName = "name"
	// ArgLoadBalancerAlgorithm is a load balancing algorithm.
	ArgLoadBalancerAlgorithm = "algorithm"
	// ArgRedirectHTTPToHTTPS is a flag that indicates whether HTTP requests to the load balancer on port 80 should be redirected to HTTPS on port 443.
	ArgRedirectHTTPToHTTPS = "redirect-http-to-https"
	// ArgEnableProxyProtocol is a flag that indicates whether PROXY protocol should be enabled on the load balancer.
	ArgEnableProxyProtocol = "enable-proxy-protocol"
	// ArgDisableLetsEncryptDNSRecords is a flag that when set will disable the creation of DNS records pointing to the load balancer IP from the apex domain in the cert.
	ArgDisableLetsEncryptDNSRecords = "disable-lets-encrypt-dns-records"
	// ArgEnableBackendKeepalive is a flag that indicates whether keepalive connections should be enabled to target droplets from the load balancer.
	ArgEnableBackendKeepalive = "enable-backend-keepalive"
	// ArgStickySessions is a list of sticky sessions settings for the load balancer.
	ArgStickySessions = "sticky-sessions"
	// ArgHealthCheck is a list of health check settings for the load balancer.
	ArgHealthCheck = "health-check"
	// ArgForwardingRules is a list of forwarding rules for the load balancer.
	ArgForwardingRules = "forwarding-rules"

	// ArgFirewallName is a name of the firewall.
	ArgFirewallName = "name"
	// ArgInboundRules is a list of inbound rules for the firewall.
	ArgInboundRules = "inbound-rules"
	// ArgOutboundRules is a list of outbound rules for the firewall.
	ArgOutboundRules = "outbound-rules"

	// ArgProjectName is the name of a project.
	ArgProjectName = "name"
	// ArgProjectDescription is the description of a project.
	ArgProjectDescription = "description"
	// ArgProjectPurpose is the purpose of a project.
	ArgProjectPurpose = "purpose"
	// ArgProjectEnvironment is the environment of a project. Should be one of 'Development', 'Staging', 'Production'.
	ArgProjectEnvironment = "environment"
	// ArgProjectIsDefault is used to change the default project.
	ArgProjectIsDefault = "is_default"
	// ArgProjectResource is a flag for your resource URNs
	ArgProjectResource = "resource"

	// ArgDatabaseEngine is a flag for specifying which database engine to use
	ArgDatabaseEngine = "engine"
	// ArgDatabaseNumNodes is the number of nodes in the database cluster
	ArgDatabaseNumNodes = "num-nodes"
	// ArgDatabaseMaintenanceDay is the new day for the maintenance window
	ArgDatabaseMaintenanceDay = "day"
	// ArgDatabaseMaintenanceHour is the new hour for the maintenance window
	ArgDatabaseMaintenanceHour = "hour"
	// ArgDatabasePoolUserName is the name of user for use with connection pool
	ArgDatabasePoolUserName = "user"
	// ArgDatabasePoolDBName is the database for use with connection pool
	ArgDatabasePoolDBName = "db"
	// ArgDatabasePoolSize is the flag for connection pool size
	ArgDatabasePoolSize = "size"
	// ArgDatabasePoolMode is the flag for connection pool mode
	ArgDatabasePoolMode = "mode"
	// ArgDatabaseUserMySQLAuthPlugin is a flag for setting the MySQL user auth plugin
	ArgDatabaseUserMySQLAuthPlugin = "mysql-auth-plugin"

	// ArgPrivateNetworkUUID is the flag for VPC UUID
	ArgPrivateNetworkUUID = "private-network-uuid"

	// ArgForce forces confirmation on actions
	ArgForce = "force"

	// ArgObjectName is the Kubernetes object name
	ArgObjectName = "name"
	// ArgObjectNamespace is the Kubernetes object namespace
	ArgObjectNamespace = "namespace"

	// ArgVPCName is a name of the VPC.
	ArgVPCName = "name"
	// ArgVPCDescription is a VPC description.
	ArgVPCDescription = "description"
	// ArgVPCDefault is the VPC default argument, to update a specific VPC to the default VPC.
	ArgVPCDefault = "default"
	// ArgVPCIPRange is a VPC range of IP addresses in CIDR notation.
	ArgVPCIPRange = "ip-range"

	// ArgReadWrite indicates a generated token should be read/write.
	ArgReadWrite = "read-write"
	// ArgRegistryExpirySeconds indicates the length of time the token will be valid in seconds.
	ArgRegistryExpirySeconds = "expiry-seconds"
	// ArgSubscriptionTier is a subscription tier slug.
	ArgSubscriptionTier = "subscription-tier"
	// ArgGCIncludeUntaggedManifests indicates that a garbage collection should delete
	// all untagged manifests.
	ArgGCIncludeUntaggedManifests = "include-untagged-manifests"
	// ArgGCExcludeUnreferencedBlobs indicates that a garbage collection should
	// not delete unreferenced blobs.
	ArgGCExcludeUnreferencedBlobs = "exclude-unreferenced-blobs"
	// ArgRegistryAuthorizationServerEndpoint is the endpoint of the OAuth authorization server
	// used to revoke credentials on logout.
	ArgRegistryAuthorizationServerEndpoint = "authorization-server-endpoint"

	// 1-Click Args

	// ArgOneClicks is the flag to pass in 1-click application slugs
	ArgOneClicks = "1-clicks"

	// ArgOneClickType is the type of 1-Click
	ArgOneClickType = "type"

	//ArgDangerous indicates whether to delete the cluster and all it's associated resources
	ArgDangerous = "dangerous"

	// ArgDatabaseFirewallRule the firewall rules.
	ArgDatabaseFirewallRule = "rule"

	// ArgDatabaseFirewallRuleUUID is the UUID for the firewall rules.
	ArgDatabaseFirewallRuleUUID = "uuid"

	// Monitoring Args

	// ArgAlertPolicyDescription is the flag to pass in the alert policy description.
	ArgAlertPolicyDescription = "description"

	// ArgAlertPolicyType is the alert policy type.
	ArgAlertPolicyType = "type"

	// ArgAlertPolicyValue is the alert policy value.
	ArgAlertPolicyValue = "value"

	// ArgAlertPolicyWindow is the alert policy window.
	ArgAlertPolicyWindow = "window"

	// ArgAlertPolicyTags is the alert policy tags.
	ArgAlertPolicyTags = "tags"

	// ArgAlertPolicyEntities is the alert policy entities.
	ArgAlertPolicyEntities = "entities"

	// ArgAlertPolicyEnabled is whether the alert policy is enabled.
	ArgAlertPolicyEnabled = "enabled"

	// ArgAlertPolicyCompare is the alert policy comparator.
	ArgAlertPolicyCompare = "compare"

	// ArgAlertPolicyEmails are the emails to send alerts to.
	ArgAlertPolicyEmails = "emails"

	// ArgAlertPolicySlackChannels are the Slack channels to send alerts to.
	ArgAlertPolicySlackChannels = "slack-channels"

	// ArgAlertPolicySlackURLs are the Slack URLs to send alerts to.
	ArgAlertPolicySlackURLs = "slack-urls"
)
