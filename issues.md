* Deleting: Does not delete the Replica of the queue(so it keeps trying) and if you try to recreate again the same image it'll not replciates as there is one on the queue with the ID of the supposed Volume that has it but no longer. The VolumeReplicaID is generated so i cannot set it
* Reset: Removing everything and restarting the service triggers a Reset(????????) but that reset does not create the `tmp/` or `files/` so it fails at restart
* Allow to create buckets:
* Remove the old UoW implementation that requires to specify the registry
