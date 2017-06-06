/*
Copyright 2016 The Doctl Authors All rights reserved.
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
	// ArgCommandWait is a wait for a droplet to be created argument.
	ArgCommandWait = "wait"
	// ArgDomainName is a domain name argument.
	ArgDomainName = "domain-name"
	// ArgDropletID is a droplet id argument.
	ArgDropletID = "droplet-id"
	// ArgDropletIDs is a list of droplet IDs.
	ArgDropletIDs = "droplet-ids"
	// ArgKernelID is a ekrnel id argument.
	ArgKernelID = "kernel-id"
	// ArgImage is an image argument.
	ArgImage = "image"
	// ArgImageID is an image id argument.
	ArgImageID = "image-id"
	// ArgImagePublic is a public image argument.
	ArgImagePublic = "public"
	// ArgImageSlug is an image slug argment.
	ArgImageSlug = "image-slug"
	// ArgIPAddress is an IP address argument.
	ArgIPAddress = "ip-address"
	// ArgDropletName is a droplet name argument.
	ArgDropletName = "droplet-name"
	// ArgResizeDisk is a resize disk argument.
	ArgResizeDisk = "resize-disk"
	// ArgSnapshotName is a snapshot name arugment.
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
	// ArgRegionSlug is a region slug argument.
	ArgRegionSlug = "region"
	// ArgSizeSlug is a size slug argument.
	ArgSizeSlug = "size"
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
	// ArgFormat is columns to include in output argment.
	ArgFormat = "format"
	// ArgNoHeader hides the output header.
	ArgNoHeader = "no-header"
	// ArgPollTime is how long before the next poll argument.
	ArgPollTime = "poll-timeout"
	// ArgTagName is a tag name
	ArgTagName = "tag-name"
	// ArgTagNames is a slice of possible tag names
	ArgTagNames = "tag-names"
	//ArgTemplate is template format
	ArgTemplate = "template"

	// ArgOutput is an output type argument.
	ArgOutput = "output"

	// ArgVolumeSize is the size of a volume.
	ArgVolumeSize = "size"
	// ArgVolumeDesc is the description of a volume.
	ArgVolumeDesc = "desc"
	// ArgVolumeRegion is the region of a volume.
	ArgVolumeRegion = "region"
	// ArgVolumeList is the IDs of many volumes.
	ArgVolumeList = "volumes"

	// ArgCertificateName is a name of the certificate.
	ArgCertificateName = "name"
	// ArgPrivateKeyPath is a path to a private key for the certificate.
	ArgPrivateKeyPath = "private-key-path"
	// ArgLeafCertificatePath is a path to a certificate leaf.
	ArgLeafCertificatePath = "leaf-certificate-path"
	// ArgCertificateChainPath is a path to a certificate chain.
	ArgCertificateChainPath = "certificate-chain-path"

	// ArgLoadBalancerName is a name of the load balancer.
	ArgLoadBalancerName = "name"
	// ArgLoadBalancerAlgorithm is a load balancing algorithm.
	ArgLoadBalancerAlgorithm = "algorithm"
	// ArgRedirectHttptoHttps is a flag that indicates whether HTTP requests to the load balancer on port 80 should be redirected to HTTPS on port 443.
	ArgRedirectHttpToHttps = "redirect-http-to-https"
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

	// ArgForce forces confirmation on actions
	ArgForce = "force"
)
