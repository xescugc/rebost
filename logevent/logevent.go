package logevent

const (
	FileCreated              = "file.created"
	FileDeleted              = "file.deleted"
	FileTTLExpired           = "file.ttl_expired"
	ReplicaCreated           = "replica.created"
	ReplicaStale             = "replica.stale"
	ReplicaDeletePropagated  = "replica.delete_propagated"
	NodeJoined               = "node.joined"
	NodeLeft                 = "node.left"
	FileScrubRepaired        = "file.scrub_repaired"
	FileScrubNoReplicas      = "file.scrub_no_replicas"
	ReplicaConsistencyPurged = "replica.consistency_purged"
	DrainPreparing           = "drain.preparing"
	DrainWaiting             = "drain.waiting"
	DrainPurging             = "drain.purging"
	DrainLeaving             = "drain.leaving"
	AuditCreate              = "audit.create"
	AuditDelete              = "audit.delete"
	AuditAccess              = "audit.access"
	AuditStat                = "audit.stat"
)
